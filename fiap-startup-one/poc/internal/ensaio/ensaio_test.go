package ensaio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	"lisura/internal/auditoria"
)

// O k6 falso é este mesmo binário de teste, chamado de novo com ENSAIO_K6_FALSO=1: imprime o
// PARA_AUDITAR da rodada e sai com 99 (limite estourado) a partir do multiplicador QUEBRA_EM.
func TestMain(m *testing.M) {
	if os.Getenv("ENSAIO_K6_FALSO") == "1" {
		mult, _ := strconv.ParseFloat(os.Getenv("MULT"), 64)
		fmt.Println("k6 falso no patamar", os.Getenv("MULT"))
		if os.Getenv("DEMORAR") == "1" {
			time.Sleep(30 * time.Second)
		}
		fmt.Printf("PARA_AUDITAR lisura-auditoria -rodada %s -enviadas 100 -pagas 90\n", os.Getenv("RODADA"))
		if q, _ := strconv.ParseFloat(os.Getenv("QUEBRA_EM"), 64); q > 0 && mult >= q {
			os.Exit(99)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

type auditado struct {
	mu      sync.Mutex
	rodadas []string
	esp     []auditoria.Esperado
	falhar  string // rodada que a auditoria acusa
}

func (a *auditado) auditar(_ context.Context, rodada string, e auditoria.Esperado) (auditoria.Relatorio, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.rodadas, a.esp = append(a.rodadas, rodada), append(a.esp, e)
	r := auditoria.Relatorio{Rodada: rodada}
	if rodada == a.falhar {
		r.Problemas = []string{"COBRADA DUAS VEZES: 1 inscrição"}
	}
	return r, nil
}

func executor(t *testing.T, a *auditado) *Executor {
	t.Helper()
	t.Setenv("ENSAIO_K6_FALSO", "1")
	return &Executor{K6: os.Args[0], Roteiros: t.TempDir(), Base: "http://api", Front: "http://front",
		Auditar: a.auditar, Faixas: func(context.Context) ([]int, error) { return []int{5, 7, 9, 11}, nil }}
}

func esperarFim(t *testing.T, x *Executor) Ensaio {
	t.Helper()
	for limite := time.Now().Add(40 * time.Second); time.Now().Before(limite); time.Sleep(50 * time.Millisecond) {
		if atual, hist := x.Estado(); atual == nil && len(hist) > 0 {
			return hist[0]
		}
	}
	t.Fatal("o ensaio não terminou")
	return Ensaio{}
}

// A escada dobra o multiplicador a cada patamar e para no primeiro que quebra, cada um numa faixa de
// CPF própria e auditado com o que o k6 contou.
func TestEscadaParaNoPrimeiroPatamarQueQuebra(t *testing.T) {
	t.Setenv("QUEBRA_EM", "40")
	a := &auditado{}
	x := executor(t, a)
	if _, err := x.Iniciar(Pedido{Cenario: "escada", Mult: 10, Duracao: "30s", Certame: 5, Teto: 1280}); err != nil {
		t.Fatal(err)
	}
	e := esperarFim(t, x)
	var mults []float64
	for _, p := range e.Patamares {
		mults = append(mults, p.Mult)
	}
	if e.Estado != "quebrou" || !slices.Equal(mults, []float64{10, 20, 40}) {
		t.Fatalf("estado %s, patamares %v; esperado quebrou em 10, 20, 40", e.Estado, mults)
	}
	ultimo := e.Patamares[2]
	if ultimo.Passou == nil || *ultimo.Passou || *ultimo.SaidaK6 != 99 || len(ultimo.Problemas) == 0 {
		t.Fatalf("o patamar que quebrou: %+v", ultimo)
	}
	if !slices.Equal(a.rodadas, []string{"1005", "2007", "4009"}) || a.esp[0] != (auditoria.Esperado{Enviadas: 100, Pagas: 90}) {
		t.Fatalf("auditadas %v com %v", a.rodadas, a.esp)
	}
}

// A auditoria acusar algo quebra o patamar, mesmo com o k6 dentro dos limites.
func TestAuditoriaComProblemaQuebraOPatamar(t *testing.T) {
	a := &auditado{falhar: "2007"}
	x := executor(t, a)
	if _, err := x.Iniciar(Pedido{Cenario: "escada", Mult: 10, Duracao: "30s", Certame: 5, Teto: 1280}); err != nil {
		t.Fatal(err)
	}
	e := esperarFim(t, x)
	if e.Estado != "quebrou" || len(e.Patamares) != 2 || e.Patamares[1].Problemas[0] != "COBRADA DUAS VEZES: 1 inscrição" {
		t.Fatalf("estado %s, patamares %+v", e.Estado, e.Patamares)
	}
}

// Um ensaio de cada vez; parar encerra o k6 e o ensaio fica como parado, sem auditar o meio-patamar.
func TestUmEnsaioDeCadaVezEPararEncerra(t *testing.T) {
	t.Setenv("DEMORAR", "1")
	a := &auditado{}
	x := executor(t, a)
	if _, err := x.Iniciar(Pedido{Cenario: "ultimo-dia", Mult: 40, Duracao: "1m", Certame: 5}); err != nil {
		t.Fatal(err)
	}
	if _, err := x.Iniciar(Pedido{Cenario: "ultimo-dia", Mult: 40, Duracao: "1m", Certame: 5}); !errors.Is(err, ErrOcupado) {
		t.Fatalf("dois ensaios ao mesmo tempo: %v", err)
	}
	time.Sleep(500 * time.Millisecond)
	inicio := time.Now()
	if !x.Parar() {
		t.Fatal("parar não achou o ensaio")
	}
	e := esperarFim(t, x)
	if e.Estado != "parado" || time.Since(inicio) > 15*time.Second || len(a.rodadas) != 0 {
		t.Fatalf("estado %s em %v, auditadas %v", e.Estado, time.Since(inicio), a.rodadas)
	}
}

func TestPedidoInvalidoNemComeca(t *testing.T) {
	x := executor(t, &auditado{})
	for _, p := range []Pedido{
		{Cenario: "outro", Mult: 1, Certame: 5},
		{Cenario: "ultimo-dia", Mult: 0, Certame: 5},
		{Cenario: "ultimo-dia", Mult: 40},
		{Cenario: "ultimo-dia", Mult: 40, Certame: 5, Duracao: "5s"},
		{Cenario: "escada", Mult: 100, Certame: 5, Teto: 50},
		{Cenario: "pico-resultado", Certame: 6},
	} {
		if _, err := x.Iniciar(p); !errors.Is(err, ErrPedido) {
			t.Errorf("%+v: %v", p, err)
		}
	}
	a, h := x.Estado()
	if a != nil {
		t.Fatal("um pedido inválido começou")
	}
	if b, _ := json.Marshal(h); string(b) != "[]" {
		t.Fatalf("histórico vazio sai como %s; a página espera []", b)
	}
}
