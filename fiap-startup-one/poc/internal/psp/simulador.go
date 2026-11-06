package psp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"net/http"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Simulador imita um prestador de pagamento, inclusive nos defeitos que um prestador real tem e
// que a POC precisa aguentar. Não move dinheiro nenhum.
type Simulador struct {
	WebhookURL string
	Segredo    []byte

	// Caos: o que um prestador real faz de vez em quando.
	AvisosPorPagamento int           // quantas vezes manda o mesmo aviso (≥ 1); 1 = sem repetição
	ProbPerderAviso    float64       // chance de o aviso nunca chegar; só a conciliação recupera
	AtrasoAviso        time.Duration // espera antes de avisar
	AtrasoCriacao      time.Duration // demora para responder à criação de cobrança
	FalharCriacoes     atomic.Int32  // as próximas N criações dão erro 503 depois de criar (resposta perdida)

	FalharDevolucoes atomic.Int32 // as próximas N devoluções dão 503 depois de feitas (resposta perdida)

	CobrancasCriadas atomic.Int32 // quantas cobranças distintas existem; o teste de duplicidade lê isto
	DevolucoesFeitas atomic.Int32 // quantas devoluções distintas saíram; o teste de "devolver duas vezes" lê isto

	// Diario, se definido, recebe uma linha JSON por mudança (cobrança criada, pagamento recebido). É o
	// que Restaurar relê para o simulador sobreviver a um reinício como um prestador de verdade
	// sobrevive: sem ele, reiniciar apagava as cobranças (o candidato não conseguia mais pagar) e o
	// extrato (a conciliação acusava divergências falsas).
	Diario io.Writer

	mu         sync.Mutex
	cobrancas  map[string]*cobrancaSim
	porChave   map[string]string // chave de idempotência -> id da cobrança
	pagamentos []Pagamento
	pagIdx     map[string]int // id do pagamento -> posição em pagamentos
	devolucoes map[string]*devolucaoReg
	devolvido  map[string]int // id do pagamento -> centavos já devolvidos
	// O DICT, o diretório de chaves Pix do Banco Central, em miniatura: e-mail, telefone e chave
	// aleatória só existem se registrados; um CPF é chave de si mesmo, salvo se marcado sem chave.
	chaves   map[string]string // chave -> CPF do titular
	semChave map[string]bool   // CPFs que não são chave Pix
	seq      int
	prefixo  string // identificadores únicos entre execuções, como os de um prestador real
	avisos   sync.WaitGroup
}

type cobrancaSim struct {
	Cobranca
	NovaCobranca
	pagamentos int
}

// registro é uma linha do diário.
type registro struct {
	Prefixo   string        `json:"prefixo,omitempty"`
	Seq       int           `json:"seq,omitempty"`
	Chave     string        `json:"chave,omitempty"`
	Cobranca  *cobrancaReg  `json:"cobranca,omitempty"`
	Pagamento *Pagamento    `json:"pagamento,omitempty"`
	Adulterar *Pagamento    `json:"adulterar,omitempty"`
	Devolucao *devolucaoReg `json:"devolucao,omitempty"`
}

type cobrancaReg struct {
	Cobranca
	NovaCobranca
}

// devolucaoReg guarda o pedido junto com a resposta: o mesmo id com outro pedido é recusado.
type devolucaoReg struct {
	Devolucao
	Pedido NovaDevolucao `json:"pedido"`
}

// anotar escreve no diário; chamado com s.mu travado, na ordem em que as mudanças aconteceram.
func (s *Simulador) anotar(r registro) {
	if s.Diario == nil {
		return
	}
	b, _ := json.Marshal(r)
	if _, err := s.Diario.Write(append(b, '\n')); err != nil {
		log.Printf("psp simulado: diário: %v", err)
	}
}

// iniciar prepara o estado vazio; chamado com s.mu travado.
func (s *Simulador) iniciar() {
	if s.cobrancas != nil {
		return
	}
	s.cobrancas = map[string]*cobrancaSim{}
	s.porChave = map[string]string{}
	s.pagIdx, s.devolucoes, s.devolvido = map[string]int{}, map[string]*devolucaoReg{}, map[string]int{}
	// Sem prefixo, reiniciar o simulador sem diário repetiria cob_000001 e colidiria com cobranças antigas.
	s.prefixo = fmt.Sprintf("%06x", rand.Uint32()&0xffffff)
	s.anotar(registro{Prefixo: s.prefixo})
}

// Restaurar relê um diário gravado por este simulador (chamar antes de Handler). Os avisos que
// estavam a caminho quando ele parou não são reenviados: um prestador real também os perderia, e a
// conciliação recupera pelo extrato.
func (s *Simulador) Restaurar(r io.Reader) (cobrancas, pagamentos int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dec := json.NewDecoder(r)
	for {
		var reg registro
		if err := dec.Decode(&reg); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			// Uma linha final pela metade (o processo morreu no meio da escrita) é o fim do diário.
			if errors.Is(err, io.ErrUnexpectedEOF) {
				break
			}
			return cobrancas, pagamentos, fmt.Errorf("diário do PSP: %w", err)
		}
		switch {
		case reg.Prefixo != "":
			s.cobrancas, s.porChave, s.prefixo = map[string]*cobrancaSim{}, map[string]string{}, reg.Prefixo
			s.devolucoes, s.devolvido = map[string]*devolucaoReg{}, map[string]int{}
		case s.cobrancas == nil:
			return cobrancas, pagamentos, errors.New("diário do PSP sem prefixo na primeira linha")
		case reg.Cobranca != nil:
			s.cobrancas[reg.Cobranca.ID] = &cobrancaSim{Cobranca: reg.Cobranca.Cobranca, NovaCobranca: reg.Cobranca.NovaCobranca}
			s.porChave[reg.Chave] = reg.Cobranca.ID
			s.CobrancasCriadas.Add(1)
			cobrancas++
		case reg.Pagamento != nil:
			s.pagamentos = append(s.pagamentos, *reg.Pagamento)
			if c := s.cobrancas[reg.Pagamento.CobrancaID]; c != nil {
				c.pagamentos++
			}
			pagamentos++
		case reg.Adulterar != nil:
			s.adulterar(reg.Adulterar.ID, reg.Adulterar.ValorCentavos)
		case reg.Devolucao != nil:
			reg.Devolucao.Pedido.ID = reg.Devolucao.Referencia // o id vai na URL, não no corpo gravado
			s.devolucoes[reg.Devolucao.Referencia] = reg.Devolucao
			s.devolvido[reg.Devolucao.PagamentoID] += reg.Devolucao.ValorCentavos
			s.DevolucoesFeitas.Add(1)
		}
		s.seq = max(s.seq, reg.Seq)
	}
	// O extrato depende da ordem por hora; um diário gravado antes de pagar() garanti-la pode ter
	// dois pagamentos fora de ordem.
	slices.SortStableFunc(s.pagamentos, func(a, b Pagamento) int { return a.PagoEm.Compare(b.PagoEm) })
	s.pagIdx = make(map[string]int, len(s.pagamentos))
	for i, p := range s.pagamentos {
		s.pagIdx[p.ID] = i
	}
	return cobrancas, pagamentos, nil
}

func (s *Simulador) Handler() http.Handler {
	s.mu.Lock()
	s.iniciar()
	s.mu.Unlock()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /cobrancas", s.criar)
	mux.HandleFunc("POST /cobrancas/{id}/pagar", s.pagar)
	mux.HandleFunc("GET /extrato", s.extrato)
	mux.HandleFunc("PUT /devolucoes/{id}", s.devolver)
	mux.HandleFunc("GET /devolucoes", s.extratoDevolucoes)
	// O DICT de teste, como nos sandboxes de prestadores reais: registrar uma chave e tirar a chave
	// CPF de alguém, para percorrer a devolução por outra chave sem recompilar o simulador.
	mux.HandleFunc("PUT /sandbox/dict/{chave}", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			CPFTitular string `json:"cpf_titular"`
		}
		if json.NewDecoder(r.Body).Decode(&in) != nil || len(in.CPFTitular) != 11 {
			http.Error(w, `corpo: {"cpf_titular": "11 dígitos"}`, http.StatusBadRequest)
			return
		}
		s.RegistrarChave(r.PathValue("chave"), in.CPFTitular)
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("DELETE /sandbox/dict/cpf/{cpf}", func(w http.ResponseWriter, r *http.Request) {
		s.SemChaveCPF(r.PathValue("cpf"))
		w.WriteHeader(http.StatusNoContent)
	})
	return mux
}

// extratoDevolucoes lista as devoluções feitas no período, por hora: o lado "saiu" do extrato.
func (s *Simulador) extratoDevolucoes(w http.ResponseWriter, r *http.Request) {
	desde, _ := time.Parse(time.RFC3339Nano, r.URL.Query().Get("desde"))
	ate, err := time.Parse(time.RFC3339Nano, r.URL.Query().Get("ate"))
	if err != nil {
		ate = time.Now()
	}
	s.mu.Lock()
	out := []Devolucao{}
	for _, d := range s.devolucoes {
		if !d.DevolvidaEm.Before(desde) && !d.DevolvidaEm.After(ate) {
			out = append(out, d.Devolucao)
		}
	}
	s.mu.Unlock()
	slices.SortFunc(out, func(a, b Devolucao) int { return a.DevolvidaEm.Compare(b.DevolvidaEm) })
	escrever(w, http.StatusOK, out)
}

// RegistrarChave registra no DICT simulado uma chave Pix (e-mail, telefone, aleatória) e o CPF do
// titular. Pode ser chamada com o simulador no ar.
func (s *Simulador) RegistrarChave(chave, cpfTitular string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.chaves == nil {
		s.chaves = map[string]string{}
	}
	s.chaves[chave] = cpfTitular
}

// SemChaveCPF marca um CPF que não é chave Pix de ninguém. Pode ser chamada com o simulador no ar.
func (s *Simulador) SemChaveCPF(cpf string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.semChave == nil {
		s.semChave = map[string]bool{}
	}
	s.semChave[cpf] = true
}

// titular é o CPF dono da chave no DICT simulado, ou "" se a chave não existe; chamado com s.mu travado.
func (s *Simulador) titular(chave string) string {
	if len(chave) == 11 && strings.Trim(chave, "0123456789") == "" {
		if s.semChave[chave] {
			return ""
		}
		return chave
	}
	return s.chaves[chave]
}

// devolver devolve (parte de) um pagamento, com as regras de um prestador real: o mesmo id devolve a
// mesma devolução; o total devolvido de um pagamento nunca passa do que foi pago; Pix volta para quem
// pagou; boleto só volta para uma chave Pix cujo titular é o CPF informado.
func (s *Simulador) devolver(w http.ResponseWriter, r *http.Request) {
	var d NovaDevolucao
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil || d.ValorCentavos <= 0 {
		http.Error(w, "devolução inválida", http.StatusBadRequest)
		return
	}
	d.ID = r.PathValue("id")
	recusar := func(motivo string) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{"codigo": motivo})
	}

	s.mu.Lock()
	if feita, ok := s.devolucoes[d.ID]; ok {
		s.mu.Unlock()
		if feita.Pedido != d {
			recusar(RecusaIDReutilizado)
			return
		}
		escrever(w, http.StatusOK, feita.Devolucao)
		return
	}
	i, ok := s.pagIdx[d.PagamentoID]
	if !ok {
		s.mu.Unlock()
		recusar(RecusaPagamento)
		return
	}
	p := s.pagamentos[i]
	motivo := ""
	switch c := s.cobrancas[p.CobrancaID]; {
	case s.devolvido[p.ID]+d.ValorCentavos > p.ValorCentavos:
		motivo = RecusaValorAcimaDoPago
	case d.ChavePix == "" && (c == nil || c.Metodo != "pix"):
		motivo = RecusaChaveObrigatoria
	case d.ChavePix != "" && s.titular(d.ChavePix) == "":
		motivo = RecusaChaveInexistente
	case d.ChavePix != "" && s.titular(d.ChavePix) != d.CPFTitular:
		motivo = RecusaTitularDiferente
	}
	if motivo != "" {
		s.mu.Unlock()
		recusar(motivo)
		return
	}
	s.seq++
	feita := &devolucaoReg{Pedido: d, Devolucao: Devolucao{ID: fmt.Sprintf("dev_%s_%06d", s.prefixo, s.seq),
		Referencia: d.ID, PagamentoID: p.ID, ValorCentavos: d.ValorCentavos, DevolvidaEm: time.Now().UTC()}}
	s.devolucoes[d.ID] = feita
	s.devolvido[p.ID] += d.ValorCentavos
	s.DevolucoesFeitas.Add(1)
	s.anotar(registro{Seq: s.seq, Devolucao: feita})
	s.mu.Unlock()

	// Feita, mas a resposta "se perde": quem pediu tem de repetir com o mesmo id.
	if s.FalharDevolucoes.Load() > 0 && s.FalharDevolucoes.Add(-1) >= 0 {
		http.Error(w, "instabilidade simulada", http.StatusServiceUnavailable)
		return
	}
	escrever(w, http.StatusCreated, feita.Devolucao)
}

func (s *Simulador) criar(w http.ResponseWriter, r *http.Request) {
	var n NovaCobranca
	if err := json.NewDecoder(r.Body).Decode(&n); err != nil || n.ValorCentavos <= 0 {
		http.Error(w, "cobrança inválida", http.StatusBadRequest)
		return
	}
	chave := r.Header.Get("Idempotency-Key")
	if chave == "" {
		http.Error(w, "Idempotency-Key obrigatório", http.StatusBadRequest)
		return
	}
	time.Sleep(s.AtrasoCriacao)

	s.mu.Lock()
	id, existe := s.porChave[chave]
	if !existe {
		s.seq++
		id = fmt.Sprintf("cob_%s_%06d", s.prefixo, s.seq)
		c := &cobrancaSim{Cobranca: Cobranca{ID: id}, NovaCobranca: n}
		if n.Metodo == "pix" {
			c.PixCopiaCola = fmt.Sprintf("00020126580014br.gov.bcb.pix0136lisura-sandbox-%s5204000053039865406%d.%02d5802BR6304SIMU",
				id, n.ValorCentavos/100, n.ValorCentavos%100)
		} else {
			c.LinhaDigitavel = fmt.Sprintf("00190.00009 0%05d.%06d 00000.000000 1 %010d", s.seq%100000, s.seq, n.ValorCentavos)
		}
		s.cobrancas[id] = c
		s.porChave[chave] = id
		s.CobrancasCriadas.Add(1)
		s.anotar(registro{Seq: s.seq, Chave: chave, Cobranca: &cobrancaReg{c.Cobranca, c.NovaCobranca}})
	}
	c := s.cobrancas[id].Cobranca
	s.mu.Unlock()

	// A cobrança foi criada, mas a resposta "se perde": o cliente precisa repetir com a mesma chave.
	if s.FalharCriacoes.Load() > 0 && s.FalharCriacoes.Add(-1) >= 0 {
		http.Error(w, "instabilidade simulada", http.StatusServiceUnavailable)
		return
	}
	status := http.StatusCreated
	if existe {
		status = http.StatusOK
	}
	escrever(w, status, c)
}

// pagar é o candidato pagando. O boleto aceita ser pago mais de uma vez, como o boleto impresso
// duas vezes na vida real. O Pix, não: a cobrança de Pix com identificador (a do QR dinâmico) passa a
// "concluída" no primeiro pagamento, e o banco de quem paga recusa o segundo. Antes o simulador aceitava,
// e o candidato que recarregava a tela do banco pagava duas vezes: medido em 05/11, a 1.280× local.
func (s *Simulador) pagar(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	c, ok := s.cobrancas[r.PathValue("id")]
	if !ok {
		s.mu.Unlock()
		http.Error(w, "cobrança inexistente", http.StatusNotFound)
		return
	}
	if c.Metodo == "pix" && c.pagamentos > 0 {
		s.mu.Unlock()
		http.Error(w, "Pix já pago", http.StatusConflict)
		return
	}
	if c.Metodo == "pix" && time.Now().After(c.VenceEm) {
		s.mu.Unlock()
		http.Error(w, "Pix vencido", http.StatusGone)
		return
	}
	c.pagamentos++
	s.seq++
	// O extrato sai por busca binária na hora do pagamento (ver extrato): a hora nunca volta, nem se
	// o relógio da máquina voltar.
	agora := time.Now().UTC()
	if n := len(s.pagamentos); n > 0 && agora.Before(s.pagamentos[n-1].PagoEm) {
		agora = s.pagamentos[n-1].PagoEm
	}
	p := Pagamento{ID: fmt.Sprintf("pag_%s_%06d", s.prefixo, s.seq), CobrancaID: c.ID, Referencia: c.Referencia,
		ValorCentavos: c.ValorCentavos, PagoEm: agora}
	s.pagamentos = append(s.pagamentos, p)
	s.pagIdx[p.ID] = len(s.pagamentos) - 1
	s.anotar(registro{Seq: s.seq, Pagamento: &p})
	s.mu.Unlock()

	s.avisos.Add(1)
	go s.avisar(p)
	escrever(w, http.StatusOK, p)
}

// clienteAviso reaproveita conexões com a API, como o cliente do lado de lá (ver NovoHTTP).
var clienteAviso = NovoHTTP(10 * time.Second)

func (s *Simulador) avisar(p Pagamento) {
	defer s.avisos.Done()
	time.Sleep(s.AtrasoAviso)
	if rand.Float64() < s.ProbPerderAviso || s.WebhookURL == "" {
		return
	}
	corpo, _ := json.Marshal(p)
	n := max(s.AvisosPorPagamento, 1)
	var wg sync.WaitGroup
	for range n { // as repetições chegam ao mesmo tempo: o pior caso
		wg.Go(func() {
			req, _ := http.NewRequest("POST", s.WebhookURL, bytes.NewReader(corpo))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set(CabecalhoAssinatura, Assinar(s.Segredo, corpo))
			if res, err := clienteAviso.Do(req); err == nil {
				fechar(res)
			}
		})
	}
	wg.Wait()
}

// EsperarAvisos bloqueia até todos os avisos em andamento terem sido entregues (para testes).
func (s *Simulador) EsperarAvisos() { s.avisos.Wait() }

func (s *Simulador) extrato(w http.ResponseWriter, r *http.Request) {
	desde, _ := time.Parse(time.RFC3339Nano, r.URL.Query().Get("desde"))
	ate, err := time.Parse(time.RFC3339Nano, r.URL.Query().Get("ate"))
	if err != nil {
		ate = time.Now()
	}
	// Os pagamentos estão em ordem de hora: as pontas da janela saem por busca binária. Antes era uma
	// varredura de todos, com a trava, a cada conciliação: medido em 05/11, a 1.280× local, 272 mil
	// pagamentos, o PSP a 3–5 núcleos por segundos a cada minuto, e cobranças e pagamentos esperando.
	s.mu.Lock()
	i, _ := slices.BinarySearchFunc(s.pagamentos, desde, func(p Pagamento, t time.Time) int { return p.PagoEm.Compare(t) })
	j, _ := slices.BinarySearchFunc(s.pagamentos, ate, func(p Pagamento, t time.Time) int {
		if p.PagoEm.After(t) {
			return 1
		}
		return -1 // o último igual a ate entra
	})
	out := slices.Clone(s.pagamentos[i:max(i, j)])
	s.mu.Unlock()
	if out == nil {
		out = []Pagamento{}
	}
	escrever(w, http.StatusOK, out)
}

// Adulterar muda o valor registrado de um pagamento no extrato (para testar a conciliação).
func (s *Simulador) Adulterar(pagamentoID string, valor int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.adulterar(pagamentoID, valor)
	s.anotar(registro{Adulterar: &Pagamento{ID: pagamentoID, ValorCentavos: valor}})
}

func (s *Simulador) adulterar(pagamentoID string, valor int) {
	for i := range s.pagamentos {
		if s.pagamentos[i].ID == pagamentoID {
			s.pagamentos[i].ValorCentavos = valor
		}
	}
}

func escrever(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
