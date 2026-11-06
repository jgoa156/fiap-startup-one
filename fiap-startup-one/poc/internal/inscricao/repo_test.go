package inscricao

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"lisura/internal/dbtest"
)

// Testes de integração: rodam contra um PostgreSQL real, num schema recriado a cada execução.
// Sem banco disponível eles são pulados; os testes da máquina de estados rodam sempre.

var pool *pgxpool.Pool

func TestMain(m *testing.M) {
	p, err := dbtest.Pool(context.Background(), "t_inscricao")
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

type cenario struct {
	certame, cargo int64
}

func precisaBanco(t *testing.T) context.Context {
	t.Helper()
	if pool == nil {
		t.Skip("sem banco de teste")
	}
	return context.Background()
}

// novoCertame cria banca, certame e cargo; publica a versão 1 do edital se comEdital.
func novoCertame(t *testing.T, comEdital bool) cenario {
	t.Helper()
	ctx := context.Background()
	n := seq.Add(1)
	var c cenario
	var banca int64
	must(t, pool.QueryRow(ctx, `INSERT INTO banca (nome, cnpj) VALUES ($1, $2) RETURNING id`,
		fmt.Sprintf("Banca %d", n), fmt.Sprintf("%014d", n)).Scan(&banca))
	must(t, pool.QueryRow(ctx, `INSERT INTO certame (banca_id, nome) VALUES ($1, 'PSS') RETURNING id`,
		banca).Scan(&c.certame))
	must(t, pool.QueryRow(ctx,
		`INSERT INTO cargo (certame_id, nome, vagas, taxa_centavos) VALUES ($1, 'Agente', 10, 8000) RETURNING id`,
		c.certame).Scan(&c.cargo))
	_, err := pool.Exec(ctx, `INSERT INTO certame_fase (certame_id, fase, abre_em, fecha_em)
		VALUES ($1, 'inscricao', now() - interval '1 day', now() + interval '1 day')`, c.certame)
	must(t, err)
	if comEdital {
		publicarEdital(t, c.certame, 1, "-1 hour")
	}
	return c
}

func publicarEdital(t *testing.T, certame int64, versao int, quando string) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO edital_versao (certame_id, versao, publicado_em, resumo)
		 VALUES ($1, $2, now() + $3::interval, $4)`, certame, versao, quando, fmt.Sprintf("v%d", versao))
	must(t, err)
}

func novoCandidato(t *testing.T) int64 {
	t.Helper()
	var id int64
	must(t, pool.QueryRow(context.Background(),
		`INSERT INTO candidato (cpf, nome, email) VALUES ($1, 'Fulano', 'f@x.test') RETURNING id`,
		fmt.Sprintf("%011d", seq.Add(1))).Scan(&id))
	return id
}

func novaInscricao(t *testing.T, c cenario) int64 {
	t.Helper()
	id, _, err := Abrir(context.Background(), pool, c.certame, novoCandidato(t), c.cargo, cand)
	must(t, err)
	return id
}

var (
	cand     = Ator{Candidato, "cpf-teste"}
	sistema  = Ator{Sistema, "gateway-sandbox"}
	analista = Ator{Analista, "analista-1"}
)

func aplicar(t *testing.T, id int64, ev Evento, a Ator, motivo string) Resultado {
	t.Helper()
	r, err := Transicionar(context.Background(), pool, Comando{id, ev, a, motivo})
	must(t, err)
	return r
}

type linhaTrilha struct {
	evento, de, para string
	versao           int
}

func trilha(t *testing.T, id int64) []linhaTrilha {
	t.Helper()
	rows, err := pool.Query(context.Background(),
		`SELECT evento, de_estado, para_estado, edital_versao FROM inscricao_evento
		 WHERE inscricao_id = $1 ORDER BY id`, id)
	must(t, err)
	defer rows.Close()
	var out []linhaTrilha
	for rows.Next() {
		var l linhaTrilha
		must(t, rows.Scan(&l.evento, &l.de, &l.para, &l.versao))
		out = append(out, l)
	}
	must(t, rows.Err())
	return out
}

func estado(t *testing.T, id int64) Estado {
	t.Helper()
	var e Estado
	must(t, pool.QueryRow(context.Background(), `SELECT estado FROM inscricao WHERE id = $1`, id).Scan(&e))
	return e
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestFluxoCompletoGravaTrilha(t *testing.T) {
	precisaBanco(t)
	id := novaInscricao(t, novoCertame(t, true))

	aplicar(t, id, FinalizarComPagamento, cand, "")
	aplicar(t, id, ConfirmarPagamento, sistema, "")
	aplicar(t, id, IniciarAnalise, sistema, "")
	aplicar(t, id, Indeferir, analista, "LAUDO_ILEGIVEL")
	aplicar(t, id, ProverRecurso, analista, "LAUDO_REENVIADO")
	aplicar(t, id, Homologar, analista, "")

	esperado := []linhaTrilha{
		{"CRIAR", "", "RASCUNHO", 1},
		{"FINALIZAR_COM_PAGAMENTO", "RASCUNHO", "AGUARDANDO_PAGAMENTO", 1},
		{"CONFIRMAR_PAGAMENTO", "AGUARDANDO_PAGAMENTO", "PAGA", 1},
		{"INICIAR_ANALISE", "PAGA", "EM_ANALISE", 1},
		{"INDEFERIR", "EM_ANALISE", "INDEFERIDA", 1},
		{"PROVER_RECURSO", "INDEFERIDA", "DEFERIDA", 1},
		{"HOMOLOGAR", "DEFERIDA", "HOMOLOGADA", 1},
	}
	got := trilha(t, id)
	if fmt.Sprint(got) != fmt.Sprint(esperado) {
		t.Fatalf("trilha\n got: %v\nwant: %v", got, esperado)
	}
	if e := estado(t, id); e != Homologada {
		t.Fatalf("estado final %s", e)
	}
}

func TestTransicaoInvalidaNaoGravaNada(t *testing.T) {
	ctx := precisaBanco(t)
	id := novaInscricao(t, novoCertame(t, true))

	_, err := Transicionar(ctx, pool, Comando{id, Homologar, analista, ""})
	if !errors.Is(err, ErrTransicaoInvalida) {
		t.Fatalf("esperava ErrTransicaoInvalida, veio %v", err)
	}
	if n := len(trilha(t, id)); n != 1 {
		t.Fatalf("trilha com %d linhas, esperava só a criação", n)
	}
	if e := estado(t, id); e != Rascunho {
		t.Fatalf("estado mudou para %s", e)
	}
}

func TestAtorEMotivoSaoExigidos(t *testing.T) {
	ctx := precisaBanco(t)
	id := novaInscricao(t, novoCertame(t, true))
	aplicar(t, id, FinalizarIsento, cand, "")
	aplicar(t, id, IniciarAnalise, sistema, "")

	if _, err := Transicionar(ctx, pool, Comando{id, Deferir, cand, ""}); !errors.Is(err, ErrAtorNaoAutorizado) {
		t.Errorf("candidato deferindo a própria inscrição: %v", err)
	}
	if _, err := Transicionar(ctx, pool, Comando{id, Indeferir, analista, ""}); !errors.Is(err, ErrMotivoObrigatorio) {
		t.Errorf("indeferimento sem motivo: %v", err)
	}
	if _, err := Transicionar(ctx, pool, Comando{999999999, Deferir, analista, ""}); !errors.Is(err, ErrNaoEncontrada) {
		t.Errorf("inscrição inexistente: %v", err)
	}
	if e := estado(t, id); e != EmAnalise {
		t.Fatalf("estado mudou para %s", e)
	}
}

func TestReenvioEIdempotente(t *testing.T) {
	precisaBanco(t)
	id := novaInscricao(t, novoCertame(t, true))
	aplicar(t, id, FinalizarComPagamento, cand, "")

	if r := aplicar(t, id, ConfirmarPagamento, sistema, ""); r.JaAplicada {
		t.Fatal("primeira confirmação marcada como reenvio")
	}
	if r := aplicar(t, id, ConfirmarPagamento, sistema, ""); !r.JaAplicada {
		t.Fatal("segunda confirmação não foi tratada como reenvio")
	}
	if n := len(trilha(t, id)); n != 3 {
		t.Fatalf("trilha com %d linhas, esperava 3 (reenvio não grava)", n)
	}
}

// O webhook do meio de pagamento chega repetido sob carga: só uma confirmação pode valer.
func TestConfirmacoesConcorrentesSoUmaVale(t *testing.T) {
	ctx := precisaBanco(t)
	id := novaInscricao(t, novoCertame(t, true))
	aplicar(t, id, FinalizarComPagamento, cand, "")

	const n = 50
	var wg sync.WaitGroup
	var aplicadas, reenvios, falhas atomic.Int32
	for range n {
		wg.Go(func() {
			r, err := Transicionar(ctx, pool, Comando{id, ConfirmarPagamento, sistema, ""})
			switch {
			case err != nil:
				falhas.Add(1)
				t.Error(err)
			case r.JaAplicada:
				reenvios.Add(1)
			default:
				aplicadas.Add(1)
			}
		})
	}
	wg.Wait()
	if aplicadas.Load() != 1 || reenvios.Load() != n-1 || falhas.Load() != 0 {
		t.Fatalf("aplicadas=%d reenvios=%d falhas=%d", aplicadas.Load(), reenvios.Load(), falhas.Load())
	}
	if n := len(trilha(t, id)); n != 3 {
		t.Fatalf("trilha com %d linhas, esperava 3", n)
	}
}

// Pagamento confirmado no mesmo instante em que o prazo vence: exatamente um dos dois vence.
func TestConfirmacaoContraExpiracao(t *testing.T) {
	ctx := precisaBanco(t)
	id := novaInscricao(t, novoCertame(t, true))
	aplicar(t, id, FinalizarComPagamento, cand, "")

	var wg sync.WaitGroup
	var transicoes atomic.Int32
	for i := range 40 {
		ev := ConfirmarPagamento
		if i%2 == 1 {
			ev = Expirar
		}
		wg.Go(func() {
			r, err := Transicionar(ctx, pool, Comando{id, ev, sistema, ""})
			if err == nil && !r.JaAplicada {
				transicoes.Add(1)
			} else if err != nil && !errors.Is(err, ErrTransicaoInvalida) {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if transicoes.Load() != 1 {
		t.Fatalf("%d transições a partir de AGUARDANDO_PAGAMENTO, esperava 1", transicoes.Load())
	}
	tr := trilha(t, id)
	if len(tr) != 3 || tr[2].para != string(estado(t, id)) {
		t.Fatalf("trilha inconsistente com o estado: %v / %s", tr, estado(t, id))
	}
}

// Duplo clique e recarregamento no último dia: a inscrição não duplica.
func TestAbrirConcorrenteNaoDuplica(t *testing.T) {
	ctx := precisaBanco(t)
	c := novoCertame(t, true)
	candidato := novoCandidato(t)

	const n = 50
	ids := make([]int64, n)
	var criadas atomic.Int32
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			id, criada, err := Abrir(ctx, pool, c.certame, candidato, c.cargo, cand)
			if err != nil {
				t.Error(err)
			}
			ids[i] = id
			if criada {
				criadas.Add(1)
			}
		})
	}
	wg.Wait()
	for _, id := range ids {
		if id != ids[0] {
			t.Fatalf("ids diferentes para o mesmo candidato: %v", ids)
		}
	}
	if criadas.Load() != 1 {
		t.Fatalf("%d criações, esperava 1", criadas.Load())
	}
	if n := len(trilha(t, ids[0])); n != 1 {
		t.Fatalf("trilha com %d linhas de criação", n)
	}
}

func TestTrilhaEImutavel(t *testing.T) {
	ctx := precisaBanco(t)
	id := novaInscricao(t, novoCertame(t, true))

	for _, sql := range []string{
		`UPDATE inscricao_evento SET motivo_codigo = 'ADULTERADO' WHERE inscricao_id = $1`,
		`DELETE FROM inscricao_evento WHERE inscricao_id = $1`,
	} {
		if _, err := pool.Exec(ctx, sql, id); err == nil {
			t.Errorf("trilha aceitou: %s", sql)
		}
	}
	if _, err := pool.Exec(ctx, `TRUNCATE inscricao_evento CASCADE`); err == nil {
		t.Error("trilha aceitou TRUNCATE")
	}
	if n := len(trilha(t, id)); n != 1 {
		t.Fatalf("trilha alterada: %d linhas", n)
	}
}

// Decisão tomada sob a versão 2 não é reavaliada com a régua da versão 3.
func TestDecisaoRegistraVersaoDoEditalEmVigor(t *testing.T) {
	precisaBanco(t)
	c := novoCertame(t, true)
	id := novaInscricao(t, c)

	aplicar(t, id, FinalizarIsento, cand, "")
	publicarEdital(t, c.certame, 2, "-1 minute") // retificação
	publicarEdital(t, c.certame, 3, "+1 day")    // ainda não publicada
	aplicar(t, id, IniciarAnalise, sistema, "")

	tr := trilha(t, id)
	if got := []int{tr[0].versao, tr[1].versao, tr[2].versao}; fmt.Sprint(got) != "[1 1 2]" {
		t.Fatalf("versões na trilha %v, esperava [1 1 2]", got)
	}
}

func TestSemEditalPublicadoNaoAbreInscricao(t *testing.T) {
	ctx := precisaBanco(t)
	c := novoCertame(t, false)
	candidato := novoCandidato(t)

	_, _, err := Abrir(ctx, pool, c.certame, candidato, c.cargo, cand)
	if !errors.Is(err, ErrEditalNaoPublicado) {
		t.Fatalf("esperava ErrEditalNaoPublicado, veio %v", err)
	}
	var n int
	must(t, pool.QueryRow(ctx, `SELECT count(*) FROM inscricao WHERE certame_id = $1`, c.certame).Scan(&n))
	if n != 0 {
		t.Fatalf("inscrição gravada sem edital: %d", n)
	}
}
