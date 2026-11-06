package inscricao

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// A trilha encadeada (migração 00010): cada evento guarda SHA-256(resumo do anterior ‖ conteúdo). A
// conta é refeita aqui, em Go, e não por uma função do banco — quem adulterasse o banco poderia
// adulterar a função também.

type consultor interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Quebra diz onde a corrente de uma inscrição deixou de bater.
type Quebra struct {
	Inscricao int64
	Evento    int64 // primeiro evento cujo resumo não confere
}

func (q Quebra) Error() string {
	return fmt.Sprintf("trilha da inscrição %d quebrada no evento %d", q.Inscricao, q.Evento)
}

// resumo é a mesma conta de trilha_resumo, campo a campo.
func resumo(anterior []byte, inscricao int64, evento, de, para, atorTipo, atorID, motivo string, versao int, em time.Time) []byte {
	conteudo := strings.Join([]string{
		strconv.FormatInt(inscricao, 10), evento, de, para, atorTipo, atorID, motivo, strconv.Itoa(versao),
		em.UTC().Format("2006-01-02T15:04:05.000000"),
	}, "|")
	h := sha256.Sum256(append(append([]byte{}, anterior...), conteudo...))
	return h[:]
}

// ResumoDoEvento é a conta da corrente para quem verifica fora do banco (a exportação para o órgão e a
// auditoria): o resumo do evento, dado o resumo do anterior (nil no primeiro). É a mesma função que
// VerificarTrilhas usa, para as duas verificações nunca divergirem.
func ResumoDoEvento(anterior []byte, inscricao int64, evento, de, para, atorTipo, atorID, motivo string, versao int, em time.Time) []byte {
	return resumo(anterior, inscricao, evento, de, para, atorTipo, atorID, motivo, versao, em)
}

// VerificarTrilhas refaz a corrente das inscrições pedidas (todas, se ids for vazio) e devolve as
// quebras e o selo (resumo do último evento) de cada inscrição verificada.
func VerificarTrilhas(ctx context.Context, q consultor, ids []int64) (quebras []Quebra, selos map[int64]string, err error) {
	sql := `SELECT id, inscricao_id, evento, de_estado, para_estado, ator_tipo, ator_id,
	               coalesce(motivo_codigo, ''), edital_versao, ocorrido_em, hash
	          FROM inscricao_evento`
	args := []any{}
	if len(ids) > 0 {
		sql += ` WHERE inscricao_id = ANY($1)`
		args = append(args, ids)
	}
	rows, err := q.Query(ctx, sql+` ORDER BY inscricao_id, id`, args...)
	if err != nil {
		return nil, nil, err
	}
	selos = map[int64]string{}
	var (
		id, insc, atual int64 = 0, 0, -1
		ev, de, para    string
		tipo, ator, mot string
		versao          int
		em              time.Time
		hash, anterior  []byte
		quebrada        bool
	)
	_, err = pgx.ForEachRow(rows, []any{&id, &insc, &ev, &de, &para, &tipo, &ator, &mot, &versao, &em, &hash}, func() error {
		if insc != atual {
			atual, anterior, quebrada = insc, nil, false
		}
		esperado := resumo(anterior, insc, ev, de, para, tipo, ator, mot, versao, em)
		if !quebrada && !bytes.Equal(esperado, hash) {
			quebras = append(quebras, Quebra{Inscricao: insc, Evento: id})
			quebrada = true // uma quebra por inscrição: dali em diante nada confere
		}
		// Segue a corrente pelo resumo gravado: assim uma adulteração aponta o evento adulterado,
		// e não todos os seguintes.
		anterior = append(anterior[:0], hash...)
		selos[insc] = hex.EncodeToString(hash)
		return nil
	})
	return quebras, selos, err
}

// Selo é o resumo do último evento da trilha, em hexadecimal: o que o cartão imprime.
func Selo(ctx context.Context, q consultor, inscricaoID int64) (string, error) {
	var h []byte
	err := q.QueryRow(ctx, `SELECT hash FROM inscricao_evento WHERE inscricao_id = $1 ORDER BY id DESC LIMIT 1`,
		inscricaoID).Scan(&h)
	return hex.EncodeToString(h), err
}

// ConfereSelo diz se um selo guardado pelo candidato (inteiro ou um prefixo de pelo menos 16
// dígitos, como sai impresso) ainda aparece na corrente da inscrição. Se a trilha foi reescrita, não
// aparece — mesmo que a corrente nova seja internamente consistente.
func ConfereSelo(ctx context.Context, q consultor, inscricaoID int64, selo string) (bool, error) {
	selo = strings.ToLower(strings.TrimSpace(selo))
	if len(selo) < 16 {
		return false, fmt.Errorf("%w: selo com menos de 16 dígitos", ErrDadoInvalido)
	}
	rows, err := q.Query(ctx, `SELECT hash FROM inscricao_evento WHERE inscricao_id = $1`, inscricaoID)
	if err != nil {
		return false, err
	}
	var h []byte
	achou := false
	_, err = pgx.ForEachRow(rows, []any{&h}, func() error {
		achou = achou || strings.HasPrefix(hex.EncodeToString(h), selo)
		return nil
	})
	return achou, err
}
