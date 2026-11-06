// Package psp é a fronteira com o prestador de serviço de pagamento: a empresa que gera o Pix e o
// boleto e avisa quando o dinheiro entrou. Na POC o prestador é o Simulador deste pacote; o
// Cliente fala com ele por HTTP, do mesmo jeito que falaria com um prestador real.
package psp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"time"
)

type NovaCobranca struct {
	// Referencia é o id da cobrança no nosso banco. Vai como chave de idempotência: pedir duas
	// vezes com a mesma referência devolve a mesma cobrança, nunca uma segunda.
	Referencia    int64     `json:"referencia"`
	ValorCentavos int       `json:"valor_centavos"`
	Metodo        string    `json:"metodo"`
	VenceEm       time.Time `json:"vence_em"`
}

type Cobranca struct {
	ID             string `json:"id"`
	PixCopiaCola   string `json:"pix_copia_cola,omitempty"`
	LinhaDigitavel string `json:"linha_digitavel,omitempty"`
}

// Pagamento é o dinheiro que entrou. Chega pelo aviso (webhook) e aparece no extrato; o ID é o
// mesmo nos dois caminhos, e é por ele que se evita processar duas vezes.
type Pagamento struct {
	ID            string    `json:"pagamento_id"`
	CobrancaID    string    `json:"cobranca_id"`
	Referencia    int64     `json:"referencia"`
	ValorCentavos int       `json:"valor_centavos"`
	PagoEm        time.Time `json:"pago_em"`
}

// NovaDevolucao pede ao prestador que devolva (parte de) um pagamento. Como na API Pix do Banco
// Central, o nosso id vai na URL (PUT /devolucoes/{id}): pedir de novo com o mesmo id devolve a mesma
// devolução, nunca uma segunda. É o que deixa a fila repetir à vontade.
type NovaDevolucao struct {
	ID            string `json:"-"`
	PagamentoID   string `json:"pagamento_id"`
	ValorCentavos int    `json:"valor_centavos"`
	// Pagamento por Pix volta para quem pagou, sem chave. Boleto não diz quem pagou: o dinheiro vai
	// por Pix para ChavePix, e o prestador recusa se o titular da chave não for CPFTitular.
	ChavePix   string `json:"chave_pix,omitempty"`
	CPFTitular string `json:"cpf_titular,omitempty"`
}

type Devolucao struct {
	ID            string    `json:"devolucao_id"` // o id do prestador
	Referencia    string    `json:"referencia"`   // o nosso
	PagamentoID   string    `json:"pagamento_id"`
	ValorCentavos int       `json:"valor_centavos"`
	DevolvidaEm   time.Time `json:"devolvida_em"`
}

// Motivos com que o prestador recusa uma devolução de vez (repetir não adianta).
const (
	RecusaChaveInexistente = "CHAVE_INEXISTENTE"   // o CPF não é chave Pix de ninguém
	RecusaTitularDiferente = "TITULAR_DIFERENTE"   // a chave é de outro CPF
	RecusaChaveObrigatoria = "CHAVE_OBRIGATORIA"   // boleto sem chave
	RecusaValorAcimaDoPago = "VALOR_ACIMA_DO_PAGO" // somada às devoluções anteriores, passa do pago
	RecusaPagamento        = "PAGAMENTO_INEXISTENTE"
	RecusaIDReutilizado    = "ID_REUTILIZADO" // o mesmo id já foi usado para outra devolução
)

// Assinar produz a assinatura que acompanha o aviso; só quem tem o segredo combinado assina.
func Assinar(segredo, corpo []byte) string {
	m := hmac.New(sha256.New, segredo)
	m.Write(corpo)
	return hex.EncodeToString(m.Sum(nil))
}

func AssinaturaValida(segredo, corpo []byte, assinatura string) bool {
	esperada, err := hex.DecodeString(assinatura)
	if err != nil {
		return false
	}
	m := hmac.New(sha256.New, segredo)
	m.Write(corpo)
	return hmac.Equal(esperada, m.Sum(nil))
}

const CabecalhoAssinatura = "X-Assinatura"
