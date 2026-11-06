package notificacao_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"

	"lisura/internal/fila"
	"lisura/internal/notificacao"
	"lisura/internal/publicacao"
)

// sesFalso responde como o POST /v2/email/outbound-emails do SES. responder decide, a cada envio
// recebido (já contado), o status e o tipo de erro; status 200 devolve um MessageId.
type sesFalso struct {
	mu        sync.Mutex
	recebidos []map[string]any
	quando    []time.Time
	responder func(n int) (status int, tipoErro string)
}

func (f *sesFalso) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || r.URL.Path != "/v2/email/outbound-emails" {
		http.Error(w, "rota inesperada "+r.URL.Path, http.StatusNotFound)
		return
	}
	var corpo map[string]any
	if err := json.NewDecoder(r.Body).Decode(&corpo); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	f.mu.Lock()
	f.recebidos = append(f.recebidos, corpo)
	f.quando = append(f.quando, time.Now())
	n := len(f.recebidos)
	f.mu.Unlock()
	status, tipo := http.StatusOK, ""
	if f.responder != nil {
		status, tipo = f.responder(n)
	}
	w.Header().Set("Content-Type", "application/json")
	if status != http.StatusOK {
		w.Header().Set("X-Amzn-ErrorType", tipo)
		w.WriteHeader(status)
		w.Write([]byte(`{"message":"simulado: ` + tipo + `"}`))
		return
	}
	w.Write([]byte(`{"MessageId":"ses-` + time.Now().Format("150405.000000000") + `"}`))
}

func (f *sesFalso) total() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.recebidos)
}

// remetenteSES aponta o cliente do SES para o falso. Sem repetição do próprio SDK: cada tentativa é
// da fila, e o teste conta todas.
func remetenteSES(t *testing.T, f *sesFalso, porSegundo float64) *notificacao.SES {
	t.Helper()
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	cli := sesv2.New(sesv2.Options{
		Region: "sa-east-1", BaseEndpoint: aws.String(srv.URL), RetryMaxAttempts: 1,
		Credentials: credentials.NewStaticCredentialsProvider("teste", "teste", ""),
	})
	return notificacao.NovoSES(cli, "avisos@lisura.test", "lisura-avisos", porSegundo)
}

func campo(m map[string]any, caminho ...string) any {
	var v any = m
	for _, c := range caminho {
		mm, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		v = mm[c]
	}
	return v
}

// O SES recebe exatamente o aviso: remetente, destinatário, assunto e texto em UTF-8, o conjunto de
// configuração, e a chave no cabeçalho e na etiqueta.
func TestSESMandaOQueOAvisoDiz(t *testing.T) {
	f := &sesFalso{}
	ses := remetenteSES(t, f, 0)
	msg := Mensagem{Chave: "evt-42", Para: "ana@exemplo.test", Assunto: "Inscrição nº 7 homologada", Texto: "Olá, Ana.\n"}
	id, err := ses.Enviar(context.Background(), msg)
	must(t, err)
	if id == "" || f.total() != 1 {
		t.Fatalf("id %q, %d envios", id, f.total())
	}
	c := f.recebidos[0]
	para, _ := campo(c, "Destination", "ToAddresses").([]any)
	cab, _ := campo(c, "Content", "Simple", "Headers").([]any)
	tag, _ := campo(c, "EmailTags").([]any)
	switch {
	case campo(c, "FromEmailAddress") != "avisos@lisura.test", len(para) != 1 || para[0] != msg.Para:
		t.Errorf("remetente ou destinatário: %v", c)
	case campo(c, "Content", "Simple", "Subject", "Data") != msg.Assunto, campo(c, "Content", "Simple", "Body", "Text", "Data") != msg.Texto,
		campo(c, "Content", "Simple", "Subject", "Charset") != "UTF-8":
		t.Errorf("assunto ou texto: %v", c)
	case campo(c, "ConfigurationSetName") != "lisura-avisos":
		t.Errorf("conjunto de configuração: %v", c)
	case len(cab) != 1 || campo(cab[0].(map[string]any), "Name") != "X-Lisura-Chave" || campo(cab[0].(map[string]any), "Value") != "evt-42":
		t.Errorf("cabeçalho da chave: %v", cab)
	case len(tag) != 1 || campo(tag[0].(map[string]any), "Value") != "evt-42":
		t.Errorf("etiqueta da chave: %v", tag)
	}
}

// A mensagem recusada é definitiva (a fila não insiste); a vazão estourada e a falha do serviço são
// passageiras (a fila repete).
func TestSESRecusaEDefinitivaVazaoNao(t *testing.T) {
	for _, c := range []struct {
		status     int
		tipo       string
		definitivo bool
	}{
		{http.StatusBadRequest, "MessageRejected", true},
		{http.StatusTooManyRequests, "TooManyRequestsException", false},
		{http.StatusBadRequest, "SendingPausedException", false},
		{http.StatusInternalServerError, "InternalFailure", false},
	} {
		f := &sesFalso{responder: func(int) (int, string) { return c.status, c.tipo }}
		_, err := remetenteSES(t, f, 0).Enviar(context.Background(), Mensagem{Chave: "evt-1", Para: "x@exemplo.test", Assunto: "a", Texto: "t"})
		if err == nil || errors.Is(err, notificacao.ErrEnderecoRecusado) != c.definitivo {
			t.Errorf("%s: erro %v; definitivo esperado %v", c.tipo, err, c.definitivo)
		}
	}
}

// Dez envios ao mesmo tempo, a 20 por segundo: saem espaçados, não de uma vez.
func TestSESEspacaOsEnvios(t *testing.T) {
	f := &sesFalso{}
	ses := remetenteSES(t, f, 20)
	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := ses.Enviar(context.Background(), Mensagem{Chave: "evt-1", Para: "x@exemplo.test", Assunto: "a", Texto: "t"})
			if err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	f.mu.Lock()
	defer f.mu.Unlock()
	primeiro, ultimo := f.quando[0], f.quando[0]
	for _, q := range f.quando {
		if q.Before(primeiro) {
			primeiro = q
		}
		if q.After(ultimo) {
			ultimo = q
		}
	}
	// 10 envios a 20/s: 9 intervalos de 50 ms. Folga para o relógio do Windows (~15 ms de resolução).
	if d := ultimo.Sub(primeiro); d < 400*time.Millisecond {
		t.Fatalf("10 envios em %v: não foram espaçados", d)
	}
}

// O limite do SES, provado: a mensagem sai, a resposta se perde, a fila repete, e a candidata recebe
// duas cópias, as duas com a mesma chave. A prova do aviso continua uma só.
func TestSESRespostaPerdidaAvisaPeloMenosUmaVez(t *testing.T) {
	c := novo(t)
	f := &sesFalso{responder: func(n int) (int, string) {
		if n == 1 {
			return http.StatusInternalServerError, "InternalFailure"
		}
		return http.StatusOK, ""
	}}
	nada := func(context.Context, json.RawMessage) error { return nil }
	tr := &fila.Trabalhador{Pool: pool, Manipuladores: map[string]fila.Manipulador{
		Tarefa: notificacao.Notificar(remetenteSES(t, f, 0), pool), publicacao.TarefaCartao: nada,
		publicacao.TarefaCertame: nada, publicacao.TarefaRelacao: nada}}
	ctx := context.Background()
	must(t, tr.Esvaziar(ctx))
	if len(c.avisos(t)) != 0 {
		t.Fatal("o aviso foi dado por entregue com a resposta perdida")
	}
	_, err := pool.Exec(ctx, `UPDATE tarefa SET executar_em = now() WHERE tipo = $1 AND estado = 'PENDENTE'`, Tarefa)
	must(t, err)
	must(t, tr.Esvaziar(ctx))
	if f.total() != 2 || len(c.avisos(t)) != 1 || !c.avisos(t)[0].Entregue {
		t.Fatalf("%d envios ao SES, %d avisos; esperado 2 e 1", f.total(), len(c.avisos(t)))
	}
	for _, r := range f.recebidos {
		if v, _ := campo(r, "EmailTags").([]any); len(v) != 1 || campo(v[0].(map[string]any), "Value") != f.recebidos[0]["EmailTags"].([]any)[0].(map[string]any)["Value"] {
			t.Fatalf("as duas cópias não levam a mesma chave: %v", f.recebidos)
		}
	}
}
