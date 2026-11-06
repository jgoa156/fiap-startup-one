package degradacao

import (
	"context"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"lisura/internal/dbtest"
)

var pool *pgxpool.Pool

func TestMain(m *testing.M) {
	p, err := dbtest.Pool(context.Background(), "teste_degradacao")
	if err != nil {
		dbtest.SemBanco(err)
		os.Exit(m.Run()) // LISURA_SEM_BANCO: roda os testes que não precisam de banco
	}
	pool = p
	code := m.Run()
	p.Close()
	os.Exit(code)
}

func precisaBanco(t *testing.T) {
	t.Helper()
	if pool == nil {
		t.Skip("sem banco (LISURA_SEM_BANCO)")
	}
	if _, err := pool.Exec(context.Background(), `
		UPDATE sala_espera SET nivel = 0, taxa = 0, folga = 0, base = 0;
		ALTER SEQUENCE senha_fila RESTART`); err != nil {
		t.Fatal(err)
	}
}

// Duas instâncias da API atrás do balanceador, o mesmo banco: a fila tem de ser uma só.
func TestDuasInstanciasUmaFila(t *testing.T) {
	precisaBanco(t)
	rel := &relogio{t: time.Date(2026, 11, 4, 9, 0, 0, 0, time.UTC)}
	nova := func() (*Controle, http.Handler) {
		c := &Controle{Chave: []byte("chave"), Agora: rel.agora, Armazem: Postgres{Pool: pool}}
		if err := c.Atualizar(context.Background()); err != nil {
			t.Fatal(err)
		}
		ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
		return c, c.Middleware(ok)
	}
	a, ha := nova()
	b, hb := nova()

	if err := a.Definir(SalaDeEspera, 1, 1); err != nil {
		t.Fatal(err)
	}
	// b ainda não releu: segue no nível 0 até o próximo Atualizar (na produção, até 1 s).
	if b.config().Nivel != Normal {
		t.Fatal("b mudou de nível sem reler")
	}
	if err := b.Atualizar(context.Background()); err != nil {
		t.Fatal(err)
	}
	if b.config().Nivel != SalaDeEspera {
		t.Fatalf("b não viu o nível trocado por a: %+v", b.config())
	}

	// O 1º visitante, por a, ocupa a folga; o 2º, por b, tem de ser o 1º da fila — não outra senha 1.
	if w := (&visitante{}).pedir(ha, "GET", "/eu"); w.Code != 200 {
		t.Fatalf("1º visitante deveria entrar: %d", w.Code)
	}
	segundo := &visitante{}
	if r := lerRecusa(t, segundo.pedir(hb, "GET", "/eu")); r.Fila == nil || r.Fila.Posicao != 1 {
		t.Fatalf("numeração não é compartilhada: %+v", r.Fila)
	}
	// A senha do 2º vale na outra instância: ele recarrega e cai em a, com a mesma posição.
	if r := lerRecusa(t, segundo.pedir(ha, "GET", "/eu")); r.Fila.Posicao != 1 {
		t.Fatalf("senha não vale em outra instância: %+v", r.Fila)
	}
	rel.andar(time.Second)
	if w := segundo.pedir(ha, "GET", "/eu"); w.Code != 200 {
		t.Fatalf("após 1 s o 2º deveria entrar: %d", w.Code)
	}
	if e := a.Estado(); e.Senhas != 2 {
		t.Fatalf("esperava 2 senhas emitidas, vieram %d", e.Senhas)
	}
}

func TestSenhasConcorrentesNaoRepetem(t *testing.T) {
	precisaBanco(t)
	arm := Postgres{Pool: pool}
	const n = 200
	vistas := make(chan int64, n)
	var wg sync.WaitGroup
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s, err := arm.Emitir(context.Background())
			if err != nil {
				t.Error(err)
				return
			}
			vistas <- s
		}()
	}
	wg.Wait()
	close(vistas)
	unicas := map[int64]bool{}
	for s := range vistas {
		if unicas[s] {
			t.Fatalf("senha %d emitida duas vezes", s)
		}
		unicas[s] = true
	}
	if len(unicas) != n {
		t.Fatalf("esperava %d senhas, vieram %d", n, len(unicas))
	}
	if e, _ := arm.Emitidas(context.Background()); e != n {
		t.Fatalf("Emitidas = %d, esperava %d", e, n)
	}
}

func TestPostgresReaberturaGuardaABase(t *testing.T) {
	precisaBanco(t)
	arm := Postgres{Pool: pool}
	ctx := context.Background()
	agora := time.Now()
	if _, err := arm.Definir(ctx, SalaDeEspera, 0, 0, agora); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		arm.Emitir(ctx)
	}
	cfg, _ := arm.Definir(ctx, SoEssencial, 5, 0, agora.Add(time.Hour)) // já estava aberta: não reabre
	if cfg.Base != 0 || !cfg.AbriuEm.Equal(agora.Truncate(time.Microsecond)) {
		t.Fatalf("subir de 1 para 2 não deveria reabrir a sala: %+v", cfg)
	}
	arm.Definir(ctx, Normal, 0, 0, agora)
	cfg, _ = arm.Definir(ctx, SalaDeEspera, 1, 0, agora.Add(2*time.Hour))
	if cfg.Base != 3 {
		t.Fatalf("na reabertura, as 3 senhas antigas deviam entrar primeiro: base = %d", cfg.Base)
	}
}
