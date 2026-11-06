// lisura sobe a API JSON em /api. As telas são o front Next (../poc-web), que repassa /api para cá.
//
//	DATABASE_URL       banco (padrão: Postgres local da POC)
//	LISURA_ADDR        endereço (padrão :8082)
//	LISURA_ARQUIVOS    pasta dos documentos enviados (padrão ./dados/arquivos)
//	LISURA_TOKEN_KEY   chave HMAC dos tokens; sem ela, uma chave aleatória por processo
//	LISURA_PSP_URL     prestador de pagamento (padrão: o simulado em http://localhost:8081)
//	LISURA_PSP_SEGREDO segredo que assina os avisos de pagamento
//	LISURA_SANDBOX     se definido, liga a rota que simula o candidato pagando (só com o PSP simulado)
//	LISURA_PUBLICO     pasta dos arquivos estáticos servidos em /publico (padrão ./dados/publico)
//	LISURA_TRABALHADORES quantos trabalhadores da fila rodam (padrão 4)
//	LISURA_S3_BUCKET   se definido, documentos e publicação vão para este bucket (credenciais pelo papel da tarefa)
//	LISURA_OPERACAO_SEGREDO autoriza POST /api/operacao (mudar o nível de degradação); sem ele, a rota recusa
//	LISURA_COFRE_BUCKET bucket com bloqueio de objeto onde vão as âncoras da trilha; sem ele, a pasta
//	                   LISURA_COFRE (padrão ./dados/cofre)
//	LISURA_ANCORA_INTERVALO de quanto em quanto tempo a trilha de cada certame é ancorada (padrão 1h)
//	LISURA_EMAIL       provedor dos avisos: ses, descarte, ou (vazio) a caixa de saída LISURA_EMAIL_CAIXA
//	LISURA_SES_DE      com LISURA_EMAIL=ses: o remetente verificado; LISURA_SES_CONJUNTO, o configuration
//	                   set; LISURA_SES_TAXA, envios por segundo abaixo da cota da conta
//	LISURA_VERSAO      rótulo da instância nas métricas (GET /metrics; padrão "local")
//	LISURA_SES_TOPICO  ARN do tópico SNS dos eventos do SES: liga POST /api/ses/eventos (devoluções e
//	                   reclamações; só aceita mensagem assinada pelo SNS deste tópico)
package main

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"lisura/internal/ancora"
	"lisura/internal/api"
	"lisura/internal/arquivo"
	"lisura/internal/arrecadacao"
	"lisura/internal/db"
	"lisura/internal/degradacao"
	"lisura/internal/fila"
	"lisura/internal/metricas"
	"lisura/internal/notificacao"
	"lisura/internal/pagamento"
	"lisura/internal/psp"
	"lisura/internal/publicacao"
	"lisura/internal/resultado"
)

func main() {
	ctx := context.Background()
	pool, err := db.Connect(ctx, db.URL())
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		log.Fatal(err)
	}

	chave := []byte(os.Getenv("LISURA_TOKEN_KEY"))
	if len(chave) == 0 {
		chave = make([]byte, 32)
		rand.Read(chave)
		log.Print("LISURA_TOKEN_KEY ausente: chave aleatória, sessões não sobrevivem a reinício")
	}
	tokens := api.Tokens{Chave: chave, Validade: 12 * time.Hour}
	// Arquivos: em disco na máquina local; com LISURA_S3_BUCKET, no S3 (documentos privados em
	// documentos/, publicação em publico/, servida pelo CloudFront).
	var store arquivo.Destino = arquivo.Disco{Dir: env("LISURA_ARQUIVOS", "dados/arquivos")}
	var destinoPublico arquivo.Destino
	if bucket := os.Getenv("LISURA_S3_BUCKET"); bucket != "" {
		cfg, err := awsconfig.LoadDefaultConfig(ctx)
		if err != nil {
			log.Fatal(err)
		}
		cliente := s3.NewFromConfig(cfg)
		docs := arquivo.S3{Cliente: cliente, Bucket: bucket, Prefixo: "documentos/"}
		if err := docs.Verificar(ctx); err != nil {
			log.Fatal(err)
		}
		store = docs
		destinoPublico = arquivo.S3{Cliente: cliente, Bucket: bucket, Prefixo: "publico/", CacheControl: "public, max-age=60"}
		log.Printf("arquivos no S3: s3://%s", bucket)
	}
	prestador := &psp.Cliente{URL: env("LISURA_PSP_URL", "http://localhost:8081"), HTTP: psp.NovoHTTP(5 * time.Second)}
	segredoPSP := []byte(env("LISURA_PSP_SEGREDO", "segredo-do-sandbox"))

	publicador := &publicacao.Publicador{Pool: pool, Dir: env("LISURA_PUBLICO", "dados/publico"), Chave: chave, Destino: destinoPublico}

	med := metricas.Nova(env("LISURA_VERSAO", "local"))
	s := &api.Server{Pool: pool, Store: store, Tokens: tokens, SegredoPSP: segredoPSP, Publicador: publicador, Metricas: med}
	if os.Getenv("LISURA_SANDBOX") != "" {
		s.Sandbox = prestador
		log.Print("modo sandbox: rota de simular pagamento ligada")
	}
	// Degradação graciosa: tudo em /api passa pela sala de espera e pelos níveis; a situação (que o
	// front consulta) e a operação (que muda o nível) ficam fora, para nunca entrarem na própria fila.
	// O nível e as senhas moram no banco, para valerem em todas as instâncias atrás do balanceador.
	controle := &degradacao.Controle{Chave: chave, Segredo: os.Getenv("LISURA_OPERACAO_SEGREDO"),
		Armazem: degradacao.Postgres{Pool: pool}}
	if controle.Segredo == "" {
		log.Print("LISURA_OPERACAO_SEGREDO ausente: rota de operação desligada")
	}
	if err := controle.Atualizar(ctx); err != nil {
		log.Fatal(err)
	}
	go controle.Acompanhar(ctx, time.Second, func(err error) { log.Printf("sala de espera: %v", err) })
	mux := http.NewServeMux()
	mux.Handle("/api/", http.StripPrefix("/api", controle.Middleware(s.Routes())))
	mux.Handle("/api/situacao", controle.Situacao())
	mux.Handle("/api/operacao", controle.Operacao())
	// Devoluções e reclamações dos avisos, que o SES publica no SNS. Fora da sala de espera: o SNS
	// não espera na fila, repete. Só existe com o tópico configurado.
	if topico := os.Getenv("LISURA_SES_TOPICO"); topico != "" {
		mux.Handle("/api/ses/eventos", notificacao.ReceberEventosSES(pool, &notificacao.VerificadorSNS{Topico: topico}))
		log.Printf("eventos do SES: recebendo do tópico %s", topico)
	}
	// Na nuvem quem serve /publico é o CloudFront, direto do S3; na máquina local, esta pasta,
	// servida sem tocar no banco.
	if destinoPublico == nil {
		mux.Handle("/publico/", http.StripPrefix("/publico/", publicacao.Servir(publicador.Dir)))
	}

	// Trabalhadores da fila (na arquitetura alvo, Lambdas lendo o SQS).
	manipuladores := publicador.Manipuladores()
	manipuladores[pagamento.TarefaCriarNoPrestador] = pagamento.CriarNoPrestador(prestador, pool)
	manipuladores[pagamento.TarefaDevolver] = pagamento.Devolver(prestador, pool)
	correio, ondeCorreio, err := remetente(ctx)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("avisos ao candidato: %s", ondeCorreio)
	manipuladores[notificacao.Tarefa] = notificacao.Notificar(correio, pool)
	for tipo, m := range (&resultado.Publicador{Pool: pool, Pub: publicador}).Manipuladores() {
		manipuladores[tipo] = m // gabarito e resultado (picos 4 e 5)
	}
	trabalhador := &fila.Trabalhador{Pool: pool, Manipuladores: manipuladores, Observar: med.ObservarFila}
	// O painel em tempo real: o Prometheus raspa daqui. Fora de /api (o balanceador da AWS só repassa
	// /api e o front, então /metrics fica dentro da VPC).
	med.Acompanhar(pool, trabalhador, controle)
	mux.Handle("GET /metrics", med.Handler())
	if err := publicacao.RepublicarTodos(ctx, pool); err != nil {
		log.Printf("republicar: %v", err)
	}
	go trabalhador.Rodar(ctx, atoi(env("LISURA_TRABALHADORES", "4")))
	go relatarFila(ctx, pool, trabalhador)

	go rotinaDePagamentos(ctx, pool, prestador)

	cofre, onde, err := arquivo.CofreDoAmbiente(ctx)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("âncoras da trilha em %s", onde)
	intervalo, err := time.ParseDuration(env("LISURA_ANCORA_INTERVALO", "1h"))
	if err != nil {
		log.Fatal(err)
	}
	go rotinaDeAncoras(ctx, pool, cofre, publicador, intervalo)

	srv := &http.Server{
		Addr:              env("LISURA_ADDR", ":8082"),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
	}
	log.Printf("ouvindo em %s", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}

// rotinaDePagamentos expira o que venceu e concilia contra o extrato do prestador, a cada minuto, em
// janelas escalonadas (pagamento.JanelaDaConciliacao). Conciliar 48 h a cada minuto baixava o extrato
// inteiro toda vez: medido em 05/11, a 1.280× local, 268 mil pagamentos e 20,7 s por rodada,
// disputando o prestador e o banco com as inscrições; 2 h a cada minuto ainda custava 7 a 17 s.
// Na arquitetura alvo isto é uma função agendada; aqui, uma goroutine.
func rotinaDePagamentos(ctx context.Context, pool *pgxpool.Pool, prestador *psp.Cliente) {
	rodada := 0
	for range time.Tick(time.Minute) {
		janela := pagamento.JanelaDaConciliacao(rodada)
		if rodada%10 == 0 {
			// Devolução cuja tarefa foi abandonada (prestador fora do ar por mais de meio minuto) volta
			// para a fila; nenhuma morre lá.
			if n, err := pagamento.RetomarDevolucoes(ctx, pool); err != nil {
				log.Printf("devoluções: %v", err)
			} else if n > 0 {
				log.Printf("devoluções: %d reenfileiradas", n)
			}
		}
		if rodada%60 == 0 {
			fecharArrecadacao(ctx, pool)
		}
		rodada++
		if n, err := pagamento.ExpirarVencidas(ctx, pool, pagamento.MargemCompensacao); err != nil {
			log.Printf("expirar: %v", err)
		} else if n > 0 {
			log.Printf("expirar: %d inscrições expiradas", n)
		}
		agora := time.Now()
		rel, executou, err := pagamento.ConciliarSozinho(ctx, pool, prestador, agora.Add(-janela), agora)
		if err != nil {
			log.Printf("conciliar: %v", err)
			continue
		}
		if !executou {
			continue // outra instância está conciliando
		}
		// A duração vai sempre para o log: é o número que mostra se a conciliação acompanha o volume.
		log.Printf("conciliar %s: %d no extrato em %s · %v · sem registro no prestador: %d", janela,
			rel.NoExtrato, time.Since(agora).Round(time.Millisecond), rel.PorResultado, rel.SemRegistroNoPSP)
	}
}

// remetente escolhe o provedor de e-mail dos avisos: LISURA_EMAIL=descarte (o ensaio de carga: aceita
// e conta, sem guardar) ou a caixa de saída num arquivo (LISURA_EMAIL_CAIXA; padrão
// dados/caixa-de-saida.jsonl), ou LISURA_EMAIL=ses, o provedor de produção (ver sesDoAmbiente).
func remetente(ctx context.Context) (notificacao.Remetente, string, error) {
	switch os.Getenv("LISURA_EMAIL") {
	case "descarte":
		return &notificacao.Descarte{}, "descarte (ensaio)", nil
	case "ses":
		return sesDoAmbiente(ctx)
	}
	caminho := env("LISURA_EMAIL_CAIXA", "dados/caixa-de-saida.jsonl")
	if err := os.MkdirAll(filepath.Dir(caminho), 0o755); err != nil {
		return nil, "", err
	}
	c, err := notificacao.AbrirCaixa(caminho)
	return c, "caixa de saída " + caminho, err
}

// fecharArrecadacao fecha, uma vez por hora, todo dia terminado de cada certame: a fotografia da conta
// que a prestação de contas confere depois (internal/arrecadacao). Fechar de novo não regrava nada.
func fecharArrecadacao(ctx context.Context, pool *pgxpool.Pool) {
	rows, err := pool.Query(ctx, `SELECT id FROM certame ORDER BY id`)
	if err != nil {
		log.Printf("fechamento: %v", err)
		return
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		log.Printf("fechamento: %v", err)
		return
	}
	for _, id := range ids {
		inicio := time.Now()
		dias, err := arrecadacao.FecharAteOntem(ctx, pool, id, "sistema")
		if err != nil {
			log.Printf("fechamento do certame %d: %v", id, err)
			continue
		}
		for _, f := range dias {
			log.Printf("fechamento do certame %d, dia %s: entrou %d (%s), saiu %d (%s), em %s", id, f.Dia,
				f.Entrou.N, arrecadacao.BRL(f.Entrou.Centavos), f.Saiu.N, arrecadacao.BRL(f.Saiu.Centavos),
				time.Since(inicio).Round(time.Millisecond))
		}
	}
}

// rotinaDeAncoras fixa a trilha de cada certame fora do banco (ver internal/ancora). Certame sem
// evento novo desde a última âncora não gera âncora. Na arquitetura alvo, uma função agendada.
func rotinaDeAncoras(ctx context.Context, pool *pgxpool.Pool, cofre arquivo.Cofre, pub ancora.Gravador, intervalo time.Duration) {
	for range time.Tick(intervalo) {
		rows, err := pool.Query(ctx, `SELECT id FROM certame ORDER BY id`)
		if err != nil {
			log.Printf("âncora: %v", err)
			continue
		}
		ids, err := pgx.CollectRows(rows, pgx.RowTo[int64])
		if err != nil {
			log.Printf("âncora: %v", err)
			continue
		}
		for _, id := range ids {
			inicio := time.Now()
			a, gerou, err := ancora.Ancorar(ctx, pool, cofre, pub, id)
			switch {
			case err != nil:
				log.Printf("âncora do certame %d: %v", id, err)
			case gerou:
				log.Printf("âncora do certame %d: nº %d %s, %d de %d inscrições, %d eventos, em %s",
					id, a.Sequencia, a.Tipo, len(a.Linhas), a.Total, a.Eventos, time.Since(inicio).Round(time.Millisecond))
			}
		}
	}
}

// relatarFila escreve no log, a cada minuto, por tipo de tarefa: quantas rodaram, quanto esperaram
// na fila e quanto levaram, e quantas estão vencidas esperando trabalhador. É o que mostra, no ensaio,
// se a fila acompanha (na nuvem o log vai para o CloudWatch).
func relatarFila(ctx context.Context, pool *pgxpool.Pool, t *fila.Trabalhador) {
	for range time.Tick(time.Minute) {
		var atrasadas int
		pool.QueryRow(ctx, `SELECT count(*) FROM tarefa WHERE estado = 'PENDENTE' AND executar_em <= now()`).Scan(&atrasadas)
		for tipo, m := range t.Medidas() {
			log.Printf("fila: %s · %d tarefas (%d falhas) · espera média %s, máx %s · execução média %s, máx %s",
				tipo, m.Tarefas, m.Falhas, m.EsperaMedia.Round(time.Millisecond), m.EsperaMax.Round(time.Millisecond),
				m.ExecucaoMedia.Round(time.Millisecond), m.ExecucaoMax.Round(time.Millisecond))
		}
		log.Printf("fila: %d tarefas vencidas esperando trabalhador", atrasadas)
	}
}

// sesDoAmbiente monta o remetente do SES: LISURA_SES_DE (remetente verificado, obrigatório),
// LISURA_SES_CONJUNTO (configuration set, opcional) e as credenciais pelo papel da tarefa. A taxa vem
// da cota da conta, lida na subida (a cota muda por pedido à AWS, não por configuração nossa);
// LISURA_SES_TAXA só pode baixá-la. Sem acesso à conta, o servidor não sobe: um remetente que só
// descobre o problema no primeiro aviso deixaria os avisos encalhados na fila.
func sesDoAmbiente(ctx context.Context) (notificacao.Remetente, string, error) {
	de := os.Getenv("LISURA_SES_DE")
	if de == "" {
		return nil, "", errors.New("LISURA_EMAIL=ses sem LISURA_SES_DE (o remetente verificado no SES)")
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, "", err
	}
	cli := sesv2.NewFromConfig(cfg)
	conta, err := cli.GetAccount(ctx, &sesv2.GetAccountInput{})
	if err != nil {
		return nil, "", fmt.Errorf("SES: %w", err)
	}
	q := conta.SendQuota
	if q == nil || q.MaxSendRate <= 0 {
		return nil, "", errors.New("SES: a conta não informou a cota de envio")
	}
	taxa := q.MaxSendRate
	if s := os.Getenv("LISURA_SES_TAXA"); s != "" {
		if t, err := strconv.ParseFloat(s, 64); err == nil && t > 0 && t < taxa {
			taxa = t
		}
	}
	onde := fmt.Sprintf("SES (%s, de %s): %.0f/s de %.0f/s da cota, %.0f de %.0f nas últimas 24 h",
		cfg.Region, de, taxa, q.MaxSendRate, q.SentLast24Hours, q.Max24HourSend)
	if !conta.ProductionAccessEnabled {
		onde += "; NO SANDBOX: só entrega para endereços verificados, o resto volta como recusado"
	}
	return notificacao.NovoSES(cli, de, os.Getenv("LISURA_SES_CONJUNTO"), taxa), onde, nil
}

func atoi(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return 1
	}
	return n
}

func env(k, padrao string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return padrao
}
