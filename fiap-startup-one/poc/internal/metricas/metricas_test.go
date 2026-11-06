package metricas

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func raspar(t *testing.T, m *Metricas) string {
	t.Helper()
	w := httptest.NewRecorder()
	m.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	b, _ := io.ReadAll(w.Body)
	return string(b)
}

// Duas inscrições diferentes são a mesma série (o padrão da rota), com o status de cada resposta; o
// que não casa com rota nenhuma também é uma série só. Com a URL no rótulo, cada inscrição viraria
// uma série e o Prometheus estouraria a memória no pico.
func TestRequisicaoMedidaPeloPadraoDaRota(t *testing.T) {
	m := Nova("teste")
	mux := http.NewServeMux()
	mux.HandleFunc("GET /inscricoes/{id}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("id") == "2" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write([]byte("ok"))
	})
	h := m.Rotas(mux)
	for _, u := range []string{"/inscricoes/1", "/inscricoes/1", "/inscricoes/2", "/qualquer/coisa/123", "/qualquer/coisa/456"} {
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, u, nil))
	}
	saida := raspar(t, m)
	for _, quer := range []string{
		`lisura_http_requisicoes_total{metodo="GET",rota="GET /inscricoes/{id}",status="200"} 2`,
		`lisura_http_requisicoes_total{metodo="GET",rota="GET /inscricoes/{id}",status="404"} 1`,
		`lisura_http_requisicoes_total{metodo="GET",rota="sem rota",status="404"} 2`,
		`lisura_http_duracao_segundos_count{metodo="GET",rota="GET /inscricoes/{id}"} 3`,
		`lisura_instancia_info{`,
		`process_cpu_seconds_total`,
	} {
		if !strings.Contains(saida, quer) {
			t.Errorf("faltou %s", quer)
		}
	}
	for _, nao := range []string{"/inscricoes/1", "/qualquer/coisa/123"} {
		if strings.Contains(saida, `rota="`+nao) {
			t.Errorf("a URL virou rótulo: %s", nao)
		}
	}
}

// Cada tarefa da fila conta por tipo e resultado, e a espera e a execução entram nos histogramas.
func TestTarefaDaFilaMedida(t *testing.T) {
	m := Nova("teste")
	m.ObservarFila("notificar", 3*time.Second, 10*time.Millisecond, false)
	m.ObservarFila("notificar", time.Second, 5*time.Millisecond, true)
	saida := raspar(t, m)
	for _, quer := range []string{
		`lisura_fila_tarefas_total{resultado="ok",tipo="notificar"} 1`,
		`lisura_fila_tarefas_total{resultado="falha",tipo="notificar"} 1`,
		`lisura_fila_espera_segundos_sum{tipo="notificar"} 4`,
		`lisura_fila_execucao_segundos_count{tipo="notificar"} 2`,
	} {
		if !strings.Contains(saida, quer) {
			t.Errorf("faltou %s\n%s", quer, saida)
		}
	}
}
