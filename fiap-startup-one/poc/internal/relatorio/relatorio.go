// Package relatorio monta o relatório do ensaio de carga — o entregável da POC (04-escopo-poc §1) —
// a partir do que cada rodada já produz: o resumo do k6 (a linha RESUMO_DO_ENSAIO, ou o arquivo
// SAIDA) e o JSON da auditoria no banco. Nenhum número é digitado à mão: o relatório só junta,
// compara com os alvos do H1 e diz onde quebrou.
package relatorio

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

// Os alvos do H1 (04-escopo-poc §6).
const (
	AlvoP95Critico = 2000.0 // ms
	AlvoConcluidas = 0.98
	AlvoMult       = 40
	AlvoDuracao    = 3 * time.Hour
)

// K6 é o resumo que o handleSummary de carga/ultimo-dia.js imprime.
type K6 struct {
	Rodada             string          `json:"rodada"`
	Mult               float64         `json:"mult"`
	AlvoPorMinuto      float64         `json:"alvo_por_minuto"`
	Duracao            string          `json:"duracao"`
	InscricoesEnviadas int             `json:"inscricoes_enviadas"`
	PagamentosVistos   int             `json:"pagamentos_vistos"`
	CartoesAbertos     int             `json:"cartoes_abertos"`
	JornadaConcluida   float64         `json:"jornada_concluida"`
	Falhas             int             `json:"falhas"`
	Recarregamentos    int             `json:"recarregamentos"`
	CliquesDuplos      int             `json:"cliques_duplos"`
	P95CriticoMs       *float64        `json:"p95_critico_ms"`
	P95Ms              float64         `json:"p95_ms"`
	Requisicoes        int             `json:"requisicoes"`
	TempoAtePixP95     float64         `json:"tempo_ate_pix_p95_s"`
	TempoAteCartaoP95  float64         `json:"tempo_ate_cartao_p95_s"`
	EsperaNaFilaP95    float64         `json:"espera_na_fila_p95_s"`
	Limites            map[string]bool `json:"limites"`
}

// Auditoria é o que cmd/auditoria imprime (só os campos que o relatório usa).
type Auditoria struct {
	Rodada                       string         `json:"rodada"`
	Enviadas                     int            `json:"inscricoes_enviadas"`
	Pagas                        int            `json:"inscricoes_pagas"`
	CobrancasRepetidas           int            `json:"cobrancas_repetidas"`
	PagamentosAplicadosDuasVezes int            `json:"pagamentos_aplicados_duas_vezes"`
	PagasSemRecebimento          int            `json:"pagas_sem_recebimento"`
	ConfirmadasDuasVezes         int            `json:"inscricoes_confirmadas_duas_vezes"`
	PagoAMaisSemReembolso        int            `json:"pago_a_mais_sem_reembolso"`
	ReembolsosPendentes          int            `json:"reembolsos_pendentes"`
	Devolucoes                   map[string]int `json:"devolucoes_por_estado"`
	Divergencias                 int            `json:"divergencias_da_conciliacao"`
	TrilhasVerificadas           int            `json:"trilhas_verificadas"`
	TrilhasQuebradas             []int64        `json:"trilhas_quebradas"`
	Problemas                    []string       `json:"problemas"`
}

// LerK6 acha o resumo num log (a linha RESUMO_DO_ENSAIO, onde quer que esteja) ou no arquivo SAIDA
// ({"resumo": ...}).
func LerK6(b []byte) (K6, error) {
	var k K6
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(make([]byte, 0, 64<<10), 16<<20)
	for sc.Scan() {
		if _, depois, ok := strings.Cut(sc.Text(), "RESUMO_DO_ENSAIO "); ok {
			// Decoder, e não Unmarshal: lê o primeiro objeto e ignora o que vier depois na mesma linha
			// (o log do CloudWatch em texto emenda mensagens com tabulação).
			return k, json.NewDecoder(strings.NewReader(depois)).Decode(&k)
		}
	}
	var saida struct {
		Resumo *K6 `json:"resumo"`
	}
	if err := json.Unmarshal(b, &saida); err == nil && saida.Resumo != nil {
		return *saida.Resumo, nil
	}
	return k, errors.New("nem linha RESUMO_DO_ENSAIO nem arquivo SAIDA do k6")
}

// LerAuditoria aceita o JSON puro ou no meio de um log (o da tarefa no ECS mistura os dois): do
// primeiro "{" ao último "}".
func LerAuditoria(b []byte) (Auditoria, error) {
	var a Auditoria
	ini, fim := bytes.IndexByte(b, '{'), bytes.LastIndexByte(b, '}')
	if ini < 0 || fim < ini {
		return a, errors.New("sem JSON da auditoria")
	}
	return a, json.Unmarshal(b[ini:fim+1], &a)
}

// Patamar é uma rodada do ensaio, com o que custou (da fatura da AWS; zero = não informado).
type Patamar struct {
	K6        K6
	Auditoria Auditoria
	CustoUSD  float64
	Notas     string
	// Inicio e Fim da rodada (UTC, como o orquestrador anotou): é a janela a procurar no Cost
	// Explorer para preencher o custo.
	Inicio, Fim time.Time
}

// Veredito é o resumo em JSON para quem decide se sobe mais um patamar (infra/scripts/ensaio.ps1).
type Veredito struct {
	H1        bool              `json:"h1"`
	Porque    string            `json:"porque"`
	Quebra    *float64          `json:"quebra_mult"` // nil: não quebrou
	Patamares []VereditoPatamar `json:"patamares"`
}

type VereditoPatamar struct {
	Mult   float64  `json:"mult"`
	Rodada string   `json:"rodada"`
	Passou bool     `json:"passou"`
	Falhou []string `json:"falhou"`
}

func (e Ensaio) Veredito() Veredito {
	v := Veredito{Patamares: []VereditoPatamar{}}
	v.H1, v.Porque = e.H1()
	if q, ok := e.Quebra(); ok {
		v.Quebra = &q.K6.Mult
	}
	for _, a := range e.Avaliacoes {
		p := VereditoPatamar{Mult: a.K6.Mult, Rodada: a.K6.Rodada, Passou: a.Passou, Falhou: []string{}}
		for _, c := range a.Criterios {
			if !c.Passou {
				p.Falhou = append(p.Falhou, c.Nome)
			}
		}
		v.Patamares = append(v.Patamares, p)
	}
	return v
}

// Criterio é uma linha do veredito.
type Criterio struct {
	Nome   string
	Valor  string
	Alvo   string
	Passou bool
}

// Avaliacao é o patamar julgado contra o H1.
type Avaliacao struct {
	Patamar
	Duracao    time.Duration
	Perdidas   int // inscrições que o candidato viu e o banco não tem (+ pagamentos idem)
	Duplicadas int // cobrança repetida, pagamento aplicado duas vezes, dois confirmados, a mais sem reembolso
	Criterios  []Criterio
	Passou     bool
	CustoHora  float64 // USD/h; 0 sem custo informado
	CustoInsc  float64 // USD por inscrição enviada
}

func Avaliar(p Patamar, orcamentoHora float64) Avaliacao {
	a := Avaliacao{Patamar: p}
	a.Duracao, _ = time.ParseDuration(p.K6.Duracao)
	a.Perdidas = max(0, p.K6.InscricoesEnviadas-p.Auditoria.Enviadas) + max(0, p.K6.PagamentosVistos-p.Auditoria.Pagas)
	a.Duplicadas = p.Auditoria.CobrancasRepetidas + p.Auditoria.PagamentosAplicadosDuasVezes +
		p.Auditoria.ConfirmadasDuasVezes + p.Auditoria.PagoAMaisSemReembolso
	p95, temP95 := 0.0, p.K6.P95CriticoMs != nil
	if temP95 {
		p95 = *p.K6.P95CriticoMs
	}
	a.Criterios = []Criterio{
		// Sem inscrição nenhuma, "zero perdida" e "zero duplicada" passariam de graça: a rodada não
		// mediu nada (no ensaio local de validação, 05/11, foi uma faixa de CPF já usada).
		{"Inscrições enviadas na rodada", fmt.Sprint(p.K6.InscricoesEnviadas), "mais que zero", p.K6.InscricoesEnviadas > 0},
		{"Inscrições perdidas", fmt.Sprint(a.Perdidas), "zero", a.Perdidas == 0},
		{"Cobranças duplicadas", fmt.Sprint(a.Duplicadas), "zero", a.Duplicadas == 0},
		{"p95 das requisições críticas", ms(p95, temP95), "< 2 s", temP95 && p95 < AlvoP95Critico},
		{"Jornadas concluídas", pct(p.K6.JornadaConcluida), "> 98%", p.K6.JornadaConcluida > AlvoConcluidas},
		{"Problemas da auditoria", fmt.Sprint(len(p.Auditoria.Problemas)), "nenhum", len(p.Auditoria.Problemas) == 0},
		// Juntar o k6 de uma rodada com a auditoria de outra daria um relatório com números de
		// mentira; o patamar não vale.
		{"k6 e auditoria da mesma rodada", p.K6.Rodada + " · " + p.Auditoria.Rodada, "iguais",
			p.K6.Rodada != "" && p.K6.Rodada == p.Auditoria.Rodada},
	}
	if p.CustoUSD > 0 && a.Duracao > 0 {
		a.CustoHora = p.CustoUSD / a.Duracao.Hours()
		if p.K6.InscricoesEnviadas > 0 {
			a.CustoInsc = p.CustoUSD / float64(p.K6.InscricoesEnviadas)
		}
		if orcamentoHora > 0 {
			a.Criterios = append(a.Criterios, Criterio{"Custo por hora de pico", usd(a.CustoHora), "≤ " + usd(orcamentoHora),
				a.CustoHora <= orcamentoHora})
		}
	}
	a.Passou = true
	for _, c := range a.Criterios {
		a.Passou = a.Passou && c.Passou
	}
	return a
}

// Ensaio é o relatório inteiro.
type Ensaio struct {
	Titulo        string
	Ambiente      string
	OrcamentoHora float64 // USD/h previsto por patamar; 0 = não julgar custo
	Avaliacoes    []Avaliacao
}

func Montar(titulo, ambiente string, orcamentoHora float64, ps []Patamar) Ensaio {
	e := Ensaio{Titulo: titulo, Ambiente: ambiente, OrcamentoHora: orcamentoHora}
	for _, p := range ps {
		e.Avaliacoes = append(e.Avaliacoes, Avaliar(p, orcamentoHora))
	}
	slices.SortStableFunc(e.Avaliacoes, func(x, y Avaliacao) int {
		switch {
		case x.K6.Mult < y.K6.Mult:
			return -1
		case x.K6.Mult > y.K6.Mult:
			return 1
		}
		return 0
	})
	return e
}

// H1 diz se a hipótese passou: algum patamar de 40× ou mais, por 3 h ou mais, passou em tudo.
func (e Ensaio) H1() (passou bool, porque string) {
	var candidatos []Avaliacao
	for _, a := range e.Avaliacoes {
		if a.K6.Mult >= AlvoMult {
			candidatos = append(candidatos, a)
		}
	}
	if len(candidatos) == 0 {
		return false, fmt.Sprintf("nenhuma rodada chegou a %d×", AlvoMult)
	}
	for _, a := range candidatos {
		if a.Passou && a.Duracao >= AlvoDuracao {
			return true, fmt.Sprintf("a rodada %s (%g×, %s) passou em todos os critérios", a.K6.Rodada, a.K6.Mult, a.K6.Duracao)
		}
	}
	for _, a := range candidatos {
		if a.Passou {
			return false, fmt.Sprintf("a rodada %s passou a %g×, mas durou %s; o H1 pede %g h", a.K6.Rodada, a.K6.Mult, a.K6.Duracao, AlvoDuracao.Hours())
		}
	}
	return false, fmt.Sprintf("nenhuma rodada de %d× ou mais passou em todos os critérios", AlvoMult)
}

// Quebra é o primeiro patamar (do menor para o maior) que falhou em algum critério. ok = false
// quando nenhum falhou: aí o ensaio não chegou ao limite e é preciso subir mais.
func (e Ensaio) Quebra() (a Avaliacao, ok bool) {
	for _, a := range e.Avaliacoes {
		if !a.Passou {
			return a, true
		}
	}
	return a, false
}

func ms(v float64, tem bool) string {
	if !tem {
		return "sem medida"
	}
	if v >= 1000 {
		return strings.Replace(fmt.Sprintf("%.2f s", v/1000), ".", ",", 1)
	}
	return strings.Replace(fmt.Sprintf("%.1f ms", v), ".", ",", 1)
}

func seg(v float64) string { return strings.Replace(fmt.Sprintf("%.1f s", v), ".", ",", 1) }
func pct(v float64) string { return strings.Replace(fmt.Sprintf("%.2f%%", v*100), ".", ",", 1) }
func usd(v float64) string { return strings.Replace(fmt.Sprintf("US$ %.2f", v), ".", ",", 1) }
func milhar(n int) string {
	s := fmt.Sprint(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "." + s[i:]
	}
	return s
}

// Markdown escreve o relatório.
func (e Ensaio) Markdown() string {
	var b strings.Builder
	w := func(f string, a ...any) { fmt.Fprintf(&b, f, a...) }
	w("# %s\n\n", e.Titulo)
	if e.Ambiente != "" {
		w("**Ambiente:** %s\n\n", e.Ambiente)
	}
	passou, porque := e.H1()
	veredito := "**H1 NÃO PASSOU**"
	if passou {
		veredito = "**H1 PASSOU**"
	}
	w("> **H1** — o sistema absorve 40 vezes a carga média por 3 horas, sem perder inscrição, sem cobrar\n")
	w("> duas vezes e sem estourar o custo previsto de infraestrutura.\n\n")
	w("%s: %s.\n\n", veredito, porque)
	if q, ok := e.Quebra(); ok {
		var falhou []string
		for _, c := range q.Criterios {
			if !c.Passou {
				falhou = append(falhou, fmt.Sprintf("%s (%s; alvo %s)", strings.ToLower(c.Nome), c.Valor, c.Alvo))
			}
		}
		w("**Ponto de quebra:** %g× (rodada %s) — %s.\n\n", q.K6.Mult, q.K6.Rodada, strings.Join(falhou, "; "))
		if passou && q.K6.Mult < AlvoMult {
			w("⚠️ **A curva não é monótona:** falhou a %g× e passou a %d× ou mais. Uma carga menor não deveria "+
				"quebrar o que uma maior aguenta — a causa precisa ser explicada antes de afirmar o H1.\n\n", q.K6.Mult, AlvoMult)
		}
	} else if len(e.Avaliacoes) > 0 {
		w("**Ponto de quebra: não alcançado.** A maior carga ensaiada (%g×) passou em tudo. Um teste que só passa não "+
			"informa o limite: é preciso subir mais (04-escopo-poc §8).\n\n", e.Avaliacoes[len(e.Avaliacoes)-1].K6.Mult)
	}
	if e.OrcamentoHora == 0 {
		w("*Custo não julgado: o orçamento por hora não foi informado.*\n\n")
	}

	w("## A curva\n\n")
	w("| Patamar | Rodada | Duração | Inscrições enviadas | p95 crítico | Concluídas | Perdidas | Duplicadas | Falhas do k6 | Custo/h | Custo/inscrição | Resultado |\n")
	w("|---:|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---|\n")
	for _, a := range e.Avaliacoes {
		custoH, custoI := "—", "—"
		if a.CustoHora > 0 {
			custoH, custoI = usd(a.CustoHora), strings.Replace(fmt.Sprintf("US$ %.4f", a.CustoInsc), ".", ",", 1)
		}
		res := "passou"
		if !a.Passou {
			res = "**falhou**"
		}
		p95, tem := 0.0, a.K6.P95CriticoMs != nil
		if tem {
			p95 = *a.K6.P95CriticoMs
		}
		w("| %g× | %s | %s | %s | %s | %s | %d | %d | %d | %s | %s | %s |\n", a.K6.Mult, a.K6.Rodada, a.K6.Duracao,
			milhar(a.K6.InscricoesEnviadas), ms(p95, tem), pct(a.K6.JornadaConcluida), a.Perdidas, a.Duplicadas,
			a.K6.Falhas, custoH, custoI, res)
	}
	w("\n")

	for _, a := range e.Avaliacoes {
		w("## %g× — rodada %s\n\n", a.K6.Mult, a.K6.Rodada)
		w("| Critério | Medido | Alvo | |\n|---|---:|---|---|\n")
		for _, c := range a.Criterios {
			marca := "✅"
			if !c.Passou {
				marca = "❌"
			}
			w("| %s | %s | %s | %s |\n", c.Nome, c.Valor, c.Alvo, marca)
		}
		w("\n")
		w("- O k6 viu **%s inscrições enviadas** e **%s pagamentos**; o banco tem %s e %s.\n",
			milhar(a.K6.InscricoesEnviadas), milhar(a.K6.PagamentosVistos), milhar(a.Auditoria.Enviadas), milhar(a.Auditoria.Pagas))
		w("- %s requisições; p95 geral %s; recarregamentos %s; cliques duplos em \"Pagar\" %s.\n",
			milhar(a.K6.Requisicoes), ms(a.K6.P95Ms, true), milhar(a.K6.Recarregamentos), milhar(a.K6.CliquesDuplos))
		w("- p95 até o Pix aparecer %s; até o cartão %s; espera na sala de espera %s.\n",
			seg(a.K6.TempoAtePixP95), seg(a.K6.TempoAteCartaoP95), seg(a.K6.EsperaNaFilaP95))
		w("- Trilhas verificadas %s, quebradas %d; divergências da conciliação %d.\n",
			milhar(a.Auditoria.TrilhasVerificadas), len(a.Auditoria.TrilhasQuebradas), a.Auditoria.Divergencias)
		// Não reprova (o candidato pagou duas vezes e o dinheiro volta), mas quem lê precisa ver: com
		// boleto acontece na vida real; com Pix, só se o prestador aceitar pagar duas vezes.
		if a.Auditoria.ReembolsosPendentes > 0 {
			w("- Pagos duas vezes pelo candidato, com reembolso marcado: %s.\n", milhar(a.Auditoria.ReembolsosPendentes))
		}
		if d := a.Auditoria.Devolucoes; len(d) > 0 {
			// A devolução é assíncrona: pendente no fim da rodada não é falha; recusada é, e já entra
			// nos problemas da auditoria.
			w("- Devoluções: %s concluídas, %s pendentes, %s aguardando chave Pix, %s recusadas.\n",
				milhar(d["CONCLUIDA"]), milhar(d["PENDENTE"]), milhar(d["AGUARDANDO_CHAVE"]), milhar(d["RECUSADA"]))
		}
		if !a.Inicio.IsZero() && !a.Fim.IsZero() {
			w("- Janela da rodada: %s a %s (UTC)", a.Inicio.UTC().Format("02/01/2006 15:04"), a.Fim.UTC().Format("15:04"))
			if a.CustoUSD == 0 {
				w(" — o custo deste intervalo, no Cost Explorer, vai em `custo_usd` do manifesto")
			}
			w(".\n")
		}
		if len(a.Auditoria.Problemas) > 0 {
			w("- **Problemas da auditoria:**\n")
			for _, p := range a.Auditoria.Problemas {
				w("  - %s\n", p)
			}
		}
		if a.Notas != "" {
			w("- %s\n", a.Notas)
		}
		w("\n")
	}
	return b.String()
}
