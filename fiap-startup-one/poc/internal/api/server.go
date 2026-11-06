// Package api é a interface HTTP do candidato. O painel da banca está fora do escopo da POC.
package api

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"lisura/internal/certame"
	"lisura/internal/conta"
	"lisura/internal/inscricao"
	"lisura/internal/metricas"
	"lisura/internal/pagamento"
	"lisura/internal/psp"
	"lisura/internal/publicacao"
	"lisura/internal/resultado"
)

type Server struct {
	Pool       *pgxpool.Pool
	Store      conta.Armazenamento
	Tokens     Tokens
	SegredoPSP []byte // assina os avisos de pagamento; aviso sem assinatura válida é recusado
	// Publicador sabe onde fica o cartão de cada inscrição (os arquivos estáticos de /publico).
	Publicador *publicacao.Publicador
	// Sandbox, quando presente, liga a rota que simula o candidato pagando no banco. Só existe com
	// o prestador simulado; em produção fica nil e a rota responde 404.
	Sandbox *psp.Cliente
	// Metricas, se dado, mede cada requisição pelo padrão da rota (o painel em tempo real).
	Metricas *metricas.Metricas
}

// rota é uma rota da API: o padrão do ServeMux (método e caminho, sem o /api) e quem responde.
type rota struct {
	padrao string
	h      http.HandlerFunc
}

// rotas é a tabela de tudo o que a API responde. A documentação (openapi.yaml) é conferida contra
// ela por teste: rota sem documentação, ou documentação de rota que não existe, quebra a suíte.
func (s *Server) rotas() []rota {
	return []rota{
		{"GET /saude", func(w http.ResponseWriter, r *http.Request) {
			responder(w, http.StatusOK, map[string]string{"status": "ok"})
		}},
		{"GET /openapi.yaml", servirOpenAPI},
		{"POST /contas", s.criarConta},
		{"POST /sessoes", s.criarSessao},
		{"POST /sessoes/sair", s.sair},
		{"GET /eu", s.auth(s.eu)},
		{"GET /conta/documentos", s.auth(s.documentos)},
		{"GET /certames", s.certamesAbertos},
		{"GET /certames/{id}/isencao", s.auth(s.minhaIsencao)},
		{"GET /inscricoes", s.auth(s.minhasInscricoes)},
		{"GET /inscricoes/{id}/pendencias", s.auth(s.pendencias)},
		{"GET /inscricoes/{id}/trilha", s.auth(s.trilha)},
		{"GET /inscricoes/{id}/cartao", s.auth(s.cartao)},
		{"GET /inscricoes/{id}/resultado", s.auth(s.resultado)},
		{"GET /inscricoes/{id}/recursos", s.auth(s.meusRecursos)},
		{"POST /inscricoes/{id}/recursos", s.auth(s.recorrer)},
		{"PUT /conta", s.auth(s.atualizarConta)},
		{"POST /conta/documentos", s.auth(s.enviarDocumento)},
		{"GET /certames/{id}", s.verCertame},
		{"POST /certames/{id}/isencao", s.auth(s.solicitarIsencao)},
		{"POST /certames/{id}/inscricoes", s.auth(s.abrirInscricao)},
		{"GET /inscricoes/{id}", s.auth(s.revisao)},
		{"PUT /inscricoes/{id}/cargo", s.auth(s.passoCargo)},
		{"PUT /inscricoes/{id}/cidade", s.auth(s.passoCidade)},
		{"PUT /inscricoes/{id}/cotas", s.auth(s.passoCotas)},
		{"PUT /inscricoes/{id}/atendimento", s.auth(s.passoAtendimento)},
		{"PUT /inscricoes/{id}/declaracoes", s.auth(s.passoDeclaracoes)},
		{"POST /inscricoes/{id}/finalizar", s.auth(s.finalizar)},
		{"POST /inscricoes/{id}/pagamento", s.auth(s.cobrar)},
		{"GET /inscricoes/{id}/pagamento", s.auth(s.cobrancas)},
		{"POST /inscricoes/{id}/pagamento/{cobranca}/simular", s.auth(s.simularPagamento)},
		{"GET /inscricoes/{id}/devolucoes", s.auth(s.devolucoes)},
		{"PUT /inscricoes/{id}/devolucoes/{devolucao}/chave", s.auth(s.informarChave)},
		{"GET /inscricoes/{id}/avisos", s.auth(s.avisos)},
		{"POST /psp/aviso", s.avisoDePagamento},
	}
}

// Padroes lista as rotas que a API responde, como o ServeMux as conhece.
func Padroes() []string {
	var out []string
	for _, r := range (&Server{}).rotas() {
		out = append(out, r.padrao)
	}
	return out
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	for _, r := range s.rotas() {
		mux.HandleFunc(r.padrao, r.h)
	}
	if s.Metricas != nil {
		return s.Metricas.Rotas(mux)
	}
	return mux
}

// ---- conta ----

func (s *Server) criarConta(w http.ResponseWriter, r *http.Request) {
	var in struct{ CPF, Nome, Email string }
	if !ler(w, r, &in) {
		return
	}
	id, err := conta.Criar(r.Context(), s.Pool, in.CPF, in.Nome, in.Email)
	if err != nil {
		falhar(w, err)
		return
	}
	s.abrirSessao(w, r, id)
	responder(w, http.StatusCreated, map[string]any{"id": id, "token": s.Tokens.Emitir(id)})
}

// criarSessao substitui o login real (gov.br) na POC: entra só com o CPF, porque todos os
// candidatos são sintéticos. Não vai para produção assim.
func (s *Server) criarSessao(w http.ResponseWriter, r *http.Request) {
	var in struct{ CPF string }
	if !ler(w, r, &in) {
		return
	}
	id, err := conta.PorCPF(r.Context(), s.Pool, in.CPF)
	if err != nil {
		falhar(w, err)
		return
	}
	s.abrirSessao(w, r, id)
	responder(w, http.StatusOK, map[string]any{"id": id, "token": s.Tokens.Emitir(id)})
}

func (s *Server) atualizarConta(w http.ResponseWriter, r *http.Request, candidato int64) {
	var in conta.Dados
	if !ler(w, r, &in) {
		return
	}
	if err := conta.AtualizarDados(r.Context(), s.Pool, candidato, in); err != nil {
		falhar(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) enviarDocumento(w http.ResponseWriter, r *http.Request, candidato int64) {
	r.Body = http.MaxBytesReader(w, r.Body, conta.MaxArquivo+64<<10)
	f, cab, err := r.FormFile("arquivo")
	if err != nil {
		var mb *http.MaxBytesError
		if errors.As(err, &mb) {
			falhar(w, conta.ErrArquivoGrande)
			return
		}
		erro(w, http.StatusBadRequest, "envie o arquivo no campo 'arquivo'", nil)
		return
	}
	defer f.Close()
	conteudo, err := io.ReadAll(io.LimitReader(f, conta.MaxArquivo+1))
	if err != nil {
		falhar(w, err)
		return
	}
	id, err := conta.SalvarDocumento(r.Context(), s.Pool, s.Store, candidato, r.FormValue("tipo"), cab.Filename, conteudo)
	if err != nil {
		falhar(w, err)
		return
	}
	responder(w, http.StatusCreated, map[string]int64{"id": id})
}

// ---- certame ----

func (s *Server) verCertame(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRota(w, r)
	if !ok {
		return
	}
	c, err := certame.Buscar(r.Context(), s.Pool, id)
	if err != nil {
		falhar(w, err)
		return
	}
	responder(w, http.StatusOK, c)
}

func (s *Server) solicitarIsencao(w http.ResponseWriter, r *http.Request, candidato int64) {
	id, ok := idDaRota(w, r)
	var in inscricao.PedidoIsencao
	if !ok || !ler(w, r, &in) {
		return
	}
	isen, err := inscricao.SolicitarIsencao(r.Context(), s.Pool, id, candidato, in)
	if err != nil {
		falhar(w, err)
		return
	}
	responder(w, http.StatusOK, isen)
}

// ---- inscrição ----

func (s *Server) abrirInscricao(w http.ResponseWriter, r *http.Request, candidato int64) {
	certameID, ok := idDaRota(w, r)
	var in struct {
		CargoID int64 `json:"cargo_id"`
	}
	if !ok || !ler(w, r, &in) {
		return
	}
	id, criada, err := inscricao.Abrir(r.Context(), s.Pool, certameID, candidato, in.CargoID, atorCandidato(candidato))
	if err != nil {
		falhar(w, err)
		return
	}
	status := http.StatusOK
	if criada {
		status = http.StatusCreated
	}
	responder(w, status, map[string]int64{"id": id})
}

func (s *Server) revisao(w http.ResponseWriter, r *http.Request, candidato int64) {
	id, ok := idDaRota(w, r)
	if !ok {
		return
	}
	i, err := inscricao.Revisao(r.Context(), s.Pool, id, candidato)
	if err != nil {
		falhar(w, err)
		return
	}
	responder(w, http.StatusOK, i)
}

func (s *Server) passoCargo(w http.ResponseWriter, r *http.Request, candidato int64) {
	var in struct {
		CargoID int64 `json:"cargo_id"`
	}
	s.passo(w, r, &in, func(id int64) error {
		return inscricao.DefinirCargo(r.Context(), s.Pool, id, candidato, in.CargoID)
	})
}

func (s *Server) passoCidade(w http.ResponseWriter, r *http.Request, candidato int64) {
	var in struct {
		CidadeID int64 `json:"cidade_id"`
	}
	s.passo(w, r, &in, func(id int64) error {
		return inscricao.DefinirCidade(r.Context(), s.Pool, id, candidato, in.CidadeID)
	})
}

func (s *Server) passoCotas(w http.ResponseWriter, r *http.Request, candidato int64) {
	var in struct {
		Cotas []string `json:"cotas"`
	}
	s.passo(w, r, &in, func(id int64) error {
		return inscricao.DefinirCotas(r.Context(), s.Pool, id, candidato, in.Cotas)
	})
}

func (s *Server) passoAtendimento(w http.ResponseWriter, r *http.Request, candidato int64) {
	var in struct {
		Necessidades     []string `json:"necessidades"`
		LaudoDocumentoID *int64   `json:"laudo_documento_id"`
	}
	s.passo(w, r, &in, func(id int64) error {
		return inscricao.DefinirAtendimento(r.Context(), s.Pool, id, candidato, in.Necessidades, in.LaudoDocumentoID)
	})
}

func (s *Server) passoDeclaracoes(w http.ResponseWriter, r *http.Request, candidato int64) {
	var in struct {
		Aceito bool `json:"aceito"`
	}
	s.passo(w, r, &in, func(id int64) error {
		return inscricao.AceitarDeclaracoes(r.Context(), s.Pool, id, candidato, in.Aceito) // false retira o aceite
	})
}

func (s *Server) passo(w http.ResponseWriter, r *http.Request, in any, salvar func(id int64) error) {
	id, ok := idDaRota(w, r)
	if !ok || !ler(w, r, in) {
		return
	}
	if err := salvar(id); err != nil {
		falhar(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) finalizar(w http.ResponseWriter, r *http.Request, candidato int64) {
	id, ok := idDaRota(w, r)
	if !ok {
		return
	}
	res, err := inscricao.Finalizar(r.Context(), s.Pool, id, candidato)
	if err != nil {
		falhar(w, err)
		return
	}
	responder(w, http.StatusOK, map[string]any{"estado": res.Para, "reenvio": res.JaAplicada})
}

// ---- infraestrutura ----

// auth aceita o token no cabeçalho (scripts, ensaio de carga) ou no cookie de sessão (o front).
// Pelo cookie, o navegador envia a credencial sozinho — então escrita vinda de outro site é recusada.
func (s *Server) auth(h func(http.ResponseWriter, *http.Request, int64)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, peloCabecalho := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !peloCabecalho {
			if c, err := r.Cookie(CookieSessao); err == nil {
				token = c.Value
			}
			if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Header.Get("Sec-Fetch-Site") == "cross-site" {
				erro(w, http.StatusForbidden, "origem não permitida", nil)
				return
			}
		}
		id, err := s.Tokens.Validar(token)
		if err != nil {
			erro(w, http.StatusUnauthorized, "autenticação necessária", nil)
			return
		}
		h(w, r, id)
	}
}

func atorCandidato(id int64) inscricao.Ator {
	return inscricao.Ator{Tipo: inscricao.Candidato, ID: strconv.FormatInt(id, 10)}
}

func idDaRota(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		erro(w, http.StatusNotFound, "não encontrado", nil)
		return 0, false
	}
	return id, true
}

func ler(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		erro(w, http.StatusBadRequest, "JSON inválido: "+err.Error(), nil)
		return false
	}
	return true
}

func responder(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func erro(w http.ResponseWriter, status int, msg string, faltando []string) {
	responder(w, status, struct {
		Erro     string   `json:"erro"`
		Faltando []string `json:"faltando,omitempty"`
	}{msg, faltando})
}

func falhar(w http.ResponseWriter, err error) {
	var inc *inscricao.ErrIncompleta
	switch {
	case errors.As(err, &inc):
		erro(w, http.StatusUnprocessableEntity, err.Error(), inc.Faltando)
	case errors.Is(err, inscricao.ErrNaoEncontrada), errors.Is(err, conta.ErrNaoEncontrada),
		errors.Is(err, certame.ErrNaoEncontrado):
		erro(w, http.StatusNotFound, err.Error(), nil)
	case errors.Is(err, inscricao.ErrDadoInvalido), errors.Is(err, conta.ErrDadoInvalido):
		erro(w, http.StatusUnprocessableEntity, err.Error(), nil)
	case errors.Is(err, pagamento.ErrMetodo), errors.Is(err, resultado.ErrRecursoInvalido):
		erro(w, http.StatusUnprocessableEntity, err.Error(), nil)
	case errors.Is(err, resultado.ErrRecursoRepetido), errors.Is(err, resultado.ErrSemPreliminar),
		errors.Is(err, resultado.ErrSemGabarito):
		erro(w, http.StatusConflict, err.Error(), nil)
	case errors.Is(err, psp.ErrPrestador):
		erro(w, http.StatusBadGateway, err.Error(), nil)
	case errors.Is(err, pagamento.ErrNaoAguardaPagamento), errors.Is(err, pagamento.ErrPrazoPagamento),
		errors.Is(err, inscricao.ErrForaDoPrazo), errors.Is(err, inscricao.ErrNaoEditavel),
		errors.Is(err, inscricao.ErrTransicaoInvalida), errors.Is(err, inscricao.ErrEditalNaoPublicado),
		errors.Is(err, inscricao.ErrIsencaoJaDecidida), errors.Is(err, conta.ErrCPFJaCadastrado),
		errors.Is(err, psp.ErrPixJaPago), errors.Is(err, inscricao.ErrCertameCancelado),
		errors.Is(err, pagamento.ErrNaoAguardaChave):
		erro(w, http.StatusConflict, err.Error(), nil)
	case errors.Is(err, conta.ErrArquivoGrande):
		erro(w, http.StatusRequestEntityTooLarge, err.Error(), nil)
	case errors.Is(err, conta.ErrFormatoArquivo):
		erro(w, http.StatusUnsupportedMediaType, err.Error(), nil)
	case errors.Is(err, inscricao.ErrAtorNaoAutorizado):
		erro(w, http.StatusForbidden, err.Error(), nil)
	default:
		log.Printf("erro interno: %v", err)
		erro(w, http.StatusInternalServerError, "erro interno", nil)
	}
}
