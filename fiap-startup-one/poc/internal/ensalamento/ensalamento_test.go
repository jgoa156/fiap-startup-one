package ensalamento

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
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
	p, err := dbtest.Pool(context.Background(), "t_ensalamento")
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

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

var seq atomic.Int64

// cenario é um certame com dois locais numa cidade e uma inscrição homologada para cada tipo de
// atendimento:
//
//	Escola A (com amamentação): 101 comum acessível, 3 lugares; 102 comum, 2 lugares
//	Escola B:                   201 especial acessível, 2 lugares; 202 individual
type cenario struct {
	certame, cargo, cidade int64
	cidadeNome             string
	insc                   map[string]int64 // por nome
}

const locaisCSV = Cabecalho + `
{cidade};Escola A;Rua Um, 1;sim;101;3;comum;sim
{cidade};Escola A;Rua Um, 1;sim;102;2;comum;não
{cidade};Escola B;Rua Dois, 2;não;201;2;especial;sim
{cidade};Escola B;Rua Dois, 2;não;202;1;individual;não
`

func novoCenario(t *testing.T) *cenario {
	t.Helper()
	if pool == nil {
		t.Skip("sem banco de teste")
	}
	ctx := context.Background()
	cert, err := seed.CriarCertame(ctx, pool, seed.Padrao())
	must(t, err)
	c := &cenario{certame: cert.ID, cargo: cert.Cargos[0], cidade: cert.Cidades[0], insc: map[string]int64{}}
	must(t, pool.QueryRow(ctx, `SELECT nome FROM cidade_prova WHERE id = $1`, c.cidade).Scan(&c.cidadeNome))
	_, _, err = ImportarLocais(ctx, pool, c.certame, strings.NewReader(strings.ReplaceAll(locaisCSV, "{cidade}", c.cidadeNome)))
	must(t, err)
	for nome, nec := range map[string][]string{
		"Paula Um": nil, "Paula Dois": nil, "Paula Três": nil,
		// "Zé" vem por último em ordem de nome: sem a regra de os restritos escolherem primeiro, as três
		// Paulas lotariam a 101, a única comum acessível, e não sobraria lugar para ele.
		"Zé Acessível": {"sala_acessivel"}, "Tempo": {"tempo_adicional"}, "Ledor": {"ledor"}, "Lactante": {"lactante"},
	} {
		c.homologar(t, nome, nec)
	}
	return c
}

// homologar cria a inscrição e a leva a HOMOLOGADA pela máquina de estados.
func (c *cenario) homologar(t *testing.T, nome string, necessidades []string) int64 {
	t.Helper()
	ctx := context.Background()
	cand, err := conta.Criar(ctx, pool, conta.GerarCPF(seq.Add(1)+time.Now().UnixNano()%1_000_000*100), nome, "e@exemplo.test")
	must(t, err)
	must(t, conta.AtualizarDados(ctx, pool, cand, conta.Dados{Nome: nome, Email: "e@exemplo.test",
		DataNascimento: "1990-01-01", Municipio: "Campinas", UF: "SP"}))
	id, _, err := inscricao.Abrir(ctx, pool, c.certame, cand, c.cargo, inscricao.Ator{Tipo: inscricao.Candidato, ID: "t"})
	must(t, err)
	must(t, inscricao.DefinirCidade(ctx, pool, id, cand, c.cidade))
	must(t, inscricao.AceitarDeclaracoes(ctx, pool, id, cand, true))
	_, err = inscricao.Finalizar(ctx, pool, id, cand)
	must(t, err)
	if necessidades != nil { // o laudo é conferido na análise da banca, fora deste teste
		_, err = pool.Exec(ctx, `UPDATE inscricao SET necessidades = $2 WHERE id = $1`, id, necessidades)
		must(t, err)
	}
	sistema, ana := inscricao.Ator{Tipo: inscricao.Sistema, ID: "t"}, inscricao.Ator{Tipo: inscricao.Analista, ID: "ana"}
	for _, p := range []struct {
		ev   inscricao.Evento
		ator inscricao.Ator
	}{{inscricao.ConfirmarPagamento, sistema}, {inscricao.IniciarAnalise, sistema}, {inscricao.Deferir, ana}, {inscricao.Homologar, ana}} {
		_, err := inscricao.Transicionar(ctx, pool, inscricao.Comando{InscricaoID: id, Evento: p.ev, Ator: p.ator})
		must(t, err)
	}
	c.insc[nome] = id
	return id
}

// salas devolve a sala ("Escola A/101") de cada inscrição, por nome.
func (c *cenario) salas(t *testing.T) map[string]string {
	t.Helper()
	por := map[int64]string{}
	rows, err := pool.Query(context.Background(), `
		SELECT a.inscricao_id, l.nome || '/' || s.nome FROM alocacao a JOIN sala s ON s.id = a.sala_id
		JOIN local_prova l ON l.id = s.local_id WHERE a.certame_id = $1`, c.certame)
	must(t, err)
	for rows.Next() {
		var id int64
		var sala string
		must(t, rows.Scan(&id, &sala))
		por[id] = sala
	}
	must(t, rows.Err())
	out := map[string]string{}
	for nome, id := range c.insc {
		if s, ok := por[id]; ok {
			out[nome] = s
		}
	}
	return out
}

func (c *cenario) conferir(t *testing.T) Conferencia {
	t.Helper()
	conf, err := Conferir(context.Background(), pool, c.certame)
	must(t, err)
	return conf
}

func temProblema(c Conferencia, prefixo string) bool {
	for _, p := range c.Problemas {
		if strings.HasPrefix(p, prefixo) {
			return true
		}
	}
	return false
}

// Cada atendimento na sala certa, os mais restritos primeiro, e os demais em ordem de nome na primeira
// sala com lugar: a acessível e a lactante pegam a 101 antes; a 101 completa com a primeira em ordem
// alfabética, e as outras duas vão para a 102.
func TestEnsalaComCapacidadeEAtendimento(t *testing.T) {
	c := novoCenario(t)
	v, err := Ensalar(context.Background(), pool, c.certame, "ana", "ENSALAMENTO_INICIAL")
	must(t, err)
	if !v.Nova || v.Versao != 1 || v.Alocadas != 7 || v.Novas != 7 {
		t.Fatalf("versão: %+v", v)
	}
	quer := map[string]string{
		"Zé Acessível": "Escola A/101", "Lactante": "Escola A/101", "Paula Dois": "Escola A/101",
		"Paula Três": "Escola A/102", "Paula Um": "Escola A/102",
		"Tempo": "Escola B/201", "Ledor": "Escola B/202",
	}
	got := c.salas(t)
	for nome, sala := range quer {
		if got[nome] != sala {
			t.Errorf("%s: %q, esperado %q", nome, got[nome], sala)
		}
	}
	if conf := c.conferir(t); len(conf.Problemas) != 0 || conf.Alocadas != 7 || conf.Versao != 1 {
		t.Fatalf("conferência: %+v", conf)
	}
	m, err := Mapa(context.Background(), pool, c.certame)
	must(t, err)
	if len(m) != 2 || m[0].Capacidade != 5 || m[0].Alocados != 5 || m[1].Capacidade != 3 || m[1].Alocados != 2 {
		t.Fatalf("mapa: %+v", m)
	}
	ls, err := Listas(context.Background(), pool, c.certame)
	must(t, err)
	if len(ls) != 4 || ls[0].Sala != "101" || len(ls[0].Presencas) != 3 || ls[0].Presencas[0].Nome != "Lactante" ||
		!strings.HasPrefix(ls[0].Presencas[0].CPF, "***.") {
		t.Fatalf("listas: %+v", ls)
	}
}

// Dois ledores e uma sala individual: ninguém é alocado, e o erro diz o que falta, onde.
func TestFaltaLugarNaoAlocaNinguem(t *testing.T) {
	c := novoCenario(t)
	c.homologar(t, "Outro Ledor", []string{"ledor", "sala_acessivel"})
	c.homologar(t, "Mais Um Ledor", []string{"transcritor"})
	_, err := Ensalar(context.Background(), pool, c.certame, "ana", "ENSALAMENTO_INICIAL")
	var falta *ErrFaltaLugar
	if !errors.As(err, &falta) {
		t.Fatalf("erro %v; esperado falta de lugar", err)
	}
	quer := []Falta{
		{Cidade: falta.Faltas[0].Cidade, Requisito: "sala individual", Faltam: 1},
		{Cidade: falta.Faltas[0].Cidade, Requisito: "sala individual acessível", Faltam: 1},
	}
	if len(falta.Faltas) != 2 || falta.Faltas[0] != quer[0] || falta.Faltas[1] != quer[1] || !strings.HasPrefix(falta.Faltas[0].Cidade, c.cidadeNome) {
		t.Fatalf("faltas: %+v", falta.Faltas)
	}
	if n := len(c.salas(t)); n != 0 {
		t.Fatalf("%d alocados com falta de lugar; esperado nenhum", n)
	}
}

// Ensalar de novo depois de uma entrada e uma saída: quem tinha sala continua nela, a nova ocupa o
// lugar que vagou, e a versão sobe. Ensalar de novo sem mudança não cria versão.
func TestReensalarMantemQuemTinhaSala(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	_, err := Ensalar(ctx, pool, c.certame, "ana", "ENSALAMENTO_INICIAL")
	must(t, err)
	antes := c.salas(t)
	// Paula Três sai da 102 (indeferida em recurso); Ana Nova entra (recurso provido) e vem antes de
	// todas em ordem de nome. Refeito do zero, Ana tomaria o último lugar da 101 e empurraria Paula Dois
	// para a 102; mantendo quem tinha sala, Ana fica com o lugar que vagou.
	_, err = pool.Exec(ctx, `UPDATE inscricao SET estado = 'INDEFERIDA' WHERE id = $1`, c.insc["Paula Três"])
	must(t, err)
	c.homologar(t, "Ana Nova", nil)

	v, err := Ensalar(ctx, pool, c.certame, "ana", "RECURSO_PROVIDO")
	must(t, err)
	if v.Versao != 2 || v.Mantidas != 6 || v.Novas != 1 || v.Removidas != 1 {
		t.Fatalf("versão: %+v", v)
	}
	depois := c.salas(t)
	for nome, sala := range antes {
		if nome != "Paula Três" && depois[nome] != sala {
			t.Errorf("%s trocou de sala: %s → %s", nome, sala, depois[nome])
		}
	}
	if depois["Ana Nova"] != antes["Paula Três"] || depois["Paula Três"] != "" {
		t.Fatalf("Ana Nova em %q (vagou %q); Paula Três em %q", depois["Ana Nova"], antes["Paula Três"], depois["Paula Três"])
	}
	if conf := c.conferir(t); len(conf.Problemas) != 0 || conf.Versao != 2 {
		t.Fatalf("conferência: %+v", conf)
	}
	if v, err := Ensalar(ctx, pool, c.certame, "ana", "SEM_MUDANCA"); err != nil || v.Nova || v.Versao != 2 {
		t.Fatalf("sem mudança: %+v, %v", v, err)
	}
}

// O banco recusa sala lotada, mesmo por fora do ensalamento.
func TestBancoRecusaSalaLotada(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	_, err := Ensalar(ctx, pool, c.certame, "ana", "ENSALAMENTO_INICIAL")
	must(t, err)
	extra := c.homologar(t, "Fura Fila", []string{"ledor"})
	_, err = pool.Exec(ctx, `INSERT INTO alocacao (inscricao_id, certame_id, sala_id, versao)
		SELECT $1, $2, s.id, 1 FROM sala s JOIN local_prova l ON l.id = s.local_id
		WHERE l.certame_id = $2 AND s.nome = '202'`, extra, c.certame)
	if err == nil || !strings.Contains(err.Error(), "lotada") {
		t.Fatalf("o banco aceitou a sala individual com dois: %v", err)
	}
}

// Mexer na alocação por fora: o ledor numa sala especial, a de tempo adicional sem sala. A
// conferência acusa as três coisas: atendimento não atendido, homologada sem sala e alocação que não
// é a do ato publicado.
func TestConferirAcusaAlocacaoMexida(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	_, err := Ensalar(ctx, pool, c.certame, "ana", "ENSALAMENTO_INICIAL")
	must(t, err)
	_, err = pool.Exec(ctx, `DELETE FROM alocacao WHERE inscricao_id = $1`, c.insc["Tempo"])
	must(t, err)
	_, err = pool.Exec(ctx, `UPDATE alocacao SET sala_id = (SELECT s.id FROM sala s JOIN local_prova l ON l.id = s.local_id
		WHERE l.certame_id = $2 AND s.nome = '201') WHERE inscricao_id = $1`, c.insc["Ledor"], c.certame)
	must(t, err)
	conf := c.conferir(t)
	for _, quer := range []string{"ATENDIMENTO ESPECIAL NÃO ATENDIDO", "HOMOLOGADA SEM SALA", "ALOCAÇÃO MUDOU FORA DO ENSALAMENTO"} {
		if !temProblema(conf, quer) {
			t.Errorf("não acusou %q: %v", quer, conf.Problemas)
		}
	}
}

// Depois do ensalamento, o cartão mostra o local, o endereço e a sala, e deixa de avisar que eles
// ainda vão sair.
func TestCartaoMostraASala(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	pub := &publicacao.Publicador{Pool: pool, Dir: t.TempDir(), Chave: []byte("k")}
	id := c.insc["Ledor"]
	ler := func() string {
		must(t, pub.PublicarCartao(ctx, id))
		b, err := os.ReadFile(filepath.Join(pub.Dir, pub.CaminhoCartao(id)))
		must(t, err)
		return string(b)
	}
	if antes := ler(); !strings.Contains(antes, "divulgados na fase de ensalamento") {
		t.Fatal("antes do ensalamento, o cartão não avisa que a sala ainda vai sair")
	}
	_, err := Ensalar(ctx, pool, c.certame, "ana", "ENSALAMENTO_INICIAL")
	must(t, err)
	depois := ler()
	if !strings.Contains(depois, "Escola B") || !strings.Contains(depois, "Rua Dois, 2") || !strings.Contains(depois, "<dd>202</dd>") ||
		strings.Contains(depois, "divulgados na fase de ensalamento") {
		t.Fatalf("cartão depois do ensalamento:\n%s", depois)
	}
}

func TestImportarLocaisRecusaLinhaErrada(t *testing.T) {
	c := novoCenario(t)
	for nome, csv := range map[string]string{
		"cidade de outro certame": Cabecalho + "\nAtlântida;Escola;Rua;não;1;10;comum;não\n",
		"individual com dois":     Cabecalho + "\n" + c.cidadeNome + ";Escola;Rua;não;1;2;individual;não\n",
		"tipo inventado":          Cabecalho + "\n" + c.cidadeNome + ";Escola;Rua;não;1;10;vip;não\n",
		"cabeçalho errado":        "cidade;local\n",
	} {
		if _, _, err := ImportarLocais(context.Background(), pool, c.certame, strings.NewReader(csv)); err == nil {
			t.Errorf("%s: aceito", nome)
		} else if nome != "cabeçalho errado" && !strings.Contains(err.Error(), "linha 2") {
			t.Errorf("%s: o erro não diz a linha: %v", nome, err)
		}
	}
}
