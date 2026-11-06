package pagamento

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"lisura/internal/fila"
	"lisura/internal/inscricao"
	"lisura/internal/psp"
)

// TarefaDevolver é o tipo da tarefa que pede ao prestador a devolução de um pagamento.
const TarefaDevolver = "devolver_pagamento"

// Devolvedor é a parte do prestador que devolve dinheiro.
type Devolvedor interface {
	Devolver(ctx context.Context, d psp.NovaDevolucao) (psp.Devolucao, error)
}

// Estados da devolução (ver a migração 00012).
const (
	DevolucaoPendente        = "PENDENTE"
	DevolucaoAguardandoChave = "AGUARDANDO_CHAVE"
	DevolucaoConcluida       = "CONCLUIDA"
	DevolucaoRecusada        = "RECUSADA"
)

type tarefaDevolucao struct {
	DevolucaoID int64 `json:"devolucao_id"`
}

// registrarDevolucao nasce na mesma transação que classificou o pagamento como a mais: o dinheiro
// marcado para voltar e a tarefa que o devolve existem juntos ou não existem. Um pagamento só gera
// uma devolução (chave única), mesmo que o aviso e a conciliação o tragam ao mesmo tempo.
func registrarDevolucao(ctx context.Context, tx pgx.Tx, p psp.Pagamento, c Cobranca, inscricaoID int64, motivo string, prio fila.Prioridade) error {
	var id int64
	err := tx.QueryRow(ctx, `
		INSERT INTO devolucao (psp_pagamento_id, cobranca_id, inscricao_id, valor_centavos, motivo, metodo_origem)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (psp_pagamento_id) DO NOTHING RETURNING id`,
		p.ID, c.ID, inscricaoID, p.ValorCentavos, motivo, c.Metodo).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	return fila.EnfileirarCom(ctx, tx, prio, TarefaDevolver, strconv.FormatInt(id, 10), tarefaDevolucao{id})
}

// Devolver é o manipulador da tarefa. Idempotente: o id da devolução no prestador é derivado do
// pagamento ("dev-" + id do pagamento), então repetir — a resposta perdida, o trabalhador que caiu —
// devolve a mesma devolução, nunca uma segunda. Recusa definitiva do prestador não é repetida: a
// devolução fica AGUARDANDO_CHAVE (o CPF não é chave Pix) ou RECUSADA (alguém precisa olhar).
func Devolver(prestador Devolvedor, pool *pgxpool.Pool) fila.Manipulador {
	return func(ctx context.Context, carga json.RawMessage) error {
		var t tarefaDevolucao
		if err := json.Unmarshal(carga, &t); err != nil {
			return err
		}
		var estado, pagamentoID, metodo, cpf string
		var chave *string
		var valor int
		err := pool.QueryRow(ctx, `
			SELECT d.estado, d.psp_pagamento_id, d.valor_centavos, d.metodo_origem, d.chave_pix, c.cpf
			FROM devolucao d JOIN inscricao i ON i.id = d.inscricao_id JOIN candidato c ON c.id = i.candidato_id
			WHERE d.id = $1`, t.DevolucaoID).Scan(&estado, &pagamentoID, &valor, &metodo, &chave, &cpf)
		if errors.Is(err, pgx.ErrNoRows) || err == nil && estado != DevolucaoPendente {
			return nil
		}
		if err != nil {
			return err
		}
		pedido := psp.NovaDevolucao{ID: "dev-" + pagamentoID, PagamentoID: pagamentoID, ValorCentavos: valor}
		if metodo == "boleto" {
			// O boleto não diz quem pagou. O dinheiro só vai para o próprio candidato: uma chave Pix
			// cujo titular, conferido pelo prestador, é o CPF da conta. Quem tomar a conta não desvia
			// a devolução para outra pessoa.
			pedido.ChavePix, pedido.CPFTitular = cpf, cpf
			if chave != nil {
				pedido.ChavePix = *chave
			}
		}
		dv, err := prestador.Devolver(ctx, pedido)
		var recusa *psp.Recusa
		switch {
		case err == nil:
			_, err = pool.Exec(ctx, `
				UPDATE devolucao SET estado = $2, psp_devolucao_id = $3, concluida_em = $4, ultimo_erro = NULL
				WHERE id = $1 AND estado = $5`, t.DevolucaoID, DevolucaoConcluida, dv.ID, dv.DevolvidaEm, DevolucaoPendente)
			return err
		case errors.As(err, &recusa):
			novo := DevolucaoRecusada
			// Chave que não existe, ou que é de outra pessoa: o dinheiro não saiu, e o candidato pode
			// informar outra chave dele. Tentar de novo é seguro porque o prestador confere o titular.
			if recusa.Motivo == psp.RecusaChaveInexistente || recusa.Motivo == psp.RecusaTitularDiferente {
				novo = DevolucaoAguardandoChave
			} else {
				log.Printf("devolução %d recusada pelo prestador: %s", t.DevolucaoID, recusa.Motivo)
			}
			_, err = pool.Exec(ctx, `UPDATE devolucao SET estado = $2, ultimo_erro = $3 WHERE id = $1 AND estado = $4`,
				t.DevolucaoID, novo, recusa.Motivo, DevolucaoPendente)
			return err
		default:
			return err // passageiro: a fila repete, com espera crescente
		}
	}
}

// RetomarDevolucoes reenfileira as devoluções PENDENTE sem tarefa a caminho: a tarefa abandonada
// depois de 5 tentativas (o prestador fora do ar por mais de meio minuto) e as linhas criadas pela
// migração. Uma devolução nunca morre na fila.
func RetomarDevolucoes(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	rows, err := pool.Query(ctx, `
		SELECT d.id FROM devolucao d
		WHERE d.estado = $1 AND NOT EXISTS (
			SELECT 1 FROM tarefa t WHERE t.tipo = $2 AND t.chave = d.id::text AND t.estado = 'PENDENTE')
		ORDER BY d.id`, DevolucaoPendente, TarefaDevolver)
	if err != nil {
		return 0, err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		return 0, err
	}
	for _, id := range ids {
		if err := fila.Enfileirar(ctx, pool, TarefaDevolver, strconv.FormatInt(id, 10), tarefaDevolucao{id}); err != nil {
			return 0, err
		}
	}
	return len(ids), nil
}

// DevolucaoVista é o que o candidato vê de uma devolução. Recusa diz por que a última chave não
// serviu (CHAVE_INEXISTENTE, TITULAR_DIFERENTE), para a tela pedir outra.
type DevolucaoVista struct {
	ID            int64      `json:"id"`
	ValorCentavos int        `json:"valor_centavos"`
	Motivo        string     `json:"motivo"`
	Estado        string     `json:"estado"`
	CriadaEm      time.Time  `json:"criada_em"`
	ConcluidaEm   *time.Time `json:"concluida_em,omitempty"`
	Recusa        *string    `json:"recusa,omitempty"`
	// ChaveInformada: o candidato já informou uma chave. Sem ela, a recusa é a da chave CPF tentada
	// sozinha, e a tela não diz "a chave que você informou" a quem não informou nenhuma.
	ChaveInformada bool `json:"chave_informada"`
}

var ErrNaoAguardaChave = errors.New("a devolução não está aguardando chave Pix")

// InformarChave é o candidato dizendo para qual chave Pix dele o dinheiro deve voltar, quando o CPF não
// é chave. Só vale enquanto a devolução espera chave. Depois de pedida ao prestador ou concluída, a
// chave não muda: quem tomar a conta não redireciona uma devolução a caminho. E a chave informada só
// recebe se o titular for o CPF da conta, conferido pelo prestador; se não for, a devolução volta a
// esperar, sem dinheiro nenhum ter saído.
func InformarChave(ctx context.Context, pool *pgxpool.Pool, inscricaoID, candidatoID, devolucaoID int64, chave string) error {
	normal, err := NormalizarChavePix(chave)
	if err != nil {
		return errors.Join(inscricao.ErrDadoInvalido, err)
	}
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		var estado string
		err := tx.QueryRow(ctx, `
			SELECT d.estado FROM devolucao d JOIN inscricao i ON i.id = d.inscricao_id
			WHERE d.id = $1 AND d.inscricao_id = $2 AND i.candidato_id = $3
			FOR UPDATE OF d`, devolucaoID, inscricaoID, candidatoID).Scan(&estado)
		if errors.Is(err, pgx.ErrNoRows) {
			return inscricao.ErrNaoEncontrada
		}
		if err != nil {
			return err
		}
		if estado != DevolucaoAguardandoChave {
			return ErrNaoAguardaChave
		}
		if _, err := tx.Exec(ctx, `UPDATE devolucao SET chave_pix = $2, estado = $3, ultimo_erro = NULL WHERE id = $1`,
			devolucaoID, normal, DevolucaoPendente); err != nil {
			return err
		}
		return fila.Enfileirar(ctx, tx, TarefaDevolver, strconv.FormatInt(devolucaoID, 10), tarefaDevolucao{devolucaoID})
	})
}

// DevolucoesDaInscricao lista as devoluções da inscrição do candidato.
func DevolucoesDaInscricao(ctx context.Context, pool *pgxpool.Pool, inscricaoID, candidatoID int64) ([]DevolucaoVista, error) {
	rows, err := pool.Query(ctx, `
		SELECT d.id, d.valor_centavos, d.motivo, d.estado, d.criada_em, d.concluida_em,
		       CASE WHEN d.estado = 'AGUARDANDO_CHAVE' THEN d.ultimo_erro END, d.chave_pix IS NOT NULL
		FROM devolucao d JOIN inscricao i ON i.id = d.inscricao_id
		WHERE d.inscricao_id = $1 AND i.candidato_id = $2 ORDER BY d.id`, inscricaoID, candidatoID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[DevolucaoVista])
}
