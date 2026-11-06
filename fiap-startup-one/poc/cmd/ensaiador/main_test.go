package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Só o painel dispara ensaio: sem o cabeçalho próprio, ou de outra origem, 403 e nada roda. É o que
// impede uma página qualquer, aberta no mesmo navegador, de mandar um POST para 127.0.0.1:8083.
func TestSoOPainelDispara(t *testing.T) {
	rodou := 0
	h := guardar([]string{"http://127.0.0.1:8090"}, func(w http.ResponseWriter, r *http.Request) { rodou++ })
	casos := []struct {
		nome, cabecalho, origem string
		passa                   bool
	}{
		{"o painel", "1", "http://127.0.0.1:8090", true},
		{"o painel, sem Origin (mesma origem em GET/POST antigo)", "1", "", true},
		{"sem o cabeçalho", "", "http://127.0.0.1:8090", false},
		{"outro site", "1", "https://malicioso.test", false},
		{"outra porta da mesma máquina", "1", "http://127.0.0.1:3000", false},
	}
	for _, c := range casos {
		r := httptest.NewRequest(http.MethodPost, "/ensaios", nil)
		if c.cabecalho != "" {
			r.Header.Set("X-Lisura-Painel", c.cabecalho)
		}
		if c.origem != "" {
			r.Header.Set("Origin", c.origem)
		}
		antes := rodou
		w := httptest.NewRecorder()
		h(w, r)
		if (rodou > antes) != c.passa || (!c.passa && w.Code != http.StatusForbidden) {
			t.Errorf("%s: status %d, rodou %v", c.nome, w.Code, rodou > antes)
		}
	}
}
