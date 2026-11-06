// Pacote externo: o teste usa a inscrição, e a inscrição usa a notificação.
package notificacao_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
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
	"lisura/internal/publicacao"
	"lisura/internal/seed"
)

type (
	Mensagem = notificacao.Mensagem
	Aviso    = notificacao.Aviso
)

var (
	Tarefa      = notificacao.Tarefa
	Resumo      = notificacao.Resumo
	AbrirCaixa  = notificacao.AbrirCaixa
	DaInscricao = notificacao.DaInscricao
)

var pool *pgxpool.Pool

func TestMain(m *testing.M) {
	p, err := dbtest.Pool(context.Background(), "t_notificacao")
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

type cenario struct {
	inscricao, candidato int64
	cpf, email           string
	correio              *notificacao.Memoria
	trabalhador          *fila.Trabalhador
}

// novo abre e finaliza uma inscrição (o evento FINALIZAR_COM_PAGAMENTO avisa) com um correio só dela.
func novo(t *testing.T) *cenario {
	t.Helper()
	if pool == nil {
		t.Skip("sem banco de teste")
	}
	ctx := context.Background()
	cert, err := seed.CriarCertame(ctx, pool, seed.Padrao())
	must(t, err)
	n := seq.Add(1)
	c := &cenario{cpf: conta.GerarCPF(n + time.Now().UnixNano()%1_000_000*100), email: "candidata" + time.Now().Format(".150405.000000000") + "@exemplo.test",
		correio: &notificacao.Memoria{}}
	c.candidato, err = conta.Criar(ctx, pool, c.cpf, "Ana Aviso", c.email)
	must(t, err)
	must(t, conta.AtualizarDados(ctx, pool, c.candidato, conta.Dados{Nome: "Ana Aviso", Email: c.email,
		DataNascimento: "1990-01-01", Municipio: "Campinas", UF: "SP"}))
	c.inscricao, _, err = inscricao.Abrir(ctx, pool, cert.ID, c.candidato, cert.Cargos[0], inscricao.Ator{Tipo: inscricao.Candidato, ID: "t"})
	must(t, err)
	must(t, inscricao.DefinirCidade(ctx, pool, c.inscricao, c.candidato, cert.Cidades[0]))
	must(t, inscricao.AceitarDeclaracoes(ctx, pool, c.inscricao, c.candidato, true))
	nada := func(context.Context, json.RawMessage) error { return nil }
	c.trabalhador = &fila.Trabalhador{Pool: pool, Manipuladores: map[string]fila.Manipulador{
		Tarefa: notificacao.Notificar(c.correio, pool), publicacao.TarefaCartao: nada, publicacao.TarefaCertame: nada,
		publicacao.TarefaRelacao: nada}}
	// Cada teste processa só a fila dele: o que outro deixou de propósito não roda com este correio.
	_, err = pool.Exec(ctx, `DELETE FROM tarefa`)
	must(t, err)
	_, err = inscricao.Finalizar(ctx, pool, c.inscricao, c.candidato)
	must(t, err)
	return c
}

func (c *cenario) evento(t *testing.T, ev inscricao.Evento, tipo inscricao.AtorTipo) {
	t.Helper()
	_, err := inscricao.Transicionar(context.Background(), pool, inscricao.Comando{InscricaoID: c.inscricao, Evento: ev,
		Ator: inscricao.Ator{Tipo: tipo, ID: "t"}})
	must(t, err)
}

func (c *cenario) esvaziar(t *testing.T) {
	t.Helper()
	must(t, c.trabalhador.Esvaziar(context.Background()))
}

func (c *cenario) avisos(t *testing.T) []Aviso {
	t.Helper()
	as, err := DaInscricao(context.Background(), pool, c.inscricao, c.candidato)
	must(t, err)
	return as
}

// Do envio à homologação: avisa o envio (com o prazo), o pagamento confirmado e a homologação; não
// avisa o começo da análise nem o deferimento intermediário. Cada aviso uma vez, mesmo pedido de novo;
// a prova guarda o resumo do texto que saiu; o texto não leva o CPF.
func TestCadaEventoQueImportaAvisaUmaVez(t *testing.T) {
	c := novo(t)
	c.evento(t, inscricao.ConfirmarPagamento, inscricao.Sistema)
	c.evento(t, inscricao.IniciarAnalise, inscricao.Sistema)
	c.evento(t, inscricao.Deferir, inscricao.Analista)
	c.evento(t, inscricao.Homologar, inscricao.Analista)
	c.esvaziar(t)

	as := c.avisos(t)
	var eventos []string
	for _, a := range as {
		eventos = append(eventos, a.Evento)
	}
	if !slices.Equal(eventos, []string{"FINALIZAR_COM_PAGAMENTO", "CONFIRMAR_PAGAMENTO", "HOMOLOGAR"}) {
		t.Fatalf("avisos: %v", eventos)
	}
	enviadas := c.correio.Enviadas()
	if len(enviadas) != 3 {
		t.Fatalf("%d mensagens", len(enviadas))
	}
	porAssunto := map[string]Mensagem{}
	for _, m := range enviadas {
		porAssunto[m.Assunto] = m
		if m.Para != c.email || strings.Contains(m.Texto, c.cpf) || strings.Contains(m.Texto, c.cpf[3:9]) {
			t.Errorf("mensagem para %s, com CPF? %q", m.Para, m.Texto)
		}
	}
	for _, a := range as {
		m, ok := porAssunto[a.Assunto]
		if !ok || a.Resumo != Resumo(m.Assunto, m.Texto) || !a.Entregue {
			t.Errorf("a prova do aviso %q não confere com o que saiu", a.Assunto)
		}
	}
	if !strings.Contains(as[0].Assunto, "falta pagar até") || !strings.Contains(as[0].Assunto, "/") {
		t.Errorf("o aviso de envio não diz o prazo: %q", as[0].Assunto)
	}

	// Pedir de novo os mesmos avisos (a tarefa reaberta) não manda nada de novo.
	rows, err := pool.Query(context.Background(), `SELECT evento_id FROM notificacao WHERE inscricao_id = $1`, c.inscricao)
	must(t, err)
	for rows.Next() {
		var id int64
		must(t, rows.Scan(&id))
		must(t, fila.Enfileirar(context.Background(), pool, Tarefa, strconv.FormatInt(id, 10), map[string]int64{"evento_id": id}))
	}
	must(t, rows.Err())
	c.esvaziar(t)
	if n := len(c.correio.Enviadas()); n != 3 || len(c.avisos(t)) != 3 {
		t.Fatalf("pedidos de novo, os avisos viraram %d mensagens", n)
	}
}

// A resposta do provedor se perde depois de a mensagem sair: a fila repete com a mesma chave, e a
// candidata recebe um e-mail, não dois.
func TestRespostaPerdidaNaoAvisaDuasVezes(t *testing.T) {
	c := novo(t)
	c.correio.Falhar.Store(1)
	c.esvaziar(t)
	if len(c.avisos(t)) != 0 {
		t.Fatal("o aviso foi dado por entregue com a resposta perdida")
	}
	_, err := pool.Exec(context.Background(), `UPDATE tarefa SET executar_em = now() WHERE tipo = $1 AND estado = 'PENDENTE'`, Tarefa)
	must(t, err)
	c.esvaziar(t)
	if n := len(c.correio.Enviadas()); n != 1 || len(c.avisos(t)) != 1 {
		t.Fatalf("%d mensagens, %d avisos; esperado 1 e 1", n, len(c.avisos(t)))
	}
}

// O provedor recusa o endereço: a tentativa fica registrada, com o motivo, e a fila não insiste.
func TestEnderecoRecusadoFicaRegistrado(t *testing.T) {
	c := novo(t)
	c.correio.Recusar(c.email)
	c.esvaziar(t)
	as := c.avisos(t)
	if len(as) != 1 || as[0].Entregue {
		t.Fatalf("avisos: %+v", as)
	}
	var falha string
	must(t, pool.QueryRow(context.Background(), `SELECT falha FROM notificacao WHERE inscricao_id = $1`, c.inscricao).Scan(&falha))
	if !strings.Contains(falha, c.email) {
		t.Fatalf("falha registrada: %q", falha)
	}
	var pendentes int
	must(t, pool.QueryRow(context.Background(), `SELECT count(*) FROM tarefa WHERE tipo = $1 AND estado <> 'CONCLUIDA'`, Tarefa).Scan(&pendentes))
	if pendentes != 0 {
		t.Fatalf("%d tarefas de aviso continuam tentando um endereço recusado", pendentes)
	}
}

// A prova do aviso não muda depois.
func TestAvisoNaoMuda(t *testing.T) {
	c := novo(t)
	c.esvaziar(t)
	for _, sql := range []string{`UPDATE notificacao SET assunto = 'outro' WHERE inscricao_id = $1`, `DELETE FROM notificacao WHERE inscricao_id = $1`} {
		if _, err := pool.Exec(context.Background(), sql, c.inscricao); err == nil {
			t.Fatalf("o banco aceitou: %s", sql)
		}
	}
}

// A caixa de saída da máquina local: a mesma chave é o mesmo envio, também depois de reabrir.
func TestCaixaSobreviveAReinicio(t *testing.T) {
	ctx := context.Background()
	caminho := filepath.Join(t.TempDir(), "caixa.jsonl")
	c, err := AbrirCaixa(caminho)
	must(t, err)
	a, err := c.Enviar(ctx, Mensagem{Chave: "evt-1", Para: "a@x.test", Assunto: "um", Texto: "t"})
	must(t, err)
	if b, _ := c.Enviar(ctx, Mensagem{Chave: "evt-1", Para: "a@x.test", Assunto: "um", Texto: "t"}); b != a {
		t.Fatalf("a mesma chave virou outro envio: %s, %s", a, b)
	}
	must(t, c.Fechar())
	c, err = AbrirCaixa(caminho)
	must(t, err)
	defer c.Fechar()
	if b, _ := c.Enviar(ctx, Mensagem{Chave: "evt-1"}); b != a {
		t.Fatalf("depois de reabrir, a mesma chave virou %s (era %s)", b, a)
	}
	if b, _ := c.Enviar(ctx, Mensagem{Chave: "evt-2", Para: "b@x.test"}); b == a {
		t.Fatal("chave nova com o id de outra")
	}
	conteudo, err := os.ReadFile(caminho)
	must(t, err)
	if n := strings.Count(string(conteudo), "\n"); n != 2 {
		t.Fatalf("%d linhas na caixa; esperado 2", n)
	}
}
