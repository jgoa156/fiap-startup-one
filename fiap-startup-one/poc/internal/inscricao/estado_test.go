package inscricao

import "testing"

var todosEstados = []Estado{
	Rascunho, IsencaoAtiva, AguardandoPagamento, Paga, Expirada,
	Cancelada, EmAnalise, Deferida, Indeferida, Homologada,
}

func TestTodoEstadoAlcancavelAPartirDoRascunho(t *testing.T) {
	visto := map[Estado]bool{Rascunho: true}
	fila := []Estado{Rascunho}
	for len(fila) > 0 {
		atual := fila[0]
		fila = fila[1:]
		for ev := range Transicoes {
			if d, ok := Proximo(atual, ev); ok && !visto[d] {
				visto[d] = true
				fila = append(fila, d)
			}
		}
	}
	for _, e := range todosEstados {
		if !visto[e] {
			t.Errorf("%s não é alcançável a partir de RASCUNHO", e)
		}
	}
}

func TestEstadosFinaisNaoTemSaida(t *testing.T) {
	for _, final := range []Estado{Cancelada, Expirada, Homologada} {
		for ev := range Transicoes {
			if final == Homologada && ev == CancelarCertame {
				continue // a anulação do certame é a única saída da homologação
			}
			if d, ok := Proximo(final, ev); ok {
				t.Errorf("%s é final mas %s leva a %s", final, ev, d)
			}
		}
	}
}

// A anulação do certame alcança toda inscrição viva e nenhuma morta.
func TestCancelarCertameAlcancaTodaInscricaoViva(t *testing.T) {
	for _, e := range todosEstados {
		_, ok := Proximo(e, CancelarCertame)
		if morta := e == Cancelada || e == Expirada; ok == morta {
			t.Errorf("CANCELAR_CERTAME a partir de %s: permitido=%v", e, ok)
		}
	}
	if !Transicoes[CancelarCertame].ExigeMotivo || Transicoes[CancelarCertame].Ator != Analista {
		t.Fatal("anular o certame exige motivo e é decisão da banca")
	}
}

func TestProximo(t *testing.T) {
	casos := []struct {
		de   Estado
		ev   Evento
		para Estado
		ok   bool
	}{
		{Rascunho, FinalizarComPagamento, AguardandoPagamento, true},
		{Rascunho, FinalizarIsento, IsencaoAtiva, true},
		{Rascunho, Cancelar, Cancelada, true},
		{AguardandoPagamento, ConfirmarPagamento, Paga, true},
		{AguardandoPagamento, Expirar, Expirada, true},
		{Paga, IniciarAnalise, EmAnalise, true},
		{IsencaoAtiva, IniciarAnalise, EmAnalise, true},
		{EmAnalise, Indeferir, Indeferida, true},
		{Indeferida, ProverRecurso, Deferida, true},
		{Deferida, Homologar, Homologada, true},
		// Proibidas
		{AguardandoPagamento, Cancelar, "", false},
		{Paga, ConfirmarPagamento, "", false},
		{Expirada, ConfirmarPagamento, "", false},
		{Rascunho, Homologar, "", false},
		{EmAnalise, Homologar, "", false},
		{Rascunho, "INEXISTENTE", "", false},
	}
	for _, c := range casos {
		para, ok := Proximo(c.de, c.ev)
		if para != c.para || ok != c.ok {
			t.Errorf("Proximo(%s, %s) = (%s, %v), esperado (%s, %v)", c.de, c.ev, para, ok, c.para, c.ok)
		}
	}
}
