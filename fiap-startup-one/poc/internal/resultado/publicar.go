package resultado

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"lisura/internal/fila"
	"lisura/internal/publicacao"
)

const (
	TarefaGabarito  = "publicar_gabarito"
	TarefaResultado = "publicar_resultado"
	// Fatias é em quantos arquivos o resultado individual é dividido: 256 (dois dígitos hexadecimais).
	// Um arquivo por candidato seriam 200 mil gravações a cada recálculo; um arquivo só, 30 MB que cada
	// candidato baixaria para achar a própria linha.
	Fatias = 256
)

type carga struct {
	ID int64 `json:"id"`
}

// Publicador gera os arquivos estáticos dos picos 4 (gabarito) e 5 (resultado), pela fila. Servir
// não toca no banco: é o mesmo destino do cartão (S3 + CloudFront na nuvem).
type Publicador struct {
	Pool *pgxpool.Pool
	Pub  *publicacao.Publicador
}

func (p *Publicador) Manipuladores() map[string]fila.Manipulador {
	ler := func(f func(context.Context, int64) error) fila.Manipulador {
		return func(ctx context.Context, b json.RawMessage) error {
			var c carga
			if err := json.Unmarshal(b, &c); err != nil {
				return err
			}
			return f(ctx, c.ID)
		}
	}
	return map[string]fila.Manipulador{TarefaGabarito: ler(p.PublicarGabarito), TarefaResultado: ler(p.PublicarResultado)}
}

// ChaveResultado é como o candidato acha a própria linha no resultado: deriva do código do cartão,
// que só ele tem (e que o front já guardou no aparelho). É um resumo de mão única: quem baixa as
// fatias vê chaves, mas não consegue voltar delas ao código do cartão de ninguém.
func ChaveResultado(tokenCartao string) string {
	h := sha256.Sum256([]byte("resultado:" + tokenCartao))
	return hex.EncodeToString(h[:12])
}

// CaminhoFatia é o arquivo em que a chave está.
func CaminhoFatia(certameID int64, chave string) string {
	return fmt.Sprintf("certames/%d/resultados/%s.json", certameID, chave[:2])
}

// ---- gabarito ----

type questaoPublica struct {
	Numero     int         `json:"numero"`
	Disciplina string      `json:"disciplina"`
	Peso       json.Number `json:"peso"`
	Resposta   *string     `json:"resposta"` // null: anulada
}

type versaoPublica struct {
	Versao      int       `json:"versao"`
	Tipo        string    `json:"tipo"`
	Motivo      string    `json:"motivo"`
	PublicadoEm time.Time `json:"publicado_em"`
}

type gabaritoCargo struct {
	CargoID   int64            `json:"cargo_id"`
	Cargo     string           `json:"cargo"`
	Atual     versaoPublica    `json:"atual"`
	Questoes  []questaoPublica `json:"questoes"`
	Historico []versaoPublica  `json:"historico"`
}

// PublicarGabarito grava certames/{id}/gabarito.json: a versão em vigor de cada cargo, com as
// anuladas marcadas, e o histórico das versões (o que mudou, quando e por quê).
func (p *Publicador) PublicarGabarito(ctx context.Context, certameID int64) error {
	rows, err := p.Pool.Query(ctx, `
		SELECT g.cargo_id, c.nome, g.versao, g.tipo, g.motivo, g.publicado_em, g.respostas
		  FROM gabarito g JOIN cargo c ON c.id = g.cargo_id
		 WHERE c.certame_id = $1 ORDER BY g.cargo_id, g.versao DESC`, certameID)
	if err != nil {
		return err
	}
	var cargos []*gabaritoCargo
	ultimas := map[int64]string{}
	var cargo int64
	var nome, respostas string
	var v versaoPublica
	if _, err := pgx.ForEachRow(rows, []any{&cargo, &nome, &v.Versao, &v.Tipo, &v.Motivo, &v.PublicadoEm, &respostas}, func() error {
		if len(cargos) == 0 || cargos[len(cargos)-1].CargoID != cargo {
			cargos = append(cargos, &gabaritoCargo{CargoID: cargo, Cargo: nome, Atual: v})
			ultimas[cargo] = respostas
		}
		c := cargos[len(cargos)-1]
		c.Historico = append(c.Historico, v)
		return nil
	}); err != nil {
		return err
	}
	for _, c := range cargos {
		qrows, err := p.Pool.Query(ctx, `SELECT numero, disciplina, peso FROM questao WHERE cargo_id = $1 ORDER BY numero`, c.CargoID)
		if err != nil {
			return err
		}
		var q questaoPublica
		var peso pgtype.Numeric
		if _, err := pgx.ForEachRow(qrows, []any{&q.Numero, &q.Disciplina, &peso}, func() error {
			q.Peso = Pontos(paraCentesimos(peso))
			q.Resposta = nil
			if r := ultimas[c.CargoID][q.Numero-1]; r != '*' {
				s := string(r)
				q.Resposta = &s
			}
			c.Questoes = append(c.Questoes, q)
			return nil
		}); err != nil {
			return err
		}
	}
	if cargos == nil {
		cargos = []*gabaritoCargo{}
	}
	b, err := json.Marshal(struct {
		Certame int64            `json:"certame_id"`
		Cargos  []*gabaritoCargo `json:"cargos"`
	}{certameID, cargos})
	if err != nil {
		return err
	}
	return p.Pub.Gravar(ctx, fmt.Sprintf("certames/%d/gabarito.json", certameID), b)
}

// ---- resultado ----

type linhaLista struct {
	Posicao int         `json:"posicao"`
	Numero  int64       `json:"numero"`
	Nota    json.Number `json:"nota"`
	Vaga    string      `json:"vaga,omitempty"`
	Empate  bool        `json:"empate,omitempty"`
}

type resultadoCargo struct {
	CargoID         int64                   `json:"cargo_id"`
	Cargo           string                  `json:"cargo"`
	Vagas           int                     `json:"vagas"`
	VagasReservadas map[string]int          `json:"vagas_reservadas"`
	Ampla           []linhaLista            `json:"ampla_concorrencia"`
	Cotas           map[string][]linhaLista `json:"cotas"`
}

// Individual é a linha de um candidato na fatia. Sem número de inscrição nem nome: a chave não se
// liga a ninguém sem o código do cartão.
type Individual struct {
	CargoID       int64                  `json:"cargo_id"`
	Situacao      string                 `json:"situacao"`
	Nota          *json.Number           `json:"nota,omitempty"`
	Acertos       *int                   `json:"acertos,omitempty"`
	PorDisciplina map[string]json.Number `json:"por_disciplina,omitempty"`
	PosicaoAmpla  *int                   `json:"posicao_ampla,omitempty"`
	PosicoesCota  map[string]int         `json:"posicoes_cota,omitempty"`
	Vaga          *string                `json:"vaga,omitempty"`
	Empate        bool                   `json:"empate,omitempty"`
}

type cabecalho struct {
	Certame     int64     `json:"certame_id"`
	Versao      int       `json:"versao"`
	Tipo        string    `json:"tipo"`
	Motivo      string    `json:"motivo"`
	CalculadoEm time.Time `json:"calculado_em"`
}

// PublicarResultado grava a última versão calculada: as fatias individuais primeiro e, por último,
// certames/{id}/resultado.json com as listas públicas (só aprovados: posição, número e nota) e o
// histórico de versões. Quem vê a lista nova já acha a própria linha.
func (p *Publicador) PublicarResultado(ctx context.Context, certameID int64) error {
	var cab cabecalho
	cab.Certame = certameID
	err := p.Pool.QueryRow(ctx, `
		SELECT versao, tipo, motivo, calculado_em FROM resultado_versao
		 WHERE certame_id = $1 ORDER BY versao DESC LIMIT 1`, certameID).Scan(&cab.Versao, &cab.Tipo, &cab.Motivo, &cab.CalculadoEm)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // nada calculado ainda
	}
	if err != nil {
		return err
	}

	posCota := map[int64]map[string]int{}
	rows, err := p.Pool.Query(ctx, `
		SELECT inscricao_id, cota_codigo, posicao FROM resultado_posicao_cota
		 WHERE certame_id = $1 AND versao = $2`, certameID, cab.Versao)
	if err != nil {
		return err
	}
	var insc int64
	var cota string
	var pos int
	if _, err := pgx.ForEachRow(rows, []any{&insc, &cota, &pos}, func() error {
		if posCota[insc] == nil {
			posCota[insc] = map[string]int{}
		}
		posCota[insc][cota] = pos
		return nil
	}); err != nil {
		return err
	}

	cargos := map[int64]*resultadoCargo{}
	var ordem []int64
	crow, err := p.Pool.Query(ctx, `SELECT id, nome, vagas FROM cargo WHERE certame_id = $1 ORDER BY id`, certameID)
	if err != nil {
		return err
	}
	var c resultadoCargo
	if _, err := pgx.ForEachRow(crow, []any{&c.CargoID, &c.Cargo, &c.Vagas}, func() error {
		rc := c
		rc.VagasReservadas, rc.Ampla, rc.Cotas = map[string]int{}, []linhaLista{}, map[string][]linhaLista{}
		cargos[c.CargoID] = &rc
		ordem = append(ordem, c.CargoID)
		return nil
	}); err != nil {
		return err
	}

	fatias := make([]map[string]Individual, Fatias)
	for i := range fatias {
		fatias[i] = map[string]Individual{}
	}
	rows, err = p.Pool.Query(ctx, `
		SELECT inscricao_id, cargo_id, situacao, nota, acertos, por_disciplina, posicao_ampla, vaga, empate
		  FROM resultado_item WHERE certame_id = $1 AND versao = $2
		 ORDER BY cargo_id, posicao_ampla NULLS LAST, inscricao_id`, certameID, cab.Versao)
	if err != nil {
		return err
	}
	var (
		it      Individual
		nota    pgtype.Numeric
		acertos *int
		disc    map[string]json.Number
		posA    *int
		vaga    *string
	)
	if _, err := pgx.ForEachRow(rows, []any{&insc, &it.CargoID, &it.Situacao, &nota, &acertos, &disc, &posA, &vaga, &it.Empate}, func() error {
		ind := Individual{CargoID: it.CargoID, Situacao: it.Situacao, Acertos: acertos, PorDisciplina: disc,
			PosicaoAmpla: posA, PosicoesCota: posCota[insc], Vaga: vaga, Empate: it.Empate}
		if nota.Valid {
			n := Pontos(paraCentesimos(nota))
			ind.Nota = &n
		}
		chave := ChaveResultado(p.Pub.Token(insc))
		f, _ := strconv.ParseUint(chave[:2], 16, 8)
		fatias[f][chave] = ind

		rc := cargos[it.CargoID]
		if posA != nil {
			l := linhaLista{Posicao: *posA, Numero: insc, Nota: *ind.Nota, Empate: it.Empate}
			if vaga != nil {
				l.Vaga = *vaga
				if *vaga != VagaAmpla {
					rc.VagasReservadas[*vaga]++
				}
			}
			rc.Ampla = append(rc.Ampla, l)
			for cota, pos := range posCota[insc] {
				lc := l
				lc.Posicao = pos
				rc.Cotas[cota] = append(rc.Cotas[cota], lc)
			}
		}
		acertos, disc, posA, vaga = nil, nil, nil, nil
		return nil
	}); err != nil {
		return err
	}

	for i, f := range fatias {
		b, err := json.Marshal(struct {
			cabecalho
			Candidatos map[string]Individual `json:"candidatos"`
		}{cab, f})
		if err != nil {
			return err
		}
		if err := p.Pub.Gravar(ctx, fmt.Sprintf("certames/%d/resultados/%02x.json", certameID, i), b); err != nil {
			return err
		}
	}

	// As listas de cota já saem em ordem: cada uma é uma subsequência da lista geral, que veio ordenada.
	var lista []*resultadoCargo
	for _, id := range ordem {
		lista = append(lista, cargos[id])
	}
	hrows, err := p.Pool.Query(ctx, `
		SELECT versao, tipo, motivo, calculado_em FROM resultado_versao WHERE certame_id = $1 ORDER BY versao DESC`, certameID)
	if err != nil {
		return err
	}
	type versaoResultado struct {
		Versao      int       `json:"versao"`
		Tipo        string    `json:"tipo"`
		Motivo      string    `json:"motivo"`
		CalculadoEm time.Time `json:"calculado_em"`
	}
	historico, err := pgx.CollectRows(hrows, pgx.RowToStructByPos[versaoResultado])
	if err != nil {
		return err
	}
	b, err := json.Marshal(struct {
		cabecalho
		Cargos    []*resultadoCargo `json:"cargos"`
		Historico []versaoResultado `json:"historico"`
	}{cab, nuncaNulo(lista), historico})
	if err != nil {
		return err
	}
	return p.Pub.Gravar(ctx, fmt.Sprintf("certames/%d/resultado.json", certameID), b)
}

func nuncaNulo[T any](v []T) []T {
	if v == nil {
		return []T{}
	}
	return v
}
