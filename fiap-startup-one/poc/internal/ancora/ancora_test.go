package ancora

import (
	"context"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"lisura/internal/arquivo"
	"lisura/internal/conta"
	"lisura/internal/dbtest"
	"lisura/internal/inscricao"
	"lisura/internal/seed"
)

var pool *pgxpool.Pool

func TestMain(m *testing.M) {
	p, err := dbtest.Pool(context.Background(), "t_ancora")
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

// certameCom cria um certame com n inscrições enviadas (cada uma com a sua trilha).
func certameCom(t *testing.T, n int) (certame int64, inscricoes []int64) {
	t.Helper()
	if pool == nil {
		t.Skip("sem banco de teste")
	}
	ctx := context.Background()
	cert, err := seed.CriarCertame(ctx, pool, seed.Padrao())
	must(t, err)
	for range n {
		k := seq.Add(1)
		email := "ancora." + strconv.FormatInt(k, 10) + "@lisura.test"
		cand, err := conta.Criar(ctx, pool, conta.GerarCPF(k+time.Now().UnixNano()%1_000_000*100), "Âncora", email)
		must(t, err)
		must(t, conta.AtualizarDados(ctx, pool, cand, conta.Dados{Nome: "Âncora", Email: email,
			DataNascimento: "1990-01-01", Municipio: "Campinas", UF: "SP"}))
		id, _, err := inscricao.Abrir(ctx, pool, cert.ID, cand, cert.Cargos[0], inscricao.Ator{Tipo: inscricao.Candidato, ID: "t"})
		must(t, err)
		must(t, inscricao.DefinirCidade(ctx, pool, id, cand, cert.Cidades[0]))
		must(t, inscricao.AceitarDeclaracoes(ctx, pool, id, cand, true))
		_, err = inscricao.Finalizar(ctx, pool, id, cand)
		must(t, err)
		inscricoes = append(inscricoes, id)
	}
	return cert.ID, inscricoes
}

// avancar acrescenta um evento à trilha da inscrição (o pagamento confirmado).
func avancar(t *testing.T, id int64) {
	t.Helper()
	_, err := inscricao.Transicionar(context.Background(), pool, inscricao.Comando{InscricaoID: id,
		Evento: inscricao.ConfirmarPagamento, Ator: inscricao.Ator{Tipo: inscricao.Sistema, ID: "t"}})
	must(t, err)
}

// superusuario faz o que a aplicação não consegue: desliga o gatilho de imutabilidade da trilha,
// executa e religa, numa transação — como faria quem quisesse apagar o rastro.
func superusuario(t *testing.T, sql string) {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	must(t, err)
	defer tx.Rollback(ctx)
	for _, s := range []string{`ALTER TABLE inscricao_evento DISABLE TRIGGER inscricao_evento_imutavel`, sql,
		`ALTER TABLE inscricao_evento ENABLE TRIGGER inscricao_evento_imutavel`} {
		_, err := tx.Exec(ctx, s)
		must(t, err)
	}
	must(t, tx.Commit(ctx))
}

// reescrever troca o ator do primeiro evento e recalcula a corrente inteira com a função do banco:
// a trilha fica consistente por dentro, e auditoria -trilhas não acha nada.
func reescrever(t *testing.T, id int64) {
	t.Helper()
	i := strconv.FormatInt(id, 10)
	superusuario(t, `UPDATE inscricao_evento SET ator_id = 'outro' WHERE id = (SELECT min(id) FROM inscricao_evento WHERE inscricao_id = `+i+`)`)
	superusuario(t, `
		DO $$
		DECLARE r inscricao_evento; anterior bytea;
		BEGIN
			FOR r IN SELECT * FROM inscricao_evento WHERE inscricao_id = `+i+` ORDER BY id LOOP
				anterior := trilha_resumo(anterior, r);
				UPDATE inscricao_evento SET hash = anterior WHERE id = r.id;
			END LOOP;
		END $$`)
	q, _, err := inscricao.VerificarTrilhas(context.Background(), pool, []int64{id})
	must(t, err)
	if len(q) != 0 {
		t.Fatalf("a reescrita devia passar na verificação interna: %v", q)
	}
}

// recibos faz o papel do site público.
type recibos struct {
	mu sync.Mutex
	m  map[string][]byte
}

func (r *recibos) Gravar(_ context.Context, caminho string, conteudo []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.m == nil {
		r.m = map[string][]byte{}
	}
	r.m[caminho] = conteudo
	return nil
}

func (r *recibos) Existe(_ context.Context, caminho string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.m[caminho]
	return ok
}

// Âncoras gravadas sem publicar recibo (pela linha de comando, ou com o site fora do ar) ganham o
// recibo na próxima rodada da rotina, mesmo sem nada novo para ancorar.
func TestReciboQueFaltouSaiNaProximaRodada(t *testing.T) {
	cert, ids := certameCom(t, 2)
	cofre := arquivo.CofreDisco{Dir: t.TempDir()}
	ancorar(t, cofre, nil, cert)
	avancar(t, ids[0])
	ancorar(t, cofre, nil, cert)
	pub := &recibos{}
	if _, gerou := ancorar(t, cofre, pub, cert); gerou {
		t.Fatal("âncora nova sem nada novo")
	}
	for seq := 1; seq <= 2; seq++ {
		texto, err := cofre.Ler(context.Background(), Chave(cert, seq))
		must(t, err)
		a, err := Ler(texto)
		must(t, err)
		if got := string(pub.m[CaminhoRecibo(cert, seq)]); got != string(Recibo(a, Resumo(texto))) {
			t.Fatalf("recibo %d: %q", seq, got)
		}
	}
}

func ancorar(t *testing.T, cofre arquivo.Cofre, pub Gravador, certame int64) (Ancora, bool) {
	t.Helper()
	a, gerou, err := Ancorar(context.Background(), pool, cofre, pub, certame)
	must(t, err)
	return a, gerou
}

func conferir(t *testing.T, cofre arquivo.Cofre, certame int64) Conferencia {
	t.Helper()
	c, err := Conferir(context.Background(), pool, cofre, certame)
	must(t, err)
	return c
}

func tipos(c Conferencia) map[string]int {
	m := map[string]int{}
	for _, p := range c.Problemas {
		m[p.Tipo]++
	}
	return m
}

func TestTextoIdaEVolta(t *testing.T) {
	a := Ancora{Certame: 7, Sequencia: 3, Tipo: Incremental, Anterior: "ab", GeradaEm: time.Date(2026, 11, 5, 10, 0, 0, 123456789, time.UTC),
		Eventos: 9, Total: 40, Linhas: []Linha{{1, 10, make([]byte, 32)}, {2, 12, append(make([]byte, 31), 1)}}}
	b, err := Ler(a.Texto())
	must(t, err)
	if string(b.Texto()) != string(a.Texto()) {
		t.Fatalf("ida e volta mudou a âncora:\n%s\n%s", a.Texto(), b.Texto())
	}
	if _, err := Ler(append(a.Texto(), "3 13 xyz\n"...)); err == nil {
		t.Fatal("linha inválida aceita")
	}
	a.Tipo = Completa // completa com total diferente das linhas: o arquivo mente sobre si
	if _, err := Ler(a.Texto()); err == nil {
		t.Fatal("completa com total errado aceita")
	}
	// O formato 1 (sem tipo nem total) continua legível: âncora já gravada no cofre não se reescreve.
	v1 := "lisura-ancora 1\ncertame 7\nsequencia 1\nanterior -\ngerada_em 2026-11-05T10:00:00Z\neventos 3\ninscricoes 1\n" +
		"1 10 " + strings.Repeat("ab", 32) + "\n"
	c, err := Ler([]byte(v1))
	must(t, err)
	if c.Tipo != Completa || c.Total != 1 || len(c.Linhas) != 1 {
		t.Fatalf("âncora do formato 1 lida como %+v", c)
	}
}

// Entre duas completas, as incrementais levam só o que mudou — e a conferência soma tudo.
func TestIncrementalSoLevaOQueMudou(t *testing.T) {
	cert, ids := certameCom(t, 5)
	cofre := arquivo.CofreDisco{Dir: t.TempDir()}
	a1, _ := ancorar(t, cofre, nil, cert)
	avancar(t, ids[3])
	a2, _ := ancorar(t, cofre, nil, cert)
	if a1.Tipo != Completa || len(a1.Linhas) != 5 || a2.Tipo != Incremental || len(a2.Linhas) != 1 ||
		a2.Linhas[0].Inscricao != ids[3] || a2.Total != 5 {
		t.Fatalf("âncora 1: %s com %d linhas; âncora 2: %s com %d linhas de %d", a1.Tipo, len(a1.Linhas), a2.Tipo, len(a2.Linhas), a2.Total)
	}
	nova := maisUma(t, cert) // inscrição nova entra na incremental
	a3, _ := ancorar(t, cofre, nil, cert)
	if a3.Tipo != Incremental || len(a3.Linhas) != 1 || a3.Linhas[0].Inscricao != nova || a3.Total != 6 {
		t.Fatalf("âncora 3: %s, %d linhas de %d", a3.Tipo, len(a3.Linhas), a3.Total)
	}
	CompletaCada = 0
	t.Cleanup(func() { CompletaCada = 24 * time.Hour })
	avancar(t, ids[0])
	a4, _ := ancorar(t, cofre, nil, cert)
	if a4.Tipo != Completa || len(a4.Linhas) != 6 {
		t.Fatalf("âncora 4 devia ser completa: %s com %d linhas", a4.Tipo, len(a4.Linhas))
	}
	if c := conferir(t, cofre, cert); len(c.Problemas) != 0 || c.Ancoras != 4 {
		t.Fatalf("%d âncoras, problemas %+v", c.Ancoras, c.Problemas)
	}

	// Reescrever uma inscrição que não mudou desde a completa: a incremental seguinte a leva (o
	// resumo mudou), e a conferência acusa.
	CompletaCada = 24 * time.Hour
	reescrever(t, ids[4])
	avancar(t, ids[1])
	a5, _ := ancorar(t, cofre, nil, cert)
	if a5.Tipo != Incremental || len(a5.Linhas) != 2 {
		t.Fatalf("âncora 5: %s com %d linhas, esperado incremental com 2 (a avançada e a reescrita)", a5.Tipo, len(a5.Linhas))
	}
	if got := tipos(conferir(t, cofre, cert)); got["RESUMO_MUDOU_ENTRE_ANCORAS"] != 1 || len(got) != 1 {
		t.Fatalf("problemas %v, esperado só RESUMO_MUDOU_ENTRE_ANCORAS", got)
	}
}

func TestAncoraSoSaiQuandoATrilhaMuda(t *testing.T) {
	cert, ids := certameCom(t, 3)
	cofre := arquivo.CofreDisco{Dir: t.TempDir()}
	pub := &recibos{}

	a1, gerou := ancorar(t, cofre, pub, cert)
	if !gerou || a1.Sequencia != 1 || a1.Anterior != "-" || len(a1.Linhas) != 3 {
		t.Fatalf("primeira âncora: gerou=%v %+v", gerou, a1)
	}
	if _, gerou := ancorar(t, cofre, pub, cert); gerou {
		t.Fatal("âncora nova sem nada novo na trilha")
	}
	avancar(t, ids[1])
	a2, gerou := ancorar(t, cofre, pub, cert)
	texto1, err := cofre.Ler(context.Background(), Chave(cert, 1))
	must(t, err)
	if !gerou || a2.Sequencia != 2 || a2.Anterior != Resumo(texto1) || a2.Eventos != a1.Eventos+1 {
		t.Fatalf("segunda âncora: gerou=%v seq=%d anterior=%s eventos=%d", gerou, a2.Sequencia, a2.Anterior, a2.Eventos)
	}
	texto2, err := cofre.Ler(context.Background(), Chave(cert, 2))
	must(t, err)
	if string(pub.m[CaminhoRecibo(cert, 2)]) != string(Recibo(a2, Resumo(texto2))) {
		t.Fatalf("recibo público: %q", pub.m[CaminhoRecibo(cert, 2)])
	}

	c := conferir(t, cofre, cert)
	if len(c.Problemas) != 0 || c.Ancoras != 2 || c.Pontos != 4 { // 3 atuais + o de ids[1] substituído
		t.Fatalf("trilha intacta acusada: %+v", c)
	}
}

// O caso que a corrente sozinha não pega: reescrita completa e consistente. A âncora pega.
func TestReescritaDepoisDaAncoraEDenunciada(t *testing.T) {
	cert, ids := certameCom(t, 3)
	cofre := arquivo.CofreDisco{Dir: t.TempDir()}
	ancorar(t, cofre, nil, cert)
	reescrever(t, ids[0])

	c := conferir(t, cofre, cert)
	if len(c.Problemas) != 1 || c.Problemas[0].Tipo != "TRILHA_DIFERENTE_DA_ANCORA" || c.Problemas[0].Inscricao != ids[0] {
		t.Fatalf("problemas %+v, esperado TRILHA_DIFERENTE_DA_ANCORA na inscrição %d", c.Problemas, ids[0])
	}
}

// Reescrever e deixar o sistema ancorar de novo não lava a trilha: a âncora velha continua no cofre.
func TestReancorarNaoLavaAReescrita(t *testing.T) {
	cert, ids := certameCom(t, 2)
	cofre := arquivo.CofreDisco{Dir: t.TempDir()}
	ancorar(t, cofre, nil, cert)
	reescrever(t, ids[1])
	avancar(t, ids[0]) // muda a contagem: a rotina gera a âncora 2, já com a trilha reescrita
	if _, gerou := ancorar(t, cofre, nil, cert); !gerou {
		t.Fatal("a segunda âncora não saiu")
	}
	c := conferir(t, cofre, cert)
	if got := tipos(c); len(c.Problemas) != 1 || got["RESUMO_MUDOU_ENTRE_ANCORAS"] != 1 || c.Problemas[0].Inscricao != ids[1] {
		t.Fatalf("problemas %+v, esperado RESUMO_MUDOU_ENTRE_ANCORAS na inscrição %d", c.Problemas, ids[1])
	}
}

func TestApagarInscricaoAncoradaEDenunciado(t *testing.T) {
	cert, ids := certameCom(t, 3)
	cofre := arquivo.CofreDisco{Dir: t.TempDir()}
	ancorar(t, cofre, nil, cert)
	superusuario(t, `DELETE FROM inscricao_evento WHERE inscricao_id = `+strconv.FormatInt(ids[2], 10))

	c := conferir(t, cofre, cert)
	if len(c.Problemas) != 1 || c.Problemas[0].Tipo != "EVENTO_ANCORADO_SUMIU" || c.Problemas[0].Inscricao != ids[2] {
		t.Fatalf("problemas %+v, esperado EVENTO_ANCORADO_SUMIU na inscrição %d", c.Problemas, ids[2])
	}
}

// Uma inscrição criada depois da âncora com data de antes ("estava aqui desde o prazo") fica de fora
// da âncora que devia tê-la.
func TestInscricaoEnfiadaNoPassadoEDenunciada(t *testing.T) {
	cert, _ := certameCom(t, 2)
	cofre := arquivo.CofreDisco{Dir: t.TempDir()}
	ancorar(t, cofre, nil, cert)
	ids := maisUma(t, cert)
	superusuario(t, `UPDATE inscricao_evento SET ocorrido_em = ocorrido_em - interval '1 day' WHERE inscricao_id = `+strconv.FormatInt(ids, 10))

	c := conferir(t, cofre, cert)
	if len(c.Problemas) != 1 || c.Problemas[0].Tipo != "INSCRICAO_FORA_DA_ANCORA" || c.Problemas[0].Inscricao != ids {
		t.Fatalf("problemas %+v, esperado INSCRICAO_FORA_DA_ANCORA na inscrição %d", c.Problemas, ids)
	}
	// A mesma inscrição com a data verdadeira (depois da âncora) é só uma inscrição nova.
	cert2, _ := certameCom(t, 1)
	ancorar(t, cofre, nil, cert2)
	maisUma(t, cert2)
	if c := conferir(t, cofre, cert2); len(c.Problemas) != 0 {
		t.Fatalf("inscrição nova, posterior à âncora, acusada: %+v", c.Problemas)
	}
}

// maisUma abre outra inscrição no mesmo certame (outro candidato).
func maisUma(t *testing.T, certame int64) int64 {
	t.Helper()
	ctx := context.Background()
	var cargo, cidade int64
	must(t, pool.QueryRow(ctx, `SELECT id FROM cargo WHERE certame_id = $1 ORDER BY id LIMIT 1`, certame).Scan(&cargo))
	must(t, pool.QueryRow(ctx, `SELECT id FROM cidade_prova WHERE certame_id = $1 ORDER BY id LIMIT 1`, certame).Scan(&cidade))
	k := seq.Add(1)
	email := "ancora." + strconv.FormatInt(k, 10) + "@lisura.test"
	cand, err := conta.Criar(ctx, pool, conta.GerarCPF(k+time.Now().UnixNano()%1_000_000*100), "Âncora", email)
	must(t, err)
	must(t, conta.AtualizarDados(ctx, pool, cand, conta.Dados{Nome: "Âncora", Email: email,
		DataNascimento: "1990-01-01", Municipio: "Campinas", UF: "SP"}))
	id, _, err := inscricao.Abrir(ctx, pool, certame, cand, cargo, inscricao.Ator{Tipo: inscricao.Candidato, ID: "t"})
	must(t, err)
	must(t, inscricao.DefinirCidade(ctx, pool, id, cand, cidade))
	return id
}

// No disco local o cofre só imita a regra; mexer nele é o que o bloqueio do S3 impede. Ainda assim,
// a cadeia das âncoras denuncia quem troca ou apaga uma âncora antiga.
func TestCofreMexidoEDenunciado(t *testing.T) {
	cert, ids := certameCom(t, 2)
	dir := t.TempDir()
	cofre := arquivo.CofreDisco{Dir: dir}
	ancorar(t, cofre, nil, cert)
	avancar(t, ids[0])
	ancorar(t, cofre, nil, cert)

	arq := filepath.Join(dir, filepath.FromSlash(Chave(cert, 1)))
	texto, err := os.ReadFile(arq)
	must(t, err)
	must(t, os.Chmod(arq, 0o644))
	// Troca um dígito do resumo da última linha: a âncora continua legível, só diz outra coisa.
	if texto[len(texto)-2] == '0' {
		texto[len(texto)-2] = '1'
	} else {
		texto[len(texto)-2] = '0'
	}
	must(t, os.WriteFile(arq, texto, 0o644))
	if got := tipos(conferir(t, cofre, cert)); got["CADEIA_DE_ANCORAS_QUEBRADA"] != 1 {
		t.Fatalf("âncora trocada não denunciada: %v", got)
	}

	must(t, os.Remove(arq))
	if got := tipos(conferir(t, cofre, cert)); got["ANCORA_FALTANDO"] != 1 || got["CADEIA_DE_ANCORAS_QUEBRADA"] != 1 {
		t.Fatalf("âncora apagada não denunciada: %v", got)
	}
}

func TestAncorasConcorrentesUmaSo(t *testing.T) {
	cert, _ := certameCom(t, 2)
	cofre := arquivo.CofreDisco{Dir: t.TempDir()}
	var gerou atomic.Int32
	var wg sync.WaitGroup
	for range 6 {
		wg.Go(func() {
			_, g, err := Ancorar(context.Background(), pool, cofre, nil, cert)
			if err != nil {
				t.Error(err)
			}
			if g {
				gerou.Add(1)
			}
		})
	}
	wg.Wait()
	l, err := cofre.Listar(context.Background(), prefixo(cert))
	must(t, err)
	if gerou.Load() != 1 || len(l) != 1 {
		t.Fatalf("%d instâncias ancoraram, %d âncoras no cofre; esperado 1 e 1", gerou.Load(), len(l))
	}
}

// Sem a trava e com a janela entre fotografia e gravação alargada, várias instâncias ancoram ao
// mesmo tempo que a trilha anda. Se uma âncora pudesse sair de uma fotografia mais velha que a da
// anterior, a conferência acusaria corrente andando para trás.
func TestAncorasDisputadasNuncaAndamParaTras(t *testing.T) {
	const rodadas, porRodada = 8, 3
	cert, ids := certameCom(t, rodadas*porRodada)
	usarTrava = false
	depoisDaFotografia = func() { time.Sleep(time.Duration(rand.IntN(40)) * time.Millisecond) }
	t.Cleanup(func() { usarTrava, depoisDaFotografia = true, func() {} })
	cofre := arquivo.CofreDisco{Dir: t.TempDir()}

	ancorar(t, cofre, nil, cert)
	for r := range rodadas {
		var wg sync.WaitGroup
		for i := range 6 {
			wg.Go(func() {
				time.Sleep(time.Duration(i*5) * time.Millisecond)
				if _, _, err := Ancorar(context.Background(), pool, cofre, nil, cert); err != nil {
					t.Error(err)
				}
			})
		}
		for _, id := range ids[r*porRodada : (r+1)*porRodada] {
			wg.Go(func() {
				time.Sleep(time.Duration(rand.IntN(30)) * time.Millisecond)
				if _, err := inscricao.Transicionar(context.Background(), pool, inscricao.Comando{InscricaoID: id,
					Evento: inscricao.ConfirmarPagamento, Ator: inscricao.Ator{Tipo: inscricao.Sistema, ID: "t"}}); err != nil {
					t.Error(err)
				}
			})
		}
		wg.Wait()
	}
	ancorar(t, cofre, nil, cert) // pega o que a última rodada deixou
	c := conferir(t, cofre, cert)
	if len(c.Problemas) != 0 {
		t.Fatalf("%d âncoras, problemas: %+v", c.Ancoras, c.Problemas)
	}
	if c.Ancoras < 3 {
		t.Fatalf("só %d âncoras: a disputa não aconteceu", c.Ancoras)
	}
	t.Logf("%d âncoras em %d rodadas disputadas, nenhuma para trás", c.Ancoras, rodadas)
}
