package pagamento

import (
	"context"
	"sync"
	"testing"

	"lisura/internal/fila"
	"lisura/internal/inscricao"
)

// devolucao lê a única devolução da inscrição do cenário.
func (c cenario) devolucao(t *testing.T) (estado, motivo, metodo string, pspID *string) {
	t.Helper()
	must(t, pool.QueryRow(context.Background(), `
		SELECT estado, motivo, metodo_origem, psp_devolucao_id FROM devolucao WHERE inscricao_id = $1`,
		c.inscricao).Scan(&estado, &motivo, &metodo, &pspID))
	return
}

func (c cenario) cpf(t *testing.T) string {
	t.Helper()
	var cpf string
	must(t, pool.QueryRow(context.Background(), `SELECT cpf FROM candidato WHERE id = $1`, c.candidato).Scan(&cpf))
	return cpf
}

// Pix e boleto pagos: o segundo dinheiro volta por Pix para a chave CPF do candidato, uma vez só,
// por mais que a conciliação passe de novo.
func TestPagoEmDobroVolta(t *testing.T) {
	c := novoCenario(t)
	pix, boleto := c.cobrar(t, "pix"), c.cobrar(t, "boleto")
	c.pagar(t, pix)
	c.pagar(t, boleto)
	for range 3 {
		c.conciliar(t)
		c.esvaziar(t)
	}
	estado, motivo, metodo, pspID := c.devolucao(t)
	if estado != DevolucaoConcluida || motivo != EmDuplicidade || metodo != "boleto" || pspID == nil {
		t.Fatalf("devolução: %s %s %s %v", estado, motivo, metodo, pspID)
	}
	if n := c.sim.DevolucoesFeitas.Load(); n != 1 {
		t.Fatalf("%d devoluções no prestador; esperado 1", n)
	}
	if c.estado(t) != inscricao.Paga {
		t.Fatalf("a devolução mexeu na inscrição: %s", c.estado(t))
	}
}

// Pix pago depois de a inscrição expirar (o aviso se perdeu e a conciliação chegou tarde): volta
// para quem pagou, sem chave nenhuma.
func TestPixDepoisDaExpiracaoVoltaParaQuemPagou(t *testing.T) {
	c := novoCenario(t)
	ctx := context.Background()
	pix := c.cobrar(t, "pix")
	c.pagar(t, pix) // o aviso não chega: o simulador do cenário não avisa
	_, err := pool.Exec(ctx, `UPDATE certame_fase SET abre_em = now() - interval '30 days', fecha_em = now() - interval '5 days'
		WHERE fase = 'pagamento' AND certame_id = (SELECT certame_id FROM inscricao WHERE id = $1)`, c.inscricao)
	must(t, err)
	_, err = pool.Exec(ctx, `UPDATE cobranca SET vence_em = now() - interval '5 days' WHERE id = $1`, pix.ID)
	must(t, err)
	if n, err := ExpirarVencidas(ctx, pool, MargemCompensacao); err != nil || n != 1 {
		t.Fatalf("expiradas=%d err=%v", n, err)
	}
	c.conciliar(t)
	c.esvaziar(t)
	if estado, motivo, metodo, _ := c.devolucao(t); estado != DevolucaoConcluida || motivo != AposExpiracao || metodo != "pix" {
		t.Fatalf("devolução: %s %s %s", estado, motivo, metodo)
	}
	if c.estado(t) != inscricao.Expirada {
		t.Fatalf("a devolução ressuscitou a inscrição: %s", c.estado(t))
	}
}

// A resposta do prestador se perde depois de a devolução sair: a fila repete e o prestador reconhece
// o pedido. Uma devolução, não duas.
func TestRespostaPerdidaNaoDevolveDuasVezes(t *testing.T) {
	c := novoCenario(t)
	c.pagar(t, c.cobrar(t, "pix"))
	c.pagar(t, c.cobrar(t, "boleto"))
	c.sim.FalharDevolucoes.Store(1)
	c.conciliar(t)
	c.esvaziar(t)
	if estado, _, _, _ := c.devolucao(t); estado != DevolucaoPendente {
		t.Fatalf("com a resposta perdida, a devolução ficou %s; esperado PENDENTE", estado)
	}
	_, err := pool.Exec(context.Background(), `UPDATE tarefa SET executar_em = now() WHERE tipo = $1 AND estado = 'PENDENTE'`, TarefaDevolver)
	must(t, err)
	c.esvaziar(t)
	if estado, _, _, _ := c.devolucao(t); estado != DevolucaoConcluida {
		t.Fatalf("a repetição deixou a devolução %s", estado)
	}
	if n := c.sim.DevolucoesFeitas.Load(); n != 1 {
		t.Fatalf("%d devoluções no prestador: a repetição devolveu de novo", n)
	}
}

// O mesmo pagamento a mais chegando 20 vezes ao mesmo tempo (aviso repetido + conciliação): uma
// devolução e uma tarefa.
func TestPagamentoAMaisRepetidoUmaDevolucao(t *testing.T) {
	c := novoCenario(t)
	pix, boleto := c.cobrar(t, "pix"), c.cobrar(t, "boleto")
	c.pagar(t, pix)
	c.conciliar(t) // o Pix confirma; o boleto pago depois é o dinheiro a mais
	aMais := c.pagar(t, boleto)
	var wg sync.WaitGroup
	for i := range 20 {
		origem := "webhook"
		if i%2 == 0 {
			origem = "conciliacao"
		}
		wg.Go(func() {
			if _, err := RegistrarPagamento(context.Background(), pool, aMais, origem); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if n := contar(t, `SELECT count(*) FROM devolucao WHERE inscricao_id = $1`, c.inscricao); n != 1 {
		t.Fatalf("%d devoluções para um pagamento a mais", n)
	}
	if n := contar(t, `SELECT count(*) FROM tarefa t JOIN devolucao d ON t.chave = d.id::text
		WHERE t.tipo = $1 AND d.inscricao_id = $2`, TarefaDevolver, c.inscricao); n != 1 {
		t.Fatalf("%d tarefas de devolução", n)
	}
}

// Boleto pago a mais, e o CPF do candidato não é chave Pix: o dinheiro não vai para lugar nenhum
// até haver uma chave dele. Não é recusa: fica AGUARDANDO_CHAVE.
func TestSemChaveCPFAguardaChave(t *testing.T) {
	c := novoCenario(t)
	c.sim.SemChaveCPF(c.cpf(t))
	c.pagar(t, c.cobrar(t, "pix"))
	c.pagar(t, c.cobrar(t, "boleto"))
	c.conciliar(t)
	c.esvaziar(t)
	if estado, _, _, _ := c.devolucao(t); estado != DevolucaoAguardandoChave {
		t.Fatalf("devolução %s; esperado AGUARDANDO_CHAVE", estado)
	}
	if n := c.sim.DevolucoesFeitas.Load(); n != 0 {
		t.Fatalf("%d devoluções saíram sem chave", n)
	}
}

// A tarefa abandonada (prestador instável além das tentativas) não abandona o dinheiro: a rotina
// devolve a devolução para a fila, e ela sai, uma vez.
func TestDevolucaoAbandonadaVoltaParaAFila(t *testing.T) {
	c := novoCenario(t)
	c.pagar(t, c.cobrar(t, "pix"))
	c.pagar(t, c.cobrar(t, "boleto"))
	c.sim.FalharDevolucoes.Store(1)
	c.conciliar(t)
	impaciente := &fila.Trabalhador{Pool: pool, Manipuladores: c.trabalhador.Manipuladores, MaxTentativas: 1}
	must(t, impaciente.Esvaziar(context.Background()))
	if n := contar(t, `SELECT count(*) FROM tarefa t JOIN devolucao d ON t.chave = d.id::text
		WHERE t.tipo = $1 AND d.inscricao_id = $2 AND t.estado = 'FALHOU'`, TarefaDevolver, c.inscricao); n != 1 {
		t.Fatalf("a tarefa não foi abandonada (%d FALHOU); o teste não testa nada", n)
	}

	n, err := RetomarDevolucoes(context.Background(), pool)
	must(t, err)
	if n < 1 {
		t.Fatal("a rotina não retomou a devolução abandonada")
	}
	c.esvaziar(t)
	if estado, _, _, _ := c.devolucao(t); estado != DevolucaoConcluida {
		t.Fatalf("retomada, a devolução ficou %s", estado)
	}
	if n := c.sim.DevolucoesFeitas.Load(); n != 1 {
		t.Fatalf("%d devoluções no prestador", n)
	}
}
