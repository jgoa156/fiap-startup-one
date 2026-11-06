package main

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

const (
	pastaTF    = "../../../infra/terraform"
	publicado  = "../../../poc-web/public/docs/infra.json"
	recursoTF  = `(?m)^resource\s+"([^"]+)"\s+"([^"]+)"`
	comentario = "rode: go run ./cmd/infradoc (na pasta poc)"
)

// A página de infraestrutura é a do Terraform de hoje: o JSON publicado é o que o gerador produz
// agora. Mudou um .tf e não regenerou, a suíte quebra.
func TestInfraDocAtual(t *testing.T) {
	agora, err := Gerar(pastaTF)
	if err != nil {
		t.Fatal(err)
	}
	antes, err := os.ReadFile(publicado)
	if err != nil {
		t.Fatalf("sem %s: %v (%s)", publicado, err, comentario)
	}
	if !bytes.Equal(bytes.ReplaceAll(antes, []byte("\r\n"), []byte("\n")), agora) {
		t.Fatalf("a página de infraestrutura está velha: %s", comentario)
	}
}

// Todo recurso do Terraform aparece na página, com o arquivo dele: o leitor de blocos não perde
// nenhum (um recurso dentro de outro bloco, ou com chaves nas aspas, já o enganaria).
func TestInfraDocTemTodoRecurso(t *testing.T) {
	d, err := Ler(pastaTF)
	if err != nil {
		t.Fatal(err)
	}
	achados := map[string]string{}
	for _, r := range d.Recursos {
		achados[r.Tipo+"."+r.Nome] = r.Arquivo
	}
	nomes, _ := filepath.Glob(filepath.Join(pastaTF, "*.tf"))
	total := 0
	for _, n := range nomes {
		b, _ := os.ReadFile(n)
		for _, m := range regexp.MustCompile(recursoTF).FindAllStringSubmatch(string(b), -1) {
			total++
			if achados[m[1]+"."+m[2]] != filepath.Base(n) {
				t.Errorf("recurso fora da página: %s.%s (%s)", m[1], m[2], filepath.Base(n))
			}
		}
	}
	if total == 0 || total != len(d.Recursos) {
		t.Fatalf("%d recursos no Terraform, %d na página", total, len(d.Recursos))
	}
	for _, v := range d.Variaveis {
		if v.Descricao == "" {
			t.Errorf("variável sem descrição: %s (%s)", v.Nome, v.Arquivo)
		}
	}
}
