package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"lisura/internal/notificacao"
	"lisura/internal/pagamento"
	"lisura/internal/psp"
)

func (s *Server) cobrar(w http.ResponseWriter, r *http.Request, candidato int64) {
	id, ok := idDaRota(w, r)
	var in struct {
		Metodo string `json:"metodo"`
	}
	if !ok || !ler(w, r, &in) {
		return
	}
	// Responde sem esperar o prestador: o Pix ou a linha digitável chegam pela fila em instantes, e
	// o front pergunta de novo até aparecerem.
	c, err := pagamento.Cobrar(r.Context(), s.Pool, id, candidato, in.Metodo)
	if err != nil {
		falhar(w, err)
		return
	}
	responder(w, http.StatusOK, c)
}

func (s *Server) cobrancas(w http.ResponseWriter, r *http.Request, candidato int64) {
	id, ok := idDaRota(w, r)
	if !ok {
		return
	}
	cs, err := pagamento.DaInscricao(r.Context(), s.Pool, id, candidato)
	if err != nil {
		falhar(w, err)
		return
	}
	responder(w, http.StatusOK, cs)
}

// devolucoes lista o dinheiro que entrou a mais e o caminho de volta de cada pagamento.
func (s *Server) devolucoes(w http.ResponseWriter, r *http.Request, candidato int64) {
	id, ok := idDaRota(w, r)
	if !ok {
		return
	}
	ds, err := pagamento.DevolucoesDaInscricao(r.Context(), s.Pool, id, candidato)
	if err != nil {
		falhar(w, err)
		return
	}
	responder(w, http.StatusOK, ds)
}

// informarChave recebe a chave Pix para onde o dinheiro de uma devolução AGUARDANDO_CHAVE deve voltar.
// 202: a devolução sai pela fila, e o prestador só paga se o titular da chave for o CPF da conta.
func (s *Server) informarChave(w http.ResponseWriter, r *http.Request, candidato int64) {
	id, ok := idDaRota(w, r)
	dev, err := strconv.ParseInt(r.PathValue("devolucao"), 10, 64)
	var in struct {
		Chave string `json:"chave"`
	}
	if !ok {
		return
	}
	if err != nil {
		erro(w, http.StatusNotFound, "não encontrado", nil)
		return
	}
	if !ler(w, r, &in) {
		return
	}
	if err := pagamento.InformarChave(r.Context(), s.Pool, id, candidato, dev, in.Chave); err != nil {
		falhar(w, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

// avisos lista os e-mails mandados ao candidato sobre a inscrição, com o resumo que prova o texto.
func (s *Server) avisos(w http.ResponseWriter, r *http.Request, candidato int64) {
	id, ok := idDaRota(w, r)
	if !ok {
		return
	}
	as, err := notificacao.DaInscricao(r.Context(), s.Pool, id, candidato)
	if err != nil {
		falhar(w, err)
		return
	}
	responder(w, http.StatusOK, as)
}

// simularPagamento é o candidato pagando no aplicativo do banco — só no sandbox.
func (s *Server) simularPagamento(w http.ResponseWriter, r *http.Request, candidato int64) {
	cobranca, err := strconv.ParseInt(r.PathValue("cobranca"), 10, 64)
	if s.Sandbox == nil || err != nil {
		erro(w, http.StatusNotFound, "não encontrado", nil)
		return
	}
	pspID, err := pagamento.PSPIDDaCobranca(r.Context(), s.Pool, cobranca, candidato)
	if err == nil {
		err = s.Sandbox.SimularPagamento(r.Context(), pspID)
	}
	if err != nil {
		falhar(w, err)
		return
	}
	w.WriteHeader(http.StatusAccepted) // a confirmação chega pelo aviso do prestador, em instantes
}

// avisoDePagamento recebe o aviso (webhook) do prestador. Só confia no que vem assinado com o
// segredo combinado. Responde 200 também para o aviso repetido: para o prestador, "já recebi" é
// sucesso — senão ele continua reenviando.
func (s *Server) avisoDePagamento(w http.ResponseWriter, r *http.Request) {
	corpo, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 64<<10))
	if err != nil {
		erro(w, http.StatusBadRequest, "aviso ilegível", nil)
		return
	}
	if !psp.AssinaturaValida(s.SegredoPSP, corpo, r.Header.Get(psp.CabecalhoAssinatura)) {
		erro(w, http.StatusUnauthorized, "assinatura inválida", nil)
		return
	}
	var p psp.Pagamento
	if err := json.Unmarshal(corpo, &p); err != nil || p.ID == "" {
		erro(w, http.StatusBadRequest, "aviso ilegível", nil)
		return
	}
	res, err := pagamento.RegistrarPagamento(r.Context(), s.Pool, p, "webhook")
	if err != nil {
		falhar(w, err) // 500: o prestador reenvia depois, e reprocessar é seguro
		return
	}
	responder(w, http.StatusOK, map[string]string{"resultado": res})
}
