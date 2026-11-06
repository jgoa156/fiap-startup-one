package degradacao

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// relogio é um relógio de mentira: a fila anda quando o teste manda.
type relogio struct{ t time.Time }

func (r *relogio) agora() time.Time      { return r.t }
func (r *relogio) andar(d time.Duration) { r.t = r.t.Add(d) }

func novo(t *testing.T) (*Controle, *relogio, http.Handler) {
	t.Helper()
	rel := &relogio{t: time.Date(2026, 11, 4, 9, 0, 0, 0, time.UTC)}
	c := &Controle{Chave: []byte("chave-de-teste"), Segredo: "op", Agora: rel.agora}
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	return c, rel, c.Middleware(ok)
}

// visitante guarda os cookies como um navegador.
type visitante struct{ cookies map[string]*http.Cookie }

func (v *visitante) pedir(h http.Handler, metodo, caminho string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(metodo, caminho, nil)
	for _, c := range v.cookies {
		r.AddCookie(c)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if v.cookies == nil {
		v.cookies = map[string]*http.Cookie{}
	}
	for _, c := range w.Result().Cookies() {
		if c.MaxAge < 0 {
			delete(v.cookies, c.Name)
		} else {
			v.cookies[c.Name] = c
		}
	}
	return w
}

func lerRecusa(t *testing.T, w *httptest.ResponseRecorder) recusa {
	t.Helper()
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("esperava 503, veio %d", w.Code)
	}
	var r recusa
	if err := json.Unmarshal(w.Body.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestNormalNaoMexeEmNada(t *testing.T) {
	_, _, h := novo(t)
	v := &visitante{}
	if w := v.pedir(h, "GET", "/eu"); w.Code != 200 || len(v.cookies) != 0 {
		t.Fatalf("nível 0 deveria passar sem cookie: %d %v", w.Code, v.cookies)
	}
}

func TestSalaDeEsperaAdmiteNaOrdemDeChegada(t *testing.T) {
	c, rel, h := novo(t)
	if err := c.Definir(SalaDeEspera, 1, 2); err != nil { // 2 entram já, depois 1 por segundo
		t.Fatal(err)
	}
	vs := make([]*visitante, 5)
	for i := range vs {
		vs[i] = &visitante{}
		w := vs[i].pedir(h, "GET", "/eu")
		if i < 2 {
			if w.Code != 200 {
				t.Fatalf("visitante %d estava na folga e não entrou: %d", i+1, w.Code)
			}
			continue
		}
		r := lerRecusa(t, w)
		if r.Fila == nil || r.Fila.Posicao != int64(i-1) || r.Fila.EsperaS != int64(i-1) {
			t.Fatalf("visitante %d: posição/espera erradas: %+v", i+1, r.Fila)
		}
		if w.Header().Get("Retry-After") == "" {
			t.Fatal("sem Retry-After")
		}
	}

	// Recarregar não perde a vez nem pega senha nova.
	if r := lerRecusa(t, vs[4].pedir(h, "GET", "/eu")); r.Fila.Posicao != 3 {
		t.Fatalf("recarregar mudou a posição: %+v", r.Fila)
	}
	if e := c.Estado(); e.Senhas != 5 {
		t.Fatalf("recarregar emitiu senha nova: %d senhas", e.Senhas)
	}

	// Um segundo depois, entra o 3º e só ele.
	rel.andar(time.Second)
	if w := vs[2].pedir(h, "GET", "/eu"); w.Code != 200 {
		t.Fatalf("3º deveria entrar após 1 s: %d", w.Code)
	}
	if r := lerRecusa(t, vs[3].pedir(h, "GET", "/eu")); r.Fila.Posicao != 1 {
		t.Fatalf("4º deveria ser o próximo: %+v", r.Fila)
	}

	// Quem entrou continua dentro (admissão renovada a cada uso) enquanto a fila espera.
	rel.andar(10 * time.Minute)
	if w := vs[0].pedir(h, "PUT", "/inscricoes/1/cargo"); w.Code != 200 {
		t.Fatalf("admitido foi barrado no meio do formulário: %d", w.Code)
	}
}

func TestAdmissaoVencePorFaltaDeUso(t *testing.T) {
	c, rel, h := novo(t)
	c.Definir(SalaDeEspera, 0, 1)
	v := &visitante{}
	if w := v.pedir(h, "GET", "/eu"); w.Code != 200 {
		t.Fatal("deveria entrar")
	}
	rel.andar(16 * time.Minute)
	if r := lerRecusa(t, v.pedir(h, "GET", "/eu")); r.Fila == nil {
		t.Fatal("admissão parada há 16 min deveria ter vencido e voltado para a fila")
	}
}

func TestSenhaFalsificadaNaoFuraFila(t *testing.T) {
	c, _, h := novo(t)
	c.Definir(SalaDeEspera, 0, 1)
	(&visitante{}).pedir(h, "GET", "/eu") // ocupa a folga

	// Alguém troca a própria senha por "senha.1" assinada com outra chave, e forja uma admissão.
	outro := &Controle{Chave: []byte("chave-errada")}
	for _, ck := range []*http.Cookie{
		{Name: cookieSenha, Value: outro.assinar("senha", 1)},
		{Name: cookieAdmissao, Value: outro.assinar("adm", time.Now().Add(time.Hour).Unix())},
		{Name: cookieAdmissao, Value: c.assinar("senha", 1)}, // assinatura certa, tipo errado
		{Name: cookieSenha, Value: "c2VuaGEuMQ.lixo"},
	} {
		v := &visitante{cookies: map[string]*http.Cookie{ck.Name: ck}}
		if r := lerRecusa(t, v.pedir(h, "GET", "/eu")); r.Fila == nil || r.Fila.Posicao < 1 {
			t.Fatalf("cookie forjado %s=%s passou: %+v", ck.Name, ck.Value, r.Fila)
		}
	}
}

func TestSalaReabertaNaoFazQuemEsperavaPerderAVez(t *testing.T) {
	c, _, h := novo(t)
	c.Definir(SalaDeEspera, 0, 0)
	antigo := &visitante{}
	lerRecusa(t, antigo.pedir(h, "GET", "/eu")) // senha 1, esperando

	c.Definir(Normal, 0, 0)
	c.Definir(SalaDeEspera, 0, 0) // reabriu: quem já tinha senha entra antes dos novos
	if w := antigo.pedir(h, "GET", "/eu"); w.Code != 200 {
		t.Fatalf("senha antiga deveria entrar na reabertura: %d", w.Code)
	}
	if r := lerRecusa(t, (&visitante{}).pedir(h, "GET", "/eu")); r.Fila.Posicao != 1 {
		t.Fatalf("novo visitante deveria ser o 1º da nova fila: %+v", r.Fila)
	}
}

func TestAvisoDoPrestadorNuncaEsperaNemCai(t *testing.T) {
	c, _, h := novo(t)
	for _, n := range []int{SalaDeEspera, SoEssencial, SoEstatico} {
		c.Definir(n, 0, 0)
		for _, caminho := range []string{"/psp/aviso", "/saude"} {
			if w := (&visitante{}).pedir(h, "POST", caminho); w.Code != 200 {
				t.Fatalf("nível %d bloqueou %s: %d", n, caminho, w.Code)
			}
		}
	}
}

func TestSoEssencialDesligaOQueNaoECritico(t *testing.T) {
	c, _, h := novo(t)
	c.Definir(SoEssencial, 0, 1000)
	casos := []struct {
		metodo, caminho string
		desligada       bool
	}{
		{"GET", "/certames", true},
		{"GET", "/certames/3/isencao", true},
		{"POST", "/certames/3/isencao", true},
		{"GET", "/inscricoes/9/trilha", true},
		{"GET", "/conta/documentos", true},
		{"GET", "/eu", false},
		{"GET", "/certames/3", false},
		{"POST", "/certames/3/inscricoes", false},
		{"PUT", "/inscricoes/9/cargo", false},
		{"POST", "/inscricoes/9/finalizar", false},
		{"POST", "/inscricoes/9/pagamento", false},
		{"GET", "/inscricoes/9/cartao", false},
		{"POST", "/conta/documentos", false}, // o laudo de PcD pode ser exigido para concluir
	}
	v := &visitante{}
	for _, k := range casos {
		w := v.pedir(h, k.metodo, k.caminho)
		if k.desligada {
			if r := lerRecusa(t, w); r.Nivel != SoEssencial || !strings.Contains(r.Erro, "temporariamente") {
				t.Fatalf("%s %s: recusa errada %+v", k.metodo, k.caminho, r)
			}
		} else if w.Code != 200 {
			t.Fatalf("%s %s é crítica e foi barrada: %d", k.metodo, k.caminho, w.Code)
		}
	}
}

func TestSoEstaticoRecusaTudoNaAPI(t *testing.T) {
	c, _, h := novo(t)
	c.Definir(SoEstatico, 0, 1000)
	for _, caminho := range []string{"/eu", "/certames/3", "/inscricoes/9/cartao"} {
		if r := lerRecusa(t, (&visitante{}).pedir(h, "GET", caminho)); r.Nivel != SoEstatico {
			t.Fatalf("%s: %+v", caminho, r)
		}
	}
}

func TestOperacaoExigeSegredo(t *testing.T) {
	c, _, _ := novo(t)
	op := c.Operacao()
	pedir := func(auth, corpo string) int {
		r := httptest.NewRequest("POST", "/operacao", strings.NewReader(corpo))
		if auth != "" {
			r.Header.Set("Authorization", auth)
		}
		w := httptest.NewRecorder()
		op.ServeHTTP(w, r)
		return w.Code
	}
	if pedir("", `{"nivel":3}`) != 401 || pedir("Operacao errado", `{"nivel":3}`) != 401 || pedir("Bearer op", `{"nivel":3}`) != 401 {
		t.Fatal("mudou o nível sem o segredo")
	}
	if c.Estado().Nivel != Normal {
		t.Fatal("nível mudou")
	}
	if pedir("Operacao op", `{"nivel":7}`) != 422 {
		t.Fatal("aceitou nível inexistente")
	}
	if pedir("Operacao op", `{"nivel":1,"taxa_por_segundo":50,"folga":200}`) != 200 || c.Estado().Nivel != SalaDeEspera {
		t.Fatal("segredo certo não mudou o nível")
	}
	sem := &Controle{} // sem segredo configurado, ninguém opera
	r := httptest.NewRequest("GET", "/operacao", nil)
	r.Header.Set("Authorization", "Operacao ")
	w := httptest.NewRecorder()
	sem.Operacao().ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatal("segredo vazio abriu a operação")
	}
}
