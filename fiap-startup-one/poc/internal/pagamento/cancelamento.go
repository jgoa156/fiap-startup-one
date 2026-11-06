package pagamento

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"lisura/internal/certame"
	"lisura/internal/fila"
	"lisura/internal/inscricao"
	"lisura/internal/psp"
	"lisura/internal/publicacao"
)

// MotivoCertameCancelado é o motivo da devolução de um pagamento válido de certame cancelado.
const MotivoCertameCancelado = "CERTAME_CANCELADO"

// MotivoCancelamento é código padronizado, nunca texto livre (como todo motivo da trilha).
var MotivoCancelamento = regexp.MustCompile(`^[A-Z][A-Z0-9_]{2,63}$`)

var ErrMotivoCancelamento = errors.New("motivo do cancelamento: código em maiúsculas, ex.: ANULADO_POR_DECISAO_JUDICIAL")

// Cancelamento conta o que CancelarCertame fez nesta chamada.
type Cancelamento struct {
	Inscricoes int           `json:"inscricoes_canceladas"`
	Devolucoes int           `json:"devolucoes_criadas"`
	Duracao    time.Duration `json:"duracao"`
}

// CancelarCertame é o "reembolso em massa" (02-produto §C): a banca anula ou suspende o certame,
// toda inscrição viva vai a CANCELADA pela máquina de estados (com trilha, motivo e analista) e todo
// pagamento confirmado ganha uma devolução, na fila com prioridade de lote.
//
// Pode ser chamada de novo: continua de onde parou e não repete nada (a transição já aplicada é
// pulada; a devolução tem chave única no pagamento). A ordem fecha as portas antes de varrer:
//  1. o certame é marcado e as cobranças pendentes expiram: não entra inscrição nem cobrança nova, e
//     o pagamento que ainda chegar cai em "pago depois da expiração", que já devolve;
//  2. as inscrições vivas são canceladas;
//  3. os pagamentos confirmados ganham devolução. Depois do passo 2 nenhum pagamento novo é
//     confirmado, então o passo 3 alcança todos.
func CancelarCertame(ctx context.Context, pool *pgxpool.Pool, certameID int64, analista, motivo string) (Cancelamento, error) {
	inicio := time.Now()
	var res Cancelamento
	if !MotivoCancelamento.MatchString(motivo) {
		return res, ErrMotivoCancelamento
	}
	if analista == "" {
		return res, fmt.Errorf("%w: analista", inscricao.ErrDadoInvalido)
	}

	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE certame SET cancelado_em = coalesce(cancelado_em, now()),
			                   cancelamento_motivo = coalesce(cancelamento_motivo, $2)
			WHERE id = $1`, certameID, motivo)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return certame.ErrNaoEncontrado
		}
		if _, err := tx.Exec(ctx, `
			UPDATE cobranca b SET estado = 'EXPIRADA' FROM inscricao i
			WHERE i.id = b.inscricao_id AND i.certame_id = $1 AND b.estado = 'PENDENTE'`, certameID); err != nil {
			return err
		}
		return publicacao.EnfileirarCertame(ctx, tx, certameID)
	})
	if err != nil {
		return res, err
	}

	if res.Inscricoes, err = cancelarInscricoes(ctx, pool, certameID, inscricao.Ator{Tipo: inscricao.Analista, ID: analista}, motivo); err != nil {
		return res, err
	}
	for {
		n, err := devolverLote(ctx, pool, certameID, 1000)
		res.Devolucoes += n
		if err != nil {
			return res, err
		}
		if n == 0 {
			break
		}
	}
	res.Duracao = time.Since(inicio)
	return res, nil
}

// cancelarInscricoes aplica CANCELAR_CERTAME a cada inscrição viva, uma transação por inscrição e 8
// de cada vez (como a homologação em lote). O cartão é republicado com a situação nova.
func cancelarInscricoes(ctx context.Context, pool *pgxpool.Pool, certameID int64, ator inscricao.Ator, motivo string) (int, error) {
	rows, err := pool.Query(ctx, `
		SELECT id FROM inscricao WHERE certame_id = $1 AND estado NOT IN ('CANCELADA', 'EXPIRADA') ORDER BY id`, certameID)
	if err != nil {
		return 0, err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		return 0, err
	}
	var (
		mu        sync.Mutex
		feitas    int
		falha     error
		wg        sync.WaitGroup
		pendentes = make(chan int64)
	)
	for range 8 {
		wg.Go(func() {
			for id := range pendentes {
				var aplicou bool
				err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
					r, err := inscricao.TransicionarTx(ctx, tx, inscricao.Comando{
						InscricaoID: id, Evento: inscricao.CancelarCertame, Ator: ator, Motivo: motivo})
					if errors.Is(err, inscricao.ErrTransicaoInvalida) {
						return nil // expirou entre a lista e a vez dela: nada a cancelar
					}
					if err != nil || r.JaAplicada {
						return err
					}
					aplicou = true
					return publicacao.EnfileirarInscricaoLote(ctx, tx, id, certameID)
				})
				mu.Lock()
				switch {
				case err != nil && falha == nil:
					falha = fmt.Errorf("cancelar inscrição %d: %w", id, err)
				case err == nil && aplicou:
					feitas++
				}
				mu.Unlock()
			}
		})
	}
	for _, id := range ids {
		pendentes <- id
	}
	close(pendentes)
	wg.Wait()
	return feitas, falha
}

// devolverLote cria, numa transação, a devolução de até n pagamentos confirmados do certame que ainda
// não têm devolução. Devolve quantas criou; zero quando acabou.
func devolverLote(ctx context.Context, pool *pgxpool.Pool, certameID int64, n int) (int, error) {
	var feitas int
	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT p.psp_pagamento_id, p.valor_centavos, b.id, b.metodo, b.inscricao_id
			FROM pagamento_recebido p JOIN cobranca b ON b.id = p.cobranca_id JOIN inscricao i ON i.id = b.inscricao_id
			WHERE i.certame_id = $1 AND p.resultado = $2
			  AND NOT EXISTS (SELECT 1 FROM devolucao d WHERE d.psp_pagamento_id = p.psp_pagamento_id)
			ORDER BY p.psp_pagamento_id LIMIT $3`, certameID, Confirmado, n)
		if err != nil {
			return err
		}
		type linha struct {
			pagamento   psp.Pagamento
			cobranca    Cobranca
			inscricaoID int64
		}
		var l linha
		var lote []linha
		if _, err := pgx.ForEachRow(rows, []any{&l.pagamento.ID, &l.pagamento.ValorCentavos, &l.cobranca.ID,
			&l.cobranca.Metodo, &l.inscricaoID}, func() error {
			lote = append(lote, l)
			return nil
		}); err != nil {
			return err
		}
		for _, x := range lote {
			if err := registrarDevolucao(ctx, tx, x.pagamento, x.cobranca, x.inscricaoID, MotivoCertameCancelado, fila.Lote); err != nil {
				return err
			}
		}
		feitas = len(lote)
		return nil
	})
	return feitas, err
}
