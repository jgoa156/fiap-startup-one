// Package resultado corrige a prova objetiva contra o gabarito, aplica nota mínima, cotas e
// desempate, classifica e publica (02-produto §6 e módulo G). Cada cálculo é uma versão nova e
// imutável: anular uma questão recalcula a nota de todo mundo, e a classificação anterior continua
// guardada. A correção física do cartão-resposta está fora do escopo da POC: a banca entrega as
// respostas lidas pela leitora óptica.
//
// Este arquivo é só a conta, sem banco: entra a prova, o gabarito, as regras e os candidatos; sai a
// classificação. As notas são inteiras, em centésimos de ponto — nota não pode depender de
// arredondamento de ponto flutuante.
package resultado

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

type Situacao string

const (
	Classificado    Situacao = "CLASSIFICADO"     // dentro das vagas
	CadastroReserva Situacao = "CADASTRO_RESERVA" // aprovado, fora das vagas
	Eliminado       Situacao = "ELIMINADO"        // abaixo da nota mínima
	Ausente         Situacao = "AUSENTE"          // sem folha de respostas
)

const VagaAmpla = "ampla"

type Questao struct {
	Disciplina string
	Peso       int64 // centésimos de ponto
}

type Cota struct {
	Codigo         string
	Percentual     int64  // centésimos de ponto percentual: 30% = 3000
	Arredondamento string // "cima": qualquer fração sobe; "meio": fração ≥ 0,5 sobe
	MinimoVagas    int    // com menos vagas que isto, a cota não reserva nenhuma
}

// Regra é o que o edital diz sobre o cargo.
type Regra struct {
	Vagas      int
	NotaMinima int64 // centésimos de ponto
	// Desempate vem depois do critério legal da idade, na ordem do edital: "disciplina:<nome>"
	// (maior nota na disciplina) ou "idade" (mais velho primeiro, idoso ou não).
	Desempate []string
	Cotas     []Cota
	// Referencia é a data em que se mede a idade (o último dia de inscrição, em geral).
	Referencia time.Time
}

type Candidato struct {
	Inscricao  int64
	Respostas  string // vazio = ausente
	Cotas      []string
	Nascimento time.Time
}

type Item struct {
	Inscricao     int64
	Situacao      Situacao
	Nota          int64 // centésimos
	Acertos       int
	PorDisciplina map[string]int64
	PosicaoAmpla  int            // 0 fora da lista
	PosicoesCota  map[string]int // posição em cada lista de cota a que concorre
	Vaga          string         // "" fora das vagas
	Empate        bool           // os critérios do edital não desfizeram: vai a sorteio
}

// Reserva é quantas vagas cada cota reserva e quantas sobram para a ampla concorrência.
type Reserva struct {
	Ampla   int
	PorCota map[string]int
}

var ErrRegra = errors.New("regra do cargo inválida")

// Reservar aplica percentual, arredondamento e mínimo de vagas de cada cota.
func Reservar(r Regra) (Reserva, error) {
	res := Reserva{Ampla: r.Vagas, PorCota: map[string]int{}}
	for _, c := range r.Cotas {
		n := 0
		if r.Vagas >= c.MinimoVagas && c.Percentual > 0 {
			num, den := int64(r.Vagas)*c.Percentual, int64(10000)
			switch c.Arredondamento {
			case "cima":
				n = int((num + den - 1) / den)
			case "meio":
				n = int((2*num + den) / (2 * den))
			default:
				return res, fmt.Errorf("%w: arredondamento %q da cota %s", ErrRegra, c.Arredondamento, c.Codigo)
			}
		}
		res.PorCota[c.Codigo] = n
		res.Ampla -= n
	}
	if res.Ampla < 0 {
		return res, fmt.Errorf("%w: as cotas reservam mais vagas (%d) do que há (%d)", ErrRegra, r.Vagas-res.Ampla, r.Vagas)
	}
	return res, nil
}

// Validar confere os critérios de desempate contra as disciplinas da prova.
func Validar(prova []Questao, r Regra) error {
	for _, d := range r.Desempate {
		if d == "idade" {
			continue
		}
		nome, ok := strings.CutPrefix(d, "disciplina:")
		if !ok || !slices.ContainsFunc(prova, func(q Questao) bool { return q.Disciplina == nome }) {
			return fmt.Errorf("%w: critério de desempate %q", ErrRegra, d)
		}
	}
	_, err := Reservar(r)
	return err
}

// Calcular corrige e classifica os candidatos de um cargo.
func Calcular(prova []Questao, gabarito string, r Regra, candidatos []Candidato) ([]Item, error) {
	if len(gabarito) != len(prova) {
		return nil, fmt.Errorf("%w: gabarito com %d respostas para %d questões", ErrRegra, len(gabarito), len(prova))
	}
	if err := Validar(prova, r); err != nil {
		return nil, err
	}
	reserva, _ := Reservar(r)

	itens := make([]Item, len(candidatos))
	for i, c := range candidatos {
		it := Item{Inscricao: c.Inscricao}
		switch {
		case c.Respostas == "":
			it.Situacao = Ausente
		case len(c.Respostas) != len(prova):
			return nil, fmt.Errorf("inscrição %d: folha com %d respostas para %d questões", c.Inscricao, len(c.Respostas), len(prova))
		default:
			it.PorDisciplina = map[string]int64{}
			for q, certa := range []byte(gabarito) {
				// Questão anulada vale para todos os presentes, marcaram o que marcaram.
				if certa == '*' || c.Respostas[q] == certa {
					it.Nota += prova[q].Peso
					it.PorDisciplina[prova[q].Disciplina] += prova[q].Peso
					it.Acertos++
				}
			}
			it.Situacao = CadastroReserva
			if it.Nota < r.NotaMinima {
				it.Situacao = Eliminado
			}
		}
		itens[i] = it
	}

	// Ordem de classificação: nota, depois desempate. A ordem final por inscrição só dá um
	// resultado reprodutível ao empate que sobra — que é marcado, porque a lei o manda a sorteio.
	nasc := make(map[int64]time.Time, len(candidatos))
	cotas := make(map[int64][]string, len(candidatos))
	for _, c := range candidatos {
		nasc[c.Inscricao] = c.Nascimento
		cotas[c.Inscricao] = c.Cotas
	}
	desempate := func(a, b *Item) int {
		if n := cmp.Compare(b.Nota, a.Nota); n != 0 {
			return n
		}
		// Estatuto da Pessoa Idosa, art. 27, parágrafo único: o primeiro critério é a idade, com
		// preferência ao mais velho — entre quem tem 60 anos ou mais na data de referência.
		ia, ib := idoso(nasc[a.Inscricao], r.Referencia), idoso(nasc[b.Inscricao], r.Referencia)
		switch {
		case ia && !ib:
			return -1
		case ib && !ia:
			return 1
		case ia && ib:
			if n := nasc[a.Inscricao].Compare(nasc[b.Inscricao]); n != 0 {
				return n
			}
		}
		for _, d := range r.Desempate {
			if d == "idade" {
				// Sem data de nascimento não há como desempatar por idade: segue para o próximo.
				na, nb := nasc[a.Inscricao], nasc[b.Inscricao]
				if n := na.Compare(nb); n != 0 && !na.IsZero() && !nb.IsZero() {
					return n
				}
				continue
			}
			disc := strings.TrimPrefix(d, "disciplina:")
			if n := cmp.Compare(b.PorDisciplina[disc], a.PorDisciplina[disc]); n != 0 {
				return n
			}
		}
		return 0
	}

	var aprovados []*Item
	for i := range itens {
		if itens[i].Situacao == CadastroReserva {
			aprovados = append(aprovados, &itens[i])
		}
	}
	slices.SortStableFunc(aprovados, func(a, b *Item) int {
		if n := desempate(a, b); n != 0 {
			return n
		}
		return cmp.Compare(a.Inscricao, b.Inscricao)
	})
	for i, it := range aprovados {
		it.PosicaoAmpla = i + 1
		if i > 0 && desempate(aprovados[i-1], it) == 0 {
			aprovados[i-1].Empate, it.Empate = true, true
		}
	}

	// Listas por cota: o cotista concorre ao mesmo tempo às duas (Lei 15.142, art. 7º; Decreto
	// 9.508, art. 8º).
	listas := map[string][]*Item{}
	for _, it := range aprovados {
		for _, c := range cotas[it.Inscricao] {
			if _, ok := reserva.PorCota[c]; !ok {
				continue
			}
			if it.PosicoesCota == nil {
				it.PosicoesCota = map[string]int{}
			}
			listas[c] = append(listas[c], it)
			it.PosicoesCota[c] = len(listas[c])
		}
	}

	// Vagas. 1) A ampla concorrência, pela lista geral. 2) Cada cota, pela própria lista, pulando
	// quem já entrou pela ampla — esse não ocupa vaga reservada (Lei 15.142, art. 7º § 2º). 3) Vaga
	// reservada que sobra por falta de cotista aprovado volta para a ampla, na ordem da lista geral
	// (Lei 15.142, art. 8º; Decreto 9.508, art. 1º § 5º). As cotas são atendidas na ordem do edital;
	// quem concorre a duas fica com a primeira em que alcançar vaga.
	dar := func(it *Item, vaga string) {
		it.Situacao, it.Vaga = Classificado, vaga
	}
	livres := reserva.Ampla
	for _, it := range aprovados {
		if livres == 0 {
			break
		}
		dar(it, VagaAmpla)
		livres--
	}
	for _, c := range r.Cotas {
		resta := reserva.PorCota[c.Codigo]
		for _, it := range listas[c.Codigo] {
			if resta == 0 {
				break
			}
			if it.Vaga == "" {
				dar(it, c.Codigo)
				resta--
			}
		}
		livres += resta
	}
	for _, it := range aprovados {
		if livres == 0 {
			break
		}
		if it.Vaga == "" {
			dar(it, VagaAmpla)
			livres--
		}
	}
	return itens, nil
}

// idoso diz se quem nasceu em nasc tinha 60 anos ou mais na data ref.
func idoso(nasc, ref time.Time) bool {
	if nasc.IsZero() {
		return false
	}
	return !nasc.AddDate(60, 0, 0).After(ref)
}
