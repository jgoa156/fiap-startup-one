// Package seed cria um certame sintético: o processo seletivo simplificado de prefeitura que é o
// primeiro cliente-alvo (02-produto §8). Usado pelo cmd/seed, pelos testes e pelo ensaio de carga.
package seed

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"lisura/internal/publicacao"
)

// A prova de todo cargo do certame sintético.
const (
	Questoes    = 20
	Portugues   = "Língua Portuguesa"
	Especificos = "Conhecimentos Específicos"
)

type Certame struct {
	ID      int64   `json:"certame_id"`
	Cargos  []int64 `json:"cargos"`
	Cidades []int64 `json:"cidades"`
}

// Janelas relativas a agora. Uma janela com Fecha <= 0 já está encerrada.
type Opcoes struct {
	Inscricao, Isencao, Pagamento Janela
}

type Janela struct{ Abre, Fecha time.Duration }

func Padrao() Opcoes {
	return Opcoes{
		Isencao:   Janela{-24 * time.Hour, 3 * 24 * time.Hour},
		Inscricao: Janela{-24 * time.Hour, 30 * 24 * time.Hour},
		Pagamento: Janela{-24 * time.Hour, 32 * 24 * time.Hour},
	}
}

func CriarCertame(ctx context.Context, pool *pgxpool.Pool, o Opcoes) (Certame, error) {
	var c Certame
	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		var banca int64
		cnpj := fmt.Sprintf("%014d", rand.Int64N(1e14))
		if err := tx.QueryRow(ctx,
			`INSERT INTO banca (nome, cnpj) VALUES ('Instituto Exemplo de Seleção', $1) RETURNING id`,
			cnpj).Scan(&banca); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx,
			`INSERT INTO certame (banca_id, nome) VALUES ($1, 'PSS 2027 — Prefeitura de Exemplo') RETURNING id`,
			banca).Scan(&c.ID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO edital_versao (certame_id, versao, publicado_em, resumo)
			 VALUES ($1, 1, now() - interval '2 days', 'Edital de abertura')`, c.ID); err != nil {
			return err
		}
		for fase, j := range map[string]Janela{"isencao": o.Isencao, "inscricao": o.Inscricao, "pagamento": o.Pagamento} {
			if _, err := tx.Exec(ctx,
				`INSERT INTO certame_fase (certame_id, fase, abre_em, fecha_em)
				 VALUES ($1, $2, now() + $3::float8 * interval '1 second', now() + $4::float8 * interval '1 second')`,
				c.ID, fase, j.Abre.Seconds(), j.Fecha.Seconds()); err != nil {
				return err
			}
		}
		for _, cg := range []struct {
			nome  string
			vagas int
			taxa  int
		}{
			{"Agente Administrativo", 40, 8000},
			{"Professor de Educação Básica", 30, 10000},
			{"Enfermeiro", 10, 12000},
		} {
			// Prova objetiva de 20 questões: 10 de Língua Portuguesa (peso 1) e 10 de Conhecimentos
			// Específicos (peso 2), 30 pontos; nota mínima 12; desempate depois da idade legal: maior
			// nota em Específicos, depois em Português, depois o mais velho.
			var id int64
			if err := tx.QueryRow(ctx, `
				INSERT INTO cargo (certame_id, nome, vagas, taxa_centavos, nota_minima, desempate)
				VALUES ($1, $2, $3, $4, 12, $5) RETURNING id`,
				c.ID, cg.nome, cg.vagas, cg.taxa, []string{
					"disciplina:" + Especificos, "disciplina:" + Portugues, "idade"}).Scan(&id); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO questao (cargo_id, numero, disciplina, peso)
				SELECT $1, n, CASE WHEN n <= 10 THEN $2 ELSE $3 END, CASE WHEN n <= 10 THEN 1 ELSE 2 END
				  FROM generate_series(1, $4::int) n`, id, Portugues, Especificos, Questoes); err != nil {
				return err
			}
			c.Cargos = append(c.Cargos, id)
		}
		for _, cid := range []string{"Campinas", "Sorocaba", "Ribeirão Preto"} {
			var id int64
			if err := tx.QueryRow(ctx,
				`INSERT INTO cidade_prova (certame_id, nome, uf) VALUES ($1, $2, 'SP') RETURNING id`,
				c.ID, cid).Scan(&id); err != nil {
				return err
			}
			c.Cidades = append(c.Cidades, id)
		}
		// Reserva nos moldes federais (Decreto 9.508/2018 e Lei 15.142/2025); num certame municipal
		// real vale a lei do município, e estes números saem do edital.
		_, err := tx.Exec(ctx, `
			INSERT INTO cota (certame_id, codigo, nome, exige_laudo, reserva_percentual, arredondamento, minimo_vagas) VALUES
			($1, 'pcd', 'Pessoa com deficiência', true, 5, 'cima', 5),
			($1, 'ppiq', 'Pessoas pretas, pardas, indígenas e quilombolas', false, 30, 'meio', 2)`, c.ID)
		if err != nil {
			return err
		}
		return publicacao.EnfileirarCertame(ctx, tx, c.ID) // a página pública sai pela fila
	})
	return c, err
}
