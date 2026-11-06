package pagamento

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"lisura/internal/conta"
	"lisura/internal/dbtest"
	"lisura/internal/fila"
	"lisura/internal/inscricao"
	"lisura/internal/notificacao"
	"lisura/internal/psp"
	"lisura/internal/publicacao"
	"lisura/internal/seed"
)

// Testes da semana 3: os defeitos que um prestador de pagamento real produz, provocados pelo
// simulador, contra o Postgres real. O entregável é "nenhuma cobrança duplicada sob repetição".

var pool *pgxpool.Pool

func TestMain(m *testing.M) {
	p, err := dbtest.Pool(context.Background(), "t_pagamento")
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

// cenario é uma inscrição aguardando pagamento e um prestador simulado só dela (sem aviso
// automático: cada teste decide como o pagamento chega).
type cenario struct {
	inscricao, candidato int64
	taxa                 int
	sim                  *psp.Simulador
	prestador            psp.Cliente
	inicio               time.Time // a conciliação do teste só olha daqui para frente
	trabalhador          *fila.Trabalhador
}

func novoCenario(t *testing.T) cenario {
	t.Helper()
	if pool == nil {
		t.Skip("sem banco de teste")
	}
	inicio := time.Now()
	ctx := context.Background()
	cert, err := seed.CriarCertame(ctx, pool, seed.Padrao())
	must(t, err)
	cand, err := conta.Criar(ctx, pool, conta.GerarCPF(seq.Add(1)+time.Now().UnixNano()%1_000_000*100), "Teste", "t@exemplo.test")
	must(t, err)
	must(t, conta.AtualizarDados(ctx, pool, cand, conta.Dados{Nome: "Teste", Email: "t@exemplo.test",
		DataNascimento: "1990-01-01", Municipio: "Campinas", UF: "SP"}))
	id, _, err := inscricao.Abrir(ctx, pool, cert.ID, cand, cert.Cargos[0], inscricao.Ator{Tipo: inscricao.Candidato, ID: "t"})
	must(t, err)
	must(t, inscricao.DefinirCidade(ctx, pool, id, cand, cert.Cidades[0]))
	must(t, inscricao.AceitarDeclaracoes(ctx, pool, id, cand, true))
	_, err = inscricao.Finalizar(ctx, pool, id, cand)
	must(t, err)

	sim := &psp.Simulador{Segredo: []byte("s")}
	srv := httptest.NewServer(sim.Handler())
	t.Cleanup(srv.Close)
	prestador := psp.Cliente{URL: srv.URL, HTTP: &http.Client{Timeout: 5 * time.Second}}
	// O trabalhador da fila, como no servidor: cria a cobrança no prestador e publica o cartão.
	pub := &publicacao.Publicador{Pool: pool, Dir: t.TempDir(), Chave: []byte("k")}
	man := pub.Manipuladores()
	man[TarefaCriarNoPrestador] = CriarNoPrestador(prestador, pool)
	man[TarefaDevolver] = Devolver(prestador, pool)
	man[notificacao.Tarefa] = notificacao.Notificar(&notificacao.Memoria{}, pool)
	c := cenario{inscricao: id, candidato: cand, taxa: 8000, sim: sim, inicio: inicio, prestador: prestador,
		trabalhador: &fila.Trabalhador{Pool: pool, Manipuladores: man}}
	c.esvaziar(t) // o que a finalização enfileirou
	// O que o teste deixou na fila (a devolução de um pagamento a mais, por exemplo) roda aqui, contra
	// este prestador: senão o próximo teste a executaria contra o dele, que não conhece o pagamento.
	t.Cleanup(func() { c.trabalhador.Esvaziar(context.Background()) })
	return c
}

func (c cenario) esvaziar(t *testing.T) {
	t.Helper()
	must(t, c.trabalhador.Esvaziar(context.Background()))
}

// cobrar é o clique em "Pagar" seguido do trabalho da fila; devolve a cobrança já com o Pix ou a
// linha digitável do prestador.
func (c cenario) cobrar(t *testing.T, metodo string) Cobranca {
	t.Helper()
	cob, err := Cobrar(context.Background(), pool, c.inscricao, c.candidato, metodo)
	must(t, err)
	c.esvaziar(t)
	cob, err = ler(pool.QueryRow(context.Background(), `SELECT `+colunas+` FROM cobranca WHERE id = $1`, cob.ID))
	must(t, err)
	if cob.PSPID == nil {
		t.Fatalf("a fila não criou a cobrança %d no prestador", cob.ID)
	}
	return cob
}

// pagar é o candidato pagando no banco; devolve o pagamento como o extrato o mostra.
func (c cenario) pagar(t *testing.T, cob Cobranca) psp.Pagamento {
	t.Helper()
	must(t, c.prestador.SimularPagamento(context.Background(), *cob.PSPID))
	ext, err := c.prestador.Extrato(context.Background(), time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	must(t, err)
	return ext[len(ext)-1]
}

func (c cenario) conciliar(t *testing.T) Relatorio {
	t.Helper()
	r, err := Conciliar(context.Background(), pool, c.prestador, c.inicio, time.Now().Add(time.Hour))
	must(t, err)
	return r
}

func (c cenario) estado(t *testing.T) inscricao.Estado {
	t.Helper()
	var e inscricao.Estado
	must(t, pool.QueryRow(context.Background(), `SELECT estado FROM inscricao WHERE id = $1`, c.inscricao).Scan(&e))
	return e
}

func (c cenario) confirmacoes(t *testing.T) int {
	t.Helper()
	var n int
	must(t, pool.QueryRow(context.Background(),
		`SELECT count(*) FROM inscricao_evento WHERE inscricao_id = $1 AND evento = 'CONFIRMAR_PAGAMENTO'`, c.inscricao).Scan(&n))
	return n
}

func contar(t *testing.T, sql string, args ...any) int {
	t.Helper()
	var n int
	must(t, pool.QueryRow(context.Background(), sql, args...).Scan(&n))
	return n
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// Duplo clique, recarregamento e 50 abas ao mesmo tempo em "Pagar com Pix": uma cobrança só, aqui
// e no prestador.
func TestCobrancaConcorrenteNaoDuplica(t *testing.T) {
	c := novoCenario(t)
	const n = 50
	ids := make([]int64, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			cob, err := Cobrar(context.Background(), pool, c.inscricao, c.candidato, "pix")
			if err != nil {
				t.Error(err)
				return
			}
			ids[i] = cob.ID
		})
	}
	wg.Wait()
	for _, id := range ids {
		if id != ids[0] {
			t.Fatalf("cobranças diferentes devolvidas: %v", ids)
		}
	}
	if got := contar(t, `SELECT count(*) FROM cobranca WHERE inscricao_id = $1`, c.inscricao); got != 1 {
		t.Fatalf("%d cobranças no banco, esperava 1", got)
	}
	if got := contar(t, `SELECT count(*) FROM tarefa WHERE tipo = $1 AND chave = $2`, TarefaCriarNoPrestador, fmt.Sprint(ids[0])); got != 1 {
		t.Fatalf("%d tarefas para a cobrança, esperava 1", got)
	}
	// Quatro trabalhadores disputando a fila ao mesmo tempo.
	for range 4 {
		wg.Go(func() { c.esvaziar(t) })
	}
	wg.Wait()
	if got := c.sim.CobrancasCriadas.Load(); got != 1 {
		t.Fatalf("%d cobranças no prestador, esperava 1", got)
	}
}

// O clique responde sem esperar o prestador: a cobrança volta já, sem o Pix, e o Pix chega pela fila.
func TestCobrancaNaoEsperaOPrestador(t *testing.T) {
	c := novoCenario(t)
	c.sim.AtrasoCriacao = 2 * time.Second // prestador lento
	inicio := time.Now()
	cob, err := Cobrar(context.Background(), pool, c.inscricao, c.candidato, "pix")
	must(t, err)
	if d := time.Since(inicio); d > time.Second {
		t.Fatalf("Cobrar esperou o prestador: %s", d)
	}
	if cob.PSPID != nil || cob.PixCopiaCola != nil {
		t.Fatalf("Pix antes de o prestador responder: %+v", cob)
	}
	c.esvaziar(t)
	if got := contar(t, `SELECT count(*) FROM cobranca WHERE id = $1 AND pix_copia_cola IS NOT NULL`, cob.ID); got != 1 {
		t.Fatal("a fila não trouxe o Pix")
	}
}

// Prestador fora do ar além das tentativas do cliente: a tarefa fica para depois, com o erro anotado,
// e quando o prestador volta a cobrança sai — uma só.
func TestPrestadorForaDoArATarefaRepete(t *testing.T) {
	c := novoCenario(t)
	c.sim.FalharCriacoes.Store(3) // o cliente tenta 3 vezes; todas perdem a resposta
	cob, err := Cobrar(context.Background(), pool, c.inscricao, c.candidato, "boleto")
	must(t, err)
	c.esvaziar(t)
	var tentativas int
	var erro *string
	must(t, pool.QueryRow(context.Background(), `SELECT tentativas, ultimo_erro FROM tarefa WHERE tipo = $1 AND chave = $2`,
		TarefaCriarNoPrestador, fmt.Sprint(cob.ID)).Scan(&tentativas, &erro))
	if tentativas != 1 || erro == nil {
		t.Fatalf("tarefa depois da falha: tentativas=%d erro=%v", tentativas, erro)
	}
	// A espera passou: a tarefa roda de novo e o prestador já responde.
	_, err = pool.Exec(context.Background(), `UPDATE tarefa SET executar_em = now() WHERE tipo = $1 AND chave = $2`,
		TarefaCriarNoPrestador, fmt.Sprint(cob.ID))
	must(t, err)
	c.esvaziar(t)
	if got := contar(t, `SELECT count(*) FROM cobranca WHERE id = $1 AND linha_digitavel IS NOT NULL`, cob.ID); got != 1 {
		t.Fatal("a cobrança não saiu depois de o prestador voltar")
	}
	if got := c.sim.CobrancasCriadas.Load(); got != 1 {
		t.Fatalf("%d cobranças no prestador, esperava 1", got)
	}
}

// O prestador cria a cobrança mas a resposta se perde: o cliente repete com a mesma chave e recebe a
// mesma cobrança.
func TestRespostaPerdidaNaoCriaSegundaCobranca(t *testing.T) {
	c := novoCenario(t)
	c.sim.FalharCriacoes.Store(2)
	cob := c.cobrar(t, "boleto")
	if cob.PSPID == nil || cob.LinhaDigitavel == nil {
		t.Fatalf("cobrança sem dados do prestador: %+v", cob)
	}
	if got := c.sim.CobrancasCriadas.Load(); got != 1 {
		t.Fatalf("%d cobranças no prestador depois de 2 respostas perdidas, esperava 1", got)
	}
}

// O mesmo aviso chega 50 vezes ao mesmo tempo: o pagamento é aplicado uma vez.
func TestAvisoRepetidoConfirmaUmaVez(t *testing.T) {
	c := novoCenario(t)
	p := c.pagar(t, c.cobrar(t, "pix"))

	const n = 50
	var confirmados, repetidos atomic.Int32
	var wg sync.WaitGroup
	for range n {
		wg.Go(func() {
			r, err := RegistrarPagamento(context.Background(), pool, p, "webhook")
			switch {
			case err != nil:
				t.Error(err)
			case r == Confirmado:
				confirmados.Add(1)
			case r == JaProcessado:
				repetidos.Add(1)
			default:
				t.Errorf("resultado inesperado %s", r)
			}
		})
	}
	wg.Wait()
	if confirmados.Load() != 1 || repetidos.Load() != n-1 {
		t.Fatalf("confirmados=%d repetidos=%d", confirmados.Load(), repetidos.Load())
	}
	if e := c.estado(t); e != inscricao.Paga {
		t.Fatalf("inscrição %s, esperava PAGA", e)
	}
	if n := c.confirmacoes(t); n != 1 {
		t.Fatalf("%d confirmações na trilha", n)
	}
}

// O aviso nunca chega: a conciliação acha o pagamento no extrato e aplica. Rodar de novo não muda nada.
func TestAvisoPerdidoRecuperadoPelaConciliacao(t *testing.T) {
	c := novoCenario(t)
	c.pagar(t, c.cobrar(t, "pix")) // simulador sem WebhookURL: nenhum aviso sai

	if e := c.estado(t); e != inscricao.AguardandoPagamento {
		t.Fatalf("sem aviso, a inscrição já estava %s", e)
	}
	r := c.conciliar(t)
	if r.PorResultado[Confirmado] != 1 {
		t.Fatalf("conciliação: %+v", r)
	}
	if e := c.estado(t); e != inscricao.Paga {
		t.Fatalf("depois da conciliação, inscrição %s", e)
	}
	r = c.conciliar(t)
	if r.PorResultado[Confirmado] != 0 || r.PorResultado[JaProcessado] != 1 {
		t.Fatalf("segunda conciliação reaplicou: %+v", r)
	}
	if n := c.confirmacoes(t); n != 1 {
		t.Fatalf("%d confirmações na trilha", n)
	}
}

// A conciliação roda a cada minuto sobre 48 h de extrato: o que já entrou pelo aviso não pode custar
// uma transação de novo. Aqui, de dois pagamentos no extrato, um já entrou: só o outro é registrado.
func TestConciliacaoSoRegistraOQueFalta(t *testing.T) {
	c := novoCenario(t)
	cobPix, cobBoleto := c.cobrar(t, "pix"), c.cobrar(t, "boleto")
	pix := c.pagar(t, cobPix)
	c.pagar(t, cobBoleto)                                                                     // pagou também o boleto, e esse aviso se perdeu
	if _, err := RegistrarPagamento(context.Background(), pool, pix, "webhook"); err != nil { // o do Pix chegou
		t.Fatal(err)
	}

	var chamadas atomic.Int64
	original := registrar
	registrar = func(ctx context.Context, p *pgxpool.Pool, pg psp.Pagamento, origem string) (string, error) {
		chamadas.Add(1)
		return original(ctx, p, pg, origem)
	}
	t.Cleanup(func() { registrar = original })

	r := c.conciliar(t)
	if chamadas.Load() != 1 {
		t.Fatalf("a conciliação registrou %d pagamentos; só 1 faltava", chamadas.Load())
	}
	if r.PorResultado[JaProcessado] != 1 || r.PorResultado[EmDuplicidade] != 1 {
		t.Fatalf("relatório: %+v", r)
	}
}

// Com várias instâncias da API, só uma concilia por vez; as outras pulam a rodada.
func TestConciliacaoUmaInstanciaPorVez(t *testing.T) {
	c := novoCenario(t)
	ctx := context.Background()
	outra, err := pool.Acquire(ctx) // outra instância, no meio da conciliação dela
	must(t, err)
	defer outra.Release()
	var pegou bool
	must(t, outra.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, 0x4c697375).Scan(&pegou))
	if !pegou {
		t.Fatal("não conseguiu a trava para simular a outra instância")
	}
	if _, executou, err := ConciliarSozinho(ctx, pool, c.prestador, c.inicio, time.Now()); err != nil || executou {
		t.Fatalf("conciliou junto com a outra instância (executou=%v, err=%v)", executou, err)
	}
	_, err = outra.Exec(ctx, `SELECT pg_advisory_unlock($1)`, 0x4c697375)
	must(t, err)
	if _, executou, err := ConciliarSozinho(ctx, pool, c.prestador, c.inicio, time.Now()); err != nil || !executou {
		t.Fatalf("com a trava livre, não conciliou (executou=%v, err=%v)", executou, err)
	}
}

// O candidato paga o Pix e também o boleto: o segundo dinheiro é marcado para reembolso e a inscrição
// é confirmada uma vez só.
func TestPixEBoletoPagosViramUmaConfirmacaoEUmReembolso(t *testing.T) {
	c := novoCenario(t)
	pix := c.cobrar(t, "pix")
	boleto := c.cobrar(t, "boleto")
	c.pagar(t, pix)
	c.pagar(t, boleto)

	r := c.conciliar(t)
	if r.PorResultado[Confirmado] != 1 || r.PorResultado[EmDuplicidade] != 1 {
		t.Fatalf("conciliação: %+v", r)
	}
	if n := c.confirmacoes(t); n != 1 {
		t.Fatalf("%d confirmações na trilha", n)
	}
	if got := contar(t, `SELECT count(*) FROM cobranca WHERE inscricao_id = $1 AND estado = 'PAGA'`, c.inscricao); got != 1 {
		t.Fatalf("%d cobranças PAGA", got)
	}
	if got := contar(t, `SELECT count(*) FROM cobranca WHERE inscricao_id = $1 AND estado = 'PAGA_EM_DUPLICIDADE' AND reembolso_pendente`, c.inscricao); got != 1 {
		t.Fatalf("%d cobranças em duplicidade com reembolso", got)
	}
}

// Boleto impresso e pago duas vezes: a cobrança continua paga uma vez, com reembolso marcado.
func TestBoletoPagoDuasVezes(t *testing.T) {
	c := novoCenario(t)
	boleto := c.cobrar(t, "boleto")
	c.pagar(t, boleto)
	c.pagar(t, boleto)

	r := c.conciliar(t)
	if r.PorResultado[Confirmado] != 1 || r.PorResultado[EmDuplicidade] != 1 {
		t.Fatalf("conciliação: %+v", r)
	}
	if got := contar(t, `SELECT count(*) FROM cobranca WHERE id = $1 AND estado = 'PAGA' AND reembolso_pendente`, boleto.ID); got != 1 {
		t.Fatal("boleto pago duas vezes sem reembolso marcado")
	}
	if n := c.confirmacoes(t); n != 1 {
		t.Fatalf("%d confirmações na trilha", n)
	}
}

// Pagamento que chega depois de a inscrição expirar não a ressuscita: vira reembolso.
func TestPagamentoDepoisDaExpiracao(t *testing.T) {
	c := novoCenario(t)
	ctx := context.Background()
	boleto := c.cobrar(t, "boleto")

	// Prazo de pagamento e vencimento no passado, além da margem de compensação.
	_, err := pool.Exec(ctx, `UPDATE certame_fase SET abre_em = now() - interval '30 days', fecha_em = now() - interval '5 days'
		WHERE fase = 'pagamento' AND certame_id = (SELECT certame_id FROM inscricao WHERE id = $1)`, c.inscricao)
	must(t, err)
	_, err = pool.Exec(ctx, `UPDATE cobranca SET vence_em = now() - interval '5 days' WHERE id = $1`, boleto.ID)
	must(t, err)
	n, err := ExpirarVencidas(ctx, pool, MargemCompensacao)
	must(t, err)
	if n != 1 || c.estado(t) != inscricao.Expirada {
		t.Fatalf("expiradas=%d estado=%s", n, c.estado(t))
	}

	c.pagar(t, boleto)
	r := c.conciliar(t)
	if r.PorResultado[AposExpiracao] != 1 {
		t.Fatalf("conciliação: %+v", r)
	}
	if c.estado(t) != inscricao.Expirada {
		t.Fatalf("pagamento tardio mudou a inscrição para %s", c.estado(t))
	}
	if got := contar(t, `SELECT count(*) FROM cobranca WHERE id = $1 AND estado = 'PAGA_APOS_EXPIRACAO' AND reembolso_pendente`, boleto.ID); got != 1 {
		t.Fatal("pagamento tardio sem reembolso marcado")
	}
}

// Boleto ainda dentro da margem de compensação: a inscrição não expira, mesmo com o prazo encerrado.
func TestBoletoNaMargemDeCompensacaoNaoExpira(t *testing.T) {
	c := novoCenario(t)
	ctx := context.Background()
	boleto := c.cobrar(t, "boleto")
	_, err := pool.Exec(ctx, `UPDATE certame_fase SET abre_em = now() - interval '30 days', fecha_em = now() - interval '1 day'
		WHERE fase = 'pagamento' AND certame_id = (SELECT certame_id FROM inscricao WHERE id = $1)`, c.inscricao)
	must(t, err)
	_, err = pool.Exec(ctx, `UPDATE cobranca SET vence_em = now() - interval '1 day' WHERE id = $1`, boleto.ID)
	must(t, err)

	n, err := ExpirarVencidas(ctx, pool, MargemCompensacao)
	must(t, err)
	if n != 0 || c.estado(t) != inscricao.AguardandoPagamento {
		t.Fatalf("expirou dentro da margem: n=%d estado=%s", n, c.estado(t))
	}
	// O boleto pago no último dia é informado depois: confirma normalmente.
	c.pagar(t, boleto)
	if r := c.conciliar(t); r.PorResultado[Confirmado] != 1 || c.estado(t) != inscricao.Paga {
		t.Fatalf("conciliação: %+v, estado %s", r, c.estado(t))
	}
}

// Valor pago diferente do cobrado: não confirma, vira divergência.
func TestValorDivergenteNaoConfirma(t *testing.T) {
	c := novoCenario(t)
	p := c.pagar(t, c.cobrar(t, "pix"))
	c.sim.Adulterar(p.ID, p.ValorCentavos-1)

	r := c.conciliar(t)
	if r.PorResultado[ValorDivergente] != 1 || c.estado(t) != inscricao.AguardandoPagamento {
		t.Fatalf("conciliação: %+v, estado %s", r, c.estado(t))
	}
	if got := contar(t, `SELECT count(*) FROM divergencia WHERE psp_pagamento_id = $1 AND tipo = 'VALOR_DIVERGENTE'`, p.ID); got != 1 {
		t.Fatal("divergência de valor não registrada")
	}
}

// Confirmado aqui e ausente do extrato do prestador: divergência para alguém olhar.
func TestConfirmadoSemRegistroNoPrestador(t *testing.T) {
	c := novoCenario(t)
	cob := c.cobrar(t, "pix")
	falso := psp.Pagamento{ID: fmt.Sprintf("pag_fantasma_%d", c.inscricao), CobrancaID: *cob.PSPID,
		Referencia: cob.ID, ValorCentavos: cob.ValorCentavos, PagoEm: time.Now()}
	r, err := RegistrarPagamento(context.Background(), pool, falso, "webhook")
	must(t, err)
	if r != Confirmado {
		t.Fatalf("resultado %s", r)
	}
	rel := c.conciliar(t)
	if rel.SemRegistroNoPSP != 1 {
		t.Fatalf("conciliação não apontou o pagamento ausente do extrato: %+v", rel)
	}
	if got := contar(t, `SELECT count(*) FROM divergencia WHERE psp_pagamento_id = $1 AND tipo = 'PAGA_SEM_REGISTRO_NO_PSP'`, falso.ID); got != 1 {
		t.Fatal("divergência não registrada")
	}
}

// Pagamento para uma referência que não existe aqui: registrado como divergência, nada muda.
func TestPagamentoDeCobrancaDesconhecida(t *testing.T) {
	novoCenario(t) // só para pular sem banco
	p := psp.Pagamento{ID: fmt.Sprintf("pag_orfao_%d", time.Now().UnixNano()), Referencia: 999_999_999, ValorCentavos: 100, PagoEm: time.Now()}
	r, err := RegistrarPagamento(context.Background(), pool, p, "webhook")
	must(t, err)
	if r != CobrancaDesconhe {
		t.Fatalf("resultado %s", r)
	}
	if got := contar(t, `SELECT count(*) FROM divergencia WHERE psp_pagamento_id = $1`, p.ID); got != 1 {
		t.Fatal("divergência não registrada")
	}
}
