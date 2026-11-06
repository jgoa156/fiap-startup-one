// Package arrecadacao é a prestação de contas do certame para a banca e o órgão (02-produto §C). O
// dinheiro não passa pelo Lisura: cai na conta da banca, no prestador dela (03-plano §3). O que o
// Lisura deve é a conta exata do que entrou e do que saiu, por dia, cargo e cidade; a prova de que
// ela fecha; a conferência contra o extrato do prestador; e dias fechados que não mudam depois.
//
// A conta que tem de fechar, centavo por centavo: cada pagamento do certame é confirmado, a mais
// (duplicidade ou depois da expiração) ou com valor divergente; cada devolução é de um pagamento a
// mais ou de um confirmado de certame cancelado. Então
//
//	entrou − saiu − a devolver − em divergência = taxas válidas
//
// onde "taxas válidas" são os pagamentos confirmados sem devolução, contados à parte. Se a igualdade
// falha, há dinheiro sem explicação: um pagamento a mais sem devolução, uma devolução maior que o
// pagado, um pagamento fora de qualquer classe.
package arrecadacao

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"lisura/internal/pagamento"
	"lisura/internal/psp"
)

// Total é uma contagem e uma soma, em centavos.
type Total struct {
	N        int   `json:"n"`
	Centavos int64 `json:"centavos"`
}

// Linha é um recorte da conta: um dia, um cargo ou uma cidade.
type Linha struct {
	Chave   string `json:"chave"`
	Entrou  Total  `json:"entrou"`
	Saiu    Total  `json:"saiu"`
	Validas Total  `json:"taxas_validas"`
}

type Prestacao struct {
	Certame   int64     `json:"certame"`
	Nome      string    `json:"nome"`
	Cancelado bool      `json:"cancelado"`
	GeradaEm  time.Time `json:"gerada_em"`

	Entrou        Total `json:"entrou"`         // todo pagamento registrado do certame
	Saiu          Total `json:"saiu"`           // devoluções concluídas no prestador
	ADevolver     Total `json:"a_devolver"`     // devoluções ainda não concluídas
	EmDivergencia Total `json:"em_divergencia"` // valor pago diferente do cobrado: alguém decide
	Validas       Total `json:"taxas_validas"`  // confirmados sem devolução: a receita do órgão
	// Diferenca = entrou − saiu − a devolver − em divergência − taxas válidas. Tem de ser zero.
	Diferenca int64 `json:"diferenca_centavos"`

	PorDia    []Linha `json:"por_dia"` // dia de registro (pagamento) e de conclusão (devolução)
	PorCargo  []Linha `json:"por_cargo"`
	PorCidade []Linha `json:"por_cidade"`
}

// Os pagamentos do certame, cada um com o dia em que o sistema o registrou e a devolução, se houver.
const pagamentosDoCertame = `
	SELECT p.psp_pagamento_id, p.valor_centavos, p.resultado, p.pago_em, b.id AS cobranca_id,
	       (p.recebido_em AT TIME ZONE 'America/Sao_Paulo')::date AS dia,
	       i.cargo_id, i.cidade_prova_id, d.id AS devolucao_id
	FROM pagamento_recebido p JOIN cobranca b ON b.id = p.cobranca_id JOIN inscricao i ON i.id = b.inscricao_id
	LEFT JOIN devolucao d ON d.psp_pagamento_id = p.psp_pagamento_id
	WHERE i.certame_id = $1`

// As devoluções do certame, com o dia em que o prestador as concluiu.
const devolucoesDoCertame = `
	SELECT d.psp_pagamento_id, d.psp_devolucao_id, d.valor_centavos, d.estado,
	       (d.concluida_em AT TIME ZONE 'America/Sao_Paulo')::date AS dia,
	       i.cargo_id, i.cidade_prova_id
	FROM devolucao d JOIN inscricao i ON i.id = d.inscricao_id
	WHERE i.certame_id = $1`

// Gerar monta a prestação de contas do certame, agora.
func Gerar(ctx context.Context, pool *pgxpool.Pool, certameID int64) (Prestacao, error) {
	p := Prestacao{Certame: certameID, GeradaEm: time.Now()}
	if err := pool.QueryRow(ctx, `SELECT nome, cancelado_em IS NOT NULL FROM certame WHERE id = $1`, certameID).
		Scan(&p.Nome, &p.Cancelado); err != nil {
		return p, err
	}
	nomes, err := nomesDoCertame(ctx, pool, certameID)
	if err != nil {
		return p, err
	}
	dias, cargos, cidades := map[string]*Linha{}, map[string]*Linha{}, map[string]*Linha{}
	linha := func(m map[string]*Linha, k string) *Linha {
		if m[k] == nil {
			m[k] = &Linha{Chave: k}
		}
		return m[k]
	}
	somar := func(t *Total, n int, c int64) { t.N += n; t.Centavos += c }

	// Pagamentos: um GROUPING SETS dá os três recortes e o total numa leitura só.
	rows, err := pool.Query(ctx, `
		WITH pag AS (`+pagamentosDoCertame+`)
		SELECT GROUPING(dia, cargo_id, cidade_prova_id), dia::text, cargo_id, cidade_prova_id,
		       count(*), coalesce(sum(valor_centavos), 0),
		       count(*) FILTER (WHERE resultado = $2 AND devolucao_id IS NULL),
		       coalesce(sum(valor_centavos) FILTER (WHERE resultado = $2 AND devolucao_id IS NULL), 0),
		       count(*) FILTER (WHERE resultado = $3),
		       coalesce(sum(valor_centavos) FILTER (WHERE resultado = $3), 0)
		FROM pag GROUP BY GROUPING SETS ((dia), (cargo_id), (cidade_prova_id), ())`,
		certameID, pagamento.Confirmado, pagamento.ValorDivergente)
	if err != nil {
		return p, err
	}
	var g int
	var dia *string
	var cargo, cidade *int64
	var n, nValidas, nDiv int
	var c, cValidas, cDiv int64
	if _, err := pgx.ForEachRow(rows, []any{&g, &dia, &cargo, &cidade, &n, &c, &nValidas, &cValidas, &nDiv, &cDiv}, func() error {
		var l *Linha
		switch g {
		case 0b011: // agrupado por dia
			l = linha(dias, deref(dia))
		case 0b101:
			l = linha(cargos, nomes.cargo(cargo))
		case 0b110:
			l = linha(cidades, nomes.cidade(cidade))
		case 0b111: // o total
			somar(&p.Entrou, n, c)
			somar(&p.Validas, nValidas, cValidas)
			somar(&p.EmDivergencia, nDiv, cDiv)
			return nil
		}
		somar(&l.Entrou, n, c)
		somar(&l.Validas, nValidas, cValidas)
		return nil
	}); err != nil {
		return p, err
	}

	rows, err = pool.Query(ctx, `
		WITH dev AS (`+devolucoesDoCertame+`)
		SELECT GROUPING(dia, cargo_id, cidade_prova_id), dia::text, cargo_id, cidade_prova_id,
		       count(*) FILTER (WHERE estado = $2), coalesce(sum(valor_centavos) FILTER (WHERE estado = $2), 0),
		       count(*) FILTER (WHERE estado <> $2), coalesce(sum(valor_centavos) FILTER (WHERE estado <> $2), 0)
		FROM dev GROUP BY GROUPING SETS ((dia), (cargo_id), (cidade_prova_id), ())`,
		certameID, pagamento.DevolucaoConcluida)
	if err != nil {
		return p, err
	}
	var nSaiu, nFalta int
	var cSaiu, cFalta int64
	if _, err := pgx.ForEachRow(rows, []any{&g, &dia, &cargo, &cidade, &nSaiu, &cSaiu, &nFalta, &cFalta}, func() error {
		switch g {
		case 0b011:
			if dia != nil { // devolução não concluída não tem dia de saída
				somar(&linha(dias, *dia).Saiu, nSaiu, cSaiu)
			}
		case 0b101:
			somar(&linha(cargos, nomes.cargo(cargo)).Saiu, nSaiu, cSaiu)
		case 0b110:
			somar(&linha(cidades, nomes.cidade(cidade)).Saiu, nSaiu, cSaiu)
		case 0b111:
			somar(&p.Saiu, nSaiu, cSaiu)
			somar(&p.ADevolver, nFalta, cFalta)
		}
		return nil
	}); err != nil {
		return p, err
	}

	p.Diferenca = p.Entrou.Centavos - p.Saiu.Centavos - p.ADevolver.Centavos - p.EmDivergencia.Centavos - p.Validas.Centavos
	p.PorDia, p.PorCargo, p.PorCidade = ordenar(dias), ordenar(cargos), ordenar(cidades)
	return p, nil
}

type nomes struct{ cargos, cidades map[int64]string }

func (n nomes) cargo(id *int64) string {
	if id == nil {
		return "(sem cargo)"
	}
	return n.cargos[*id]
}

func (n nomes) cidade(id *int64) string {
	if id == nil {
		return "(sem cidade)"
	}
	return n.cidades[*id]
}

func nomesDoCertame(ctx context.Context, pool *pgxpool.Pool, certameID int64) (nomes, error) {
	out := nomes{map[int64]string{}, map[int64]string{}}
	for _, q := range []struct {
		sql string
		m   map[int64]string
	}{
		{`SELECT id, nome FROM cargo WHERE certame_id = $1`, out.cargos},
		{`SELECT id, nome || ' (' || uf || ')' FROM cidade_prova WHERE certame_id = $1`, out.cidades},
	} {
		rows, err := pool.Query(ctx, q.sql, certameID)
		if err != nil {
			return out, err
		}
		var id int64
		var nome string
		if _, err := pgx.ForEachRow(rows, []any{&id, &nome}, func() error { q.m[id] = nome; return nil }); err != nil {
			return out, err
		}
	}
	return out, nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func ordenar(m map[string]*Linha) []Linha {
	out := make([]Linha, 0, len(m))
	for _, l := range m {
		out = append(out, *l)
	}
	slices.SortFunc(out, func(a, b Linha) int { return strings.Compare(a.Chave, b.Chave) })
	return out
}

// ---- fechamento do dia ----

var ErrDiaAberto = errors.New("o dia ainda não acabou: só se fecha dia anterior a hoje")

// Fechamento é a conta de um dia, gravada uma vez e nunca mais alterada.
type Fechamento struct {
	Certame    int64     `json:"certame"`
	Dia        string    `json:"dia"` // AAAA-MM-DD, no horário de Brasília
	Entrou     Total     `json:"entrou"`
	Saiu       Total     `json:"saiu"`
	Resumo     string    `json:"resumo"`
	FechadoEm  time.Time `json:"fechado_em"`
	FechadoPor string    `json:"fechado_por"`
}

// contaDoDia refaz, da base, a conta de um dia: totais e o resumo dos ids e valores, em ordem.
func contaDoDia(ctx context.Context, q pgx.Tx, certameID int64, dia string) (Fechamento, error) {
	f := Fechamento{Certame: certameID, Dia: dia}
	// A ordem é byte a byte (COLLATE "C"), qualquer que seja a do banco: na colação en_US (a padrão do
	// Aurora) o "_" dos ids conta diferente, e o mesmo dia teria outro resumo noutro banco, e outro
	// ainda no auditor que o refaz fora dele (exportacao.Verificar).
	rows, err := q.Query(ctx, `
		SELECT tipo, id, valor_centavos FROM (
			SELECT 'P' AS tipo, psp_pagamento_id AS id, valor_centavos FROM (`+pagamentosDoCertame+`) x WHERE dia = $2::date
			UNION ALL
			SELECT 'D', psp_devolucao_id, valor_centavos FROM (`+devolucoesDoCertame+`) y WHERE dia = $2::date AND estado = $3) z
		ORDER BY tipo COLLATE "C", id COLLATE "C"`, certameID, dia, pagamento.DevolucaoConcluida)
	if err != nil {
		return f, err
	}
	h := sha256.New()
	var tipo, id string
	var valor int64
	_, err = pgx.ForEachRow(rows, []any{&tipo, &id, &valor}, func() error {
		fmt.Fprintf(h, "%s|%s|%d\n", tipo, id, valor)
		if tipo == "P" {
			f.Entrou.N++
			f.Entrou.Centavos += valor
		} else {
			f.Saiu.N++
			f.Saiu.Centavos += valor
		}
		return nil
	})
	f.Resumo = hex.EncodeToString(h.Sum(nil))
	return f, err
}

// Fechar grava a conta de um dia já terminado. Fechar de novo devolve o fechamento existente (criado =
// false), sem regravar: o que foi fechado fica como foi.
func Fechar(ctx context.Context, pool *pgxpool.Pool, certameID int64, dia, quem string) (f Fechamento, criado bool, err error) {
	err = pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		var aberto bool
		if err := tx.QueryRow(ctx, `SELECT $1::date >= (now() AT TIME ZONE 'America/Sao_Paulo')::date`, dia).Scan(&aberto); err != nil {
			return err
		}
		if aberto {
			return ErrDiaAberto
		}
		if f, err = contaDoDia(ctx, tx, certameID, dia); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `
			INSERT INTO arrecadacao_fechamento (certame_id, dia, pagamentos, entrou_centavos, devolucoes, saiu_centavos, resumo, fechado_por)
			VALUES ($1, $2::date, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (certame_id, dia) DO NOTHING`,
			certameID, dia, f.Entrou.N, f.Entrou.Centavos, f.Saiu.N, f.Saiu.Centavos, f.Resumo, quem)
		if err != nil {
			return err
		}
		criado = tag.RowsAffected() == 1
		f, err = lerFechamento(ctx, tx, certameID, dia)
		return err
	})
	return f, criado, err
}

func lerFechamento(ctx context.Context, tx pgx.Tx, certameID int64, dia string) (Fechamento, error) {
	f := Fechamento{Certame: certameID}
	err := tx.QueryRow(ctx, `
		SELECT dia::text, pagamentos, entrou_centavos, devolucoes, saiu_centavos, resumo, fechado_em, fechado_por
		FROM arrecadacao_fechamento WHERE certame_id = $1 AND dia = $2::date`, certameID, dia).
		Scan(&f.Dia, &f.Entrou.N, &f.Entrou.Centavos, &f.Saiu.N, &f.Saiu.Centavos, &f.Resumo, &f.FechadoEm, &f.FechadoPor)
	return f, err
}

// FecharAteOntem fecha todo dia terminado em que o certame teve movimento e que ainda não foi fechado.
func FecharAteOntem(ctx context.Context, pool *pgxpool.Pool, certameID int64, quem string) ([]Fechamento, error) {
	rows, err := pool.Query(ctx, `
		SELECT DISTINCT dia::text FROM (
			SELECT dia FROM (`+pagamentosDoCertame+`) x
			UNION SELECT dia FROM (`+devolucoesDoCertame+`) y WHERE dia IS NOT NULL) d
		WHERE dia < (now() AT TIME ZONE 'America/Sao_Paulo')::date
		  AND NOT EXISTS (SELECT 1 FROM arrecadacao_fechamento f WHERE f.certame_id = $1 AND f.dia = d.dia)
		ORDER BY 1`, certameID)
	if err != nil {
		return nil, err
	}
	dias, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, err
	}
	var out []Fechamento
	for _, d := range dias {
		f, criado, err := Fechar(ctx, pool, certameID, d, quem)
		if err != nil {
			return out, err
		}
		if criado {
			out = append(out, f)
		}
	}
	return out, nil
}

// ---- conferência ----

// Extratos é a parte do prestador que a conferência lê.
type Extratos interface {
	Extrato(ctx context.Context, desde, ate time.Time) ([]psp.Pagamento, error)
	ExtratoDevolucoes(ctx context.Context, desde, ate time.Time) ([]psp.Devolucao, error)
}

type Conferencia struct {
	Certame         int64    `json:"certame"`
	Pagamentos      int      `json:"pagamentos_conferidos"`
	Devolucoes      int      `json:"devolucoes_conferidas"`
	DiasFechados    int      `json:"dias_fechados_conferidos"`
	Problemas       []string `json:"problemas"`
	DuracaoSegundos float64  `json:"duracao_s"`
}

// Conferir prova a prestação de contas: a conta fecha; cada dia fechado continua igual; cada pagamento
// e cada devolução concluída daqui está no extrato do prestador com o mesmo valor, e vice-versa.
func Conferir(ctx context.Context, pool *pgxpool.Pool, prestador Extratos, certameID int64) (Conferencia, error) {
	inicio := time.Now()
	c := Conferencia{Certame: certameID, Problemas: []string{}} // [] e não null no JSON: nada achado
	problema := func(f string, a ...any) { c.Problemas = append(c.Problemas, fmt.Sprintf(f, a...)) }

	p, err := Gerar(ctx, pool, certameID)
	if err != nil {
		return c, err
	}
	if p.Diferenca != 0 {
		problema("A CONTA NÃO FECHA: entrou %s, saiu %s, a devolver %s, em divergência %s, taxas válidas %s; diferença %s",
			BRL(p.Entrou.Centavos), BRL(p.Saiu.Centavos), BRL(p.ADevolver.Centavos), BRL(p.EmDivergencia.Centavos),
			BRL(p.Validas.Centavos), BRL(p.Diferenca))
	}

	// Dias fechados: refeitos da base agora, comparados com o que foi gravado.
	err = pgx.BeginTxFunc(ctx, pool, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly}, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `SELECT dia::text FROM arrecadacao_fechamento WHERE certame_id = $1 ORDER BY dia`, certameID)
		if err != nil {
			return err
		}
		dias, err := pgx.CollectRows(rows, pgx.RowTo[string])
		if err != nil {
			return err
		}
		for _, d := range dias {
			gravado, err := lerFechamento(ctx, tx, certameID, d)
			if err != nil {
				return err
			}
			agora, err := contaDoDia(ctx, tx, certameID, d)
			if err != nil {
				return err
			}
			if agora.Resumo != gravado.Resumo {
				problema("DIA %s FECHADO MUDOU: fechado com %d pagamentos (%s) e %d devoluções (%s); agora %d (%s) e %d (%s)",
					d, gravado.Entrou.N, BRL(gravado.Entrou.Centavos), gravado.Saiu.N, BRL(gravado.Saiu.Centavos),
					agora.Entrou.N, BRL(agora.Entrou.Centavos), agora.Saiu.N, BRL(agora.Saiu.Centavos))
			}
			c.DiasFechados++
		}
		return nil
	})
	if err != nil {
		return c, err
	}

	// Contra o prestador. A janela vai da primeira cobrança do certame até agora: pega também o
	// pagamento que o prestador tem e nós não (aviso perdido que a conciliação ainda não achou).
	type nosso struct {
		valor int64
		ok    bool
	}
	var desde *time.Time
	if err := pool.QueryRow(ctx, `SELECT min(b.criada_em) FROM cobranca b JOIN inscricao i ON i.id = b.inscricao_id
		WHERE i.certame_id = $1`, certameID).Scan(&desde); err != nil {
		return c, err
	}
	if desde == nil {
		c.DuracaoSegundos = time.Since(inicio).Seconds()
		return c, nil // nenhuma cobrança: nada a conferir no prestador
	}
	ate := time.Now().Add(time.Minute)

	cobrancas := map[int64]bool{}
	pags := map[string]*nosso{}
	devs := map[string]*nosso{} // id da devolução no prestador
	devPorRef := map[string]bool{}
	rows, err := pool.Query(ctx, `SELECT b.id FROM cobranca b JOIN inscricao i ON i.id = b.inscricao_id WHERE i.certame_id = $1`, certameID)
	if err != nil {
		return c, err
	}
	var cob int64
	if _, err := pgx.ForEachRow(rows, []any{&cob}, func() error { cobrancas[cob] = true; return nil }); err != nil {
		return c, err
	}
	rows, err = pool.Query(ctx, `SELECT psp_pagamento_id, valor_centavos FROM (`+pagamentosDoCertame+`) x`, certameID)
	if err != nil {
		return c, err
	}
	var id string
	var valor int64
	if _, err := pgx.ForEachRow(rows, []any{&id, &valor}, func() error { pags[id] = &nosso{valor: valor}; return nil }); err != nil {
		return c, err
	}
	rows, err = pool.Query(ctx, `SELECT psp_pagamento_id, psp_devolucao_id, valor_centavos, estado FROM (`+devolucoesDoCertame+`) y`, certameID)
	if err != nil {
		return c, err
	}
	var pagID string
	var devID *string
	var estado string
	if _, err := pgx.ForEachRow(rows, []any{&pagID, &devID, &valor, &estado}, func() error {
		devPorRef["dev-"+pagID] = estado == pagamento.DevolucaoConcluida
		if devID != nil {
			devs[*devID] = &nosso{valor: valor}
		}
		return nil
	}); err != nil {
		return c, err
	}

	extrato, err := prestador.Extrato(ctx, *desde, ate)
	if err != nil {
		return c, err
	}
	for _, e := range extrato {
		if !cobrancas[e.Referencia] {
			continue // de outro certame
		}
		n := pags[e.ID]
		switch {
		case n == nil:
			problema("NO PRESTADOR, SEM REGISTRO AQUI: pagamento %s de %s", e.ID, BRL(int64(e.ValorCentavos)))
		case n.valor != int64(e.ValorCentavos):
			problema("VALOR DIFERENTE DO PRESTADOR: pagamento %s, aqui %s, lá %s", e.ID, BRL(n.valor), BRL(int64(e.ValorCentavos)))
			n.ok = true
		default:
			n.ok = true
		}
	}
	for id, n := range pags {
		c.Pagamentos++
		if !n.ok {
			problema("AQUI, SEM REGISTRO NO PRESTADOR: pagamento %s de %s", id, BRL(n.valor))
		}
	}

	saidas, err := prestador.ExtratoDevolucoes(ctx, *desde, ate)
	if err != nil {
		return c, err
	}
	for _, e := range saidas {
		// De quem é a devolução, decide o pagamento, não a nossa linha de devolução: se ela sumiu daqui,
		// a devolução do prestador tem de aparecer como sem registro, não como de outro certame.
		if pags[e.PagamentoID] == nil {
			continue // de outro certame
		}
		concluida, temLinha := devPorRef[e.Referencia]
		n := devs[e.ID]
		switch {
		case n == nil && temLinha && !concluida:
			problema("DEVOLVIDO NO PRESTADOR, AINDA NÃO CONCLUÍDO AQUI: %s de %s (a fila conclui na próxima tentativa)", e.Referencia, BRL(int64(e.ValorCentavos)))
		case n == nil:
			problema("DEVOLUÇÃO DO PRESTADOR SEM REGISTRO AQUI: %s de %s", e.ID, BRL(int64(e.ValorCentavos)))
		case n.valor != int64(e.ValorCentavos):
			problema("DEVOLUÇÃO COM VALOR DIFERENTE DO PRESTADOR: %s, aqui %s, lá %s", e.ID, BRL(n.valor), BRL(int64(e.ValorCentavos)))
			n.ok = true
		default:
			n.ok = true
		}
	}
	for id, n := range devs {
		c.Devolucoes++
		if !n.ok {
			problema("DEVOLUÇÃO CONCLUÍDA AQUI, SEM REGISTRO NO PRESTADOR: %s de %s", id, BRL(n.valor))
		}
	}
	slices.Sort(c.Problemas)
	c.DuracaoSegundos = time.Since(inicio).Seconds()
	return c, nil
}

// BRL escreve centavos como R$ 1.234,56.
func BRL(c int64) string {
	sinal := ""
	if c < 0 {
		sinal, c = "-", -c
	}
	inteiro := fmt.Sprint(c / 100)
	var b strings.Builder
	for i, r := range inteiro {
		if i > 0 && (len(inteiro)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}
	return fmt.Sprintf("%sR$ %s,%02d", sinal, b.String(), c%100)
}
