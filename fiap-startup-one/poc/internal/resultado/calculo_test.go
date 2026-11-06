package resultado

import (
	"errors"
	"testing"
	"time"
)

// prova5 tem 5 questões: 3 de português (peso 1) e 2 de matemática (peso 2) — 7 pontos.
var prova5 = []Questao{
	{"Português", 100}, {"Português", 100}, {"Português", 100},
	{"Matemática", 200}, {"Matemática", 200},
}

var ref = time.Date(2027, 3, 31, 0, 0, 0, 0, time.UTC)

func nascido(anos int) time.Time { return ref.AddDate(-anos, 0, -1) }

func porInscricao(itens []Item) map[int64]Item {
	m := map[int64]Item{}
	for _, it := range itens {
		m[it.Inscricao] = it
	}
	return m
}

func TestCorrecao(t *testing.T) {
	itens, err := Calcular(prova5, "AB*DE", Regra{Vagas: 10, Referencia: ref}, []Candidato{
		{Inscricao: 1, Respostas: "ABCDE"}, // tudo certo
		{Inscricao: 2, Respostas: "..*.."}, // em branco e rasura: só a anulada
		{Inscricao: 3, Respostas: "BACDA"}, // marcou errado a anulada também: vale igual
		{Inscricao: 4, Respostas: ""},      // sem folha
	})
	if err != nil {
		t.Fatal(err)
	}
	m := porInscricao(itens)
	if it := m[1]; it.Nota != 700 || it.Acertos != 5 || it.PorDisciplina["Matemática"] != 400 {
		t.Errorf("tudo certo: %+v", it)
	}
	if it := m[2]; it.Nota != 100 || it.Acertos != 1 {
		t.Errorf("só a anulada: %+v", it)
	}
	if it := m[3]; it.Nota != 300 || it.PorDisciplina["Português"] != 100 || it.PorDisciplina["Matemática"] != 200 {
		t.Errorf("anulada vale para quem marcou qualquer coisa: %+v", it)
	}
	if it := m[4]; it.Situacao != Ausente || it.PosicaoAmpla != 0 {
		t.Errorf("sem folha tem de ser ausente, fora da lista: %+v", it)
	}
}

func TestFolhaOuGabaritoDeTamanhoErradoNaoCalcula(t *testing.T) {
	if _, err := Calcular(prova5, "ABCD", Regra{Vagas: 1}, nil); !errors.Is(err, ErrRegra) {
		t.Errorf("gabarito curto: %v", err)
	}
	if _, err := Calcular(prova5, "ABCDE", Regra{Vagas: 1}, []Candidato{{Inscricao: 1, Respostas: "ABC"}}); err == nil {
		t.Error("folha curta foi aceita")
	}
}

// Os números de cada lei, conferidos em 05/11/2026 no texto do Planalto.
func TestReservaDeVagas(t *testing.T) {
	pcd := Cota{Codigo: "pcd", Percentual: 500, Arredondamento: "cima", MinimoVagas: 5}    // Decreto 9.508
	ppiq := Cota{Codigo: "ppiq", Percentual: 3000, Arredondamento: "meio", MinimoVagas: 2} // Lei 15.142
	for _, c := range []struct {
		vagas, pcd, ppiq int
	}{
		{1, 0, 0},   // 1 vaga: nenhuma das duas reserva
		{2, 0, 1},   // 0,6 → 1
		{3, 0, 1},   // 0,9 → 1
		{4, 0, 1},   // 1,2 → 1
		{5, 1, 2},   // pcd 0,25 sobe para 1; ppiq 1,5 → 2
		{7, 1, 2},   // 2,1 → 2
		{10, 1, 3},  // pcd 0,5 → 1
		{40, 2, 12}, // números exatos
		{41, 3, 12}, // pcd 2,05 → 3 (qualquer fração sobe); ppiq 12,3 → 12
	} {
		res, err := Reservar(Regra{Vagas: c.vagas, Cotas: []Cota{pcd, ppiq}})
		if err != nil {
			t.Fatal(err)
		}
		if res.PorCota["pcd"] != c.pcd || res.PorCota["ppiq"] != c.ppiq || res.Ampla != c.vagas-c.pcd-c.ppiq {
			t.Errorf("%d vagas: %+v, esperado pcd %d ppiq %d", c.vagas, res, c.pcd, c.ppiq)
		}
	}
	if _, err := Reservar(Regra{Vagas: 2, Cotas: []Cota{{Codigo: "a", Percentual: 6000, Arredondamento: "cima", MinimoVagas: 1},
		{Codigo: "b", Percentual: 6000, Arredondamento: "cima", MinimoVagas: 1}}}); !errors.Is(err, ErrRegra) {
		t.Errorf("cotas que somam mais que as vagas: %v", err)
	}
}

func TestNotaMinimaElimina(t *testing.T) {
	itens, _ := Calcular(prova5, "ABCDE", Regra{Vagas: 10, NotaMinima: 400, Referencia: ref}, []Candidato{
		{Inscricao: 1, Respostas: "ABCDE"}, // 7
		{Inscricao: 2, Respostas: "ABC.."}, // 3: eliminado
		{Inscricao: 3, Respostas: "...DE"}, // 4: no limite, passa
	})
	m := porInscricao(itens)
	if m[2].Situacao != Eliminado || m[2].PosicaoAmpla != 0 {
		t.Errorf("abaixo da mínima: %+v", m[2])
	}
	if m[3].Situacao != Classificado || m[3].PosicaoAmpla != 2 {
		t.Errorf("na mínima: %+v", m[3])
	}
}

func TestDesempate(t *testing.T) {
	r := Regra{Vagas: 10, Desempate: []string{"disciplina:Matemática", "idade"}, Referencia: ref}
	// Todos com 3 pontos.
	itens, err := Calcular(prova5, "ABCDE", r, []Candidato{
		{Inscricao: 1, Respostas: "ABC..", Nascimento: nascido(30)},            // 3 em português
		{Inscricao: 2, Respostas: "A..D.", Nascimento: nascido(25)},            // 1 + 2 em matemática
		{Inscricao: 3, Respostas: "ABC..", Nascimento: nascido(61)},            // idoso: passa à frente de todos
		{Inscricao: 4, Respostas: "ABC..", Nascimento: nascido(70)},            // idoso mais velho: primeiro
		{Inscricao: 5, Respostas: "A...E", Nascimento: nascido(40)},            // matemática 2, mais velho que o 2
		{Inscricao: 6, Respostas: "ABC..", Nascimento: nascido(30)},            // igual ao 1 em tudo: sorteio
		{Inscricao: 7, Respostas: "ABC..", Nascimento: ref.AddDate(-60, 0, 0)}, // faz 60 no dia: é idoso
	})
	if err != nil {
		t.Fatal(err)
	}
	m := porInscricao(itens)
	ordem := []int64{4, 3, 7, 5, 2, 1, 6}
	for pos, insc := range ordem {
		if m[insc].PosicaoAmpla != pos+1 {
			t.Errorf("inscrição %d na posição %d, esperado %d", insc, m[insc].PosicaoAmpla, pos+1)
		}
	}
	for insc, emp := range map[int64]bool{1: true, 6: true, 2: false, 3: false, 4: false, 5: false, 7: false} {
		if m[insc].Empate != emp {
			t.Errorf("inscrição %d: empate %v, esperado %v", insc, m[insc].Empate, emp)
		}
	}
}

func TestSemNascimentoNaoDesempataPorIdade(t *testing.T) {
	r := Regra{Vagas: 10, Desempate: []string{"idade"}, Referencia: ref}
	itens, _ := Calcular(prova5, "ABCDE", r, []Candidato{
		{Inscricao: 1, Respostas: "ABCDE", Nascimento: nascido(30)},
		{Inscricao: 2, Respostas: "ABCDE"}, // data zero não pode virar "o mais velho de todos"
	})
	m := porInscricao(itens)
	if !m[1].Empate || !m[2].Empate {
		t.Errorf("sem data de nascimento o empate tem de ficar: %+v %+v", m[1], m[2])
	}
}

func TestDesempatePorDisciplinaInexistenteERecusado(t *testing.T) {
	if _, err := Calcular(prova5, "ABCDE", Regra{Vagas: 1, Desempate: []string{"disciplina:Física"}}, nil); !errors.Is(err, ErrRegra) {
		t.Errorf("disciplina fora da prova: %v", err)
	}
	if _, err := Calcular(prova5, "ABCDE", Regra{Vagas: 1, Desempate: []string{"altura"}}, nil); !errors.Is(err, ErrRegra) {
		t.Errorf("critério desconhecido: %v", err)
	}
}

// 5 vagas: 3 ampla, 2 reservadas (ppiq 30% "meio" → 1,5 → 2). As notas caem de 1 em 1 ponto conforme
// a inscrição sobe, então a lista geral é a ordem das inscrições.
func cincoVagas(cotas map[int64][]string, n int) []Item {
	r := Regra{Vagas: 5, Referencia: ref, Cotas: []Cota{{Codigo: "ppiq", Percentual: 3000, Arredondamento: "meio", MinimoVagas: 2}}}
	prova := make([]Questao, n)
	gab := make([]byte, n)
	for i := range prova {
		prova[i] = Questao{"Geral", 100}
		gab[i] = 'A'
	}
	var cs []Candidato
	for i := 1; i <= n; i++ {
		resp := []byte(string(gab))
		for j := 0; j < i-1; j++ {
			resp[j] = '.'
		}
		cs = append(cs, Candidato{Inscricao: int64(i), Respostas: string(resp), Cotas: cotas[int64(i)]})
	}
	itens, err := Calcular(prova, string(gab), r, cs)
	if err != nil {
		panic(err)
	}
	return itens
}

func TestCotistaNaAmplaNaoOcupaVagaReservada(t *testing.T) {
	// O 2 é cotista e entra pela ampla; as 2 reservadas vão aos próximos cotistas, 6 e 8.
	m := porInscricao(cincoVagas(map[int64][]string{2: {"ppiq"}, 6: {"ppiq"}, 8: {"ppiq"}, 9: {"ppiq"}}, 10))
	for insc, vaga := range map[int64]string{1: "ampla", 2: "ampla", 3: "ampla", 6: "ppiq", 8: "ppiq", 4: "", 5: "", 9: ""} {
		if m[insc].Vaga != vaga {
			t.Errorf("inscrição %d: vaga %q, esperado %q", insc, m[insc].Vaga, vaga)
		}
	}
	if m[4].Situacao != CadastroReserva {
		t.Errorf("o 4 fica no cadastro reserva: %+v", m[4])
	}
	if m[6].PosicoesCota["ppiq"] != 2 || m[2].PosicoesCota["ppiq"] != 1 || m[1].PosicoesCota != nil {
		t.Errorf("lista da cota: 2→1º, 6→2º; não cotista fora: %+v %+v %+v", m[2], m[6], m[1])
	}
}

func TestVagaReservadaSemCotistaVoltaParaAAmpla(t *testing.T) {
	// Só um cotista aprovado (o 7): uma reservada fica com ele, a outra volta para o 4.
	m := porInscricao(cincoVagas(map[int64][]string{7: {"ppiq"}}, 8))
	for insc, vaga := range map[int64]string{1: "ampla", 2: "ampla", 3: "ampla", 7: "ppiq", 4: "ampla", 5: "", 6: ""} {
		if m[insc].Vaga != vaga {
			t.Errorf("inscrição %d: vaga %q, esperado %q", insc, m[insc].Vaga, vaga)
		}
	}
	classificados := 0
	for _, it := range m {
		if it.Situacao == Classificado {
			classificados++
		}
	}
	if classificados != 5 {
		t.Errorf("%d classificados para 5 vagas", classificados)
	}
}
