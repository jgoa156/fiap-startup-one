// infradoc lê o Terraform (infra/terraform/*.tf) e grava o que a página de infraestrutura
// (/docs/infra no front) desenha: cada recurso com o comentário que o explica, quem ele usa, as
// variáveis com descrição e padrão, e as saídas. A página não é escrita à mão: muda quando o
// Terraform muda, e TestInfraDocAtual quebra a suíte se o JSON publicado ficar para trás.
//
//	infradoc [-tf ../infra/terraform] [-saida ../poc-web/public/docs/infra.json]
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

type Arquivo struct {
	Nome   string `json:"nome"`
	Resumo string `json:"resumo"` // o comentário do topo do arquivo
}

type Bloco struct {
	Tipo        string   `json:"tipo"` // o tipo do recurso (aws_ecs_service) ou "variable", "output", "data.<tipo>"
	Nome        string   `json:"nome"`
	Arquivo     string   `json:"arquivo"`
	Descricao   string   `json:"descricao,omitempty"` // os comentários logo acima do bloco
	Referencias []string `json:"referencias,omitempty"`
	Padrao      string   `json:"padrao,omitempty"`      // variable: o default
	Condicional bool     `json:"condicional,omitempty"` // tem count/for_each: pode existir ou não
}

type Doc struct {
	Arquivos  []Arquivo `json:"arquivos"`
	Recursos  []Bloco   `json:"recursos"`
	Dados     []Bloco   `json:"dados"`
	Variaveis []Bloco   `json:"variaveis"`
	Saidas    []Bloco   `json:"saidas"`
}

var (
	cabeca     = regexp.MustCompile(`^(resource|data)\s+"([^"]+)"\s+"([^"]+)"\s*\{`)
	cabecaUm   = regexp.MustCompile(`^(variable|output)\s+"([^"]+)"\s*\{`)
	referencia = regexp.MustCompile(`\b(data\.[a-z0-9_]+\.[a-z0-9_]+|aws_[a-z0-9_]+\.[a-z0-9_]+|random_[a-z0-9_]+\.[a-z0-9_]+|var\.[a-z0-9_]+)`)
	descricao  = regexp.MustCompile(`^\s*description\s*=\s*"(.*)"\s*$`)
	padrao     = regexp.MustCompile(`^\s*default\s*=\s*(.+?)\s*$`)
	contagem   = regexp.MustCompile(`^\s*(count|for_each)\s*=`)
)

// Ler monta a documentação de uma pasta de Terraform.
func Ler(pasta string) (Doc, error) {
	nomes, err := filepath.Glob(filepath.Join(pasta, "*.tf"))
	if err != nil {
		return Doc{}, err
	}
	slices.Sort(nomes)
	d := Doc{Arquivos: []Arquivo{}, Recursos: []Bloco{}, Dados: []Bloco{}, Variaveis: []Bloco{}, Saidas: []Bloco{}}
	for _, n := range nomes {
		if err := lerArquivo(n, &d); err != nil {
			return Doc{}, err
		}
	}
	return d, nil
}

func lerArquivo(caminho string, d *Doc) error {
	f, err := os.Open(caminho)
	if err != nil {
		return err
	}
	defer f.Close()
	nome := filepath.Base(caminho)
	var comentario []string
	var topo []string
	noTopo := true
	var atual *Bloco
	profundidade := 0
	refs := map[string]bool{}
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 64<<10), 1<<20)
	for s.Scan() {
		l := s.Text()
		t := strings.TrimSpace(l)
		if atual == nil {
			switch {
			case strings.HasPrefix(t, "#"):
				c := strings.TrimSpace(strings.TrimPrefix(t, "#"))
				comentario = append(comentario, c)
				continue
			case t == "":
				if noTopo && len(comentario) > 0 {
					topo, comentario, noTopo = comentario, nil, false
				}
				comentario = nil
				continue
			}
			noTopo = false
			var b *Bloco
			if m := cabeca.FindStringSubmatch(t); m != nil {
				tipo := m[2]
				if m[1] == "data" {
					tipo = "data." + tipo
				}
				b = &Bloco{Tipo: tipo, Nome: m[3], Arquivo: nome}
			} else if m := cabecaUm.FindStringSubmatch(t); m != nil {
				b = &Bloco{Tipo: m[1], Nome: m[2], Arquivo: nome}
			}
			if b != nil {
				b.Descricao = strings.Join(comentario, " ")
				atual, profundidade, refs = b, 0, map[string]bool{}
			}
			comentario = nil
			if atual == nil {
				continue
			}
		}
		// Dentro de um bloco: conta chaves (fora de aspas basta para este Terraform) e junta referências.
		if atual.Tipo == "variable" || atual.Tipo == "output" {
			if m := descricao.FindStringSubmatch(l); m != nil && atual.Descricao == "" {
				atual.Descricao = strings.ReplaceAll(m[1], `\"`, `"`)
			}
			if m := padrao.FindStringSubmatch(l); m != nil && profundidade == 1 {
				atual.Padrao = m[1]
			}
		}
		if contagem.MatchString(l) && profundidade == 1 {
			atual.Condicional = true
		}
		sem := l
		if i := strings.Index(sem, "#"); i >= 0 && !strings.Contains(sem[:i], `"`) {
			sem = sem[:i]
		}
		for _, r := range referencia.FindAllString(sem, -1) {
			if r != atual.Tipo+"."+atual.Nome {
				refs[r] = true
			}
		}
		profundidade += strings.Count(sem, "{") - strings.Count(sem, "}")
		if profundidade <= 0 {
			for r := range refs {
				atual.Referencias = append(atual.Referencias, r)
			}
			slices.Sort(atual.Referencias)
			switch {
			case atual.Tipo == "variable":
				d.Variaveis = append(d.Variaveis, *atual)
			case atual.Tipo == "output":
				d.Saidas = append(d.Saidas, *atual)
			case strings.HasPrefix(atual.Tipo, "data."):
				d.Dados = append(d.Dados, *atual)
			default:
				d.Recursos = append(d.Recursos, *atual)
			}
			atual = nil
		}
	}
	if noTopo && len(comentario) > 0 {
		topo = comentario
	}
	d.Arquivos = append(d.Arquivos, Arquivo{Nome: nome, Resumo: strings.Join(topo, " ")})
	return s.Err()
}

// Gerar devolve o JSON (estável: mesma entrada, mesmos bytes).
func Gerar(pasta string) ([]byte, error) {
	d, err := Ler(pasta)
	if err != nil {
		return nil, err
	}
	b, err := json.MarshalIndent(d, "", "  ")
	return append(b, '\n'), err
}

func main() {
	tf := flag.String("tf", "../infra/terraform", "pasta do Terraform")
	saida := flag.String("saida", "../poc-web/public/docs/infra.json", "onde gravar o JSON")
	flag.Parse()
	b, err := Gerar(*tf)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(*saida), 0o755); err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(*saida, b, 0o644); err != nil {
		log.Fatal(err)
	}
	log.Printf("infraestrutura documentada em %s", *saida)
}
