// Package dbtest prepara um banco de teste isolado por pacote: cada pacote recebe o próprio schema,
// recriado e migrado do zero, para que `go test ./...` possa rodar pacotes em paralelo.
package dbtest

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"lisura/internal/db"
)

func url() string {
	if u := os.Getenv("TEST_DATABASE_URL"); u != "" {
		return u
	}
	return "postgres://postgres@localhost:5432/lisura_test?sslmode=disable"
}

// SemBanco decide o que fazer quando o banco de teste não sobe: por padrão a suíte FALHA, porque
// teste que pula em silêncio parece verde sem ter testado nada. Para pular de propósito (máquina
// sem Postgres), defina LISURA_SEM_BANCO=1.
func SemBanco(err error) {
	if os.Getenv("LISURA_SEM_BANCO") != "" {
		fmt.Fprintln(os.Stderr, "LISURA_SEM_BANCO definido: testes de integração pulados:", err)
		return
	}
	fmt.Fprintln(os.Stderr, "banco de teste indisponível (suba com scripts\\db-up.ps1 ou defina LISURA_SEM_BANCO=1):", err)
	os.Exit(1)
}

// Pool devolve um pool cujo search_path é um schema novo e migrado. Sem banco disponível devolve
// erro, e o chamador decide se pula os testes.
func Pool(ctx context.Context, schema string) (*pgxpool.Pool, error) {
	conn, err := pgx.Connect(ctx, url())
	if err != nil {
		return nil, err
	}
	ident := pgx.Identifier{schema}.Sanitize()
	_, err = conn.Exec(ctx, fmt.Sprintf(`DROP SCHEMA IF EXISTS %s CASCADE; CREATE SCHEMA %s`, ident, ident))
	conn.Close(ctx)
	if err != nil {
		return nil, err
	}

	cfg, err := pgxpool.ParseConfig(url())
	if err != nil {
		return nil, err
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := db.Migrate(ctx, pool); err != nil {
		pool.Close()
		return nil, fmt.Errorf("migrar schema %s: %w", schema, err)
	}
	return pool, nil
}
