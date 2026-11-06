package cronograma

import (
	"context"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"lisura/internal/conta"
	"lisura/internal/dbtest"
	"lisura/internal/inscricao"
	"lisura/internal/seed"
)

var pool *pgxpool.Pool

func TestMain(m *testing.M) {
	p, err := dbtest.Pool(context.Background(), "t_cronograma")
	if err != nil {
		dbtest.SemBanco(err)
	}
	pool = p
	code := m.Run()
	if pool != nil {
		pool.Close()
	}
	os.Exit(code)
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

const dia = 24 * time.Hour

// valido é um cronograma de PSS que cumpre todas as regras, a partir de t0.
func valido(t0 time.Time) Cronograma {
	j := func(de, ate int) Janela {
		return Janela{t0.Add(time.Duration(de) * dia), t0.Add(time.Duration(ate) * dia)}
	}
	return Cronograma{
		"isencao": j(0, 5), "inscricao": j(0, 30), "pagamento": j(0, 32), "homologacao": j(36, 40),
		"ensalamento": j(40, 45), "aplicacao": j(55, 56), "gabarito": j(56, 59), "resultado": j(62, 63),
	}
}

func regrasDe(qs []Quebra) []string {
	var out []string
	for _, q := range qs {
		out = append(out, q.Regra)
	}
	return out
}

func TestCronogramaValidoNaoQuebra(t *testing.T) {
	if q := valido(time.Now()).Conferir(); len(q) != 0 {
		t.Fatalf("o cronograma válido quebrou: %+v", q)
	}
}

// Cada regra, quebrada sozinha, é acusada pelo nome, e só ela.
func TestCadaRegraQuebraSozinha(t *testing.T) {
	t0 := time.Date(2027, 2, 1, 9, 0, 0, 0, time.UTC)
	mudar := func(fase string, de, ate int) func(Cronograma) {
		return func(c Cronograma) {
			c[fase] = Janela{t0.Add(time.Duration(de) * dia), t0.Add(time.Duration(ate) * dia)}
		}
	}
	for regra, estragar := range map[string]func(Cronograma){
		"ISENCAO_SEM_TEMPO_PARA_PAGAR":     mudar("isencao", 0, 28),      // fecha 4 dias antes do fim do pagamento
		"PAGAMENTO_NAO_COBRE_A_INSCRICAO":  mudar("pagamento", 0, 29),    // antes do fim da inscrição
		"HOMOLOGACAO_ANTES_DA_COMPENSACAO": mudar("homologacao", 34, 40), // 2 dias depois do pagamento; a margem é 3
		"ENSALAMENTO_ANTES_DA_HOMOLOGACAO": mudar("ensalamento", 39, 45),
		"LOCAL_SEM_ANTECEDENCIA":           mudar("aplicacao", 51, 52), // 6 dias depois do ensalamento
		"RECURSO_ANTES_DA_PROVA":           func(c Cronograma) { c["gabarito"] = Janela{t0.Add(55*dia + 12*time.Hour), t0.Add(59 * dia)} },
		"RECURSO_CURTO":                    func(c Cronograma) { c["gabarito"] = Janela{t0.Add(56 * dia), t0.Add(57*dia + 12*time.Hour)} },
		"RESULTADO_ANTES_DO_RECURSO":       mudar("resultado", 58, 63),
		"FASE_FECHA_ANTES_DE_ABRIR":        mudar("resultado", 63, 62),
	} {
		c := valido(t0)
		estragar(c)
		if got := regrasDe(c.Conferir()); !slices.Equal(got, []string{regra}) {
			t.Errorf("%s: acusou %v", regra, got)
		}
	}
}

// A regra só vale quando o certame tem as duas fases: o PSS sem homologação no cronograma não quebra.
func TestFaseQueNaoExisteNaoQuebra(t *testing.T) {
	c := valido(time.Now())
	delete(c, "homologacao")
	// O ensalamento logo depois do pagamento (sem homologação, nada a conferir ali) e até o dia 50: só
	// a antecedência do local, que vale entre ensalamento e prova, quebra.
	c["ensalamento"] = Janela{c["pagamento"].FechaEm, c["pagamento"].FechaEm.Add(18 * dia)}
	if q := c.Conferir(); len(q) != 1 || q[0].Regra != "LOCAL_SEM_ANTECEDENCIA" {
		t.Fatalf("quebras: %+v", q)
	}
}

func certame(t *testing.T) int64 {
	t.Helper()
	if pool == nil {
		t.Skip("sem banco de teste")
	}
	c, err := seed.CriarCertame(context.Background(), pool, seed.Padrao())
	must(t, err)
	return c.ID
}

func janela(t *testing.T, cert int64, fase string) Janela {
	t.Helper()
	c, err := Ler(context.Background(), pool, cert)
	must(t, err)
	return c[fase]
}

func versao(t *testing.T, cert int64) (v int, resumo string) {
	t.Helper()
	must(t, pool.QueryRow(context.Background(), `SELECT versao, resumo FROM edital_versao WHERE certame_id = $1 ORDER BY versao DESC LIMIT 1`,
		cert).Scan(&v, &resumo))
	return
}

// Prorrogar a inscrição em um dia (o pagamento, que vai até o dia 32, ainda cobre): grava a janela
// nova, publica a versão 2 do edital com o antes, o depois e o motivo, e diz quem a mudança alcança.
func TestRetificarProrrogaEPublicaVersao(t *testing.T) {
	ctx := context.Background()
	cert := certame(t)
	cand, err := conta.Criar(ctx, pool, conta.GerarCPF(time.Now().UnixNano()%1_000_000_000), "Rascunho", "r@exemplo.test")
	must(t, err)
	var cargo int64
	must(t, pool.QueryRow(ctx, `SELECT min(id) FROM cargo WHERE certame_id = $1`, cert).Scan(&cargo))
	_, _, err = inscricao.Abrir(ctx, pool, cert, cand, cargo, inscricao.Ator{Tipo: inscricao.Candidato, ID: "t"})
	must(t, err)

	nova := janela(t, cert, "inscricao").FechaEm.Add(dia)
	e, err := Retificar(ctx, pool, cert, Retificacao{Fase: "inscricao", Fecha: nova, Motivo: "PRORROGACAO_POR_INSTABILIDADE", Quem: "ana"})
	must(t, err)
	if e.Versao != 2 || len(e.Quebras) != 0 || e.Afetados["RASCUNHO"] != 1 {
		t.Fatalf("efeito: %+v", e)
	}
	if got := janela(t, cert, "inscricao").FechaEm; !got.Equal(nova) {
		t.Fatalf("a inscrição fecha em %s; esperado %s", got, nova)
	}
	if v, resumo := versao(t, cert); v != 2 || !strings.Contains(resumo, "inscricao") || !strings.Contains(resumo, "PRORROGACAO_POR_INSTABILIDADE") {
		t.Fatalf("versão %d: %q", v, resumo)
	}
}

// Prorrogar a inscrição para depois do fim do pagamento: recusado, com a regra que quebra, e nada
// gravado. A simulação mostra o mesmo sem gravar.
func TestRetificacaoQueQuebraNaoGrava(t *testing.T) {
	ctx := context.Background()
	cert := certame(t)
	antes := janela(t, cert, "inscricao")
	r := Retificacao{Fase: "inscricao", Fecha: janela(t, cert, "pagamento").FechaEm.Add(dia), Motivo: "PRORROGACAO", Quem: "ana"}

	s, err := Simular(ctx, pool, cert, r)
	must(t, err)
	if got := regrasDe(s.Quebras); !slices.Equal(got, []string{"PAGAMENTO_NAO_COBRE_A_INSCRICAO"}) || s.Versao != 0 {
		t.Fatalf("simulação: %+v", s)
	}
	_, err = Retificar(ctx, pool, cert, r)
	var q *ErrQuebra
	if !errors.As(err, &q) || q.Quebras[0].Regra != "PAGAMENTO_NAO_COBRE_A_INSCRICAO" {
		t.Fatalf("erro %v", err)
	}
	if !janela(t, cert, "inscricao").FechaEm.Equal(antes.FechaEm) {
		t.Fatal("a retificação recusada mudou a janela")
	}
	if v, _ := versao(t, cert); v != 1 {
		t.Fatalf("a retificação recusada publicou a versão %d", v)
	}
}

// O que já estava quebrado antes não impede corrigir outra coisa: aparece em ja_quebradas.
func TestJaQuebradoNaoImpedeOutraRetificacao(t *testing.T) {
	ctx := context.Background()
	cert := certame(t)
	_, err := pool.Exec(ctx, `UPDATE certame_fase SET fecha_em = now() + interval '29 days' WHERE certame_id = $1 AND fase = 'pagamento'`, cert)
	must(t, err)
	e, err := Retificar(ctx, pool, cert, Retificacao{Fase: "isencao", Fecha: janela(t, cert, "isencao").FechaEm.Add(-dia),
		Motivo: "ANTECIPACAO_DA_ISENCAO", Quem: "ana"})
	must(t, err)
	if len(e.Quebras) != 0 || !slices.Equal(regrasDe(e.JaQuebradas), []string{"PAGAMENTO_NAO_COBRE_A_INSCRICAO"}) {
		t.Fatalf("efeito: %+v", e)
	}
}

// A fase encerrada não muda; o início da fase aberta não muda; o fim não vai para o passado; sem
// motivo não há retificação.
func TestLimitesDaRetificacao(t *testing.T) {
	ctx := context.Background()
	cert := certame(t)
	_, err := pool.Exec(ctx, `UPDATE certame_fase SET abre_em = now() - interval '10 days', fecha_em = now() - interval '1 day'
		WHERE certame_id = $1 AND fase = 'isencao'`, cert)
	must(t, err)
	agora := time.Now()
	for nome, c := range map[string]struct {
		r    Retificacao
		quer error
	}{
		"fase encerrada":             {Retificacao{Fase: "isencao", Fecha: agora.Add(dia)}, ErrFaseEncerrada},
		"início da fase aberta":      {Retificacao{Fase: "inscricao", Abre: agora.Add(dia)}, ErrInicioDeFaseAberta},
		"fim no passado":             {Retificacao{Fase: "inscricao", Fecha: agora.Add(-time.Hour)}, ErrFimNoPassado},
		"fase que o certame não tem": {Retificacao{Fase: "aplicacao", Fecha: agora.Add(dia)}, ErrFaseDesconhecida},
		"fase inventada":             {Retificacao{Fase: "vestibular", Fecha: agora.Add(dia)}, ErrFaseDesconhecida},
	} {
		c.r.Motivo, c.r.Quem = "TESTE", "ana"
		if _, err := Retificar(ctx, pool, cert, c.r); !errors.Is(err, c.quer) {
			t.Errorf("%s: %v; esperado %v", nome, err, c.quer)
		}
	}
	if _, err := Retificar(ctx, pool, cert, Retificacao{Fase: "inscricao", Fecha: agora.Add(40 * dia), Quem: "ana"}); !errors.Is(err, ErrSemMotivo) {
		t.Fatalf("sem motivo: %v", err)
	}
	if v, _ := versao(t, cert); v != 1 {
		t.Fatalf("as recusadas publicaram a versão %d", v)
	}
}
