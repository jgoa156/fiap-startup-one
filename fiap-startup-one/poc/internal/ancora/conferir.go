package ancora

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"lisura/internal/arquivo"
)

type consultor interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// Problema é o que a conferência achou, em texto para gente ler.
type Problema struct {
	Tipo      string `json:"tipo"`
	Sequencia int    `json:"sequencia,omitempty"`
	Inscricao int64  `json:"inscricao,omitempty"`
	Evento    int64  `json:"evento,omitempty"`
	Detalhe   string `json:"detalhe"`
}

type Conferencia struct {
	Certame   int64      `json:"certame"`
	Ancoras   int        `json:"ancoras"`
	Pontos    int        `json:"pontos_conferidos"` // pares (inscrição, evento) ancorados conferidos no banco
	Problemas []Problema `json:"problemas"`
}

// Atraso tolerado entre um evento e a âncora que o cobre. Um evento que já existia há mais que isso
// quando a âncora foi gerada e não está nela foi enfiado no passado.
const Folga = 10 * time.Minute

// Conferir refaz, contra o banco, tudo o que o cofre guarda do certame:
//
//   - a cadeia das âncoras: sequência sem buraco e cada uma citando o resumo da anterior;
//   - de uma âncora para a seguinte, nenhuma inscrição some e nenhuma corrente anda para trás ou muda
//     de resumo no mesmo evento;
//   - cada ponto ancorado (inscrição, evento, resumo) ainda está no banco, igual. Como o resumo de um
//     evento cobre a corrente inteira até ele, conferir o ponto confere o que veio antes — desde que a
//     corrente também confira por dentro (auditoria -trilhas);
//   - nenhuma inscrição com evento anterior à última âncora (menos a Folga) ficou fora dela.
//
// Cada ponto é conferido uma vez: um ponto que se repete de uma âncora para a outra é o mesmo.
func Conferir(ctx context.Context, q consultor, cofre arquivo.Cofre, certame int64) (Conferencia, error) {
	c := Conferencia{Certame: certame, Problemas: []Problema{}}
	chaves, err := cofre.Listar(ctx, prefixo(certame))
	if err != nil {
		return c, err
	}
	c.Ancoras = len(chaves)
	acusar := func(p Problema) { c.Problemas = append(c.Problemas, p) }

	atual := map[int64]Linha{} // inscrição -> ponto mais recente ancorado
	sumidas := map[int64]bool{}
	var pontos []Linha // pontos substituídos por um mais novo: conferidos no fim, junto com os atuais
	var ultima Ancora
	anterior := "-"
	for i, k := range chaves {
		texto, err := cofre.Ler(ctx, k)
		if err != nil {
			return c, err
		}
		a, err := Ler(texto)
		if err != nil {
			acusar(Problema{Tipo: "ANCORA_ILEGIVEL", Sequencia: i + 1, Detalhe: fmt.Sprintf("%s: %v", k, err)})
			return c, nil // sem ler esta, a cadeia daqui em diante não tem como ser conferida
		}
		if a.Sequencia != i+1 || a.Certame != certame || k != Chave(certame, a.Sequencia) {
			acusar(Problema{Tipo: "ANCORA_FALTANDO", Sequencia: i + 1,
				Detalhe: fmt.Sprintf("%s traz certame %d, sequência %d", k, a.Certame, a.Sequencia)})
		}
		if a.Anterior != anterior {
			acusar(Problema{Tipo: "CADEIA_DE_ANCORAS_QUEBRADA", Sequencia: a.Sequencia,
				Detalhe: fmt.Sprintf("cita a anterior como %s; a anterior tem resumo %s", a.Anterior, anterior)})
		}
		anterior = Resumo(texto)

		vistas := make(map[int64]bool, len(a.Linhas))
		for _, l := range a.Linhas {
			vistas[l.Inscricao] = true
			antes, tinha := atual[l.Inscricao]
			switch {
			case !tinha:
			case l.Evento < antes.Evento:
				acusar(Problema{Tipo: "CORRENTE_ANDOU_PARA_TRAS", Sequencia: a.Sequencia, Inscricao: l.Inscricao, Evento: l.Evento,
					Detalhe: fmt.Sprintf("a âncora anterior já tinha o evento %d", antes.Evento)})
			case l.Evento == antes.Evento && !bytes.Equal(l.Resumo, antes.Resumo):
				acusar(Problema{Tipo: "RESUMO_MUDOU_ENTRE_ANCORAS", Sequencia: a.Sequencia, Inscricao: l.Inscricao, Evento: l.Evento,
					Detalhe: "o mesmo evento aparece com outro resumo: a trilha foi reescrita entre as duas âncoras"})
			case l.Evento > antes.Evento:
				pontos = append(pontos, antes)
			}
			atual[l.Inscricao] = l
		}
		// Só a completa lista todo mundo; na incremental, quem não aparece não mudou.
		for id, l := range atual {
			if a.Tipo == Completa && !vistas[id] {
				acusar(Problema{Tipo: "INSCRICAO_SUMIU_DA_ANCORA", Sequencia: a.Sequencia, Inscricao: id, Evento: l.Evento,
					Detalhe: "estava na âncora anterior e não está nesta"})
				delete(atual, id)
				sumidas[id] = true
				pontos = append(pontos, l) // continua tendo de estar no banco
			}
		}
		if a.Total != len(atual) {
			acusar(Problema{Tipo: "ANCORA_INCONSISTENTE", Sequencia: a.Sequencia,
				Detalhe: fmt.Sprintf("declara %d inscrições; somada às anteriores dá %d", a.Total, len(atual))})
		}
		ultima = a
	}
	for _, l := range atual {
		pontos = append(pontos, l)
	}
	c.Pontos = len(pontos)

	// Os pontos contra o banco, em lotes.
	const lote = 50_000
	for ini := 0; ini < len(pontos); ini += lote {
		parte := pontos[ini:min(ini+lote, len(pontos))]
		ids := make([]int64, len(parte))
		for i, p := range parte {
			ids[i] = p.Evento
		}
		rows, err := q.Query(ctx, `SELECT id, inscricao_id, hash FROM inscricao_evento WHERE id = ANY($1)`, ids)
		if err != nil {
			return c, err
		}
		type gravado struct {
			inscricao int64
			hash      []byte
		}
		noBanco := make(map[int64]gravado, len(parte))
		var id, insc int64
		var h []byte
		if _, err := pgx.ForEachRow(rows, []any{&id, &insc, &h}, func() error {
			noBanco[id] = gravado{insc, append([]byte(nil), h...)}
			return nil
		}); err != nil {
			return c, err
		}
		for _, p := range parte {
			g, ok := noBanco[p.Evento]
			switch {
			case !ok:
				acusar(Problema{Tipo: "EVENTO_ANCORADO_SUMIU", Inscricao: p.Inscricao, Evento: p.Evento,
					Detalhe: "o evento está no cofre e não está no banco"})
			case g.inscricao != p.Inscricao || !bytes.Equal(g.hash, p.Resumo):
				acusar(Problema{Tipo: "TRILHA_DIFERENTE_DA_ANCORA", Inscricao: p.Inscricao, Evento: p.Evento,
					Detalhe: "o resumo do evento no banco não é o que foi ancorado: a corrente foi reescrita"})
			}
		}
	}

	if c.Ancoras == 0 {
		return c, nil
	}
	// Inscrição que já tinha evento quando a última âncora foi gerada tem de estar nela.
	rows, err := q.Query(ctx, `
		SELECT i.id, min(e.id), min(e.ocorrido_em) FROM inscricao i JOIN inscricao_evento e ON e.inscricao_id = i.id
		WHERE i.certame_id = $1 GROUP BY i.id HAVING min(e.ocorrido_em) < $2`, certame, ultima.GeradaEm.Add(-Folga))
	if err != nil {
		return c, err
	}
	var id, ev int64
	var em time.Time
	_, err = pgx.ForEachRow(rows, []any{&id, &ev, &em}, func() error {
		if _, ok := atual[id]; !ok && !sumidas[id] {
			acusar(Problema{Tipo: "INSCRICAO_FORA_DA_ANCORA", Inscricao: id, Evento: ev,
				Detalhe: fmt.Sprintf("primeiro evento em %s, antes da âncora %d, e ela não o tem", em.UTC().Format(time.RFC3339), ultima.Sequencia)})
		}
		return nil
	})
	return c, err
}
