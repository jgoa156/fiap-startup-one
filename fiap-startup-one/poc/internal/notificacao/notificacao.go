// Package notificacao avisa o candidato, por e-mail, de cada mudança de estado que importa para ele
// (02-produto §7 B) e guarda a prova do aviso.
//
// O aviso nasce na mesma transação do evento da trilha, como tarefa da fila: não há evento sem aviso
// nem aviso sem evento. A tarefa vai com prioridade de lote: o e-mail nunca passa na frente do Pix nem
// do cartão; no pico ele espera, e sai quando o pico escoa. Um aviso por evento, nunca dois: a chave
// do envio é o id do evento, e o provedor que recebe a mesma chave devolve o mesmo envio. O SES não
// tem essa chave, e com ele a garantia é "pelo menos um" (ver ses.go).
package notificacao

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"lisura/internal/fila"
)

// Tarefa é o tipo da tarefa de aviso.
const Tarefa = "notificar"

// Mensagem é um e-mail. Chave é a de idempotência: a mesma chave, o mesmo envio.
type Mensagem struct {
	Chave, Para, Assunto, Texto string
}

// Remetente entrega a mensagem e devolve o id do envio no provedor.
type Remetente interface {
	Enviar(ctx context.Context, m Mensagem) (string, error)
}

// ErrEnderecoRecusado: o provedor recusou o endereço de vez (repetir não adianta).
var ErrEnderecoRecusado = errors.New("o provedor recusou o endereço")

type modelo struct{ assunto, texto *template.Template }

func novo(assunto, texto string) modelo {
	return modelo{template.Must(template.New("a").Parse(assunto)), template.Must(template.New("t").Parse(texto))}
}

// Os eventos que avisam, e o que dizem. Criar o rascunho e iniciar a análise não avisam: não pedem
// nada do candidato nem mudam o que ele pode fazer.
var modelos = map[string]modelo{
	"FINALIZAR_COM_PAGAMENTO": novo("Inscrição nº {{.Numero}} enviada: falta pagar até {{.Prazo}}",
		"Olá, {{.Nome}}.\n\nSua inscrição nº {{.Numero}} em {{.Certame}} foi enviada. Falta o pagamento da taxa, até {{.Prazo}}: gere o Pix ou o boleto na sua área do candidato.\n"),
	"FINALIZAR_ISENTO": novo("Inscrição nº {{.Numero}} enviada, com isenção da taxa",
		"Olá, {{.Nome}}.\n\nSua inscrição nº {{.Numero}} em {{.Certame}} foi enviada com isenção da taxa. O cartão de confirmação está na sua área do candidato.\n"),
	"CONFIRMAR_PAGAMENTO": novo("Pagamento confirmado: inscrição nº {{.Numero}}",
		"Olá, {{.Nome}}.\n\nO pagamento da sua inscrição nº {{.Numero}} em {{.Certame}} foi confirmado. O cartão de confirmação está na sua área do candidato.\n"),
	"EXPIRAR": novo("Inscrição nº {{.Numero}} expirada",
		"Olá, {{.Nome}}.\n\nO prazo de pagamento da sua inscrição nº {{.Numero}} em {{.Certame}} terminou sem pagamento confirmado, e ela expirou. Se você pagou depois do prazo, o valor volta automaticamente.\n"),
	"INDEFERIR": novo("Inscrição nº {{.Numero}} indeferida",
		"Olá, {{.Nome}}.\n\nSua inscrição nº {{.Numero}} em {{.Certame}} foi indeferida. Motivo: {{.Motivo}}. Você pode recorrer pela sua área do candidato, dentro do prazo do edital.\n"),
	"PROVER_RECURSO": novo("Recurso provido: inscrição nº {{.Numero}} deferida",
		"Olá, {{.Nome}}.\n\nSeu recurso foi provido, e a inscrição nº {{.Numero}} em {{.Certame}} foi deferida.\n"),
	"HOMOLOGAR": novo("Inscrição nº {{.Numero}} homologada",
		"Olá, {{.Nome}}.\n\nSua inscrição nº {{.Numero}} em {{.Certame}} foi homologada. O local e a sala de prova saem no ensalamento e aparecem no seu cartão de confirmação.\n"),
	"CANCELAR_CERTAME": novo("{{.Certame}} foi cancelado",
		"Olá, {{.Nome}}.\n\n{{.Certame}} foi cancelado. Motivo: {{.Motivo}}. Se você pagou a taxa da inscrição nº {{.Numero}}, o valor volta automaticamente: o Pix, para a conta de onde saiu; o boleto, por Pix para a chave do seu CPF.\n"),
}

// Avisa diz se o evento gera aviso.
func Avisa(evento string) bool {
	_, ok := modelos[evento]
	return ok
}

type carga struct {
	EventoID int64 `json:"evento_id"`
}

// Enfileirar pede o aviso do evento, se ele avisa. Chamado na transação que gravou o evento.
func Enfileirar(ctx context.Context, tx pgx.Tx, eventoID int64, evento string) error {
	if !Avisa(evento) {
		return nil
	}
	return fila.EnfileirarCom(ctx, tx, fila.Lote, Tarefa, strconv.FormatInt(eventoID, 10), carga{eventoID})
}

type dados struct {
	Nome, Certame, Motivo, Prazo string
	Numero                       int64
}

// Notificar é o manipulador da tarefa: monta o aviso, entrega e grava a prova. Idempotente: o
// evento já avisado não é avisado de novo; a entrega repetida (resposta perdida, trabalhador que
// caiu) leva a mesma chave, e o provedor devolve o mesmo envio.
func Notificar(rem Remetente, pool *pgxpool.Pool) fila.Manipulador {
	return func(ctx context.Context, b json.RawMessage) error {
		var c carga
		if err := json.Unmarshal(b, &c); err != nil {
			return err
		}
		var ja bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM notificacao WHERE evento_id = $1)`, c.EventoID).Scan(&ja); err != nil || ja {
			return err
		}
		var evento, email string
		var motivo, cancelamento *string
		var prazo *time.Time
		var d dados
		err := pool.QueryRow(ctx, `
			SELECT e.evento, e.motivo_codigo, i.id, c.nome, c.email, k.nome, k.cancelamento_motivo,
			       (SELECT fecha_em FROM certame_fase WHERE certame_id = k.id AND fase = 'pagamento')
			FROM inscricao_evento e JOIN inscricao i ON i.id = e.inscricao_id
			JOIN candidato c ON c.id = i.candidato_id JOIN certame k ON k.id = i.certame_id
			WHERE e.id = $1`, c.EventoID).Scan(&evento, &motivo, &d.Numero, &d.Nome, &email, &d.Certame, &cancelamento, &prazo)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		m, ok := modelos[evento]
		if !ok {
			return nil
		}
		switch {
		case motivo != nil:
			d.Motivo = *motivo
		case cancelamento != nil:
			d.Motivo = *cancelamento
		}
		if prazo != nil {
			d.Prazo = prazo.In(brasilia).Format("02/01/2006 às 15:04")
		}
		var assunto, texto bytes.Buffer
		if err := m.assunto.Execute(&assunto, d); err != nil {
			return err
		}
		if err := m.texto.Execute(&texto, d); err != nil {
			return err
		}
		msg := Mensagem{Chave: "evt-" + strconv.FormatInt(c.EventoID, 10), Para: email, Assunto: assunto.String(), Texto: texto.String()}
		// O endereço que devolveu de vez ou reclamou não recebe mais (ver a migração 00020): o aviso fica
		// registrado como não enviado, com o porquê, e o candidato o vê na área dele.
		var suprimido string
		err = pool.QueryRow(ctx, `SELECT motivo FROM email_suprimido WHERE email = lower($1)`, email).Scan(&suprimido)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		var id string
		if suprimido != "" {
			err = fmt.Errorf("%w: %s está suprimido (%s)", ErrEnderecoRecusado, email, strings.ToLower(suprimido))
		} else {
			id, err = rem.Enviar(ctx, msg)
		}
		var pid, falha *string
		switch {
		case errors.Is(err, ErrEnderecoRecusado):
			f := err.Error()
			falha = &f
			log.Printf("aviso do evento %d: %v", c.EventoID, err)
		case err != nil:
			return err // passageiro: a fila repete com a mesma chave
		default:
			pid = &id
		}
		_, err = pool.Exec(ctx, `
			INSERT INTO notificacao (evento_id, inscricao_id, destinatario, assunto, resumo, provedor_id, falha)
			VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT (evento_id) DO NOTHING`,
			c.EventoID, d.Numero, email, msg.Assunto, Resumo(msg.Assunto, msg.Texto), pid, falha)
		return err
	}
}

var brasilia = time.FixedZone("BRT", -3*3600) // sem horário de verão desde 2019

// Aviso é o que o candidato vê de um aviso enviado.
type Aviso struct {
	Evento    string    `json:"evento"`
	Assunto   string    `json:"assunto"`
	EnviadaEm time.Time `json:"enviada_em"`
	Entregue  bool      `json:"entregue"` // o provedor aceitou
	Resumo    string    `json:"resumo"`
	// Devolvido: depois de aceito, o servidor do destinatário recusou de vez, ou o destinatário
	// reclamou. Falha: por que não saiu, ou por que voltou.
	Devolvido bool   `json:"devolvido"`
	Falha     string `json:"falha,omitempty"`
	Para      string `json:"para"` // o endereço da conta quando o aviso saiu
}

// DaInscricao lista os avisos da inscrição do candidato, do mais antigo ao mais novo.
func DaInscricao(ctx context.Context, pool *pgxpool.Pool, inscricaoID, candidatoID int64) ([]Aviso, error) {
	rows, err := pool.Query(ctx, `
		SELECT e.evento, n.assunto, n.enviada_em, n.provedor_id IS NOT NULL, n.resumo, o.id IS NOT NULL,
		       coalesce(n.falha, o.detalhe, ''), n.destinatario
		FROM notificacao n JOIN inscricao_evento e ON e.id = n.evento_id JOIN inscricao i ON i.id = n.inscricao_id
		LEFT JOIN LATERAL (SELECT id, detalhe FROM notificacao_ocorrencia
		                   WHERE provedor_id = n.provedor_id AND tipo IN ('DEVOLVIDA', 'RECLAMACAO')
		                   ORDER BY id LIMIT 1) o ON true
		WHERE n.inscricao_id = $1 AND i.candidato_id = $2 ORDER BY n.evento_id`, inscricaoID, candidatoID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[Aviso])
}

// Resumo é o SHA-256 que a prova guarda, para conferir uma cópia do aviso.
func Resumo(assunto, texto string) string {
	h := sha256.Sum256([]byte(assunto + "\n" + texto))
	return hex.EncodeToString(h[:])
}
