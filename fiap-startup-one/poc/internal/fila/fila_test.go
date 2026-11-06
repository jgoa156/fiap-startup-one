package fila

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"lisura/internal/dbtest"
)

var pool *pgxpool.Pool

func TestMain(m *testing.M) {
	p, err := dbtest.Pool(context.Background(), "t_fila")
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

func precisa(t *testing.T) context.Context {
	t.Helper()
	if pool == nil {
		t.Skip("sem banco de teste")
	}
	// Cada teste começa com a fila vazia: o trabalhador pega qualquer tarefa vencida, inclusive a que
	// outro teste deixou pendente de propósito.
	_, err := pool.Exec(context.Background(), `DELETE FROM tarefa`)
	must(t, err)
	return context.Background()
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func estado(t *testing.T, tipo, chave string) (e string, tentativas, versao int) {
	t.Helper()
	must(t, pool.QueryRow(context.Background(),
		`SELECT estado, tentativas, versao FROM tarefa WHERE tipo = $1 AND chave = $2`, tipo, chave).Scan(&e, &tentativas, &versao))
	return
}

// Oito trabalhadores disputando a mesma tarefa: ela roda uma vez.
func TestUmaTarefaUmTrabalhador(t *testing.T) {
	ctx := precisa(t)
	must(t, Enfileirar(ctx, pool, "unica", "a", map[string]int{"x": 1}))
	var execucoes atomic.Int32
	tr := &Trabalhador{Pool: pool, Manipuladores: map[string]Manipulador{
		"unica": func(context.Context, json.RawMessage) error {
			execucoes.Add(1)
			time.Sleep(200 * time.Millisecond) // segura a tarefa enquanto os outros tentam
			return nil
		},
	}}
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			if _, err := tr.ProcessarUma(ctx); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if n := execucoes.Load(); n != 1 {
		t.Fatalf("%d execuções, esperava 1", n)
	}
	if e, _, _ := estado(t, "unica", "a"); e != "CONCLUIDA" {
		t.Fatalf("estado %s", e)
	}
}

// A mesma tarefa enfileirada várias vezes enquanto espera vira uma só (no pico, mil mudanças no mesmo
// certame viram uma republicação).
func TestEnfileirarDeduplica(t *testing.T) {
	ctx := precisa(t)
	for range 10 {
		must(t, Enfileirar(ctx, pool, "dedup", "b", nil))
	}
	var n int
	must(t, pool.QueryRow(ctx, `SELECT count(*) FROM tarefa WHERE tipo = 'dedup'`).Scan(&n))
	if n != 1 {
		t.Fatalf("%d linhas", n)
	}
	if e, _, v := estado(t, "dedup", "b"); e != "PENDENTE" || v != 10 {
		t.Fatalf("estado %s versão %d", e, v)
	}
}

// 500 cartões da homologação em lote, vencidos há mais tempo, e depois um Pix e um cartão normal: o
// trabalhador pega o Pix primeiro, depois o normal, e só então o lote — em ordem de chegada.
func TestUrgenteENormalPassamNaFrenteDoLote(t *testing.T) {
	ctx := precisa(t)
	for i := range 500 {
		must(t, EnfileirarCom(ctx, pool, Lote, "lote", string(rune('a'+i%26))+time.Now().Format(".000000000"), nil))
	}
	must(t, Enfileirar(ctx, pool, "normal", "n", nil))
	must(t, EnfileirarCom(ctx, pool, Urgente, "pix", "p", nil))
	var ordem []string
	var mu sync.Mutex
	anotar := func(tipo string) Manipulador {
		return func(context.Context, json.RawMessage) error {
			mu.Lock()
			ordem = append(ordem, tipo)
			mu.Unlock()
			return nil
		}
	}
	tr := &Trabalhador{Pool: pool, Manipuladores: map[string]Manipulador{
		"lote": anotar("lote"), "normal": anotar("normal"), "pix": anotar("pix")}}
	for range 3 {
		_, err := tr.ProcessarUma(ctx)
		must(t, err)
	}
	if len(ordem) != 3 || ordem[0] != "pix" || ordem[1] != "normal" || ordem[2] != "lote" {
		t.Fatalf("ordem %v, esperado [pix normal lote]", ordem)
	}
}

// O mesmo cartão enfileirado em lote e, antes de rodar, de novo porque o candidato mexeu nele: sobe para
// normal. O contrário (normal pendente reenfileirado em lote) não rebaixa.
func TestReenfileirarPromoveNuncaRebaixa(t *testing.T) {
	ctx := precisa(t)
	prioridade := func(chave string) (p int) {
		must(t, pool.QueryRow(ctx, `SELECT prioridade FROM tarefa WHERE tipo = 'cartao' AND chave = $1`, chave).Scan(&p))
		return
	}
	must(t, EnfileirarCom(ctx, pool, Lote, "cartao", "1", nil))
	must(t, Enfileirar(ctx, pool, "cartao", "1", nil))
	if p := prioridade("1"); p != int(Normal) {
		t.Fatalf("lote reenfileirado como normal ficou com prioridade %d", p)
	}
	must(t, Enfileirar(ctx, pool, "cartao", "2", nil))
	must(t, EnfileirarCom(ctx, pool, Lote, "cartao", "2", nil))
	if p := prioridade("2"); p != int(Normal) {
		t.Fatalf("normal pendente reenfileirado em lote foi rebaixado para %d", p)
	}
	// Depois de concluída, a próxima vez vale a prioridade nova.
	_, err := pool.Exec(ctx, `UPDATE tarefa SET estado = 'CONCLUIDA' WHERE chave = '2'`)
	must(t, err)
	must(t, EnfileirarCom(ctx, pool, Lote, "cartao", "2", nil))
	if p := prioridade("2"); p != int(Lote) {
		t.Fatalf("tarefa concluída reaberta em lote ficou com prioridade %d", p)
	}
}

// Enfileirar muitas numa instrução só segue as regras de enfileirar uma por uma: a pendente em lote é
// promovida e sobe de versão, a concluída reabre do zero, a nova nasce, e a chave repetida na mesma
// chamada vira uma tarefa só, com a última carga.
func TestEnfileirarMuitosComoUmPorUm(t *testing.T) {
	ctx := precisa(t)
	must(t, EnfileirarCom(ctx, pool, Lote, "cartao", "pendente", nil))
	must(t, Enfileirar(ctx, pool, "cartao", "concluida", nil))
	_, err := pool.Exec(ctx, `UPDATE tarefa SET estado = 'CONCLUIDA', tentativas = 3 WHERE chave = 'concluida'`)
	must(t, err)

	chaves := []string{"pendente", "concluida", "nova", "nova"}
	cargas := []any{1, 2, 3, 4}
	for i := range 1000 {
		chaves = append(chaves, fmt.Sprintf("massa-%04d", i))
		cargas = append(cargas, i)
	}
	must(t, EnfileirarMuitos(ctx, pool, Normal, "cartao", chaves, cargas))

	ler := func(chave string) (estado string, tentativas, versao, prioridade int, carga string) {
		must(t, pool.QueryRow(ctx, `SELECT estado, tentativas, versao, prioridade, carga::text FROM tarefa
			WHERE tipo = 'cartao' AND chave = $1`, chave).Scan(&estado, &tentativas, &versao, &prioridade, &carga))
		return
	}
	if e, _, v, p, c := ler("pendente"); e != "PENDENTE" || v != 2 || p != int(Normal) || c != "1" {
		t.Errorf("pendente: %s v%d p%d carga %s", e, v, p, c)
	}
	if e, tent, _, _, c := ler("concluida"); e != "PENDENTE" || tent != 0 || c != "2" {
		t.Errorf("concluída: %s, %d tentativas, carga %s", e, tent, c)
	}
	if e, _, v, _, c := ler("nova"); e != "PENDENTE" || v != 1 || c != "4" {
		t.Errorf("nova: %s v%d carga %s (esperado a última carga, 4)", e, v, c)
	}
	var n int
	must(t, pool.QueryRow(ctx, `SELECT count(*) FROM tarefa WHERE tipo = 'cartao'`).Scan(&n))
	if n != 3+1000 {
		t.Fatalf("%d tarefas; esperado 1.003", n)
	}
	if err := EnfileirarMuitos(ctx, pool, Normal, "cartao", []string{"x"}, nil); err == nil {
		t.Fatal("aceitou chaves e cargas de tamanhos diferentes")
	}
}

// Falha repetida: espera crescente entre tentativas e, depois do limite, FALHOU com o erro anotado.
func TestFalhaRepeteEDesiste(t *testing.T) {
	ctx := precisa(t)
	must(t, Enfileirar(ctx, pool, "quebrada", "c", nil))
	tr := &Trabalhador{Pool: pool, MaxTentativas: 3, Manipuladores: map[string]Manipulador{
		"quebrada": func(context.Context, json.RawMessage) error { return errors.New("prestador fora do ar") },
	}}
	for tentativa := 1; tentativa <= 3; tentativa++ {
		fez, err := tr.ProcessarUma(ctx)
		must(t, err)
		if !fez {
			t.Fatalf("tentativa %d: a tarefa não estava disponível", tentativa)
		}
		var espera time.Duration
		var segundos float64
		must(t, pool.QueryRow(ctx, `SELECT extract(epoch FROM executar_em - now()) FROM tarefa WHERE tipo = 'quebrada'`).Scan(&segundos))
		espera = time.Duration(segundos * float64(time.Second))
		if tentativa < 3 && espera < time.Duration(tentativa)*time.Second/2 {
			t.Fatalf("tentativa %d reagendada para daqui a só %s", tentativa, espera)
		}
		_, err = pool.Exec(ctx, `UPDATE tarefa SET executar_em = now() WHERE tipo = 'quebrada'`)
		must(t, err)
	}
	var erro string
	e, n, _ := estado(t, "quebrada", "c")
	must(t, pool.QueryRow(ctx, `SELECT ultimo_erro FROM tarefa WHERE tipo = 'quebrada'`).Scan(&erro))
	if e != "FALHOU" || n != 3 || erro != "prestador fora do ar" {
		t.Fatalf("estado %s, %d tentativas, erro %q", e, n, erro)
	}
	if fez, _ := tr.ProcessarUma(ctx); fez {
		t.Fatal("tarefa abandonada foi pega de novo")
	}
}

// O cartão muda enquanto está sendo publicado: a tarefa não é dada como concluída com a versão velha,
// roda de novo.
func TestReenfileiradaDuranteExecucaoRodaDeNovo(t *testing.T) {
	ctx := precisa(t)
	must(t, Enfileirar(ctx, pool, "cartao", "d", nil))
	var execucoes atomic.Int32
	tr := &Trabalhador{Pool: pool, Manipuladores: map[string]Manipulador{
		"cartao": func(ctx context.Context, _ json.RawMessage) error {
			if execucoes.Add(1) == 1 {
				return Enfileirar(ctx, pool, "cartao", "d", nil) // a mudança chega no meio
			}
			return nil
		},
	}}
	must(t, tr.Esvaziar(ctx))
	if n := execucoes.Load(); n != 2 {
		t.Fatalf("%d execuções, esperava 2", n)
	}
	if e, _, _ := estado(t, "cartao", "d"); e != "CONCLUIDA" {
		t.Fatalf("estado %s", e)
	}
}

// Mudanças em rajada viram UMA execução no fim da janela — e mudança nova não adia a janela (senão,
// com mudanças contínuas, a relação nunca seria republicada).
func TestEnfileirarDepoisAgrupaSemAdiar(t *testing.T) {
	ctx := precisa(t)
	var execucoes atomic.Int32
	tr := &Trabalhador{Pool: pool, Manipuladores: map[string]Manipulador{
		"relacao": func(context.Context, json.RawMessage) error { execucoes.Add(1); return nil },
	}}
	must(t, EnfileirarDepois(ctx, pool, "relacao", "f", nil, time.Hour))
	var marcada time.Time
	must(t, pool.QueryRow(ctx, `SELECT executar_em FROM tarefa WHERE tipo = 'relacao'`).Scan(&marcada))

	for range 50 { // a rajada de inscrições
		must(t, EnfileirarDepois(ctx, pool, "relacao", "f", nil, time.Hour))
	}
	must(t, tr.Esvaziar(ctx))
	if execucoes.Load() != 0 {
		t.Fatal("rodou antes da janela")
	}
	var agora time.Time
	must(t, pool.QueryRow(ctx, `SELECT executar_em FROM tarefa WHERE tipo = 'relacao'`).Scan(&agora))
	if !agora.Equal(marcada) {
		t.Fatalf("mudança nova adiou a janela: de %s para %s", marcada, agora)
	}
	if n := contar(t, `SELECT count(*) FROM tarefa WHERE tipo = 'relacao'`); n != 1 {
		t.Fatalf("%d linhas: a rajada não foi agrupada", n)
	}

	// A janela vence: uma execução para as 51 mudanças.
	_, err := pool.Exec(ctx, `UPDATE tarefa SET executar_em = now() WHERE tipo = 'relacao'`)
	must(t, err)
	must(t, tr.Esvaziar(ctx))
	if execucoes.Load() != 1 {
		t.Fatalf("%d execuções para uma rajada", execucoes.Load())
	}
	// Terminada, a próxima mudança abre uma janela nova (não roda já).
	must(t, EnfileirarDepois(ctx, pool, "relacao", "f", nil, time.Hour))
	must(t, tr.Esvaziar(ctx))
	if execucoes.Load() != 1 {
		t.Fatal("mudança depois de concluída rodou sem esperar a janela")
	}
}

// O caso que o pico produz: chega mudança ENQUANTO a relação está sendo republicada. A nova
// execução tem de esperar a janela, não rodar já — senão a relação roda em laço sob carga (medido
// em 05/11: ~100 republicações por minuto com a janela de 30 s "ligada").
func TestMudancaDuranteExecucaoRespeitaAJanela(t *testing.T) {
	ctx := precisa(t)
	var execucoes atomic.Int32
	tr := &Trabalhador{Pool: pool, Manipuladores: map[string]Manipulador{
		"relacao2": func(ctx context.Context, _ json.RawMessage) error {
			execucoes.Add(1)
			return EnfileirarDepois(ctx, pool, "relacao2", "h", nil, time.Hour) // uma inscrição mudou no meio
		},
	}}
	must(t, EnfileirarDepois(ctx, pool, "relacao2", "h", nil, time.Hour))
	_, err := pool.Exec(ctx, `UPDATE tarefa SET executar_em = now() WHERE tipo = 'relacao2'`) // a janela venceu
	must(t, err)
	must(t, tr.Esvaziar(ctx))
	if execucoes.Load() != 1 {
		t.Fatalf("%d execuções: a mudança do meio furou a janela", execucoes.Load())
	}
	var e string
	var falta time.Duration
	must(t, pool.QueryRow(ctx, `SELECT estado, extract(epoch FROM executar_em - now())::float8 * interval '1 second'
		FROM tarefa WHERE tipo = 'relacao2'`).Scan(&e, &falta))
	if e != "PENDENTE" || falta < 50*time.Minute {
		t.Fatalf("a mudança do meio tinha de esperar a janela de 1 h: estado %s, faltam %s", e, falta)
	}
}

func contar(t *testing.T, sql string) int {
	t.Helper()
	var n int
	must(t, pool.QueryRow(context.Background(), sql).Scan(&n))
	return n
}

// A medição da fila separa espera (falta de trabalhador) de execução (tarefa lenta).
func TestMedidasSeparamEsperaDeExecucao(t *testing.T) {
	ctx := precisa(t)
	must(t, Enfileirar(ctx, pool, "medida", "g", nil))
	_, err := pool.Exec(ctx, `UPDATE tarefa SET executar_em = now() - interval '2 seconds' WHERE tipo = 'medida'`)
	must(t, err)
	tr := &Trabalhador{Pool: pool, Manipuladores: map[string]Manipulador{
		"medida": func(context.Context, json.RawMessage) error { time.Sleep(50 * time.Millisecond); return nil },
	}}
	must(t, tr.Esvaziar(ctx))
	m := tr.Medidas()["medida"]
	if m.Tarefas != 1 || m.EsperaMedia < 2*time.Second || m.EsperaMedia > 3*time.Second {
		t.Fatalf("espera medida errada: %+v", m)
	}
	if m.ExecucaoMedia < 50*time.Millisecond || m.ExecucaoMedia > time.Second {
		t.Fatalf("execução medida errada: %+v", m)
	}
	if len(tr.Medidas()) != 0 {
		t.Fatal("Medidas não zerou depois da leitura")
	}
}

// Tarefa sem manipulador conhecido não some: falha com o motivo anotado.
func TestTipoDesconhecido(t *testing.T) {
	ctx := precisa(t)
	must(t, Enfileirar(ctx, pool, "inexistente", "e", nil))
	tr := &Trabalhador{Pool: pool, Manipuladores: map[string]Manipulador{}}
	_, err := tr.ProcessarUma(ctx)
	must(t, err)
	var erro string
	must(t, pool.QueryRow(ctx, `SELECT ultimo_erro FROM tarefa WHERE tipo = 'inexistente'`).Scan(&erro))
	if erro == "" {
		t.Fatal("falha sem motivo anotado")
	}
}
