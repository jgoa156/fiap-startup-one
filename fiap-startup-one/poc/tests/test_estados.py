"""A máquina de estados em Python, contra o diagrama de 02-produto §5."""

import pytest

from app.inscricao.estados import FINAIS, TRANSICOES, Estado, TransicaoInvalida, proximos, validar

E = Estado


def test_transicoes_sao_exatamente_as_do_diagrama():
    esperado = {
        (E.RASCUNHO, E.ISENCAO_ATIVA), (E.RASCUNHO, E.AGUARDANDO_PAGAMENTO),
        (E.RASCUNHO, E.CANCELADA), (E.AGUARDANDO_PAGAMENTO, E.PAGA),
        (E.AGUARDANDO_PAGAMENTO, E.EXPIRADA), (E.ISENCAO_ATIVA, E.EM_ANALISE),
        (E.PAGA, E.EM_ANALISE), (E.EM_ANALISE, E.DEFERIDA), (E.EM_ANALISE, E.INDEFERIDA),
        (E.DEFERIDA, E.HOMOLOGADA), (E.INDEFERIDA, E.HOMOLOGADA),
        # decisões de 29/10/2026
        (E.AGUARDANDO_PAGAMENTO, E.CANCELADA), (E.AGUARDANDO_PAGAMENTO, E.RASCUNHO),
    }
    assert set(TRANSICOES) == esperado


@pytest.mark.parametrize("final", sorted(FINAIS))
def test_estado_final_nao_tem_saida(final):
    assert proximos(final) == []


def test_todo_estado_e_alcancavel_a_partir_do_rascunho():
    vistos, fila = {E.RASCUNHO}, [E.RASCUNHO]
    while fila:
        for p in proximos(fila.pop()):
            if p not in vistos:
                vistos.add(p)
                fila.append(p)
    assert vistos == set(E)


def test_todo_estado_nao_final_tem_saida():
    for e in set(E) - FINAIS:
        assert proximos(e), e


@pytest.mark.parametrize("de,para", [
    (E.RASCUNHO, E.PAGA), (E.RASCUNHO, E.HOMOLOGADA), (E.PAGA, E.RASCUNHO),
    (E.HOMOLOGADA, E.EM_ANALISE), (E.EXPIRADA, E.PAGA), (E.CANCELADA, E.RASCUNHO),
    (E.PAGA, E.CANCELADA), (E.PAGA, E.AGUARDANDO_PAGAMENTO), (E.EM_ANALISE, E.RASCUNHO),
])
def test_transicao_invalida_e_recusada(de, para):
    with pytest.raises(TransicaoInvalida):
        validar(de, para)
