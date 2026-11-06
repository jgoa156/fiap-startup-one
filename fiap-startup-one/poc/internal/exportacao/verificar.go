package exportacao

import (
	"bytes"
	"cmp"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"lisura/internal/ancora"
	"lisura/internal/inscricao"
)

// Verificacao é o que a conferência do pacote achou.
type Verificacao struct {
	Certame         int64    `json:"certame"`
	Selo            string   `json:"selo"`
	Arquivos        int      `json:"arquivos"`
	Eventos         int      `json:"eventos"`
	Inscricoes      int      `json:"inscricoes"`
	PontosAncorados int      `json:"pontos_ancorados"`
	DiasFechados    int      `json:"dias_fechados"`
	Problemas       []string `json:"problemas"`
}

// Verificar confere o pacote só com o que está na pasta. Não abre o banco: é o que o auditor faz.
func Verificar(pasta string) (Verificacao, error) {
	v := Verificacao{Problemas: []string{}}
	problema := func(f string, a ...any) { v.Problemas = append(v.Problemas, fmt.Sprintf(f, a...)) }

	b, err := os.ReadFile(filepath.Join(pasta, "manifesto.json"))
	if err != nil {
		return v, err
	}
	var m Manifesto
	if err := json.Unmarshal(b, &m); err != nil {
		return v, fmt.Errorf("manifesto: %w", err)
	}
	v.Certame, v.Selo = m.Certame, m.Selo

	// 1. Cada arquivo bate com o manifesto; o manifesto, com o selo; nada a mais na pasta.
	listados := map[string]bool{"manifesto.json": true}
	for _, a := range m.Arquivos {
		listados[a.Nome] = true
		c, err := os.ReadFile(filepath.Join(pasta, filepath.FromSlash(a.Nome)))
		if errors.Is(err, fs.ErrNotExist) {
			problema("ARQUIVO FALTANDO: %s", a.Nome)
			continue
		}
		if err != nil {
			return v, err
		}
		h := sha256.Sum256(c)
		if hex.EncodeToString(h[:]) != a.SHA256 || int64(len(c)) != a.Bytes {
			problema("ARQUIVO ALTERADO: %s", a.Nome)
		}
		v.Arquivos++
	}
	if selo(m.Arquivos) != m.Selo {
		problema("SELO NÃO CONFERE: o manifesto foi alterado")
	}
	filepath.WalkDir(pasta, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			rel, _ := filepath.Rel(pasta, p)
			if !listados[filepath.ToSlash(rel)] {
				problema("ARQUIVO A MAIS: %s", filepath.ToSlash(rel))
			}
		}
		return nil
	})

	// 2. A corrente da trilha, refeita evento por evento, com a mesma conta do banco.
	ultimo := map[int64]string{}    // inscrição → resumo do último evento
	porEvento := map[int64]string{} // evento → resumo
	quebrada := map[int64]bool{}
	var anterior []byte
	atual := int64(-1)
	err = lerCSV(pasta, "trilha.csv", func(r map[string]string) error {
		id, insc := num(r["evento_id"]), num(r["inscricao"])
		versao, _ := strconv.Atoi(r["edital_versao"])
		em, err := time.Parse("2006-01-02T15:04:05.000000", r["ocorrido_em_utc"])
		if err != nil {
			return fmt.Errorf("evento %d: %w", id, err)
		}
		if insc != atual {
			atual, anterior = insc, nil
		}
		esperado := hex.EncodeToString(inscricao.ResumoDoEvento(anterior, insc, r["evento"], r["de_estado"], r["para_estado"],
			r["ator_tipo"], r["ator_id"], r["motivo"], versao, em))
		if esperado != r["resumo"] && !quebrada[insc] {
			problema("TRILHA QUEBRADA: inscrição %d, no evento %d", insc, id)
			quebrada[insc] = true
		}
		// Segue pelo resumo gravado: a adulteração aponta o evento adulterado, não todos os seguintes.
		anterior, _ = hex.DecodeString(r["resumo"])
		ultimo[insc], porEvento[id] = r["resumo"], r["resumo"]
		v.Eventos++
		return nil
	})
	if err != nil {
		return v, err
	}

	// 3. O selo de cada inscrição é o último evento da trilha dela.
	err = lerCSV(pasta, "inscricoes.csv", func(r map[string]string) error {
		insc := num(r["inscricao"])
		if r["selo"] != ultimo[insc] {
			problema("SELO DA INSCRIÇÃO %d NÃO É O FIM DA TRILHA", insc)
		}
		v.Inscricoes++
		return nil
	})
	if err != nil {
		return v, err
	}

	// 4. Cada ponto de cada âncora do cofre está na trilha exportada, com o mesmo resumo.
	for _, a := range m.Arquivos {
		if !strings.HasPrefix(a.Nome, "ancoras/") {
			continue
		}
		c, err := os.ReadFile(filepath.Join(pasta, filepath.FromSlash(a.Nome)))
		if err != nil {
			continue // já acusado como faltando
		}
		anc, err := ancora.Ler(c)
		if err != nil {
			problema("ÂNCORA ILEGÍVEL: %s", a.Nome)
			continue
		}
		for _, l := range anc.Linhas {
			if porEvento[l.Evento] != hex.EncodeToString(l.Resumo) {
				problema("ÂNCORA Nº %d NÃO CONFERE: inscrição %d, evento %d", anc.Sequencia, l.Inscricao, l.Evento)
			}
			v.PontosAncorados++
		}
	}

	// 5. Cada dia fechado, refeito dos pagamentos e das devoluções exportados.
	type linha struct{ tipo, id, valor string }
	type dia struct {
		pag, dev     int
		entrou, saiu int64
		linhas       []linha
	}
	dias := map[string]*dia{}
	pegar := func(d string) *dia {
		if dias[d] == nil {
			dias[d] = &dia{}
		}
		return dias[d]
	}
	err = lerCSV(pasta, "pagamentos.csv", func(r map[string]string) error {
		d := pegar(r["dia"])
		d.pag++
		d.entrou += num(r["valor_centavos"])
		d.linhas = append(d.linhas, linha{"P", r["psp_pagamento_id"], r["valor_centavos"]})
		return nil
	})
	if err == nil {
		err = lerCSV(pasta, "devolucoes.csv", func(r map[string]string) error {
			if r["estado"] != "CONCLUIDA" {
				return nil
			}
			d := pegar(r["dia"])
			d.dev++
			d.saiu += num(r["valor_centavos"])
			d.linhas = append(d.linhas, linha{"D", r["psp_devolucao_id"], r["valor_centavos"]})
			return nil
		})
	}
	if err == nil {
		err = lerCSV(pasta, "fechamentos.csv", func(r map[string]string) error {
			d := pegar(r["dia"])
			// A ordem do fechamento: por tipo e depois por id, byte a byte (COLLATE "C"). Ordenar a
			// linha inteira daria outra ordem: "abc|" vem depois de "abc_d", e o id "abc" vem antes.
			slices.SortFunc(d.linhas, func(a, b linha) int {
				return cmp.Or(strings.Compare(a.tipo, b.tipo), strings.Compare(a.id, b.id))
			})
			var buf bytes.Buffer
			for _, l := range d.linhas {
				buf.WriteString(l.tipo + "|" + l.id + "|" + l.valor + "\n")
			}
			h := sha256.Sum256(buf.Bytes())
			if hex.EncodeToString(h[:]) != r["resumo"] || int64(d.pag) != num(r["pagamentos"]) || d.entrou != num(r["entrou_centavos"]) ||
				int64(d.dev) != num(r["devolucoes"]) || d.saiu != num(r["saiu_centavos"]) {
				problema("DIA %s FECHADO NÃO CONFERE com os pagamentos e as devoluções exportados", r["dia"])
			}
			v.DiasFechados++
			return nil
		})
	}
	return v, err
}

func num(s string) int64 {
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

// lerCSV lê um CSV do pacote (separador ;, com cabeçalho) e chama f com cada linha, pelo nome da coluna.
func lerCSV(pasta, nome string, f func(map[string]string) error) error {
	arq, err := os.Open(filepath.Join(pasta, nome))
	if errors.Is(err, fs.ErrNotExist) {
		return nil // já acusado como faltando
	}
	if err != nil {
		return err
	}
	defer arq.Close()
	r := csv.NewReader(arq)
	r.Comma = ';'
	cab, err := r.Read()
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("%s: %w", nome, err)
	}
	linha := map[string]string{}
	for {
		reg, err := r.Read()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("%s: %w", nome, err)
		}
		for i, c := range cab {
			linha[c] = reg[i]
		}
		if err := f(linha); err != nil {
			return err
		}
	}
}
