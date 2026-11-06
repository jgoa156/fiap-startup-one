package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Tokens são assinados e sem estado: validar não consulta o banco, o que importa no pico.
type Tokens struct {
	Chave    []byte
	Validade time.Duration
}

var errToken = errors.New("token inválido")

var b64 = base64.RawURLEncoding

func (t Tokens) Emitir(candidatoID int64) string {
	payload := fmt.Sprintf("%d.%d", candidatoID, time.Now().Add(t.Validade).Unix())
	return b64.EncodeToString([]byte(payload)) + "." + b64.EncodeToString(t.assinar(payload))
}

func (t Tokens) Validar(token string) (int64, error) {
	p, sig, ok := strings.Cut(token, ".")
	if !ok {
		return 0, errToken
	}
	payload, err1 := b64.DecodeString(p)
	assinatura, err2 := b64.DecodeString(sig)
	if err1 != nil || err2 != nil || !hmac.Equal(assinatura, t.assinar(string(payload))) {
		return 0, errToken
	}
	var id, exp int64
	if _, err := fmt.Sscanf(string(payload), "%d.%d", &id, &exp); err != nil || time.Now().Unix() > exp {
		return 0, errToken
	}
	return id, nil
}

func (t Tokens) assinar(payload string) []byte {
	m := hmac.New(sha256.New, t.Chave)
	m.Write([]byte(payload))
	return m.Sum(nil)
}
