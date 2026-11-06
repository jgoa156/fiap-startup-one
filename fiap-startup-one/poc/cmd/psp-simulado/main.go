// psp-simulado sobe o prestador de pagamento de mentira, com os defeitos de um de verdade.
//
//	go run ./cmd/psp-simulado -repetir 3 -perder 0.2
//	go run ./cmd/psp-simulado -diario dados/psp-diario.jsonl   (sobrevive a reinício)
package main

import (
	"errors"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"lisura/internal/psp"
)

func main() {
	addr := flag.String("addr", ":8081", "endereço")
	aviso := flag.String("aviso", "http://localhost:8082/api/psp/aviso", "para onde mandar o aviso de pagamento")
	// O segredo vem do ambiente na nuvem (um argumento de linha de comando aparece na definição da tarefa).
	padrao := os.Getenv("LISURA_PSP_SEGREDO")
	if padrao == "" {
		padrao = "segredo-do-sandbox"
	}
	segredo := flag.String("segredo", padrao, "segredo que assina o aviso (igual a LISURA_PSP_SEGREDO da API)")
	repetir := flag.Int("repetir", 1, "quantas vezes mandar cada aviso, ao mesmo tempo")
	perder := flag.Float64("perder", 0, "chance de um aviso nunca chegar (0 a 1)")
	atraso := flag.Duration("atraso", time.Second, "espera antes de avisar")
	diario := flag.String("diario", "", "arquivo onde guardar cobranças e pagamentos, relido ao subir (vazio: só memória)")
	flag.Parse()

	s := &psp.Simulador{WebhookURL: *aviso, Segredo: []byte(*segredo),
		AvisosPorPagamento: *repetir, ProbPerderAviso: *perder, AtrasoAviso: *atraso}
	if *diario != "" {
		if f, err := os.Open(*diario); err == nil {
			c, p, err := s.Restaurar(f)
			f.Close()
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("diário %s relido: %d cobranças, %d pagamentos", *diario, c, p)
		} else if !errors.Is(err, fs.ErrNotExist) {
			log.Fatal(err)
		}
		f, err := os.OpenFile(*diario, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		s.Diario = f
	}
	log.Printf("PSP simulado em %s · aviso para %s · repetir=%d perder=%.0f%% atraso=%s",
		*addr, *aviso, *repetir, *perder*100, *atraso)
	log.Fatal(http.ListenAndServe(*addr, s.Handler()))
}
