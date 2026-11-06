package notificacao

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

// MensagemSNS é o corpo que o SNS manda por HTTPS (o que importa dele).
type MensagemSNS struct {
	Type             string
	MessageId        string
	Token            string
	TopicArn         string
	Subject          string
	Message          string
	Timestamp        string
	SignatureVersion string
	Signature        string
	SigningCertURL   string
	SubscribeURL     string
}

// ErrSNSInvalida: a mensagem não veio do SNS do nosso tópico (assinatura, tópico ou certificado).
var ErrSNSInvalida = errors.New("mensagem do SNS inválida")

// hostSNS é o único lugar de onde o certificado e a confirmação podem vir. Sem isto, quem forja a
// mensagem aponta o próprio certificado e assina com a própria chave.
var hostSNS = regexp.MustCompile(`^sns\.[a-z0-9-]+\.amazonaws\.com$`)

// VerificadorSNS confere que a mensagem veio do SNS, do tópico configurado, como a AWS documenta:
// certificado baixado só de sns.<região>.amazonaws.com por HTTPS, e a assinatura RSA da mensagem
// canônica (SHA1 na versão 1, SHA256 na 2).
type VerificadorSNS struct {
	Topico string // o ARN aceito; qualquer outro é recusado
	// Baixar faz o GET (o certificado e a confirmação da inscrição). Nil: HTTP com 10 s de prazo.
	// Os testes trocam por um que não sai da máquina.
	Baixar func(ctx context.Context, url string) ([]byte, error)

	mu    sync.Mutex
	certs map[string]*x509.Certificate
}

func (v *VerificadorSNS) baixar(ctx context.Context, u string) ([]byte, error) {
	if v.Baixar != nil {
		return v.Baixar(ctx, u)
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", u, resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 64<<10))
}

// urlDaAWS aceita só https://sns.<região>.amazonaws.com/...
func urlDaAWS(u string) error {
	p, err := url.Parse(u)
	if err != nil || p.Scheme != "https" || !hostSNS.MatchString(p.Hostname()) || p.Port() != "" || p.User != nil {
		return fmt.Errorf("%w: endereço fora do SNS: %q", ErrSNSInvalida, u)
	}
	return nil
}

func (v *VerificadorSNS) certificado(ctx context.Context, u string) (*x509.Certificate, error) {
	if err := urlDaAWS(u); err != nil {
		return nil, err
	}
	if !strings.HasSuffix(strings.ToLower(u), ".pem") {
		return nil, fmt.Errorf("%w: certificado que não é .pem: %q", ErrSNSInvalida, u)
	}
	v.mu.Lock()
	c, ok := v.certs[u]
	v.mu.Unlock()
	if ok && time.Now().Before(c.NotAfter) {
		return c, nil
	}
	b, err := v.baixar(ctx, u)
	if err != nil {
		return nil, err // passageiro: o SNS manda de novo
	}
	bloco, _ := pem.Decode(b)
	if bloco == nil {
		return nil, fmt.Errorf("%w: certificado sem PEM", ErrSNSInvalida)
	}
	c, err = x509.ParseCertificate(bloco.Bytes)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSNSInvalida, err)
	}
	if agora := time.Now(); agora.Before(c.NotBefore) || agora.After(c.NotAfter) {
		return nil, fmt.Errorf("%w: certificado fora da validade", ErrSNSInvalida)
	}
	v.mu.Lock()
	if v.certs == nil {
		v.certs = map[string]*x509.Certificate{}
	}
	v.certs[u] = c
	v.mu.Unlock()
	return c, nil
}

// Canonica é o texto que o SNS assina: os campos em ordem alfabética, "nome\nvalor\n", só os que o
// tipo da mensagem leva (Subject só se veio).
func Canonica(m MensagemSNS) string {
	var b strings.Builder
	campo := func(nome, valor string) { b.WriteString(nome + "\n" + valor + "\n") }
	campo("Message", m.Message)
	campo("MessageId", m.MessageId)
	if m.Type == "Notification" {
		if m.Subject != "" {
			campo("Subject", m.Subject)
		}
	} else {
		campo("SubscribeURL", m.SubscribeURL)
	}
	campo("Timestamp", m.Timestamp)
	if m.Type != "Notification" {
		campo("Token", m.Token)
	}
	campo("TopicArn", m.TopicArn)
	campo("Type", m.Type)
	return b.String()
}

// Verificar devolve nil só para mensagem do nosso tópico com assinatura válida. Erro com
// ErrSNSInvalida é forjada (não adianta repetir); outro erro é passageiro (o certificado não baixou).
func (v *VerificadorSNS) Verificar(ctx context.Context, m MensagemSNS) error {
	if v.Topico == "" || m.TopicArn != v.Topico {
		return fmt.Errorf("%w: tópico %q", ErrSNSInvalida, m.TopicArn)
	}
	switch m.Type {
	case "Notification", "SubscriptionConfirmation", "UnsubscribeConfirmation":
	default:
		return fmt.Errorf("%w: tipo %q", ErrSNSInvalida, m.Type)
	}
	var h crypto.Hash
	switch m.SignatureVersion {
	case "1":
		h = crypto.SHA1
	case "2":
		h = crypto.SHA256
	default:
		return fmt.Errorf("%w: versão de assinatura %q", ErrSNSInvalida, m.SignatureVersion)
	}
	assinatura, err := base64.StdEncoding.DecodeString(m.Signature)
	if err != nil {
		return fmt.Errorf("%w: assinatura: %v", ErrSNSInvalida, err)
	}
	c, err := v.certificado(ctx, m.SigningCertURL)
	if err != nil {
		return err
	}
	chave, ok := c.PublicKey.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("%w: chave que não é RSA", ErrSNSInvalida)
	}
	texto := []byte(Canonica(m))
	var resumo []byte
	if h == crypto.SHA1 {
		s := sha1.Sum(texto)
		resumo = s[:]
	} else {
		s := sha256.Sum256(texto)
		resumo = s[:]
	}
	if err := rsa.VerifyPKCS1v15(chave, h, resumo, assinatura); err != nil {
		return fmt.Errorf("%w: assinatura não confere", ErrSNSInvalida)
	}
	return nil
}

// Confirmar aceita a inscrição do nosso endereço no tópico, visitando o SubscribeURL (que também só
// pode ser do SNS). Chamado depois de Verificar.
func (v *VerificadorSNS) Confirmar(ctx context.Context, m MensagemSNS) error {
	if err := urlDaAWS(m.SubscribeURL); err != nil {
		return err
	}
	_, err := v.baixar(ctx, m.SubscribeURL)
	return err
}
