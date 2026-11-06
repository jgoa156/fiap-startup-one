// Package ensaio dispara os ensaios de carga a partir da página de operação (/operacao no front): o
// mesmo k6 e a mesma auditoria do infra/scripts/ensaio.ps1, mas acionados por um botão, com o
// andamento na tela e as métricas indo, em tempo real, para o Prometheus (e dali para o Grafana).
//
// Os cenários:
//   - ultimo-dia: o pico de inscrição e pagamento do último dia, num multiplicador da carga média;
//   - escada: o teste de estresse, subindo o multiplicador (dobrando) até quebrar ou até o teto;
//   - pico-resultado: a base inteira abrindo o resultado ao mesmo tempo, só arquivo estático.
//
// Um ensaio de cada vez: dois ao mesmo tempo disputariam a mesma máquina e nenhum mediria nada.
package ensaio

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"lisura/internal/auditoria"
)

// Cenario descreve um cenário para a página.
type Cenario struct {
	ID        string `json:"id"`
	Nome      string `json:"nome"`
	Descricao string `json:"descricao"`
}

var Cenarios = []Cenario{
	{"ultimo-dia", "Último dia de inscrição", "Inscrição, Pix e boleto no multiplicador escolhido da carga média (40× é o H1). No fim, a auditoria confere que nada se perdeu e nada foi cobrado duas vezes."},
	{"escada", "Estresse: subir até quebrar", "Começa no multiplicador escolhido e dobra a cada patamar, até um patamar falhar (k6 ou auditoria) ou chegar ao teto. Mostra onde o sistema quebra."},
	{"pico-resultado", "Pico do resultado", "A base inteira abre o resultado ao mesmo tempo: só arquivo estático, nenhuma chamada à API. Chegadas por segundo escolhidas."},
}

// Pedido é o que a página manda.
type Pedido struct {
	Cenario string  `json:"cenario"`
	Mult    float64 `json:"mult"`    // ultimo-dia e escada
	Duracao string  `json:"duracao"` // por patamar, como o k6 entende (30s, 2m)
	Certame int64   `json:"certame"`
	Teto    float64 `json:"teto"` // escada
	Taxa    int     `json:"taxa"` // pico-resultado: chegadas por segundo
}

// Patamar é uma rodada do k6.
type Patamar struct {
	Mult      float64    `json:"mult,omitempty"`
	Taxa      int        `json:"taxa,omitempty"`
	Rodada    string     `json:"rodada"`
	Inicio    time.Time  `json:"inicio"`
	Fim       *time.Time `json:"fim,omitempty"`
	SaidaK6   *int       `json:"saida_k6,omitempty"` // 0 passou; 99 = um limite (threshold) do k6 estourou
	Enviadas  int        `json:"enviadas"`
	Pagas     int        `json:"pagas"`
	Problemas []string   `json:"problemas"`
	Passou    *bool      `json:"passou,omitempty"`
}

// Ensaio é um disparo, com um ou mais patamares.
type Ensaio struct {
	ID        string     `json:"id"`
	Pedido    Pedido     `json:"pedido"`
	Estado    string     `json:"estado"` // rodando, passou, quebrou, parado, erro
	Inicio    time.Time  `json:"inicio"`
	Fim       *time.Time `json:"fim,omitempty"`
	Patamares []Patamar  `json:"patamares"`
	Erro      string     `json:"erro,omitempty"`
	Log       []string   `json:"log"` // as últimas linhas do k6
}

const linhasDeLog = 300

var (
	ErrOcupado = errors.New("já há um ensaio rodando")
	ErrPedido  = errors.New("pedido inválido")
)

// Executor roda um ensaio de cada vez.
type Executor struct {
	K6       string        // o executável do k6
	Roteiros string        // a pasta carga/ (ultimo-dia.js, pico-resultado.js)
	Base     string        // a API (BASE_URL do k6)
	Front    string        // o front de produção (FRONT_URL)
	Escrita  string        // remote write do Prometheus; vazio: sem métricas do k6
	TokenKey string        // a LISURA_TOKEN_KEY da API (o pico do resultado deriva as chaves dela)
	Pool     *pgxpool.Pool // faixas de CPF livres, faixa de ids do pico, auditoria
	// Auditar confere a rodada. Padrão: auditoria.Auditar no Pool. Os testes trocam.
	Auditar func(ctx context.Context, rodada string, e auditoria.Esperado) (auditoria.Relatorio, error)
	// Faixas devolve as faixas de CPF ainda livres (cada rodada usa uma). Padrão: lidas do Pool.
	Faixas func(ctx context.Context) ([]int, error)
	// Comecou e Terminou avisam (as métricas do executor).
	Comecou, Terminou func(e Ensaio)

	mu        sync.Mutex
	atual     *Ensaio
	cancelar  context.CancelFunc
	historico []Ensaio
}

// Estado é a fotografia para a página.
func (x *Executor) Estado() (atual *Ensaio, historico []Ensaio) {
	x.mu.Lock()
	defer x.mu.Unlock()
	if x.atual != nil {
		c := *x.atual
		c.Patamares = append([]Patamar(nil), x.atual.Patamares...)
		c.Log = append([]string(nil), x.atual.Log...)
		atual = &c
	}
	// []Ensaio{} e não nil: vazio sai como [] no JSON, não null (a página lê o tamanho).
	return atual, append([]Ensaio{}, x.historico...)
}

func validar(p *Pedido) error {
	if p.Duracao == "" {
		p.Duracao = "1m"
	}
	if d, err := time.ParseDuration(p.Duracao); err != nil || d < 10*time.Second || d > 3*time.Hour {
		return fmt.Errorf("%w: duração entre 10s e 3h (veio %q)", ErrPedido, p.Duracao)
	}
	switch p.Cenario {
	case "ultimo-dia", "escada":
		if p.Mult <= 0 || p.Mult > 2560 {
			return fmt.Errorf("%w: multiplicador entre 0 e 2560", ErrPedido)
		}
		if p.Certame <= 0 {
			return fmt.Errorf("%w: certame", ErrPedido)
		}
		if p.Cenario == "escada" {
			if p.Teto == 0 {
				p.Teto = 1280
			}
			if p.Teto < p.Mult || p.Teto > 2560 {
				return fmt.Errorf("%w: teto entre o multiplicador inicial e 2560", ErrPedido)
			}
		}
	case "pico-resultado":
		if p.Taxa <= 0 || p.Taxa > 20000 {
			return fmt.Errorf("%w: chegadas por segundo entre 1 e 20000", ErrPedido)
		}
		if p.Certame <= 0 {
			return fmt.Errorf("%w: certame calculado", ErrPedido)
		}
	default:
		return fmt.Errorf("%w: cenário %q", ErrPedido, p.Cenario)
	}
	return nil
}

// Iniciar dispara o ensaio em segundo plano e devolve já.
func (x *Executor) Iniciar(p Pedido) (Ensaio, error) {
	if err := validar(&p); err != nil {
		return Ensaio{}, err
	}
	x.mu.Lock()
	defer x.mu.Unlock()
	if x.atual != nil {
		return Ensaio{}, ErrOcupado
	}
	ctx, cancel := context.WithCancel(context.Background())
	e := &Ensaio{ID: time.Now().Format("20060102-150405"), Pedido: p, Estado: "rodando", Inicio: time.Now(), Patamares: []Patamar{}, Log: []string{}}
	x.atual, x.cancelar = e, cancel
	if x.Comecou != nil {
		x.Comecou(*e)
	}
	go x.rodar(ctx, e)
	return *e, nil
}

// Parar interrompe o ensaio em andamento (o k6 é encerrado; o patamar fica como parado).
func (x *Executor) Parar() bool {
	x.mu.Lock()
	defer x.mu.Unlock()
	if x.cancelar == nil {
		return false
	}
	x.cancelar()
	return true
}

func (x *Executor) anotar(e *Ensaio, linha string) {
	x.mu.Lock()
	defer x.mu.Unlock()
	e.Log = append(e.Log, linha)
	if len(e.Log) > linhasDeLog {
		e.Log = e.Log[len(e.Log)-linhasDeLog:]
	}
}

func (x *Executor) mudar(f func()) {
	x.mu.Lock()
	defer x.mu.Unlock()
	f()
}

func (x *Executor) rodar(ctx context.Context, e *Ensaio) {
	estado, erro := "passou", ""
	defer func() {
		x.mu.Lock()
		agora := time.Now()
		if ctx.Err() != nil && estado != "erro" {
			estado = "parado"
		}
		e.Estado, e.Erro, e.Fim = estado, erro, &agora
		x.historico = append([]Ensaio{*e}, x.historico...)
		if len(x.historico) > 20 {
			x.historico = x.historico[:20]
		}
		x.atual, x.cancelar = nil, nil
		fim := *e
		x.mu.Unlock()
		if x.Terminou != nil {
			x.Terminou(fim)
		}
	}()

	p := e.Pedido
	if p.Cenario == "pico-resultado" {
		ok, err := x.picoResultado(ctx, e)
		if err != nil {
			estado, erro = "erro", err.Error()
		} else if !ok {
			estado = "quebrou"
		}
		return
	}
	faixas, err := x.faixas(ctx)
	if err != nil {
		estado, erro = "erro", err.Error()
		return
	}
	mult := p.Mult
	for i := 0; ; i++ {
		if i >= len(faixas) {
			estado, erro = "erro", "acabaram as faixas de CPF livres (99 por banco)"
			return
		}
		passou, err := x.ultimoDia(ctx, e, mult, faixas[i])
		if err != nil {
			estado, erro = "erro", err.Error()
			return
		}
		if ctx.Err() != nil {
			return
		}
		if !passou {
			estado = "quebrou"
			return
		}
		if p.Cenario != "escada" || mult*2 > p.Teto {
			return
		}
		mult *= 2
	}
}

var paraAuditar = regexp.MustCompile(`PARA_AUDITAR lisura-auditoria -rodada (\S+) -enviadas (\d+) -pagas (\d+)`)

// ultimoDia roda um patamar do último dia e audita a rodada.
func (x *Executor) ultimoDia(ctx context.Context, e *Ensaio, mult float64, faixa int) (bool, error) {
	rodada := fmt.Sprintf("%d%02d", int(mult+0.5), faixa)
	pat := Patamar{Mult: mult, Rodada: rodada, Inicio: time.Now(), Problemas: []string{}}
	x.mudar(func() { e.Patamares = append(e.Patamares, pat) })
	i := len(e.Patamares) - 1
	env := []string{
		"CERTAME=" + strconv.FormatInt(e.Pedido.Certame, 10), "MULT=" + strconv.FormatFloat(mult, 'f', -1, 64),
		"DURACAO=" + e.Pedido.Duracao, "RODADA=" + rodada, "PENSAR=0.05", "RAJADAS=0",
		"BASE_URL=" + x.Base, "FRONT_URL=" + x.Front,
	}
	var achou []string
	saida, err := x.k6(ctx, e, "ultimo-dia.js", env, func(l string) {
		if m := paraAuditar.FindStringSubmatch(l); m != nil {
			achou = m
		}
	})
	fim := time.Now()
	x.mudar(func() { e.Patamares[i].Fim, e.Patamares[i].SaidaK6 = &fim, &saida })
	if err != nil || ctx.Err() != nil {
		return false, err
	}
	var problemas []string
	enviadas, pagas := 0, 0
	if achou == nil {
		problemas = append(problemas, "o k6 não imprimiu PARA_AUDITAR")
	} else {
		enviadas, _ = strconv.Atoi(achou[2])
		pagas, _ = strconv.Atoi(achou[3])
		x.anotar(e, "--- auditando a rodada "+rodada)
		r, err := x.auditar(ctx, rodada, auditoria.Esperado{Enviadas: enviadas, Pagas: pagas})
		if err != nil {
			return false, fmt.Errorf("auditoria: %w", err)
		}
		problemas = append(problemas, r.Problemas...)
		if enviadas == 0 {
			problemas = append(problemas, "nenhuma inscrição enviada: o front ou a API estão no ar?")
		}
	}
	if saida != 0 {
		problemas = append(problemas, fmt.Sprintf("o k6 saiu com %d (um limite estourou)", saida))
	}
	passou := len(problemas) == 0
	x.mudar(func() {
		e.Patamares[i].Enviadas, e.Patamares[i].Pagas = enviadas, pagas
		e.Patamares[i].Problemas, e.Patamares[i].Passou = problemas, &passou
	})
	return passou, nil
}

func (x *Executor) picoResultado(ctx context.Context, e *Ensaio) (bool, error) {
	if x.TokenKey == "" {
		return false, errors.New("o executor não tem a LISURA_TOKEN_KEY da API")
	}
	var idMin, idMax *int64
	if err := x.Pool.QueryRow(ctx, `SELECT min(id), max(id) FROM inscricao WHERE certame_id = $1 AND estado = 'HOMOLOGADA'`,
		e.Pedido.Certame).Scan(&idMin, &idMax); err != nil {
		return false, err
	}
	if idMin == nil {
		return false, fmt.Errorf("o certame %d não tem inscrição homologada (calcule o resultado antes)", e.Pedido.Certame)
	}
	pat := Patamar{Taxa: e.Pedido.Taxa, Rodada: "pico5", Inicio: time.Now(), Problemas: []string{}}
	x.mudar(func() { e.Patamares = append(e.Patamares, pat) })
	env := []string{
		"PUBLICO_URL=" + x.Base, "CERTAME=" + strconv.FormatInt(e.Pedido.Certame, 10),
		"ID_MIN=" + strconv.FormatInt(*idMin, 10), "ID_MAX=" + strconv.FormatInt(*idMax, 10),
		"TOKEN_KEY=" + x.TokenKey, "TAXA=" + strconv.Itoa(e.Pedido.Taxa), "DURACAO=" + e.Pedido.Duracao,
	}
	saida, err := x.k6(ctx, e, "pico-resultado.js", env, nil)
	fim := time.Now()
	passou := err == nil && saida == 0
	problemas := []string{}
	if saida != 0 {
		problemas = append(problemas, fmt.Sprintf("o k6 saiu com %d (um limite estourou)", saida))
	}
	x.mudar(func() {
		e.Patamares[0].Fim, e.Patamares[0].SaidaK6, e.Patamares[0].Passou, e.Patamares[0].Problemas = &fim, &saida, &passou, problemas
	})
	return passou, err
}

// k6 roda um roteiro, mandando cada linha da saída para o log do ensaio (e para quem quiser ler),
// e as métricas para o Prometheus. Devolve o código de saída do k6.
func (x *Executor) k6(ctx context.Context, e *Ensaio, roteiro string, env []string, linha func(string)) (int, error) {
	args := []string{"run", "--quiet", "--tag", "ensaio=" + e.ID}
	if x.Escrita != "" {
		args = append(args, "--out", "experimental-prometheus-rw")
		env = append(env, "K6_PROMETHEUS_RW_SERVER_URL="+x.Escrita, "K6_PROMETHEUS_RW_PUSH_INTERVAL=5s",
			"K6_PROMETHEUS_RW_TREND_STATS=p(95),p(99),avg,max")
	}
	args = append(args, filepath.Join(x.Roteiros, roteiro))
	cmd := exec.CommandContext(ctx, x.K6, args...)
	cmd.Dir = x.Roteiros
	cmd.Env = append(os.Environ(), env...)
	cmd.WaitDelay = 10 * time.Second
	leitor, escritor := io.Pipe()
	cmd.Stdout, cmd.Stderr = escritor, escritor
	x.anotar(e, "--- "+filepath.Base(roteiro)+" "+strings.Join(publicas(env), " "))
	if err := cmd.Start(); err != nil {
		return -1, fmt.Errorf("k6: %w", err)
	}
	lido := make(chan struct{})
	go func() {
		defer close(lido)
		s := bufio.NewScanner(leitor)
		s.Buffer(make([]byte, 64<<10), 1<<20)
		for s.Scan() {
			l := s.Text()
			x.anotar(e, l)
			if linha != nil {
				linha(l)
			}
		}
	}()
	err := cmd.Wait()
	escritor.Close()
	<-lido
	var sair *exec.ExitError
	switch {
	case err == nil:
		return 0, nil
	case ctx.Err() != nil:
		return -1, nil // parado pela página
	case errors.As(err, &sair):
		return sair.ExitCode(), nil
	}
	return -1, err
}

// publicas tira do log as variáveis que não são para tela (a chave dos tokens).
func publicas(env []string) []string {
	var out []string
	for _, v := range env {
		if !strings.HasPrefix(v, "TOKEN_KEY=") && !strings.HasPrefix(v, "K6_") {
			out = append(out, v)
		}
	}
	return out
}

func (x *Executor) auditar(ctx context.Context, rodada string, esp auditoria.Esperado) (auditoria.Relatorio, error) {
	if x.Auditar != nil {
		return x.Auditar(ctx, rodada, esp)
	}
	return auditoria.Auditar(ctx, x.Pool, rodada, esp)
}

func (x *Executor) faixas(ctx context.Context) ([]int, error) {
	if x.Faixas != nil {
		return x.Faixas(ctx)
	}
	return FaixasLivres(ctx, x.Pool)
}

// FaixasLivres são as faixas de CPF (os dois primeiros dígitos) que nenhum candidato usa ainda: o k6
// gera os CPFs da rodada numa faixa, e repetir uma faixa faria a rodada esbarrar em contas antigas.
func FaixasLivres(ctx context.Context, pool *pgxpool.Pool) ([]int, error) {
	rows, err := pool.Query(ctx, `SELECT DISTINCT left(cpf, 2) FROM candidato`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	usadas := map[string]bool{}
	for rows.Next() {
		var f string
		if err := rows.Scan(&f); err != nil {
			return nil, err
		}
		usadas[f] = true
	}
	var livres []int
	for f := 1; f <= 99; f++ {
		if !usadas[fmt.Sprintf("%02d", f)] {
			livres = append(livres, f)
		}
	}
	return livres, rows.Err()
}
