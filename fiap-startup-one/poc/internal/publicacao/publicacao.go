// Package publicacao tira os picos de leitura do banco (arquitetura alvo, frente 2): o que toda a
// base consulta ao mesmo tempo vira arquivo estático, gerado pela fila e servido por CDN. Na POC os
// arquivos vão para uma pasta e o Go os serve em /publico; na produção, S3 + CloudFront. Servir não
// toca no banco: se o backend cair, o que já foi publicado continua no ar.
package publicacao

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"lisura/internal/arquivo"
	"lisura/internal/certame"
	"lisura/internal/fila"
)

const (
	TarefaCertame = "publicar_certame"
	TarefaCartao  = "publicar_cartao"
	TarefaRelacao = "publicar_relacao" // uma fatia da relação de inscritos
)

// AgruparRelacao é quanto a republicação de uma fatia da relação espera juntando mudanças: medido em
// 05/11, a 640× local a relação era refeita ~2 vezes por segundo; com 30 s, sai no máximo 2 vezes por
// minuto — e o CDN já a guarda por 60 s.
var AgruparRelacao = 30 * time.Second

// TamanhoFatia é quantos números de inscrição cabem numa fatia da relação. A relação inteira, refeita
// a cada mudança, crescia com o certame: medido em 06/11, o certame 5 (491 mil inscrições enviadas)
// levava até 3 s e 32,7 MB a cada republicação, duas por minuto no pico. A fatia é o número da
// inscrição dividido por isto; a mudança de uma inscrição refaz só a fatia dela. Variável para os
// testes usarem fatias pequenas.
var TamanhoFatia int64 = 10000

func chaveFatia(certameID, fatia int64) string {
	return strconv.FormatInt(certameID, 10) + ":" + strconv.FormatInt(fatia, 10)
}

type cargaFatia struct {
	Certame int64 `json:"certame"`
	Fatia   int64 `json:"fatia"`
}

// EnfileirarInscricao pede a republicação do cartão (já: o candidato está esperando por ele) e da
// fatia da relação em que a inscrição está (agrupada: só importa o estado mais recente). Chamado na
// transação que mudou o estado da inscrição.
func EnfileirarInscricao(ctx context.Context, tx pgx.Tx, inscricaoID, certameID int64) error {
	return enfileirarInscricao(ctx, tx, inscricaoID, certameID, fila.Normal)
}

// EnfileirarInscricaoLote é o mesmo para trabalho em massa (a homologação em lote): o cartão vai com
// prioridade de lote, para 100 mil republicações não passarem na frente do Pix e do cartão de quem
// está usando o sistema agora. A fatia da relação continua normal: é uma tarefa só por fatia.
func EnfileirarInscricaoLote(ctx context.Context, tx pgx.Tx, inscricaoID, certameID int64) error {
	return enfileirarInscricao(ctx, tx, inscricaoID, certameID, fila.Lote)
}

func enfileirarInscricao(ctx context.Context, tx pgx.Tx, inscricaoID, certameID int64, p fila.Prioridade) error {
	if err := fila.EnfileirarCom(ctx, tx, p, TarefaCartao, strconv.FormatInt(inscricaoID, 10), carga{ID: inscricaoID}); err != nil {
		return err
	}
	f := inscricaoID / TamanhoFatia
	if err := anotarFatia(ctx, tx, certameID, f); err != nil {
		return err
	}
	return fila.EnfileirarDepois(ctx, tx, TarefaRelacao, chaveFatia(certameID, f), cargaFatia{certameID, f}, AgruparRelacao)
}

// anotarFatia registra que a fatia tem inscrição enviada (o índice da relação lê daqui).
func anotarFatia(ctx context.Context, q interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}, certameID, fatia int64) error {
	_, err := q.Exec(ctx, `INSERT INTO relacao_fatia (certame_id, tamanho, fatia) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
		certameID, TamanhoFatia, fatia)
	return err
}

// EnfileirarCartoesLote pede a republicação dos cartões de muitas inscrições numa instrução só, com
// prioridade de lote (o ensalamento de um certame inteiro).
func EnfileirarCartoesLote(ctx context.Context, tx pgx.Tx, inscricoes []int64) error {
	chaves, cargas := make([]string, len(inscricoes)), make([]any, len(inscricoes))
	for i, id := range inscricoes {
		chaves[i], cargas[i] = strconv.FormatInt(id, 10), carga{ID: id}
	}
	return fila.EnfileirarMuitos(ctx, tx, fila.Lote, TarefaCartao, chaves, cargas)
}

// EnfileirarFatiasDe pede a republicação das fatias da relação em que estão estas inscrições do
// certame (cada fatia uma vez), agrupada como a de uma inscrição só: para quem muda muitas de uma vez
// fora de uma transação de inscrição (a expiração em massa).
func EnfileirarFatiasDe(ctx context.Context, pool *pgxpool.Pool, certameID int64, inscricoes []int64) error {
	vistas := map[int64]bool{}
	for _, id := range inscricoes {
		f := id / TamanhoFatia
		if vistas[f] {
			continue
		}
		vistas[f] = true
		if err := anotarFatia(ctx, pool, certameID, f); err != nil {
			return err
		}
		if err := fila.EnfileirarDepois(ctx, pool, TarefaRelacao, chaveFatia(certameID, f), cargaFatia{certameID, f}, AgruparRelacao); err != nil {
			return err
		}
	}
	return nil
}

func EnfileirarCertame(ctx context.Context, tx pgx.Tx, certameID int64) error {
	return fila.Enfileirar(ctx, tx, TarefaCertame, strconv.FormatInt(certameID, 10), carga{ID: certameID})
}

// RepublicarTodos enfileira a publicação de todos os certames (na subida do servidor: cobre os que
// existiam antes de a publicação existir e qualquer arquivo perdido). Antes, anota as fatias de toda
// inscrição enviada: as de antes da anotação existir, e as de um tamanho de fatia novo.
func RepublicarTodos(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `
		INSERT INTO relacao_fatia (certame_id, tamanho, fatia)
		SELECT DISTINCT certame_id, $1::bigint, id / $1 FROM inscricao WHERE estado <> 'RASCUNHO'
		ON CONFLICT DO NOTHING`, TamanhoFatia); err != nil {
		return err
	}
	rows, err := pool.Query(ctx, `SELECT id FROM certame`)
	if err != nil {
		return err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := fila.Enfileirar(ctx, pool, TarefaCertame, strconv.FormatInt(id, 10), carga{ID: id}); err != nil {
			return err
		}
	}
	return nil
}

type carga struct {
	ID int64 `json:"id"`
}

type Publicador struct {
	Pool  *pgxpool.Pool
	Dir   string // usado quando Destino é nil: a pasta servida em /publico
	Chave []byte // gera o código do endereço do cartão
	// Destino é para onde os arquivos vão; na nuvem, o bucket servido pelo CloudFront.
	Destino arquivo.Destino
}

func (p *Publicador) destino() arquivo.Destino {
	if p.Destino != nil {
		return p.Destino
	}
	return arquivo.Disco{Dir: p.Dir}
}

// Token é o código do endereço do cartão: derivado da inscrição com uma chave secreta, então não dá
// para adivinhar o cartão de outra pessoa a partir do número de inscrição.
func (p *Publicador) Token(inscricaoID int64) string {
	m := hmac.New(sha256.New, p.Chave)
	m.Write([]byte("cartao:" + strconv.FormatInt(inscricaoID, 10)))
	return hex.EncodeToString(m.Sum(nil))[:32]
}

func (p *Publicador) CaminhoCartao(inscricaoID int64) string {
	return "cartoes/" + p.Token(inscricaoID) + ".html"
}

// Existe diz se o arquivo já foi publicado. Na dúvida (destino fora do ar), diz que não: a tela
// pergunta de novo em instantes.
func (p *Publicador) Existe(ctx context.Context, caminho string) bool {
	ok, err := p.destino().Existe(ctx, caminho)
	return ok && err == nil
}

func (p *Publicador) Manipuladores() map[string]fila.Manipulador {
	return map[string]fila.Manipulador{
		TarefaCertame: p.manipular(p.PublicarCertame),
		TarefaCartao:  p.manipular(p.PublicarCartao),
		TarefaRelacao: func(ctx context.Context, b json.RawMessage) error {
			var c cargaFatia
			if err := json.Unmarshal(b, &c); err != nil {
				return err
			}
			return p.PublicarFatia(ctx, c.Certame, c.Fatia)
		},
	}
}

func (p *Publicador) manipular(f func(context.Context, int64) error) fila.Manipulador {
	return func(ctx context.Context, b json.RawMessage) error {
		var c carga
		if err := json.Unmarshal(b, &c); err != nil {
			return err
		}
		return f(ctx, c.ID)
	}
}

// Servir entrega os arquivos publicados. Não recebe o pool: por construção, não consulta o banco.
func Servir(dir string) http.Handler {
	arquivos := http.FileServer(arquivo.Pasta(dir)) // não lista pastas (ver arquivo.Pasta)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Cache curto: o CDN absorve o pico, e uma republicação aparece em até um minuto.
		w.Header().Set("Cache-Control", "public, max-age=60")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		arquivos.ServeHTTP(w, r)
	})
}

func (p *Publicador) gravar(ctx context.Context, caminho string, conteudo []byte) error {
	return p.destino().Put(ctx, caminho, conteudo)
}

// Gravar publica um arquivo pelo mesmo destino: para outros pacotes que geram conteúdo público
// (o resultado).
func (p *Publicador) Gravar(ctx context.Context, caminho string, conteudo []byte) error {
	return p.gravar(ctx, caminho, conteudo)
}

// ---- certame: dados públicos e relação de inscrições ----

type itemRelacao struct {
	Numero   int64  `json:"numero"`
	Cargo    string `json:"cargo"`
	Situacao string `json:"situacao"`
}

// PublicarCertame grava os dados públicos do certame e o índice da relação, e pede a republicação de
// todas as fatias (com prioridade de lote): o nome do cargo, que pode ter mudado, está em cada linha.
func (p *Publicador) PublicarCertame(ctx context.Context, id int64) error {
	c, err := certame.Buscar(ctx, p.Pool, id)
	if errors.Is(err, certame.ErrNaoEncontrado) {
		return nil // nada a publicar; não adianta repetir
	}
	if err != nil {
		return err
	}
	publico, err := json.Marshal(c)
	if err != nil {
		return err
	}
	if err := p.gravar(ctx, "certames/"+strconv.FormatInt(id, 10)+".json", publico); err != nil {
		return err
	}
	fatias, err := p.publicarIndice(ctx, id)
	if err != nil {
		return err
	}
	chaves, cargas := make([]string, len(fatias)), make([]any, len(fatias))
	for i, f := range fatias {
		chaves[i], cargas[i] = chaveFatia(id, f), cargaFatia{id, f}
	}
	return fila.EnfileirarMuitos(ctx, p.Pool, fila.Lote, TarefaRelacao, chaves, cargas)
}

// PublicarFatia grava uma fatia da relação de inscrições enviadas: só número, cargo e situação, nenhum
// dado pessoal. Lê só as inscrições do certame naquela faixa de números.
func (p *Publicador) PublicarFatia(ctx context.Context, certameID, fatia int64) error {
	rows, err := p.Pool.Query(ctx, `
		SELECT i.id, g.nome, i.estado FROM inscricao i JOIN cargo g ON g.id = i.cargo_id
		WHERE i.certame_id = $1 AND i.id >= $2 AND i.id < $3 AND i.estado <> 'RASCUNHO' ORDER BY i.id`,
		certameID, fatia*TamanhoFatia, (fatia+1)*TamanhoFatia)
	if err != nil {
		return err
	}
	relacao, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (itemRelacao, error) {
		var it itemRelacao
		var estado string
		err := r.Scan(&it.Numero, &it.Cargo, &estado)
		it.Situacao = situacao(estado)
		return it, err
	})
	if err != nil {
		return err
	}
	if relacao == nil {
		relacao = []itemRelacao{}
	}
	lista, err := json.Marshal(struct {
		Certame    int64         `json:"certame_id"`
		Fatia      int64         `json:"fatia"`
		Atualizada time.Time     `json:"atualizada_em"`
		Inscricoes []itemRelacao `json:"inscricoes"`
	}{certameID, fatia, time.Now().UTC(), relacao})
	if err != nil {
		return err
	}
	if err := p.gravar(ctx, CaminhoFatia(certameID, fatia), lista); err != nil {
		return err
	}
	// A fatia nova (a primeira inscrição numa faixa) entra no índice.
	_, err = p.publicarIndice(ctx, certameID)
	return err
}

// CaminhoFatia é onde a fatia da relação fica publicada.
func CaminhoFatia(certameID, fatia int64) string {
	return "certames/" + strconv.FormatInt(certameID, 10) + "/relacao/" + strconv.FormatInt(fatia, 10) + ".json"
}

// publicarIndice grava certames/{id}/relacao.json: as fatias que têm inscrição enviada do certame, e
// onde está cada uma. As fatias vêm de relacao_fatia, anotadas quando a primeira inscrição de cada uma
// pediu publicação (ver a migração 00019: deduzi-las das inscrições custava 271 ms no certame grande).
func (p *Publicador) publicarIndice(ctx context.Context, certameID int64) ([]int64, error) {
	rows, err := p.Pool.Query(ctx, `SELECT fatia FROM relacao_fatia WHERE certame_id = $1 AND tamanho = $2 ORDER BY fatia`,
		certameID, TamanhoFatia)
	if err != nil {
		return nil, err
	}
	fatias, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		return nil, err
	}
	type entrada struct {
		Fatia int64  `json:"fatia"`
		URL   string `json:"url"`
	}
	indice := struct {
		Certame      int64     `json:"certame_id"`
		Atualizada   time.Time `json:"atualizada_em"`
		TamanhoFatia int64     `json:"tamanho_fatia"`
		Fatias       []entrada `json:"fatias"`
	}{Certame: certameID, Atualizada: time.Now().UTC(), TamanhoFatia: TamanhoFatia, Fatias: []entrada{}}
	for _, f := range fatias {
		indice.Fatias = append(indice.Fatias, entrada{f, "/publico/" + CaminhoFatia(certameID, f)})
	}
	b, err := json.Marshal(indice)
	if err != nil {
		return nil, err
	}
	return fatias, p.gravar(ctx, "certames/"+strconv.FormatInt(certameID, 10)+"/relacao.json", b)
}

func situacao(estado string) string {
	return map[string]string{
		"ISENCAO_ATIVA": "Enviada com isenção", "AGUARDANDO_PAGAMENTO": "Aguardando pagamento",
		"PAGA": "Paga", "EXPIRADA": "Expirada", "CANCELADA": "Cancelada", "EM_ANALISE": "Em análise",
		"DEFERIDA": "Deferida", "INDEFERIDA": "Indeferida", "HOMOLOGADA": "Homologada",
	}[estado]
}

// ---- cartão de confirmação ----

//go:embed cartao.html
var modelos embed.FS

var cartao = template.Must(template.ParseFS(modelos, "cartao.html"))

type dadosCartao struct {
	Numero                          int64
	Nome, CPF, Certame, Banca       string
	Cargo, Cidade, Situacao, Codigo string
	Cotas, Necessidades             []string
	GeradoEm                        string
	Local, Endereco, Sala           string // vazios antes do ensalamento
	// Selo é o começo do resumo do último evento da trilha (migração 00010). Impresso, é a prova do
	// candidato de que a trilha não foi reescrita depois.
	Selo string
}

// Estados em que a inscrição vale e o cartão existe.
var comCartao = map[string]bool{"PAGA": true, "ISENCAO_ATIVA": true, "EM_ANALISE": true, "DEFERIDA": true, "HOMOLOGADA": true}

var rotuloNecessidade = map[string]string{
	"ledor": "Ledor", "transcritor": "Transcritor", "prova_ampliada": "Prova ampliada",
	"sala_acessivel": "Sala de fácil acesso", "tempo_adicional": "Tempo adicional", "lactante": "Espaço para amamentação",
}

// PublicarCartao gera a página do cartão para imprimir. Antes de a inscrição valer (paga ou isenta),
// não há cartão.
func (p *Publicador) PublicarCartao(ctx context.Context, inscricaoID int64) error {
	var d dadosCartao
	var cpf, estado string
	var necessidades []string
	err := p.Pool.QueryRow(ctx, `
		SELECT i.id, c.nome, c.cpf, ce.nome, b.nome, g.nome, coalesce(cp.nome || ' — ' || cp.uf, '—'), i.estado,
		       i.necessidades,
		       coalesce((SELECT array_agg(ct.nome ORDER BY ct.nome) FROM inscricao_cota ic
		                 JOIN cota ct ON ct.certame_id = ic.certame_id AND ct.codigo = ic.cota_codigo
		                 WHERE ic.inscricao_id = i.id), '{}'),
		       coalesce(l.nome, ''), coalesce(l.endereco, ''), coalesce(s.nome, '')
		FROM inscricao i
		JOIN candidato c ON c.id = i.candidato_id
		JOIN certame ce ON ce.id = i.certame_id
		JOIN banca b ON b.id = ce.banca_id
		JOIN cargo g ON g.id = i.cargo_id
		LEFT JOIN cidade_prova cp ON cp.id = i.cidade_prova_id
		LEFT JOIN alocacao a ON a.inscricao_id = i.id
		LEFT JOIN sala s ON s.id = a.sala_id
		LEFT JOIN local_prova l ON l.id = s.local_id
		WHERE i.id = $1`, inscricaoID,
	).Scan(&d.Numero, &d.Nome, &cpf, &d.Certame, &d.Banca, &d.Cargo, &d.Cidade, &estado, &necessidades, &d.Cotas,
		&d.Local, &d.Endereco, &d.Sala)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if !comCartao[estado] {
		return nil
	}
	d.CPF = "***." + cpf[3:6] + "." + cpf[6:9] + "-**" // o cartão pode ser impresso e deixado por aí
	d.Situacao = situacao(estado)
	d.Codigo = p.Token(inscricaoID)[:12]
	for _, n := range necessidades {
		d.Necessidades = append(d.Necessidades, rotuloNecessidade[n])
	}
	d.GeradoEm = time.Now().Format("02/01/2006 15:04")
	// O mesmo que inscricao.Selo (que não pode ser importado daqui: inscricao importa publicacao).
	if err := p.Pool.QueryRow(ctx, `
		SELECT left(encode(hash, 'hex'), 16) FROM inscricao_evento WHERE inscricao_id = $1 ORDER BY id DESC LIMIT 1`,
		inscricaoID).Scan(&d.Selo); err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := cartao.Execute(&buf, d); err != nil {
		return err
	}
	return p.gravar(ctx, p.CaminhoCartao(inscricaoID), buf.Bytes())
}
