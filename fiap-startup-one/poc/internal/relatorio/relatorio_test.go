package relatorio

import (
	"strings"
	"testing"
)

func p95(v float64) *float64 { return &v }

// rodada limpa no patamar pedido: tudo no alvo, auditoria batendo com o k6.
func limpa(mult float64, rodada, duracao string) Patamar {
	return Patamar{
		K6: K6{Rodada: rodada, Mult: mult, Duracao: duracao, InscricoesEnviadas: 1000, PagamentosVistos: 900,
			JornadaConcluida: 0.995, P95CriticoMs: p95(180)},
		Auditoria: Auditoria{Rodada: rodada, Enviadas: 1000, Pagas: 900},
	}
}

func TestLerK6DoLogEDoArquivo(t *testing.T) {
	log := "time=... level=info msg=x\n\nRESUMO_DO_ENSAIO {\"rodada\":\"401001\",\"mult\":40,\"duracao\":\"3h\"," +
		"\"inscricoes_enviadas\":104871,\"jornada_concluida\":0.991,\"p95_critico_ms\":412.5}\nPARA_AUDITAR lisura-auditoria ...\n"
	k, err := LerK6([]byte(log))
	if err != nil || k.Rodada != "401001" || k.Mult != 40 || k.InscricoesEnviadas != 104871 || *k.P95CriticoMs != 412.5 {
		t.Fatalf("do log: %+v %v", k, err)
	}
	k, err = LerK6([]byte(`{"resumo":{"rodada":"7","mult":10,"p95_critico_ms":null},"metricas":{}}`))
	if err != nil || k.Rodada != "7" || k.P95CriticoMs != nil {
		t.Fatalf("do arquivo SAIDA: %+v %v", k, err)
	}
	// Do CloudWatch em texto: mensagens emendadas na mesma linha, separadas por tabulação.
	k, err = LerK6([]byte("x\tRESUMO_DO_ENSAIO {\"rodada\":\"9\",\"mult\":80}\tPARA_AUDITAR lisura-auditoria -rodada 9\n"))
	if err != nil || k.Rodada != "9" || k.Mult != 80 {
		t.Fatalf("de linha emendada: %+v %v", k, err)
	}
	if _, err := LerK6([]byte("só log, sem resumo")); err == nil {
		t.Fatal("log sem resumo aceito")
	}
}

func TestLerAuditoriaNoMeioDoLog(t *testing.T) {
	log := "2026/11/05 12:00:00 conectando\n{\n  \"rodada\": \"401001\",\n  \"inscricoes_enviadas\": 104871,\n" +
		"  \"problemas\": [\"COBRANÇA DUPLICADA: 1 inscrições\"]\n}\n2026/11/05 12:00:09 fim\n"
	a, err := LerAuditoria([]byte(log))
	if err != nil || a.Rodada != "401001" || a.Enviadas != 104871 || len(a.Problemas) != 1 {
		t.Fatalf("%+v %v", a, err)
	}
}

func TestH1PassaSoCom40xPor3Horas(t *testing.T) {
	e := Montar("t", "", 0, []Patamar{limpa(10, "1", "3h"), limpa(20, "2", "3h"), limpa(40, "3", "3h")})
	if ok, porque := e.H1(); !ok {
		t.Fatalf("40× por 3 h limpo não passou: %s", porque)
	}
	if _, quebrou := e.Quebra(); quebrou {
		t.Fatal("quebra acusada sem falha")
	}
	if !strings.Contains(e.Markdown(), "Ponto de quebra: não alcançado") {
		t.Fatal("o relatório não avisa que é preciso subir mais")
	}

	curta := Montar("t", "", 0, []Patamar{limpa(40, "3", "2m")})
	if ok, porque := curta.H1(); ok || !strings.Contains(porque, "durou 2m") {
		t.Fatalf("40× por 2 min não pode passar no H1: %v %s", ok, porque)
	}
	if ok, _ := Montar("t", "", 0, []Patamar{limpa(20, "2", "3h")}).H1(); ok {
		t.Fatal("H1 passou sem rodada de 40×")
	}
}

// Cada critério, sozinho, derruba o patamar.
func TestCadaCriterioDerruba(t *testing.T) {
	casos := map[string]func(*Patamar){
		"rodada vazia": func(p *Patamar) {
			p.K6.InscricoesEnviadas, p.K6.PagamentosVistos, p.Auditoria.Enviadas, p.Auditoria.Pagas = 0, 0, 0, 0
		},
		"inscrição perdida":         func(p *Patamar) { p.Auditoria.Enviadas = 999 },
		"pagamento perdido":         func(p *Patamar) { p.Auditoria.Pagas = 899 },
		"cobrança duplicada":        func(p *Patamar) { p.Auditoria.CobrancasRepetidas = 1 },
		"pagamento duas vezes":      func(p *Patamar) { p.Auditoria.PagamentosAplicadosDuasVezes = 1 },
		"confirmada duas vezes":     func(p *Patamar) { p.Auditoria.ConfirmadasDuasVezes = 1 },
		"a mais sem reembolso":      func(p *Patamar) { p.Auditoria.PagoAMaisSemReembolso = 1 },
		"p95 de 2 s":                func(p *Patamar) { p.K6.P95CriticoMs = p95(2000) },
		"p95 sem medida":            func(p *Patamar) { p.K6.P95CriticoMs = nil },
		"98% concluídas":            func(p *Patamar) { p.K6.JornadaConcluida = 0.98 },
		"problema na auditoria":     func(p *Patamar) { p.Auditoria.Problemas = []string{"TRILHA ADULTERADA"} },
		"auditoria de outra rodada": func(p *Patamar) { p.Auditoria.Rodada = "outra" },
		"custo acima do previsto":   func(p *Patamar) { p.CustoUSD = 15 }, // 15 / 3 h = 5/h > 4,44
	}
	for nome, estragar := range casos {
		p := limpa(40, "3", "3h")
		estragar(&p)
		a := Avaliar(p, 4.44)
		if a.Passou {
			t.Errorf("%s: o patamar passou", nome)
		}
		if ok, _ := Montar("t", "", 4.44, []Patamar{p}).H1(); ok {
			t.Errorf("%s: o H1 passou", nome)
		}
	}
	if a := Avaliar(limpa(40, "3", "3h"), 4.44); !a.Passou {
		t.Fatalf("o patamar limpo não passou: %+v", a.Criterios)
	}
}

// O ponto de quebra é o menor patamar que falhou, não o primeiro da lista nem o maior.
func TestQuebraEOMenorPatamarQueFalhou(t *testing.T) {
	ruim20 := limpa(20, "2", "3h")
	ruim20.K6.P95CriticoMs = p95(4672)
	ruim80 := limpa(80, "5", "3h")
	ruim80.Auditoria.CobrancasRepetidas = 3
	e := Montar("t", "", 0, []Patamar{ruim80, limpa(40, "3", "3h"), ruim20, limpa(10, "1", "3h")})
	q, ok := e.Quebra()
	if !ok || q.K6.Mult != 20 {
		t.Fatalf("quebra em %v×, esperado 20×", q.K6.Mult)
	}
	md := e.Markdown()
	if !strings.Contains(md, "**Ponto de quebra:** 20× (rodada 2) — p95 das requisições críticas (4,67 s; alvo < 2 s)") {
		t.Fatalf("relatório sem o ponto de quebra certo:\n%s", md)
	}
	// 40× passou: o H1 passa mesmo com a quebra a 20× (a curva não é monótona e o relatório diz isso).
	if ok, _ := e.H1(); !ok {
		t.Fatal("H1 devia passar pela rodada de 40×")
	}
	if !strings.Contains(md, "A curva não é monótona") {
		t.Fatal("o relatório não avisa que falhou abaixo e passou acima")
	}
	// O mesmo julgamento, em JSON, para o orquestrador decidir se sobe mais.
	v := e.Veredito()
	if !v.H1 || v.Quebra == nil || *v.Quebra != 20 || len(v.Patamares) != 4 ||
		v.Patamares[1].Passou || len(v.Patamares[1].Falhou) != 1 || v.Patamares[1].Falhou[0] != "p95 das requisições críticas" ||
		!v.Patamares[2].Passou || v.Patamares[3].Falhou[0] != "Cobranças duplicadas" {
		t.Fatalf("veredito %+v", v)
	}
	if i, j := strings.Index(md, "| 10× |"), strings.Index(md, "| 80× |"); i < 0 || j < i {
		t.Fatal("a curva não está em ordem de patamar")
	}
}

func TestCustoPorHoraEPorInscricao(t *testing.T) {
	p := limpa(40, "3", "3h")
	p.CustoUSD = 12
	a := Avaliar(p, 4.44)
	if a.CustoHora != 4 || a.CustoInsc != 0.012 || !a.Passou {
		t.Fatalf("custo/h %v, custo/inscrição %v, passou %v", a.CustoHora, a.CustoInsc, a.Passou)
	}
	if md := Montar("t", "", 4.44, []Patamar{p}).Markdown(); !strings.Contains(md, "US$ 4,00") || !strings.Contains(md, "US$ 0,0120") {
		t.Fatalf("custo fora do relatório:\n%s", md)
	}
	if md := Montar("t", "", 0, []Patamar{p}).Markdown(); !strings.Contains(md, "Custo não julgado") {
		t.Fatal("sem orçamento, o relatório devia dizer que não julgou custo")
	}
}
