package api

import (
	"bufio"
	"bytes"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// Rotas que o servidor serve fora do api.Server (cmd/lisura monta direto no mux): documentadas, mas
// não estão na tabela de rotas.
var foraDaTabela = []string{"GET /situacao", "POST /operacao", "POST /ses/eventos"}

// documentadas lê do openapi.yaml os pares "MÉTODO /caminho". O leitor só entende o recuo deste
// arquivo (caminho com 2 espaços, método com 4): se o formato mudar e nada for achado, o teste falha.
func documentadas(t *testing.T) []string {
	t.Helper()
	caminho := regexp.MustCompile(`^  (/\S*):\s*$`)
	metodo := regexp.MustCompile(`^    (get|post|put|patch|delete):`)
	var out []string
	var atual string
	emPaths := false
	s := bufio.NewScanner(bytes.NewReader(OpenAPI))
	for s.Scan() {
		l := s.Text()
		if l == "paths:" {
			emPaths = true
			continue
		}
		if !emPaths {
			continue
		}
		if m := caminho.FindStringSubmatch(l); m != nil {
			atual = m[1]
		} else if m := metodo.FindStringSubmatch(l); m != nil && atual != "" {
			out = append(out, strings.ToUpper(m[1])+" "+atual)
		}
	}
	if len(out) == 0 {
		t.Fatal("nenhuma rota lida do openapi.yaml: o formato mudou?")
	}
	return out
}

// Toda rota do servidor está documentada, e toda rota documentada existe: a documentação não fica
// para trás de uma rota nova nem promete uma que foi tirada.
func TestOpenAPICobreTodasAsRotas(t *testing.T) {
	doc := documentadas(t)
	reais := append(Padroes(), foraDaTabela...)
	for _, r := range reais {
		if !slices.Contains(doc, r) {
			t.Errorf("rota sem documentação no openapi.yaml: %s", r)
		}
	}
	for _, d := range doc {
		if !slices.Contains(reais, d) {
			t.Errorf("documentada, mas o servidor não responde: %s", d)
		}
	}
}
