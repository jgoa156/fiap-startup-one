// relatorio escreve o relatório do ensaio (o entregável da POC) a partir de um manifesto que aponta,
// para cada patamar, o log do k6 e o da auditoria daquela rodada:
//
//	lisura-relatorio -manifesto ensaio.json -saida relatorio.md
//
//	{
//	  "titulo": "Ensaio do H1 — AWS sa-east-1, 12/11/2026",
//	  "ambiente": "ECS Fargate 4×1 vCPU, Aurora Serverless v2 0,5–16 ACU",
//	  "orcamento_usd_hora": 4.44,
//	  "patamares": [
//	    {"k6": "k6-40x.log", "auditoria": "auditoria-40x.json", "custo_usd": 11.80, "notas": "..."}
//	  ]
//	}
//
// Os caminhos são relativos ao manifesto. custo_usd vem da fatura (Cost Explorer), por patamar; sem
// ele, o custo fica de fora. Sai com 1 se o H1 não passou — o relatório é escrito do mesmo jeito.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"lisura/internal/relatorio"
)

type manifesto struct {
	Titulo        string  `json:"titulo"`
	Ambiente      string  `json:"ambiente"`
	OrcamentoHora float64 `json:"orcamento_usd_hora"`
	Patamares     []struct {
		K6        string    `json:"k6"`
		Auditoria string    `json:"auditoria"`
		CustoUSD  float64   `json:"custo_usd"`
		Notas     string    `json:"notas"`
		Inicio    time.Time `json:"inicio"`
		Fim       time.Time `json:"fim"`
	} `json:"patamares"`
}

func main() {
	caminho := flag.String("manifesto", "", "manifesto do ensaio (JSON)")
	saida := flag.String("saida", "", "onde escrever o relatório (padrão: a saída padrão)")
	veredito := flag.String("veredito", "", "se dado, escreve aqui o veredito em JSON (para o orquestrador)")
	flag.Parse()
	if *caminho == "" {
		flag.Usage()
		os.Exit(2)
	}
	b, err := os.ReadFile(*caminho)
	if err != nil {
		log.Fatal(err)
	}
	var m manifesto
	if err := json.Unmarshal(b, &m); err != nil {
		log.Fatalf("manifesto: %v", err)
	}
	dir := filepath.Dir(*caminho)
	ler := func(nome string) []byte {
		if !filepath.IsAbs(nome) {
			nome = filepath.Join(dir, nome)
		}
		b, err := os.ReadFile(nome)
		if err != nil {
			log.Fatal(err)
		}
		return b
	}
	var ps []relatorio.Patamar
	for i, p := range m.Patamares {
		k, err := relatorio.LerK6(ler(p.K6))
		if err != nil {
			log.Fatalf("patamar %d, %s: %v", i+1, p.K6, err)
		}
		a, err := relatorio.LerAuditoria(ler(p.Auditoria))
		if err != nil {
			log.Fatalf("patamar %d, %s: %v", i+1, p.Auditoria, err)
		}
		ps = append(ps, relatorio.Patamar{K6: k, Auditoria: a, CustoUSD: p.CustoUSD, Notas: p.Notas, Inicio: p.Inicio, Fim: p.Fim})
	}
	e := relatorio.Montar(m.Titulo, m.Ambiente, m.OrcamentoHora, ps)
	md := e.Markdown()
	if *saida == "" {
		fmt.Print(md)
	} else if err := os.WriteFile(*saida, []byte(md), 0o644); err != nil {
		log.Fatal(err)
	}
	if *veredito != "" {
		b, _ := json.MarshalIndent(e.Veredito(), "", "  ")
		if err := os.WriteFile(*veredito, b, 0o644); err != nil {
			log.Fatal(err)
		}
	}
	if ok, porque := e.H1(); !ok {
		log.Printf("H1 não passou: %s", porque)
		os.Exit(1)
	}
}
