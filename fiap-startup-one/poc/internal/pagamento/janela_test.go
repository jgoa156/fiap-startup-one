package pagamento

import (
	"testing"
	"time"
)

// Um pagamento de qualquer idade até a margem do boleto mais um fim de semana entra numa conciliação
// em tempo limitado, comece o relógio onde começar; e o minuto típico baixa só 10 min de extrato.
func TestJanelasDaConciliacaoCobremTudo(t *testing.T) {
	if JanelaDaConciliacao(0) < MargemCompensacao+48*time.Hour {
		t.Fatalf("a varredura mais larga (%s) não cobre o boleto informado depois de 3 dias úteis e um fim de semana",
			JanelaDaConciliacao(0))
	}
	casos := []struct {
		idade  time.Duration // idade do pagamento no extrato quando o aviso se perde
		limite int           // rodadas (minutos) até alguma janela alcançá-lo
	}{
		{30 * time.Second, 1},
		{9 * time.Minute, 1},
		{90 * time.Minute, 10},
		{30 * time.Hour, 60},
		{5 * 24 * time.Hour, 1440}, // boleto de quinta, informado na terça
	}
	for _, c := range casos {
		pior := 0
		for inicio := range 1440 {
			k := inicio
			for JanelaDaConciliacao(k) < c.idade+time.Duration(k-inicio)*time.Minute {
				k++
				if k-inicio > 2*1440 {
					t.Fatalf("pagamento de %s nunca é alcançado, começando na rodada %d", c.idade, inicio)
				}
			}
			pior = max(pior, k-inicio)
		}
		if pior >= c.limite {
			t.Errorf("pagamento de %s: no pior caso, %d rodadas até ser conferido; esperado menos de %d", c.idade, pior, c.limite)
		}
	}

	var curtas int
	for n := range 1440 {
		if JanelaDaConciliacao(n) == 10*time.Minute {
			curtas++
		}
	}
	if curtas != 1296 { // 1440 − 144 rodadas de 10 em 10
		t.Fatalf("%d rodadas curtas por dia; esperado 1.296", curtas)
	}
}
