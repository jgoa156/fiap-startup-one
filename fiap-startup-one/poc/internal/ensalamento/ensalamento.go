// Package ensalamento aloca cada inscrição homologada numa sala da cidade de prova que ela escolheu
// (02-produto §7 F; "ensalamento simples por capacidade" é do MVP, §8). Respeita a capacidade de cada
// sala e o atendimento especial:
//
//	ledor ou transcritor          → sala individual (um candidato e quem o atende)
//	tempo adicional, prova ampliada → sala especial (o grupo fica mais tempo, com fiscal próprio)
//	sala de fácil acesso          → sala acessível (térreo, rampa ou elevador)
//	lactante                      → local com espaço para amamentar
//
// Os mais restritos escolhem primeiro; dentro de cada grupo, a ordem é cargo e nome, e a sala é a
// primeira com lugar, na ordem de local e nome: o mesmo dado de entrada dá sempre a mesma alocação.
// Falta de lugar não aloca ninguém: diz quantos lugares de que tipo faltam em que cidade.
//
// Ensalar de novo mantém quem já tem sala válida (o candidato não troca de sala porque alguém entrou
// por recurso) e aloca só quem chegou, no lugar que sobrou. Cada ensalamento é um ato publicado, com
// o SHA-256 da alocação inteira, e não muda depois.
package ensalamento

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"lisura/internal/publicacao"
)

// Requisito é o que a sala de um candidato precisa ter.
type Requisito struct {
	Tipo        string // comum, especial ou individual
	Acessivel   bool
	Amamentacao bool
}

// RequisitoDe traduz o atendimento especial pedido na inscrição.
func RequisitoDe(necessidades []string) Requisito {
	r := Requisito{Tipo: "comum"}
	for _, n := range necessidades {
		switch n {
		case "ledor", "transcritor":
			r.Tipo = "individual"
		case "tempo_adicional", "prova_ampliada":
			if r.Tipo != "individual" {
				r.Tipo = "especial"
			}
		case "sala_acessivel":
			r.Acessivel = true
		case "lactante":
			r.Amamentacao = true
		}
	}
	return r
}

func (r Requisito) String() string {
	s := "sala " + r.Tipo
	if r.Acessivel {
		s += " acessível"
	}
	if r.Amamentacao {
		s += " em local com amamentação"
	}
	return s
}

// prioridade: quem tem menos salas possíveis escolhe primeiro.
func (r Requisito) prioridade() int {
	switch {
	case r.Tipo == "individual":
		return 0
	case r.Tipo == "especial":
		return 1
	case r.Acessivel:
		return 2
	case r.Amamentacao:
		return 3
	}
	return 4
}

// Sala é uma sala com o que importa para alocar.
type Sala struct {
	ID, Local, Cidade int64
	LocalNome, Nome   string
	Capacidade        int
	Tipo              string
	Acessivel         bool
	Amamentacao       bool // do local
}

// Atende diz se a sala serve a quem precisa de r. A sala especial ou individual não recebe quem não
// precisa dela: tem fiscal e tempo próprios.
func (s Sala) Atende(r Requisito) bool {
	return s.Tipo == r.Tipo && (!r.Acessivel || s.Acessivel) && (!r.Amamentacao || s.Amamentacao)
}

// Falta é lugar que não existe.
type Falta struct {
	Cidade    string `json:"cidade"`
	Requisito string `json:"requisito"`
	Faltam    int    `json:"faltam"`
}

// ErrFaltaLugar: o ensalamento não aloca ninguém quando falta lugar para alguém.
type ErrFaltaLugar struct{ Faltas []Falta }

func (e *ErrFaltaLugar) Error() string {
	var partes []string
	for _, f := range e.Faltas {
		partes = append(partes, fmt.Sprintf("%s: faltam %d lugares em %s", f.Cidade, f.Faltam, f.Requisito))
	}
	return "falta lugar: " + strings.Join(partes, "; ")
}

var ErrSemSalas = errors.New("o certame não tem salas cadastradas")

// Versao é um ensalamento publicado.
type Versao struct {
	Certame   int64         `json:"certame"`
	Versao    int           `json:"versao"`
	Alocadas  int           `json:"alocadas"`
	Mantidas  int           `json:"mantidas"`
	Novas     int           `json:"novas"`
	Removidas int           `json:"removidas"`
	Resumo    string        `json:"resumo"`
	Nova      bool          `json:"nova"` // false: nada mudou desde a última versão, que é devolvida
	Duracao   time.Duration `json:"duracao"`
}

type candidato struct {
	inscricao int64
	cidade    *int64
	cargo     string
	nome      string
	req       Requisito
}

func salasDoCertame(ctx context.Context, q pgx.Tx, certameID int64) ([]Sala, error) {
	rows, err := q.Query(ctx, `
		SELECT s.id, l.id, l.cidade_prova_id, l.nome, s.nome, s.capacidade, s.tipo, s.acessivel, l.amamentacao
		FROM sala s JOIN local_prova l ON l.id = s.local_id
		WHERE l.certame_id = $1 ORDER BY l.cidade_prova_id, l.nome, s.nome, s.id`, certameID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Sala, error) {
		var s Sala
		err := r.Scan(&s.ID, &s.Local, &s.Cidade, &s.LocalNome, &s.Nome, &s.Capacidade, &s.Tipo, &s.Acessivel, &s.Amamentacao)
		return s, err
	})
}

// Ensalar aloca o certame. Tudo numa transação, com uma trava por certame: dois ensalamentos ao
// mesmo tempo não se misturam, e um que falha não deixa nada pela metade.
func Ensalar(ctx context.Context, pool *pgxpool.Pool, certameID int64, quem, motivo string) (Versao, error) {
	inicio := time.Now()
	v := Versao{Certame: certameID}
	if quem == "" || motivo == "" {
		return v, errors.New("ensalar: quem e motivo são obrigatórios (vão para o ato publicado)")
	}
	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1, $2)`, int32(0x456e73), int32(certameID)); err != nil {
			return err
		}
		salas, err := salasDoCertame(ctx, tx, certameID)
		if err != nil {
			return err
		}
		if len(salas) == 0 {
			return ErrSemSalas
		}
		porID := make(map[int64]*Sala, len(salas))
		livres := make(map[int64]int, len(salas))
		porCidade := map[int64][]*Sala{}
		for i := range salas {
			s := &salas[i]
			porID[s.ID], livres[s.ID] = s, s.Capacidade
			porCidade[s.Cidade] = append(porCidade[s.Cidade], s)
		}

		rows, err := tx.Query(ctx, `
			SELECT i.id, i.cidade_prova_id, g.nome, c.nome, i.necessidades
			FROM inscricao i JOIN cargo g ON g.id = i.cargo_id JOIN candidato c ON c.id = i.candidato_id
			WHERE i.certame_id = $1 AND i.estado = 'HOMOLOGADA'`, certameID)
		if err != nil {
			return err
		}
		elegiveis := map[int64]*candidato{}
		var c candidato
		var nec []string
		if _, err := pgx.ForEachRow(rows, []any{&c.inscricao, &c.cidade, &c.cargo, &c.nome, &nec}, func() error {
			x := c
			x.req = RequisitoDe(nec)
			elegiveis[x.inscricao] = &x
			return nil
		}); err != nil {
			return err
		}

		// Quem já tem sala e continua podendo usá-la, fica nela.
		rows, err = tx.Query(ctx, `SELECT inscricao_id, sala_id FROM alocacao WHERE certame_id = $1`, certameID)
		if err != nil {
			return err
		}
		mantida := map[int64]bool{}
		var remover []int64
		var insc, salaID int64
		if _, err := pgx.ForEachRow(rows, []any{&insc, &salaID}, func() error {
			e, s := elegiveis[insc], porID[salaID]
			if e != nil && s != nil && e.cidade != nil && *e.cidade == s.Cidade && s.Atende(e.req) && livres[salaID] > 0 {
				mantida[insc] = true
				livres[salaID]--
			} else {
				remover = append(remover, insc)
			}
			return nil
		}); err != nil {
			return err
		}

		var novos []*candidato
		for id, e := range elegiveis {
			if !mantida[id] {
				novos = append(novos, e)
			}
		}
		slices.SortFunc(novos, func(a, b *candidato) int {
			return cmp.Or(cmp.Compare(a.req.prioridade(), b.req.prioridade()), strings.Compare(a.cargo, b.cargo),
				strings.Compare(a.nome, b.nome), cmp.Compare(a.inscricao, b.inscricao))
		})

		// Um cursor por cidade e requisito: as salas são varridas em ordem, uma vez, e quem vem depois
		// continua de onde o anterior parou.
		cursor := map[lugar]int{}
		faltas := map[lugar]int{}
		destino := make(map[int64]int64, len(novos))
		semCidade := 0
		for _, e := range novos {
			if e.cidade == nil {
				semCidade++
				continue
			}
			k := lugar{*e.cidade, e.req}
			lista := porCidade[*e.cidade]
			i := cursor[k]
			for i < len(lista) && (!lista[i].Atende(e.req) || livres[lista[i].ID] == 0) {
				i++
			}
			cursor[k] = i
			if i == len(lista) {
				faltas[k]++
				continue
			}
			destino[e.inscricao] = lista[i].ID
			livres[lista[i].ID]--
		}
		if len(faltas) > 0 || semCidade > 0 {
			return erroDeFalta(ctx, tx, faltas, semCidade)
		}

		var anterior int
		if err := tx.QueryRow(ctx, `SELECT coalesce(max(versao), 0) FROM ensalamento_versao WHERE certame_id = $1`,
			certameID).Scan(&anterior); err != nil {
			return err
		}
		v.Mantidas, v.Novas, v.Removidas, v.Alocadas = len(mantida), len(destino), len(remover), len(mantida)+len(destino)
		if anterior > 0 && v.Novas == 0 && v.Removidas == 0 {
			// Nada mudou: o ato publicado continua sendo o último.
			return tx.QueryRow(ctx, `SELECT versao, resumo FROM ensalamento_versao WHERE certame_id = $1 AND versao = $2`,
				certameID, anterior).Scan(&v.Versao, &v.Resumo)
		}
		v.Versao, v.Nova = anterior+1, true

		if len(remover) > 0 {
			if _, err := tx.Exec(ctx, `DELETE FROM alocacao WHERE inscricao_id = ANY($1)`, remover); err != nil {
				return err
			}
		}
		linhas := make([][]any, 0, len(destino))
		for _, e := range novos {
			linhas = append(linhas, []any{e.inscricao, certameID, destino[e.inscricao], v.Versao})
		}
		if _, err := tx.CopyFrom(ctx, pgx.Identifier{"alocacao"}, []string{"inscricao_id", "certame_id", "sala_id", "versao"},
			pgx.CopyFromRows(linhas)); err != nil {
			return err
		}
		if v.Resumo, err = resumoAtual(ctx, tx, certameID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO ensalamento_versao (certame_id, versao, alocadas, mantidas, novas, removidas, resumo, motivo, criado_por)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			certameID, v.Versao, v.Alocadas, v.Mantidas, v.Novas, v.Removidas, v.Resumo, motivo, quem); err != nil {
			return err
		}
		// O cartão de quem ganhou ou perdeu sala é republicado (com local e sala, ou sem).
		return publicacao.EnfileirarCartoesLote(ctx, tx, append(slices.Collect(maps.Keys(destino)), remover...))
	})
	v.Duracao = time.Since(inicio)
	return v, err
}

// lugar é uma cidade e um requisito: a unidade em que se procura sala e em que se conta a falta.
type lugar struct {
	cidade int64
	req    Requisito
}

func erroDeFalta(ctx context.Context, tx pgx.Tx, faltas map[lugar]int, semCidade int) error {
	e := &ErrFaltaLugar{}
	for k, n := range faltas {
		var nome string
		if err := tx.QueryRow(ctx, `SELECT nome || ' (' || uf || ')' FROM cidade_prova WHERE id = $1`, k.cidade).Scan(&nome); err != nil {
			return err
		}
		e.Faltas = append(e.Faltas, Falta{Cidade: nome, Requisito: k.req.String(), Faltam: n})
	}
	if semCidade > 0 {
		e.Faltas = append(e.Faltas, Falta{Cidade: "(sem cidade de prova)", Requisito: "qualquer sala", Faltam: semCidade})
	}
	slices.SortFunc(e.Faltas, func(a, b Falta) int {
		return cmp.Or(strings.Compare(a.Cidade, b.Cidade), strings.Compare(a.Requisito, b.Requisito))
	})
	return e
}

// resumoAtual é o SHA-256 da alocação inteira do certame, em ordem de inscrição.
func resumoAtual(ctx context.Context, q pgx.Tx, certameID int64) (string, error) {
	rows, err := q.Query(ctx, `SELECT inscricao_id, sala_id FROM alocacao WHERE certame_id = $1 ORDER BY inscricao_id`, certameID)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	var insc, sala int64
	_, err = pgx.ForEachRow(rows, []any{&insc, &sala}, func() error {
		fmt.Fprintf(h, "%d|%d\n", insc, sala)
		return nil
	})
	return hex.EncodeToString(h.Sum(nil)), err
}
