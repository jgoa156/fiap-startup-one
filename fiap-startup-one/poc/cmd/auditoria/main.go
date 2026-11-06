// auditoria confere no banco o resultado de uma rodada do ensaio de carga e imprime o relatório em
// JSON. Sai com código 1 se achar inscrição perdida, cobrança duplicada ou pagamento sem lastro.
//
//	lisura-auditoria -rodada 123456 -enviadas 104871 -pagas 93120
//
// -enviadas e -pagas são os contadores do k6 (inscricoes_enviadas, pagamentos_vistos): o que o
// candidato viu. O banco tem de ter pelo menos isso.
//
//	lisura-auditoria -trilhas    refaz a corrente de todas as trilhas
//	lisura-auditoria -ancoras    confere o banco contra as âncoras do cofre (LISURA_COFRE_BUCKET ou LISURA_COFRE)
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"lisura/internal/ancora"
	"lisura/internal/arquivo"
	"lisura/internal/auditoria"
	"lisura/internal/db"
	"lisura/internal/inscricao"
)

func main() {
	rodada := flag.String("rodada", "", "rodada do ensaio (a RODADA do k6)")
	enviadas := flag.Int("enviadas", 0, "inscrições que o k6 viu enviadas")
	pagas := flag.Int("pagas", 0, "pagamentos que o k6 viu confirmados")
	trilhas := flag.Bool("trilhas", false, "em vez de uma rodada, refaz a corrente de TODAS as trilhas do banco")
	ancoras := flag.Bool("ancoras", false, "em vez de uma rodada, confere o banco contra as âncoras do cofre, certame a certame")
	flag.Parse()

	ctx := context.Background()
	pool, err := db.Connect(ctx, db.URL())
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if *ancoras {
		conferirAncoras(ctx, pool)
		return
	}
	if *trilhas {
		inicio := time.Now()
		quebras, selos, err := inscricao.VerificarTrilhas(ctx, pool, nil)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%d trilhas verificadas em %s; %d quebradas\n", len(selos), time.Since(inicio).Round(time.Millisecond), len(quebras))
		for _, q := range quebras {
			fmt.Println(q)
		}
		if len(quebras) > 0 {
			os.Exit(1)
		}
		return
	}
	r, err := auditoria.Auditar(ctx, pool, *rodada, auditoria.Esperado{Enviadas: *enviadas, Pagas: *pagas})
	if err != nil {
		log.Fatal(err)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(r)
	if len(r.Problemas) > 0 {
		os.Exit(1)
	}
}

// conferirAncoras imprime uma conferência por certame (JSON) e sai com 1 se alguma acusar algo.
// A corrente por dentro é a outra metade (-trilhas): a âncora confere o ponto, a corrente o caminho.
func conferirAncoras(ctx context.Context, pool *pgxpool.Pool) {
	cofre, onde, err := arquivo.CofreDoAmbiente(ctx)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("cofre: %s", onde)
	rows, err := pool.Query(ctx, `SELECT id FROM certame ORDER BY id`)
	if err != nil {
		log.Fatal(err)
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		log.Fatal(err)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	falhou := false
	for _, id := range ids {
		inicio := time.Now()
		c, err := ancora.Conferir(ctx, pool, cofre, id)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("certame %d: %d âncoras, %d pontos conferidos em %s, %d problemas",
			id, c.Ancoras, c.Pontos, time.Since(inicio).Round(time.Millisecond), len(c.Problemas))
		enc.Encode(c)
		falhou = falhou || len(c.Problemas) > 0
	}
	if falhou {
		os.Exit(1)
	}
}
