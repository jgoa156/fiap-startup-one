package notificacao

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// eventoSES é o que o SES publica no SNS por um conjunto de configuração ("eventType") ou pelas
// notificações da identidade ("notificationType"): o mesmo formato, com o nome do campo trocado.
type eventoSES struct {
	EventType        string `json:"eventType"`
	NotificationType string `json:"notificationType"`
	Mail             struct {
		MessageID string `json:"messageId"`
	} `json:"mail"`
	Bounce *struct {
		BounceType        string `json:"bounceType"` // Permanent, Transient, Undetermined
		BounceSubType     string `json:"bounceSubType"`
		Timestamp         string `json:"timestamp"`
		BouncedRecipients []struct {
			EmailAddress   string `json:"emailAddress"`
			DiagnosticCode string `json:"diagnosticCode"`
		} `json:"bouncedRecipients"`
	} `json:"bounce"`
	Complaint *struct {
		Timestamp             string `json:"timestamp"`
		ComplaintFeedbackType string `json:"complaintFeedbackType"`
		ComplainedRecipients  []struct {
			EmailAddress string `json:"emailAddress"`
		} `json:"complainedRecipients"`
	} `json:"complaint"`
	Delivery *struct {
		Timestamp  string   `json:"timestamp"`
		Recipients []string `json:"recipients"`
	} `json:"delivery"`
}

// Ocorrencia é uma linha de notificacao_ocorrencia a gravar.
type Ocorrencia struct {
	Tipo, Destinatario, Detalhe string
	Em                          time.Time
}

func quando(s string) time.Time {
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t
	}
	return time.Now()
}

// Ocorrencias traduz o evento do SES. Envio, abertura, clique e outros tipos não interessam à prova
// e saem vazios; o id do envio sai junto.
func Ocorrencias(mensagem string) (provedorID string, ocs []Ocorrencia, err error) {
	var e eventoSES
	if err := json.Unmarshal([]byte(mensagem), &e); err != nil {
		return "", nil, err
	}
	tipo := e.EventType
	if tipo == "" {
		tipo = e.NotificationType
	}
	switch {
	case tipo == "Bounce" && e.Bounce != nil:
		t := "DEVOLVIDA_TEMPORARIA" // Transient e Undetermined: o SES já tentou de novo; não suprime
		if e.Bounce.BounceType == "Permanent" {
			t = "DEVOLVIDA"
		}
		for _, r := range e.Bounce.BouncedRecipients {
			d := e.Bounce.BounceType + "/" + e.Bounce.BounceSubType
			if r.DiagnosticCode != "" {
				d += ": " + r.DiagnosticCode
			}
			ocs = append(ocs, Ocorrencia{t, r.EmailAddress, d, quando(e.Bounce.Timestamp)})
		}
	case tipo == "Complaint" && e.Complaint != nil:
		for _, r := range e.Complaint.ComplainedRecipients {
			ocs = append(ocs, Ocorrencia{"RECLAMACAO", r.EmailAddress, e.Complaint.ComplaintFeedbackType, quando(e.Complaint.Timestamp)})
		}
	case tipo == "Delivery" && e.Delivery != nil:
		for _, r := range e.Delivery.Recipients {
			ocs = append(ocs, Ocorrencia{"ENTREGUE", r, "", quando(e.Delivery.Timestamp)})
		}
	}
	return e.Mail.MessageID, ocs, nil
}

// Registrar grava as ocorrências de uma mensagem do SNS e suprime o endereço que devolveu de vez ou
// reclamou. A mesma mensagem do SNS entregue de novo (o SNS repete até ouvir 200) não duplica nada.
func Registrar(ctx context.Context, pool *pgxpool.Pool, snsID, provedorID string, ocs []Ocorrencia) error {
	if len(ocs) == 0 {
		return nil
	}
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		for _, o := range ocs {
			var id int64
			err := tx.QueryRow(ctx, `
				INSERT INTO notificacao_ocorrencia (sns_id, provedor_id, tipo, destinatario, detalhe, ocorrida_em)
				VALUES ($1, $2, $3, $4, nullif($5, ''), $6) ON CONFLICT (sns_id, destinatario) DO NOTHING
				RETURNING id`, snsID, provedorID, o.Tipo, o.Destinatario, o.Detalhe, o.Em).Scan(&id)
			if errors.Is(err, pgx.ErrNoRows) {
				continue // já registrada
			}
			if err != nil {
				return err
			}
			if o.Tipo == "DEVOLVIDA" || o.Tipo == "RECLAMACAO" {
				if _, err := tx.Exec(ctx, `
					INSERT INTO email_suprimido (email, motivo, ocorrencia_id) VALUES (lower($1), $2, $3)
					ON CONFLICT (email) DO NOTHING`, o.Destinatario, o.Tipo, id); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// ReceberEventosSES é o endereço que o SNS chama: confirma a inscrição no tópico e grava os eventos
// do SES. Responde 403 ao que não veio do SNS do nosso tópico (o SNS de verdade nunca recebe isso),
// 500 ao que falhou aqui (o SNS repete) e 200 ao resto.
func ReceberEventosSES(pool *pgxpool.Pool, v *VerificadorSNS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "só POST", http.StatusMethodNotAllowed)
			return
		}
		corpo, err := io.ReadAll(io.LimitReader(r.Body, 256<<10))
		if err != nil {
			http.Error(w, "corpo", http.StatusBadRequest)
			return
		}
		var m MensagemSNS
		if err := json.Unmarshal(corpo, &m); err != nil {
			http.Error(w, "JSON", http.StatusBadRequest)
			return
		}
		ctx := r.Context()
		if err := v.Verificar(ctx, m); err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, ErrSNSInvalida) {
				status = http.StatusForbidden
			}
			log.Printf("eventos do SES: %v", err)
			http.Error(w, "recusada", status)
			return
		}
		switch m.Type {
		case "SubscriptionConfirmation":
			if err := v.Confirmar(ctx, m); err != nil {
				log.Printf("eventos do SES: confirmar a inscrição: %v", err)
				http.Error(w, "confirmação", http.StatusInternalServerError)
				return
			}
			log.Printf("eventos do SES: inscrição no tópico %s confirmada", m.TopicArn)
		case "Notification":
			provedorID, ocs, err := Ocorrencias(m.Message)
			if err != nil || (len(ocs) > 0 && provedorID == "") {
				// Assinada pelo SNS, mas não é um evento do SES: ninguém publica outra coisa no tópico.
				log.Printf("eventos do SES: mensagem %s fora do formato: %v", m.MessageId, err)
				break
			}
			if err := Registrar(ctx, pool, m.MessageId, provedorID, ocs); err != nil {
				log.Printf("eventos do SES: %v", err)
				http.Error(w, "gravar", http.StatusInternalServerError)
				return
			}
			for _, o := range ocs {
				if o.Tipo != "ENTREGUE" {
					log.Printf("eventos do SES: %s para %s (%s)", strings.ToLower(o.Tipo), o.Destinatario, o.Detalhe)
				}
			}
		}
		w.WriteHeader(http.StatusOK)
	})
}
