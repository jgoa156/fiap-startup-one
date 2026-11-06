package conta

import "testing"

func TestCPF(t *testing.T) {
	for cpf, ok := range map[string]bool{
		"52998224725": true, // exemplo público de CPF válido
		"52998224724": false,
		"11111111111": false,
		"5299822472":  false,
		"5299822472a": false,
	} {
		if CPFValido(cpf) != ok {
			t.Errorf("CPFValido(%s) != %v", cpf, ok)
		}
	}
	for base := int64(1); base <= 2000; base++ {
		if c := GerarCPF(base); !CPFValido(c) {
			t.Fatalf("GerarCPF(%d) = %s inválido", base, c)
		}
	}
}
