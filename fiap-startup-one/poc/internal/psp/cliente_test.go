package psp

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// 32 cobranças simultâneas, 20 vezes seguidas: as conexões abertas na primeira rajada têm de ser
// reaproveitadas nas seguintes. Com o transporte padrão (2 ociosas por destino) e o corpo fechado
// sem ler, quase toda chamada abria uma conexão nova.
func TestClienteReaproveitaConexoes(t *testing.T) {
	var novas atomic.Int64
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{\"ok\":true}\n")) // o "\n" final é o que um Decode deixa por ler
	}))
	srv.Config.ConnState = func(_ net.Conn, s http.ConnState) {
		if s == http.StateNew {
			novas.Add(1)
		}
	}
	srv.Start()
	defer srv.Close()

	c := Cliente{URL: srv.URL, HTTP: NovoHTTP(5 * time.Second)}
	const simultaneas, rodadas = 32, 20
	for range rodadas {
		var wg sync.WaitGroup
		for range simultaneas {
			wg.Go(func() {
				if err := c.SimularPagamento(context.Background(), "cob_1"); err != nil {
					t.Error(err)
				}
			})
		}
		wg.Wait()
	}
	if n := novas.Load(); n > 2*simultaneas {
		t.Fatalf("%d conexões novas para %d chamadas: o cliente não está reaproveitando", n, simultaneas*rodadas)
	}
	t.Logf("%d conexões novas para %d chamadas", novas.Load(), simultaneas*rodadas)
}
