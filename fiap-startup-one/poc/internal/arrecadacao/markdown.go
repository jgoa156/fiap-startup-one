package arrecadacao

import (
	"fmt"
	"strings"
)

// Markdown escreve a prestação de contas como a banca a lê e assina.
func (p Prestacao) Markdown() string {
	var b strings.Builder
	w := func(f string, a ...any) { fmt.Fprintf(&b, f, a...) }
	w("# Prestação de contas — %s (certame %d)\n\n", p.Nome, p.Certame)
	if p.Cancelado {
		w("**Certame cancelado:** todo pagamento confirmado é devolvido.\n\n")
	}
	w("Gerada em %s. Valores em reais. O dia de um pagamento é o dia em que o sistema o registrou (horário de Brasília); ", p.GeradaEm.Format("02/01/2006 15:04"))
	w("o de uma devolução, o dia em que o prestador a concluiu. O dinheiro não passa pelo Lisura: está na conta da banca, no prestador dela.\n\n")

	w("| | Quantos | Valor |\n|---|---:|---:|\n")
	for _, l := range []struct {
		nome string
		t    Total
	}{
		{"Entrou (todo pagamento registrado)", p.Entrou},
		{"Saiu (devoluções concluídas)", p.Saiu},
		{"A devolver (devoluções em andamento)", p.ADevolver},
		{"Em divergência (valor pago diferente do cobrado)", p.EmDivergencia},
		{"**Taxas válidas** (confirmadas, sem devolução)", p.Validas},
	} {
		w("| %s | %s | %s |\n", l.nome, milhar(l.t.N), BRL(l.t.Centavos))
	}
	if p.Diferenca == 0 {
		w("\n**A conta fecha:** entrou − saiu − a devolver − em divergência = taxas válidas, centavo por centavo.\n")
	} else {
		w("\n**A CONTA NÃO FECHA: diferença de %s.** Há dinheiro sem explicação; esta prestação não pode ser assinada.\n", BRL(p.Diferenca))
	}

	tabela := func(titulo string, linhas []Linha) {
		if len(linhas) == 0 {
			return
		}
		w("\n## %s\n\n| | Pagamentos | Entrou | Devoluções | Saiu | Saldo | Taxas válidas |\n|---|---:|---:|---:|---:|---:|---:|\n", titulo)
		for _, l := range linhas {
			w("| %s | %s | %s | %s | %s | %s | %s |\n", l.Chave, milhar(l.Entrou.N), BRL(l.Entrou.Centavos),
				milhar(l.Saiu.N), BRL(l.Saiu.Centavos), BRL(l.Entrou.Centavos-l.Saiu.Centavos), BRL(l.Validas.Centavos))
		}
	}
	tabela("Por dia", p.PorDia)
	tabela("Por cargo", p.PorCargo)
	tabela("Por cidade de prova", p.PorCidade)
	return b.String()
}

func milhar(n int) string {
	s := fmt.Sprint(n)
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}
	return b.String()
}
