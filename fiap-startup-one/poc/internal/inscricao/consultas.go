package inscricao

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Consultas de leitura para a área do candidato. Todas são escopadas pelo candidato.

type Resumo struct {
	ID          int64     `json:"id"`
	CertameID   int64     `json:"certame_id"`
	CertameNome string    `json:"certame_nome"`
	CargoNome   string    `json:"cargo_nome"`
	Estado      Estado    `json:"estado"`
	Atualizada  time.Time `json:"atualizada_em"`
}

func DoCandidato(ctx context.Context, pool *pgxpool.Pool, candidatoID int64) ([]Resumo, error) {
	rows, err := pool.Query(ctx, `
		SELECT i.id, c.id, c.nome, g.nome, i.estado, i.atualizada_em
		FROM inscricao i
		JOIN certame c ON c.id = i.certame_id
		JOIN cargo g ON g.id = i.cargo_id
		WHERE i.candidato_id = $1
		ORDER BY i.atualizada_em DESC`, candidatoID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[Resumo])
}

type EventoTrilha struct {
	Evento Evento    `json:"evento"`
	Para   Estado    `json:"para"`
	Em     time.Time `json:"em"`
}

func Trilha(ctx context.Context, pool *pgxpool.Pool, id, candidatoID int64) ([]EventoTrilha, error) {
	rows, err := pool.Query(ctx, `
		SELECT e.evento, e.para_estado, e.ocorrido_em
		FROM inscricao_evento e JOIN inscricao i ON i.id = e.inscricao_id
		WHERE e.inscricao_id = $1 AND i.candidato_id = $2
		ORDER BY e.id`, id, candidatoID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[EventoTrilha])
}

// MinhaIsencao devolve o pedido de isenção do candidato no certame, ou nil se não houver.
func MinhaIsencao(ctx context.Context, pool *pgxpool.Pool, certameID, candidatoID int64) (*Isencao, error) {
	var i Isencao
	err := pool.QueryRow(ctx, `
		SELECT id, estado, motivo_codigo, solicitada_em, decidida_em FROM isencao
		WHERE certame_id = $1 AND candidato_id = $2`, certameID, candidatoID,
	).Scan(&i.ID, &i.Estado, &i.MotivoCodigo, &i.SolicitadaEm, &i.DecididaEm)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &i, nil
}
