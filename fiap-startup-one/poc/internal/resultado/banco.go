package resultado

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"lisura/internal/fila"
	"lisura/internal/inscricao"
	"lisura/internal/publicacao"
)

var (
	ErrProvaFixa          = errors.New("a prova não muda depois do primeiro gabarito")
	ErrSemProva           = errors.New("cargo sem prova cadastrada")
	ErrSemGabarito        = errors.New("cargo sem gabarito publicado")
	ErrGabaritoPreliminar = errors.New("resultado definitivo exige gabarito definitivo em todos os cargos")
	ErrGabaritoInvalido   = errors.New("gabarito inválido")
)

// DefinirProva cadastra as questões do cargo, na ordem. Depois do primeiro gabarito a prova fica
// fixa: mudar o número de questões invalidaria as folhas já lidas.
func DefinirProva(ctx context.Context, pool *pgxpool.Pool, cargoID int64, prova []Questao) error {
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		if err := travarCargo(ctx, tx, cargoID); err != nil {
			return err
		}
		var temGabarito bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM gabarito WHERE cargo_id = $1)`, cargoID).Scan(&temGabarito); err != nil {
			return err
		}
		if temGabarito {
			return ErrProvaFixa
		}
		if _, err := tx.Exec(ctx, `DELETE FROM questao WHERE cargo_id = $1`, cargoID); err != nil {
			return err
		}
		for i, q := range prova {
			if _, err := tx.Exec(ctx, `INSERT INTO questao (cargo_id, numero, disciplina, peso) VALUES ($1, $2, $3, $4)`,
				cargoID, i+1, q.Disciplina, numeric(q.Peso)); err != nil {
				return err
			}
		}
		return nil
	})
}

// PublicarGabarito grava uma versão nova do gabarito do cargo e pede a publicação estática. Versão
// antiga nunca muda: a correção de um erro é outra versão, com motivo.
func PublicarGabarito(ctx context.Context, pool *pgxpool.Pool, cargoID int64, tipo, respostas, motivo string) (versao int, err error) {
	err = pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		if err := travarCargo(ctx, tx, cargoID); err != nil {
			return err
		}
		versao, err = publicarGabarito(ctx, tx, cargoID, tipo, respostas, motivo)
		return err
	})
	return versao, err
}

// publicarGabarito exige o cargo já travado pela transação.
func publicarGabarito(ctx context.Context, tx pgx.Tx, cargoID int64, tipo, respostas, motivo string) (versao int, err error) {
	if tipo != "preliminar" && tipo != "definitivo" {
		return 0, fmt.Errorf("%w: tipo %q", ErrGabaritoInvalido, tipo)
	}
	if motivo == "" {
		return 0, fmt.Errorf("%w: motivo obrigatório", ErrGabaritoInvalido)
	}
	certameID, err := certameDoCargo(ctx, tx, cargoID)
	if err != nil {
		return 0, err
	}
	if tipo == "preliminar" {
		// Depois do definitivo, só outro definitivo (retificação, por decisão judicial, por exemplo):
		// voltar a preliminar reabriria recursos já respondidos.
		var temDefinitivo bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM gabarito WHERE cargo_id = $1 AND tipo = 'definitivo')`,
			cargoID).Scan(&temDefinitivo); err != nil {
			return 0, err
		}
		if temDefinitivo {
			return 0, fmt.Errorf("%w: o cargo já tem gabarito definitivo", ErrGabaritoInvalido)
		}
	}
	err = func() error {
		var questoes int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM questao WHERE cargo_id = $1`, cargoID).Scan(&questoes); err != nil {
			return err
		}
		if questoes == 0 {
			return ErrSemProva
		}
		if len(respostas) != questoes {
			return fmt.Errorf("%w: %d respostas para %d questões", ErrGabaritoInvalido, len(respostas), questoes)
		}
		for _, c := range respostas {
			if !slices.Contains([]rune("ABCDE*"), c) {
				return fmt.Errorf("%w: %q não é resposta", ErrGabaritoInvalido, c)
			}
		}
		if err := tx.QueryRow(ctx, `
			INSERT INTO gabarito (cargo_id, versao, tipo, respostas, motivo)
			SELECT $1, coalesce(max(versao), 0) + 1, $2, $3, $4 FROM gabarito WHERE cargo_id = $1
			RETURNING versao`, cargoID, tipo, respostas, motivo).Scan(&versao); err != nil {
			return err
		}
		return fila.Enfileirar(ctx, tx, TarefaGabarito, strconv.FormatInt(certameID, 10), carga{ID: certameID})
	}()
	return versao, err
}

type Folha struct {
	Inscricao int64
	Respostas string
}

type Importacao struct {
	Gravadas  int
	Recusadas map[int64]string // inscrição → motivo
}

// ImportarFolhas recebe um lote da leitora óptica da banca. Folha que não confere (inscrição que não
// existe ou não foi homologada, tamanho diferente da prova, caractere estranho, inscrição repetida no
// lote) é recusada com o motivo, e o resto entra; reimportar a mesma inscrição substitui a leitura.
func ImportarFolhas(ctx context.Context, pool *pgxpool.Pool, lote string, folhas []Folha) (Importacao, error) {
	imp := Importacao{Recusadas: map[int64]string{}}
	vistas := map[int64]int{}
	for _, f := range folhas {
		vistas[f.Inscricao]++
	}
	var linhas [][]any
	for _, f := range folhas {
		if vistas[f.Inscricao] > 1 {
			imp.Recusadas[f.Inscricao] = "inscrição repetida no lote"
			continue
		}
		linhas = append(linhas, []any{f.Inscricao, f.Respostas})
	}
	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `CREATE TEMP TABLE folha_lote (inscricao_id bigint, respostas text) ON COMMIT DROP`); err != nil {
			return err
		}
		if _, err := tx.CopyFrom(ctx, pgx.Identifier{"folha_lote"}, []string{"inscricao_id", "respostas"}, pgx.CopyFromRows(linhas)); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `
			WITH conferida AS (
				SELECT l.inscricao_id, l.respostas,
				       CASE WHEN i.id IS NULL THEN 'inscrição inexistente'
				            WHEN i.estado <> 'HOMOLOGADA' THEN 'inscrição não homologada'
				            WHEN l.respostas !~ '^[A-E.*]+$' THEN 'marcação inválida'
				            WHEN length(l.respostas) <> (SELECT count(*) FROM questao q WHERE q.cargo_id = i.cargo_id)
				                THEN 'número de respostas diferente da prova'
				       END AS motivo
				  FROM folha_lote l LEFT JOIN inscricao i ON i.id = l.inscricao_id
			), gravada AS (
				INSERT INTO folha_resposta (inscricao_id, respostas, lote)
				SELECT inscricao_id, respostas, $1 FROM conferida WHERE motivo IS NULL
				ON CONFLICT (inscricao_id) DO UPDATE
				   SET respostas = EXCLUDED.respostas, lote = EXCLUDED.lote, importada_em = now()
				RETURNING 0
			)
			SELECT inscricao_id, motivo FROM conferida WHERE motivo IS NOT NULL
			UNION ALL
			SELECT NULL, NULL FROM gravada`, lote)
		if err != nil {
			return err
		}
		var id *int64
		var motivo *string
		_, err = pgx.ForEachRow(rows, []any{&id, &motivo}, func() error {
			if id == nil {
				imp.Gravadas++
			} else {
				imp.Recusadas[*id] = *motivo
			}
			return nil
		})
		return err
	})
	return imp, err
}

// HomologarLote leva à lista definitiva toda inscrição que vale (paga ou isenta), pela máquina de
// estados e com trilha: análise, deferimento e homologação, com o analista registrado. É o
// "deferimento em lote" do módulo D; a análise individual de laudo fica com a banca, antes disto.
// Cada inscrição é uma transação, e o cartão é republicado com a situação nova.
func HomologarLote(ctx context.Context, pool *pgxpool.Pool, certameID int64, analista string) (int, error) {
	rows, err := pool.Query(ctx, `
		SELECT id FROM inscricao
		 WHERE certame_id = $1 AND estado IN ('PAGA', 'ISENCAO_ATIVA', 'EM_ANALISE', 'DEFERIDA')
		 ORDER BY id`, certameID)
	if err != nil {
		return 0, err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		return 0, err
	}
	sistema := inscricao.Ator{Tipo: inscricao.Sistema, ID: "homologacao-em-lote"}
	pessoa := inscricao.Ator{Tipo: inscricao.Analista, ID: analista}
	var (
		mu        sync.Mutex
		feitas    int
		falha     error
		wg        sync.WaitGroup
		pendentes = make(chan int64)
	)
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for id := range pendentes {
				err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
					for _, passo := range []struct {
						ev   inscricao.Evento
						ator inscricao.Ator
					}{{inscricao.IniciarAnalise, sistema}, {inscricao.Deferir, pessoa}, {inscricao.Homologar, pessoa}} {
						_, err := inscricao.TransicionarTx(ctx, tx, inscricao.Comando{InscricaoID: id, Evento: passo.ev, Ator: passo.ator})
						// O passo que já ficou para trás (a inscrição já estava em análise ou deferida)
						// não se aplica: segue para o próximo.
						if err != nil && !errors.Is(err, inscricao.ErrTransicaoInvalida) {
							return err
						}
					}
					return publicacao.EnfileirarInscricaoLote(ctx, tx, id, certameID)
				})
				mu.Lock()
				if err != nil && falha == nil {
					falha = fmt.Errorf("homologar inscrição %d: %w", id, err)
				} else if err == nil {
					feitas++
				}
				mu.Unlock()
			}
		}()
	}
	for _, id := range ids {
		pendentes <- id
	}
	close(pendentes)
	wg.Wait()
	return feitas, falha
}

// ResumoCargo conta o que o cálculo produziu num cargo.
type ResumoCargo struct {
	Cargo           string         `json:"cargo"`
	Gabarito        int            `json:"gabarito_versao"`
	Candidatos      int            `json:"candidatos"`
	PorSituacao     map[string]int `json:"por_situacao"`
	VagasReservadas map[string]int `json:"vagas_reservadas"`
	Empates         int            `json:"empates"`
}

type Resumo struct {
	Versao   int                   `json:"versao"`
	Tipo     string                `json:"tipo"`
	Duracao  time.Duration         `json:"duracao_ns"`
	PorCargo map[int64]ResumoCargo `json:"por_cargo"`
}

// CalcularCertame corrige e classifica todos os cargos do certame e grava o resultado como uma versão
// nova, numa transação só: a versão aparece inteira ou não aparece. A publicação estática sai pela
// fila, na mesma transação.
func CalcularCertame(ctx context.Context, pool *pgxpool.Pool, certameID int64, tipo, motivo string) (Resumo, error) {
	inicio := time.Now()
	res := Resumo{Tipo: tipo, PorCargo: map[int64]ResumoCargo{}}
	if tipo != "preliminar" && tipo != "definitivo" {
		return res, fmt.Errorf("%w: tipo %q", ErrRegra, tipo)
	}
	if motivo == "" {
		return res, fmt.Errorf("%w: motivo obrigatório", ErrRegra)
	}
	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		// Dois cálculos ao mesmo tempo disputariam o número da versão: o certame fica travado.
		var referencia *time.Time
		err := tx.QueryRow(ctx, `
			SELECT (SELECT fecha_em FROM certame_fase WHERE certame_id = c.id AND fase = 'inscricao')
			  FROM certame c WHERE c.id = $1 FOR UPDATE`, certameID).Scan(&referencia)
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("certame %d não encontrado", certameID)
		}
		if err != nil {
			return err
		}
		if referencia == nil {
			return fmt.Errorf("%w: certame sem fase de inscrição (data de referência da idade)", ErrRegra)
		}
		cotas, err := cotasDoCertame(ctx, tx, certameID)
		if err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, `
			SELECT coalesce(max(versao), 0) + 1 FROM resultado_versao WHERE certame_id = $1`, certameID).Scan(&res.Versao); err != nil {
			return err
		}
		cargos, err := cargosDoCertame(ctx, tx, certameID)
		if err != nil {
			return err
		}
		gabaritos := map[string]int{}
		type calculado struct {
			cargo int64
			itens []Item
		}
		var todos []calculado
		for _, cg := range cargos {
			prova, gab, err := provaEGabarito(ctx, tx, cg.id, tipo)
			if err != nil {
				return fmt.Errorf("cargo %s: %w", cg.nome, err)
			}
			candidatos, err := candidatosDoCargo(ctx, tx, cg.id)
			if err != nil {
				return err
			}
			regra := Regra{Vagas: cg.vagas, NotaMinima: cg.notaMinima, Desempate: cg.desempate, Cotas: cotas, Referencia: *referencia}
			itens, err := Calcular(prova, gab.respostas, regra, candidatos)
			if err != nil {
				return fmt.Errorf("cargo %s: %w", cg.nome, err)
			}
			reserva, _ := Reservar(regra)
			rc := ResumoCargo{Cargo: cg.nome, Gabarito: gab.versao, Candidatos: len(itens), PorSituacao: map[string]int{},
				VagasReservadas: reserva.PorCota}
			for _, it := range itens {
				rc.PorSituacao[string(it.Situacao)]++
				if it.Empate {
					rc.Empates++
				}
			}
			res.PorCargo[cg.id] = rc
			gabaritos[strconv.FormatInt(cg.id, 10)] = gab.versao
			todos = append(todos, calculado{cg.id, itens})
		}
		gj, _ := json.Marshal(gabaritos)
		if _, err := tx.Exec(ctx, `
			INSERT INTO resultado_versao (certame_id, versao, tipo, gabaritos, motivo) VALUES ($1, $2, $3, $4, $5)`,
			certameID, res.Versao, tipo, gj, motivo); err != nil {
			return err
		}
		var linhas, posicoes [][]any
		for _, c := range todos {
			for _, it := range c.itens {
				l := []any{certameID, res.Versao, it.Inscricao, c.cargo, string(it.Situacao), nil, nil, nil, nil, nil, it.Empate}
				if it.Situacao != Ausente {
					disc := map[string]json.Number{}
					for d, v := range it.PorDisciplina {
						disc[d] = Pontos(v)
					}
					l[5], l[6], l[7] = numeric(it.Nota), it.Acertos, disc
				}
				if it.PosicaoAmpla > 0 {
					l[8] = it.PosicaoAmpla
				}
				if it.Vaga != "" {
					l[9] = it.Vaga
				}
				linhas = append(linhas, l)
				for cota, pos := range it.PosicoesCota {
					posicoes = append(posicoes, []any{certameID, res.Versao, it.Inscricao, cota, pos})
				}
			}
		}
		if _, err := tx.CopyFrom(ctx, pgx.Identifier{"resultado_item"},
			[]string{"certame_id", "versao", "inscricao_id", "cargo_id", "situacao", "nota", "acertos",
				"por_disciplina", "posicao_ampla", "vaga", "empate"}, pgx.CopyFromRows(linhas)); err != nil {
			return err
		}
		if _, err := tx.CopyFrom(ctx, pgx.Identifier{"resultado_posicao_cota"},
			[]string{"certame_id", "versao", "inscricao_id", "cota_codigo", "posicao"}, pgx.CopyFromRows(posicoes)); err != nil {
			return err
		}
		return fila.Enfileirar(ctx, tx, TarefaResultado, strconv.FormatInt(certameID, 10), carga{ID: certameID})
	})
	res.Duracao = time.Since(inicio)
	return res, err
}

// Pontos escreve centésimos como número decimal ("7.25"), sem passar por ponto flutuante.
func Pontos(c int64) json.Number {
	sinal := ""
	if c < 0 {
		sinal, c = "-", -c
	}
	return json.Number(fmt.Sprintf("%s%d.%02d", sinal, c/100, c%100))
}

func numeric(c int64) pgtype.Numeric {
	return pgtype.Numeric{Int: big.NewInt(c), Exp: -2, Valid: true}
}

// paraCentesimos converte o numeric(…,2) do banco, que o pgx lê como pgtype.Numeric.
func paraCentesimos(n pgtype.Numeric) int64 {
	if !n.Valid {
		return 0
	}
	v := new(big.Int).Set(n.Int)
	for e := n.Exp; e < -2; e++ {
		v.Quo(v, big.NewInt(10))
	}
	for e := n.Exp; e > -2; e-- {
		v.Mul(v, big.NewInt(10))
	}
	return v.Int64()
}

func travarCargo(ctx context.Context, tx pgx.Tx, cargoID int64) error {
	var id int64
	err := tx.QueryRow(ctx, `SELECT id FROM cargo WHERE id = $1 FOR UPDATE`, cargoID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("cargo %d não encontrado", cargoID)
	}
	return err
}

func certameDoCargo(ctx context.Context, tx pgx.Tx, cargoID int64) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, `SELECT certame_id FROM cargo WHERE id = $1`, cargoID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("cargo %d não encontrado", cargoID)
	}
	return id, err
}

func cotasDoCertame(ctx context.Context, tx pgx.Tx, certameID int64) ([]Cota, error) {
	rows, err := tx.Query(ctx, `
		SELECT codigo, reserva_percentual, arredondamento, minimo_vagas FROM cota
		 WHERE certame_id = $1 ORDER BY codigo`, certameID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Cota, error) {
		var c Cota
		var pct pgtype.Numeric
		err := r.Scan(&c.Codigo, &pct, &c.Arredondamento, &c.MinimoVagas)
		c.Percentual = paraCentesimos(pct)
		return c, err
	})
}

type cargoRegra struct {
	id         int64
	nome       string
	vagas      int
	notaMinima int64
	desempate  []string
}

func cargosDoCertame(ctx context.Context, tx pgx.Tx, certameID int64) ([]cargoRegra, error) {
	rows, err := tx.Query(ctx, `
		SELECT id, nome, vagas, nota_minima, desempate FROM cargo WHERE certame_id = $1 ORDER BY id`, certameID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (cargoRegra, error) {
		var c cargoRegra
		var nm pgtype.Numeric
		err := r.Scan(&c.id, &c.nome, &c.vagas, &nm, &c.desempate)
		c.notaMinima = paraCentesimos(nm)
		return c, err
	})
}

type gabaritoUsado struct {
	versao    int
	respostas string
}

func provaEGabarito(ctx context.Context, tx pgx.Tx, cargoID int64, tipo string) ([]Questao, gabaritoUsado, error) {
	var g gabaritoUsado
	rows, err := tx.Query(ctx, `SELECT disciplina, peso FROM questao WHERE cargo_id = $1 ORDER BY numero`, cargoID)
	if err != nil {
		return nil, g, err
	}
	prova, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Questao, error) {
		var q Questao
		var peso pgtype.Numeric
		err := r.Scan(&q.Disciplina, &peso)
		q.Peso = paraCentesimos(peso)
		return q, err
	})
	if err != nil {
		return nil, g, err
	}
	if len(prova) == 0 {
		return nil, g, ErrSemProva
	}
	var tipoGab string
	err = tx.QueryRow(ctx, `
		SELECT versao, tipo, respostas FROM gabarito WHERE cargo_id = $1 ORDER BY versao DESC LIMIT 1`,
		cargoID).Scan(&g.versao, &tipoGab, &g.respostas)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, g, ErrSemGabarito
	}
	if err != nil {
		return nil, g, err
	}
	if tipo == "definitivo" && tipoGab != "definitivo" {
		return nil, g, ErrGabaritoPreliminar
	}
	return prova, g, nil
}

func candidatosDoCargo(ctx context.Context, tx pgx.Tx, cargoID int64) ([]Candidato, error) {
	rows, err := tx.Query(ctx, `
		SELECT i.id, coalesce(f.respostas, ''), c.data_nascimento,
		       coalesce((SELECT array_agg(ic.cota_codigo ORDER BY ic.cota_codigo)
		                   FROM inscricao_cota ic WHERE ic.inscricao_id = i.id), '{}')
		  FROM inscricao i
		  JOIN candidato c ON c.id = i.candidato_id
		  LEFT JOIN folha_resposta f ON f.inscricao_id = i.id
		 WHERE i.cargo_id = $1 AND i.estado = 'HOMOLOGADA'`, cargoID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Candidato, error) {
		var c Candidato
		var nasc *time.Time
		err := r.Scan(&c.Inscricao, &c.Respostas, &nasc, &c.Cotas)
		if nasc != nil {
			c.Nascimento = *nasc
		}
		return c, err
	})
}
