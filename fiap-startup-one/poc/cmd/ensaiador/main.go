// ensaiador é o executor dos ensaios de carga que a página de operação (/operacao no front) aciona:
// roda o k6, audita cada patamar e manda as métricas do k6 ao Prometheus. Só na máquina local (na
// AWS o k6 roda como tarefa do ECS, por infra/scripts/tarefa.ps1).
//
//	ENSAIADOR_ADDR      endereço (padrão 127.0.0.1:8083: só desta máquina)
//	ENSAIADOR_ORIGENS   origens do painel aceitas (padrão http://127.0.0.1:8090,http://localhost:8090)
//	K6                  o executável do k6 (padrão C:\Users\rockb\tools\k6\k6.exe, ou k6 no PATH)
//	ENSAIADOR_ROTEIROS  a pasta carga/ (padrão ../carga)
//	BASE_URL, FRONT_URL a API e o front que o k6 visita (padrão :8082 e 127.0.0.1:8090)
//	PROMETHEUS_ESCRITA  remote write do Prometheus (padrão http://127.0.0.1:9090/api/v1/write; "" desliga)
//	LISURA_TOKEN_KEY    a mesma da API (o pico do resultado deriva as chaves dela)
//	LISURA_OPERACAO_SEGREDO o da API: o painel muda o nível de degradação por aqui
//	DATABASE_URL        o banco (faixas de CPF livres e auditoria)
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"lisura/internal/db"
	"lisura/internal/ensaio"
)

func env(k, padrao string) string {
	if v, ok := os.LookupEnv(k); ok {
		return v
	}
	return padrao
}

func main() {
	ctx := context.Background()
	pool, err := db.Connect(ctx, db.URL())
	if err != nil {
		log.Fatal(err)
	}
	k6 := env("K6", "")
	if k6 == "" {
		if p, err := exec.LookPath("k6"); err == nil {
			k6 = p
		} else {
			k6 = `C:\Users\rockb\tools\k6\k6.exe`
		}
	}

	// O que o Grafana mostra como "ensaio rodando", e a anotação de início e fim no painel.
	rodando := prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "lisura_ensaio_rodando",
		Help: "1 enquanto um ensaio roda, por cenário."}, []string{"cenario"})
	patamar := prometheus.NewGauge(prometheus.GaugeOpts{Name: "lisura_ensaio_multiplicador",
		Help: "Multiplicador da carga média no patamar em andamento (0 parado)."})
	reg := prometheus.NewRegistry()
	reg.MustRegister(rodando, patamar)
	for _, c := range ensaio.Cenarios {
		rodando.WithLabelValues(c.ID).Set(0)
	}

	x := &ensaio.Executor{
		K6: k6, Roteiros: env("ENSAIADOR_ROTEIROS", "../carga"),
		Base: env("BASE_URL", "http://localhost:8082"), Front: env("FRONT_URL", "http://127.0.0.1:8090"),
		Escrita: env("PROMETHEUS_ESCRITA", "http://127.0.0.1:9090/api/v1/write"), TokenKey: os.Getenv("LISURA_TOKEN_KEY"),
		Pool:     pool,
		Comecou:  func(e ensaio.Ensaio) { rodando.WithLabelValues(e.Pedido.Cenario).Set(1) },
		Terminou: func(e ensaio.Ensaio) { rodando.WithLabelValues(e.Pedido.Cenario).Set(0); patamar.Set(0) },
	}
	go func() { // o multiplicador do patamar em andamento, para o painel
		for range time.Tick(2 * time.Second) {
			if a, _ := x.Estado(); a != nil && len(a.Patamares) > 0 {
				patamar.Set(a.Patamares[len(a.Patamares)-1].Mult)
			}
		}
	}()

	origens := strings.Split(env("ENSAIADOR_ORIGENS", "http://127.0.0.1:8090,http://localhost:8090"), ",")
	mux := http.NewServeMux()
	mux.HandleFunc("GET /estado", func(w http.ResponseWriter, r *http.Request) {
		atual, hist := x.Estado()
		responder(w, http.StatusOK, map[string]any{"cenarios": ensaio.Cenarios, "atual": atual, "historico": hist})
	})
	mux.HandleFunc("POST /ensaios", guardar(origens, func(w http.ResponseWriter, r *http.Request) {
		var p ensaio.Pedido
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&p); err != nil {
			responder(w, http.StatusBadRequest, map[string]string{"erro": "JSON inválido"})
			return
		}
		e, err := x.Iniciar(p)
		switch {
		case errors.Is(err, ensaio.ErrOcupado):
			responder(w, http.StatusConflict, map[string]string{"erro": err.Error()})
		case err != nil:
			responder(w, http.StatusUnprocessableEntity, map[string]string{"erro": err.Error()})
		default:
			log.Printf("ensaio %s: %+v", e.ID, e.Pedido)
			responder(w, http.StatusAccepted, e)
		}
	}))
	mux.HandleFunc("POST /parar", guardar(origens, func(w http.ResponseWriter, r *http.Request) {
		if !x.Parar() {
			responder(w, http.StatusConflict, map[string]string{"erro": "nenhum ensaio rodando"})
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	// O nível de degradação, repassado à API com o segredo de operação, que fica aqui e não na tela.
	segredo := os.Getenv("LISURA_OPERACAO_SEGREDO")
	mux.HandleFunc("POST /nivel", guardar(origens, func(w http.ResponseWriter, r *http.Request) {
		if segredo == "" {
			responder(w, http.StatusServiceUnavailable, map[string]string{"erro": "o executor não tem o LISURA_OPERACAO_SEGREDO"})
			return
		}
		corpo, _ := io.ReadAll(http.MaxBytesReader(w, r.Body, 4<<10))
		req, _ := http.NewRequestWithContext(r.Context(), http.MethodPost, x.Base+"/api/operacao", bytes.NewReader(corpo))
		req.Header.Set("Authorization", "Operacao "+segredo)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			responder(w, http.StatusBadGateway, map[string]string{"erro": err.Error()})
			return
		}
		defer resp.Body.Close()
		w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}))
	mux.Handle("GET /metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	addr := env("ENSAIADOR_ADDR", "127.0.0.1:8083")
	log.Printf("ensaiador em %s (k6 %s, roteiros %s)", addr, k6, x.Roteiros)
	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Fatal(srv.ListenAndServe())
}

// guardar só deixa passar o painel: o cabeçalho X-Lisura-Painel (outro site não o manda sem uma
// pré-verificação CORS, que este servidor não autoriza) e, se veio, a origem do front.
func guardar(origens []string, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Lisura-Painel") != "1" {
			responder(w, http.StatusForbidden, map[string]string{"erro": "só o painel de operação"})
			return
		}
		if o := r.Header.Get("Origin"); o != "" {
			if u, err := url.Parse(o); err != nil || !slices.Contains(origens, u.Scheme+"://"+u.Host) {
				responder(w, http.StatusForbidden, map[string]string{"erro": "origem não autorizada"})
				return
			}
		}
		h(w, r)
	}
}

func responder(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
