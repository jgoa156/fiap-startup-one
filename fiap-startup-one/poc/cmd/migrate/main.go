// migrate aplica as migrações no banco de DATABASE_URL.
package main

import (
	"context"
	"log"

	"lisura/internal/db"
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
	log.Print("migrações aplicadas")
}
