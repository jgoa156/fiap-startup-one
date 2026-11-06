package inscricao

import (
	"context"
	"strconv"
	"testing"
)

// superusuario faz o que a aplicação não consegue: desliga o gatilho de imutabilidade, mexe na
// trilha e religa — tudo numa transação, como faria quem quisesse apagar o rastro.
func superusuario(t *testing.T, sql string, args ...any) {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	must(t, err)
	defer tx.Rollback(ctx)
	for _, s := range []string{`ALTER TABLE inscricao_evento DISABLE TRIGGER inscricao_evento_imutavel`, sql,
		`ALTER TABLE inscricao_evento ENABLE TRIGGER inscricao_evento_imutavel`} {
		a := args
		if s != sql {
			a = nil
		}
		_, err := tx.Exec(ctx, s, a...)
		must(t, err)
	}
	must(t, tx.Commit(ctx))
}

// inscricaoComTrilha devolve uma inscrição com 4 eventos e os ids deles, em ordem.
func inscricaoComTrilha(t *testing.T) (int64, []int64) {
	t.Helper()
	id := novaInscricao(t, novoCertame(t, true))
	aplicar(t, id, FinalizarIsento, cand, "")
	aplicar(t, id, IniciarAnalise, sistema, "")
	aplicar(t, id, Indeferir, analista, "LAUDO_ILEGIVEL")
	rows, err := pool.Query(context.Background(), `SELECT id FROM inscricao_evento WHERE inscricao_id = $1 ORDER BY id`, id)
	must(t, err)
	var ev []int64
	for rows.Next() {
		var e int64
		must(t, rows.Scan(&e))
		ev = append(ev, e)
	}
	if len(ev) != 4 {
		t.Fatalf("%d eventos", len(ev))
	}
	return id, ev
}

func verificar(t *testing.T, id int64) []Quebra {
	t.Helper()
	q, _, err := VerificarTrilhas(context.Background(), pool, []int64{id})
	must(t, err)
	return q
}

func TestTrilhaIntactaConfere(t *testing.T) {
	precisaBanco(t)
	id, _ := inscricaoComTrilha(t)
	if q := verificar(t, id); len(q) != 0 {
		t.Fatalf("trilha sem adulteração acusada: %v", q)
	}
	// A conta em Go e a do banco são a mesma: o selo bate.
	selo, err := Selo(context.Background(), pool, id)
	must(t, err)
	_, selos, err := VerificarTrilhas(context.Background(), pool, []int64{id})
	must(t, err)
	if selos[id] != selo || len(selo) != 64 {
		t.Fatalf("selo %q, verificação %q", selo, selos[id])
	}
}

// Trocar o motivo de um indeferimento é a adulteração que interessa a quem quer apagar um erro.
func TestAlterarEventoAntigoQuebraACorrente(t *testing.T) {
	precisaBanco(t)
	id, ev := inscricaoComTrilha(t)
	superusuario(t, `UPDATE inscricao_evento SET motivo_codigo = 'OUTRO_MOTIVO' WHERE id = $1`, ev[3])
	q := verificar(t, id)
	if len(q) != 1 || q[0].Evento != ev[3] {
		t.Fatalf("quebras %v, esperado no evento %d", q, ev[3])
	}
	superusuario(t, `UPDATE inscricao_evento SET ator_id = 'ninguem' WHERE id = $1`, ev[1])
	if q := verificar(t, id); len(q) != 1 || q[0].Evento != ev[1] {
		t.Fatalf("quebras %v, esperado a primeira no evento %d", q, ev[1])
	}
}

func TestApagarEventoQuebraACorrente(t *testing.T) {
	precisaBanco(t)
	id, ev := inscricaoComTrilha(t)
	superusuario(t, `DELETE FROM inscricao_evento WHERE id = $1`, ev[2])
	if q := verificar(t, id); len(q) != 1 || q[0].Evento != ev[3] {
		t.Fatalf("quebras %v, esperado no evento seguinte ao apagado (%d)", q, ev[3])
	}
}

// Quem reescreve a corrente inteira, recalculando cada resumo, deixa uma trilha consistente — e a
// verificação não acha nada. Quem acha é o selo impresso no cartão do candidato.
func TestReescreverTudoSoOSeloDenuncia(t *testing.T) {
	ctx := precisaBanco(t)
	id, ev := inscricaoComTrilha(t)
	selo, err := Selo(ctx, pool, id)
	must(t, err)
	if ok, err := ConfereSelo(ctx, pool, id, selo[:16]); err != nil || !ok {
		t.Fatalf("selo impresso não confere na trilha intacta: %v %v", ok, err)
	}

	superusuario(t, `UPDATE inscricao_evento SET motivo_codigo = 'OUTRO_MOTIVO' WHERE id = $1`, ev[3])
	// Recalcula a corrente toda com a mesma função do banco.
	superusuario(t, `
		DO $$
		DECLARE r inscricao_evento; anterior bytea;
		BEGIN
			FOR r IN SELECT * FROM inscricao_evento WHERE inscricao_id = `+itoa(id)+` ORDER BY id LOOP
				anterior := trilha_resumo(anterior, r);
				UPDATE inscricao_evento SET hash = anterior WHERE id = r.id;
			END LOOP;
		END $$`)
	if q := verificar(t, id); len(q) != 0 {
		t.Fatalf("a reescrita completa devia passar na verificação interna: %v", q)
	}
	if ok, err := ConfereSelo(ctx, pool, id, selo[:16]); err != nil || ok {
		t.Fatalf("o selo do cartão devia denunciar a reescrita: %v %v", ok, err)
	}
	if _, err := ConfereSelo(ctx, pool, id, "abc"); err == nil {
		t.Error("selo curto demais aceito")
	}
}

// Quem grava não escolhe o próprio resumo.
func TestResumoGravadoEOCalculado(t *testing.T) {
	ctx := precisaBanco(t)
	id, _ := inscricaoComTrilha(t)
	_, err := pool.Exec(ctx, `INSERT INTO inscricao_evento
		(inscricao_id, evento, de_estado, para_estado, ator_tipo, ator_id, edital_versao, hash)
		VALUES ($1, 'PROVER_RECURSO', 'INDEFERIDA', 'DEFERIDA', 'analista', 'a', 1, '\x00')`, id)
	must(t, err)
	if q := verificar(t, id); len(q) != 0 {
		t.Fatalf("o gatilho devia ter substituído o resumo enviado: %v", q)
	}
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
