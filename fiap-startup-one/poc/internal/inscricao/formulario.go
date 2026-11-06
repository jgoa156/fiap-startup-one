package inscricao

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"lisura/internal/publicacao"
)

// Os passos do formulário (02-produto §3) salvam um de cada vez. Todos exigem a inscrição em
// RASCUNHO, do próprio candidato, com a janela de inscrição aberta.

// Necessidades de atendimento especial: lista fechada, como o fundamento do recurso.
var Necessidades = []string{
	"ledor", "transcritor", "prova_ampliada", "sala_acessivel", "tempo_adicional", "lactante",
}

func DefinirCargo(ctx context.Context, pool *pgxpool.Pool, id, candidatoID, cargoID int64) error {
	return passo(ctx, pool, id, candidatoID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE inscricao SET cargo_id = $2, atualizada_em = now() WHERE id = $1`, id, cargoID)
		return err
	})
}

func DefinirCidade(ctx context.Context, pool *pgxpool.Pool, id, candidatoID, cidadeID int64) error {
	return passo(ctx, pool, id, candidatoID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE inscricao SET cidade_prova_id = $2, atualizada_em = now() WHERE id = $1`, id, cidadeID)
		return err
	})
}

// DefinirCotas substitui as cotas escolhidas; lista vazia é concorrer só na ampla.
func DefinirCotas(ctx context.Context, pool *pgxpool.Pool, id, candidatoID int64, cotas []string) error {
	return passo(ctx, pool, id, candidatoID, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM inscricao_cota WHERE inscricao_id = $1`, id); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO inscricao_cota (inscricao_id, certame_id, cota_codigo)
			SELECT i.id, i.certame_id, c FROM inscricao i, unnest($2::text[]) AS c WHERE i.id = $1`,
			id, slices.Compact(slices.Sorted(slices.Values(cotas))))
		return err
	})
}

// DefinirAtendimento grava as necessidades e o consentimento de compartilhar o laudo com esta
// banca. Sem laudoID, o consentimento anterior desta inscrição é revogado.
func DefinirAtendimento(ctx context.Context, pool *pgxpool.Pool, id, candidatoID int64, necessidades []string, laudoID *int64) error {
	for _, n := range necessidades {
		if !slices.Contains(Necessidades, n) {
			return fmt.Errorf("%w: necessidade %q", ErrDadoInvalido, n)
		}
	}
	necessidades = slices.Compact(slices.Sorted(slices.Values(necessidades)))
	if necessidades == nil {
		necessidades = []string{} // a coluna é NOT NULL; nil iria como NULL
	}
	return passo(ctx, pool, id, candidatoID, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `UPDATE inscricao SET necessidades = $2, atualizada_em = now() WHERE id = $1`,
			id, necessidades); err != nil {
			return err
		}
		if laudoID == nil {
			_, err := tx.Exec(ctx, `DELETE FROM inscricao_documento WHERE inscricao_id = $1 AND finalidade = 'laudo'`, id)
			return err
		}
		if err := exigirDocumento(ctx, tx, *laudoID, candidatoID, "laudo"); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO inscricao_documento (inscricao_id, candidato_id, finalidade, documento_id)
			VALUES ($1, $2, 'laudo', $3)
			ON CONFLICT (inscricao_id, finalidade)
			DO UPDATE SET documento_id = EXCLUDED.documento_id, consentido_em = now()`,
			id, candidatoID, *laudoID)
		return err
	})
}

// AceitarDeclaracoes registra o aceite; desmarcar (aceito=false) retira o aceite.
func AceitarDeclaracoes(ctx context.Context, pool *pgxpool.Pool, id, candidatoID int64, aceito bool) error {
	return passo(ctx, pool, id, candidatoID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			UPDATE inscricao SET declaracoes_aceitas_em = CASE WHEN $2 THEN now() END, atualizada_em = now()
			WHERE id = $1`, id, aceito)
		return err
	})
}

func passo(ctx context.Context, pool *pgxpool.Pool, id, candidatoID int64, salvar func(pgx.Tx) error) error {
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		var certameID int64
		var estado Estado
		err := tx.QueryRow(ctx,
			`SELECT certame_id, estado FROM inscricao WHERE id = $1 AND candidato_id = $2 FOR UPDATE`,
			id, candidatoID).Scan(&certameID, &estado)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNaoEncontrada
		}
		if err != nil {
			return err
		}
		if estado != Rascunho {
			return ErrNaoEditavel
		}
		if err := prazoAberto(ctx, tx, certameID, "inscricao"); err != nil {
			return err
		}
		return traduzir(salvar(tx))
	})
}

// exigirDocumento confere que o documento existe, é da conta do candidato e é do tipo pedido.
func exigirDocumento(ctx context.Context, tx pgx.Tx, docID, candidatoID int64, tipo string) error {
	var t string
	err := tx.QueryRow(ctx, `SELECT tipo FROM documento WHERE id = $1 AND candidato_id = $2`, docID, candidatoID).Scan(&t)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && t != tipo) {
		return fmt.Errorf("%w: documento %d não é um %s da conta", ErrDadoInvalido, docID, tipo)
	}
	return err
}

// Inscricao é a visão de revisão (passo 7).
type Inscricao struct {
	ID                   int64      `json:"id"`
	CertameID            int64      `json:"certame_id"`
	Estado               Estado     `json:"estado"`
	CargoID              int64      `json:"cargo_id"`
	CidadeProvaID        *int64     `json:"cidade_prova_id"`
	Cotas                []string   `json:"cotas"`
	Necessidades         []string   `json:"necessidades"`
	LaudoDocumentoID     *int64     `json:"laudo_documento_id"`
	DeclaracoesAceitasEm *time.Time `json:"declaracoes_aceitas_em"`
}

func Revisao(ctx context.Context, pool *pgxpool.Pool, id, candidatoID int64) (Inscricao, error) {
	var i Inscricao
	err := pool.QueryRow(ctx, `
		SELECT i.id, i.certame_id, i.estado, i.cargo_id, i.cidade_prova_id,
		       coalesce((SELECT array_agg(cota_codigo ORDER BY cota_codigo) FROM inscricao_cota WHERE inscricao_id = i.id), '{}'),
		       i.necessidades,
		       (SELECT documento_id FROM inscricao_documento WHERE inscricao_id = i.id AND finalidade = 'laudo'),
		       i.declaracoes_aceitas_em
		FROM inscricao i WHERE i.id = $1 AND i.candidato_id = $2`, id, candidatoID,
	).Scan(&i.ID, &i.CertameID, &i.Estado, &i.CargoID, &i.CidadeProvaID, &i.Cotas, &i.Necessidades,
		&i.LaudoDocumentoID, &i.DeclaracoesAceitasEm)
	if errors.Is(err, pgx.ErrNoRows) {
		return i, ErrNaoEncontrada
	}
	return i, err
}

// Finalizar confere o formulário e encaminha a inscrição: com isenção deferida vai para
// ISENCAO_ATIVA, sem ela para AGUARDANDO_PAGAMENTO. Repetir o clique não muda nada.
func Finalizar(ctx context.Context, pool *pgxpool.Pool, id, candidatoID int64) (res Resultado, err error) {
	err = pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		s, err := lerSituacao(ctx, tx, id, candidatoID, true)
		if err != nil {
			return err
		}
		if s.estado == AguardandoPagamento || s.estado == IsencaoAtiva {
			res = Resultado{De: s.estado, Para: s.estado, JaAplicada: true}
			return nil
		}
		if s.estado != Rascunho {
			return ErrNaoEditavel
		}
		if err := prazoAberto(ctx, tx, s.certameID, "inscricao"); err != nil {
			return err
		}
		if len(s.faltando) > 0 {
			return &ErrIncompleta{Faltando: s.faltando}
		}
		ev := FinalizarComPagamento
		if s.isento {
			ev = FinalizarIsento
		}
		res, err = transicionar(ctx, tx, Comando{id, ev, Ator{Candidato, strconv.FormatInt(candidatoID, 10)}, ""})
		if err != nil {
			return err
		}
		// Cartão (se isenta, já vale) e relação do certame são republicados pela fila.
		return publicacao.EnfileirarInscricao(ctx, tx, id, s.certameID)
	})
	return res, err
}

// Pendencias diz o que ainda falta para finalizar, sem alterar nada (painel da tela).
func Pendencias(ctx context.Context, pool *pgxpool.Pool, id, candidatoID int64) ([]string, error) {
	s, err := lerSituacao(ctx, pool, id, candidatoID, false)
	return s.faltando, err
}

type consulta interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type situacao struct {
	certameID int64
	estado    Estado
	isento    bool
	faltando  []string
}

func lerSituacao(ctx context.Context, q consulta, id, candidatoID int64, travar bool) (situacao, error) {
	var s situacao
	var dadosPessoais, cidade, declaracoes, precisaLaudo, temLaudo bool
	sql := `
		SELECT i.certame_id, i.estado,
		       c.data_nascimento IS NOT NULL AND c.municipio IS NOT NULL AND c.uf IS NOT NULL,
		       i.cidade_prova_id IS NOT NULL,
		       i.declaracoes_aceitas_em IS NOT NULL,
		       cardinality(i.necessidades) > 0 OR EXISTS (
		           SELECT 1 FROM inscricao_cota ic
		           JOIN cota ct ON ct.certame_id = ic.certame_id AND ct.codigo = ic.cota_codigo
		           WHERE ic.inscricao_id = i.id AND ct.exige_laudo),
		       EXISTS (SELECT 1 FROM inscricao_documento WHERE inscricao_id = i.id AND finalidade = 'laudo'),
		       EXISTS (SELECT 1 FROM isencao s
		               WHERE s.certame_id = i.certame_id AND s.candidato_id = i.candidato_id AND s.estado = 'DEFERIDA')
		FROM inscricao i JOIN candidato c ON c.id = i.candidato_id
		WHERE i.id = $1 AND i.candidato_id = $2`
	if travar {
		sql += ` FOR UPDATE OF i`
	}
	err := q.QueryRow(ctx, sql, id, candidatoID).Scan(&s.certameID, &s.estado, &dadosPessoais, &cidade,
		&declaracoes, &precisaLaudo, &temLaudo, &s.isento)
	if errors.Is(err, pgx.ErrNoRows) {
		return s, ErrNaoEncontrada
	}
	if err != nil {
		return s, err
	}
	for _, f := range []struct {
		ok    bool
		passo string
	}{
		{dadosPessoais, "dados_pessoais"},
		{cidade, "cidade_prova"},
		{!precisaLaudo || temLaudo, "laudo"},
		{declaracoes, "declaracoes"},
	} {
		if !f.ok {
			s.faltando = append(s.faltando, f.passo)
		}
	}
	return s, nil
}
