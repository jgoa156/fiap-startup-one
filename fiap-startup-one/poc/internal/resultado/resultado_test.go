package resultado_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"lisura/internal/conta"
	"lisura/internal/dbtest"
	"lisura/internal/publicacao"
	"lisura/internal/resultado"
	"lisura/internal/seed"
)

var pool *pgxpool.Pool

func TestMain(m *testing.M) {
	p, err := dbtest.Pool(context.Background(), "t_resultado")
	if err != nil {
		dbtest.SemBanco(err)
		os.Exit(0)
	}
	pool = p
	code := m.Run()
	pool.Close()
	os.Exit(code)
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

var seq atomic.Int64

type pessoa struct {
	estado     string
	cotas      []string
	nascimento string // vazio: sem data
}

// inscrever grava candidatos e inscrições direto no banco, já no estado pedido: o caminho até lá
// (formulário, pagamento) tem os próprios testes; aqui importa o que vem depois.
func inscrever(t *testing.T, certameID, cargoID int64, ps []pessoa) []int64 {
	t.Helper()
	ctx := context.Background()
	var ids []int64
	for _, p := range ps {
		cpf := conta.GerarCPF(seq.Add(1) + time.Now().UnixNano()%1_000_000*1000)
		var cand, insc int64
		var nasc *string
		if p.nascimento != "" {
			nasc = &p.nascimento
		}
		must(t, pool.QueryRow(ctx, `INSERT INTO candidato (cpf, nome, email, data_nascimento)
			VALUES ($1, 'Pessoa Sintética', 'p@exemplo.test', $2::date) RETURNING id`, cpf, nasc).Scan(&cand))
		must(t, pool.QueryRow(ctx, `INSERT INTO inscricao (certame_id, candidato_id, cargo_id, estado)
			VALUES ($1, $2, $3, $4) RETURNING id`, certameID, cand, cargoID, p.estado).Scan(&insc))
		for _, c := range p.cotas {
			_, err := pool.Exec(ctx, `INSERT INTO inscricao_cota (inscricao_id, certame_id, cota_codigo) VALUES ($1, $2, $3)`,
				insc, certameID, c)
			must(t, err)
		}
		ids = append(ids, insc)
	}
	return ids
}

func homologadas(n int, cotas func(i int) []string) []pessoa {
	ps := make([]pessoa, n)
	for i := range ps {
		ps[i] = pessoa{estado: "HOMOLOGADA", nascimento: "1990-01-01"}
		if cotas != nil {
			ps[i].cotas = cotas(i)
		}
	}
	return ps
}

const gabarito = "ABCDEABCDEABCDEABCDE"

// folha acerta as `acertos` primeiras questões e erra o resto.
func folha(acertos int) string {
	b := []byte(gabarito)
	for i := acertos; i < len(b); i++ {
		b[i] = map[byte]byte{'A': 'B', 'B': 'C', 'C': 'D', 'D': 'E', 'E': 'A'}[b[i]]
	}
	return string(b)
}

func ler[T any](t *testing.T, dir, caminho string) T {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(caminho)))
	must(t, err)
	var v T
	must(t, json.Unmarshal(b, &v))
	return v
}

type listaPublica struct {
	Versao int    `json:"versao"`
	Tipo   string `json:"tipo"`
	Cargos []struct {
		CargoID         int64          `json:"cargo_id"`
		VagasReservadas map[string]int `json:"vagas_reservadas"`
		Ampla           []struct {
			Posicao int     `json:"posicao"`
			Numero  int64   `json:"numero"`
			Nota    float64 `json:"nota"`
			Vaga    string  `json:"vaga"`
		} `json:"ampla_concorrencia"`
		Cotas map[string][]struct {
			Posicao int   `json:"posicao"`
			Numero  int64 `json:"numero"`
		} `json:"cotas"`
	} `json:"cargos"`
	Historico []struct {
		Versao int `json:"versao"`
	} `json:"historico"`
}

type fatia struct {
	Versao     int                             `json:"versao"`
	Candidatos map[string]resultado.Individual `json:"candidatos"`
}

func individual(t *testing.T, pub *publicacao.Publicador, certameID, insc int64) (resultado.Individual, int) {
	t.Helper()
	chave := resultado.ChaveResultado(pub.Token(insc))
	f := ler[fatia](t, pub.Dir, resultado.CaminhoFatia(certameID, chave))
	ind, ok := f.Candidatos[chave]
	if !ok {
		t.Fatalf("inscrição %d não está na fatia", insc)
	}
	return ind, f.Versao
}

// O caminho inteiro do pico 5: folhas lidas, gabarito preliminar, cálculo, publicação; depois o
// gabarito definitivo anula uma questão, tudo é recalculado e republicado, e a versão anterior fica.
func TestResultadoPontaAPonta(t *testing.T) {
	ctx := context.Background()
	c, err := seed.CriarCertame(ctx, pool, seed.Padrao())
	must(t, err)
	enfermeiro := c.Cargos[2] // 10 vagas: 6 ampla, 1 pcd (0,5 sobe para 1), 3 ppiq (30%)
	// 20 candidatos: os de índice ímpar até 9 são ppiq, o 19 é pcd. O índice i (até 17) acerta as
	// 20−i primeiras questões; 1–10 valem 1 ponto, 11–20 valem 2, nota mínima 12. Aprovados: 0 a 9
	// (o 9 tem 11 acertos = 10 + 2). O 18 faltou; o pcd acertou 9 e fica abaixo da mínima.
	ids := inscrever(t, c.ID, enfermeiro, homologadas(20, func(i int) []string {
		switch {
		case i == 19:
			return []string{"pcd"}
		case i%2 == 1 && i <= 9:
			return []string{"ppiq"}
		}
		return nil
	}))
	var folhas []resultado.Folha
	for i, id := range ids[:18] {
		folhas = append(folhas, resultado.Folha{Inscricao: id, Respostas: folha(20 - i)})
	}
	folhas = append(folhas, resultado.Folha{Inscricao: ids[19], Respostas: folha(9)})
	imp, err := resultado.ImportarFolhas(ctx, pool, "lote-1", folhas)
	must(t, err)
	if imp.Gravadas != 19 || len(imp.Recusadas) != 0 {
		t.Fatalf("importação: %+v", imp)
	}
	for _, cg := range c.Cargos {
		_, err := resultado.PublicarGabarito(ctx, pool, cg, "preliminar", gabarito, "Gabarito preliminar")
		must(t, err)
	}
	res, err := resultado.CalcularCertame(ctx, pool, c.ID, "preliminar", "Resultado preliminar")
	must(t, err)
	if res.Versao != 1 {
		t.Fatalf("versão %d", res.Versao)
	}
	rc := res.PorCargo[enfermeiro]
	if rc.PorSituacao["CLASSIFICADO"] != 10 || rc.PorSituacao["AUSENTE"] != 1 || rc.PorSituacao["ELIMINADO"] != 9 || rc.Candidatos != 20 {
		t.Errorf("resumo do cargo: %+v", rc)
	}

	dir := t.TempDir()
	pub := &publicacao.Publicador{Pool: pool, Dir: dir, Chave: []byte("chave-de-teste")}
	rp := &resultado.Publicador{Pool: pool, Pub: pub}
	must(t, rp.PublicarResultado(ctx, c.ID))
	must(t, rp.PublicarGabarito(ctx, c.ID))

	lista := ler[listaPublica](t, dir, "certames/"+itoa(c.ID)+"/resultado.json")
	var cargo = lista.Cargos[2]
	if cargo.CargoID != enfermeiro {
		t.Fatalf("cargos fora de ordem: %+v", lista.Cargos)
	}
	if len(cargo.Ampla) != 10 {
		t.Fatalf("ampla com %d linhas, esperado os 10 aprovados", len(cargo.Ampla))
	}
	// Lista geral: 0 a 9 em ordem. As 6 vagas da ampla vão para 0–5. A do pcd não tem pcd aprovado.
	// Das 3 da ppiq, os cotistas 1, 3 e 5 já entraram pela ampla e não a consomem: 7 e 9 ficam com
	// duas, e a terceira volta para a ampla junto com a do pcd: 6 e 8.
	vagas := map[int64]string{}
	for i, l := range cargo.Ampla {
		if l.Numero != ids[i] || l.Posicao != i+1 {
			t.Fatalf("posição %d: %+v", i+1, l)
		}
		vagas[l.Numero] = l.Vaga
	}
	for i, esperado := range []string{"ampla", "ampla", "ampla", "ampla", "ampla", "ampla", "ampla", "ppiq", "ampla", "ppiq"} {
		if vagas[ids[i]] != esperado {
			t.Errorf("índice %d: vaga %q, esperado %q", i, vagas[ids[i]], esperado)
		}
	}
	if len(cargo.Cotas["ppiq"]) != 5 || cargo.Cotas["ppiq"][3].Numero != ids[7] || cargo.Cotas["ppiq"][3].Posicao != 4 {
		t.Errorf("lista ppiq: %+v", cargo.Cotas["ppiq"])
	}

	ind, v := individual(t, pub, c.ID, ids[7])
	if v != 1 || ind.Situacao != "CLASSIFICADO" || *ind.Vaga != "ppiq" || *ind.PosicaoAmpla != 8 || ind.PosicoesCota["ppiq"] != 4 {
		t.Errorf("individual do 7: %+v", ind)
	}
	if ind, _ := individual(t, pub, c.ID, ids[18]); ind.Situacao != "AUSENTE" || ind.Nota != nil {
		t.Errorf("ausente: %+v", ind)
	}
	if ind, _ := individual(t, pub, c.ID, ids[19]); ind.Situacao != "ELIMINADO" || ind.Nota.String() != "9.00" {
		t.Errorf("pcd eliminado: %+v", ind)
	}
	semDadoPessoal(t, dir)

	// Definitivo: a questão 20 (peso 2) é anulada. Todo presente ganha 2 pontos; o pcd vai a 11, ainda
	// abaixo da mínima.
	for _, cg := range c.Cargos {
		_, err := resultado.PublicarGabarito(ctx, pool, cg, "definitivo", gabarito[:19]+"*", "Questão 20 anulada em recurso")
		must(t, err)
	}
	res2, err := resultado.CalcularCertame(ctx, pool, c.ID, "definitivo", "Gabarito definitivo")
	must(t, err)
	if res2.Versao != 2 || res2.PorCargo[enfermeiro].Gabarito != 2 {
		t.Fatalf("recálculo: %+v", res2)
	}
	must(t, rp.PublicarResultado(ctx, c.ID))
	ind, v = individual(t, pub, c.ID, ids[19])
	if v != 2 || ind.Nota.String() != "11.00" {
		t.Errorf("pcd depois da anulação: versão %d, %+v", v, ind)
	}
	lista = ler[listaPublica](t, dir, "certames/"+itoa(c.ID)+"/resultado.json")
	if lista.Versao != 2 || lista.Tipo != "definitivo" || len(lista.Historico) != 2 {
		t.Errorf("lista definitiva: versão %d %s, histórico %d", lista.Versao, lista.Tipo, len(lista.Historico))
	}
	// A versão 1 continua no banco como foi calculada.
	var nota1 float64
	must(t, pool.QueryRow(ctx, `SELECT nota FROM resultado_item WHERE certame_id = $1 AND versao = 1 AND inscricao_id = $2`,
		c.ID, ids[19]).Scan(&nota1))
	if nota1 != 9 {
		t.Errorf("versão 1 mudou: nota %v", nota1)
	}

	g := ler[struct {
		Cargos []struct {
			Atual struct {
				Versao int    `json:"versao"`
				Tipo   string `json:"tipo"`
			} `json:"atual"`
			Questoes []struct {
				Resposta *string `json:"resposta"`
			} `json:"questoes"`
			Historico []json.RawMessage `json:"historico"`
		} `json:"cargos"`
	}](t, dir, "certames/"+itoa(c.ID)+"/gabarito.json")
	// O gabarito publicado antes da anulação ainda mostra a versão 1; republicado, a 2.
	must(t, rp.PublicarGabarito(ctx, c.ID))
	g2 := ler[struct {
		Cargos []struct {
			Atual struct {
				Versao int `json:"versao"`
			} `json:"atual"`
			Questoes []struct {
				Resposta *string `json:"resposta"`
			} `json:"questoes"`
			Historico []json.RawMessage `json:"historico"`
		} `json:"cargos"`
	}](t, dir, "certames/"+itoa(c.ID)+"/gabarito.json")
	if g.Cargos[0].Atual.Versao != 1 || len(g.Cargos[0].Questoes) != 20 || g.Cargos[0].Questoes[19].Resposta == nil {
		t.Errorf("gabarito preliminar publicado: %+v", g.Cargos[0])
	}
	if g2.Cargos[0].Atual.Versao != 2 || g2.Cargos[0].Questoes[19].Resposta != nil || len(g2.Cargos[0].Historico) != 2 {
		t.Errorf("gabarito definitivo publicado: %+v", g2.Cargos[0])
	}
}

// Nenhum arquivo publicado do resultado tem CPF, nome ou e-mail.
func semDadoPessoal(t *testing.T, dir string) {
	t.Helper()
	filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, _ := os.ReadFile(p)
		for _, proibido := range []string{"Sintética", "@exemplo.test", "cpf", "nome\""} {
			if bytes.Contains(b, []byte(proibido)) {
				t.Errorf("%s contém %q", p, proibido)
			}
		}
		return nil
	})
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

func TestDefinitivoExigeGabaritoDefinitivo(t *testing.T) {
	ctx := context.Background()
	c, err := seed.CriarCertame(ctx, pool, seed.Padrao())
	must(t, err)
	for i, cg := range c.Cargos {
		tipo := "definitivo"
		if i == 1 {
			tipo = "preliminar" // um cargo ainda no preliminar
		}
		_, err := resultado.PublicarGabarito(ctx, pool, cg, tipo, gabarito, "teste")
		must(t, err)
	}
	if _, err := resultado.CalcularCertame(ctx, pool, c.ID, "definitivo", "x"); !errors.Is(err, resultado.ErrGabaritoPreliminar) {
		t.Errorf("definitivo com um gabarito preliminar: %v", err)
	}
	var versoes int
	must(t, pool.QueryRow(ctx, `SELECT count(*) FROM resultado_versao WHERE certame_id = $1`, c.ID).Scan(&versoes))
	if versoes != 0 {
		t.Errorf("cálculo recusado deixou %d versões", versoes)
	}
}

func TestGabaritoConfereAProvaEFixaAProva(t *testing.T) {
	ctx := context.Background()
	c, err := seed.CriarCertame(ctx, pool, seed.Padrao())
	must(t, err)
	cg := c.Cargos[0]
	if _, err := resultado.PublicarGabarito(ctx, pool, cg, "preliminar", gabarito[:19], "x"); !errors.Is(err, resultado.ErrGabaritoInvalido) {
		t.Errorf("19 respostas para 20 questões: %v", err)
	}
	if _, err := resultado.PublicarGabarito(ctx, pool, cg, "preliminar", strings.Repeat("F", 20), "x"); !errors.Is(err, resultado.ErrGabaritoInvalido) {
		t.Errorf("letra F: %v", err)
	}
	if _, err := resultado.PublicarGabarito(ctx, pool, cg, "preliminar", gabarito, ""); !errors.Is(err, resultado.ErrGabaritoInvalido) {
		t.Errorf("sem motivo: %v", err)
	}
	v, err := resultado.PublicarGabarito(ctx, pool, cg, "preliminar", gabarito, "Gabarito preliminar")
	must(t, err)
	if v != 1 {
		t.Errorf("primeira versão: %d", v)
	}
	if err := resultado.DefinirProva(ctx, pool, cg, []resultado.Questao{{Disciplina: "X", Peso: 100}}); !errors.Is(err, resultado.ErrProvaFixa) {
		t.Errorf("mudar a prova depois do gabarito: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE gabarito SET respostas = $2 WHERE cargo_id = $1`, cg, strings.Repeat("A", 20)); err == nil {
		t.Error("versão publicada do gabarito foi alterada")
	}
}

func TestImportarFolhasRecusaOQueNaoConfere(t *testing.T) {
	ctx := context.Background()
	c, err := seed.CriarCertame(ctx, pool, seed.Padrao())
	must(t, err)
	ids := inscrever(t, c.ID, c.Cargos[0], []pessoa{
		{estado: "HOMOLOGADA"}, {estado: "HOMOLOGADA"}, {estado: "PAGA"}, {estado: "HOMOLOGADA"},
		{estado: "HOMOLOGADA"}, {estado: "HOMOLOGADA"},
	})
	imp, err := resultado.ImportarFolhas(ctx, pool, "lote-a", []resultado.Folha{
		{ids[0], gabarito},                     // certa
		{ids[1], gabarito[:19]},                // curta
		{ids[2], gabarito},                     // não homologada
		{ids[3], "ABCDEABCDEABCDEABCDx"},       // marcação estranha
		{ids[4], gabarito}, {ids[4], folha(3)}, // duas leituras da mesma folha no lote
		{999_999_999, gabarito},          // não existe
		{ids[5], "..........*........."}, // em branco e rasura: válida
	})
	must(t, err)
	if imp.Gravadas != 2 {
		t.Errorf("gravadas %d, esperado 2", imp.Gravadas)
	}
	for id, motivo := range map[int64]string{
		ids[1]: "número de respostas diferente da prova", ids[2]: "inscrição não homologada",
		ids[3]: "marcação inválida", ids[4]: "inscrição repetida no lote", 999_999_999: "inscrição inexistente",
	} {
		if imp.Recusadas[id] != motivo {
			t.Errorf("inscrição %d: motivo %q, esperado %q", id, imp.Recusadas[id], motivo)
		}
	}
	// Reimportar substitui a leitura.
	imp, err = resultado.ImportarFolhas(ctx, pool, "lote-b", []resultado.Folha{{ids[0], folha(5)}})
	must(t, err)
	var resp, lote string
	must(t, pool.QueryRow(ctx, `SELECT respostas, lote FROM folha_resposta WHERE inscricao_id = $1`, ids[0]).Scan(&resp, &lote))
	if imp.Gravadas != 1 || resp != folha(5) || lote != "lote-b" {
		t.Errorf("reimportação: %+v %q %q", imp, resp, lote)
	}
}

func TestHomologarLotePelaMaquinaDeEstados(t *testing.T) {
	ctx := context.Background()
	c, err := seed.CriarCertame(ctx, pool, seed.Padrao())
	must(t, err)
	ids := inscrever(t, c.ID, c.Cargos[0], []pessoa{
		{estado: "PAGA"}, {estado: "ISENCAO_ATIVA"}, {estado: "EM_ANALISE"}, {estado: "DEFERIDA"},
		{estado: "AGUARDANDO_PAGAMENTO"}, {estado: "RASCUNHO"}, {estado: "INDEFERIDA"},
	})
	n, err := resultado.HomologarLote(ctx, pool, c.ID, "analista-7")
	must(t, err)
	if n != 4 {
		t.Errorf("homologadas %d, esperado 4", n)
	}
	esperado := []string{"HOMOLOGADA", "HOMOLOGADA", "HOMOLOGADA", "HOMOLOGADA", "AGUARDANDO_PAGAMENTO", "RASCUNHO", "INDEFERIDA"}
	eventos := []int{3, 3, 2, 1, 0, 0, 0}
	for i, id := range ids {
		var estado string
		var ev, doAnalista int
		must(t, pool.QueryRow(ctx, `SELECT i.estado, count(e.id), count(e.id) FILTER (WHERE e.ator_id = 'analista-7')
			FROM inscricao i LEFT JOIN inscricao_evento e ON e.inscricao_id = i.id WHERE i.id = $1 GROUP BY i.estado`,
			id).Scan(&estado, &ev, &doAnalista))
		if estado != esperado[i] || ev != eventos[i] {
			t.Errorf("inscrição %d: %s com %d eventos, esperado %s com %d", i, estado, ev, esperado[i], eventos[i])
		}
		if i < 4 && doAnalista < 1 {
			t.Errorf("inscrição %d homologada sem o analista na trilha", i)
		}
	}
	// De novo: nada a fazer, nada gravado.
	n, err = resultado.HomologarLote(ctx, pool, c.ID, "analista-7")
	must(t, err)
	if n != 0 {
		t.Errorf("segunda rodada homologou %d", n)
	}
}

// Recálculos disparados juntos (dois operadores, ou a tarefa repetida) não podem gravar duas vezes a
// mesma versão nem misturar linhas de uma na outra.
func TestCalculosConcorrentesGanhamVersoesDistintas(t *testing.T) {
	ctx := context.Background()
	c, err := seed.CriarCertame(ctx, pool, seed.Padrao())
	must(t, err)
	ids := inscrever(t, c.ID, c.Cargos[1], homologadas(30, nil))
	var folhas []resultado.Folha
	for i, id := range ids {
		folhas = append(folhas, resultado.Folha{Inscricao: id, Respostas: folha(i % 21)})
	}
	_, err = resultado.ImportarFolhas(ctx, pool, "l", folhas)
	must(t, err)
	for _, cg := range c.Cargos {
		_, err := resultado.PublicarGabarito(ctx, pool, cg, "preliminar", gabarito, "x")
		must(t, err)
	}
	var wg sync.WaitGroup
	versoes := make([]int, 5)
	for i := range versoes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := resultado.CalcularCertame(ctx, pool, c.ID, "preliminar", "concorrente")
			if err != nil {
				t.Error(err)
			}
			versoes[i] = r.Versao
		}()
	}
	wg.Wait()
	vistas := map[int]bool{}
	for _, v := range versoes {
		vistas[v] = true
	}
	if len(vistas) != 5 {
		t.Errorf("versões %v", versoes)
	}
	var porVersao []int
	rows, err := pool.Query(ctx, `SELECT count(*) FROM resultado_item WHERE certame_id = $1 GROUP BY versao ORDER BY versao`, c.ID)
	must(t, err)
	for rows.Next() {
		var n int
		must(t, rows.Scan(&n))
		porVersao = append(porVersao, n)
	}
	for _, n := range porVersao {
		if n != 30 {
			t.Errorf("linhas por versão: %v", porVersao)
			break
		}
	}
	if _, err := pool.Exec(ctx, `UPDATE resultado_item SET nota = 30 WHERE certame_id = $1`, c.ID); err == nil {
		t.Error("resultado calculado foi alterado")
	}
}
