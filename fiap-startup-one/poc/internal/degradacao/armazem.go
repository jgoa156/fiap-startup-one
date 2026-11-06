package degradacao

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Config é o estado da sala de espera que todas as instâncias precisam enxergar igual.
type Config struct {
	Nivel   int
	Taxa    float64   // admissões por segundo
	Folga   int64     // quantos entram de imediato quando a sala abre
	AbriuEm time.Time // quando a sala de espera começou a contar
	Base    int64     // senhas emitidas antes desta abertura: já esperaram, entram primeiro
}

// Armazem guarda a Config e numera as senhas. Em memória serve a um processo (testes, POC local);
// no Postgres, a várias instâncias atrás do balanceador.
type Armazem interface {
	Ler(ctx context.Context) (Config, error)
	// Definir troca nível, taxa e folga. Se a sala está abrindo agora, marca AbriuEm = agora e
	// Base = senhas já emitidas, tudo de uma vez.
	Definir(ctx context.Context, nivel int, taxa float64, folga int64, agora time.Time) (Config, error)
	Emitir(ctx context.Context) (int64, error)
	Emitidas(ctx context.Context) (int64, error)
}

// ---- em memória ----

type Memoria struct {
	mu     sync.Mutex
	cfg    Config
	senhas int64
}

func (m *Memoria) Ler(context.Context) (Config, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cfg, nil
}

func (m *Memoria) Definir(_ context.Context, nivel int, taxa float64, folga int64, agora time.Time) (Config, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if nivel >= SalaDeEspera && m.cfg.Nivel < SalaDeEspera {
		// Não zera o contador: uma senha antiga (sala fechada e reaberta) não fura nem perde a vez.
		m.cfg.AbriuEm, m.cfg.Base = agora, m.senhas
	}
	m.cfg.Nivel, m.cfg.Taxa, m.cfg.Folga = nivel, taxa, folga
	return m.cfg, nil
}

func (m *Memoria) Emitir(context.Context) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.senhas++
	return m.senhas, nil
}

func (m *Memoria) Emitidas(context.Context) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.senhas, nil
}

// ---- Postgres ----

type Postgres struct{ Pool *pgxpool.Pool }

func (p Postgres) Ler(ctx context.Context) (Config, error) {
	var c Config
	err := p.Pool.QueryRow(ctx, `SELECT nivel, taxa, folga, abriu_em, base FROM sala_espera WHERE id = 1`).
		Scan(&c.Nivel, &c.Taxa, &c.Folga, &c.AbriuEm, &c.Base)
	return c, err
}

// emitidas lê a sequência sem avançá-la; antes do primeiro nextval, last_value já vale 1.
const emitidas = `(SELECT CASE WHEN is_called THEN last_value ELSE 0 END FROM senha_fila)`

func (p Postgres) Definir(ctx context.Context, nivel int, taxa float64, folga int64, agora time.Time) (Config, error) {
	var c Config
	err := p.Pool.QueryRow(ctx, `
		UPDATE sala_espera SET
			abriu_em = CASE WHEN $1::int >= 1 AND nivel < 1 THEN $4 ELSE abriu_em END,
			base     = CASE WHEN $1::int >= 1 AND nivel < 1 THEN `+emitidas+` ELSE base END,
			nivel = $1, taxa = $2, folga = $3, mudou_em = now()
		WHERE id = 1
		RETURNING nivel, taxa, folga, abriu_em, base`, nivel, taxa, folga, agora).
		Scan(&c.Nivel, &c.Taxa, &c.Folga, &c.AbriuEm, &c.Base)
	return c, err
}

func (p Postgres) Emitir(ctx context.Context) (int64, error) {
	var n int64
	err := p.Pool.QueryRow(ctx, `SELECT nextval('senha_fila')`).Scan(&n)
	return n, err
}

func (p Postgres) Emitidas(ctx context.Context) (int64, error) {
	var n int64
	err := p.Pool.QueryRow(ctx, `SELECT `+emitidas).Scan(&n)
	return n, err
}
