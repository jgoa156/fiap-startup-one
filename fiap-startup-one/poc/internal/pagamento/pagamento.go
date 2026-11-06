// Package pagamento orquestra a cobrança da taxa: pede ao prestador (PSP), recebe o aviso de que
// o dinheiro entrou, confere contra o extrato e expira o que venceu. Nunca cobra duas vezes, e o
// que entrar a mais volta para quem pagou (devolucao.go), uma vez só.
package pagamento

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"lisura/internal/fila"
	"lisura/internal/inscricao"
	"lisura/internal/psp"
	"lisura/internal/publicacao"
)

type Prestador interface {
	CriarCobranca(ctx context.Context, n psp.NovaCobranca) (psp.Cobranca, error)
	Extrato(ctx context.Context, desde, ate time.Time) ([]psp.Pagamento, error)
}

var (
	ErrNaoAguardaPagamento = errors.New("a inscrição não está aguardando pagamento")
	ErrPrazoPagamento      = errors.New("prazo de pagamento encerrado")
	ErrMetodo              = errors.New("método de pagamento inválido")
)

// ValidadePix: o Pix gerado vale por um dia (ou até o fim do prazo de pagamento, o que vier antes).
const ValidadePix = 24 * time.Hour

// MargemCompensacao: um boleto pago no último dia pode levar até 3 dias úteis para ser informado.
// A inscrição só expira depois disso.
const MargemCompensacao = 3 * 24 * time.Hour

type executor interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

var sistema = inscricao.Ator{Tipo: inscricao.Sistema, ID: "pagamento"}

type Cobranca struct {
	ID                int64      `json:"id"`
	Metodo            string     `json:"metodo"`
	ValorCentavos     int        `json:"valor_centavos"`
	Estado            string     `json:"estado"`
	PixCopiaCola      *string    `json:"pix_copia_cola,omitempty"`
	LinhaDigitavel    *string    `json:"linha_digitavel,omitempty"`
	VenceEm           time.Time  `json:"vence_em"`
	PagaEm            *time.Time `json:"paga_em,omitempty"`
	ReembolsoPendente bool       `json:"reembolso_pendente"`
	PSPID             *string    `json:"-"`
}

const colunas = `id, metodo, valor_centavos, estado, pix_copia_cola, linha_digitavel, vence_em, paga_em,
	reembolso_pendente, psp_id`

func ler(row pgx.Row) (Cobranca, error) {
	var c Cobranca
	err := row.Scan(&c.ID, &c.Metodo, &c.ValorCentavos, &c.Estado, &c.PixCopiaCola, &c.LinhaDigitavel,
		&c.VenceEm, &c.PagaEm, &c.ReembolsoPendente, &c.PSPID)
	return c, err
}

// Cobrar gera (ou devolve) a cobrança do método escolhido. Duplo clique, recarregamento e reenvio
// devolvem a mesma cobrança: o banco aceita uma só PENDENTE por inscrição e método.
//
// O prestador NÃO é chamado aqui: a cobrança e a tarefa que vai criá-la no prestador nascem na
// mesma transação, e o trabalhador da fila faz a chamada (CriarNoPrestador). No pico, um prestador
// lento atrasa só o aparecimento do Pix na tela, não segura a requisição nem uma conexão do banco.
func Cobrar(ctx context.Context, pool *pgxpool.Pool, inscricaoID, candidatoID int64, metodo string) (Cobranca, error) {
	if metodo != "pix" && metodo != "boleto" {
		return Cobranca{}, ErrMetodo
	}
	var c Cobranca
	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		var estado inscricao.Estado
		var taxa int
		var fimPagamento *time.Time
		var cancelado bool
		err := tx.QueryRow(ctx, `
			SELECT i.estado, g.taxa_centavos,
			       (SELECT fecha_em FROM certame_fase WHERE certame_id = i.certame_id AND fase = 'pagamento'),
			       (SELECT cancelado_em IS NOT NULL FROM certame WHERE id = i.certame_id)
			FROM inscricao i JOIN cargo g ON g.id = i.cargo_id
			WHERE i.id = $1 AND i.candidato_id = $2
			FOR UPDATE OF i`, inscricaoID, candidatoID).Scan(&estado, &taxa, &fimPagamento, &cancelado)
		if errors.Is(err, pgx.ErrNoRows) {
			return inscricao.ErrNaoEncontrada
		}
		if err != nil {
			return err
		}
		if cancelado {
			return inscricao.ErrCertameCancelado
		}
		if estado != inscricao.AguardandoPagamento {
			return ErrNaoAguardaPagamento
		}
		if fimPagamento == nil || !time.Now().Before(*fimPagamento) {
			return ErrPrazoPagamento
		}
		// Cobrança vencida não é devolvida como se valesse: sai do caminho e uma nova é gerada.
		if _, err := tx.Exec(ctx, `
			UPDATE cobranca SET estado = 'EXPIRADA'
			WHERE inscricao_id = $1 AND estado = 'PENDENTE' AND vence_em <= now()`, inscricaoID); err != nil {
			return err
		}
		vence := *fimPagamento
		if metodo == "pix" && time.Now().Add(ValidadePix).Before(vence) {
			vence = time.Now().Add(ValidadePix)
		}
		c, err = ler(tx.QueryRow(ctx, `
			INSERT INTO cobranca (inscricao_id, metodo, valor_centavos, vence_em) VALUES ($1, $2, $3, $4)
			ON CONFLICT (inscricao_id, metodo) WHERE estado = 'PENDENTE' DO NOTHING
			RETURNING `+colunas, inscricaoID, metodo, taxa, vence))
		if errors.Is(err, pgx.ErrNoRows) {
			c, err = ler(tx.QueryRow(ctx, `SELECT `+colunas+` FROM cobranca
				WHERE inscricao_id = $1 AND metodo = $2 AND estado = 'PENDENTE'`, inscricaoID, metodo))
		}
		if err != nil || c.PSPID != nil {
			return err
		}
		// Urgente: o candidato está na tela esperando o Pix aparecer.
		return fila.EnfileirarCom(ctx, tx, fila.Urgente, TarefaCriarNoPrestador, strconv.FormatInt(c.ID, 10), tarefaCobranca{c.ID})
	})
	return c, err
}

// TarefaCriarNoPrestador é o tipo da tarefa que cria a cobrança no prestador.
const TarefaCriarNoPrestador = "criar_cobranca_no_prestador"

type tarefaCobranca struct {
	CobrancaID int64 `json:"cobranca_id"`
}

// CriarNoPrestador é o manipulador da tarefa: pede a cobrança ao prestador e guarda o Pix ou a linha
// digitável. Idempotente: se já tem psp_id, não faz nada; se repetir a chamada, o prestador recebe
// a mesma chave de idempotência (nosso id) e devolve a mesma cobrança.
func CriarNoPrestador(prestador Prestador, pool *pgxpool.Pool) func(context.Context, json.RawMessage) error {
	return func(ctx context.Context, carga json.RawMessage) error {
		var t tarefaCobranca
		if err := json.Unmarshal(carga, &t); err != nil {
			return err
		}
		c, err := ler(pool.QueryRow(ctx, `SELECT `+colunas+` FROM cobranca WHERE id = $1`, t.CobrancaID))
		if err != nil || c.PSPID != nil || c.Estado != "PENDENTE" {
			return err
		}
		r, err := prestador.CriarCobranca(ctx, psp.NovaCobranca{
			Referencia: c.ID, ValorCentavos: c.ValorCentavos, Metodo: c.Metodo, VenceEm: c.VenceEm})
		if err != nil {
			return err
		}
		_, err = pool.Exec(ctx, `
			UPDATE cobranca SET psp_id = $2, pix_copia_cola = NULLIF($3, ''), linha_digitavel = NULLIF($4, '')
			WHERE id = $1 AND psp_id IS NULL`, c.ID, r.ID, r.PixCopiaCola, r.LinhaDigitavel)
		return err
	}
}

// Resultados do processamento de um pagamento informado pelo prestador.
const (
	Confirmado       = "CONFIRMADO"
	JaProcessado     = "JA_PROCESSADO"
	EmDuplicidade    = "PAGO_EM_DUPLICIDADE"
	AposExpiracao    = "PAGO_APOS_EXPIRACAO"
	ValorDivergente  = "VALOR_DIVERGENTE"
	CobrancaDesconhe = "COBRANCA_DESCONHECIDA"
)

// RegistrarPagamento aplica um pagamento informado pelo prestador, pelo aviso ou pela conciliação.
// Cada pagamento é aplicado uma vez só: a chave primária de pagamento_recebido barra as repetições,
// mesmo simultâneas, e tudo — registro, cobrança e inscrição — muda na mesma transação.
func RegistrarPagamento(ctx context.Context, pool *pgxpool.Pool, p psp.Pagamento, origem string) (resultado string, err error) {
	err = pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			INSERT INTO pagamento_recebido (psp_pagamento_id, valor_centavos, pago_em, origem, resultado)
			VALUES ($1, $2, $3, $4, '')
			ON CONFLICT (psp_pagamento_id) DO NOTHING`, p.ID, p.ValorCentavos, p.PagoEm, origem)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			resultado = JaProcessado
			return nil
		}

		resultado, err = aplicar(ctx, tx, p)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			UPDATE pagamento_recebido SET resultado = $2,
			       cobranca_id = (SELECT id FROM cobranca WHERE id = $3)
			WHERE psp_pagamento_id = $1`, p.ID, resultado, p.Referencia)
		return err
	})
	return resultado, err
}

func aplicar(ctx context.Context, tx pgx.Tx, p psp.Pagamento) (string, error) {
	var c Cobranca
	var inscricaoID, certameID int64
	var estadoInscricao inscricao.Estado
	err := tx.QueryRow(ctx, `
		SELECT c.id, c.metodo, c.valor_centavos, c.estado, c.inscricao_id, i.estado, i.certame_id
		FROM cobranca c JOIN inscricao i ON i.id = c.inscricao_id
		WHERE c.id = $1
		FOR UPDATE OF c, i`, p.Referencia).Scan(&c.ID, &c.Metodo, &c.ValorCentavos, &c.Estado, &inscricaoID, &estadoInscricao, &certameID)
	if errors.Is(err, pgx.ErrNoRows) {
		return CobrancaDesconhe, divergir(ctx, tx, "COBRANCA_DESCONHECIDA", p.ID, nil,
			fmt.Sprintf("pagamento de %d centavos para referência %d inexistente", p.ValorCentavos, p.Referencia))
	}
	if err != nil {
		return "", err
	}
	if p.ValorCentavos != c.ValorCentavos {
		return ValorDivergente, divergir(ctx, tx, "VALOR_DIVERGENTE", p.ID, &c.ID,
			fmt.Sprintf("pago %d, cobrado %d", p.ValorCentavos, c.ValorCentavos))
	}

	novoEstado, resultado := "PAGA", Confirmado
	switch {
	case c.Estado == "PAGA" || estadoInscricao != inscricao.AguardandoPagamento && estadoInscricao != inscricao.Expirada &&
		estadoInscricao != inscricao.Cancelada:
		// A inscrição já foi paga (por esta ou por outra cobrança): o dinheiro tem de voltar.
		novoEstado, resultado = "PAGA_EM_DUPLICIDADE", EmDuplicidade
	case estadoInscricao != inscricao.AguardandoPagamento:
		novoEstado, resultado = "PAGA_APOS_EXPIRACAO", AposExpiracao
	}

	if resultado != Confirmado {
		// O dinheiro entrou sem dever: a devolução e a tarefa que a faz nascem aqui, na mesma transação.
		if err := registrarDevolucao(ctx, tx, p, c, inscricaoID, resultado, fila.Normal); err != nil {
			return "", err
		}
	}
	if resultado == Confirmado {
		if _, err := inscricao.TransicionarTx(ctx, tx, inscricao.Comando{
			InscricaoID: inscricaoID, Evento: inscricao.ConfirmarPagamento, Ator: sistema}); err != nil {
			return "", err
		}
		// A inscrição passou a valer: o cartão de confirmação é publicado pela fila.
		if err := publicacao.EnfileirarInscricao(ctx, tx, inscricaoID, certameID); err != nil {
			return "", err
		}
	}
	if c.Estado == "PAGA" {
		// A própria cobrança já estava paga (boleto pago duas vezes): a linha não muda de estado;
		// o reembolso fica marcado nela.
		_, err = tx.Exec(ctx, `UPDATE cobranca SET reembolso_pendente = true WHERE id = $1`, c.ID)
		return resultado, err
	}
	_, err = tx.Exec(ctx, `
		UPDATE cobranca SET estado = $2, paga_em = $3, valor_pago_centavos = $4,
		       reembolso_pendente = ($2 <> 'PAGA')
		WHERE id = $1`, c.ID, novoEstado, p.PagoEm, p.ValorCentavos)
	return resultado, err
}

func divergir(ctx context.Context, tx executor, tipo, pagamentoID string, cobrancaID *int64, detalhe string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO divergencia (tipo, psp_pagamento_id, cobranca_id, detalhe) VALUES ($1, $2, $3, $4)
		ON CONFLICT (tipo, psp_pagamento_id) DO NOTHING`, tipo, pagamentoID, cobrancaID, detalhe)
	return err
}

// ExpirarVencidas é o trabalho periódico: marca cobranças vencidas e expira as inscrições cujo
// prazo de pagamento acabou há mais que a margem de compensação, sem nenhum pagamento a caminho.
func ExpirarVencidas(ctx context.Context, pool *pgxpool.Pool, margem time.Duration) (expiradas int, err error) {
	if _, err := pool.Exec(ctx, `
		UPDATE cobranca SET estado = 'EXPIRADA'
		WHERE estado = 'PENDENTE' AND vence_em + $1::float8 * interval '1 second' <= now()`,
		margem.Seconds()); err != nil {
		return 0, err
	}
	rows, err := pool.Query(ctx, `
		SELECT i.id, i.certame_id FROM inscricao i
		JOIN certame_fase f ON f.certame_id = i.certame_id AND f.fase = 'pagamento'
		WHERE i.estado = 'AGUARDANDO_PAGAMENTO'
		  AND f.fecha_em + $1::float8 * interval '1 second' <= now()
		  AND NOT EXISTS (SELECT 1 FROM cobranca c WHERE c.inscricao_id = i.id AND c.estado = 'PENDENTE')`,
		margem.Seconds())
	if err != nil {
		return 0, err
	}
	type vencida struct{ ID, Certame int64 }
	vencidas, err := pgx.CollectRows(rows, pgx.RowToStructByPos[vencida])
	if err != nil {
		return 0, err
	}
	porCertame := map[int64][]int64{}
	for _, v := range vencidas {
		r, err := inscricao.Transicionar(ctx, pool, inscricao.Comando{InscricaoID: v.ID, Evento: inscricao.Expirar, Ator: sistema})
		switch {
		case errors.Is(err, inscricao.ErrTransicaoInvalida):
			// Um pagamento chegou entre a consulta e a trava: ele venceu, e é o certo.
		case err != nil:
			return expiradas, err
		case !r.JaAplicada:
			expiradas++
			porCertame[v.Certame] = append(porCertame[v.Certame], v.ID)
		}
	}
	// A relação pública mostra as expiradas: só as fatias delas são refeitas.
	for c, ids := range porCertame {
		if err := publicacao.EnfileirarFatiasDe(ctx, pool, c, ids); err != nil {
			return expiradas, err
		}
	}
	return expiradas, nil
}

// DaInscricao lista as cobranças da inscrição, mais recente primeiro.
func DaInscricao(ctx context.Context, pool *pgxpool.Pool, inscricaoID, candidatoID int64) ([]Cobranca, error) {
	rows, err := pool.Query(ctx, `
		SELECT `+colunas+` FROM cobranca c
		WHERE c.inscricao_id = $1 AND EXISTS (SELECT 1 FROM inscricao i WHERE i.id = $1 AND i.candidato_id = $2)
		ORDER BY c.id DESC`, inscricaoID, candidatoID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Cobranca, error) { return ler(r) })
}

// PSPIDDaCobranca serve ao botão de sandbox ("simular que paguei").
func PSPIDDaCobranca(ctx context.Context, pool *pgxpool.Pool, cobrancaID, candidatoID int64) (string, error) {
	var id *string
	err := pool.QueryRow(ctx, `
		SELECT c.psp_id FROM cobranca c JOIN inscricao i ON i.id = c.inscricao_id
		WHERE c.id = $1 AND i.candidato_id = $2`, cobrancaID, candidatoID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && id == nil) {
		return "", inscricao.ErrNaoEncontrada
	}
	if err != nil {
		return "", err
	}
	return *id, nil
}
