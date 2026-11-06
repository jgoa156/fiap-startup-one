package api

import (
	_ "embed"
	"net/http"
)

// A especificação OpenAPI da API, embutida no binário: a página de documentação (/docs/api no front)
// lê GET /api/openapi.yaml, e o que está lá é o que este binário serve.
//
//go:embed openapi.yaml
var OpenAPI []byte

func servirOpenAPI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(OpenAPI)
}
