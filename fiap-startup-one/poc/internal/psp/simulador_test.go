package psp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// diario é um io.Writer seguro para as escritas concorrentes do simulador.
type diario struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (d *diario) Write(p []byte) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.b.Write(p)
}

func (d *diario) bytes() []byte {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]byte(nil), d.b.Bytes()...)
}

func subir(t *testing.T, s *Simulador) Cliente {
	t.Helper()
	srv := httptest.NewServer(s.Handler())
	t.Cleanup(srv.Close)
	return Cliente{URL: srv.URL, HTTP: NovoHTTP(5 * time.Second)}
}

func ids(t *testing.T, c Cliente) []string {
	t.Helper()
	ex, err := c.Extrato(context.Background(), time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, p := range ex {
		out = append(out, p.ID)
	}
	return out
}

// Reiniciar o simulador com o diário é reiniciar um prestador de verdade: a cobrança antiga continua
// pagável, a mesma referência devolve a mesma cobrança, o extrato tem os pagamentos de antes e os
// identificadores novos não colidem com os antigos.
func TestSimuladorSobreviveAReinicio(t *testing.T) {
	ctx := context.Background()
	d := &diario{}
	antes := subir(t, &Simulador{Diario: d})
	pix, err := antes.CriarCobranca(ctx, NovaCobranca{Referencia: 1, ValorCentavos: 8000, Metodo: "pix", VenceEm: time.Now().Add(time.Hour)})
	must(t, err)
	boleto, err := antes.CriarCobranca(ctx, NovaCobranca{Referencia: 2, ValorCentavos: 8000, Metodo: "boleto", VenceEm: time.Now().Add(time.Hour)})
	must(t, err)
	aberto, err := antes.CriarCobranca(ctx, NovaCobranca{Referencia: 4, ValorCentavos: 8000, Metodo: "pix", VenceEm: time.Now().Add(time.Hour)})
	must(t, err)
	must(t, antes.SimularPagamento(ctx, pix.ID))
	must(t, antes.SimularPagamento(ctx, boleto.ID))
	must(t, antes.SimularPagamento(ctx, boleto.ID)) // boleto pago duas vezes: as duas ficam no extrato
	extratoAntes := ids(t, antes)

	s := &Simulador{}
	c, p, err := s.Restaurar(bytes.NewReader(d.bytes()))
	must(t, err)
	if c != 3 || p != 3 || s.CobrancasCriadas.Load() != 3 {
		t.Fatalf("relido: %d cobranças, %d pagamentos (contador %d)", c, p, s.CobrancasCriadas.Load())
	}
	s.Diario = d
	depois := subir(t, s)

	if !slices.Equal(ids(t, depois), extratoAntes) {
		t.Fatalf("extrato depois do reinício %v, antes %v", ids(t, depois), extratoAntes)
	}
	mesma, err := depois.CriarCobranca(ctx, NovaCobranca{Referencia: 1, ValorCentavos: 8000, Metodo: "pix", VenceEm: time.Now().Add(time.Hour)})
	must(t, err)
	if mesma.ID != pix.ID || mesma.PixCopiaCola != pix.PixCopiaCola {
		t.Fatalf("a mesma referência devolveu outra cobrança: %+v, antes %+v", mesma, pix)
	}
	// O Pix pago antes do reinício continua pago; o que estava em aberto continua pagável.
	if err := depois.SimularPagamento(ctx, pix.ID); !errors.Is(err, ErrPixJaPago) {
		t.Fatalf("Pix pago antes do reinício aceitou outro pagamento: %v", err)
	}
	must(t, depois.SimularPagamento(ctx, aberto.ID))
	nova, err := depois.CriarCobranca(ctx, NovaCobranca{Referencia: 3, ValorCentavos: 8000, Metodo: "pix", VenceEm: time.Now().Add(time.Hour)})
	must(t, err)
	if nova.ID == pix.ID || nova.ID == boleto.ID || nova.ID == aberto.ID {
		t.Fatalf("id novo %s repetiu um antigo", nova.ID)
	}
	must(t, depois.SimularPagamento(ctx, nova.ID))
	agora := ids(t, depois)
	if len(agora) != 5 || len(slices.Compact(slices.Sorted(slices.Values(agora)))) != 5 {
		t.Fatalf("extrato %v: esperado 5 pagamentos distintos", agora)
	}

	// Reiniciar de novo, do diário que agora tem as duas vidas.
	s3 := &Simulador{}
	if c, p, err := s3.Restaurar(bytes.NewReader(d.bytes())); err != nil || c != 4 || p != 5 {
		t.Fatalf("segundo reinício: %d cobranças, %d pagamentos, %v", c, p, err)
	}
}

// Um Pix se paga uma vez; o boleto, quantas vezes o candidato pagar. Pagar de novo o Pix (o candidato
// que recarrega a tela do banco) não gera outro pagamento no extrato.
func TestPixSoSePagaUmaVez(t *testing.T) {
	ctx := context.Background()
	c := subir(t, &Simulador{})
	pix, err := c.CriarCobranca(ctx, NovaCobranca{Referencia: 1, ValorCentavos: 8000, Metodo: "pix", VenceEm: time.Now().Add(time.Hour)})
	must(t, err)
	boleto, err := c.CriarCobranca(ctx, NovaCobranca{Referencia: 2, ValorCentavos: 8000, Metodo: "boleto", VenceEm: time.Now().Add(time.Hour)})
	must(t, err)

	var wg sync.WaitGroup
	erros := make(chan error, 20)
	for range 20 { // as repetições ao mesmo tempo: o recarregar que chega junto com o original
		wg.Go(func() { erros <- c.SimularPagamento(ctx, pix.ID) })
	}
	wg.Wait()
	close(erros)
	var ok, recusados int
	for err := range erros {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, ErrPixJaPago):
			recusados++
		default:
			t.Fatal(err)
		}
	}
	if ok != 1 || recusados != 19 {
		t.Fatalf("Pix pago %d vezes, %d recusados; esperado 1 e 19", ok, recusados)
	}
	must(t, c.SimularPagamento(ctx, boleto.ID))
	must(t, c.SimularPagamento(ctx, boleto.ID))
	if n := len(ids(t, c)); n != 3 {
		t.Fatalf("extrato com %d pagamentos; esperado 1 do Pix e 2 do boleto", n)
	}
}

// O extrato devolve exatamente a janela pedida, com as duas pontas, também depois de reler um diário
// fora de ordem (gravado antes de pagar() garantir a ordem).
func TestExtratoDevolveAJanelaPedida(t *testing.T) {
	ctx := context.Background()
	base := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	em := func(min int) time.Time { return base.Add(time.Duration(min) * time.Minute) }
	var linhas strings.Builder
	linhas.WriteString(`{"prefixo":"abc"}` + "\n")
	for _, m := range []int{0, 10, 30, 20, 40} { // o 30 antes do 20
		b, _ := json.Marshal(registro{Pagamento: &Pagamento{ID: fmt.Sprintf("p%02d", m), PagoEm: em(m)}})
		linhas.Write(append(b, '\n'))
	}
	s := &Simulador{}
	_, _, err := s.Restaurar(strings.NewReader(linhas.String()))
	must(t, err)
	c := subir(t, s)
	for _, caso := range []struct {
		desde, ate int
		quer       []string
	}{
		{10, 30, []string{"p10", "p20", "p30"}}, // as duas pontas entram
		{11, 29, []string{"p20"}},
		{-60, 120, []string{"p00", "p10", "p20", "p30", "p40"}},
		{41, 50, nil},
		{30, 10, nil}, // janela invertida
	} {
		ex, err := c.Extrato(ctx, em(caso.desde), em(caso.ate))
		must(t, err)
		var got []string
		for _, p := range ex {
			got = append(got, p.ID)
		}
		if !slices.Equal(got, caso.quer) {
			t.Errorf("extrato de %d a %d: %v, esperado %v", caso.desde, caso.ate, got, caso.quer)
		}
	}
}

// O DICT de teste pelo HTTP: a chave registrada passa a ser do CPF dado, e o CPF tirado deixa de ser
// chave; o corpo sem CPF de 11 dígitos é recusado sem mudar nada.
func TestDictDeTestePeloHTTP(t *testing.T) {
	s := &Simulador{}
	srv := httptest.NewServer(s.Handler())
	t.Cleanup(srv.Close)
	pedir := func(metodo, caminho, corpo string) int {
		r, err := http.NewRequest(metodo, srv.URL+caminho, strings.NewReader(corpo))
		must(t, err)
		resp, err := http.DefaultClient.Do(r)
		must(t, err)
		resp.Body.Close()
		return resp.StatusCode
	}
	if st := pedir("PUT", "/sandbox/dict/ana@exemplo.test", `{"cpf_titular":"12345678909"}`); st != http.StatusNoContent {
		t.Fatalf("registrar: %d", st)
	}
	if st := pedir("PUT", "/sandbox/dict/bia@exemplo.test", `{"cpf_titular":"123"}`); st != http.StatusBadRequest {
		t.Fatalf("CPF curto: %d", st)
	}
	if st := pedir("DELETE", "/sandbox/dict/cpf/99999999999", ""); st != http.StatusNoContent {
		t.Fatalf("tirar a chave CPF: %d", st)
	}
	s.mu.Lock()
	ana, bia, sem, com := s.titular("ana@exemplo.test"), s.titular("bia@exemplo.test"), s.titular("99999999999"), s.titular("12345678909")
	s.mu.Unlock()
	if ana != "12345678909" || bia != "" || sem != "" || com != "12345678909" {
		t.Fatalf("DICT: ana %q, bia %q, sem chave %q, com chave %q", ana, bia, sem, com)
	}
}

// A devolução no prestador: o mesmo id, pedido 20 vezes ao mesmo tempo ou de novo depois de a
// resposta se perder ou de o prestador reiniciar, é uma devolução só; o total nunca passa do pago; o
// boleto só volta para a chave Pix do próprio CPF.
func TestDevolucaoNoPrestador(t *testing.T) {
	ctx := context.Background()
	d := &diario{}
	s := &Simulador{Diario: d}
	s.SemChaveCPF("99999999999")
	s.RegistrarChave("ana@exemplo.test", "12345678909")
	c := subir(t, s)
	vence := time.Now().Add(time.Hour)
	pix, err := c.CriarCobranca(ctx, NovaCobranca{Referencia: 1, ValorCentavos: 8000, Metodo: "pix", VenceEm: vence})
	must(t, err)
	boleto, err := c.CriarCobranca(ctx, NovaCobranca{Referencia: 2, ValorCentavos: 8000, Metodo: "boleto", VenceEm: vence})
	must(t, err)
	must(t, c.SimularPagamento(ctx, pix.ID))
	must(t, c.SimularPagamento(ctx, boleto.ID))
	ex, err := c.Extrato(ctx, time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	must(t, err)
	pagPix, pagBoleto := ex[0].ID, ex[1].ID

	motivo := func(err error) string {
		var r *Recusa
		if errors.As(err, &r) {
			return r.Motivo
		}
		return fmt.Sprint(err)
	}

	// 20 pedidos simultâneos com o mesmo id: uma devolução, a mesma para todos.
	pedido := NovaDevolucao{ID: "10", PagamentoID: pagPix, ValorCentavos: 8000}
	var wg sync.WaitGroup
	ids := make(chan string, 20)
	for range 20 {
		wg.Go(func() {
			dv, err := c.Devolver(ctx, pedido)
			if err != nil {
				t.Error(err)
			}
			ids <- dv.ID
		})
	}
	wg.Wait()
	close(ids)
	var vistos []string
	for id := range ids {
		vistos = append(vistos, id)
	}
	if n := len(slices.Compact(slices.Sorted(slices.Values(vistos)))); n != 1 || s.DevolucoesFeitas.Load() != 1 {
		t.Fatalf("o mesmo id virou %d devoluções diferentes (%d feitas)", n, s.DevolucoesFeitas.Load())
	}

	casos := []struct {
		nome   string
		pedido NovaDevolucao
		motivo string
	}{
		{"outro id, mesmo Pix já devolvido", NovaDevolucao{ID: "11", PagamentoID: pagPix, ValorCentavos: 1}, RecusaValorAcimaDoPago},
		{"mesmo id, outro valor", NovaDevolucao{ID: "10", PagamentoID: pagPix, ValorCentavos: 7000}, RecusaIDReutilizado},
		{"boleto sem chave", NovaDevolucao{ID: "12", PagamentoID: pagBoleto, ValorCentavos: 8000}, RecusaChaveObrigatoria},
		{"CPF que não é chave", NovaDevolucao{ID: "13", PagamentoID: pagBoleto, ValorCentavos: 8000,
			ChavePix: "99999999999", CPFTitular: "99999999999"}, RecusaChaveInexistente},
		{"chave de outro CPF", NovaDevolucao{ID: "14", PagamentoID: pagBoleto, ValorCentavos: 8000,
			ChavePix: "11111111111", CPFTitular: "22222222222"}, RecusaTitularDiferente},
		{"e-mail de outro titular", NovaDevolucao{ID: "18", PagamentoID: pagBoleto, ValorCentavos: 8000,
			ChavePix: "ana@exemplo.test", CPFTitular: "22222222222"}, RecusaTitularDiferente},
		{"e-mail que não é chave", NovaDevolucao{ID: "19", PagamentoID: pagBoleto, ValorCentavos: 8000,
			ChavePix: "bia@exemplo.test", CPFTitular: "12345678909"}, RecusaChaveInexistente},
		{"pagamento que não existe", NovaDevolucao{ID: "15", PagamentoID: "pag_x", ValorCentavos: 8000}, RecusaPagamento},
	}
	for _, k := range casos {
		if _, err := c.Devolver(ctx, k.pedido); motivo(err) != k.motivo {
			t.Errorf("%s: %s; esperado %s", k.nome, motivo(err), k.motivo)
		}
	}

	// A resposta se perde depois de a devolução sair: repetir devolve a mesma.
	s.FalharDevolucoes.Store(1)
	boletoDoCPF := NovaDevolucao{ID: "16", PagamentoID: pagBoleto, ValorCentavos: 8000, ChavePix: "12345678909", CPFTitular: "12345678909"}
	if _, err := c.Devolver(ctx, boletoDoCPF); !errors.Is(err, ErrPrestador) {
		t.Fatalf("resposta perdida: %v; esperado ErrPrestador", err)
	}
	primeira, err := c.Devolver(ctx, boletoDoCPF)
	must(t, err)
	if s.DevolucoesFeitas.Load() != 2 {
		t.Fatalf("%d devoluções feitas; a repetição depois da resposta perdida fez outra", s.DevolucoesFeitas.Load())
	}

	// Reiniciado do diário, o prestador reconhece os ids e os totais de antes.
	s2 := &Simulador{}
	_, _, err = s2.Restaurar(bytes.NewReader(d.bytes()))
	must(t, err)
	c2 := subir(t, s2)
	depois, err := c2.Devolver(ctx, boletoDoCPF)
	must(t, err)
	if depois.ID != primeira.ID || s2.DevolucoesFeitas.Load() != 2 {
		t.Fatalf("depois do reinício: %s (antes %s), %d feitas", depois.ID, primeira.ID, s2.DevolucoesFeitas.Load())
	}
	if _, err := c2.Devolver(ctx, NovaDevolucao{ID: "17", PagamentoID: pagBoleto, ValorCentavos: 1,
		ChavePix: "12345678909", CPFTitular: "12345678909"}); motivo(err) != RecusaValorAcimaDoPago {
		t.Fatalf("depois do reinício, devolveu além do pago: %s", motivo(err))
	}
}

// O processo morto no meio de uma escrita deixa a última linha pela metade: vale até a anterior.
func TestDiarioCortadoValeAteALinhaInteira(t *testing.T) {
	ctx := context.Background()
	d := &diario{}
	c := subir(t, &Simulador{Diario: d})
	for ref := range int64(3) {
		_, err := c.CriarCobranca(ctx, NovaCobranca{Referencia: ref + 1, ValorCentavos: 100, Metodo: "boleto", VenceEm: time.Now().Add(time.Hour)})
		must(t, err)
	}
	b := d.bytes()
	cortado := b[:len(b)-10]
	if n, _, err := (&Simulador{}).Restaurar(bytes.NewReader(cortado)); err != nil || n != 2 {
		t.Fatalf("diário cortado: %d cobranças, %v; esperado 2", n, err)
	}
	if _, _, err := (&Simulador{}).Restaurar(strings.NewReader(`{"seq":1,"chave":"1","cobranca":{"id":"x"}}` + "\n")); err == nil {
		t.Fatal("diário sem prefixo aceito")
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
