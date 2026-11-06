package ensalamento

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Conferencia é a prova do ensalamento, refeita da base.
type Conferencia struct {
	Certame   int64    `json:"certame"`
	Alocadas  int      `json:"alocadas"`
	Versao    int      `json:"versao"`
	Problemas []string `json:"problemas"`
	Duracao   float64  `json:"duracao_s"`
}

// Conferir prova que cada inscrição homologada tem uma sala, na cidade dela, que atende o atendimento
// especial pedido; que nenhuma sala passa da capacidade; que ninguém fora da homologação tem sala; e
// que a alocação de agora é a do último ato publicado (o resumo bate).
func Conferir(ctx context.Context, pool *pgxpool.Pool, certameID int64) (Conferencia, error) {
	inicio := time.Now()
	c := Conferencia{Certame: certameID, Problemas: []string{}}
	err := pgx.BeginTxFunc(ctx, pool, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly}, func(tx pgx.Tx) error {
		problema := func(f string, a ...any) { c.Problemas = append(c.Problemas, fmt.Sprintf(f, a...)) }
		contar := func(sql string) (int, error) {
			var n int
			return n, tx.QueryRow(ctx, sql, certameID).Scan(&n)
		}
		for _, q := range []struct{ sql, oque string }{
			{`SELECT count(*) FROM inscricao i WHERE i.certame_id = $1 AND i.estado = 'HOMOLOGADA'
			  AND NOT EXISTS (SELECT 1 FROM alocacao a WHERE a.inscricao_id = i.id)`, "HOMOLOGADA SEM SALA"},
			{`SELECT count(*) FROM alocacao a JOIN inscricao i ON i.id = a.inscricao_id
			  WHERE a.certame_id = $1 AND i.estado <> 'HOMOLOGADA'`, "SALA PARA QUEM NÃO ESTÁ HOMOLOGADA"},
			{`SELECT count(*) FROM alocacao a JOIN inscricao i ON i.id = a.inscricao_id
			  JOIN sala s ON s.id = a.sala_id JOIN local_prova l ON l.id = s.local_id
			  WHERE a.certame_id = $1 AND (i.certame_id <> l.certame_id OR i.cidade_prova_id IS DISTINCT FROM l.cidade_prova_id)`,
				"SALA EM OUTRA CIDADE"},
			{`SELECT count(*) FROM (SELECT s.id FROM sala s JOIN local_prova l ON l.id = s.local_id
			  JOIN alocacao a ON a.sala_id = s.id WHERE l.certame_id = $1
			  GROUP BY s.id, s.capacidade HAVING count(*) > s.capacidade) x`, "SALA ACIMA DA CAPACIDADE"},
		} {
			n, err := contar(q.sql)
			if err != nil {
				return err
			}
			if n > 0 {
				problema("%s: %d", q.oque, n)
			}
		}

		// O atendimento especial, conferido com a mesma regra da alocação.
		rows, err := tx.Query(ctx, `
			SELECT i.id, i.necessidades, s.tipo, s.acessivel, l.amamentacao
			FROM alocacao a JOIN inscricao i ON i.id = a.inscricao_id
			JOIN sala s ON s.id = a.sala_id JOIN local_prova l ON l.id = s.local_id
			WHERE a.certame_id = $1`, certameID)
		if err != nil {
			return err
		}
		var insc int64
		var nec []string
		var s Sala
		var naoAtendidas []int64
		if _, err := pgx.ForEachRow(rows, []any{&insc, &nec, &s.Tipo, &s.Acessivel, &s.Amamentacao}, func() error {
			c.Alocadas++
			if !s.Atende(RequisitoDe(nec)) {
				naoAtendidas = append(naoAtendidas, insc)
			}
			return nil
		}); err != nil {
			return err
		}
		if len(naoAtendidas) > 0 {
			slices.Sort(naoAtendidas)
			problema("ATENDIMENTO ESPECIAL NÃO ATENDIDO: %d inscrições (%v)", len(naoAtendidas), naoAtendidas[:min(10, len(naoAtendidas))])
		}

		var resumo string
		err = tx.QueryRow(ctx, `SELECT versao, resumo FROM ensalamento_versao WHERE certame_id = $1 ORDER BY versao DESC LIMIT 1`,
			certameID).Scan(&c.Versao, &resumo)
		if errors.Is(err, pgx.ErrNoRows) {
			if c.Alocadas > 0 {
				problema("ALOCAÇÃO SEM ATO PUBLICADO: %d inscrições com sala e nenhum ensalamento", c.Alocadas)
			}
			return nil
		}
		if err != nil {
			return err
		}
		agora, err := resumoAtual(ctx, tx, certameID)
		if err != nil {
			return err
		}
		if agora != resumo {
			problema("ALOCAÇÃO MUDOU FORA DO ENSALAMENTO: a versão %d publicou %s…, a de agora é %s…", c.Versao, resumo[:12], agora[:12])
		}
		return nil
	})
	c.Duracao = time.Since(inicio).Seconds()
	return c, err
}

// Ocupacao é uma linha do mapa de ocupação: um local.
type Ocupacao struct {
	Cidade     string `json:"cidade"`
	Local      string `json:"local"`
	Salas      int    `json:"salas"`
	Capacidade int    `json:"capacidade"`
	Alocados   int    `json:"alocados"`
	Especiais  int    `json:"lugares_especiais"`  // em sala especial ou individual
	Acessiveis int    `json:"lugares_acessiveis"` // em sala acessível
}

// Mapa é o mapa de ocupação por local.
func Mapa(ctx context.Context, pool *pgxpool.Pool, certameID int64) ([]Ocupacao, error) {
	rows, err := pool.Query(ctx, `
		WITH ocupados AS (SELECT sala_id, count(*) AS n FROM alocacao WHERE certame_id = $1 GROUP BY sala_id)
		SELECT cp.nome || ' (' || cp.uf || ')', l.nome, count(*), sum(s.capacidade),
		       coalesce(sum(o.n), 0),
		       coalesce(sum(s.capacidade) FILTER (WHERE s.tipo <> 'comum'), 0),
		       coalesce(sum(s.capacidade) FILTER (WHERE s.acessivel), 0)
		FROM local_prova l JOIN cidade_prova cp ON cp.id = l.cidade_prova_id JOIN sala s ON s.local_id = l.id
		LEFT JOIN ocupados o ON o.sala_id = s.id
		WHERE l.certame_id = $1 GROUP BY cp.nome, cp.uf, l.nome ORDER BY 1, 2`, certameID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[Ocupacao])
}

// MapaMarkdown escreve o mapa de ocupação.
func MapaMarkdown(m []Ocupacao) string {
	var b strings.Builder
	b.WriteString("| Cidade | Local | Salas | Lugares | Alocados | Livres | Especiais | Acessíveis |\n|---|---|---:|---:|---:|---:|---:|---:|\n")
	var t Ocupacao
	for _, o := range m {
		fmt.Fprintf(&b, "| %s | %s | %d | %d | %d | %d | %d | %d |\n", o.Cidade, o.Local, o.Salas, o.Capacidade, o.Alocados,
			o.Capacidade-o.Alocados, o.Especiais, o.Acessiveis)
		t.Salas, t.Capacidade, t.Alocados, t.Especiais, t.Acessiveis = t.Salas+o.Salas, t.Capacidade+o.Capacidade,
			t.Alocados+o.Alocados, t.Especiais+o.Especiais, t.Acessiveis+o.Acessiveis
	}
	fmt.Fprintf(&b, "| **Total** | | %d | %d | %d | %d | %d | %d |\n", t.Salas, t.Capacidade, t.Alocados, t.Capacidade-t.Alocados,
		t.Especiais, t.Acessiveis)
	return b.String()
}

// Presenca é uma linha da lista de presença de uma sala.
type Presenca struct {
	Inscricao    int64
	Nome, CPF    string // CPF mascarado: a lista fica na porta da sala
	Cargo        string
	Necessidades []string
}

// ListaDaSala é a lista de presença de uma sala, em ordem de nome.
type ListaDaSala struct {
	Cidade, Local, Endereco, Sala string
	Capacidade                    int
	Presencas                     []Presenca
}

// Listas devolve a lista de presença de cada sala com alguém alocado, em ordem de cidade, local e sala.
func Listas(ctx context.Context, pool *pgxpool.Pool, certameID int64) ([]ListaDaSala, error) {
	rows, err := pool.Query(ctx, `
		SELECT cp.nome || ' (' || cp.uf || ')', l.nome, l.endereco, s.nome, s.capacidade,
		       i.id, c.nome, c.cpf, g.nome, i.necessidades
		FROM alocacao a JOIN inscricao i ON i.id = a.inscricao_id JOIN candidato c ON c.id = i.candidato_id
		JOIN cargo g ON g.id = i.cargo_id JOIN sala s ON s.id = a.sala_id
		JOIN local_prova l ON l.id = s.local_id JOIN cidade_prova cp ON cp.id = l.cidade_prova_id
		WHERE a.certame_id = $1 ORDER BY cp.nome, l.nome, s.nome, c.nome, i.id`, certameID)
	if err != nil {
		return nil, err
	}
	var out []ListaDaSala
	var cidade, local, endereco, sala string
	var capacidade int
	var p Presenca
	var cpf string
	_, err = pgx.ForEachRow(rows, []any{&cidade, &local, &endereco, &sala, &capacidade, &p.Inscricao, &p.Nome, &cpf, &p.Cargo, &p.Necessidades}, func() error {
		if n := len(out); n == 0 || out[n-1].Cidade != cidade || out[n-1].Local != local || out[n-1].Sala != sala {
			out = append(out, ListaDaSala{Cidade: cidade, Local: local, Endereco: endereco, Sala: sala, Capacidade: capacidade})
		}
		p.CPF = "***." + cpf[3:6] + "." + cpf[6:9] + "-**"
		out[len(out)-1].Presencas = append(out[len(out)-1].Presencas, p)
		return nil
	})
	return out, err
}
