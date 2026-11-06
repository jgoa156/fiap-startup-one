package pagamento

import (
	"context"
	"errors"
	"testing"
	"time"

	"lisura/internal/conta"
	"lisura/internal/inscricao"
)

// vizinho abre outra inscrição no certame do cenário, pronta para pagar (o mesmo prestador e a mesma
// fila do cenário). Devolve um cenário que só troca a inscrição e o candidato.
func (c cenario) vizinho(t *testing.T) cenario {
	t.Helper()
	ctx := context.Background()
	var certID, cargo, cidade int64
	must(t, pool.QueryRow(ctx, `SELECT certame_id, cargo_id, cidade_prova_id FROM inscricao WHERE id = $1`,
		c.inscricao).Scan(&certID, &cargo, &cidade))
	n := seq.Add(1)
	cand, err := conta.Criar(ctx, pool, conta.GerarCPF(n+time.Now().UnixNano()%1_000_000*100), "Vizinha", "v@exemplo.test")
	must(t, err)
	must(t, conta.AtualizarDados(ctx, pool, cand, conta.Dados{Nome: "Vizinha", Email: "v@exemplo.test",
		DataNascimento: "1991-02-02", Municipio: "Campinas", UF: "SP"}))
	id, _, err := inscricao.Abrir(ctx, pool, certID, cand, cargo, inscricao.Ator{Tipo: inscricao.Candidato, ID: "t"})
	must(t, err)
	must(t, inscricao.DefinirCidade(ctx, pool, id, cand, cidade))
	must(t, inscricao.AceitarDeclaracoes(ctx, pool, id, cand, true))
	v := c
	v.inscricao, v.candidato = id, cand
	return v
}

func (c cenario) finalizar(t *testing.T) {
	t.Helper()
	_, err := inscricao.Finalizar(context.Background(), pool, c.inscricao, c.candidato)
	must(t, err)
	c.esvaziar(t)
}

func (c cenario) certame(t *testing.T) int64 {
	t.Helper()
	var id int64
	must(t, pool.QueryRow(context.Background(), `SELECT certame_id FROM inscricao WHERE id = $1`, c.inscricao).Scan(&id))
	return id
}

// O certame é anulado com inscrições em todos os pontos do caminho: rascunho, aguardando o Pix,
// paga por Pix, paga por boleto. Toda inscrição viva vai a CANCELADA com trilha; cada pagamento volta
// uma vez; rodar de novo não faz nada; o Pix pago depois da anulação também volta; e o certame não
// aceita mais inscrição nem cobrança.
func TestCancelarCertameDevolveTudoUmaVez(t *testing.T) {
	ctx := context.Background()
	aguardando := novoCenario(t)
	pixPendente := aguardando.cobrar(t, "pix") // gerou o Pix e ainda não pagou
	pagaPix := aguardando.vizinho(t)
	pagaPix.finalizar(t)
	pagaPix.pagar(t, pagaPix.cobrar(t, "pix"))
	pagaBoleto := aguardando.vizinho(t)
	pagaBoleto.finalizar(t)
	pagaBoleto.pagar(t, pagaBoleto.cobrar(t, "boleto"))
	rascunho := aguardando.vizinho(t)
	aguardando.conciliar(t)
	if pagaPix.estado(t) != inscricao.Paga || pagaBoleto.estado(t) != inscricao.Paga {
		t.Fatalf("preparação: %s, %s", pagaPix.estado(t), pagaBoleto.estado(t))
	}
	cert := aguardando.certame(t)

	res, err := CancelarCertame(ctx, pool, cert, "ana", "ANULADO_POR_DECISAO_JUDICIAL")
	must(t, err)
	if res.Inscricoes != 4 || res.Devolucoes != 2 {
		t.Fatalf("cancelamento: %+v; esperado 4 inscrições e 2 devoluções", res)
	}
	for _, x := range []cenario{aguardando, pagaPix, pagaBoleto, rascunho} {
		if x.estado(t) != inscricao.Cancelada {
			t.Errorf("inscrição %d ficou %s", x.inscricao, x.estado(t))
		}
	}
	if n := contar(t, `SELECT count(*) FROM inscricao_evento e JOIN inscricao i ON i.id = e.inscricao_id
		WHERE i.certame_id = $1 AND e.evento = 'CANCELAR_CERTAME' AND e.ator_tipo = 'analista' AND e.ator_id = 'ana'
		AND e.motivo_codigo = 'ANULADO_POR_DECISAO_JUDICIAL'`, cert); n != 4 {
		t.Fatalf("%d cancelamentos na trilha com analista e motivo; esperado 4", n)
	}
	aguardando.esvaziar(t)
	if n := contar(t, `SELECT count(*) FROM devolucao d JOIN inscricao i ON i.id = d.inscricao_id
		WHERE i.certame_id = $1 AND d.estado = 'CONCLUIDA' AND d.motivo = 'CERTAME_CANCELADO'`, cert); n != 2 {
		t.Fatalf("%d devoluções concluídas; esperado 2 (Pix para quem pagou, boleto para a chave CPF)", n)
	}

	// De novo: nada novo.
	res, err = CancelarCertame(ctx, pool, cert, "ana", "ANULADO_POR_DECISAO_JUDICIAL")
	must(t, err)
	aguardando.esvaziar(t)
	if res.Inscricoes != 0 || res.Devolucoes != 0 || aguardando.sim.DevolucoesFeitas.Load() != 2 {
		t.Fatalf("a segunda chamada fez %+v; %d devoluções no prestador", res, aguardando.sim.DevolucoesFeitas.Load())
	}

	// O Pix gerado antes da anulação ainda é pagável no banco: o dinheiro entra e volta.
	aguardando.pagar(t, pixPendente)
	aguardando.conciliar(t)
	aguardando.esvaziar(t)
	if n := aguardando.sim.DevolucoesFeitas.Load(); n != 3 || aguardando.estado(t) != inscricao.Cancelada {
		t.Fatalf("Pix pago depois da anulação: %d devoluções, inscrição %s", n, aguardando.estado(t))
	}

	// As portas fechadas.
	if _, err := Cobrar(ctx, pool, aguardando.inscricao, aguardando.candidato, "boleto"); !errors.Is(err, inscricao.ErrCertameCancelado) {
		t.Fatalf("cobrança em certame cancelado: %v", err)
	}
	var cargo int64
	must(t, pool.QueryRow(ctx, `SELECT cargo_id FROM inscricao WHERE id = $1`, aguardando.inscricao).Scan(&cargo))
	nova, err := conta.Criar(ctx, pool, conta.GerarCPF(seq.Add(1)+time.Now().UnixNano()%1_000_000*100), "Atrasada", "a@exemplo.test")
	must(t, err)
	if _, _, err := inscricao.Abrir(ctx, pool, cert, nova, cargo, inscricao.Ator{Tipo: inscricao.Candidato, ID: "t"}); !errors.Is(err, inscricao.ErrCertameCancelado) {
		t.Fatalf("inscrição nova em certame cancelado: %v", err)
	}
}

// O instante entre marcar o certame cancelado e cancelar cada inscrição: a inscrição ainda aguarda
// pagamento, e mesmo assim nenhuma cobrança nova nasce.
func TestCobrarNoMeioDoCancelamento(t *testing.T) {
	c := novoCenario(t)
	ctx := context.Background()
	_, err := pool.Exec(ctx, `UPDATE certame SET cancelado_em = now(), cancelamento_motivo = 'ANULADO_POR_DECISAO_JUDICIAL' WHERE id = $1`, c.certame(t))
	must(t, err)
	if c.estado(t) != inscricao.AguardandoPagamento {
		t.Fatalf("preparação: a inscrição está %s", c.estado(t))
	}
	antes := contar(t, `SELECT count(*) FROM cobranca WHERE inscricao_id = $1`, c.inscricao)
	if _, err := Cobrar(ctx, pool, c.inscricao, c.candidato, "pix"); !errors.Is(err, inscricao.ErrCertameCancelado) {
		t.Fatalf("cobrança com o certame já cancelado: %v", err)
	}
	if n := contar(t, `SELECT count(*) FROM cobranca WHERE inscricao_id = $1`, c.inscricao); n != antes {
		t.Fatalf("%d cobranças nasceram com o certame cancelado", n-antes)
	}
}

func TestCancelarCertameExigeMotivoPadronizado(t *testing.T) {
	c := novoCenario(t)
	for _, motivo := range []string{"", "anulado", "porque sim"} {
		if _, err := CancelarCertame(context.Background(), pool, c.certame(t), "ana", motivo); !errors.Is(err, ErrMotivoCancelamento) {
			t.Errorf("motivo %q aceito: %v", motivo, err)
		}
	}
	if c.estado(t) != inscricao.AguardandoPagamento {
		t.Fatal("o cancelamento recusado mexeu na inscrição")
	}
}
