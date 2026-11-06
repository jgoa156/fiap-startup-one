package arrecadacao

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"lisura/internal/conta"
	"lisura/internal/dbtest"
	"lisura/internal/fila"
	"lisura/internal/inscricao"
	"lisura/internal/notificacao"
	"lisura/internal/pagamento"
	"lisura/internal/psp"
	"lisura/internal/publicacao"
	"lisura/internal/seed"
)

var pool *pgxpool.Pool

func TestMain(m *testing.M) {
	p, err := dbtest.Pool(context.Background(), "t_arrecadacao")
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

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

var seq atomic.Int64

// certameDeTeste é um certame só do teste, com o prestador simulado e a fila que cria a cobrança e
// faz a devolução, como no servidor.
type certameDeTeste struct {
	id          int64
	cargo       int64
	cidade      int64
	sim         *psp.Simulador
	prestador   psp.Cliente
	trabalhador *fila.Trabalhador
}

func novo(t *testing.T) *certameDeTeste {
	t.Helper()
	if pool == nil {
		t.Skip("sem banco de teste")
	}
	cert, err := seed.CriarCertame(context.Background(), pool, seed.Padrao())
	must(t, err)
	sim := &psp.Simulador{}
	srv := httptest.NewServer(sim.Handler())
	t.Cleanup(srv.Close)
	c := &certameDeTeste{id: cert.ID, cargo: cert.Cargos[0], cidade: cert.Cidades[0], sim: sim,
		prestador: psp.Cliente{URL: srv.URL, HTTP: psp.NovoHTTP(5 * time.Second)}}
	c.trabalhador = &fila.Trabalhador{Pool: pool, Manipuladores: map[string]fila.Manipulador{
		pagamento.TarefaCriarNoPrestador: pagamento.CriarNoPrestador(c.prestador, pool),
		pagamento.TarefaDevolver:         pagamento.Devolver(c.prestador, pool),
		notificacao.Tarefa:               notificacao.Notificar(&notificacao.Memoria{}, pool),
		publicacao.TarefaCartao:          func(context.Context, json.RawMessage) error { return nil },
		publicacao.TarefaCertame:         func(context.Context, json.RawMessage) error { return nil },
		publicacao.TarefaRelacao:         func(context.Context, json.RawMessage) error { return nil },
	}}
	t.Cleanup(func() { c.trabalhador.Esvaziar(context.Background()) })
	return c
}

func (c *certameDeTeste) esvaziar(t *testing.T) {
	t.Helper()
	must(t, c.trabalhador.Esvaziar(context.Background()))
}

type candidata struct{ inscricao, candidato int64 }

func (c *certameDeTeste) inscrever(t *testing.T) candidata {
	t.Helper()
	ctx := context.Background()
	cand, err := conta.Criar(ctx, pool, conta.GerarCPF(seq.Add(1)+time.Now().UnixNano()%1_000_000*100), "Ana", "a@exemplo.test")
	must(t, err)
	must(t, conta.AtualizarDados(ctx, pool, cand, conta.Dados{Nome: "Ana", Email: "a@exemplo.test",
		DataNascimento: "1990-01-01", Municipio: "Campinas", UF: "SP"}))
	id, _, err := inscricao.Abrir(ctx, pool, c.id, cand, c.cargo, inscricao.Ator{Tipo: inscricao.Candidato, ID: "t"})
	must(t, err)
	must(t, inscricao.DefinirCidade(ctx, pool, id, cand, c.cidade))
	must(t, inscricao.AceitarDeclaracoes(ctx, pool, id, cand, true))
	_, err = inscricao.Finalizar(ctx, pool, id, cand)
	must(t, err)
	return candidata{id, cand}
}

// cobrar gera a cobrança e a cria no prestador; devolve o id dela no prestador.
func (c *certameDeTeste) cobrar(t *testing.T, a candidata, metodo string) string {
	t.Helper()
	cob, err := pagamento.Cobrar(context.Background(), pool, a.inscricao, a.candidato, metodo)
	must(t, err)
	c.esvaziar(t)
	var pspID string
	must(t, pool.QueryRow(context.Background(), `SELECT psp_id FROM cobranca WHERE id = $1`, cob.ID).Scan(&pspID))
	return pspID
}

// pagarNoBanco é o candidato pagando; devolve o pagamento como o extrato o mostra, sem registrá-lo.
func (c *certameDeTeste) pagarNoBanco(t *testing.T, cobrancaPSP string) psp.Pagamento {
	t.Helper()
	must(t, c.prestador.SimularPagamento(context.Background(), cobrancaPSP))
	ex, err := c.prestador.Extrato(context.Background(), time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	must(t, err)
	return ex[len(ex)-1]
}

func (c *certameDeTeste) registrar(t *testing.T, p psp.Pagamento) {
	t.Helper()
	_, err := pagamento.RegistrarPagamento(context.Background(), pool, p, "webhook")
	must(t, err)
	c.esvaziar(t) // a devolução, se houver
}

func (c *certameDeTeste) cpf(t *testing.T, a candidata) string {
	t.Helper()
	var cpf string
	must(t, pool.QueryRow(context.Background(), `SELECT cpf FROM candidato WHERE id = $1`, a.candidato).Scan(&cpf))
	return cpf
}

func temProblema(c Conferencia, prefixo string) bool {
	for _, p := range c.Problemas {
		if strings.HasPrefix(p, prefixo) {
			return true
		}
	}
	return false
}

// Um certame com cada destino do dinheiro: confirmado; pago em dobro e devolvido; pago com valor
// diferente; pago em dobro com a devolução esperando chave. A conta fecha centavo por centavo, os
// recortes somam o total e o extrato do prestador confere.
func TestAContaFecha(t *testing.T) {
	ctx := context.Background()
	c := novo(t)
	a, b, d, e := c.inscrever(t), c.inscrever(t), c.inscrever(t), c.inscrever(t)

	c.registrar(t, c.pagarNoBanco(t, c.cobrar(t, a, "pix"))) // confirmado

	bPix, bBoleto := c.cobrar(t, b, "pix"), c.cobrar(t, b, "boleto")
	c.registrar(t, c.pagarNoBanco(t, bPix))
	c.registrar(t, c.pagarNoBanco(t, bBoleto)) // a mais: devolvido para a chave CPF

	div := c.pagarNoBanco(t, c.cobrar(t, d, "pix"))
	c.sim.Adulterar(div.ID, div.ValorCentavos-1)
	div.ValorCentavos--
	c.registrar(t, div) // valor divergente: não confirma, não devolve, alguém decide

	c.sim.SemChaveCPF(c.cpf(t, e))
	ePix, eBoleto := c.cobrar(t, e, "pix"), c.cobrar(t, e, "boleto")
	c.registrar(t, c.pagarNoBanco(t, ePix))
	c.registrar(t, c.pagarNoBanco(t, eBoleto)) // a mais, devolução aguardando chave

	p, err := Gerar(ctx, pool, c.id)
	must(t, err)
	v := int64(div.ValorCentavos + 1) // a taxa do cargo
	quer := map[string]Total{
		"entrou": {6, 6*v - 1}, "saiu": {1, v}, "a devolver": {1, v}, "em divergência": {1, v - 1}, "válidas": {3, 3 * v},
	}
	got := map[string]Total{"entrou": p.Entrou, "saiu": p.Saiu, "a devolver": p.ADevolver, "em divergência": p.EmDivergencia, "válidas": p.Validas}
	for k, q := range quer {
		if got[k] != q {
			t.Errorf("%s: %+v, esperado %+v", k, got[k], q)
		}
	}
	if p.Diferenca != 0 {
		t.Fatalf("a conta não fecha: diferença %s", BRL(p.Diferenca))
	}
	for nome, linhas := range map[string][]Linha{"dia": p.PorDia, "cargo": p.PorCargo, "cidade": p.PorCidade} {
		var soma Linha
		for _, l := range linhas {
			soma.Entrou.N += l.Entrou.N
			soma.Entrou.Centavos += l.Entrou.Centavos
			soma.Saiu.Centavos += l.Saiu.Centavos
			soma.Validas.Centavos += l.Validas.Centavos
		}
		if soma.Entrou != p.Entrou || soma.Saiu.Centavos != p.Saiu.Centavos || soma.Validas.Centavos != p.Validas.Centavos {
			t.Errorf("por %s não soma o total: %+v", nome, soma)
		}
	}
	if !strings.Contains(p.Markdown(), "**A conta fecha:**") {
		t.Fatalf("o relatório não diz que a conta fecha:\n%s", p.Markdown())
	}

	conf, err := Conferir(ctx, pool, c.prestador, c.id)
	must(t, err)
	if len(conf.Problemas) != 0 || conf.Pagamentos != 6 || conf.Devolucoes != 1 {
		t.Fatalf("conferência: %+v", conf)
	}
}

// A devolução apagada daqui (o dinheiro saiu no prestador e a conta deixou de mostrar): a conta não
// fecha, e a devolução do prestador aparece como sem registro.
func TestContaQueNaoFechaEAcusada(t *testing.T) {
	ctx := context.Background()
	c := novo(t)
	b := c.inscrever(t)
	bPix, bBoleto := c.cobrar(t, b, "pix"), c.cobrar(t, b, "boleto")
	c.registrar(t, c.pagarNoBanco(t, bPix))
	c.registrar(t, c.pagarNoBanco(t, bBoleto))
	_, err := pool.Exec(ctx, `DELETE FROM devolucao WHERE inscricao_id = $1`, b.inscricao)
	must(t, err)

	p, err := Gerar(ctx, pool, c.id)
	must(t, err)
	if p.Diferenca == 0 || !strings.Contains(p.Markdown(), "A CONTA NÃO FECHA") {
		t.Fatalf("a devolução apagada não desequilibrou a conta: %+v", p)
	}
	conf, err := Conferir(ctx, pool, c.prestador, c.id)
	must(t, err)
	if !temProblema(conf, "A CONTA NÃO FECHA") || !temProblema(conf, "DEVOLUÇÃO DO PRESTADOR SEM REGISTRO AQUI") {
		t.Fatalf("conferência: %v", conf.Problemas)
	}
}

// O dia fechado não muda: nem por escrita direta (o banco recusa), nem pelo boleto pago naquele dia e
// informado depois (entra no dia do registro). Só uma adulteração o muda, e a conferência acusa.
func TestDiaFechadoNaoMuda(t *testing.T) {
	ctx := context.Background()
	c := novo(t)
	a, b := c.inscrever(t), c.inscrever(t)
	c.registrar(t, c.pagarNoBanco(t, c.cobrar(t, a, "pix")))
	_, err := pool.Exec(ctx, `UPDATE pagamento_recebido p SET recebido_em = recebido_em - interval '1 day'
		FROM cobranca b WHERE b.id = p.cobranca_id AND b.inscricao_id = $1`, a.inscricao)
	must(t, err)
	var ontem, hoje string
	must(t, pool.QueryRow(ctx, `SELECT ((now() AT TIME ZONE 'America/Sao_Paulo')::date - 1)::text,
		(now() AT TIME ZONE 'America/Sao_Paulo')::date::text`).Scan(&ontem, &hoje))

	if _, _, err := Fechar(ctx, pool, c.id, hoje, "ana"); !errors.Is(err, ErrDiaAberto) {
		t.Fatalf("fechou o dia de hoje: %v", err)
	}
	fs, err := FecharAteOntem(ctx, pool, c.id, "ana")
	must(t, err)
	if len(fs) != 1 || fs[0].Dia != ontem || fs[0].Entrou.N != 1 {
		t.Fatalf("fechamentos: %+v", fs)
	}
	if _, criado, err := Fechar(ctx, pool, c.id, ontem, "bia"); err != nil || criado {
		t.Fatalf("fechar de novo regravou (criado=%v, %v)", criado, err)
	}
	for _, sql := range []string{
		`UPDATE arrecadacao_fechamento SET entrou_centavos = 0 WHERE certame_id = $1`,
		`DELETE FROM arrecadacao_fechamento WHERE certame_id = $1`,
	} {
		if _, err := pool.Exec(ctx, sql, c.id); err == nil {
			t.Fatalf("o banco aceitou mexer no dia fechado: %s", sql)
		}
	}

	// Boleto pago ontem, informado hoje: entra hoje, e ontem continua como foi fechado.
	tarde := c.pagarNoBanco(t, c.cobrar(t, b, "boleto"))
	tarde.PagoEm = time.Now().Add(-24 * time.Hour)
	c.registrar(t, tarde)
	conf, err := Conferir(ctx, pool, c.prestador, c.id)
	must(t, err)
	if len(conf.Problemas) != 0 || conf.DiasFechados != 1 {
		t.Fatalf("o pagamento informado tarde mexeu no dia fechado: %+v", conf)
	}

	// Alguém empurra esse pagamento para o dia fechado.
	_, err = pool.Exec(ctx, `UPDATE pagamento_recebido SET recebido_em = recebido_em - interval '1 day' WHERE psp_pagamento_id = $1`, tarde.ID)
	must(t, err)
	conf, err = Conferir(ctx, pool, c.prestador, c.id)
	must(t, err)
	if !temProblema(conf, "DIA "+ontem+" FECHADO MUDOU") {
		t.Fatalf("a adulteração do dia fechado passou: %v", conf.Problemas)
	}
}

// Contra o extrato do prestador: o pagamento que só existe aqui, o que só existe lá e o que tem outro
// valor lá.
func TestConferirContraOPrestador(t *testing.T) {
	ctx := context.Background()
	c := novo(t)
	a, b, d := c.inscrever(t), c.inscrever(t), c.inscrever(t)

	// Só aqui: um pagamento que o prestador não fez (o aviso forjado que passou, a linha inserida à mão).
	cobA := c.cobrar(t, a, "pix")
	var refA int64
	must(t, pool.QueryRow(ctx, `SELECT id FROM cobranca WHERE psp_id = $1`, cobA).Scan(&refA))
	var taxa int
	must(t, pool.QueryRow(ctx, `SELECT valor_centavos FROM cobranca WHERE id = $1`, refA).Scan(&taxa))
	c.registrar(t, psp.Pagamento{ID: "pag_fantasma_" + strconv.FormatInt(refA, 10), CobrancaID: cobA, Referencia: refA,
		ValorCentavos: taxa, PagoEm: time.Now()})

	// Só lá: pago no banco, e nem aviso nem conciliação chegaram ainda.
	c.pagarNoBanco(t, c.cobrar(t, b, "pix"))

	// Outro valor lá: o extrato do prestador mudou depois do registro.
	p := c.pagarNoBanco(t, c.cobrar(t, d, "pix"))
	c.registrar(t, p)
	c.sim.Adulterar(p.ID, p.ValorCentavos+100)

	conf, err := Conferir(ctx, pool, c.prestador, c.id)
	must(t, err)
	for _, quer := range []string{"AQUI, SEM REGISTRO NO PRESTADOR", "NO PRESTADOR, SEM REGISTRO AQUI", "VALOR DIFERENTE DO PRESTADOR"} {
		if !temProblema(conf, quer) {
			t.Errorf("não acusou %q; problemas: %v", quer, conf.Problemas)
		}
	}
}

func TestBRL(t *testing.T) {
	for c, quer := range map[int64]string{0: "R$ 0,00", 5: "R$ 0,05", 100: "R$ 1,00", 123456789: "R$ 1.234.567,89", -12345: "-R$ 123,45"} {
		if got := BRL(c); got != quer {
			t.Errorf("BRL(%d) = %q, esperado %q", c, got, quer)
		}
	}
}
