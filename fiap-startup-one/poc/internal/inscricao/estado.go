// Package inscricao implementa a máquina de estados da inscrição e a sua trilha de auditoria.
package inscricao

type Estado string

const (
	Rascunho            Estado = "RASCUNHO"
	IsencaoAtiva        Estado = "ISENCAO_ATIVA"
	AguardandoPagamento Estado = "AGUARDANDO_PAGAMENTO"
	Paga                Estado = "PAGA"
	Expirada            Estado = "EXPIRADA"
	Cancelada           Estado = "CANCELADA"
	EmAnalise           Estado = "EM_ANALISE"
	Deferida            Estado = "DEFERIDA"
	Indeferida          Estado = "INDEFERIDA"
	Homologada          Estado = "HOMOLOGADA"
)

type Evento string

const (
	FinalizarIsento       Evento = "FINALIZAR_ISENTO"
	FinalizarComPagamento Evento = "FINALIZAR_COM_PAGAMENTO"
	Cancelar              Evento = "CANCELAR"
	ConfirmarPagamento    Evento = "CONFIRMAR_PAGAMENTO"
	Expirar               Evento = "EXPIRAR"
	IniciarAnalise        Evento = "INICIAR_ANALISE"
	Deferir               Evento = "DEFERIR"
	Indeferir             Evento = "INDEFERIR"
	ProverRecurso         Evento = "PROVER_RECURSO"
	Homologar             Evento = "HOMOLOGAR"
	// CancelarCertame é a anulação ou suspensão do certame inteiro, pela banca: leva toda inscrição
	// viva a CANCELADA, e o que foi pago volta (pagamento.CancelarCertame).
	CancelarCertame Evento = "CANCELAR_CERTAME"

	// Criar não é transição: é o evento que abre a trilha, sem estado de origem.
	Criar Evento = "CRIAR"
)

type AtorTipo string

const (
	Candidato AtorTipo = "candidato"
	Analista  AtorTipo = "analista"
	Sistema   AtorTipo = "sistema"
)

type Transicao struct {
	Origens     []Estado
	Destino     Estado
	Ator        AtorTipo // único tipo de ator que pode disparar o evento
	ExigeMotivo bool     // decisão desfavorável ou revertida precisa de motivo padronizado
}

// Transicoes é a máquina de estados de 02-produto-features-e-fluxos.md §5.
// Desvio consciente do diagrama: recurso provido volta para DEFERIDA, e a homologação é um
// evento só, aplicado na publicação da lista definitiva.
var Transicoes = map[Evento]Transicao{
	FinalizarIsento:       {Origens: []Estado{Rascunho}, Destino: IsencaoAtiva, Ator: Candidato},
	FinalizarComPagamento: {Origens: []Estado{Rascunho}, Destino: AguardandoPagamento, Ator: Candidato},
	Cancelar:              {Origens: []Estado{Rascunho}, Destino: Cancelada, Ator: Candidato},
	ConfirmarPagamento:    {Origens: []Estado{AguardandoPagamento}, Destino: Paga, Ator: Sistema},
	Expirar:               {Origens: []Estado{AguardandoPagamento}, Destino: Expirada, Ator: Sistema},
	IniciarAnalise:        {Origens: []Estado{IsencaoAtiva, Paga}, Destino: EmAnalise, Ator: Sistema},
	Deferir:               {Origens: []Estado{EmAnalise}, Destino: Deferida, Ator: Analista},
	Indeferir:             {Origens: []Estado{EmAnalise}, Destino: Indeferida, Ator: Analista, ExigeMotivo: true},
	ProverRecurso:         {Origens: []Estado{Indeferida}, Destino: Deferida, Ator: Analista, ExigeMotivo: true},
	Homologar:             {Origens: []Estado{Deferida}, Destino: Homologada, Ator: Analista},
	// Certame anulado depois da homologação (fraude achada depois da prova) também cancela: é a única
	// saída de HOMOLOGADA. EXPIRADA e CANCELADA continuam sem saída.
	CancelarCertame: {Origens: []Estado{Rascunho, IsencaoAtiva, AguardandoPagamento, Paga, EmAnalise, Deferida, Indeferida, Homologada},
		Destino: Cancelada, Ator: Analista, ExigeMotivo: true},
}

// Proximo devolve o estado de destino se o evento é permitido a partir de atual.
func Proximo(atual Estado, ev Evento) (Estado, bool) {
	t, ok := Transicoes[ev]
	if !ok {
		return "", false
	}
	for _, o := range t.Origens {
		if o == atual {
			return t.Destino, true
		}
	}
	return "", false
}
