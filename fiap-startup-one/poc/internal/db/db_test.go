package db_test

import (
	"context"
	"os"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"lisura/internal/dbtest"
	"lisura/migrations"
)

var pool *pgxpool.Pool

func TestMain(m *testing.M) {
	var err error
	pool, err = dbtest.Pool(context.Background(), "teste_db")
	if err != nil {
		dbtest.SemBanco(err)
		os.Exit(0)
	}
	code := m.Run()
	pool.Close()
	os.Exit(code)
}

// retrato descreve o schema em linhas de texto comparáveis: tabelas e sequências, colunas (tipo,
// nulidade, padrão), restrições, índices, gatilhos, funções e tipos. A tabela de versões do goose
// fica de fora: é dele, não das migrações.
const retrato = `
WITH rel AS (
    SELECT oid, relname, relkind FROM pg_class
     WHERE relnamespace = current_schema()::regnamespace AND relname NOT LIKE 'goose_db_version%'
)
SELECT 'rel ' || relkind::text || ' ' || relname FROM rel WHERE relkind IN ('r', 'S', 'v', 'm')
UNION ALL
SELECT 'col ' || r.relname || '.' || a.attname || ' ' || format_type(a.atttypid, a.atttypmod)
       || CASE WHEN a.attnotnull THEN ' not null' ELSE '' END
       || coalesce(' default ' || pg_get_expr(d.adbin, d.adrelid), '')
  FROM pg_attribute a
  JOIN rel r ON r.oid = a.attrelid AND r.relkind = 'r'
  LEFT JOIN pg_attrdef d ON d.adrelid = a.attrelid AND d.adnum = a.attnum
 WHERE a.attnum > 0 AND NOT a.attisdropped
UNION ALL
SELECT 'con ' || r.relname || ' ' || c.conname || ' ' || pg_get_constraintdef(c.oid)
  FROM pg_constraint c JOIN rel r ON r.oid = c.conrelid
UNION ALL
SELECT 'idx ' || pg_get_indexdef(i.indexrelid)
  FROM pg_index i JOIN rel r ON r.oid = i.indrelid
UNION ALL
SELECT 'trg ' || pg_get_triggerdef(t.oid)
  FROM pg_trigger t JOIN rel r ON r.oid = t.tgrelid WHERE NOT t.tgisinternal
UNION ALL
SELECT 'fun ' || p.proname || '(' || pg_get_function_identity_arguments(p.oid) || ')'
  FROM pg_proc p WHERE p.pronamespace = current_schema()::regnamespace
UNION ALL
SELECT 'typ ' || typname FROM pg_type
 WHERE typnamespace = current_schema()::regnamespace AND typtype IN ('e', 'd')
ORDER BY 1`

func retratar(t *testing.T, ctx context.Context) []string {
	t.Helper()
	rows, err := pool.Query(ctx, retrato)
	if err != nil {
		t.Fatal(err)
	}
	linhas, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	return linhas
}

// diferenca lista o que só existe em a e o que só existe em b.
func diferenca(a, b []string) (soA, soB []string) {
	for _, s := range a {
		if !slices.Contains(b, s) {
			soA = append(soA, s)
		}
	}
	for _, s := range b {
		if !slices.Contains(a, s) {
			soB = append(soB, s)
		}
	}
	return soA, soB
}

// Desfazer uma migração em produção é a rota de fuga de uma implantação ruim na véspera do prazo
// (uma das 7 causas de queda). Rota de fuga que nunca rodou não é rota. Aqui cada migração, uma
// por vez, é aplicada, desfeita e reaplicada: desfeita, o schema tem de voltar exatamente ao que
// era antes dela (coluna, restrição, índice e gatilho); reaplicada, ao que era depois.
func TestCadaMigracaoDesfazERefaz(t *testing.T) {
	ctx := context.Background()
	completo := retratar(t, ctx)

	conn := stdlib.OpenDBFromPool(pool)
	defer conn.Close()
	p, err := goose.NewProvider(goose.DialectPostgres, conn, migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.DownTo(ctx, 0); err != nil {
		t.Fatalf("desfazer tudo: %v", err)
	}
	if vazio := retratar(t, ctx); len(vazio) != 0 {
		t.Fatalf("depois de desfazer tudo, sobrou no schema: %v", vazio)
	}

	antes := []string{}
	for _, fonte := range p.ListSources() {
		if _, err := p.UpByOne(ctx); err != nil {
			t.Fatalf("aplicar %s: %v", fonte.Path, err)
		}
		depois := retratar(t, ctx)
		if _, err := p.Down(ctx); err != nil {
			t.Fatalf("desfazer %s: %v", fonte.Path, err)
		}
		if sobra, falta := diferenca(retratar(t, ctx), antes); len(sobra)+len(falta) > 0 {
			t.Errorf("%s desfeita não volta ao anterior:\n  sobrou %q\n  faltou %q", fonte.Path, sobra, falta)
		}
		if _, err := p.UpByOne(ctx); err != nil {
			t.Fatalf("reaplicar %s: %v", fonte.Path, err)
		}
		if sobra, falta := diferenca(retratar(t, ctx), depois); len(sobra)+len(falta) > 0 {
			t.Errorf("%s reaplicada difere da 1ª aplicação:\n  sobrou %q\n  faltou %q", fonte.Path, sobra, falta)
		}
		antes = depois
	}
	if sobra, falta := diferenca(retratar(t, ctx), completo); len(sobra)+len(falta) > 0 {
		t.Errorf("schema refeito difere do original:\n  sobrou %q\n  faltou %q", sobra, falta)
	}
}
