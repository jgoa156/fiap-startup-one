package pagamento

import (
	"errors"
	"net/mail"
	"regexp"
	"strings"

	"lisura/internal/conta"
)

var ErrChavePix = errors.New("chave Pix inválida")

var (
	telefonePix  = regexp.MustCompile(`^\+55[1-9][0-9][0-9]{8,9}$`) // +55, DDD e o número
	aleatoriaPix = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	cpfOuCNPJ    = regexp.MustCompile(`^[0-9.\-/]+$`)
)

// NormalizarChavePix confere o formato de uma chave Pix e a devolve como o DICT a guarda: CPF só com
// os dígitos (e com os verificadores certos), e-mail e chave aleatória em minúsculas, telefone no
// formato +55DDNNNNNNNNN. Chave de CNPJ é recusada: é de empresa, nunca do candidato. O formato certo
// não diz de quem é a chave; isso o prestador confere na devolução.
func NormalizarChavePix(s string) (string, error) {
	s = strings.TrimSpace(s)
	switch {
	case strings.HasPrefix(s, "+"):
		t := strings.NewReplacer(" ", "", "(", "", ")", "", "-", "").Replace(s)
		if !telefonePix.MatchString(t) {
			return "", errors.Join(ErrChavePix, errors.New("telefone: +55, DDD e número"))
		}
		return t, nil
	case strings.Contains(s, "@"):
		e := strings.ToLower(s)
		if a, err := mail.ParseAddress(e); err != nil || a.Address != e || len(e) > 77 {
			return "", errors.Join(ErrChavePix, errors.New("e-mail"))
		}
		return e, nil
	case aleatoriaPix.MatchString(strings.ToLower(s)):
		return strings.ToLower(s), nil
	case cpfOuCNPJ.MatchString(s):
		d := strings.NewReplacer(".", "", "-", "", "/", "").Replace(s)
		switch {
		case len(d) == 14:
			return "", errors.Join(ErrChavePix, errors.New("chave de CNPJ é de empresa, não do candidato"))
		case len(d) != 11 || !conta.CPFValido(d):
			return "", errors.Join(ErrChavePix, errors.New("CPF"))
		}
		return d, nil
	}
	return "", ErrChavePix
}
