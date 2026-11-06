package exportacao

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"lisura/internal/ancora"
	"lisura/internal/arquivo"
	"lisura/internal/arrecadacao"
	"lisura/internal/conta"
	"lisura/internal/dbtest"
	"lisura/internal/inscricao"
	"lisura/internal/seed"
)

var pool *pgxpool.Pool

func TestMain(m *testing.M) {
	p, err := dbtest.Pool(context.Background(), "t_exportacao")
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

// exportado monta um certame com trilha, pagamentos (um deles de ontem, com o dia fechado), uma
// devolução concluída e uma âncora no cofre, exporta e devolve a pasta.
func exportado(t *testing.T) string {
	t.Helper()
	if pool == nil {
		t.Skip("sem banco de teste")
	}
	ctx := context.Background()
	cert, err := seed.CriarCertame(ctx, pool, seed.Padrao())
	must(t, err)
	sistema := inscricao.Ator{Tipo: inscricao.Sistema, ID: "t"}
	var pagas []int64
	for i := range 3 {
		cand, err := conta.Criar(ctx, pool, conta.GerarCPF(int64(i)+time.Now().UnixNano()%1_000_000*100), "Exportada", "x@exemplo.test")
		must(t, err)
		must(t, conta.AtualizarDados(ctx, pool, cand, conta.Dados{Nome: "Exportada", Email: "x@exemplo.test",
			DataNascimento: "1990-01-01", Municipio: "Campinas", UF: "SP"}))
		id, _, err := inscricao.Abrir(ctx, pool, cert.ID, cand, cert.Cargos[0], inscricao.Ator{Tipo: inscricao.Candidato, ID: "t"})
		must(t, err)
		must(t, inscricao.DefinirCidade(ctx, pool, id, cand, cert.Cidades[0]))
		must(t, inscricao.AceitarDeclaracoes(ctx, pool, id, cand, true))
		_, err = inscricao.Finalizar(ctx, pool, id, cand)
		must(t, err)
		if i < 2 {
			_, err = inscricao.Transicionar(ctx, pool, inscricao.Comando{InscricaoID: id, Evento: inscricao.ConfirmarPagamento, Ator: sistema})
			must(t, err)
			pagas = append(pagas, id)
		}
	}
	pag := func(id string, insc int64, metodo, resultado string, ontem bool) {
		_, err := pool.Exec(ctx, `
			WITH c AS (INSERT INTO cobranca (inscricao_id, metodo, valor_centavos, estado, vence_em)
			           VALUES ($2, $3, 8000, CASE WHEN $4 = 'CONFIRMADO' THEN 'PAGA' ELSE 'PAGA_EM_DUPLICIDADE' END, now() + interval '1 day') RETURNING id)
			INSERT INTO pagamento_recebido (psp_pagamento_id, cobranca_id, valor_centavos, pago_em, origem, resultado, recebido_em)
			SELECT $1, c.id, 8000, now(), 'webhook', $4, CASE WHEN $5 THEN now() - interval '1 day' ELSE now() END FROM c`,
			id, insc, metodo, resultado, ontem)
		must(t, err)
	}
	// Um id prefixo do outro ("pag_b_N" e "pag_b_N_dup"): é o caso em que ordenar a linha inteira dá
	// outra ordem que ordenar por id, e o dia fechado só confere se o auditor ordenar como o banco.
	sufixo := strconv.FormatInt(cert.ID, 10)
	pag("pag_a_"+sufixo, pagas[0], "pix", "CONFIRMADO", true)
	pag("pag_b_"+sufixo, pagas[1], "pix", "CONFIRMADO", true)
	pag("pag_b_"+sufixo+"_dup", pagas[1], "boleto", "PAGO_EM_DUPLICIDADE", true)
	_, err = pool.Exec(ctx, `
		INSERT INTO devolucao (psp_pagamento_id, cobranca_id, inscricao_id, valor_centavos, motivo, metodo_origem, estado, psp_devolucao_id, concluida_em)
		SELECT p.psp_pagamento_id, p.cobranca_id, $2, 8000, 'PAGO_EM_DUPLICIDADE', 'boleto', 'CONCLUIDA', 'dev_' || $1, now() - interval '1 day'
		FROM pagamento_recebido p WHERE p.psp_pagamento_id = $1`, "pag_b_"+sufixo+"_dup", pagas[1])
	must(t, err)
	// Um aviso que voltou, com o endereço no diagnóstico do servidor de e-mail, como costuma vir.
	_, err = pool.Exec(ctx, `
		WITH n AS (INSERT INTO notificacao (evento_id, inscricao_id, destinatario, assunto, resumo, provedor_id)
		           SELECT e.id, $1, 'x@exemplo.test', 'Pagamento confirmado', repeat('a', 64), 'ses-exp-' || e.id
		           FROM inscricao_evento e WHERE e.inscricao_id = $1 ORDER BY e.id DESC LIMIT 1 RETURNING provedor_id)
		INSERT INTO notificacao_ocorrencia (sns_id, provedor_id, tipo, destinatario, detalhe, ocorrida_em)
		SELECT 'sns-' || provedor_id, provedor_id, 'DEVOLVIDA', 'x@exemplo.test',
		       'Permanent/General: smtp; 550 5.1.1 <x@exemplo.test>: user unknown', now() FROM n`, pagas[0])
	must(t, err)
	_, err = arrecadacao.FecharAteOntem(ctx, pool, cert.ID, "ana")
	must(t, err)

	cofre := arquivo.CofreDisco{Dir: t.TempDir()}
	_, gerou, err := ancora.Ancorar(ctx, pool, cofre, nil, cert.ID)
	must(t, err)
	if !gerou {
		t.Fatal("a âncora não saiu")
	}
	pasta := filepath.Join(t.TempDir(), "pacote")
	_, err = Exportar(ctx, pool, cofre, cert.ID, pasta)
	must(t, err)
	return pasta
}

// reselar é o que um adulterador competente faria depois de mexer num arquivo: refaz o SHA-256 de
// cada um e o selo, para o manifesto voltar a bater.
func reselar(t *testing.T, pasta string) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(pasta, "manifesto.json"))
	must(t, err)
	var m Manifesto
	must(t, json.Unmarshal(b, &m))
	for i, a := range m.Arquivos {
		c, err := os.ReadFile(filepath.Join(pasta, filepath.FromSlash(a.Nome)))
		must(t, err)
		h := sha256.Sum256(c)
		m.Arquivos[i].SHA256, m.Arquivos[i].Bytes = hex.EncodeToString(h[:]), int64(len(c))
	}
	m.Selo = selo(m.Arquivos)
	b, _ = json.MarshalIndent(m, "", "  ")
	must(t, os.WriteFile(filepath.Join(pasta, "manifesto.json"), b, 0o644))
}

// mexerCSV reescreve um CSV do pacote linha a linha.
func mexerCSV(t *testing.T, pasta, nome string, f func(cab []string, linhas [][]string) [][]string) {
	t.Helper()
	arq, err := os.Open(filepath.Join(pasta, nome))
	must(t, err)
	r := csv.NewReader(arq)
	r.Comma = ';'
	todas, err := r.ReadAll()
	arq.Close()
	must(t, err)
	saida, err := os.Create(filepath.Join(pasta, nome))
	must(t, err)
	w := csv.NewWriter(saida)
	w.Comma = ';'
	must(t, w.WriteAll(append([][]string{todas[0]}, f(todas[0], todas[1:])...)))
	must(t, saida.Close())
}

func coluna(cab []string, nome string) int {
	for i, c := range cab {
		if c == nome {
			return i
		}
	}
	return -1
}

func temProblema(v Verificacao, prefixo string) bool {
	for _, p := range v.Problemas {
		if strings.HasPrefix(p, prefixo) {
			return true
		}
	}
	return false
}

// O pacote íntegro confere em tudo: arquivos, selo, corrente, selos das inscrições, âncora e o dia
// fechado refeito dos pagamentos e das devoluções. Sem CPF inteiro, nome nem e-mail.
func TestPacoteIntegroConfere(t *testing.T) {
	pasta := exportado(t)
	v, err := Verificar(pasta)
	must(t, err)
	// Eventos: criar e finalizar nas três, e confirmar o pagamento em duas.
	if len(v.Problemas) != 0 || v.Inscricoes != 3 || v.Eventos != 3*2+2 || v.PontosAncorados != 3 || v.DiasFechados != 1 {
		t.Fatalf("verificação: %+v", v)
	}
	insc, err := os.ReadFile(filepath.Join(pasta, "inscricoes.csv"))
	must(t, err)
	if !strings.Contains(string(insc), "***.") {
		t.Fatalf("CPF sem máscara:\n%s", insc)
	}
	// Nenhum arquivo do pacote leva nome ou e-mail: nem o aviso que voltou com o endereço no diagnóstico.
	must(t, filepath.WalkDir(pasta, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(p)
		if err == nil && (strings.Contains(string(b), "Exportada") || strings.Contains(string(b), "@exemplo.test")) {
			t.Errorf("dado pessoal em %s:\n%s", d.Name(), b)
		}
		return err
	}))
	oc, err := os.ReadFile(filepath.Join(pasta, "avisos_ocorrencias.csv"))
	must(t, err)
	if !strings.Contains(string(oc), "DEVOLVIDA;Permanent/General") {
		t.Fatalf("a devolução do aviso não está no pacote:\n%s", oc)
	}
}

// Um evento da trilha alterado (quem decidiu), com o manifesto refeito: a corrente refeita do arquivo
// acusa a inscrição e o evento.
func TestTrilhaAlteradaEReseladaEAcusada(t *testing.T) {
	pasta := exportado(t)
	mexerCSV(t, pasta, "trilha.csv", func(cab []string, ls [][]string) [][]string {
		ls[1][coluna(cab, "ator_id")] = "outra pessoa"
		return ls
	})
	reselar(t, pasta)
	v, err := Verificar(pasta)
	must(t, err)
	if !temProblema(v, "TRILHA QUEBRADA") || temProblema(v, "ARQUIVO ALTERADO") {
		t.Fatalf("problemas: %v", v.Problemas)
	}
}

// A trilha reescrita inteira, com a corrente refeita para ficar coerente, e o manifesto refeito: só a
// âncora do cofre, que estava fora do alcance de quem reescreveu, acusa.
func TestTrilhaReescritaCoerenteEAcusadaPelaAncora(t *testing.T) {
	pasta := exportado(t)
	mexerCSV(t, pasta, "trilha.csv", func(cab []string, ls [][]string) [][]string {
		c := func(n string) int { return coluna(cab, n) }
		ls[0][c("ator_id")] = "outra pessoa"
		var anterior []byte
		atual := ""
		for _, l := range ls {
			if l[c("inscricao")] != atual {
				atual, anterior = l[c("inscricao")], nil
			}
			insc, _ := strconv.ParseInt(l[c("inscricao")], 10, 64)
			versao, _ := strconv.Atoi(l[c("edital_versao")])
			em, _ := time.Parse("2006-01-02T15:04:05.000000", l[c("ocorrido_em_utc")])
			anterior = inscricao.ResumoDoEvento(anterior, insc, l[c("evento")], l[c("de_estado")], l[c("para_estado")], l[c("ator_tipo")],
				l[c("ator_id")], l[c("motivo")], versao, em)
			l[c("resumo")] = hex.EncodeToString(anterior)
		}
		return ls
	})
	// O selo da inscrição também é refeito, para o reescritor não ser pego por ele.
	ultimo := map[string]string{}
	mexerCSV(t, pasta, "trilha.csv", func(cab []string, ls [][]string) [][]string {
		for _, l := range ls {
			ultimo[l[coluna(cab, "inscricao")]] = l[coluna(cab, "resumo")]
		}
		return ls
	})
	mexerCSV(t, pasta, "inscricoes.csv", func(cab []string, ls [][]string) [][]string {
		for _, l := range ls {
			l[coluna(cab, "selo")] = ultimo[l[coluna(cab, "inscricao")]]
		}
		return ls
	})
	reselar(t, pasta)
	v, err := Verificar(pasta)
	must(t, err)
	if !temProblema(v, "ÂNCORA Nº 1 NÃO CONFERE") || temProblema(v, "TRILHA QUEBRADA") || temProblema(v, "SELO DA INSCRIÇÃO") {
		t.Fatalf("problemas: %v", v.Problemas)
	}
}

// Um pagamento de um dia fechado some do pacote, com o manifesto refeito: o dia fechado acusa.
func TestPagamentoSumidoEAcusadoPeloDiaFechado(t *testing.T) {
	pasta := exportado(t)
	mexerCSV(t, pasta, "pagamentos.csv", func(_ []string, ls [][]string) [][]string { return ls[1:] })
	reselar(t, pasta)
	v, err := Verificar(pasta)
	must(t, err)
	if !temProblema(v, "DIA ") || !strings.Contains(strings.Join(v.Problemas, "|"), "FECHADO NÃO CONFERE") {
		t.Fatalf("problemas: %v", v.Problemas)
	}
}

// Arquivo a mais, arquivo a menos e o manifesto mexido sem refazer o selo.
func TestArquivosEManifesto(t *testing.T) {
	pasta := exportado(t)
	must(t, os.WriteFile(filepath.Join(pasta, "intruso.csv"), []byte("x"), 0o644))
	must(t, os.Remove(filepath.Join(pasta, "avisos.csv")))
	b, err := os.ReadFile(filepath.Join(pasta, "manifesto.json"))
	must(t, err)
	var m Manifesto
	must(t, json.Unmarshal(b, &m))
	m.Arquivos = m.Arquivos[1:] // tira um arquivo do manifesto, sem refazer o selo
	b, _ = json.Marshal(m)
	must(t, os.WriteFile(filepath.Join(pasta, "manifesto.json"), b, 0o644))
	v, err := Verificar(pasta)
	must(t, err)
	for _, quer := range []string{"ARQUIVO A MAIS: intruso.csv", "ARQUIVO FALTANDO: avisos.csv", "SELO NÃO CONFERE"} {
		if !temProblema(v, quer) {
			t.Errorf("não acusou %q: %v", quer, v.Problemas)
		}
	}
}
