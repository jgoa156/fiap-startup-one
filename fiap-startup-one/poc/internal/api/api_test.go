package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"lisura/internal/arquivo"
	"lisura/internal/conta"
	"lisura/internal/dbtest"
	"lisura/internal/fila"
	"lisura/internal/inscricao"
	"lisura/internal/notificacao"
	"lisura/internal/pagamento"
	"lisura/internal/psp"
	"lisura/internal/publicacao"
	"lisura/internal/resultado"
	"lisura/internal/seed"
)

// Testes ponta a ponta: HTTP de verdade, banco de verdade, o mesmo caminho que o ensaio de carga
// vai exercitar.

var (
	pool *pgxpool.Pool
	srv  *httptest.Server
	// Prestador simulado que manda cada aviso 5 vezes ao mesmo tempo, assinado com segredoPSP.
	sim        = &psp.Simulador{Segredo: segredoPSP, AvisosPorPagamento: 5}
	segredoPSP = []byte("segredo-de-teste")
	// O provedor de e-mail dos avisos ao candidato.
	correio = &notificacao.Memoria{}
)

func TestMain(m *testing.M) {
	p, err := dbtest.Pool(context.Background(), "t_api")
	if err != nil {
		dbtest.SemBanco(err)
	} else {
		pool = p
		dir, _ := os.MkdirTemp("", "lisura-arquivos-")
		defer os.RemoveAll(dir)
		simSrv := httptest.NewServer(sim.Handler())
		defer simSrv.Close()
		prestador := &psp.Cliente{URL: simSrv.URL, HTTP: &http.Client{Timeout: 5 * time.Second}}
		publico, _ := os.MkdirTemp("", "lisura-publico-")
		defer os.RemoveAll(publico)
		pub := &publicacao.Publicador{Pool: p, Dir: publico, Chave: []byte("chave-de-teste")}
		s := &Server{Pool: p, Store: arquivo.Disco{Dir: dir},
			Tokens:     Tokens{Chave: []byte("chave-de-teste"), Validade: time.Hour},
			SegredoPSP: segredoPSP, Sandbox: prestador, Publicador: pub}
		// Como no cmd/lisura: a API na raiz (os testes chamam sem o /api) e os estáticos em /publico.
		mux := http.NewServeMux()
		mux.Handle("/publico/", http.StripPrefix("/publico/", publicacao.Servir(publico)))
		mux.Handle("/", s.Routes())
		srv = httptest.NewServer(mux)
		defer srv.Close()
		sim.WebhookURL = srv.URL + "/psp/aviso"
		man := pub.Manipuladores()
		man[pagamento.TarefaCriarNoPrestador] = pagamento.CriarNoPrestador(prestador, p)
		man[pagamento.TarefaDevolver] = pagamento.Devolver(prestador, p)
		man[notificacao.Tarefa] = notificacao.Notificar(correio, p)
		for tipo, m := range (&resultado.Publicador{Pool: p, Pub: pub}).Manipuladores() {
			man[tipo] = m
		}
		trab := &fila.Trabalhador{Pool: p, Manipuladores: man, Espera: 50 * time.Millisecond}
		ctx, parar := context.WithCancel(context.Background())
		defer parar()
		go trab.Rodar(ctx, 2)
	}
	code := m.Run()
	if pool != nil {
		pool.Close()
	}
	os.Exit(code)
}

type cliente struct {
	t     *testing.T
	token string
}

type resposta struct {
	status int
	corpo  map[string]any
}

func (c *cliente) req(metodo, caminho string, corpo any) resposta {
	c.t.Helper()
	var r io.Reader
	if corpo != nil {
		b, _ := json.Marshal(corpo)
		r = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(metodo, srv.URL+caminho, r)
	return c.enviar(req)
}

func (c *cliente) upload(tipo, nome string, conteudo []byte) resposta {
	c.t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	mw.WriteField("tipo", tipo)
	fw, _ := mw.CreateFormFile("arquivo", nome)
	fw.Write(conteudo)
	mw.Close()
	req, _ := http.NewRequest("POST", srv.URL+"/conta/documentos", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return c.enviar(req)
}

func (c *cliente) enviar(req *http.Request) resposta {
	c.t.Helper()
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer res.Body.Close()
	out := resposta{status: res.StatusCode}
	json.NewDecoder(res.Body).Decode(&out.corpo)
	return out
}

func (c *cliente) exigir(r resposta, status int) resposta {
	c.t.Helper()
	if r.status != status {
		c.t.Fatalf("status %d, esperado %d: %v", r.status, status, r.corpo)
	}
	return r
}

func id(r resposta) int64 { return int64(r.corpo["id"].(float64)) }

var seq atomic.Int64

// novoCandidato cria uma conta com CPF sintético válido e dados pessoais completos.
func novoCandidato(t *testing.T) *cliente {
	t.Helper()
	c := &cliente{t: t}
	n := seq.Add(1) + time.Now().UnixNano()%1_000_000*100
	r := c.exigir(c.req("POST", "/contas", map[string]string{
		"cpf": conta.GerarCPF(n), "nome": "Candidata Sintética", "email": "c@exemplo.test"}), 201)
	c.token = r.corpo["token"].(string)
	c.exigir(c.req("PUT", "/conta", map[string]string{
		"nome": "Candidata Sintética", "email": "c@exemplo.test", "data_nascimento": "1995-04-12",
		"municipio": "Campinas", "uf": "sp"}), 204)
	return c
}

func novoCertame(t *testing.T, o seed.Opcoes) seed.Certame {
	t.Helper()
	if pool == nil {
		t.Skip("sem banco de teste")
	}
	c, err := seed.CriarCertame(context.Background(), pool, o)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

var (
	pdf = []byte("%PDF-1.4\n1 0 obj << /Type /Catalog >> endobj\ntrailer << >>\n%%EOF\n")
	png = append([]byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR"), make([]byte, 64)...)
	exe = append([]byte("MZ\x90\x00\x03\x00\x00\x00"), make([]byte, 64)...)
)

func faltando(r resposta) []string {
	var out []string
	for _, f := range r.corpo["faltando"].([]any) {
		out = append(out, f.(string))
	}
	return out
}

func TestInscricaoPontaAPonta(t *testing.T) {
	cert := novoCertame(t, seed.Padrao())
	pub := (&cliente{t: t}).exigir((&cliente{t: t}).req("GET", fmt.Sprintf("/certames/%d", cert.ID), nil), 200)
	if n := len(pub.corpo["cargos"].([]any)); n != 3 {
		t.Fatalf("%d cargos no certame público", n)
	}

	c := &cliente{t: t}
	cpf := conta.GerarCPF(seq.Add(1) + 900_000_000%1_000_000)
	r := c.exigir(c.req("POST", "/contas", map[string]string{"cpf": cpf, "nome": "Ana", "email": "ana@exemplo.test"}), 201)
	c.token = r.corpo["token"].(string)
	c.exigir(c.req("POST", "/contas", map[string]string{"cpf": cpf, "nome": "Ana", "email": "ana@exemplo.test"}), 409)

	base := fmt.Sprintf("/certames/%d/inscricoes", cert.ID)
	insc := id(c.exigir(c.req("POST", base, map[string]int64{"cargo_id": cert.Cargos[0]}), 201))
	if again := id(c.exigir(c.req("POST", base, map[string]int64{"cargo_id": cert.Cargos[0]}), 200)); again != insc {
		t.Fatalf("reabrir devolveu outra inscrição: %d != %d", again, insc)
	}
	ins := fmt.Sprintf("/inscricoes/%d", insc)

	// Finalizar cedo diz exatamente o que falta.
	r = c.exigir(c.req("POST", ins+"/finalizar", nil), 422)
	if got := faltando(r); !slices.Equal(got, []string{"dados_pessoais", "cidade_prova", "declaracoes"}) {
		t.Fatalf("faltando %v", got)
	}

	c.exigir(c.req("PUT", "/conta", map[string]string{"nome": "Ana Souza", "email": "ana@exemplo.test",
		"data_nascimento": "1990-01-31", "municipio": "Campinas", "uf": "SP"}), 204)
	c.exigir(c.req("PUT", ins+"/cargo", map[string]int64{"cargo_id": cert.Cargos[2]}), 204)
	c.exigir(c.req("PUT", ins+"/cidade", map[string]int64{"cidade_id": cert.Cidades[1]}), 204)
	c.exigir(c.req("PUT", ins+"/cotas", map[string][]string{"cotas": {"pcd", "ppiq", "pcd"}}), 204)

	// Cota PcD exige laudo compartilhado com esta inscrição.
	r = c.exigir(c.req("POST", ins+"/finalizar", nil), 422)
	if got := faltando(r); !slices.Equal(got, []string{"laudo", "declaracoes"}) {
		t.Fatalf("faltando %v", got)
	}

	laudo := id(c.exigir(c.upload("laudo", "laudo.pdf", pdf), 201))
	c.exigir(c.req("PUT", ins+"/atendimento", map[string]any{
		"necessidades": []string{"ledor", "tempo_adicional"}, "laudo_documento_id": laudo}), 204)
	c.exigir(c.req("PUT", ins+"/atendimento", map[string]any{"necessidades": []string{"telepatia"}}), 422)
	c.exigir(c.req("PUT", ins+"/declaracoes", map[string]bool{"aceito": true}), 204)
	c.exigir(c.req("PUT", ins+"/declaracoes", map[string]bool{"aceito": false}), 204) // retira o aceite
	if got := faltando(c.exigir(c.req("POST", ins+"/finalizar", nil), 422)); !slices.Equal(got, []string{"declaracoes"}) {
		t.Fatalf("depois de retirar o aceite, faltando %v", got)
	}
	c.exigir(c.req("PUT", ins+"/declaracoes", map[string]bool{"aceito": true}), 204)

	rev := c.exigir(c.req("GET", ins, nil), 200).corpo
	if rev["estado"] != "RASCUNHO" || int64(rev["cargo_id"].(float64)) != cert.Cargos[2] ||
		fmt.Sprint(rev["cotas"]) != "[pcd ppiq]" || int64(rev["laudo_documento_id"].(float64)) != laudo {
		t.Fatalf("revisão inesperada: %v", rev)
	}

	r = c.exigir(c.req("POST", ins+"/finalizar", nil), 200)
	if r.corpo["estado"] != "AGUARDANDO_PAGAMENTO" || r.corpo["reenvio"] != false {
		t.Fatalf("finalizar: %v", r.corpo)
	}
	r = c.exigir(c.req("POST", ins+"/finalizar", nil), 200)
	if r.corpo["reenvio"] != true {
		t.Fatalf("segundo clique não reconhecido como reenvio: %v", r.corpo)
	}
	c.exigir(c.req("PUT", ins+"/cargo", map[string]int64{"cargo_id": cert.Cargos[0]}), 409)

	var eventos []string
	rows, _ := pool.Query(context.Background(),
		`SELECT evento FROM inscricao_evento WHERE inscricao_id = $1 ORDER BY id`, insc)
	for rows.Next() {
		var e string
		rows.Scan(&e)
		eventos = append(eventos, e)
	}
	if !slices.Equal(eventos, []string{"CRIAR", "FINALIZAR_COM_PAGAMENTO"}) {
		t.Fatalf("trilha %v", eventos)
	}
}

func TestIsencaoDeferidaDispensaPagamento(t *testing.T) {
	ctx := context.Background()
	cert := novoCertame(t, seed.Padrao())
	c := novoCandidato(t)
	rota := fmt.Sprintf("/certames/%d/isencao", cert.ID)

	c.exigir(c.req("POST", rota, map[string]string{"hipotese": "cadunico", "nis": "12345678901"}), 422)
	c.exigir(c.req("POST", rota, map[string]string{"hipotese": "astrologia"}), 422)
	c.exigir(c.req("POST", rota, map[string]string{"hipotese": "doador_medula"}), 422)

	nis := nisValido(seq.Add(1) + 1_000_000)
	r := c.exigir(c.req("POST", rota, map[string]string{"hipotese": "cadunico", "nis": nis}), 200)
	if r.corpo["estado"] != "EM_ANALISE" {
		t.Fatalf("isenção completa deveria ir para análise humana: %v", r.corpo)
	}
	if err := inscricao.DecidirIsencao(ctx, pool, id(r), inscricao.Ator{Tipo: inscricao.Candidato, ID: "x"}, true, ""); err == nil {
		t.Fatal("candidato decidiu a própria isenção")
	}
	if err := inscricao.DecidirIsencao(ctx, pool, id(r), inscricao.Ator{Tipo: inscricao.Analista, ID: "analista-7"}, true, ""); err != nil {
		t.Fatal(err)
	}

	// Regra que fecha sozinha: o mesmo NIS por outro CPF no mesmo certame.
	outro := novoCandidato(t)
	r2 := outro.exigir(outro.req("POST", rota, map[string]string{"hipotese": "cadunico", "nis": nis}), 200)
	if r2.corpo["estado"] != "INDEFERIDA" || r2.corpo["motivo_codigo"] != "NIS_DE_OUTRO_CANDIDATO" {
		t.Fatalf("NIS repetido: %v", r2.corpo)
	}

	insc := id(c.exigir(c.req("POST", fmt.Sprintf("/certames/%d/inscricoes", cert.ID),
		map[string]int64{"cargo_id": cert.Cargos[0]}), 201))
	ins := fmt.Sprintf("/inscricoes/%d", insc)
	c.exigir(c.req("PUT", ins+"/cidade", map[string]int64{"cidade_id": cert.Cidades[0]}), 204)
	c.exigir(c.req("PUT", ins+"/declaracoes", map[string]bool{"aceito": true}), 204)
	r = c.exigir(c.req("POST", ins+"/finalizar", nil), 200)
	if r.corpo["estado"] != "ISENCAO_ATIVA" {
		t.Fatalf("isento deveria ir para ISENCAO_ATIVA: %v", r.corpo)
	}
}

func TestUmCandidatoNaoAlcancaOOutro(t *testing.T) {
	cert := novoCertame(t, seed.Padrao())
	outroCert := novoCertame(t, seed.Padrao())
	a, b := novoCandidato(t), novoCandidato(t)

	inscA := id(a.exigir(a.req("POST", fmt.Sprintf("/certames/%d/inscricoes", cert.ID),
		map[string]int64{"cargo_id": cert.Cargos[0]}), 201))
	inscB := id(b.exigir(b.req("POST", fmt.Sprintf("/certames/%d/inscricoes", cert.ID),
		map[string]int64{"cargo_id": cert.Cargos[0]}), 201))
	laudoA := id(a.exigir(a.upload("laudo", "laudo.pdf", pdf), 201))

	b.exigir(b.req("GET", fmt.Sprintf("/inscricoes/%d", inscA), nil), 404)
	b.exigir(b.req("PUT", fmt.Sprintf("/inscricoes/%d/declaracoes", inscA), map[string]bool{"aceito": true}), 404)
	b.exigir(b.req("POST", fmt.Sprintf("/inscricoes/%d/finalizar", inscA), nil), 404)
	// B não compartilha o laudo de A, nem com a inscrição de B.
	b.exigir(b.req("PUT", fmt.Sprintf("/inscricoes/%d/atendimento", inscB),
		map[string]any{"necessidades": []string{"ledor"}, "laudo_documento_id": laudoA}), 422)
	// Cargo, cidade e cota de outro certame são recusados pelo banco.
	a.exigir(a.req("PUT", fmt.Sprintf("/inscricoes/%d/cargo", inscA), map[string]int64{"cargo_id": outroCert.Cargos[0]}), 422)
	a.exigir(a.req("PUT", fmt.Sprintf("/inscricoes/%d/cidade", inscA), map[string]int64{"cidade_id": outroCert.Cidades[0]}), 422)
	a.exigir(a.req("PUT", fmt.Sprintf("/inscricoes/%d/cotas", inscA), map[string][]string{"cotas": {"inexistente"}}), 422)

	anon := &cliente{t: t}
	anon.exigir(anon.req("GET", fmt.Sprintf("/inscricoes/%d", inscA), nil), 401)
	forjado := &cliente{t: t, token: a.token[:len(a.token)-2] + "xx"}
	forjado.exigir(forjado.req("GET", fmt.Sprintf("/inscricoes/%d", inscA), nil), 401)
	outraChave := &cliente{t: t, token: Tokens{Chave: []byte("outra"), Validade: time.Hour}.Emitir(1)}
	outraChave.exigir(outraChave.req("GET", fmt.Sprintf("/inscricoes/%d", inscA), nil), 401)
}

// Pico 5: o candidato pergunta uma vez onde está o resultado e, daí em diante, lê só o arquivo
// estático — sem sessão e sem banco. Outro candidato não recebe o endereço.
func TestResultadoPeloEnderecoEstatico(t *testing.T) {
	cert := novoCertame(t, seed.Padrao())
	a, b := novoCandidato(t), novoCandidato(t)
	insc := id(a.exigir(a.req("POST", fmt.Sprintf("/certames/%d/inscricoes", cert.ID),
		map[string]int64{"cargo_id": cert.Cargos[2]}), 201))
	ctx := context.Background()
	// O caminho até a homologação tem os próprios testes; aqui a inscrição já chega homologada.
	if _, err := pool.Exec(ctx, `UPDATE inscricao SET estado = 'HOMOLOGADA' WHERE id = $1`, insc); err != nil {
		t.Fatal(err)
	}
	gab := strings.Repeat("ABCDE", 4)
	if _, err := resultado.ImportarFolhas(ctx, pool, "l", []resultado.Folha{{Inscricao: insc, Respostas: gab}}); err != nil {
		t.Fatal(err)
	}
	for _, cg := range cert.Cargos {
		if _, err := resultado.PublicarGabarito(ctx, pool, cg, "preliminar", gab, "Gabarito preliminar"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := resultado.CalcularCertame(ctx, pool, cert.ID, "preliminar", "Resultado preliminar"); err != nil {
		t.Fatal(err)
	}

	r := a.exigir(a.req("GET", fmt.Sprintf("/inscricoes/%d/resultado", insc), nil), 200)
	url, chave := r.corpo["url"].(string), r.corpo["chave"].(string)
	b.exigir(b.req("GET", fmt.Sprintf("/inscricoes/%d/resultado", insc), nil), 404)
	anon := &cliente{t: t}
	anon.exigir(anon.req("GET", fmt.Sprintf("/inscricoes/%d/resultado", insc), nil), 401)

	var fatia struct {
		Versao     int                             `json:"versao"`
		Candidatos map[string]resultado.Individual `json:"candidatos"`
	}
	// A lista sai por último: quando ela aparece, as fatias já estão lá.
	ate(t, "a lista do resultado ser publicada pela fila", func() bool {
		res, err := http.Get(srv.URL + r.corpo["lista"].(string))
		if err != nil {
			return false
		}
		res.Body.Close()
		return res.StatusCode == 200
	})
	res, err := http.Get(srv.URL + url) // sem sessão
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("fatia: %v %v", err, res)
	}
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(&fatia); err != nil {
		t.Fatal(err)
	}
	ind, ok := fatia.Candidatos[chave]
	if fatia.Versao != 1 || !ok || ind.Situacao != "CLASSIFICADO" || ind.Nota.String() != "30.00" || *ind.PosicaoAmpla != 1 {
		t.Errorf("linha do candidato: versão %d, %v %+v", fatia.Versao, ok, ind)
	}
}

func TestRecursoPelaAPI(t *testing.T) {
	cert := novoCertame(t, seed.Padrao())
	a, b := novoCandidato(t), novoCandidato(t)
	insc := id(a.exigir(a.req("POST", fmt.Sprintf("/certames/%d/inscricoes", cert.ID),
		map[string]int64{"cargo_id": cert.Cargos[0]}), 201))
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `UPDATE inscricao SET estado = 'HOMOLOGADA' WHERE id = $1`, insc); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO certame_fase (certame_id, fase, abre_em, fecha_em)
		VALUES ($1, 'gabarito', now() - interval '1 hour', now() + interval '1 day')`, cert.ID); err != nil {
		t.Fatal(err)
	}
	caminho := fmt.Sprintf("/inscricoes/%d/recursos", insc)
	// Antes do gabarito: não há o que recorrer.
	pedido := map[string]any{"numero": 4, "fundamento": "gabarito_errado", "resposta_pedida": "C", "argumento": "A alternativa C está correta."}
	a.exigir(a.req("POST", caminho, pedido), 409)
	if _, err := resultado.PublicarGabarito(ctx, pool, cert.Cargos[0], "preliminar", strings.Repeat("ABCDE", 4), "Gabarito preliminar"); err != nil {
		t.Fatal(err)
	}
	a.exigir(a.req("POST", caminho, pedido), 201)
	a.exigir(a.req("POST", caminho, pedido), 409)
	a.exigir(a.req("POST", caminho, map[string]any{"numero": 5, "fundamento": "porque_sim", "argumento": "x"}), 422)
	b.exigir(b.req("POST", caminho, pedido), 404)
	b.exigir(b.req("GET", caminho, nil), 404)

	req, _ := http.NewRequest("GET", srv.URL+caminho, nil)
	req.Header.Set("Authorization", "Bearer "+a.token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var lista []resultado.MeuRecurso
	if err := json.NewDecoder(res.Body).Decode(&lista); err != nil {
		t.Fatal(err)
	}
	if len(lista) != 1 || lista[0].Numero != 4 || lista[0].Decisao != nil || *lista[0].RespostaPedida != "C" {
		t.Errorf("meus recursos: %+v", lista)
	}
}

func TestPrazoEncerradoTrava(t *testing.T) {
	encerrado := seed.Padrao()
	encerrado.Inscricao = seed.Janela{Abre: -48 * time.Hour, Fecha: -time.Minute}
	fechado := novoCertame(t, encerrado)
	c := novoCandidato(t)
	c.exigir(c.req("POST", fmt.Sprintf("/certames/%d/inscricoes", fechado.ID),
		map[string]int64{"cargo_id": fechado.Cargos[0]}), 409)

	// Prazo que fecha com a inscrição em andamento: nenhum passo nem a finalização passam.
	cert := novoCertame(t, seed.Padrao())
	insc := id(c.exigir(c.req("POST", fmt.Sprintf("/certames/%d/inscricoes", cert.ID),
		map[string]int64{"cargo_id": cert.Cargos[0]}), 201))
	c.exigir(c.req("PUT", fmt.Sprintf("/inscricoes/%d/cidade", insc), map[string]int64{"cidade_id": cert.Cidades[0]}), 204)
	c.exigir(c.req("PUT", fmt.Sprintf("/inscricoes/%d/declaracoes", insc), map[string]bool{"aceito": true}), 204)
	if _, err := pool.Exec(context.Background(),
		`UPDATE certame_fase SET fecha_em = now() WHERE certame_id = $1 AND fase = 'inscricao'`, cert.ID); err != nil {
		t.Fatal(err)
	}
	c.exigir(c.req("PUT", fmt.Sprintf("/inscricoes/%d/cidade", insc), map[string]int64{"cidade_id": cert.Cidades[1]}), 409)
	c.exigir(c.req("POST", fmt.Sprintf("/inscricoes/%d/finalizar", insc), nil), 409)
}

func TestUploadConfereConteudo(t *testing.T) {
	novoCertame(t, seed.Padrao()) // só para pular sem banco
	c := novoCandidato(t)
	c.exigir(c.upload("laudo", "foto.png", png), 201)
	c.exigir(c.upload("laudo", "laudo.pdf", exe), 415)  // extensão mente, conteúdo não
	c.exigir(c.upload("selfie", "laudo.pdf", pdf), 422) // tipo fora da lista
	c.exigir(c.upload("laudo", "grande.pdf", append(pdf, make([]byte, conta.MaxArquivo)...)), 413)
}

// O front usa o cookie de sessão; escrita vinda de outro site com esse cookie é recusada.
func TestSessaoPorCookie(t *testing.T) {
	novoCertame(t, seed.Padrao()) // só para pular sem banco
	c := novoCandidato(t)
	cpf := conta.GerarCPF(seq.Add(1) + 700_000_000%1_000_000)
	res, err := http.Post(srv.URL+"/contas", "application/json",
		bytes.NewReader([]byte(`{"cpf":"`+cpf+`","nome":"Bia","email":"bia@exemplo.test"}`)))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	var sessao *http.Cookie
	for _, ck := range res.Cookies() {
		if ck.Name == CookieSessao {
			sessao = ck
		}
	}
	if sessao == nil || !sessao.HttpOnly || sessao.SameSite != http.SameSiteLaxMode {
		t.Fatalf("cookie de sessão ausente ou sem HttpOnly/SameSite: %+v", sessao)
	}

	comCookie := func(metodo, caminho, corpo, sitio string) int {
		req, _ := http.NewRequest(metodo, srv.URL+caminho, bytes.NewReader([]byte(corpo)))
		req.AddCookie(sessao)
		req.Header.Set("Content-Type", "application/json")
		if sitio != "" {
			req.Header.Set("Sec-Fetch-Site", sitio)
		}
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		r.Body.Close()
		return r.StatusCode
	}
	if st := comCookie("GET", "/eu", "", "same-origin"); st != 200 {
		t.Fatalf("GET /eu com cookie: %d", st)
	}
	dados := `{"nome":"Bia","email":"bia@exemplo.test","data_nascimento":"1990-01-01","municipio":"Campinas","uf":"SP"}`
	if st := comCookie("PUT", "/conta", dados, "same-origin"); st != 204 {
		t.Fatalf("PUT com cookie, mesmo site: %d", st)
	}
	if st := comCookie("PUT", "/conta", dados, "cross-site"); st != 403 {
		t.Fatalf("PUT com cookie vindo de outro site deveria ser 403: %d", st)
	}
	// O token no cabeçalho (scripts, ensaio de carga) não depende de origem.
	c.exigir(c.req("GET", "/eu", nil), 200)
}

// Do "Pagar com Pix" ao "Paga", pelo HTTP: o prestador manda o aviso 5 vezes ao mesmo tempo, e a
// inscrição é confirmada uma vez. Aviso com assinatura falsa é recusado sem efeito.
func TestPagamentoPontaAPontaComAvisosRepetidos(t *testing.T) {
	cert := novoCertame(t, seed.Padrao())
	c := novoCandidato(t)
	insc := id(c.exigir(c.req("POST", fmt.Sprintf("/certames/%d/inscricoes", cert.ID), map[string]int64{"cargo_id": cert.Cargos[1]}), 201))
	ins := fmt.Sprintf("/inscricoes/%d", insc)
	c.exigir(c.req("POST", ins+"/pagamento", map[string]string{"metodo": "pix"}), 409) // ainda em rascunho
	c.exigir(c.req("PUT", ins+"/cidade", map[string]int64{"cidade_id": cert.Cidades[0]}), 204)
	c.exigir(c.req("PUT", ins+"/declaracoes", map[string]bool{"aceito": true}), 204)
	c.exigir(c.req("POST", ins+"/finalizar", nil), 200)

	cob := c.exigir(c.req("POST", ins+"/pagamento", map[string]string{"metodo": "pix"}), 200)
	if again := c.exigir(c.req("POST", ins+"/pagamento", map[string]string{"metodo": "pix"}), 200); id(again) != id(cob) {
		t.Fatalf("segundo clique gerou outra cobrança: %v x %v", id(again), id(cob))
	}
	if cob.corpo["valor_centavos"].(float64) != 10000 {
		t.Fatalf("cobrança errada: %v", cob.corpo)
	}
	// O clique respondeu sem esperar o prestador; o Pix chega pela fila, como o front espera.
	ate(t, "o Pix chegar pela fila", func() bool {
		var lista []map[string]any
		r, err := http.DefaultClient.Do(c.pedido("GET", ins+"/pagamento"))
		if err != nil {
			return false
		}
		defer r.Body.Close()
		json.NewDecoder(r.Body).Decode(&lista)
		return len(lista) == 1 && lista[0]["pix_copia_cola"] != nil
	})
	c.exigir(c.req("POST", fmt.Sprintf("%s/pagamento/%d/simular", ins, id(cob)), nil), 202)
	sim.EsperarAvisos()

	if e := c.exigir(c.req("GET", ins, nil), 200).corpo["estado"]; e != "PAGA" {
		t.Fatalf("depois dos avisos, estado %v", e)
	}

	// Paga, a inscrição ganha o cartão de confirmação: arquivo estático, sem sessão, com cache público.
	var url string
	ate(t, "o cartão ser publicado", func() bool {
		r := c.exigir(c.req("GET", ins+"/cartao", nil), 200)
		url, _ = r.corpo["url"].(string)
		return r.corpo["disponivel"] == true
	})
	res, err := http.Get(srv.URL + url) // sem o cookie nem o token
	if err != nil {
		t.Fatal(err)
	}
	pagina, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || !bytes.Contains(pagina, []byte("Candidata Sintética")) ||
		!bytes.Contains(pagina, []byte("Cartão de confirmação")) || !strings.HasPrefix(res.Header.Get("Cache-Control"), "public") {
		t.Fatalf("cartão: %d %q\n%s", res.StatusCode, res.Header.Get("Cache-Control"), pagina)
	}
	// Outro candidato não descobre onde está o cartão alheio.
	outro := novoCandidato(t)
	outro.exigir(outro.req("GET", ins+"/cartao", nil), 404)
	var confirmacoes int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM inscricao_evento WHERE inscricao_id = $1 AND evento = 'CONFIRMAR_PAGAMENTO'`, insc).Scan(&confirmacoes); err != nil {
		t.Fatal(err)
	}
	if confirmacoes != 1 {
		t.Fatalf("%d confirmações na trilha para 5 avisos", confirmacoes)
	}

	// Aviso forjado: sem a assinatura do segredo combinado, nada acontece.
	corpo := []byte(fmt.Sprintf(`{"pagamento_id":"pag_forjado","referencia":%d,"valor_centavos":10000,"pago_em":"2026-11-04T10:00:00Z"}`, id(cob)))
	for _, assinatura := range []string{"", "00", psp.Assinar([]byte("outro-segredo"), corpo)} {
		req, _ := http.NewRequest("POST", srv.URL+"/psp/aviso", bytes.NewReader(corpo))
		req.Header.Set(psp.CabecalhoAssinatura, assinatura)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusUnauthorized {
			t.Fatalf("aviso com assinatura %q aceito: %d", assinatura, res.StatusCode)
		}
	}
	var forjados int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM pagamento_recebido WHERE psp_pagamento_id = 'pag_forjado'`).Scan(&forjados); err != nil {
		t.Fatal(err)
	}
	if forjados != 0 {
		t.Fatal("aviso forjado foi registrado")
	}
}

// ate espera a condição por até 5 s — o tempo da fila trabalhar.
// Pela API, o caminho de quem pagou Pix e boleto e não tem o CPF como chave Pix: a devolução aparece
// esperando chave; outro candidato não a alcança; chave mal escrita é 422; a chave dela faz o dinheiro
// voltar; e depois a chave não muda mais (409).
func TestChavePixPelaAPI(t *testing.T) {
	cert := novoCertame(t, seed.Padrao())
	c := novoCandidato(t)
	insc := id(c.exigir(c.req("POST", fmt.Sprintf("/certames/%d/inscricoes", cert.ID), map[string]int64{"cargo_id": cert.Cargos[0]}), 201))
	ins := fmt.Sprintf("/inscricoes/%d", insc)
	c.exigir(c.req("PUT", ins+"/cidade", map[string]int64{"cidade_id": cert.Cidades[0]}), 204)
	c.exigir(c.req("PUT", ins+"/declaracoes", map[string]bool{"aceito": true}), 204)
	c.exigir(c.req("POST", ins+"/finalizar", nil), 200)
	var cpf string
	if err := pool.QueryRow(context.Background(),
		`SELECT d.cpf FROM candidato d JOIN inscricao i ON i.candidato_id = d.id WHERE i.id = $1`, insc).Scan(&cpf); err != nil {
		t.Fatal(err)
	}
	sim.SemChaveCPF(cpf)

	pix := id(c.exigir(c.req("POST", ins+"/pagamento", map[string]string{"metodo": "pix"}), 200))
	boleto := id(c.exigir(c.req("POST", ins+"/pagamento", map[string]string{"metodo": "boleto"}), 200))
	listar := func(caminho string) []map[string]any {
		var lista []map[string]any
		r, err := http.DefaultClient.Do(c.pedido("GET", caminho))
		if err != nil {
			return nil
		}
		defer r.Body.Close()
		json.NewDecoder(r.Body).Decode(&lista)
		return lista
	}
	ate(t, "o Pix e o boleto chegarem pela fila", func() bool {
		l := listar(ins + "/pagamento")
		return len(l) == 2 && (l[0]["pix_copia_cola"] != nil || l[0]["linha_digitavel"] != nil) &&
			(l[1]["pix_copia_cola"] != nil || l[1]["linha_digitavel"] != nil)
	})
	c.exigir(c.req("POST", fmt.Sprintf("%s/pagamento/%d/simular", ins, pix), nil), 202)
	sim.EsperarAvisos()
	c.exigir(c.req("POST", fmt.Sprintf("%s/pagamento/%d/simular", ins, boleto), nil), 202)
	sim.EsperarAvisos()

	var dev float64
	ate(t, "a devolução do boleto esperar chave", func() bool {
		l := listar(ins + "/devolucoes")
		if len(l) == 1 && l[0]["estado"] == "AGUARDANDO_CHAVE" {
			dev = l[0]["id"].(float64)
			return true
		}
		return false
	})
	// A recusa da chave CPF, tentada sozinha, não é "a chave que você informou".
	if l := listar(ins + "/devolucoes"); l[0]["chave_informada"] != false {
		t.Fatalf("sem chave informada, a devolução diz que há: %v", l[0])
	}
	chave := fmt.Sprintf("%s/devolucoes/%d/chave", ins, int64(dev))
	outro := novoCandidato(t)
	outro.exigir(outro.req("PUT", chave, map[string]string{"chave": "x@exemplo.test"}), 404)
	c.exigir(c.req("PUT", chave, map[string]string{"chave": "11.222.333/0001-81"}), 422)
	// Uma chave que não existe: vai ao prestador, volta a esperar, com a recusa e a chave informada.
	c.exigir(c.req("PUT", chave, map[string]string{"chave": "ninguem@exemplo.test"}), 202)
	ate(t, "a chave inexistente ser recusada", func() bool {
		l := listar(ins + "/devolucoes")
		return len(l) == 1 && l[0]["estado"] == "AGUARDANDO_CHAVE" && l[0]["recusa"] == "CHAVE_INEXISTENTE" && l[0]["chave_informada"] == true
	})

	sim.RegistrarChave("candidata@exemplo.test", cpf)
	c.exigir(c.req("PUT", chave, map[string]string{"chave": "Candidata@Exemplo.test"}), 202)
	ate(t, "a devolução concluir", func() bool {
		l := listar(ins + "/devolucoes")
		return len(l) == 1 && l[0]["estado"] == "CONCLUIDA"
	})
	c.exigir(c.req("PUT", chave, map[string]string{"chave": "outra@exemplo.test"}), 409)
}

// Pela API, a candidata vê os avisos da inscrição dela (o envio e o pagamento confirmado), com o
// resumo que prova o texto; outra candidata não vê nada da inscrição alheia.
func TestAvisosPelaAPI(t *testing.T) {
	cert := novoCertame(t, seed.Padrao())
	c := novoCandidato(t)
	insc := id(c.exigir(c.req("POST", fmt.Sprintf("/certames/%d/inscricoes", cert.ID), map[string]int64{"cargo_id": cert.Cargos[0]}), 201))
	ins := fmt.Sprintf("/inscricoes/%d", insc)
	c.exigir(c.req("PUT", ins+"/cidade", map[string]int64{"cidade_id": cert.Cidades[0]}), 204)
	c.exigir(c.req("PUT", ins+"/declaracoes", map[string]bool{"aceito": true}), 204)
	c.exigir(c.req("POST", ins+"/finalizar", nil), 200)
	cob := id(c.exigir(c.req("POST", ins+"/pagamento", map[string]string{"metodo": "pix"}), 200))
	listar := func(cl *cliente) []map[string]any {
		var l []map[string]any
		r, err := http.DefaultClient.Do(cl.pedido("GET", ins+"/avisos"))
		if err != nil {
			return nil
		}
		defer r.Body.Close()
		json.NewDecoder(r.Body).Decode(&l)
		return l
	}
	ate(t, "o Pix chegar", func() bool {
		var l []map[string]any
		r, err := http.DefaultClient.Do(c.pedido("GET", ins+"/pagamento"))
		if err != nil {
			return false
		}
		defer r.Body.Close()
		json.NewDecoder(r.Body).Decode(&l)
		return len(l) == 1 && l[0]["pix_copia_cola"] != nil
	})
	c.exigir(c.req("POST", fmt.Sprintf("%s/pagamento/%d/simular", ins, cob), nil), 202)
	sim.EsperarAvisos()
	ate(t, "os dois avisos saírem", func() bool {
		l := listar(c)
		return len(l) == 2 && l[0]["evento"] == "FINALIZAR_COM_PAGAMENTO" && l[1]["evento"] == "CONFIRMAR_PAGAMENTO" &&
			l[1]["entregue"] == true && len(l[1]["resumo"].(string)) == 64
	})
	if l := listar(novoCandidato(t)); len(l) != 0 {
		t.Fatalf("outra candidata viu %d avisos da inscrição alheia", len(l))
	}
}

func ate(t *testing.T, oque string, cond func() bool) {
	t.Helper()
	for limite := time.Now().Add(5 * time.Second); time.Now().Before(limite); time.Sleep(50 * time.Millisecond) {
		if cond() {
			return
		}
	}
	t.Fatalf("esperei 5 s por %s", oque)
}

// pedido monta uma requisição autenticada sem corpo.
func (c *cliente) pedido(metodo, caminho string) *http.Request {
	req, _ := http.NewRequest(metodo, srv.URL+caminho, nil)
	req.Header.Set("Authorization", "Bearer "+c.token)
	return req
}

// nisValido gera um NIS sintético com dígito verificador correto.
func nisValido(base int64) string {
	for dv := '0'; dv <= '9'; dv++ {
		n := fmt.Sprintf("%010d%c", base%10_000_000_000, dv)
		if inscricao.NISValido(n) {
			return n
		}
	}
	panic("sem dígito verificador")
}
