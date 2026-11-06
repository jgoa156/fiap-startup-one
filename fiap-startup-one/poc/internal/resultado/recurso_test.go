package resultado_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"lisura/internal/inscricao"
	"lisura/internal/resultado"
	"lisura/internal/seed"
)

func abrirRecursos(t *testing.T, certameID int64) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `INSERT INTO certame_fase (certame_id, fase, abre_em, fecha_em)
		VALUES ($1, 'gabarito', now() - interval '1 hour', now() + interval '1 day')`, certameID)
	must(t, err)
}

func candidatoDe(t *testing.T, insc int64) int64 {
	t.Helper()
	var c int64
	must(t, pool.QueryRow(context.Background(), `SELECT candidato_id FROM inscricao WHERE id = $1`, insc).Scan(&c))
	return c
}

// O efeito dominó do 02-produto §6: dezenas de recursos sobre a mesma questão viram uma linha para a
// banca, uma decisão, e a mesma resposta para cada recorrente; o definitivo só sai com tudo decidido.
func TestMuitosRecursosUmaDecisaoPorQuestao(t *testing.T) {
	ctx := context.Background()
	c, err := seed.CriarCertame(ctx, pool, seed.Padrao())
	must(t, err)
	cargo := c.Cargos[2]
	ids := inscrever(t, c.ID, cargo, homologadas(60, nil))
	abrirRecursos(t, c.ID)
	_, err = resultado.PublicarGabarito(ctx, pool, cargo, "preliminar", gabarito, "Gabarito preliminar")
	must(t, err)

	// 50 recursos sobre a questão 3 (30 pedem a letra B, 20 dizem que o enunciado é ambíguo) e 10
	// sobre a 7, todos ao mesmo tempo.
	var wg sync.WaitGroup
	for i, id := range ids {
		var p resultado.Pedido
		switch {
		case i < 30:
			p = resultado.Pedido{Numero: 3, Fundamento: "gabarito_errado", RespostaPedida: "b", Argumento: "A alternativa B está correta."}
		case i < 50:
			p = resultado.Pedido{Numero: 3, Fundamento: "enunciado_ambiguo", Argumento: "O enunciado admite duas leituras."}
		default:
			p = resultado.Pedido{Numero: 7, Fundamento: "nenhuma_correta", Argumento: "Nenhuma alternativa responde."}
		}
		cand := candidatoDe(t, id)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := resultado.Recorrer(ctx, pool, id, cand, p); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()

	versao, grupos, err := resultado.Agrupar(ctx, pool, cargo)
	must(t, err)
	if versao != 1 || len(grupos) != 2 {
		t.Fatalf("versão %d, grupos %+v", versao, grupos)
	}
	q3, q7 := grupos[0], grupos[1]
	if q3.Numero != 3 || q3.Recursos != 50 || q3.PorFundamento["gabarito_errado"] != 30 ||
		q3.PorFundamento["enunciado_ambiguo"] != 20 || q3.Pedidas["B"] != 30 || q3.Disciplina != seed.Portugues {
		t.Errorf("questão 3: %+v", q3)
	}
	if q7.Numero != 7 || q7.Recursos != 10 || q7.Decisao != nil {
		t.Errorf("questão 7: %+v", q7)
	}

	if _, _, _, err := resultado.Consolidar(ctx, pool, cargo); !errors.Is(err, resultado.ErrDecisoesPendente) {
		t.Errorf("consolidar sem decisão: %v", err)
	}
	must(t, resultado.Decidir(ctx, pool, cargo, 3, "altera", "b", "Procede: a alternativa B é a correta.", "banca-portugues"))
	if err := resultado.Decidir(ctx, pool, cargo, 3, "mantem", "", "Outra opinião.", "banca-portugues"); !errors.Is(err, resultado.ErrJaDecidida) {
		t.Errorf("decidir de novo: %v", err)
	}
	if _, _, _, err := resultado.Consolidar(ctx, pool, cargo); !errors.Is(err, resultado.ErrDecisoesPendente) {
		t.Errorf("consolidar com a 7 pendente: %v", err)
	}
	must(t, resultado.Decidir(ctx, pool, cargo, 7, "anula", "", "Procede: nenhuma alternativa responde.", "banca-portugues"))

	// Cada recorrente recebe a decisão da questão dele.
	for _, i := range []int{0, 35, 55} {
		rs, err := resultado.MeusRecursos(ctx, pool, ids[i], candidatoDe(t, ids[i]))
		must(t, err)
		if len(rs) != 1 || rs[0].Decisao == nil {
			t.Fatalf("recurso %d sem resposta: %+v", i, rs)
		}
		esperada := map[int]string{0: "altera", 35: "altera", 55: "anula"}[i]
		if *rs[0].Decisao != esperada || rs[0].Fundamentacao == nil {
			t.Errorf("recurso %d: decisão %s, esperado %s", i, *rs[0].Decisao, esperada)
		}
	}

	v, alteradas, anuladas, err := resultado.Consolidar(ctx, pool, cargo)
	must(t, err)
	var resp, tipo string
	must(t, pool.QueryRow(ctx, `SELECT respostas, tipo FROM gabarito WHERE cargo_id = $1 AND versao = $2`, cargo, v).Scan(&resp, &tipo))
	if v != 2 || alteradas != 1 || anuladas != 1 || tipo != "definitivo" || resp[2] != 'B' || resp[6] != '*' ||
		resp[:2]+resp[3:6]+resp[7:] != gabarito[:2]+gabarito[3:6]+gabarito[7:] {
		t.Errorf("definitivo: versão %d %s %q (%d alteradas, %d anuladas)", v, tipo, resp, alteradas, anuladas)
	}

	// Fechado: nada mais entra no preliminar.
	if _, err := resultado.Recorrer(ctx, pool, ids[59], candidatoDe(t, ids[59]),
		resultado.Pedido{Numero: 9, Fundamento: "fora_do_conteudo", Argumento: "x"}); !errors.Is(err, resultado.ErrSemPreliminar) {
		t.Errorf("recurso depois do definitivo: %v", err)
	}
	if err := resultado.Decidir(ctx, pool, cargo, 9, "anula", "", "x", "y"); !errors.Is(err, resultado.ErrSemPreliminar) {
		t.Errorf("decisão depois do definitivo: %v", err)
	}
	if _, err := resultado.PublicarGabarito(ctx, pool, cargo, "preliminar", gabarito, "volta"); !errors.Is(err, resultado.ErrGabaritoInvalido) {
		t.Errorf("preliminar depois do definitivo: %v", err)
	}
}

func TestRecursoRecusado(t *testing.T) {
	ctx := context.Background()
	c, err := seed.CriarCertame(ctx, pool, seed.Padrao())
	must(t, err)
	cargo := c.Cargos[0]
	ids := inscrever(t, c.ID, cargo, []pessoa{{estado: "HOMOLOGADA"}, {estado: "HOMOLOGADA"}, {estado: "PAGA"}})
	a, b := candidatoDe(t, ids[0]), candidatoDe(t, ids[1])
	ok := resultado.Pedido{Numero: 1, Fundamento: "mais_de_uma_correta", Argumento: "A e C estão corretas."}

	if _, err := resultado.Recorrer(ctx, pool, ids[0], a, ok); !errors.Is(err, inscricao.ErrForaDoPrazo) {
		t.Errorf("sem janela de recurso: %v", err)
	}
	abrirRecursos(t, c.ID)
	if _, err := resultado.Recorrer(ctx, pool, ids[0], a, ok); !errors.Is(err, resultado.ErrSemGabarito) {
		t.Errorf("sem gabarito: %v", err)
	}
	_, err = resultado.PublicarGabarito(ctx, pool, cargo, "preliminar", gabarito, "Gabarito preliminar")
	must(t, err)

	for nome, caso := range map[string]struct {
		insc, cand int64
		p          resultado.Pedido
		erro       error
	}{
		"inscrição de outro":     {ids[0], b, ok, inscricao.ErrNaoEncontrada},
		"não homologada":         {ids[2], candidatoDe(t, ids[2]), ok, inscricao.ErrTransicaoInvalida},
		"questão inexistente":    {ids[0], a, resultado.Pedido{Numero: 21, Fundamento: "nenhuma_correta", Argumento: "x"}, resultado.ErrRecursoInvalido},
		"fundamento inventado":   {ids[0], a, resultado.Pedido{Numero: 1, Fundamento: "nao_gostei", Argumento: "x"}, resultado.ErrRecursoInvalido},
		"pede letra sem dizer":   {ids[0], a, resultado.Pedido{Numero: 1, Fundamento: "gabarito_errado", Argumento: "x"}, resultado.ErrRecursoInvalido},
		"letra com outro motivo": {ids[0], a, resultado.Pedido{Numero: 1, Fundamento: "nenhuma_correta", RespostaPedida: "C", Argumento: "x"}, resultado.ErrRecursoInvalido},
		"letra fora de A–E":      {ids[0], a, resultado.Pedido{Numero: 1, Fundamento: "gabarito_errado", RespostaPedida: "Z", Argumento: "x"}, resultado.ErrRecursoInvalido},
		"sem argumento":          {ids[0], a, resultado.Pedido{Numero: 1, Fundamento: "nenhuma_correta", Argumento: "  "}, resultado.ErrRecursoInvalido},
	} {
		if _, err := resultado.Recorrer(ctx, pool, caso.insc, caso.cand, caso.p); !errors.Is(err, caso.erro) {
			t.Errorf("%s: %v, esperado %v", nome, err, caso.erro)
		}
	}
	_, err = resultado.Recorrer(ctx, pool, ids[0], a, ok)
	must(t, err)
	if _, err := resultado.Recorrer(ctx, pool, ids[0], a, ok); !errors.Is(err, resultado.ErrRecursoRepetido) {
		t.Errorf("segundo recurso na mesma questão: %v", err)
	}
	if err := resultado.Decidir(ctx, pool, cargo, 1, "altera", "A", "x", "y"); !errors.Is(err, resultado.ErrRecursoInvalido) {
		t.Errorf("alterar para a mesma letra: %v", err)
	}
	must(t, resultado.Decidir(ctx, pool, cargo, 1, "mantem", "", "Improcede.", "banca"))
	if _, err := pool.Exec(ctx, `UPDATE decisao_questao SET decisao = 'anula' WHERE cargo_id = $1`, cargo); err == nil {
		t.Error("decisão gravada foi alterada")
	}
}

// Dois operadores fecham os recursos ao mesmo tempo: sai um definitivo só.
func TestConsolidacoesConcorrentesUmDefinitivo(t *testing.T) {
	ctx := context.Background()
	c, err := seed.CriarCertame(ctx, pool, seed.Padrao())
	must(t, err)
	cargo := c.Cargos[1]
	_, err = resultado.PublicarGabarito(ctx, pool, cargo, "preliminar", gabarito, "Gabarito preliminar")
	must(t, err)
	// Para as 6 se sobreporem de verdade (sozinhas, terminam antes da próxima começar), uma transação
	// do teste segura uma versão 2 não confirmada: quem tenta gravar a versão 2 — ou travar o cargo —
	// espera por ela. Desfeita a transação, as 6 seguem juntas.
	segura, err := pool.Begin(ctx)
	must(t, err)
	_, err = segura.Exec(ctx, `INSERT INTO gabarito (cargo_id, versao, tipo, respostas, motivo)
		VALUES ($1, 2, 'definitivo', $2, 'segura')`, cargo, gabarito)
	must(t, err)
	var ok, recusadas atomic.Int32
	var wg sync.WaitGroup
	for range 6 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, _, err := resultado.Consolidar(ctx, pool, cargo)
			switch {
			case err == nil:
				ok.Add(1)
			case errors.Is(err, resultado.ErrSemPreliminar):
				recusadas.Add(1)
			default:
				t.Error(err)
			}
		}()
	}
	time.Sleep(300 * time.Millisecond)
	must(t, segura.Rollback(ctx))
	wg.Wait()
	var definitivos int
	must(t, pool.QueryRow(ctx, `SELECT count(*) FROM gabarito WHERE cargo_id = $1 AND tipo = 'definitivo'`, cargo).Scan(&definitivos))
	if ok.Load() != 1 || recusadas.Load() != 5 || definitivos != 1 {
		t.Errorf("%d consolidações, %d recusadas, %d definitivos", ok.Load(), recusadas.Load(), definitivos)
	}
}
