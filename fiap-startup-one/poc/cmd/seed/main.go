// seed cria um certame sintético com inscrição e isenção abertas e imprime os ids em JSON.
package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"lisura/internal/db"
	"lisura/internal/seed"
)

func main() {
	ctx := context.Background()
	pool, err := db.Connect(ctx, db.URL())
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		log.Fatal(err)
	}
	c, err := seed.CriarCertame(ctx, pool, seed.Padrao())
	if err != nil {
		log.Fatal(err)
	}
	json.NewEncoder(os.Stdout).Encode(c)
}
