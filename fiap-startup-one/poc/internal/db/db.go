// Package db abre o pool de conexões e aplica as migrações.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"

	"lisura/migrations"
)

// URL lê DATABASE_URL, com o banco local da POC como padrão.
func URL() string {
	if u := os.Getenv("DATABASE_URL"); u != "" {
		return u
	}
	return "postgres://postgres@localhost:5432/lisura?sslmode=disable"
}

func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		// Só host e banco: na nuvem a URL carrega a senha, que não pode ir para o log.
		cfg := pool.Config().ConnConfig
		return nil, fmt.Errorf("conectar em %s:%d/%s: %w", cfg.Host, cfg.Port, cfg.Database, err)
	}
	return pool, nil
}

// Migrate aplica todas as migrações pendentes.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	conn := stdlib.OpenDBFromPool(pool)
	defer conn.Close()
	return migrate(ctx, conn)
}

func migrate(ctx context.Context, conn *sql.DB) error {
	// Várias instâncias sobem juntas na nuvem: a trava consultiva do Postgres faz uma migrar e as
	// outras esperarem, em vez de todas aplicarem a mesma migração ao mesmo tempo.
	trava, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return err
	}
	p, err := goose.NewProvider(goose.DialectPostgres, conn, migrations.FS, goose.WithSessionLocker(trava))
	if err != nil {
		return err
	}
	_, err = p.Up(ctx)
	return err
}
