// Package cronograma confere as dependências entre as fases do certame e faz a retificação de datas
// (02-produto §7 A: "mudou uma data, o sistema mostra o que quebra"). As regras tornam explícito o
// que o resto do sistema já supõe: a homologação não pode abrir antes de o boleto pago no último dia
// de pagamento ser informado, ou a conciliação e a homologação se contradizem.
//
// A retificação é uma versão nova do edital, com o antes, o depois e o motivo: a versão antiga não
// muda, e toda decisão da trilha continua dizendo sob qual versão foi tomada.
package cronograma

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"lisura/internal/pagamento"
	"lisura/internal/publicacao"
)

// As fases, na ordem do ciclo de vida do certame (02-produto §2).
var Fases = []string{"isencao", "inscricao", "pagamento", "homologacao", "ensalamento", "aplicacao", "gabarito", "resultado"}

// Folgas das regras. Saem do edital num certame real; aqui, valores usuais.
var (
	ParaPagarDepoisDaIsencao = 5 * 24 * time.Hour // quem tem a isenção negada precisa de tempo para pagar
	AntecedenciaDoLocal      = 7 * 24 * time.Hour // o cartão com o local sai pelo menos uma semana antes da prova
	PrazoMinimoDeRecurso     = 2 * 24 * time.Hour // a janela de recurso contra o gabarito
)

type Janela struct {
	AbreEm  time.Time `json:"abre_em"`
	FechaEm time.Time `json:"fecha_em"`
}

// Cronograma é a janela de cada fase; a fase que o certame não tem não aparece.
type Cronograma map[string]Janela

// Quebra é uma regra que o cronograma não cumpre.
type Quebra struct {
	Regra   string   `json:"regra"`
	Fases   []string `json:"fases"`
	Detalhe string   `json:"detalhe"`
}

type regra struct {
	codigo string
	a, b   string // as duas fases; a regra só vale se o certame tiver as duas
	// confere devolve "" se a regra se cumpre, ou o que está errado.
	confere func(a, b Janela) string
}

func data(t time.Time) string { return t.Format("02/01/2006 15:04") }

func dias(d time.Duration) string { return fmt.Sprintf("%d dias", int(d.Hours()/24)) }

var regras = []regra{
	{"ISENCAO_SEM_TEMPO_PARA_PAGAR", "isencao", "pagamento", func(i, p Janela) string {
		if limite := p.FechaEm.Add(-ParaPagarDepoisDaIsencao); i.FechaEm.After(limite) {
			return fmt.Sprintf("a isenção fecha em %s; tem de fechar até %s (%s antes do fim do pagamento), para quem tiver o pedido negado poder pagar",
				data(i.FechaEm), data(limite), dias(ParaPagarDepoisDaIsencao))
		}
		return ""
	}},
	{"PAGAMENTO_NAO_COBRE_A_INSCRICAO", "inscricao", "pagamento", func(i, p Janela) string {
		if p.FechaEm.Before(i.FechaEm) {
			return fmt.Sprintf("o pagamento fecha em %s, antes do fim da inscrição (%s): quem finalizar no fim não consegue pagar",
				data(p.FechaEm), data(i.FechaEm))
		}
		if p.AbreEm.After(i.FechaEm) {
			return fmt.Sprintf("o pagamento só abre em %s, depois do fim da inscrição (%s)", data(p.AbreEm), data(i.FechaEm))
		}
		return ""
	}},
	{"HOMOLOGACAO_ANTES_DA_COMPENSACAO", "pagamento", "homologacao", func(p, h Janela) string {
		if limite := p.FechaEm.Add(pagamento.MargemCompensacao); h.AbreEm.Before(limite) {
			return fmt.Sprintf("a homologação abre em %s; o boleto pago no último dia pode ser informado até %s (margem de compensação de %s)",
				data(h.AbreEm), data(limite), dias(pagamento.MargemCompensacao))
		}
		return ""
	}},
	{"ENSALAMENTO_ANTES_DA_HOMOLOGACAO", "homologacao", "ensalamento", func(h, e Janela) string {
		if e.AbreEm.Before(h.FechaEm) {
			return fmt.Sprintf("o ensalamento abre em %s, antes do fim da homologação (%s): só a inscrição homologada ganha sala",
				data(e.AbreEm), data(h.FechaEm))
		}
		return ""
	}},
	{"LOCAL_SEM_ANTECEDENCIA", "ensalamento", "aplicacao", func(e, a Janela) string {
		if limite := e.FechaEm.Add(AntecedenciaDoLocal); a.AbreEm.Before(limite) {
			return fmt.Sprintf("a prova é em %s; o ensalamento fecha em %s, e o local tem de sair %s antes",
				data(a.AbreEm), data(e.FechaEm), dias(AntecedenciaDoLocal))
		}
		return ""
	}},
	{"RECURSO_ANTES_DA_PROVA", "aplicacao", "gabarito", func(a, g Janela) string {
		if g.AbreEm.Before(a.FechaEm) {
			return fmt.Sprintf("o recurso contra o gabarito abre em %s, antes do fim da prova (%s)", data(g.AbreEm), data(a.FechaEm))
		}
		return ""
	}},
	{"RECURSO_CURTO", "gabarito", "gabarito", func(g, _ Janela) string {
		if d := g.FechaEm.Sub(g.AbreEm); d < PrazoMinimoDeRecurso {
			return fmt.Sprintf("a janela de recurso tem %s; o mínimo é %s", d.Round(time.Hour), dias(PrazoMinimoDeRecurso))
		}
		return ""
	}},
	{"RESULTADO_ANTES_DO_RECURSO", "gabarito", "resultado", func(g, r Janela) string {
		if r.AbreEm.Before(g.FechaEm) {
			return fmt.Sprintf("o resultado abre em %s, antes do fim do recurso contra o gabarito (%s)", data(r.AbreEm), data(g.FechaEm))
		}
		return ""
	}},
}

// Conferir devolve as regras que o cronograma não cumpre, na ordem do ciclo.
func (c Cronograma) Conferir() []Quebra {
	out := []Quebra{}
	for _, f := range Fases {
		if j, ok := c[f]; ok && !j.AbreEm.Before(j.FechaEm) {
			out = append(out, Quebra{"FASE_FECHA_ANTES_DE_ABRIR", []string{f}, fmt.Sprintf("%s abre em %s e fecha em %s", f, data(j.AbreEm), data(j.FechaEm))})
		}
	}
	for _, r := range regras {
		a, okA := c[r.a]
		b, okB := c[r.b]
		if !okA || !okB {
			continue
		}
		if d := r.confere(a, b); d != "" {
			fases := []string{r.a}
			if r.b != r.a {
				fases = append(fases, r.b)
			}
			out = append(out, Quebra{r.codigo, fases, d})
		}
	}
	return out
}

// Ler devolve o cronograma do certame.
func Ler(ctx context.Context, q interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, certameID int64) (Cronograma, error) {
	rows, err := q.Query(ctx, `SELECT fase, abre_em, fecha_em FROM certame_fase WHERE certame_id = $1`, certameID)
	if err != nil {
		return nil, err
	}
	c := Cronograma{}
	var fase string
	var j Janela
	_, err = pgx.ForEachRow(rows, []any{&fase, &j.AbreEm, &j.FechaEm}, func() error { c[fase] = j; return nil })
	return c, err
}

var (
	ErrFaseDesconhecida   = errors.New("fase desconhecida")
	ErrFaseEncerrada      = errors.New("fase já encerrada não muda")
	ErrInicioDeFaseAberta = errors.New("o início de uma fase já aberta não muda")
	ErrFimNoPassado       = errors.New("o novo fim já passou")
	ErrSemMotivo          = errors.New("a retificação precisa de motivo")
)

// ErrQuebra: a retificação deixaria o cronograma quebrado; nada foi gravado.
type ErrQuebra struct{ Quebras []Quebra }

func (e *ErrQuebra) Error() string {
	var d []string
	for _, q := range e.Quebras {
		d = append(d, q.Regra+": "+q.Detalhe)
	}
	return "a retificação quebra o cronograma: " + strings.Join(d, "; ")
}

// Retificacao é o pedido: a fase e a janela nova. Abre ou Fecha zero mantém o que está.
type Retificacao struct {
	Fase         string
	Abre, Fecha  time.Time
	Motivo, Quem string
}

// Efeito é o resultado (ou a simulação) de uma retificação.
type Efeito struct {
	Antes, Depois Janela         `json:"-"`
	Versao        int            `json:"edital_versao"` // a versão nova; 0 na simulação
	Quebras       []Quebra       `json:"quebras"`       // o que a mudança quebraria (recusa a retificação)
	JaQuebradas   []Quebra       `json:"ja_quebradas"`  // o que já estava quebrado antes dela (não recusa)
	Afetados      map[string]int `json:"afetados"`      // quem a mudança alcança, por situação
	Resumo        string         `json:"resumo"`
}

// Simular diz o que a retificação faria, sem gravar nada.
func Simular(ctx context.Context, pool *pgxpool.Pool, certameID int64, r Retificacao) (Efeito, error) {
	var e Efeito
	err := pgx.BeginTxFunc(ctx, pool, pgx.TxOptions{AccessMode: pgx.ReadOnly}, func(tx pgx.Tx) error {
		var err error
		e, err = calcular(ctx, tx, certameID, r)
		return err
	})
	return e, err
}

// Retificar grava a retificação como uma versão nova do edital, se ela não quebrar o cronograma.
func Retificar(ctx context.Context, pool *pgxpool.Pool, certameID int64, r Retificacao) (Efeito, error) {
	var e Efeito
	if strings.TrimSpace(r.Motivo) == "" || r.Quem == "" {
		return e, ErrSemMotivo
	}
	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		// Duas retificações ao mesmo tempo no mesmo certame se enfileiram: a segunda confere o
		// cronograma já com a primeira.
		if _, err := tx.Exec(ctx, `SELECT 1 FROM certame WHERE id = $1 FOR UPDATE`, certameID); err != nil {
			return err
		}
		var err error
		if e, err = calcular(ctx, tx, certameID, r); err != nil {
			return err
		}
		if len(e.Quebras) > 0 {
			return &ErrQuebra{e.Quebras}
		}
		if _, err := tx.Exec(ctx, `UPDATE certame_fase SET abre_em = $3, fecha_em = $4 WHERE certame_id = $1 AND fase = $2`,
			certameID, r.Fase, e.Depois.AbreEm, e.Depois.FechaEm); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, `
			INSERT INTO edital_versao (certame_id, versao, publicado_em, resumo)
			SELECT $1, coalesce(max(versao), 0) + 1, now(), $2 FROM edital_versao WHERE certame_id = $1
			RETURNING versao`, certameID, e.Resumo+" Motivo: "+r.Motivo+". Por: "+r.Quem+".").Scan(&e.Versao); err != nil {
			return err
		}
		return publicacao.EnfileirarCertame(ctx, tx, certameID) // o cronograma público muda
	})
	return e, err
}

func calcular(ctx context.Context, tx pgx.Tx, certameID int64, r Retificacao) (Efeito, error) {
	e := Efeito{Afetados: map[string]int{}}
	if !slices.Contains(Fases, r.Fase) {
		return e, fmt.Errorf("%w: %q", ErrFaseDesconhecida, r.Fase)
	}
	c, err := Ler(ctx, tx, certameID)
	if err != nil {
		return e, err
	}
	antes, existe := c[r.Fase]
	if !existe {
		return e, fmt.Errorf("%w: o certame não tem a fase %s", ErrFaseDesconhecida, r.Fase)
	}
	var agora time.Time
	if err := tx.QueryRow(ctx, `SELECT now()`).Scan(&agora); err != nil {
		return e, err
	}
	depois := antes
	if !r.Abre.IsZero() {
		depois.AbreEm = r.Abre
	}
	if !r.Fecha.IsZero() {
		depois.FechaEm = r.Fecha
	}
	switch {
	case !antes.FechaEm.After(agora):
		return e, fmt.Errorf("%w: %s fechou em %s", ErrFaseEncerrada, r.Fase, data(antes.FechaEm))
	case !antes.AbreEm.After(agora) && !depois.AbreEm.Equal(antes.AbreEm):
		return e, fmt.Errorf("%w: %s abriu em %s", ErrInicioDeFaseAberta, r.Fase, data(antes.AbreEm))
	case !depois.FechaEm.After(agora):
		return e, fmt.Errorf("%w: %s", ErrFimNoPassado, data(depois.FechaEm))
	}
	// Só recusa o que a mudança quebra. O que já estava quebrado (um cronograma antigo, uma regra nova)
	// é mostrado, mas não impede corrigir outra coisa.
	ja := map[string]bool{}
	e.JaQuebradas = c.Conferir()
	for _, q := range e.JaQuebradas {
		ja[q.Regra+"|"+strings.Join(q.Fases, ",")] = true
	}
	c[r.Fase] = depois
	e.Antes, e.Depois, e.Quebras = antes, depois, []Quebra{}
	for _, q := range c.Conferir() {
		if !ja[q.Regra+"|"+strings.Join(q.Fases, ",")] {
			e.Quebras = append(e.Quebras, q)
		}
	}
	e.Resumo = fmt.Sprintf("Retificação do cronograma: %s de %s–%s para %s–%s.", r.Fase,
		data(antes.AbreEm), data(antes.FechaEm), data(depois.AbreEm), data(depois.FechaEm))

	// Quem a mudança alcança: a inscrição aberta alcança os rascunhos; o pagamento, quem ainda não
	// pagou; as fases depois da homologação, quem já está na lista.
	alcance := map[string][]string{
		"isencao":     {"RASCUNHO"},
		"inscricao":   {"RASCUNHO"},
		"pagamento":   {"AGUARDANDO_PAGAMENTO"},
		"homologacao": {"PAGA", "ISENCAO_ATIVA", "EM_ANALISE", "DEFERIDA"},
		"ensalamento": {"HOMOLOGADA"}, "aplicacao": {"HOMOLOGADA"}, "gabarito": {"HOMOLOGADA"}, "resultado": {"HOMOLOGADA"},
	}[r.Fase]
	rows, err := tx.Query(ctx, `SELECT estado, count(*) FROM inscricao WHERE certame_id = $1 AND estado = ANY($2) GROUP BY estado`,
		certameID, alcance)
	if err != nil {
		return e, err
	}
	var estado string
	var n int
	_, err = pgx.ForEachRow(rows, []any{&estado, &n}, func() error { e.Afetados[estado] = n; return nil })
	return e, err
}
