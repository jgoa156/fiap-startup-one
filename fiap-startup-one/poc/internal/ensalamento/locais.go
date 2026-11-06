package ensalamento

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Cabecalho é a primeira linha do CSV de locais e salas (separador ;).
const Cabecalho = "cidade;local;endereco;amamentacao;sala;capacidade;tipo;acessivel"

// ImportarLocais lê o CSV de locais e salas da banca, uma sala por linha. A cidade é o nome de uma
// cidade de prova do certame. Local e sala que já existem são atualizados (a mesma planilha pode ser
// importada de novo, corrigida). Uma linha errada recusa o arquivo inteiro, com o número da linha.
func ImportarLocais(ctx context.Context, pool *pgxpool.Pool, certameID int64, r io.Reader) (locais, salas int, err error) {
	leitor := csv.NewReader(r)
	leitor.Comma = ';'
	leitor.FieldsPerRecord = 8
	cab, err := leitor.Read()
	if err != nil {
		return 0, 0, fmt.Errorf("locais: %w", err)
	}
	if strings.Join(cab, ";") != Cabecalho {
		return 0, 0, fmt.Errorf("locais: a primeira linha tem de ser %q", Cabecalho)
	}
	err = pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		cidades := map[string]int64{}
		rows, err := tx.Query(ctx, `SELECT id, nome FROM cidade_prova WHERE certame_id = $1`, certameID)
		if err != nil {
			return err
		}
		var id int64
		var nome string
		if _, err := pgx.ForEachRow(rows, []any{&id, &nome}, func() error { cidades[nome] = id; return nil }); err != nil {
			return err
		}
		vistos := map[int64]bool{}
		for linha := 2; ; linha++ {
			reg, err := leitor.Read()
			if errors.Is(err, io.EOF) {
				return nil
			}
			if err != nil {
				return fmt.Errorf("locais, linha %d: %w", linha, err)
			}
			falhar := func(f string, a ...any) error {
				return fmt.Errorf("locais, linha %d: %s", linha, fmt.Sprintf(f, a...))
			}
			cidade, ok := cidades[strings.TrimSpace(reg[0])]
			if !ok {
				return falhar("cidade %q não é cidade de prova deste certame", reg[0])
			}
			capacidade, err := strconv.Atoi(strings.TrimSpace(reg[5]))
			if err != nil || capacidade <= 0 {
				return falhar("capacidade %q", reg[5])
			}
			tipo := strings.TrimSpace(reg[6])
			if tipo != "comum" && tipo != "especial" && tipo != "individual" {
				return falhar("tipo %q: comum, especial ou individual", reg[6])
			}
			if tipo == "individual" && capacidade != 1 {
				return falhar("sala individual tem capacidade 1, não %d", capacidade)
			}
			amamentacao, err1 := simNao(reg[3])
			acessivel, err2 := simNao(reg[7])
			if err := errors.Join(err1, err2); err != nil {
				return falhar("%v", err)
			}
			var localID int64
			if err := tx.QueryRow(ctx, `
				INSERT INTO local_prova (certame_id, cidade_prova_id, nome, endereco, amamentacao) VALUES ($1, $2, $3, $4, $5)
				ON CONFLICT (certame_id, cidade_prova_id, nome) DO UPDATE SET endereco = EXCLUDED.endereco, amamentacao = EXCLUDED.amamentacao
				RETURNING id`, certameID, cidade, strings.TrimSpace(reg[1]), strings.TrimSpace(reg[2]), amamentacao).Scan(&localID); err != nil {
				return falhar("%v", err)
			}
			if !vistos[localID] {
				vistos[localID] = true
				locais++
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO sala (local_id, nome, capacidade, tipo, acessivel) VALUES ($1, $2, $3, $4, $5)
				ON CONFLICT (local_id, nome) DO UPDATE SET capacidade = EXCLUDED.capacidade, tipo = EXCLUDED.tipo, acessivel = EXCLUDED.acessivel`,
				localID, strings.TrimSpace(reg[4]), capacidade, tipo, acessivel); err != nil {
				return falhar("%v", err)
			}
			salas++
		}
	})
	return locais, salas, err
}

func simNao(s string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "sim", "s", "1", "true":
		return true, nil
	case "não", "nao", "n", "0", "false", "":
		return false, nil
	}
	return false, fmt.Errorf("%q não é sim nem não", s)
}
