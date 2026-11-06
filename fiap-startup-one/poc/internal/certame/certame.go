// Package certame expõe os dados públicos de um certame. Na arquitetura alvo esta resposta vira
// objeto estático no CDN (frente 2); aqui ela ainda vem do banco.
package certame

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNaoEncontrado = errors.New("certame não encontrado")

type Publico struct {
	ID           int64    `json:"id"`
	Nome         string   `json:"nome"`
	Banca        string   `json:"banca"`
	EditalVersao *int     `json:"edital_versao"`
	Cargos       []Cargo  `json:"cargos"`
	CidadesProva []Cidade `json:"cidades_prova"`
	Cotas        []Cota   `json:"cotas"`
	Cronograma   []Fase   `json:"cronograma"`
}

type Cargo struct {
	ID           int64  `json:"id"`
	Nome         string `json:"nome"`
	Vagas        int    `json:"vagas"`
	TaxaCentavos int    `json:"taxa_centavos"`
}

type Cidade struct {
	ID   int64  `json:"id"`
	Nome string `json:"nome"`
	UF   string `json:"uf"`
}

type Cota struct {
	Codigo     string `json:"codigo"`
	Nome       string `json:"nome"`
	ExigeLaudo bool   `json:"exige_laudo"`
}

type Fase struct {
	Fase    string    `json:"fase"`
	AbreEm  time.Time `json:"abre_em"`
	FechaEm time.Time `json:"fecha_em"`
}

func Buscar(ctx context.Context, pool *pgxpool.Pool, id int64) (Publico, error) {
	p := Publico{ID: id}
	err := pool.QueryRow(ctx, `
		SELECT c.nome, b.nome,
		       (SELECT max(versao) FROM edital_versao WHERE certame_id = c.id AND publicado_em <= now())
		FROM certame c JOIN banca b ON b.id = c.banca_id WHERE c.id = $1`, id,
	).Scan(&p.Nome, &p.Banca, &p.EditalVersao)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, ErrNaoEncontrado
	}
	if err != nil {
		return p, err
	}
	if p.Cargos, err = lista[Cargo](ctx, pool,
		`SELECT id, nome, vagas, taxa_centavos FROM cargo WHERE certame_id = $1 ORDER BY id`, id); err != nil {
		return p, err
	}
	if p.CidadesProva, err = lista[Cidade](ctx, pool,
		`SELECT id, nome, uf FROM cidade_prova WHERE certame_id = $1 ORDER BY nome`, id); err != nil {
		return p, err
	}
	if p.Cotas, err = lista[Cota](ctx, pool,
		`SELECT codigo, nome, exige_laudo FROM cota WHERE certame_id = $1 ORDER BY codigo`, id); err != nil {
		return p, err
	}
	p.Cronograma, err = lista[Fase](ctx, pool,
		`SELECT fase, abre_em, fecha_em FROM certame_fase WHERE certame_id = $1 ORDER BY abre_em`, id)
	return p, err
}

// Aberta diz se a fase está com a janela aberta agora.
func (p Publico) Aberta(fase string) bool {
	agora := time.Now()
	for _, f := range p.Cronograma {
		if f.Fase == fase {
			return !agora.Before(f.AbreEm) && agora.Before(f.FechaEm)
		}
	}
	return false
}

// Janela devolve a janela da fase, se o cronograma a tiver.
func (p Publico) Janela(fase string) *Fase {
	for _, f := range p.Cronograma {
		if f.Fase == fase {
			return &f
		}
	}
	return nil
}

type Item struct {
	ID             int64     `json:"id"`
	Nome           string    `json:"nome"`
	Banca          string    `json:"banca"`
	InscricaoAbre  time.Time `json:"inscricao_abre_em"`
	InscricaoFecha time.Time `json:"inscricao_fecha_em"`
}

// ComInscricao lista os certames cuja inscrição ainda não encerrou.
func ComInscricao(ctx context.Context, pool *pgxpool.Pool) ([]Item, error) {
	return lista[Item](ctx, pool, `
		SELECT c.id, c.nome, b.nome, f.abre_em, f.fecha_em
		FROM certame c
		JOIN banca b ON b.id = c.banca_id
		JOIN certame_fase f ON f.certame_id = c.id AND f.fase = 'inscricao'
		WHERE f.fecha_em > now()
		ORDER BY f.fecha_em`)
}

func lista[T any](ctx context.Context, pool *pgxpool.Pool, sql string, args ...any) ([]T, error) {
	rows, err := pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[T])
}
