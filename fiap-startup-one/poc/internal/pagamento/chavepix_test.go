package pagamento

import (
	"context"
	"errors"
	"testing"

	"lisura/internal/conta"
	"lisura/internal/inscricao"
)

func TestChavePixNormalizada(t *testing.T) {
	cpf := conta.GerarCPF(123456789)
	formatado := cpf[:3] + "." + cpf[3:6] + "." + cpf[6:9] + "-" + cpf[9:]
	validas := map[string]string{
		cpf:                                    cpf,
		formatado:                              cpf,
		"  Ana.Souza@Exemplo.COM ":             "ana.souza@exemplo.com",
		"+55 (11) 98765-4321":                  "+5511987654321",
		"+551133334444":                        "+551133334444",
		"123E4567-E89B-12D3-A456-426614174000": "123e4567-e89b-12d3-a456-426614174000",
	}
	for entrada, quer := range validas {
		if got, err := NormalizarChavePix(entrada); err != nil || got != quer {
			t.Errorf("%q: %q, %v; esperado %q", entrada, got, err, quer)
		}
	}
	invalidas := []string{
		"", "123", cpf[:10] + "0", // 123.456.789-09 com o último dígito trocado
		"11.222.333/0001-81",               // CNPJ: de empresa, nunca do candidato
		"11999998888",                      // telefone sem +55: é lido como CPF, e não é um
		"+1 202 555 0100", "+550987654321", // outro país; DDD começando em 0
		"ana@", "ana @exemplo.com", "Ana <ana@exemplo.com>",
		"123e4567e89b12d3a456426614174000", // aleatória sem os hífens
	}
	for _, entrada := range invalidas {
		if got, err := NormalizarChavePix(entrada); !errors.Is(err, ErrChavePix) {
			t.Errorf("%q aceita como %q", entrada, got)
		}
	}
}

// O CPF do candidato não é chave Pix: a devolução do boleto espera. O candidato informa uma chave de
// outra pessoa: o prestador recusa, nada sai, e a devolução volta a esperar. Informa a dele: o dinheiro
// volta. Depois disso, a chave não muda mais.
func TestCandidatoInformaChaveEDinheiroVolta(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	cpf := c.cpf(t)
	c.sim.SemChaveCPF(cpf)
	pix, boleto := c.cobrar(t, "pix"), c.cobrar(t, "boleto")
	c.pagar(t, pix)
	c.pagar(t, boleto)
	c.conciliar(t)
	c.esvaziar(t)
	var dev int64
	must(t, pool.QueryRow(ctx, `SELECT id FROM devolucao WHERE inscricao_id = $1`, c.inscricao).Scan(&dev))
	if estado, _, _, _ := c.devolucao(t); estado != DevolucaoAguardandoChave {
		t.Fatalf("devolução %s; esperado AGUARDANDO_CHAVE", estado)
	}

	outro := c.vizinho(t)
	if err := InformarChave(ctx, pool, c.inscricao, outro.candidato, dev, "x@exemplo.test"); !errors.Is(err, inscricao.ErrNaoEncontrada) {
		t.Fatalf("outro candidato informou a chave: %v", err)
	}
	if err := InformarChave(ctx, pool, c.inscricao, c.candidato, dev, "11.222.333/0001-81"); !errors.Is(err, inscricao.ErrDadoInvalido) {
		t.Fatalf("CNPJ aceito: %v", err)
	}

	// A chave de outra pessoa (quem tomou a conta tentaria a dele): recusada, e nada sai.
	c.sim.RegistrarChave("golpe@exemplo.test", conta.GerarCPF(987654321))
	must(t, InformarChave(ctx, pool, c.inscricao, c.candidato, dev, "golpe@exemplo.test"))
	c.esvaziar(t)
	vista, err := DevolucoesDaInscricao(ctx, pool, c.inscricao, c.candidato)
	must(t, err)
	if len(vista) != 1 || vista[0].Estado != DevolucaoAguardandoChave || vista[0].Recusa == nil || *vista[0].Recusa != "TITULAR_DIFERENTE" {
		t.Fatalf("chave de outro titular: %+v", vista)
	}
	if n := c.sim.DevolucoesFeitas.Load(); n != 0 {
		t.Fatalf("%d devoluções saíram para a chave de outra pessoa", n)
	}

	// A chave dela, digitada em maiúsculas.
	c.sim.RegistrarChave("ana@exemplo.test", cpf)
	must(t, InformarChave(ctx, pool, c.inscricao, c.candidato, dev, "ANA@exemplo.test"))
	// Pedida e ainda não concluída, a chave já não muda: é o caso que só esta conferência protege (a
	// concluída o banco também barra, pela restrição de estado da tabela).
	if err := InformarChave(ctx, pool, c.inscricao, c.candidato, dev, "golpe@exemplo.test"); !errors.Is(err, ErrNaoAguardaChave) {
		t.Fatalf("trocou a chave de uma devolução a caminho: %v", err)
	}
	c.esvaziar(t)
	if estado, _, _, _ := c.devolucao(t); estado != DevolucaoConcluida || c.sim.DevolucoesFeitas.Load() != 1 {
		t.Fatalf("devolução %s, %d no prestador", estado, c.sim.DevolucoesFeitas.Load())
	}
	if err := InformarChave(ctx, pool, c.inscricao, c.candidato, dev, "outra@exemplo.test"); !errors.Is(err, ErrNaoAguardaChave) {
		t.Fatalf("trocou a chave de uma devolução concluída: %v", err)
	}
}
