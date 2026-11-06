// Package metricas expõe o que o sistema está fazendo agora, no formato do Prometheus (o coletor de
// séries temporais que o Grafana lê): requisições por rota, a fila de tarefas, a sala de espera, as
// conexões do banco e o processo. O painel de operação e o ensaio de carga leem daqui, em tempo real.
//
// Na máquina local o Prometheus raspa GET /metrics de cada instância; na AWS, o mesmo endereço
// fica dentro da VPC (o balanceador só repassa /api e o front).
package metricas

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"lisura/internal/degradacao"
	"lisura/internal/fila"
)

// Faixas dos histogramas: do pedido rápido (1 ms) ao limite do H1 (2 s) e além.
var faixasHTTP = []float64{.001, .0025, .005, .01, .025, .05, .1, .25, .5, 1, 2, 5, 10}

// Faixas da fila: de milissegundos a minutos (o aviso por e-mail espera o pico escoar).
var faixasFila = []float64{.005, .025, .1, .5, 1, 2.5, 5, 15, 30, 60, 120, 300}

// Metricas é o registro de uma instância.
type Metricas struct {
	Registro *prometheus.Registry

	requisicoes *prometheus.CounterVec
	duracao     *prometheus.HistogramVec
	tarefas     *prometheus.CounterVec
	espera      *prometheus.HistogramVec
	execucao    *prometheus.HistogramVec
}

// Nova cria o registro com as métricas do Go e do processo (CPU, memória, goroutines) e uma série
// lisura_instancia_info, que diz qual instância é qual e quando subiu: o painel mostra o servidor
// mudando (subiu, caiu, trocou de versão) pela troca desta série.
func Nova(versao string) *Metricas {
	r := prometheus.NewRegistry()
	r.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	host, _ := os.Hostname()
	info := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "lisura_instancia_info", Help: "Uma série por instância no ar; o valor é a hora de subida (Unix).",
		ConstLabels: prometheus.Labels{"versao": versao, "host": host, "pid": strconv.Itoa(os.Getpid())},
	})
	info.Set(float64(time.Now().Unix()))
	m := &Metricas{
		Registro: r,
		requisicoes: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "lisura_http_requisicoes_total", Help: "Requisições respondidas, por rota, método e status."},
			[]string{"rota", "metodo", "status"}),
		duracao: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "lisura_http_duracao_segundos", Help: "Tempo de resposta, por rota e método.", Buckets: faixasHTTP},
			[]string{"rota", "metodo"}),
		tarefas: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "lisura_fila_tarefas_total", Help: "Tarefas executadas, por tipo e resultado (ok, falha)."},
			[]string{"tipo", "resultado"}),
		espera: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "lisura_fila_espera_segundos", Help: "Quanto a tarefa esperou na fila (falta de trabalhador).", Buckets: faixasFila},
			[]string{"tipo"}),
		execucao: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "lisura_fila_execucao_segundos", Help: "Quanto a tarefa levou para rodar (tarefa lenta).", Buckets: faixasFila},
			[]string{"tipo"}),
	}
	r.MustRegister(info, m.requisicoes, m.duracao, m.tarefas, m.espera, m.execucao)
	return m
}

// Handler é o GET /metrics.
func (m *Metricas) Handler() http.Handler {
	return promhttp.HandlerFor(m.Registro, promhttp.HandlerOpts{Registry: m.Registro})
}

type gravador struct {
	http.ResponseWriter
	status int
}

func (g *gravador) WriteHeader(s int) {
	if g.status == 0 {
		g.status = s
	}
	g.ResponseWriter.WriteHeader(s)
}

func (g *gravador) Write(b []byte) (int, error) {
	if g.status == 0 {
		g.status = http.StatusOK
	}
	return g.ResponseWriter.Write(b)
}

// Flush deixa passar o streaming (o fluxo de eventos do ensaio).
func (g *gravador) Flush() {
	if f, ok := g.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Rotas mede cada requisição pelo padrão da rota (GET /inscricoes/{id}), nunca pela URL: com o id
// no rótulo, cada inscrição viraria uma série, e o Prometheus estoura a memória.
func (m *Metricas) Rotas(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, padrao := mux.Handler(r)
		if padrao == "" {
			padrao = "sem rota"
		}
		g := &gravador{ResponseWriter: w}
		inicio := time.Now()
		mux.ServeHTTP(g, r)
		if g.status == 0 {
			g.status = http.StatusOK
		}
		m.requisicoes.WithLabelValues(padrao, r.Method, strconv.Itoa(g.status)).Inc()
		m.duracao.WithLabelValues(padrao, r.Method).Observe(time.Since(inicio).Seconds())
	})
}

// ObservarFila é o gancho fila.Trabalhador.Observar.
func (m *Metricas) ObservarFila(tipo string, espera, execucao time.Duration, falhou bool) {
	resultado := "ok"
	if falhou {
		resultado = "falha"
	}
	m.tarefas.WithLabelValues(tipo, resultado).Inc()
	m.espera.WithLabelValues(tipo).Observe(espera.Seconds())
	m.execucao.WithLabelValues(tipo).Observe(execucao.Seconds())
}

// estado lê, a cada raspagem, o que só existe no momento: a fila pendente, os trabalhadores, a sala
// de espera e o banco. Nada disto é contado em memória: a fila pendente é de todas as instâncias, e
// só o banco sabe.
type estado struct {
	pool        *pgxpool.Pool
	trabalhador *fila.Trabalhador
	controle    *degradacao.Controle

	pendentes, vencidas, maisVelha        *prometheus.Desc
	trabalhadores, rodando                *prometheus.Desc
	nivel, senhas, liberadas, admitidos   *prometheus.Desc
	conexoes, esperasConexao, maxConexoes *prometheus.Desc
}

// Acompanhar registra as medidas lidas na hora. Qualquer um dos três pode ser nil.
func (m *Metricas) Acompanhar(pool *pgxpool.Pool, t *fila.Trabalhador, c *degradacao.Controle) {
	d := func(nome, ajuda string, rotulos ...string) *prometheus.Desc {
		return prometheus.NewDesc(nome, ajuda, rotulos, nil)
	}
	m.Registro.MustRegister(&estado{pool: pool, trabalhador: t, controle: c,
		pendentes:      d("lisura_fila_pendentes", "Tarefas pendentes (de todas as instâncias), por tipo e prioridade.", "tipo", "prioridade"),
		vencidas:       d("lisura_fila_vencidas", "Pendentes cuja hora já chegou: esperando trabalhador.", "tipo", "prioridade"),
		maisVelha:      d("lisura_fila_mais_velha_segundos", "Há quanto tempo a tarefa vencida mais antiga espera, por tipo.", "tipo"),
		trabalhadores:  d("lisura_fila_trabalhadores", "Trabalhadores da fila de pé nesta instância."),
		rodando:        d("lisura_fila_rodando", "Tarefas em execução agora nesta instância."),
		nivel:          d("lisura_degradacao_nivel", "Nível de degradação: 0 normal, 1 sala de espera, 2 só o essencial, 3 só consulta."),
		senhas:         d("lisura_sala_senhas_emitidas", "Senhas da sala de espera emitidas (todas as instâncias)."),
		liberadas:      d("lisura_sala_senhas_liberadas", "Senhas já chamadas pela sala de espera."),
		admitidos:      d("lisura_sala_admissoes_total", "Admissões dadas por esta instância."),
		conexoes:       d("lisura_db_conexoes", "Conexões do pool do banco, por estado (em_uso, ociosas).", "estado"),
		maxConexoes:    d("lisura_db_conexoes_max", "Tamanho máximo do pool do banco."),
		esperasConexao: d("lisura_db_esperas_total", "Vezes que um pedido esperou por conexão livre no pool."),
	})
}

func (e *estado) Describe(ch chan<- *prometheus.Desc) {
	for _, d := range []*prometheus.Desc{e.pendentes, e.vencidas, e.maisVelha, e.trabalhadores, e.rodando,
		e.nivel, e.senhas, e.liberadas, e.admitidos, e.conexoes, e.maxConexoes, e.esperasConexao} {
		ch <- d
	}
}

func (e *estado) Collect(ch chan<- prometheus.Metric) {
	g := func(d *prometheus.Desc, v float64, rotulos ...string) {
		ch <- prometheus.MustNewConstMetric(d, prometheus.GaugeValue, v, rotulos...)
	}
	if e.trabalhador != nil {
		g(e.trabalhadores, float64(e.trabalhador.Ativos()))
		g(e.rodando, float64(e.trabalhador.Rodando()))
	}
	if e.controle != nil {
		s := e.controle.Estado()
		g(e.nivel, float64(s.Nivel))
		g(e.senhas, float64(s.Senhas))
		g(e.liberadas, float64(s.Liberadas))
		ch <- prometheus.MustNewConstMetric(e.admitidos, prometheus.CounterValue, float64(s.Admitidos))
	}
	if e.pool == nil {
		return
	}
	st := e.pool.Stat()
	g(e.conexoes, float64(st.AcquiredConns()), "em_uso")
	g(e.conexoes, float64(st.IdleConns()), "ociosas")
	g(e.maxConexoes, float64(st.MaxConns()))
	ch <- prometheus.MustNewConstMetric(e.esperasConexao, prometheus.CounterValue, float64(st.EmptyAcquireCount()))

	// Só as pendentes, pelo índice parcial (tarefa_a_executar): as concluídas, que são milhões, não
	// entram. Prazo curto: uma raspagem lenta não pode segurar uma conexão no pico.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	rows, err := e.pool.Query(ctx, `
		SELECT tipo, prioridade, count(*), count(*) FILTER (WHERE executar_em <= now()),
		       coalesce(extract(epoch FROM now() - min(executar_em) FILTER (WHERE executar_em <= now())), 0)::float8
		FROM tarefa WHERE estado = 'PENDENTE' GROUP BY tipo, prioridade`)
	if err != nil {
		return
	}
	defer rows.Close()
	velha := map[string]float64{}
	for rows.Next() {
		var tipo string
		var prio int16
		var n, venc int64
		var idade float64
		if rows.Scan(&tipo, &prio, &n, &venc, &idade) != nil {
			return
		}
		p := strconv.Itoa(int(prio))
		g(e.pendentes, float64(n), tipo, p)
		g(e.vencidas, float64(venc), tipo, p)
		velha[tipo] = max(velha[tipo], idade)
	}
	for tipo, idade := range velha {
		g(e.maisVelha, idade, tipo)
	}
}
