package conta

import (
	"fmt"
	"strings"
)

func SoDigitos(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, s)
}

// CPFValido confere os dois dígitos verificadores e recusa sequências repetidas.
func CPFValido(cpf string) bool {
	if len(cpf) != 11 || strings.Count(cpf, cpf[:1]) == 11 || SoDigitos(cpf) != cpf {
		return false
	}
	return cpf[9:] == digitos(cpf[:9])
}

// GerarCPF devolve um CPF válido e sintético a partir de uma base de até 9 dígitos. Os dados de
// candidato da POC são todos gerados, nunca reais (04-escopo-poc §4).
func GerarCPF(base int64) string {
	b := fmt.Sprintf("%09d", base%1_000_000_000)
	return b + digitos(b)
}

func digitos(base9 string) string {
	d := []byte(base9)
	for _, n := range []int{10, 11} {
		soma := 0
		for i := 0; i < n-1; i++ {
			soma += int(d[i]-'0') * (n - i)
		}
		dv := soma * 10 % 11 % 10
		d = append(d, byte('0'+dv))
	}
	return string(d[9:])
}
