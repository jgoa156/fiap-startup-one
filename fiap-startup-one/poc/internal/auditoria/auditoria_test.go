package auditoria

import (
	"context"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"lisura/internal/conta"
	"lisura/internal/dbtest"
	"lisura/internal/inscricao"
	"lisura/internal/pagamento"
	"lisura/internal/seed"
)

var pool *pgxpool.Pool

func TestMain(m *testing.M) {
	p, err := dbtest.Pool(context.Background(), "t_auditoria")
	if err != nil {
		dbtest.SemBanco(err)
	}
	pool = p
	code := m.Run()
	if pool != nil {
		pool.Close()
	}
	os.Exit(code)
}

var seq atomic.Int64

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// enviada cria um candidato da rodada com uma inscrição aguardando pagamento.
func enviada(t *testing.T, rodada string) (inscricaoID int64) {
	t.Helper()
	if pool == nil {
		t.Skip("sem banco de teste")
	}
	ctx := context.Background()
	cert, err := seed.CriarCertame(ctx, pool, seed.Padrao())
	must(t, err)
	n := seq.Add(1)
	email := "ensaio." + rodada + "." + string(rune('a'+n)) + "@lisura.test"
	cand, err := conta.Criar(ctx, pool, conta.GerarCPF(n+time.Now().UnixNano()%1_000_000*100), "Ensaio", email)
	must(t, err)
	must(t, conta.AtualizarDados(ctx, pool, cand, conta.Dados{Nome: "Ensaio", Email: email,
		DataNascimento: "1990-01-01", Municipio: "Campinas", UF: "SP"}))
	id, _, err := inscricao.Abrir(ctx, pool, cert.ID, cand, cert.Cargos[0], inscricao.Ator{Tipo: inscricao.Candidato, ID: "t"})
	must(t, err)
	must(t, inscricao.DefinirCidade(ctx, pool, id, cand, cert.Cidades[0]))
	must(t, inscricao.AceitarDeclaracoes(ctx, pool, id, cand, true))
	_, err = inscricao.Finalizar(ctx, pool, id, cand)
	must(t, err)
	return id
}

func cobranca(t *testing.T, inscricaoID int64, metodo, estado string) int64 {
	t.Helper()
	var id int64
	must(t, pool.QueryRow(context.Background(), `
		INSERT INTO cobranca (inscricao_id, metodo, valor_centavos, estado, vence_em)
		VALUES ($1, $2, 8000, $3, now() + interval '3 days') RETURNING id`, inscricaoID, metodo, estado).Scan(&id))
	return id
}

func confirmar(t *testing.T, inscricaoID int64) {
	t.Helper()
	_, err := inscricao.Transicionar(context.Background(), pool, inscricao.Comando{InscricaoID: inscricaoID,
		Evento: inscricao.ConfirmarPagamento, Ator: inscricao.Ator{Tipo: inscricao.Sistema, ID: "t"}})
	must(t, err)
}

// pagaDireito é o que o sistema faz num pagamento normal: cobrança PAGA, recebimento registrado,
// inscrição PAGA.
func pagaDireito(t *testing.T, inscricaoID int64) {
	t.Helper()
	recebido(t, cobranca(t, inscricaoID, "pix", "PAGA"), pagamento.Confirmado)
	confirmar(t, inscricaoID)
}

// recebido registra um pagamento do prestador na cobrança, com o resultado que o sistema deu a ele, e
// devolve o id do pagamento.
func recebido(t *testing.T, cobrancaID int64, resultado string) string {
	t.Helper()
	id := "pg_" + strconv.FormatInt(seq.Add(1), 10)
	_, err := pool.Exec(context.Background(), `
		INSERT INTO pagamento_recebido (psp_pagamento_id, cobranca_id, valor_centavos, pago_em, origem, resultado)
		VALUES ($1, $2, 8000, now(), 'webhook', $3)`, id, cobrancaID, resultado)
	must(t, err)
	return id
}

// cancelarCertameDe marca o certame da inscrição como cancelado, sem passar pelo cancelamento: é o
// estado que a auditoria tem de acusar se o cancelamento parar no meio.
func cancelarCertameDe(t *testing.T, inscricaoID int64) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		UPDATE certame SET cancelado_em = now(), cancelamento_motivo = 'TESTE'
		WHERE id = (SELECT certame_id FROM inscricao WHERE id = $1)`, inscricaoID)
	must(t, err)
}

// devolvido registra a devolução de um pagamento, no estado dado.
func devolvido(t *testing.T, pagamentoID string, valor int, estado string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO devolucao (psp_pagamento_id, cobranca_id, inscricao_id, valor_centavos, motivo, metodo_origem, estado,
		                       psp_devolucao_id, concluida_em)
		SELECT p.psp_pagamento_id, b.id, b.inscricao_id, $2, 'PAGO_EM_DUPLICIDADE', b.metodo, $3::text,
		       CASE WHEN $3::text = 'CONCLUIDA' THEN 'dev_' || p.psp_pagamento_id END,
		       CASE WHEN $3::text = 'CONCLUIDA' THEN now() END
		FROM pagamento_recebido p JOIN cobranca b ON b.id = p.cobranca_id WHERE p.psp_pagamento_id = $1`,
		pagamentoID, valor, estado)
	must(t, err)
}

func TestRodadaLimpaNaoAcusaNada(t *testing.T) {
	a := enviada(t, "limpa")
	pagaDireito(t, a)
	enviada(t, "limpa") // enviada, ainda sem pagar
	enviada(t, "outra") // de outra rodada: não entra na conta
	// O candidato pagou duas vezes e o sistema já devolveu: não é problema.
	b := enviada(t, "limpa")
	pagaDireito(t, b)
	aMais := cobranca(t, b, "boleto", "PAGA_EM_DUPLICIDADE")
	_, err := pool.Exec(context.Background(), `UPDATE cobranca SET reembolso_pendente = true WHERE id = $1`, aMais)
	must(t, err)
	devolvido(t, recebido(t, aMais, pagamento.EmDuplicidade), 8000, pagamento.DevolucaoConcluida)

	r, err := Auditar(context.Background(), pool, "limpa", Esperado{Enviadas: 3, Pagas: 2})
	must(t, err)
	if r.Candidatos != 3 || r.Enviadas != 3 || r.Pagas != 2 || r.ReembolsosPendentes != 1 ||
		r.Devolucoes[pagamento.DevolucaoConcluida] != 1 {
		t.Fatalf("contagem errada: %+v", r)
	}
	if len(r.Problemas) != 0 {
		t.Fatalf("rodada limpa acusou problemas: %v", r.Problemas)
	}
}

// Cada defeito que o ensaio procura, plantado de propósito: a auditoria tem de acusar cada um.
func TestAcusaCadaDefeito(t *testing.T) {
	casos := []struct {
		nome    string
		plantar func(t *testing.T, rodada string) Esperado
		acusa   string
	}{
		{"inscrição perdida", func(t *testing.T, rodada string) Esperado {
			enviada(t, rodada)
			return Esperado{Enviadas: 2} // o candidato viu duas; o banco guardou uma
		}, "INSCRIÇÃO PERDIDA"},
		{"pagamento perdido", func(t *testing.T, rodada string) Esperado {
			enviada(t, rodada)
			return Esperado{Pagas: 1} // a tela mostrou PAGA, o banco não tem
		}, "PAGAMENTO PERDIDO"},
		{"cobrança duplicada", func(t *testing.T, rodada string) Esperado {
			i := enviada(t, rodada)
			cobranca(t, i, "pix", "EXPIRADA") // o índice só barra duas PENDENTES; esta passaria
			cobranca(t, i, "pix", "PENDENTE")
			return Esperado{}
		}, "COBRANÇA DUPLICADA"},
		{"cobrada duas vezes", func(t *testing.T, rodada string) Esperado {
			// Boleto pago duas vezes com os dois pagamentos confirmados, em vez de o segundo virar
			// reembolso. (Duas cobranças PAGA na mesma inscrição o índice cobranca_uma_paga_por_inscricao
			// já barra; o que sobra é o mesmo boleto contado duas vezes.)
			i := enviada(t, rodada)
			boleto := cobranca(t, i, "boleto", "PAGA")
			recebido(t, boleto, pagamento.Confirmado)
			recebido(t, boleto, pagamento.Confirmado)
			confirmar(t, i)
			return Esperado{}
		}, "COBRADA DUAS VEZES"},
		{"pago a mais sem reembolso", func(t *testing.T, rodada string) Esperado {
			i := enviada(t, rodada)
			pagaDireito(t, i)
			recebido(t, cobranca(t, i, "boleto", "PAGA_EM_DUPLICIDADE"), pagamento.EmDuplicidade)
			return Esperado{}
		}, "PAGO A MAIS SEM DEVOLUÇÃO"},
		{"devolução de pagamento válido", func(t *testing.T, rodada string) Esperado {
			i := enviada(t, rodada)
			pix := cobranca(t, i, "pix", "PAGA")
			devolvido(t, recebido(t, pix, pagamento.Confirmado), 8000, pagamento.DevolucaoConcluida)
			confirmar(t, i)
			return Esperado{}
		}, "DEVOLUÇÃO DE PAGAMENTO VÁLIDO"},
		{"devolução acima do pago", func(t *testing.T, rodada string) Esperado {
			i := enviada(t, rodada)
			pagaDireito(t, i)
			devolvido(t, recebido(t, cobranca(t, i, "boleto", "PAGA_EM_DUPLICIDADE"), pagamento.EmDuplicidade),
				8001, pagamento.DevolucaoPendente)
			return Esperado{}
		}, "DEVOLUÇÃO ACIMA DO PAGO"},
		{"inscrição viva em certame cancelado", func(t *testing.T, rodada string) Esperado {
			cancelarCertameDe(t, enviada(t, rodada)) // marcado, mas a inscrição não passou pela máquina
			return Esperado{}
		}, "INSCRIÇÃO VIVA EM CERTAME CANCELADO"},
		{"pagamento retido em certame cancelado", func(t *testing.T, rodada string) Esperado {
			i := enviada(t, rodada)
			pagaDireito(t, i)
			cancelarCertameDe(t, i)
			_, err := pool.Exec(context.Background(), `UPDATE inscricao SET estado = 'CANCELADA' WHERE id = $1`, i)
			must(t, err)
			return Esperado{}
		}, "PAGAMENTO RETIDO EM CERTAME CANCELADO"},
		{"devolução recusada", func(t *testing.T, rodada string) Esperado {
			i := enviada(t, rodada)
			pagaDireito(t, i)
			devolvido(t, recebido(t, cobranca(t, i, "boleto", "PAGA_EM_DUPLICIDADE"), pagamento.EmDuplicidade),
				8000, pagamento.DevolucaoRecusada)
			return Esperado{}
		}, "DEVOLUÇÃO RECUSADA"},
		{"paga sem recebimento", func(t *testing.T, rodada string) Esperado {
			confirmar(t, enviada(t, rodada))
			return Esperado{}
		}, "PAGA SEM RECEBIMENTO"},
		{"trilha adulterada", func(t *testing.T, rodada string) Esperado {
			i := enviada(t, rodada)
			// Um superusuário desliga a imutabilidade e reescreve quem criou a inscrição.
			ctx := context.Background()
			tx, err := pool.Begin(ctx)
			must(t, err)
			defer tx.Rollback(ctx)
			for _, s := range []string{
				`ALTER TABLE inscricao_evento DISABLE TRIGGER inscricao_evento_imutavel`,
				`UPDATE inscricao_evento SET ator_id = 'outro' WHERE inscricao_id = ` + strconv.FormatInt(i, 10) + ` AND evento = 'CRIAR'`,
				`ALTER TABLE inscricao_evento ENABLE TRIGGER inscricao_evento_imutavel`,
			} {
				_, err := tx.Exec(ctx, s)
				must(t, err)
			}
			must(t, tx.Commit(ctx))
			return Esperado{}
		}, "TRILHA ADULTERADA"},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			rodada := strings.ReplaceAll(c.nome, " ", "")
			esp := c.plantar(t, rodada)
			r, err := Auditar(context.Background(), pool, rodada, esp)
			must(t, err)
			if !strings.Contains(strings.Join(r.Problemas, "|"), c.acusa) {
				t.Fatalf("não acusou %q; problemas = %v", c.acusa, r.Problemas)
			}
		})
	}
}

func TestRodadaVaziaRecusada(t *testing.T) {
	if pool == nil {
		t.Skip("sem banco de teste")
	}
	if _, err := Auditar(context.Background(), pool, "", Esperado{}); err == nil {
		t.Fatal("rodada vazia auditaria o banco inteiro")
	}
}
