// banca faz, pela linha de comando, o que o painel da banca faria (o painel está fora do escopo da
// POC): levar as inscrições à lista definitiva, publicar gabarito, importar as folhas lidas e
// calcular o resultado. A publicação estática sai pela fila do servidor (cmd/lisura).
//
//	banca homologar -certame 5 -analista ana
//	banca gabarito  -cargo 13 -tipo preliminar -respostas ABCDE... -motivo "Gabarito preliminar"
//	banca folhas    -arquivo leitura.csv -lote lote-01          (CSV: inscricao;respostas)
//	banca sintetico -certame 5 -lote ensaio                     (folhas sintéticas, para o ensaio)
//	banca calcular  -certame 5 -tipo preliminar -motivo "Resultado preliminar"
//	banca ancorar   -certame 5        (fixa a trilha no cofre agora, ex.: no fim das inscrições)
//	banca cancelar  -certame 5 -analista ana -motivo ANULADO_POR_DECISAO_JUDICIAL  (devolve tudo o que foi pago)
//	banca arrecadacao -certame 5 [-json]   (prestação de contas: por dia, cargo e cidade; sai com 1 se a conta não fecha)
//	banca fechar      -certame 5 -analista ana   (fecha cada dia terminado; dia fechado não muda)
//	banca conferir    -certame 5   (a conta, os dias fechados e o extrato do prestador; sai com 1 se achar problema)
//	banca locais      -certame 5 -arquivo locais.csv   (locais e salas; cabeçalho em ensalamento.Cabecalho)
//	banca ensalar     -certame 5 -analista ana -motivo ENSALAMENTO_INICIAL   (sai com 1 e diz o que falta se faltar lugar)
//	banca mapa        -certame 5   (ocupação por local)
//	banca listas      -certame 5 -pasta listas   (uma lista de presença por sala, CSV)
//	banca conferir-ensalamento -certame 5
//	banca cronograma -certame 5   (as fases e o que quebra; sai com 1 se algo quebra)
//	banca retificar  -certame 5 -fase inscricao -fecha 2026-12-10T23:59 -analista ana -motivo PRORROGACAO [-simular]
//	banca exportar   -certame 5 -pasta pacote   (o pacote para o órgão e a auditoria)
//	banca verificar-exportacao -pasta pacote    (sem banco: o que o auditor roda; sai com 1 se achar algo)
package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"lisura/internal/ancora"
	"lisura/internal/arquivo"
	"lisura/internal/arrecadacao"
	"lisura/internal/cronograma"
	"lisura/internal/db"
	"lisura/internal/ensalamento"
	"lisura/internal/exportacao"
	"lisura/internal/pagamento"
	"lisura/internal/psp"
	"lisura/internal/resultado"
)

func main() {
	if len(os.Args) < 2 {
		uso()
	}
	ctx := context.Background()
	fs := flag.NewFlagSet(os.Args[1], flag.ExitOnError)
	certame := fs.Int64("certame", 0, "certame")
	cargo := fs.Int64("cargo", 0, "cargo")
	analista := fs.String("analista", "", "quem homologa (vai para a trilha)")
	tipo := fs.String("tipo", "preliminar", "preliminar ou definitivo")
	respostas := fs.String("respostas", "", "gabarito: uma letra por questão, * para anulada")
	motivo := fs.String("motivo", "", "motivo (obrigatório para gabarito e cálculo)")
	csvLeitora := fs.String("arquivo", "", "CSV da leitora: inscricao;respostas")
	lote := fs.String("lote", "", "identificação do lote de folhas")
	ausentes := fs.Float64("ausentes", 0.12, "sintético: fração de ausentes")
	semente := fs.Uint64("semente", 1, "sintético: semente (mesma semente, mesmas folhas)")
	json_ := fs.Bool("json", false, "arrecadacao: JSON em vez de Markdown")
	pasta := fs.String("pasta", "", "listas: pasta onde gravar uma lista de presença (CSV) por sala")
	fase := fs.String("fase", "", "retificar: a fase (isencao, inscricao, pagamento, homologacao, ensalamento, aplicacao, gabarito, resultado)")
	abre := fs.String("abre", "", "retificar: novo início, AAAA-MM-DDTHH:MM no horário de Brasília (vazio mantém)")
	fecha := fs.String("fecha", "", "retificar: novo fim, AAAA-MM-DDTHH:MM no horário de Brasília (vazio mantém)")
	simular := fs.Bool("simular", false, "retificar: só mostrar o que quebraria e quem seria alcançado, sem gravar")
	fs.Parse(os.Args[2:])

	if os.Args[1] == "verificar-exportacao" {
		// Sem banco: é o que o auditor roda com a pasta recebida, e nada mais.
		inicio := time.Now()
		exigir(*pasta != "", "-pasta")
		v, err := exportacao.Verificar(*pasta)
		fatal(err)
		b, _ := json.MarshalIndent(v, "", "  ")
		fmt.Println(string(b))
		log.Printf("conferido em %s", time.Since(inicio).Round(time.Millisecond))
		if len(v.Problemas) > 0 {
			os.Exit(1)
		}
		return
	}

	pool, err := db.Connect(ctx, db.URL())
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		log.Fatal(err)
	}

	inicio := time.Now()
	switch os.Args[1] {
	case "homologar":
		exigir(*certame > 0 && *analista != "", "-certame e -analista")
		n, err := resultado.HomologarLote(ctx, pool, *certame, *analista)
		fatal(err)
		fmt.Printf("%d inscrições homologadas em %s\n", n, time.Since(inicio).Round(time.Millisecond))
	case "gabarito":
		exigir(*cargo > 0 && *respostas != "" && *motivo != "", "-cargo, -respostas e -motivo")
		v, err := resultado.PublicarGabarito(ctx, pool, *cargo, *tipo, strings.ToUpper(*respostas), *motivo)
		fatal(err)
		fmt.Printf("gabarito %s versão %d do cargo %d gravado; a publicação sai pela fila\n", *tipo, v, *cargo)
	case "folhas":
		exigir(*csvLeitora != "" && *lote != "", "-arquivo e -lote")
		folhas, err := lerCSV(*csvLeitora)
		fatal(err)
		importar(ctx, pool, *lote, folhas)
	case "sintetico":
		exigir(*certame > 0 && *lote != "", "-certame e -lote")
		folhas, err := sinteticas(ctx, pool, *certame, *ausentes, *semente)
		fatal(err)
		importar(ctx, pool, *lote, folhas)
		fmt.Printf("em %s\n", time.Since(inicio).Round(time.Millisecond))
	case "calcular":
		exigir(*certame > 0 && *motivo != "", "-certame e -motivo")
		res, err := resultado.CalcularCertame(ctx, pool, *certame, *tipo, *motivo)
		fatal(err)
		b, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(b))
		fmt.Printf("versão %d calculada em %s; a publicação sai pela fila\n", res.Versao, res.Duracao.Round(time.Millisecond))
	case "cancelar":
		exigir(*certame > 0 && *analista != "" && *motivo != "", "-certame, -analista e -motivo (código, ex.: ANULADO_POR_DECISAO_JUDICIAL)")
		res, err := pagamento.CancelarCertame(ctx, pool, *certame, *analista, *motivo)
		fatal(err)
		fmt.Printf("certame %d cancelado: %d inscrições canceladas, %d devoluções criadas, em %s; as devoluções saem pela fila\n",
			*certame, res.Inscricoes, res.Devolucoes, res.Duracao.Round(time.Millisecond))
	case "arrecadacao":
		exigir(*certame > 0, "-certame")
		p, err := arrecadacao.Gerar(ctx, pool, *certame)
		fatal(err)
		if *json_ {
			b, _ := json.MarshalIndent(p, "", "  ")
			fmt.Println(string(b))
		} else {
			fmt.Print(p.Markdown())
		}
		log.Printf("gerada em %s", time.Since(inicio).Round(time.Millisecond))
		if p.Diferenca != 0 {
			os.Exit(1)
		}
	case "fechar":
		exigir(*certame > 0 && *analista != "", "-certame e -analista (quem fecha)")
		dias, err := arrecadacao.FecharAteOntem(ctx, pool, *certame, *analista)
		fatal(err)
		for _, f := range dias {
			fmt.Printf("%s fechado: entrou %d (%s), saiu %d (%s), resumo %s\n", f.Dia, f.Entrou.N, arrecadacao.BRL(f.Entrou.Centavos),
				f.Saiu.N, arrecadacao.BRL(f.Saiu.Centavos), f.Resumo[:16])
		}
		fmt.Printf("%d dias fechados agora, em %s\n", len(dias), time.Since(inicio).Round(time.Millisecond))
	case "conferir":
		exigir(*certame > 0, "-certame")
		prestador := psp.Cliente{URL: envOu("LISURA_PSP_URL", "http://localhost:8081"), HTTP: psp.NovoHTTP(2 * time.Minute)}
		c, err := arrecadacao.Conferir(ctx, pool, prestador, *certame)
		fatal(err)
		b, _ := json.MarshalIndent(c, "", "  ")
		fmt.Println(string(b))
		if len(c.Problemas) > 0 {
			os.Exit(1)
		}
	case "locais":
		exigir(*certame > 0 && *csvLeitora != "", "-certame e -arquivo (CSV: "+ensalamento.Cabecalho+")")
		f, err := os.Open(*csvLeitora)
		fatal(err)
		defer f.Close()
		l, s, err := ensalamento.ImportarLocais(ctx, pool, *certame, f)
		fatal(err)
		fmt.Printf("%d locais e %d salas importados em %s\n", l, s, time.Since(inicio).Round(time.Millisecond))
	case "ensalar":
		exigir(*certame > 0 && *analista != "" && *motivo != "", "-certame, -analista e -motivo")
		v, err := ensalamento.Ensalar(ctx, pool, *certame, *analista, *motivo)
		var falta *ensalamento.ErrFaltaLugar
		if errors.As(err, &falta) {
			for _, x := range falta.Faltas {
				fmt.Printf("FALTA LUGAR em %s: %d lugares em %s\n", x.Cidade, x.Faltam, x.Requisito)
			}
			os.Exit(1)
		}
		fatal(err)
		if !v.Nova {
			fmt.Printf("nada mudou desde a versão %d (%d alocadas, resumo %s…), em %s\n", v.Versao, v.Alocadas, v.Resumo[:16], v.Duracao.Round(time.Millisecond))
			return
		}
		fmt.Printf("ensalamento versão %d: %d alocadas (%d mantidas, %d novas, %d removidas), resumo %s…, em %s; os cartões saem pela fila\n",
			v.Versao, v.Alocadas, v.Mantidas, v.Novas, v.Removidas, v.Resumo[:16], v.Duracao.Round(time.Millisecond))
	case "mapa":
		exigir(*certame > 0, "-certame")
		m, err := ensalamento.Mapa(ctx, pool, *certame)
		fatal(err)
		fmt.Print(ensalamento.MapaMarkdown(m))
	case "listas":
		exigir(*certame > 0 && *pasta != "", "-certame e -pasta")
		ls, err := ensalamento.Listas(ctx, pool, *certame)
		fatal(err)
		fatal(os.MkdirAll(*pasta, 0o755))
		for i, l := range ls {
			escreverLista(*pasta, i+1, l)
		}
		fmt.Printf("%d listas de presença em %s, em %s\n", len(ls), *pasta, time.Since(inicio).Round(time.Millisecond))
	case "conferir-ensalamento":
		exigir(*certame > 0, "-certame")
		c, err := ensalamento.Conferir(ctx, pool, *certame)
		fatal(err)
		b, _ := json.MarshalIndent(c, "", "  ")
		fmt.Println(string(b))
		if len(c.Problemas) > 0 {
			os.Exit(1)
		}
	case "cronograma":
		exigir(*certame > 0, "-certame")
		c, err := cronograma.Ler(ctx, pool, *certame)
		fatal(err)
		for _, f := range cronograma.Fases {
			if j, ok := c[f]; ok {
				fmt.Printf("%-12s %s → %s\n", f, j.AbreEm.Local().Format("02/01/2006 15:04"), j.FechaEm.Local().Format("02/01/2006 15:04"))
			}
		}
		qs := c.Conferir()
		for _, q := range qs {
			fmt.Printf("QUEBRA %s: %s\n", q.Regra, q.Detalhe)
		}
		if len(qs) > 0 {
			os.Exit(1)
		}
	case "retificar":
		exigir(*certame > 0 && *fase != "" && (*abre != "" || *fecha != ""), "-certame, -fase e -abre ou -fecha (AAAA-MM-DDTHH:MM, horário de Brasília)")
		r := cronograma.Retificacao{Fase: *fase, Motivo: *motivo, Quem: *analista}
		var err error
		if r.Abre, err = horario(*abre); err == nil {
			r.Fecha, err = horario(*fecha)
		}
		fatal(err)
		var e cronograma.Efeito
		if *simular {
			e, err = cronograma.Simular(ctx, pool, *certame, r)
		} else {
			e, err = cronograma.Retificar(ctx, pool, *certame, r)
		}
		var q *cronograma.ErrQuebra
		if err != nil && !errors.As(err, &q) {
			fatal(err)
		}
		b, _ := json.MarshalIndent(e, "", "  ")
		fmt.Println(string(b))
		if err != nil {
			fmt.Println("RECUSADA:", err)
			os.Exit(1)
		}
		if len(e.Quebras) > 0 { // simulação que quebraria: o mesmo código de saída da recusa
			os.Exit(1)
		}
	case "exportar":
		exigir(*certame > 0 && *pasta != "", "-certame e -pasta (vazia)")
		cofre, _, err := arquivo.CofreDoAmbiente(ctx)
		fatal(err)
		m, err := exportacao.Exportar(ctx, pool, cofre, *certame, *pasta)
		fatal(err)
		var total int64
		for _, a := range m.Arquivos {
			total += a.Bytes
		}
		fmt.Printf("certame %d exportado em %s: %d arquivos, %d bytes, selo %s, em %s\n", *certame, *pasta, len(m.Arquivos), total, m.Selo,
			time.Since(inicio).Round(time.Millisecond))
	case "ancorar":
		// O recibo público fica para a rotina do servidor, que o republica se faltar.
		exigir(*certame > 0, "-certame")
		cofre, onde, err := arquivo.CofreDoAmbiente(ctx)
		fatal(err)
		a, gerou, err := ancora.Ancorar(ctx, pool, cofre, nil, *certame)
		fatal(err)
		if !gerou {
			fmt.Printf("nada novo desde a âncora nº %d (ou outra instância ancorou agora)\n", a.Sequencia)
			return
		}
		texto, err := cofre.Ler(ctx, ancora.Chave(*certame, a.Sequencia))
		fatal(err)
		fmt.Printf("âncora nº %d (%s) em %s: %d de %d inscrições, %d eventos, %d bytes, resumo %s, em %s\n", a.Sequencia, a.Tipo, onde,
			len(a.Linhas), a.Total, a.Eventos, len(texto), ancora.Resumo(texto), time.Since(inicio).Round(time.Millisecond))
	default:
		uso()
	}
}

func importar(ctx context.Context, pool *pgxpool.Pool, lote string, folhas []resultado.Folha) {
	// Em blocos: um lote de 200 mil folhas numa transação só seguraria as linhas tempo demais.
	total, recusadas := 0, map[string]int{}
	for i := 0; i < len(folhas); i += 20_000 {
		bloco := folhas[i:min(i+20_000, len(folhas))]
		imp, err := resultado.ImportarFolhas(ctx, pool, lote, bloco)
		fatal(err)
		total += imp.Gravadas
		for _, m := range imp.Recusadas {
			recusadas[m]++
		}
	}
	fmt.Printf("%d folhas gravadas; recusadas: %v\n", total, recusadas)
}

func lerCSV(caminho string) ([]resultado.Folha, error) {
	f, err := os.Open(caminho)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.Comma = ';'
	linhas, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	var folhas []resultado.Folha
	for n, l := range linhas {
		if len(l) != 2 {
			return nil, fmt.Errorf("linha %d: esperado inscricao;respostas", n+1)
		}
		id, err := strconv.ParseInt(strings.TrimSpace(l[0]), 10, 64)
		if err != nil {
			if n == 0 {
				continue // cabeçalho
			}
			return nil, fmt.Errorf("linha %d: inscrição %q", n+1, l[0])
		}
		folhas = append(folhas, resultado.Folha{Inscricao: id, Respostas: strings.ToUpper(strings.TrimSpace(l[1]))})
	}
	return folhas, nil
}

// sinteticas gera a leitura de cada inscrição homologada contra o gabarito em vigor do cargo: cada
// candidato tem uma chance de acerto própria (entre 25% e 95%), erra para uma letra qualquer e deixa
// 2% em branco. Dado sintético, nunca real.
func sinteticas(ctx context.Context, pool *pgxpool.Pool, certame int64, ausentes float64, semente uint64) ([]resultado.Folha, error) {
	rows, err := pool.Query(ctx, `
		SELECT i.id, g.respostas
		  FROM inscricao i
		  JOIN LATERAL (SELECT respostas FROM gabarito WHERE cargo_id = i.cargo_id ORDER BY versao DESC LIMIT 1) g ON true
		 WHERE i.certame_id = $1 AND i.estado = 'HOMOLOGADA' ORDER BY i.id`, certame)
	if err != nil {
		return nil, err
	}
	rnd := rand.New(rand.NewPCG(semente, 0x4c69737572))
	var folhas []resultado.Folha
	var id int64
	var gab string
	_, err = pgx.ForEachRow(rows, []any{&id, &gab}, func() error {
		if rnd.Float64() < ausentes {
			return nil
		}
		acerto := 0.25 + 0.7*rnd.Float64()
		b := []byte(gab)
		for i, certa := range b {
			switch sorte := rnd.Float64(); {
			case sorte < 0.02:
				b[i] = '.'
			case certa == '*' || sorte < acerto:
				if certa == '*' {
					b[i] = "ABCDE"[rnd.IntN(5)]
				}
			default:
				b[i] = "ABCDE"[(strings.IndexByte("ABCDE", certa)+1+rnd.IntN(4))%5]
			}
		}
		folhas = append(folhas, resultado.Folha{Inscricao: id, Respostas: string(b)})
		return nil
	})
	return folhas, err
}

func exigir(ok bool, o string) {
	if !ok {
		log.Fatalf("faltou %s", o)
	}
}

func fatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// horario lê AAAA-MM-DDTHH:MM no horário de Brasília (UTC−3, sem horário de verão desde 2019).
// Vazio é o tempo zero: a retificação mantém o que está.
func horario(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	return time.ParseInLocation("2006-01-02T15:04", s, time.FixedZone("BRT", -3*3600))
}

// escreverLista grava a lista de presença de uma sala: um CSV por sala, numerado na ordem do mapa.
func escreverLista(pasta string, n int, l ensalamento.ListaDaSala) {
	f, err := os.Create(fmt.Sprintf("%s/%05d.csv", pasta, n))
	fatal(err)
	defer f.Close()
	w := csv.NewWriter(f)
	w.Comma = ';'
	w.Write([]string{"cidade", "local", "endereco", "sala", "inscricao", "nome", "cpf", "cargo", "atendimento", "assinatura"})
	for _, p := range l.Presencas {
		w.Write([]string{l.Cidade, l.Local, l.Endereco, l.Sala, strconv.FormatInt(p.Inscricao, 10), p.Nome, p.CPF, p.Cargo,
			strings.Join(p.Necessidades, ","), ""})
	}
	w.Flush()
	fatal(w.Error())
}

func envOu(nome, padrao string) string {
	if v := os.Getenv(nome); v != "" {
		return v
	}
	return padrao
}

func uso() {
	fmt.Fprintln(os.Stderr, "uso: banca homologar|gabarito|folhas|sintetico|calcular|ancorar|cancelar|arrecadacao|fechar|conferir|locais|ensalar|mapa|listas|conferir-ensalamento|cronograma|retificar|exportar|verificar-exportacao [opções]  (banca <comando> -h)")
	os.Exit(2)
}
