package publicacao_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"lisura/internal/conta"
	"lisura/internal/dbtest"
	"lisura/internal/inscricao"
	"lisura/internal/publicacao"
	"lisura/internal/seed"
)

var pool *pgxpool.Pool

func TestMain(m *testing.M) {
	p, err := dbtest.Pool(context.Background(), "t_publicacao")
	if err != nil {
		dbtest.SemBanco(err)
	}
	pool = p
	code := m.Run()
	if pool != nil {
		pool.Close()
	}
	os.Exit(code)
}

var seq atomic.Int64

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// inscricaoEnviada devolve uma inscrição aguardando pagamento, o certame e o CPF do candidato.
func inscricaoEnviada(t *testing.T) (insc, cert int64, cpf string) {
	t.Helper()
	if pool == nil {
		t.Skip("sem banco de teste")
	}
	ctx := context.Background()
	c, err := seed.CriarCertame(ctx, pool, seed.Padrao())
	must(t, err)
	cpf = conta.GerarCPF(seq.Add(1) + time.Now().UnixNano()%1_000_000*100)
	cand, err := conta.Criar(ctx, pool, cpf, "Rita Sintética", "rita@exemplo.test")
	must(t, err)
	must(t, conta.AtualizarDados(ctx, pool, cand, conta.Dados{Nome: "Rita Sintética", Email: "rita@exemplo.test",
		DataNascimento: "1988-08-08", Municipio: "Sorocaba", UF: "SP"}))
	insc, _, err = inscricao.Abrir(ctx, pool, c.ID, cand, c.Cargos[2], inscricao.Ator{Tipo: inscricao.Candidato, ID: "r"})
	must(t, err)
	must(t, inscricao.DefinirCidade(ctx, pool, insc, cand, c.Cidades[1]))
	must(t, inscricao.AceitarDeclaracoes(ctx, pool, insc, cand, true))
	_, err = inscricao.Finalizar(ctx, pool, insc, cand)
	must(t, err)
	return insc, c.ID, cpf
}

func ler(t *testing.T, dir, caminho string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(caminho)))
	must(t, err)
	return b
}

// Sem pagamento, sem cartão; paga, o cartão sai com o nome e o CPF mascarado — nunca o CPF inteiro.
func TestCartaoSoDepoisDeValer(t *testing.T) {
	insc, _, cpf := inscricaoEnviada(t)
	p := &publicacao.Publicador{Pool: pool, Dir: t.TempDir(), Chave: []byte("k")}
	ctx := context.Background()

	must(t, p.PublicarCartao(ctx, insc))
	if p.Existe(ctx, p.CaminhoCartao(insc)) {
		t.Fatal("cartão publicado antes do pagamento")
	}
	_, err := inscricao.Transicionar(ctx, pool, inscricao.Comando{InscricaoID: insc, Evento: inscricao.ConfirmarPagamento,
		Ator: inscricao.Ator{Tipo: inscricao.Sistema, ID: "teste"}})
	must(t, err)
	must(t, p.PublicarCartao(ctx, insc))
	pagina := ler(t, p.Dir, p.CaminhoCartao(insc))
	for _, deve := range []string{"Rita Sintética", "Enfermeiro", "Sorocaba", "Paga", "***." + cpf[3:6] + "." + cpf[6:9] + "-**"} {
		if !bytes.Contains(pagina, []byte(deve)) {
			t.Errorf("cartão sem %q", deve)
		}
	}
	if bytes.Contains(pagina, []byte(cpf)) {
		t.Error("cartão expõe o CPF inteiro")
	}
	// O selo impresso é o começo do resumo do último evento da trilha, e confere nela.
	selo, err := inscricao.Selo(ctx, pool, insc)
	must(t, err)
	if !bytes.Contains(pagina, []byte("selo da trilha "+selo[:16])) {
		t.Errorf("cartão sem o selo %s", selo[:16])
	}
	if ok, err := inscricao.ConfereSelo(ctx, pool, insc, selo[:16]); err != nil || !ok {
		t.Errorf("selo do cartão não confere: %v %v", ok, err)
	}
}

// A relação pública mostra número, cargo e situação — e nada que identifique a pessoa. Sai em fatias,
// com um índice que diz onde está cada uma.
func TestRelacaoSemDadoPessoal(t *testing.T) {
	insc, cert, cpf := inscricaoEnviada(t)
	p := &publicacao.Publicador{Pool: pool, Dir: t.TempDir(), Chave: []byte("k")}
	ctx := context.Background()
	must(t, p.PublicarCertame(ctx, cert))
	fatia := insc / publicacao.TamanhoFatia
	must(t, p.PublicarFatia(ctx, cert, fatia))
	id := strconv.FormatInt(cert, 10)
	indice := ler(t, p.Dir, "certames/"+id+"/relacao.json")
	if !bytes.Contains(indice, []byte(`"url":"/publico/`+publicacao.CaminhoFatia(cert, fatia)+`"`)) {
		t.Fatalf("o índice não aponta a fatia da inscrição: %s", indice)
	}
	relacao := ler(t, p.Dir, publicacao.CaminhoFatia(cert, fatia))
	if !bytes.Contains(relacao, []byte(`"numero":`+strconv.FormatInt(insc, 10))) || !bytes.Contains(relacao, []byte("Aguardando pagamento")) {
		t.Fatalf("relação sem a inscrição: %s", relacao)
	}
	for _, nao := range []string{cpf, "Rita", "rita@"} {
		if bytes.Contains(relacao, []byte(nao)) || bytes.Contains(indice, []byte(nao)) {
			t.Errorf("relação expõe %q", nao)
		}
	}
	if !bytes.Contains(ler(t, p.Dir, "certames/"+id+".json"), []byte(`"cargos"`)) {
		t.Fatal("dados públicos do certame não publicados")
	}
}

// A mudança de uma inscrição pede a republicação só da fatia dela, não da relação inteira: com uma
// inscrição por fatia, mudar a primeira deixa a fatia da segunda quieta. A fatia de cada uma tem só a
// inscrição dela, e o índice aponta as duas.
func TestMudancaRefazSoAFatiaDela(t *testing.T) {
	antes := publicacao.TamanhoFatia
	publicacao.TamanhoFatia = 1
	t.Cleanup(func() { publicacao.TamanhoFatia = antes })
	ctx := context.Background()
	a, cert, _ := inscricaoEnviada(t)
	b := outraEnviada(t, cert)
	p := &publicacao.Publicador{Pool: pool, Dir: t.TempDir(), Chave: []byte("k")}
	fatias := func() map[string]string {
		rows, err := pool.Query(ctx, `SELECT chave, estado FROM tarefa WHERE tipo = $1 AND chave LIKE $2`,
			publicacao.TarefaRelacao, strconv.FormatInt(cert, 10)+":%")
		must(t, err)
		out := map[string]string{}
		for rows.Next() {
			var k, e string
			must(t, rows.Scan(&k, &e))
			out[k] = e
		}
		must(t, rows.Err())
		return out
	}
	chave := func(insc int64) string { return strconv.FormatInt(cert, 10) + ":" + strconv.FormatInt(insc, 10) }
	if f := fatias(); len(f) != 2 || f[chave(a)] == "" || f[chave(b)] == "" {
		t.Fatalf("as duas finalizações não pediram cada uma a sua fatia: %v", f)
	}
	_, err := pool.Exec(ctx, `UPDATE tarefa SET estado = 'CONCLUIDA' WHERE tipo = $1`, publicacao.TarefaRelacao)
	must(t, err)

	// A primeira muda de situação.
	tx, err := pool.Begin(ctx)
	must(t, err)
	_, err = tx.Exec(ctx, `UPDATE inscricao SET estado = 'PAGA' WHERE id = $1`, a)
	must(t, err)
	must(t, publicacao.EnfileirarInscricao(ctx, tx, a, cert))
	must(t, tx.Commit(ctx))
	if f := fatias(); f[chave(a)] != "PENDENTE" || f[chave(b)] != "CONCLUIDA" {
		t.Fatalf("a mudança de uma inscrição mexeu na fatia da outra: %v", f)
	}

	must(t, p.PublicarFatia(ctx, cert, a))
	must(t, p.PublicarFatia(ctx, cert, b))
	fa, fb := ler(t, p.Dir, publicacao.CaminhoFatia(cert, a)), ler(t, p.Dir, publicacao.CaminhoFatia(cert, b))
	na, nb := []byte(`"numero":`+strconv.FormatInt(a, 10)), []byte(`"numero":`+strconv.FormatInt(b, 10))
	if !bytes.Contains(fa, na) || bytes.Contains(fa, nb) || !bytes.Contains(fb, nb) || !bytes.Contains(fa, []byte(`"Paga"`)) {
		t.Fatalf("fatias:\n%s\n%s", fa, fb)
	}
	indice := ler(t, p.Dir, "certames/"+strconv.FormatInt(cert, 10)+"/relacao.json")
	for _, f := range []int64{a, b} {
		if !bytes.Contains(indice, []byte(publicacao.CaminhoFatia(cert, f))) {
			t.Errorf("o índice não aponta a fatia %d: %s", f, indice)
		}
	}
}

// Uma fatia sem anotação (inscrição de antes da 00019, ou de outro tamanho de fatia) é anotada na
// subida do servidor, e o índice passa a apontá-la.
func TestRepublicarTodosAnotaAFatiaQueFalta(t *testing.T) {
	insc, cert, _ := inscricaoEnviada(t)
	ctx := context.Background()
	_, err := pool.Exec(ctx, `DELETE FROM relacao_fatia WHERE certame_id = $1`, cert)
	must(t, err)
	must(t, publicacao.RepublicarTodos(ctx, pool))
	p := &publicacao.Publicador{Pool: pool, Dir: t.TempDir(), Chave: []byte("k")}
	must(t, p.PublicarCertame(ctx, cert))
	indice := ler(t, p.Dir, "certames/"+strconv.FormatInt(cert, 10)+"/relacao.json")
	if !bytes.Contains(indice, []byte(publicacao.CaminhoFatia(cert, insc/publicacao.TamanhoFatia))) {
		t.Fatalf("a fatia sem anotação ficou fora do índice: %s", indice)
	}
}

// outraEnviada abre e finaliza mais uma inscrição no mesmo certame.
func outraEnviada(t *testing.T, cert int64) int64 {
	t.Helper()
	ctx := context.Background()
	var cargo, cidade int64
	must(t, pool.QueryRow(ctx, `SELECT (SELECT min(id) FROM cargo WHERE certame_id = $1), (SELECT min(id) FROM cidade_prova WHERE certame_id = $1)`,
		cert).Scan(&cargo, &cidade))
	cand, err := conta.Criar(ctx, pool, conta.GerarCPF(seq.Add(1)+time.Now().UnixNano()%1_000_000*100), "Outra", "o@exemplo.test")
	must(t, err)
	must(t, conta.AtualizarDados(ctx, pool, cand, conta.Dados{Nome: "Outra", Email: "o@exemplo.test",
		DataNascimento: "1990-01-01", Municipio: "Campinas", UF: "SP"}))
	id, _, err := inscricao.Abrir(ctx, pool, cert, cand, cargo, inscricao.Ator{Tipo: inscricao.Candidato, ID: "o"})
	must(t, err)
	must(t, inscricao.DefinirCidade(ctx, pool, id, cand, cidade))
	must(t, inscricao.AceitarDeclaracoes(ctx, pool, id, cand, true))
	_, err = inscricao.Finalizar(ctx, pool, id, cand)
	must(t, err)
	return id
}

// O endereço do cartão não se deduz do número de inscrição.
func TestTokenNaoSeDeduzDoNumero(t *testing.T) {
	a := &publicacao.Publicador{Chave: []byte("segredo-a")}
	b := &publicacao.Publicador{Chave: []byte("segredo-b")}
	if a.Token(1) == a.Token(2) || a.Token(1) == b.Token(1) || len(a.Token(1)) != 32 {
		t.Fatalf("tokens: %s %s %s", a.Token(1), a.Token(2), b.Token(1))
	}
}

// Servir entrega o que foi publicado com cache público, e não entrega o que não existe.
func TestServirArquivoPublicado(t *testing.T) {
	dir := t.TempDir()
	must(t, os.MkdirAll(filepath.Join(dir, "certames"), 0o750))
	must(t, os.WriteFile(filepath.Join(dir, "certames", "9.json"), []byte(`{"id":9}`), 0o640))
	srv := httptest.NewServer(http.StripPrefix("/publico/", publicacao.Servir(dir)))
	defer srv.Close()

	res, err := http.Get(srv.URL + "/publico/certames/9.json")
	must(t, err)
	corpo, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || string(corpo) != `{"id":9}` || res.Header.Get("Cache-Control") != "public, max-age=60" {
		t.Fatalf("%d %q %q", res.StatusCode, corpo, res.Header.Get("Cache-Control"))
	}
	res, err = http.Get(srv.URL + "/publico/certames/10.json")
	must(t, err)
	res.Body.Close()
	if res.StatusCode != 404 {
		t.Fatalf("arquivo inexistente: %d", res.StatusCode)
	}
	// A pasta não se lista: senão o endereço de todos os cartões sairia numa página só.
	res, err = http.Get(srv.URL + "/publico/certames/")
	must(t, err)
	res.Body.Close()
	if res.StatusCode != 404 {
		t.Fatalf("a pasta foi listada: %d", res.StatusCode)
	}
}
