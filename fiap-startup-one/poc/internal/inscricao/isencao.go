package inscricao

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Isenção de taxa (02-produto §4): hipótese de lista fechada, documento específico da hipótese,
// triagem por regra, e análise humana só no que a regra não fecha.

type PedidoIsencao struct {
	Hipotese    string `json:"hipotese"` // cadunico | doador_medula
	NIS         string `json:"nis,omitempty"`
	DocumentoID *int64 `json:"documento_id,omitempty"`
}

type Isencao struct {
	ID           int64      `json:"id"`
	Estado       string     `json:"estado"`
	MotivoCodigo *string    `json:"motivo_codigo"`
	SolicitadaEm time.Time  `json:"solicitada_em"`
	DecididaEm   *time.Time `json:"decidida_em"`
}

var ErrIsencaoJaDecidida = errors.New("isenção já decidida")

// SolicitarIsencao valida o formato (erro devolvido ao candidato, que corrige) e aplica a triagem
// por regra (decisão registrada). Um pedido por candidato e certame; repetir devolve o existente.
func SolicitarIsencao(ctx context.Context, pool *pgxpool.Pool, certameID, candidatoID int64, p PedidoIsencao) (Isencao, error) {
	switch p.Hipotese {
	case "cadunico":
		if !NISValido(p.NIS) {
			return Isencao{}, fmt.Errorf("%w: NIS", ErrDadoInvalido)
		}
		p.DocumentoID = nil
	case "doador_medula":
		if p.DocumentoID == nil {
			return Isencao{}, fmt.Errorf("%w: comprovante obrigatório", ErrDadoInvalido)
		}
		p.NIS = ""
	default:
		return Isencao{}, fmt.Errorf("%w: hipótese %q", ErrDadoInvalido, p.Hipotese)
	}

	var out Isencao
	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		if err := prazoAberto(ctx, tx, certameID, "isencao"); err != nil {
			return err
		}
		if p.DocumentoID != nil {
			if err := exigirDocumento(ctx, tx, *p.DocumentoID, candidatoID, "comprovante_isencao"); err != nil {
				return err
			}
		}
		versao, err := versaoEdital(ctx, tx, certameID)
		if err != nil {
			return err
		}

		// Regra que fecha o caso sem análise humana: o mesmo NIS já usado por outro CPF no certame.
		estado, motivo := "EM_ANALISE", ""
		if p.NIS != "" {
			var duplicado bool
			if err := tx.QueryRow(ctx, `
				SELECT EXISTS (SELECT 1 FROM isencao WHERE certame_id = $1 AND nis = $2 AND candidato_id <> $3)`,
				certameID, p.NIS, candidatoID).Scan(&duplicado); err != nil {
				return err
			}
			if duplicado {
				estado, motivo = "INDEFERIDA", "NIS_DE_OUTRO_CANDIDATO"
			}
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO isencao (certame_id, candidato_id, hipotese, nis, documento_id, estado,
			                     motivo_codigo, decidida_por, decidida_em, edital_versao)
			VALUES ($1, $2, $3, NULLIF($4::text, ''), $5, $6, NULLIF($7::text, ''),
			        CASE WHEN $7::text <> '' THEN 'regra' END, CASE WHEN $7::text <> '' THEN now() END, $8)
			ON CONFLICT (certame_id, candidato_id) DO NOTHING`,
			certameID, candidatoID, p.Hipotese, p.NIS, p.DocumentoID, estado, motivo, versao)
		if err != nil {
			return traduzir(err)
		}
		return tx.QueryRow(ctx, `
			SELECT id, estado, motivo_codigo, solicitada_em, decidida_em FROM isencao
			WHERE certame_id = $1 AND candidato_id = $2`, certameID, candidatoID,
		).Scan(&out.ID, &out.Estado, &out.MotivoCodigo, &out.SolicitadaEm, &out.DecididaEm)
	})
	return out, err
}

// DecidirIsencao é a análise humana. Indeferir exige motivo padronizado.
func DecidirIsencao(ctx context.Context, pool *pgxpool.Pool, isencaoID int64, analista Ator, deferir bool, motivo string) error {
	if analista.Tipo != Analista {
		return ErrAtorNaoAutorizado
	}
	estado := "DEFERIDA"
	if !deferir {
		if motivo == "" {
			return ErrMotivoObrigatorio
		}
		estado = "INDEFERIDA"
	} else {
		motivo = ""
	}
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		var certameID int64
		var atual string
		err := tx.QueryRow(ctx, `SELECT certame_id, estado FROM isencao WHERE id = $1 FOR UPDATE`, isencaoID).
			Scan(&certameID, &atual)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNaoEncontrada
		}
		if err != nil {
			return err
		}
		if atual != "EM_ANALISE" {
			return ErrIsencaoJaDecidida
		}
		versao, err := versaoEdital(ctx, tx, certameID)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			UPDATE isencao SET estado = $2, motivo_codigo = NULLIF($3, ''), decidida_por = $4,
			                   decidida_em = now(), edital_versao = $5
			WHERE id = $1`, isencaoID, estado, motivo, analista.ID, versao)
		return err
	})
}

// NISValido confere os 11 dígitos e o dígito verificador do NIS/PIS.
func NISValido(nis string) bool {
	if len(nis) != 11 {
		return false
	}
	pesos := []int{3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
	soma := 0
	for i, r := range nis {
		if r < '0' || r > '9' {
			return false
		}
		if i < 10 {
			soma += int(r-'0') * pesos[i]
		}
	}
	dv := 11 - soma%11
	if dv >= 10 {
		dv = 0
	}
	return int(nis[10]-'0') == dv
}
