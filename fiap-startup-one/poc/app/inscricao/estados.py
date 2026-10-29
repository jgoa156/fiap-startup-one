"""Máquina de estados da inscrição (02-produto-features-e-fluxos.md §5, mais as
decisões de 29/10/2026 registradas nas migrações 0003 e 0004).

Esta é a cópia da regra que a aplicação usa para responder rápido e com
mensagem clara. A cópia que vale é a do banco (migrações 0002 a 0004), que recusa a
transição mesmo que alguém pule esta camada. Um teste garante que as duas são
iguais.
"""

from enum import StrEnum


class Estado(StrEnum):
    RASCUNHO = "RASCUNHO"
    ISENCAO_ATIVA = "ISENCAO_ATIVA"
    AGUARDANDO_PAGAMENTO = "AGUARDANDO_PAGAMENTO"
    PAGA = "PAGA"
    EXPIRADA = "EXPIRADA"
    CANCELADA = "CANCELADA"
    EM_ANALISE = "EM_ANALISE"
    DEFERIDA = "DEFERIDA"
    INDEFERIDA = "INDEFERIDA"
    INDEFERIDA_DEFINITIVA = "INDEFERIDA_DEFINITIVA"
    HOMOLOGADA = "HOMOLOGADA"


FINAIS = frozenset({
    Estado.EXPIRADA, Estado.CANCELADA, Estado.HOMOLOGADA, Estado.INDEFERIDA_DEFINITIVA,
})

# (de, para) -> evento
TRANSICOES: dict[tuple[Estado, Estado], str] = {
    (Estado.RASCUNHO, Estado.ISENCAO_ATIVA): "finaliza_isento",
    (Estado.RASCUNHO, Estado.AGUARDANDO_PAGAMENTO): "finaliza_pagante",
    (Estado.RASCUNHO, Estado.CANCELADA): "cancela",
    (Estado.AGUARDANDO_PAGAMENTO, Estado.PAGA): "confirma_pagamento",
    (Estado.AGUARDANDO_PAGAMENTO, Estado.EXPIRADA): "vence_prazo",
    # Decisões de 29/10/2026, migração 0003.
    (Estado.AGUARDANDO_PAGAMENTO, Estado.CANCELADA): "cancela",
    (Estado.AGUARDANDO_PAGAMENTO, Estado.RASCUNHO): "reabre_para_edicao",
    (Estado.ISENCAO_ATIVA, Estado.EM_ANALISE): "envia_para_analise",
    (Estado.PAGA, Estado.EM_ANALISE): "envia_para_analise",
    (Estado.EM_ANALISE, Estado.DEFERIDA): "defere",
    (Estado.EM_ANALISE, Estado.INDEFERIDA): "indefere",
    (Estado.DEFERIDA, Estado.HOMOLOGADA): "homologa",
    (Estado.INDEFERIDA, Estado.HOMOLOGADA): "provê_recurso",
    # Decisão de 29/10/2026, migração 0004.
    (Estado.INDEFERIDA, Estado.INDEFERIDA_DEFINITIVA): "nega_recurso",
}


class TransicaoInvalida(Exception):
    def __init__(self, de: Estado, para: Estado):
        self.de, self.para = de, para
        super().__init__(f"transição {de} -> {para} não é permitida")


def permitida(de: Estado, para: Estado) -> bool:
    return (de, para) in TRANSICOES


def validar(de: Estado, para: Estado) -> str:
    """Devolve o nome do evento ou levanta TransicaoInvalida."""
    try:
        return TRANSICOES[(de, para)]
    except KeyError:
        raise TransicaoInvalida(de, para) from None


def proximos(de: Estado) -> list[Estado]:
    return [p for (d, p) in TRANSICOES if d == de]
