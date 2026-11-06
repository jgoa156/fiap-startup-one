package notificacao_test

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"lisura/internal/inscricao"
	"lisura/internal/notificacao"
)

const topicoTeste = "arn:aws:sns:sa-east-1:123456789012:lisura-avisos"

// snsLocal faz o papel do SNS: uma chave RSA e um certificado gerados na hora, servidos no endereço
// de onde o SNS de verdade serve o dele. Nada sai da máquina.
type snsLocal struct {
	chave *rsa.PrivateKey
	pem   []byte
	url   string

	mu       sync.Mutex
	baixados []string
}

func novoSNS(t *testing.T) *snsLocal {
	t.Helper()
	chave, err := rsa.GenerateKey(rand.Reader, 2048)
	must(t, err)
	modelo := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "sns.amazonaws.com"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, modelo, modelo, &chave.PublicKey, chave)
	must(t, err)
	return &snsLocal{chave: chave, pem: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		url: "https://sns.sa-east-1.amazonaws.com/SimpleNotificationService-teste.pem"}
}

func (s *snsLocal) verificador() *notificacao.VerificadorSNS {
	return &notificacao.VerificadorSNS{Topico: topicoTeste, Baixar: func(_ context.Context, u string) ([]byte, error) {
		s.mu.Lock()
		s.baixados = append(s.baixados, u)
		s.mu.Unlock()
		switch {
		case u == s.url:
			return s.pem, nil
		case strings.Contains(u, "Action=ConfirmSubscription"):
			return []byte("<ConfirmSubscriptionResponse/>"), nil
		}
		return nil, fmt.Errorf("404 %s", u)
	}}
}

func (s *snsLocal) baixou(trecho string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, u := range s.baixados {
		if strings.Contains(u, trecho) {
			return true
		}
	}
	return false
}

// assinar assina como o SNS, com esta chave ou com outra (a de quem forja).
func assinar(t *testing.T, m notificacao.MensagemSNS, versao string, chave *rsa.PrivateKey) notificacao.MensagemSNS {
	t.Helper()
	m.SignatureVersion = versao
	texto := []byte(notificacao.Canonica(m))
	var h crypto.Hash
	var resumo []byte
	if versao == "1" {
		s := sha1.Sum(texto)
		h, resumo = crypto.SHA1, s[:]
	} else {
		s := sha256.Sum256(texto)
		h, resumo = crypto.SHA256, s[:]
	}
	sig, err := rsa.SignPKCS1v15(rand.Reader, chave, h, resumo)
	must(t, err)
	m.Signature = base64.StdEncoding.EncodeToString(sig)
	return m
}

var snsSeq int

func (s *snsLocal) notificacao(t *testing.T, eventoSES string) notificacao.MensagemSNS {
	t.Helper()
	snsSeq++
	return assinar(t, notificacao.MensagemSNS{Type: "Notification", MessageId: fmt.Sprintf("sns-%d-%d", time.Now().UnixNano(), snsSeq),
		TopicArn: topicoTeste, Message: eventoSES, Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		SigningCertURL: s.url}, "2", s.chave)
}

func devolucao(provedorID, para, tipo string) string {
	b, _ := json.Marshal(map[string]any{
		"eventType": "Bounce",
		"mail":      map[string]any{"messageId": provedorID, "destination": []string{para}},
		"bounce": map[string]any{"bounceType": tipo, "bounceSubType": "General", "timestamp": time.Now().UTC().Format(time.RFC3339Nano),
			"bouncedRecipients": []map[string]string{{"emailAddress": para, "diagnosticCode": "smtp; 550 5.1.1 user unknown"}}},
	})
	return string(b)
}

func postar(t *testing.T, h http.Handler, m notificacao.MensagemSNS) int {
	t.Helper()
	b, _ := json.Marshal(m)
	r := httptest.NewRequest(http.MethodPost, "/api/ses/eventos", bytes.NewReader(b))
	r.Header.Set("x-amz-sns-message-type", m.Type)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w.Code
}

// Só passa a mensagem do nosso tópico, assinada pela chave do certificado que veio do SNS. O
// certificado de outro lugar nem é baixado.
func TestSNSSoAceitaONossoTopicoAssinado(t *testing.T) {
	s := novoSNS(t)
	v := s.verificador()
	ctx := context.Background()
	m := s.notificacao(t, `{"eventType":"Delivery"}`)
	must(t, v.Verificar(ctx, m))
	must(t, v.Verificar(ctx, assinar(t, m, "1", s.chave)))

	outra, err := rsa.GenerateKey(rand.Reader, 2048)
	must(t, err)
	alterada := m
	alterada.Message = `{"eventType":"Bounce"}`
	deOutroTopico := m
	deOutroTopico.TopicArn = "arn:aws:sns:sa-east-1:999999999999:outro"
	deOutroTopico = assinar(t, deOutroTopico, "2", s.chave)
	certForaDoSNS := m
	certForaDoSNS.SigningCertURL = "https://sns.sa-east-1.amazonaws.com.forjado.test/cert.pem"
	certSemHTTPS := m
	certSemHTTPS.SigningCertURL = "http://sns.sa-east-1.amazonaws.com/SimpleNotificationService-teste.pem"
	for nome, x := range map[string]notificacao.MensagemSNS{
		"texto alterado depois de assinar": alterada,
		"outro tópico":                     deOutroTopico,
		"assinada por outra chave":         assinar(t, m, "2", outra),
		"certificado fora do SNS":          certForaDoSNS,
		"certificado sem HTTPS":            certSemHTTPS,
	} {
		if err := v.Verificar(ctx, x); !errors.Is(err, notificacao.ErrSNSInvalida) {
			t.Errorf("%s: %v", nome, err)
		}
	}
	if s.baixou("forjado") || s.baixou("http://") {
		t.Fatalf("baixou certificado de fora do SNS: %v", s.baixados)
	}
}

// A devolução definitiva fica registrada uma vez (mesmo entregue duas), o endereço é suprimido sem
// importar maiúsculas, o candidato vê que o aviso voltou, e o aviso seguinte não sai.
func TestDevolucaoSuprimeEOCandidatoVe(t *testing.T) {
	c := novo(t)
	c.esvaziar(t)
	ctx := context.Background()
	var provedorID string
	must(t, pool.QueryRow(ctx, `SELECT provedor_id FROM notificacao WHERE inscricao_id = $1`, c.inscricao).Scan(&provedorID))
	s := novoSNS(t)
	h := notificacao.ReceberEventosSES(pool, s.verificador())
	m := s.notificacao(t, devolucao(provedorID, strings.ToUpper(c.email), "Permanent"))
	for range 2 {
		if st := postar(t, h, m); st != http.StatusOK {
			t.Fatalf("status %d", st)
		}
	}
	if n := contarAqui(t, `SELECT count(*) FROM notificacao_ocorrencia WHERE provedor_id = $1`, provedorID); n != 1 {
		t.Fatalf("%d ocorrências da mesma mensagem; esperado 1", n)
	}
	if n := contarAqui(t, `SELECT count(*) FROM email_suprimido WHERE email = $1 AND motivo = 'DEVOLVIDA'`, c.email); n != 1 {
		t.Fatal("o endereço que devolveu de vez não foi suprimido")
	}
	as := c.avisos(t)
	if len(as) != 1 || !as[0].Devolvido || !strings.Contains(as[0].Falha, "550 5.1.1") {
		t.Fatalf("o candidato não vê que o aviso voltou: %+v", as)
	}

	c.evento(t, inscricao.ConfirmarPagamento, inscricao.Sistema)
	c.esvaziar(t)
	as = c.avisos(t)
	if n := len(c.correio.Enviadas()); n != 1 || len(as) != 2 || as[1].Entregue || !strings.Contains(as[1].Falha, "suprimido") {
		t.Fatalf("aviso para endereço suprimido: %d mensagens no provedor, avisos %+v", n, as)
	}
}

// A devolução temporária (caixa cheia, servidor fora) fica registrada e não suprime: o próximo aviso sai.
func TestDevolucaoTemporariaNaoSuprime(t *testing.T) {
	c := novo(t)
	c.esvaziar(t)
	ctx := context.Background()
	var provedorID string
	must(t, pool.QueryRow(ctx, `SELECT provedor_id FROM notificacao WHERE inscricao_id = $1`, c.inscricao).Scan(&provedorID))
	s := novoSNS(t)
	if st := postar(t, notificacao.ReceberEventosSES(pool, s.verificador()), s.notificacao(t, devolucao(provedorID, c.email, "Transient"))); st != http.StatusOK {
		t.Fatalf("status %d", st)
	}
	if n := contarAqui(t, `SELECT count(*) FROM notificacao_ocorrencia WHERE provedor_id = $1 AND tipo = 'DEVOLVIDA_TEMPORARIA'`, provedorID); n != 1 {
		t.Fatalf("%d devoluções temporárias registradas", n)
	}
	c.evento(t, inscricao.ConfirmarPagamento, inscricao.Sistema)
	c.esvaziar(t)
	if n := len(c.correio.Enviadas()); n != 2 || c.avisos(t)[0].Devolvido {
		t.Fatalf("a devolução temporária calou o candidato: %d mensagens, %+v", n, c.avisos(t))
	}
}

// Quem forja uma devolução para calar os avisos de alguém recebe 403, e nada muda.
func TestDevolucaoForjadaNaoCalaNinguem(t *testing.T) {
	c := novo(t)
	c.esvaziar(t)
	ctx := context.Background()
	var provedorID string
	must(t, pool.QueryRow(ctx, `SELECT provedor_id FROM notificacao WHERE inscricao_id = $1`, c.inscricao).Scan(&provedorID))
	s := novoSNS(t)
	forjador, err := rsa.GenerateKey(rand.Reader, 2048)
	must(t, err)
	m := assinar(t, s.notificacao(t, devolucao(provedorID, c.email, "Permanent")), "2", forjador)
	if st := postar(t, notificacao.ReceberEventosSES(pool, s.verificador()), m); st != http.StatusForbidden {
		t.Fatalf("devolução forjada: status %d", st)
	}
	if n := contarAqui(t, `SELECT count(*) FROM email_suprimido WHERE email = $1`, c.email); n != 0 {
		t.Fatal("a devolução forjada suprimiu o endereço")
	}
	if n := contarAqui(t, `SELECT count(*) FROM notificacao_ocorrencia WHERE provedor_id = $1`, provedorID); n != 0 {
		t.Fatal("a devolução forjada foi registrada")
	}
}

// A inscrição do nosso endereço no tópico é confirmada pelo SubscribeURL do SNS; um SubscribeURL de
// outro lugar não é visitado.
func TestInscricaoNoTopicoConfirmada(t *testing.T) {
	s := novoSNS(t)
	h := notificacao.ReceberEventosSES(pool, s.verificador())
	confirmar := func(subscribe string) int {
		return postar(t, h, assinar(t, notificacao.MensagemSNS{Type: "SubscriptionConfirmation", MessageId: "c-" + subscribe,
			Token: "tok", TopicArn: topicoTeste, Message: "Você foi inscrito.", SubscribeURL: subscribe,
			Timestamp: time.Now().UTC().Format(time.RFC3339Nano), SigningCertURL: s.url}, "1", s.chave))
	}
	if st := confirmar("https://sns.sa-east-1.amazonaws.com/?Action=ConfirmSubscription&Token=tok"); st != http.StatusOK || !s.baixou("Action=ConfirmSubscription") {
		t.Fatalf("confirmação: status %d, baixados %v", st, s.baixados)
	}
	if st := confirmar("https://forjado.test/?Action=ConfirmSubscription&Token=tok"); st == http.StatusOK || s.baixou("forjado") {
		t.Fatalf("SubscribeURL de fora do SNS: status %d, baixados %v", st, s.baixados)
	}
}

func contarAqui(t *testing.T, sql string, args ...any) int {
	t.Helper()
	var n int
	must(t, pool.QueryRow(context.Background(), sql, args...).Scan(&n))
	return n
}
