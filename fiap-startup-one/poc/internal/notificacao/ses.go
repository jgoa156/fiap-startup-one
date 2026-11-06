package notificacao

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

// SES entrega pelo Amazon SES (API v2), o provedor de produção.
//
// O que ele não garante, dito sem rodeio: o SES não tem chave de idempotência. Se a mensagem sai e a
// resposta se perde, a fila repete e a candidata recebe o aviso duas vezes. Com o SES, "um aviso por
// evento" vira "pelo menos um aviso por evento": para um aviso que abre prazo de recurso, chegar duas
// vezes é melhor do que não chegar. A prova continua uma só (a tabela notificacao é única por evento),
// e cada envio leva a chave no cabeçalho X-Lisura-Chave e na etiqueta "chave", para o atendimento
// reconhecer a cópia.
//
// A taxa: o SES limita envios por segundo e por dia, por conta e região, e estourar o limite dá erro
// de vazão. O remetente espaça os envios na taxa configurada, em vez de bater no limite e depender
// das repetições da fila.
type SES struct {
	Cliente  *sesv2.Client
	De       string // remetente verificado no SES
	Conjunto string // configuration set (eventos de devolução e reclamação); vazio, nenhum

	intervalo time.Duration
	mu        sync.Mutex
	proxima   time.Time
}

// NovoSES cria o remetente. porSegundo é a taxa máxima de envio (0: sem espaçamento).
func NovoSES(cliente *sesv2.Client, de, conjunto string, porSegundo float64) *SES {
	s := &SES{Cliente: cliente, De: de, Conjunto: conjunto}
	if porSegundo > 0 {
		s.intervalo = time.Duration(float64(time.Second) / porSegundo)
	}
	return s
}

// esperarVez reserva o próximo horário livre na taxa e espera por ele.
func (s *SES) esperarVez(ctx context.Context) error {
	if s.intervalo == 0 {
		return nil
	}
	s.mu.Lock()
	agora := time.Now()
	if s.proxima.Before(agora) {
		s.proxima = agora
	}
	vez := s.proxima
	s.proxima = s.proxima.Add(s.intervalo)
	s.mu.Unlock()
	t := time.NewTimer(time.Until(vez))
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *SES) Enviar(ctx context.Context, msg Mensagem) (string, error) {
	if err := s.esperarVez(ctx); err != nil {
		return "", err
	}
	in := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(s.De),
		Destination:      &types.Destination{ToAddresses: []string{msg.Para}},
		Content: &types.EmailContent{Simple: &types.Message{
			Subject: &types.Content{Data: aws.String(msg.Assunto), Charset: aws.String("UTF-8")},
			Body:    &types.Body{Text: &types.Content{Data: aws.String(msg.Texto), Charset: aws.String("UTF-8")}},
			Headers: []types.MessageHeader{{Name: aws.String("X-Lisura-Chave"), Value: aws.String(msg.Chave)}},
		}},
		EmailTags: []types.MessageTag{{Name: aws.String("chave"), Value: aws.String(msg.Chave)}},
	}
	if s.Conjunto != "" {
		in.ConfigurationSetName = aws.String(s.Conjunto)
	}
	out, err := s.Cliente.SendEmail(ctx, in)
	// MessageRejected é a recusa da mensagem em si (no sandbox do SES, o destinatário não
	// verificado): repetir não adianta. Vazão, conta pausada e falha do serviço são passageiros, e a
	// fila repete. O endereço que só devolve depois (bounce) não aparece aqui: chega pelo conjunto de
	// configuração, fora desta chamada.
	var recusa *types.MessageRejected
	if errors.As(err, &recusa) {
		return "", fmt.Errorf("%w: %s: %s", ErrEnderecoRecusado, msg.Para, recusa.ErrorMessage())
	}
	if err != nil {
		return "", err
	}
	return aws.ToString(out.MessageId), nil
}
