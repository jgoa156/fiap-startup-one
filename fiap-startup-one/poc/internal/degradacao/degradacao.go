// Package degradacao é a degradação graciosa da arquitetura alvo (§4.5): quando falta capacidade,
// a resposta é uma fila honesta — posição e tempo estimado — e não um erro 502. Na produção a sala
// de espera fica na borda (Cloudflare Waiting Room); na POC, num middleware na frente da API, com
// o mesmo comportamento visto pelo candidato.
//
// Níveis (a POC agrupa os 5 da arquitetura alvo em 3 que dá para demonstrar):
//
//	0 — normal.
//	1 — sala de espera: entra quem tem admissão; os demais recebem senha, posição e estimativa.
//	2 — só o essencial: funções fora do caminho crítico (lista de certames, isenção, histórico,
//	    carteira de documentos) respondem "temporariamente desligada". Alvo: níveis 2 e 3.
//	3 — só estático: a API recusa tudo, menos o aviso do prestador (pagamento que entrou não pode
//	    se perder) e a saúde. Cartão, relação e dados do certame seguem em /publico. Alvo: nível 4.
package degradacao

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	Normal = iota
	SalaDeEspera
	SoEssencial
	SoEstatico
)

const (
	cookieSenha    = "lisura_fila"
	cookieAdmissao = "lisura_admitido"
)

// Controle guarda o nível e a sala de espera. O estado mora no Armazem (Postgres, com várias
// instâncias) e cada instância trabalha com uma cópia lida a cada segundo (Acompanhar): o pedido
// comum não toca no banco; só a emissão de uma senha nova toca.
type Controle struct {
	Chave    []byte        // assina senha e admissão: o candidato não fabrica uma senha menor
	Segredo  string        // autoriza a rota de operação
	Validade time.Duration // quanto dura uma admissão sem uso (renova a cada pedido); padrão 15 min
	Agora    func() time.Time
	Armazem  Armazem // padrão: em memória (um processo só)

	iniciar   sync.Once
	mu        sync.Mutex
	cfg       Config
	admitidos atomic.Int64 // admissões efetivas nesta instância (para o painel de operação)
}

// Estado é o que a operação vê e o que o front consulta.
type Estado struct {
	Nivel     int     `json:"nivel"`
	Taxa      float64 `json:"taxa_por_segundo"`
	Senhas    int64   `json:"senhas_emitidas"`
	Liberadas int64   `json:"senhas_liberadas"`
	Admitidos int64   `json:"admissoes_nesta_instancia"`
}

func (c *Controle) armazem() Armazem {
	c.iniciar.Do(func() {
		if c.Armazem == nil {
			c.Armazem = &Memoria{}
		}
	})
	return c.Armazem
}

// Atualizar relê o estado compartilhado.
func (c *Controle) Atualizar(ctx context.Context) error {
	cfg, err := c.armazem().Ler(ctx)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.cfg = cfg
	c.mu.Unlock()
	return nil
}

// Acompanhar relê o estado a cada intervalo, até o contexto acabar. Um nível trocado por outra
// instância vale aqui em até um intervalo.
func (c *Controle) Acompanhar(ctx context.Context, intervalo time.Duration, falha func(error)) {
	t := time.NewTicker(intervalo)
	defer t.Stop()
	for {
		if err := c.Atualizar(ctx); err != nil && falha != nil {
			falha(err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

func (c *Controle) config() Config {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cfg
}

func (c *Controle) agora() time.Time {
	if c.Agora != nil {
		return c.Agora()
	}
	return time.Now()
}

func (c *Controle) validade() time.Duration {
	if c.Validade > 0 {
		return c.Validade
	}
	return 15 * time.Minute
}

// Definir muda o nível. Ao abrir a sala de espera, a contagem recomeça: as primeiras `folga` senhas
// entram já e depois entram `taxa` por segundo.
func (c *Controle) Definir(nivel int, taxa float64, folga int64) error {
	if nivel < Normal || nivel > SoEstatico {
		return fmt.Errorf("nível %d inexistente", nivel)
	}
	if taxa < 0 || folga < 0 {
		return fmt.Errorf("taxa e folga não podem ser negativas")
	}
	cfg, err := c.armazem().Definir(context.Background(), nivel, taxa, folga, c.agora())
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.cfg = cfg
	c.mu.Unlock()
	return nil
}

func (c *Controle) Estado() Estado {
	cfg := c.config()
	senhas, err := c.armazem().Emitidas(context.Background())
	if err != nil {
		senhas = -1
	}
	return Estado{Nivel: cfg.Nivel, Taxa: cfg.Taxa, Senhas: senhas, Liberadas: min(c.liberadas(cfg), senhas), Admitidos: c.admitidos.Load()}
}

// liberadas é até que senha a fila já andou.
func (c *Controle) liberadas(cfg Config) int64 {
	if cfg.Nivel < SalaDeEspera {
		return math.MaxInt64
	}
	return cfg.Base + cfg.Folga + int64(c.agora().Sub(cfg.AbriuEm).Seconds()*cfg.Taxa)
}

// ---- classificação das rotas (caminhos já sem o prefixo /api) ----

// sempre passam: o aviso do prestador (dinheiro que entrou) e a saúde.
func isento(r *http.Request) bool {
	return r.URL.Path == "/psp/aviso" || r.URL.Path == "/saude"
}

// naoCritica é o que sai do ar no nível 2: tudo fora de "entrar, inscrever, pagar, ver a inscrição".
func naoCritica(r *http.Request) bool {
	p := r.URL.Path
	switch {
	case r.Method == http.MethodGet && p == "/certames":
		return true // a lista; a página de cada certame também existe estática
	case strings.HasSuffix(p, "/isencao"):
		return true // o pedido de isenção tem prazo próprio, antes do pico da inscrição
	case strings.HasSuffix(p, "/trilha"):
		return true
	case r.Method == http.MethodGet && p == "/conta/documentos":
		return true
	}
	return false
}

// ---- middleware ----

type recusa struct {
	Erro  string   `json:"erro"`
	Nivel int      `json:"degradado"`
	Fila  *posFila `json:"fila,omitempty"`
}

type posFila struct {
	Posicao int64 `json:"posicao"`
	EsperaS int64 `json:"espera_segundos"`
}

func (c *Controle) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isento(r) {
			next.ServeHTTP(w, r)
			return
		}
		nivel := c.config().Nivel

		switch {
		case nivel >= SoEstatico:
			c.recusar(w, nivel, "O sistema está em modo somente consulta. Cartão de confirmação, relação e dados do certame continuam disponíveis.", nil)
			return
		case nivel >= SoEssencial && naoCritica(r):
			c.recusar(w, nivel, "Esta função está temporariamente desligada para dar prioridade às inscrições e aos pagamentos.", nil)
			return
		}
		if nivel >= SalaDeEspera && !c.admitir(w, r) {
			return
		}
		next.ServeHTTP(w, r)
	})
}

// admitir deixa passar quem tem admissão válida; quem não tem recebe (ou mantém) uma senha. Recarregar
// a página não perde a vez: a senha vai num cookie assinado.
func (c *Controle) admitir(w http.ResponseWriter, r *http.Request) bool {
	agora := c.agora()
	if ck, err := r.Cookie(cookieAdmissao); err == nil {
		if exp, ok := c.lerAssinado(ck.Value, "adm"); ok && agora.Unix() < exp {
			c.darAdmissao(w, r, agora) // renova: a admissão vence por falta de uso, não no meio do formulário
			return true
		}
	}

	senha := int64(-1)
	if ck, err := r.Cookie(cookieSenha); err == nil {
		if s, ok := c.lerAssinado(ck.Value, "senha"); ok {
			senha = s
		}
	}
	if senha < 0 {
		var err error
		if senha, err = c.armazem().Emitir(r.Context()); err != nil {
			// Sem como numerar a fila, não dá para prometer posição: pede para tentar de novo.
			w.Header().Set("Retry-After", "10")
			c.recusar(w, c.config().Nivel, "Muita gente acessando agora. Tente de novo em alguns segundos.", nil)
			return false
		}
		http.SetCookie(w, &http.Cookie{Name: cookieSenha, Value: c.assinar("senha", senha), Path: "/",
			MaxAge: 6 * 3600, HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: Seguro(r)})
	}

	cfg := c.config()
	liberadas := c.liberadas(cfg)
	taxa, nivel := cfg.Taxa, cfg.Nivel
	if senha <= liberadas {
		c.admitidos.Add(1)
		c.darAdmissao(w, r, agora)
		http.SetCookie(w, &http.Cookie{Name: cookieSenha, Value: "", Path: "/", MaxAge: -1})
		return true
	}

	posicao := senha - liberadas
	espera := int64(math.MaxInt32)
	if taxa > 0 {
		espera = int64(math.Ceil(float64(posicao) / taxa))
	}
	w.Header().Set("Retry-After", strconv.FormatInt(min(max(espera, 5), 60), 10))
	c.recusar(w, nivel, "Muita gente acessando agora. Você está na fila — não feche esta página.",
		&posFila{Posicao: posicao, EsperaS: espera})
	return false
}

func (c *Controle) darAdmissao(w http.ResponseWriter, r *http.Request, agora time.Time) {
	http.SetCookie(w, &http.Cookie{Name: cookieAdmissao, Value: c.assinar("adm", agora.Add(c.validade()).Unix()),
		Path: "/", MaxAge: int(c.validade().Seconds()), HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: Seguro(r)})
}

// Seguro diz se o navegador chegou por HTTPS. Na nuvem o TLS termina no CloudFront/balanceador, que
// avisa pelo X-Forwarded-Proto; sem isso o cookie sairia sem a marca Secure.
func Seguro(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" || r.Header.Get("CloudFront-Forwarded-Proto") == "https"
}

func (c *Controle) recusar(w http.ResponseWriter, nivel int, msg string, f *posFila) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusServiceUnavailable)
	json.NewEncoder(w).Encode(recusa{Erro: msg, Nivel: nivel, Fila: f})
}

// ---- assinatura das senhas ----

var b64 = base64.RawURLEncoding

func (c *Controle) assinar(tipo string, n int64) string {
	payload := tipo + "." + strconv.FormatInt(n, 10)
	m := hmac.New(sha256.New, c.Chave)
	m.Write([]byte(payload))
	return b64.EncodeToString([]byte(payload)) + "." + b64.EncodeToString(m.Sum(nil))
}

func (c *Controle) lerAssinado(v, tipo string) (int64, bool) {
	p, sig, ok := strings.Cut(v, ".")
	if !ok {
		return 0, false
	}
	payload, err1 := b64.DecodeString(p)
	assinatura, err2 := b64.DecodeString(sig)
	if err1 != nil || err2 != nil {
		return 0, false
	}
	m := hmac.New(sha256.New, c.Chave)
	m.Write(payload)
	if !hmac.Equal(assinatura, m.Sum(nil)) {
		return 0, false
	}
	t, n, ok := strings.Cut(string(payload), ".")
	if !ok || t != tipo {
		return 0, false
	}
	x, err := strconv.ParseInt(n, 10, 64)
	return x, err == nil
}

// ---- rota de operação ----

// Operacao expõe GET (estado) e POST {nivel, taxa_por_segundo, folga} para quem tem o segredo.
// Na produção o nível sobe sozinho por métrica; na POC é acionado à mão (e pelo ensaio de carga).
func (c *Controle) Operacao() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Operacao ")
		if c.Segredo == "" || !ok || !hmac.Equal([]byte(got), []byte(c.Segredo)) {
			http.Error(w, "não autorizado", http.StatusUnauthorized)
			return
		}
		if r.Method == http.MethodPost {
			var in struct {
				Nivel int     `json:"nivel"`
				Taxa  float64 `json:"taxa_por_segundo"`
				Folga int64   `json:"folga"`
			}
			if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&in); err != nil {
				http.Error(w, "JSON inválido", http.StatusBadRequest)
				return
			}
			if err := c.Definir(in.Nivel, in.Taxa, in.Folga); err != nil {
				http.Error(w, err.Error(), http.StatusUnprocessableEntity)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(c.Estado())
	})
}

// Situacao é pública e nunca entra na fila: o front pergunta para saber em que nível está.
func (c *Controle) Situacao() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		e := c.Estado()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		json.NewEncoder(w).Encode(map[string]int{"nivel": e.Nivel})
	})
}
