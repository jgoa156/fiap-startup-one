package psp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Cliente fala com o prestador por HTTP.
type Cliente struct {
	URL  string
	HTTP *http.Client // com timeout; sem ele, um prestador lento prende a requisição do candidato
}

var ErrPrestador = errors.New("prestador de pagamento indisponível")

// ErrPixJaPago: o banco de quem paga recusou pagar de novo um Pix que já foi pago (só no sandbox).
var ErrPixJaPago = errors.New("este Pix já foi pago")

// NovoHTTP é o cliente HTTP para falar com o prestador sob carga. O transporte padrão do Go guarda
// só 2 conexões ociosas por destino: com dezenas de cobranças por segundo, cada chamada além dessas
// 2 abria uma conexão nova e a descartava (com um prestador real, um aperto de mão TLS por
// cobrança). Medido em 05/11, a 640× local, isso esgotou as portas da máquina.
func NovoHTTP(timeout time.Duration) *http.Client {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 512
	t.MaxIdleConnsPerHost = 128
	t.IdleConnTimeout = 90 * time.Second
	return &http.Client{Timeout: timeout, Transport: t}
}

// fechar lê o que sobrou da resposta antes de fechar: só assim o Go devolve a conexão para ser
// reaproveitada (fechada com corpo por ler, ela é descartada).
func fechar(res *http.Response) {
	io.Copy(io.Discard, io.LimitReader(res.Body, 64<<10))
	res.Body.Close()
}

// CriarCobranca tenta até 3 vezes. Repetir é seguro porque a referência vai como chave de
// idempotência: se a primeira tentativa criou a cobrança e só a resposta se perdeu, a segunda
// recebe a mesma cobrança.
func (c Cliente) CriarCobranca(ctx context.Context, n NovaCobranca) (Cobranca, error) {
	corpo, _ := json.Marshal(n)
	var ultimo error
	for tentativa := range 3 {
		if tentativa > 0 {
			select {
			case <-ctx.Done():
				return Cobranca{}, ctx.Err()
			case <-time.After(time.Duration(tentativa) * 200 * time.Millisecond):
			}
		}
		req, _ := http.NewRequestWithContext(ctx, "POST", c.URL+"/cobrancas", bytes.NewReader(corpo))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", strconv.FormatInt(n.Referencia, 10))
		res, err := c.HTTP.Do(req)
		if err != nil {
			ultimo = err
			continue
		}
		var out Cobranca
		err = json.NewDecoder(res.Body).Decode(&out)
		fechar(res)
		switch {
		case res.StatusCode >= 500:
			ultimo = fmt.Errorf("status %d", res.StatusCode)
			continue
		case res.StatusCode >= 400:
			return Cobranca{}, fmt.Errorf("%w: recusou a cobrança (status %d)", ErrPrestador, res.StatusCode)
		case err != nil:
			ultimo = err
			continue
		}
		return out, nil
	}
	return Cobranca{}, fmt.Errorf("%w: %v", ErrPrestador, ultimo)
}

// Extrato lista os pagamentos recebidos no período: a fonte da verdade para a conciliação.
func (c Cliente) Extrato(ctx context.Context, desde, ate time.Time) ([]Pagamento, error) {
	q := url.Values{"desde": {desde.UTC().Format(time.RFC3339Nano)}, "ate": {ate.UTC().Format(time.RFC3339Nano)}}
	req, _ := http.NewRequestWithContext(ctx, "GET", c.URL+"/extrato?"+q.Encode(), nil)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPrestador, err)
	}
	defer fechar(res)
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: extrato status %d", ErrPrestador, res.StatusCode)
	}
	var out []Pagamento
	return out, json.NewDecoder(res.Body).Decode(&out)
}

// ExtratoDevolucoes lista as devoluções feitas no período: a conferência da prestação de contas.
func (c Cliente) ExtratoDevolucoes(ctx context.Context, desde, ate time.Time) ([]Devolucao, error) {
	q := url.Values{"desde": {desde.UTC().Format(time.RFC3339Nano)}, "ate": {ate.UTC().Format(time.RFC3339Nano)}}
	req, _ := http.NewRequestWithContext(ctx, "GET", c.URL+"/devolucoes?"+q.Encode(), nil)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPrestador, err)
	}
	defer fechar(res)
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: extrato de devoluções status %d", ErrPrestador, res.StatusCode)
	}
	var out []Devolucao
	return out, json.NewDecoder(res.Body).Decode(&out)
}

// Recusa é a devolução que o prestador recusou de vez; Motivo é uma das constantes Recusa*.
type Recusa struct{ Motivo string }

func (r *Recusa) Error() string { return "o prestador recusou a devolução: " + r.Motivo }

// Devolver pede a devolução. Uma chamada só: quem repete é a fila, com o mesmo id, e o prestador
// devolve a mesma devolução. Erro *Recusa é definitivo; ErrPrestador, passageiro.
func (c Cliente) Devolver(ctx context.Context, d NovaDevolucao) (Devolucao, error) {
	corpo, _ := json.Marshal(d)
	req, _ := http.NewRequestWithContext(ctx, "PUT", c.URL+"/devolucoes/"+url.PathEscape(d.ID), bytes.NewReader(corpo))
	req.Header.Set("Content-Type", "application/json")
	res, err := c.HTTP.Do(req)
	if err != nil {
		return Devolucao{}, fmt.Errorf("%w: %v", ErrPrestador, err)
	}
	defer fechar(res)
	switch {
	case res.StatusCode >= 500:
		return Devolucao{}, fmt.Errorf("%w: devolução status %d", ErrPrestador, res.StatusCode)
	case res.StatusCode >= 400:
		var r struct {
			Codigo string `json:"codigo"`
		}
		if json.NewDecoder(res.Body).Decode(&r) != nil || r.Codigo == "" {
			r.Codigo = "STATUS_" + strconv.Itoa(res.StatusCode)
		}
		return Devolucao{}, &Recusa{Motivo: r.Codigo}
	}
	var out Devolucao
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return Devolucao{}, fmt.Errorf("%w: %v", ErrPrestador, err)
	}
	return out, nil
}

// SimularPagamento só existe no sandbox: é o "candidato pagando" no aplicativo do banco.
func (c Cliente) SimularPagamento(ctx context.Context, cobrancaID string) error {
	req, _ := http.NewRequestWithContext(ctx, "POST", c.URL+"/cobrancas/"+url.PathEscape(cobrancaID)+"/pagar", nil)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	fechar(res)
	if res.StatusCode == http.StatusConflict {
		return ErrPixJaPago
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("sandbox recusou o pagamento (status %d)", res.StatusCode)
	}
	return nil
}
