package api

import (
	"net/http"
	"strconv"

	"lisura/internal/certame"
	"lisura/internal/conta"
	"lisura/internal/degradacao"
	"lisura/internal/inscricao"
	"lisura/internal/resultado"
)

// Sessão por cookie para o front: HttpOnly (o JavaScript da página não lê o token) e
// SameSite=Lax (o navegador não o manda em formulário vindo de outro site).
const CookieSessao = "lisura_sessao"

func (s *Server) abrirSessao(w http.ResponseWriter, r *http.Request, id int64) {
	http.SetCookie(w, &http.Cookie{
		Name: CookieSessao, Value: s.Tokens.Emitir(id), Path: "/",
		MaxAge: int(s.Tokens.Validade.Seconds()), HttpOnly: true, Secure: degradacao.Seguro(r),
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *Server) sair(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: CookieSessao, Value: "", Path: "/", MaxAge: -1, HttpOnly: true,
		SameSite: http.SameSiteLaxMode})
	w.WriteHeader(http.StatusNoContent)
}

// eu devolve quem está logado: o front pergunta isto para montar a moldura.
func (s *Server) eu(w http.ResponseWriter, r *http.Request, candidato int64) {
	cpf, dados, err := conta.Buscar(r.Context(), s.Pool, candidato)
	if err != nil {
		falhar(w, err)
		return
	}
	responder(w, http.StatusOK, struct {
		ID    int64       `json:"id"`
		CPF   string      `json:"cpf"`
		Dados conta.Dados `json:"dados"`
	}{candidato, cpf, dados})
}

func (s *Server) documentos(w http.ResponseWriter, r *http.Request, candidato int64) {
	docs, err := conta.Documentos(r.Context(), s.Pool, candidato)
	if err != nil {
		falhar(w, err)
		return
	}
	responder(w, http.StatusOK, nuncaNulo(docs))
}

func (s *Server) certamesAbertos(w http.ResponseWriter, r *http.Request) {
	lista, err := certame.ComInscricao(r.Context(), s.Pool)
	if err != nil {
		falhar(w, err)
		return
	}
	responder(w, http.StatusOK, nuncaNulo(lista))
}

func (s *Server) minhaIsencao(w http.ResponseWriter, r *http.Request, candidato int64) {
	id, ok := idDaRota(w, r)
	if !ok {
		return
	}
	isen, err := inscricao.MinhaIsencao(r.Context(), s.Pool, id, candidato)
	if err != nil {
		falhar(w, err)
		return
	}
	responder(w, http.StatusOK, isen) // null quando não há pedido
}

func (s *Server) minhasInscricoes(w http.ResponseWriter, r *http.Request, candidato int64) {
	lista, err := inscricao.DoCandidato(r.Context(), s.Pool, candidato)
	if err != nil {
		falhar(w, err)
		return
	}
	responder(w, http.StatusOK, nuncaNulo(lista))
}

func (s *Server) pendencias(w http.ResponseWriter, r *http.Request, candidato int64) {
	id, ok := idDaRota(w, r)
	if !ok {
		return
	}
	f, err := inscricao.Pendencias(r.Context(), s.Pool, id, candidato)
	if err != nil {
		falhar(w, err)
		return
	}
	responder(w, http.StatusOK, map[string][]string{"faltando": nuncaNulo(f)})
}

func (s *Server) trilha(w http.ResponseWriter, r *http.Request, candidato int64) {
	id, ok := idDaRota(w, r)
	if !ok {
		return
	}
	t, err := inscricao.Trilha(r.Context(), s.Pool, id, candidato)
	if err != nil {
		falhar(w, err)
		return
	}
	responder(w, http.StatusOK, nuncaNulo(t))
}

// cartao devolve o endereço estático do cartão de confirmação. O endereço tem um código que só o dono
// da inscrição recebe; o arquivo em si é servido pelo CDN, sem sessão e sem banco.
func (s *Server) cartao(w http.ResponseWriter, r *http.Request, candidato int64) {
	id, ok := idDaRota(w, r)
	if !ok {
		return
	}
	if _, err := inscricao.Revisao(r.Context(), s.Pool, id, candidato); err != nil {
		falhar(w, err) // 404 se não for do candidato
		return
	}
	if s.Publicador == nil {
		erro(w, http.StatusNotFound, "publicação desligada", nil)
		return
	}
	caminho := s.Publicador.CaminhoCartao(id)
	responder(w, http.StatusOK, map[string]any{"disponivel": s.Publicador.Existe(r.Context(), caminho), "url": "/publico/" + caminho})
}

// resultado devolve onde o candidato acha o próprio resultado: a fatia estática e a chave da linha
// dele. No pico 5 o front nem precisa desta rota se já guardou o código do cartão (a chave deriva
// dele); ela existe para quem entra de outro aparelho. A consulta é uma linha por chave primária.
func (s *Server) resultado(w http.ResponseWriter, r *http.Request, candidato int64) {
	id, ok := idDaRota(w, r)
	if !ok {
		return
	}
	i, err := inscricao.Revisao(r.Context(), s.Pool, id, candidato)
	if err != nil {
		falhar(w, err) // 404 se não for do candidato
		return
	}
	if s.Publicador == nil {
		erro(w, http.StatusNotFound, "publicação desligada", nil)
		return
	}
	chave := resultado.ChaveResultado(s.Publicador.Token(id))
	responder(w, http.StatusOK, map[string]string{
		"chave":    chave,
		"url":      "/publico/" + resultado.CaminhoFatia(i.CertameID, chave),
		"lista":    "/publico/certames/" + strconv.FormatInt(i.CertameID, 10) + "/resultado.json",
		"gabarito": "/publico/certames/" + strconv.FormatInt(i.CertameID, 10) + "/gabarito.json",
	})
}

func (s *Server) recorrer(w http.ResponseWriter, r *http.Request, candidato int64) {
	id, ok := idDaRota(w, r)
	var in resultado.Pedido
	if !ok || !ler(w, r, &in) {
		return
	}
	rid, err := resultado.Recorrer(r.Context(), s.Pool, id, candidato, in)
	if err != nil {
		falhar(w, err)
		return
	}
	responder(w, http.StatusCreated, map[string]int64{"id": rid})
}

// meusRecursos lista os recursos da inscrição com a resposta de cada um: a decisão da questão.
func (s *Server) meusRecursos(w http.ResponseWriter, r *http.Request, candidato int64) {
	id, ok := idDaRota(w, r)
	if !ok {
		return
	}
	if _, err := inscricao.Revisao(r.Context(), s.Pool, id, candidato); err != nil {
		falhar(w, err)
		return
	}
	lista, err := resultado.MeusRecursos(r.Context(), s.Pool, id, candidato)
	if err != nil {
		falhar(w, err)
		return
	}
	responder(w, http.StatusOK, nuncaNulo(lista))
}

// nuncaNulo faz lista vazia sair como [] no JSON, não null: o front não precisa tratar os dois.
func nuncaNulo[T any](v []T) []T {
	if v == nil {
		return []T{}
	}
	return v
}
