package resultado

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"lisura/internal/inscricao"
)

var (
	ErrRecursoInvalido  = errors.New("recurso inválido")
	ErrRecursoRepetido  = errors.New("já existe recurso desta inscrição para esta questão")
	ErrSemPreliminar    = errors.New("não há gabarito preliminar aberto a recurso")
	ErrJaDecidida       = errors.New("questão já decidida")
	ErrDecisoesPendente = errors.New("há questões com recurso sem decisão")
)

// Fundamentos aceitos, os mesmos da tabela: o candidato escolhe um, e o argumento vai junto.
var Fundamentos = []string{"gabarito_errado", "mais_de_uma_correta", "nenhuma_correta", "fora_do_conteudo", "enunciado_ambiguo"}

type Pedido struct {
	Numero         int    `json:"numero"`
	Fundamento     string `json:"fundamento"`
	RespostaPedida string `json:"resposta_pedida,omitempty"` // só com gabarito_errado
	Argumento      string `json:"argumento"`
}

// Recorrer registra o recurso do candidato contra o gabarito preliminar em vigor do cargo da
// inscrição. Exige a inscrição homologada, a janela de recurso aberta ("gabarito" no cronograma) e
// a questão existente; um recurso por questão.
func Recorrer(ctx context.Context, pool *pgxpool.Pool, inscricaoID, candidatoID int64, p Pedido) (id int64, err error) {
	p.RespostaPedida = strings.ToUpper(strings.TrimSpace(p.RespostaPedida))
	p.Argumento = strings.TrimSpace(p.Argumento)
	switch {
	case !contem(Fundamentos, p.Fundamento):
		return 0, fmt.Errorf("%w: fundamento %q", ErrRecursoInvalido, p.Fundamento)
	case (p.Fundamento == "gabarito_errado") != (p.RespostaPedida != ""):
		return 0, fmt.Errorf("%w: a resposta pedida acompanha só o fundamento gabarito_errado", ErrRecursoInvalido)
	case p.RespostaPedida != "" && !strings.Contains("ABCDE", p.RespostaPedida) || len(p.RespostaPedida) > 1:
		return 0, fmt.Errorf("%w: resposta %q", ErrRecursoInvalido, p.RespostaPedida)
	case p.Argumento == "" || len([]rune(p.Argumento)) > 4000:
		return 0, fmt.Errorf("%w: argumento de 1 a 4000 caracteres", ErrRecursoInvalido)
	}
	err = pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		var cargo, certame int64
		var estado string
		err := tx.QueryRow(ctx, `SELECT cargo_id, certame_id, estado FROM inscricao WHERE id = $1 AND candidato_id = $2`,
			inscricaoID, candidatoID).Scan(&cargo, &certame, &estado)
		if errors.Is(err, pgx.ErrNoRows) {
			return inscricao.ErrNaoEncontrada // de outro candidato também: não diz que existe
		}
		if err != nil {
			return err
		}
		if estado != string(inscricao.Homologada) {
			return fmt.Errorf("%w: só concorre quem está na lista definitiva de inscritos", inscricao.ErrTransicaoInvalida)
		}
		var aberta bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM certame_fase
			               WHERE certame_id = $1 AND fase = 'gabarito' AND abre_em <= now() AND now() < fecha_em)`,
			certame).Scan(&aberta); err != nil {
			return err
		}
		if !aberta {
			return fmt.Errorf("%w: recurso contra o gabarito", inscricao.ErrForaDoPrazo)
		}
		// Trava compartilhada: recursos não esperam uns pelos outros, mas esperam uma consolidação em
		// andamento (que trava o cargo de forma exclusiva) — e depois dela veem o definitivo e recusam.
		if _, err := tx.Exec(ctx, `SELECT 1 FROM cargo WHERE id = $1 FOR SHARE`, cargo); err != nil {
			return err
		}
		versao, tipo, err := gabaritoAtual(ctx, tx, cargo)
		if err != nil {
			return err
		}
		if tipo != "preliminar" {
			return ErrSemPreliminar
		}
		var resp *string
		if p.RespostaPedida != "" {
			resp = &p.RespostaPedida
		}
		err = tx.QueryRow(ctx, `
			INSERT INTO recurso_questao (inscricao_id, cargo_id, gabarito_versao, numero, fundamento, resposta_pedida, argumento)
			VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
			inscricaoID, cargo, versao, p.Numero, p.Fundamento, resp, p.Argumento).Scan(&id)
		var pg *pgconn.PgError
		if errors.As(err, &pg) {
			switch pg.Code {
			case "23505":
				return ErrRecursoRepetido
			case "23503":
				return fmt.Errorf("%w: questão %d não existe na prova", ErrRecursoInvalido, p.Numero)
			}
		}
		return err
	})
	return id, err
}

// MeuRecurso é o recurso com a resposta que ele recebeu: a decisão da questão, se já houver.
type MeuRecurso struct {
	ID             int64      `json:"id"`
	Numero         int        `json:"numero"`
	Fundamento     string     `json:"fundamento"`
	RespostaPedida *string    `json:"resposta_pedida,omitempty"`
	CriadoEm       time.Time  `json:"criado_em"`
	Decisao        *string    `json:"decisao"` // null: em análise
	NovaResposta   *string    `json:"nova_resposta,omitempty"`
	Fundamentacao  *string    `json:"fundamentacao,omitempty"`
	JulgadoEm      *time.Time `json:"julgado_em,omitempty"`
}

func MeusRecursos(ctx context.Context, pool *pgxpool.Pool, inscricaoID, candidatoID int64) ([]MeuRecurso, error) {
	rows, err := pool.Query(ctx, `
		SELECT r.id, r.numero, r.fundamento, r.resposta_pedida, r.criado_em,
		       d.decisao, d.nova_resposta, d.fundamentacao, d.julgado_em
		  FROM recurso_questao r
		  JOIN inscricao i ON i.id = r.inscricao_id AND i.candidato_id = $2
		  LEFT JOIN decisao_questao d
		         ON d.cargo_id = r.cargo_id AND d.gabarito_versao = r.gabarito_versao AND d.numero = r.numero
		 WHERE r.inscricao_id = $1 ORDER BY r.numero`, inscricaoID, candidatoID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[MeuRecurso])
}

// Grupo é o que a banca examinadora vê: uma linha por questão com recurso, com quantos pediram o quê.
type Grupo struct {
	Numero        int            `json:"numero"`
	Disciplina    string         `json:"disciplina"`
	Recursos      int            `json:"recursos"`
	PorFundamento map[string]int `json:"por_fundamento"`
	Pedidas       map[string]int `json:"respostas_pedidas"` // em gabarito_errado: quantos pediram cada letra
	Decisao       *string        `json:"decisao"`
}

// Agrupar devolve os recursos do gabarito em vigor do cargo agrupados por questão: 500 recursos sobre
// a mesma questão viram uma linha, uma análise e uma decisão.
func Agrupar(ctx context.Context, pool *pgxpool.Pool, cargoID int64) (versao int, grupos []Grupo, err error) {
	err = pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		v, _, err := gabaritoAtual(ctx, tx, cargoID)
		if err != nil {
			return err
		}
		versao = v
		rows, err := tx.Query(ctx, `
			SELECT r.numero, q.disciplina, r.fundamento, coalesce(r.resposta_pedida, ''), count(*), d.decisao
			  FROM recurso_questao r
			  JOIN questao q ON q.cargo_id = r.cargo_id AND q.numero = r.numero
			  LEFT JOIN decisao_questao d
			         ON d.cargo_id = r.cargo_id AND d.gabarito_versao = r.gabarito_versao AND d.numero = r.numero
			 WHERE r.cargo_id = $1 AND r.gabarito_versao = $2
			 GROUP BY r.numero, q.disciplina, r.fundamento, r.resposta_pedida, d.decisao
			 ORDER BY r.numero`, cargoID, v)
		if err != nil {
			return err
		}
		var numero, n int
		var disc, fund, pedida string
		var dec *string
		_, err = pgx.ForEachRow(rows, []any{&numero, &disc, &fund, &pedida, &n, &dec}, func() error {
			if len(grupos) == 0 || grupos[len(grupos)-1].Numero != numero {
				grupos = append(grupos, Grupo{Numero: numero, Disciplina: disc, Decisao: dec,
					PorFundamento: map[string]int{}, Pedidas: map[string]int{}})
			}
			g := &grupos[len(grupos)-1]
			g.Recursos += n
			g.PorFundamento[fund] += n
			if pedida != "" {
				g.Pedidas[pedida] += n
			}
			return nil
		})
		return err
	})
	return versao, grupos, err
}

// Decidir julga uma questão do gabarito preliminar em vigor: mantém, altera para outra letra ou anula.
// Vale para todos os recursos da questão; decisão gravada não muda.
func Decidir(ctx context.Context, pool *pgxpool.Pool, cargoID int64, numero int, decisao, novaResposta, fundamentacao, examinador string) error {
	novaResposta = strings.ToUpper(strings.TrimSpace(novaResposta))
	switch {
	case decisao != "mantem" && decisao != "altera" && decisao != "anula":
		return fmt.Errorf("%w: decisão %q", ErrRecursoInvalido, decisao)
	case (decisao == "altera") != (novaResposta != ""):
		return fmt.Errorf("%w: a nova resposta acompanha só a decisão altera", ErrRecursoInvalido)
	case novaResposta != "" && (len(novaResposta) != 1 || !strings.Contains("ABCDE", novaResposta)):
		return fmt.Errorf("%w: resposta %q", ErrRecursoInvalido, novaResposta)
	case strings.TrimSpace(fundamentacao) == "" || examinador == "":
		return fmt.Errorf("%w: fundamentação e examinador são obrigatórios", ErrRecursoInvalido)
	}
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		if err := travarCargo(ctx, tx, cargoID); err != nil {
			return err
		}
		versao, tipo, err := gabaritoAtual(ctx, tx, cargoID)
		if err != nil {
			return err
		}
		if tipo != "preliminar" {
			return ErrSemPreliminar
		}
		var atual string
		if err := tx.QueryRow(ctx, `SELECT substr(respostas, $3, 1) FROM gabarito WHERE cargo_id = $1 AND versao = $2`,
			cargoID, versao, numero).Scan(&atual); err != nil {
			return err
		}
		if decisao == "altera" && novaResposta == atual {
			return fmt.Errorf("%w: a questão %d já tem a resposta %s", ErrRecursoInvalido, numero, atual)
		}
		var nova *string
		if novaResposta != "" {
			nova = &novaResposta
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO decisao_questao (cargo_id, gabarito_versao, numero, decisao, nova_resposta, fundamentacao, julgado_por)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`, cargoID, versao, numero, decisao, nova, fundamentacao, examinador)
		var pg *pgconn.PgError
		if errors.As(err, &pg) {
			switch pg.Code {
			case "23505":
				return ErrJaDecidida
			case "23503":
				return fmt.Errorf("%w: questão %d não existe na prova", ErrRecursoInvalido, numero)
			}
		}
		return err
	})
}

// Consolidar fecha a fase de recursos do cargo: aplica as decisões ao gabarito preliminar e publica o
// definitivo como versão nova. Recusa enquanto houver questão com recurso e sem decisão — publicar o
// definitivo deixaria esses recursos sem resposta.
//
// O cargo fica travado do começo ao fim: duas consolidações juntas não publicam dois definitivos, e
// nenhum recurso ou decisão entra no meio (Recorrer e Decidir esperam a trava).
func Consolidar(ctx context.Context, pool *pgxpool.Pool, cargoID int64) (versao int, alteradas, anuladas int, err error) {
	err = pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		if err := travarCargo(ctx, tx, cargoID); err != nil {
			return err
		}
		v, tipo, err := gabaritoAtual(ctx, tx, cargoID)
		if err != nil {
			return err
		}
		if tipo != "preliminar" {
			return ErrSemPreliminar
		}
		rows, err := tx.Query(ctx, `
			SELECT DISTINCT r.numero FROM recurso_questao r
			 WHERE r.cargo_id = $1 AND r.gabarito_versao = $2
			   AND NOT EXISTS (SELECT 1 FROM decisao_questao d
			                   WHERE d.cargo_id = r.cargo_id AND d.gabarito_versao = r.gabarito_versao AND d.numero = r.numero)
			 ORDER BY 1`, cargoID, v)
		if err != nil {
			return err
		}
		pendentes, err := pgx.CollectRows(rows, pgx.RowTo[int])
		if err != nil {
			return err
		}
		if len(pendentes) > 0 {
			return fmt.Errorf("%w: questões %v", ErrDecisoesPendente, pendentes)
		}
		var preliminar string
		if err := tx.QueryRow(ctx, `SELECT respostas FROM gabarito WHERE cargo_id = $1 AND versao = $2`, cargoID, v).Scan(&preliminar); err != nil {
			return err
		}
		b := []byte(preliminar)
		drows, err := tx.Query(ctx, `
			SELECT numero, decisao, coalesce(nova_resposta, '') FROM decisao_questao
			 WHERE cargo_id = $1 AND gabarito_versao = $2`, cargoID, v)
		if err != nil {
			return err
		}
		var numero int
		var dec, nova string
		_, err = pgx.ForEachRow(drows, []any{&numero, &dec, &nova}, func() error {
			switch dec {
			case "altera":
				b[numero-1] = nova[0]
				alteradas++
			case "anula":
				b[numero-1] = '*'
				anuladas++
			}
			return nil
		})
		if err != nil {
			return err
		}
		versao, err = publicarGabarito(ctx, tx, cargoID, "definitivo", string(b),
			fmt.Sprintf("Gabarito definitivo após recursos: %d alterada(s), %d anulada(s)", alteradas, anuladas))
		return err
	})
	if err != nil {
		return 0, 0, 0, err
	}
	return versao, alteradas, anuladas, nil
}

func gabaritoAtual(ctx context.Context, tx pgx.Tx, cargoID int64) (versao int, tipo string, err error) {
	err = tx.QueryRow(ctx, `SELECT versao, tipo FROM gabarito WHERE cargo_id = $1 ORDER BY versao DESC LIMIT 1`,
		cargoID).Scan(&versao, &tipo)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, "", ErrSemGabarito
	}
	return versao, tipo, err
}

func contem(lista []string, s string) bool {
	for _, x := range lista {
		if x == s {
			return true
		}
	}
	return false
}
