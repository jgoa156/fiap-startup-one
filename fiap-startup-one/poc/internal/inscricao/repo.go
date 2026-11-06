package inscricao

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"lisura/internal/notificacao"
)

var (
	ErrNaoEncontrada      = errors.New("inscrição não encontrada")
	ErrTransicaoInvalida  = errors.New("transição inválida")
	ErrAtorNaoAutorizado  = errors.New("ator não autorizado para este evento")
	ErrMotivoObrigatorio  = errors.New("motivo padronizado obrigatório")
	ErrEditalNaoPublicado = errors.New("certame sem edital publicado")
	ErrEventoDesconhecido = errors.New("evento desconhecido")
	ErrForaDoPrazo        = errors.New("fora do prazo")
	ErrNaoEditavel        = errors.New("inscrição já finalizada não pode ser alterada")
	ErrDadoInvalido       = errors.New("dado inválido")
	ErrCertameCancelado   = errors.New("certame cancelado")
)

// ErrIncompleta lista o que falta para finalizar a inscrição.
type ErrIncompleta struct{ Faltando []string }

func (e *ErrIncompleta) Error() string {
	return "inscrição incompleta: falta " + strings.Join(e.Faltando, ", ")
}

type Ator struct {
	Tipo AtorTipo
	ID   string
}

type Comando struct {
	InscricaoID int64
	Evento      Evento
	Ator        Ator
	Motivo      string // código de motivo padronizado, nunca texto livre
}

type Resultado struct {
	De, Para Estado
	EventoID int64
	// JaAplicada indica um reenvio: a inscrição já estava no destino e nada foi gravado.
	JaAplicada bool
}

// Abrir abre a inscrição em RASCUNHO e o primeiro registro da trilha. É idempotente: reenviar
// para o mesmo candidato e certame devolve a inscrição existente, sem duplicar.
func Abrir(ctx context.Context, pool *pgxpool.Pool, certameID, candidatoID, cargoID int64, ator Ator) (id int64, criada bool, err error) {
	err = pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		if err := prazoAberto(ctx, tx, certameID, "inscricao"); err != nil {
			return err
		}
		err := tx.QueryRow(ctx, `
			INSERT INTO inscricao (certame_id, candidato_id, cargo_id)
			VALUES ($1, $2, $3)
			ON CONFLICT (certame_id, candidato_id) DO NOTHING
			RETURNING id`, certameID, candidatoID, cargoID).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			return tx.QueryRow(ctx,
				`SELECT id FROM inscricao WHERE certame_id = $1 AND candidato_id = $2`,
				certameID, candidatoID).Scan(&id)
		}
		if err != nil {
			return traduzir(err)
		}
		criada = true
		versao, err := versaoEdital(ctx, tx, certameID)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO inscricao_evento
				(inscricao_id, evento, de_estado, para_estado, ator_tipo, ator_id, edital_versao)
			VALUES ($1, $2, '', $3, $4, $5, $6)`,
			id, Criar, Rascunho, ator.Tipo, ator.ID, versao)
		return err
	})
	return id, criada, err
}

// Transicionar aplica um evento à inscrição. Estado e trilha mudam na mesma transação; a linha
// fica travada durante a decisão, então eventos concorrentes sobre a mesma inscrição são
// serializados e só um vence.
func Transicionar(ctx context.Context, pool *pgxpool.Pool, cmd Comando) (res Resultado, err error) {
	err = pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		res, err = transicionar(ctx, tx, cmd)
		return err
	})
	return res, err
}

// TransicionarTx aplica o evento dentro de uma transação já aberta: para quem precisa mudar o estado
// da inscrição junto com outra coisa (o pagamento) — ou tudo, ou nada.
func TransicionarTx(ctx context.Context, tx pgx.Tx, cmd Comando) (Resultado, error) {
	return transicionar(ctx, tx, cmd)
}

func transicionar(ctx context.Context, tx pgx.Tx, cmd Comando) (Resultado, error) {
	t, ok := Transicoes[cmd.Evento]
	if !ok {
		return Resultado{}, fmt.Errorf("%w: %s", ErrEventoDesconhecido, cmd.Evento)
	}
	if cmd.Ator.Tipo != t.Ator {
		return Resultado{}, fmt.Errorf("%w: %s exige %s, recebeu %s", ErrAtorNaoAutorizado, cmd.Evento, t.Ator, cmd.Ator.Tipo)
	}
	if t.ExigeMotivo && cmd.Motivo == "" {
		return Resultado{}, fmt.Errorf("%w: %s", ErrMotivoObrigatorio, cmd.Evento)
	}

	var atual Estado
	var certameID int64
	err := tx.QueryRow(ctx,
		`SELECT estado, certame_id FROM inscricao WHERE id = $1 FOR UPDATE`,
		cmd.InscricaoID).Scan(&atual, &certameID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Resultado{}, ErrNaoEncontrada
	}
	if err != nil {
		return Resultado{}, err
	}

	destino, ok := Proximo(atual, cmd.Evento)
	if !ok {
		if atual == t.Destino {
			return Resultado{De: atual, Para: atual, JaAplicada: true}, nil
		}
		return Resultado{}, fmt.Errorf("%w: %s a partir de %s", ErrTransicaoInvalida, cmd.Evento, atual)
	}

	versao, err := versaoEdital(ctx, tx, certameID)
	if err != nil {
		return Resultado{}, err
	}
	if _, err := tx.Exec(ctx,
		`UPDATE inscricao SET estado = $2, atualizada_em = now() WHERE id = $1`,
		cmd.InscricaoID, destino); err != nil {
		return Resultado{}, err
	}
	res := Resultado{De: atual, Para: destino}
	err = tx.QueryRow(ctx, `
		INSERT INTO inscricao_evento
			(inscricao_id, evento, de_estado, para_estado, ator_tipo, ator_id, motivo_codigo, edital_versao)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), $8)
		RETURNING id`,
		cmd.InscricaoID, cmd.Evento, atual, destino, cmd.Ator.Tipo, cmd.Ator.ID, cmd.Motivo, versao,
	).Scan(&res.EventoID)
	if err != nil {
		return res, err
	}
	// O aviso ao candidato nasce com o evento, na mesma transação (se o evento avisa).
	return res, notificacao.Enfileirar(ctx, tx, res.EventoID, string(cmd.Evento))
}

// versaoEdital é a versão do edital em vigor agora: a régua sob a qual a decisão é tomada.
func versaoEdital(ctx context.Context, tx pgx.Tx, certameID int64) (int, error) {
	var v *int
	if err := tx.QueryRow(ctx,
		`SELECT max(versao) FROM edital_versao WHERE certame_id = $1 AND publicado_em <= now()`,
		certameID).Scan(&v); err != nil {
		return 0, err
	}
	if v == nil {
		return 0, ErrEditalNaoPublicado
	}
	return *v, nil
}

// prazoAberto confere a janela da fase no cronograma, pelo relógio do banco. Certame cancelado não
// tem prazo aberto nenhum.
func prazoAberto(ctx context.Context, tx pgx.Tx, certameID int64, fase string) error {
	var aberto, cancelado bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM certame_fase
		               WHERE certame_id = $1 AND fase = $2 AND abre_em <= now() AND now() < fecha_em),
		       EXISTS (SELECT 1 FROM certame WHERE id = $1 AND cancelado_em IS NOT NULL)`,
		certameID, fase).Scan(&aberto, &cancelado); err != nil {
		return err
	}
	if cancelado {
		return ErrCertameCancelado
	}
	if !aberto {
		return fmt.Errorf("%w: %s", ErrForaDoPrazo, fase)
	}
	return nil
}

// traduzir converte violação de chave estrangeira ou de CHECK em ErrDadoInvalido: é o banco
// recusando cargo, cidade, cota ou documento que não pertence ao certame ou ao candidato.
func traduzir(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && (pg.Code == "23503" || pg.Code == "23514") {
		return fmt.Errorf("%w: %s", ErrDadoInvalido, pg.ConstraintName)
	}
	return err
}
