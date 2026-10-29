"""As garantias de dinheiro sob concorrência real: conexões separadas, ao mesmo tempo.

É aqui que "zero cobrança duplicada" é provado, e não só afirmado. Cada tentativa
abre a sua própria conexão, como fariam requisições paralelas de verdade.
"""

from concurrent.futures import ThreadPoolExecutor

import pytest

from tests.conftest import ErroBanco

N = 30


def _pagante(banco, cpf: str) -> int:
    """Candidato com inscrição finalizada, aguardando pagamento. Devolve o id da inscrição."""
    banco.script(f"""
        SELECT set_config('certamis.ator', 'candidato:{cpf}', false),
               set_config('certamis.motivo', 'teste', false);
        INSERT INTO candidato (cpf, nome, email) VALUES ('{cpf}', 'S', '{cpf}@exemplo.invalid');
        INSERT INTO inscricao (certame_id, candidato_id, cargo_id, cidade_id, modalidade_id,
                               declaracoes_aceitas_em)
        VALUES (1, (SELECT id FROM candidato WHERE cpf = '{cpf}'), 1, 1, 1, now());
        UPDATE inscricao SET estado = 'AGUARDANDO_PAGAMENTO'
         WHERE candidato_id = (SELECT id FROM candidato WHERE cpf = '{cpf}');
    """)
    return int(banco.valor(
        f"SELECT i.id FROM inscricao i JOIN candidato c ON c.id = i.candidato_id WHERE c.cpf = '{cpf}'"))


def _em_paralelo(fn, args):
    with ThreadPoolExecutor(max_workers=len(args)) as ex:
        return list(ex.map(fn, args))


def _invariantes_limpos(banco):
    assert banco.linhas(
        "SELECT tipo, referencia FROM conciliacao"
        " WHERE tipo IN ('INSCRICAO_PAGA_SEM_COBRANCA_PAGA', 'MAIS_DE_UM_EVENTO_APLICADO')") == []
    assert banco.linhas(
        "SELECT inscricao_id FROM cobranca WHERE status IN ('PENDENTE', 'PAGA')"
        " GROUP BY inscricao_id HAVING count(*) > 1") == []


def test_cobranca_pedida_em_paralelo_com_chaves_diferentes_gera_uma_so(banco_com_certame):
    b = banco_com_certame
    iid = _pagante(b, "80000000001")
    ids = _em_paralelo(
        lambda k: b.valor(f"SELECT (gerar_cobranca({iid}, 'chave-paralela-{k:04d}', 'PIX')).id"),
        range(N))
    assert len(set(ids)) == 1
    assert b.valor(f"SELECT count(*) FROM cobranca WHERE inscricao_id = {iid}") == "1"


def test_mesmo_evento_em_paralelo_aplica_uma_vez(banco_com_certame):
    b = banco_com_certame
    iid = _pagante(b, "80000000002")
    ref = b.valor(f"SELECT (gerar_cobranca({iid}, 'chave-unica-0001', 'PIX')).referencia")
    res = _em_paralelo(
        lambda _: b.valor(f"SELECT confirmar_pagamento('sandbox', 'ev-repetido', '{ref}', 8000)"),
        range(N))
    assert res.count("APLICADO") == 1 and res.count("DUPLICADO") == N - 1
    assert b.valor("SELECT count(*) FROM pagamento_evento") == "1"
    _invariantes_limpos(b)


def test_eventos_diferentes_para_a_mesma_cobranca_aplicam_um(banco_com_certame):
    """O provedor reenvia com outro id (acontece em reprocessamento): só o primeiro vale."""
    b = banco_com_certame
    iid = _pagante(b, "80000000003")
    ref = b.valor(f"SELECT (gerar_cobranca({iid}, 'chave-unica-0002', 'PIX')).referencia")
    res = _em_paralelo(
        lambda k: b.valor(f"SELECT confirmar_pagamento('sandbox', 'ev-{k}', '{ref}', 8000)"),
        range(N))
    assert res.count("APLICADO") == 1 and res.count("COBRANCA_INATIVA") == N - 1
    assert b.valor(f"SELECT estado FROM inscricao WHERE id = {iid}") == "PAGA"
    assert b.valor(
        f"SELECT count(*) FROM inscricao_transicao WHERE inscricao_id = {iid} AND para = 'PAGA'") == "1"
    _invariantes_limpos(b)


@pytest.mark.parametrize("rodada", range(3))
def test_cancelamento_disputando_com_pagamento(banco_com_certame, rodada):
    """Para cada inscrição, cancelamento e confirmação chegam juntos. Um dos dois vence,
    sem deadlock, e o pagamento perdedor fica na conciliação em vez de sumir."""
    b = banco_com_certame
    inscricoes = []
    for k in range(10):
        iid = _pagante(b, f"8100000{rodada}{k:03d}")
        ref = b.valor(f"SELECT (gerar_cobranca({iid}, 'chave-disputa-{k:04d}', 'BOLETO')).referencia")
        inscricoes.append((iid, ref))

    def cancelar(par):
        iid, _ = par
        try:
            b.script(f"""SELECT set_config('certamis.ator', 'candidato', false),
                                set_config('certamis.motivo', 'desistiu', false);
                         UPDATE inscricao SET estado = 'CANCELADA' WHERE id = {iid};""")
            return "cancelou"
        except (ErroBanco, AssertionError, Exception) as e:  # CT001: já estava PAGA
            assert "CT001" in str(e) or "não é permitida" in str(e), e
            return "recusado"

    def pagar(par):
        _, ref = par
        return b.valor(f"SELECT confirmar_pagamento('sandbox', 'ev-{ref}', '{ref}', 8000)")

    tarefas = [(cancelar, p) for p in inscricoes] + [(pagar, p) for p in inscricoes]
    with ThreadPoolExecutor(max_workers=len(tarefas)) as ex:
        resultados = list(ex.map(lambda t: t[0](t[1]), tarefas))
    canc, pag = resultados[:10], resultados[10:]

    for (iid, _), c, p in zip(inscricoes, canc, pag):
        estado = b.valor(f"SELECT estado FROM inscricao WHERE id = {iid}")
        if p == "APLICADO":
            assert (estado, c) == ("PAGA", "recusado")
        else:
            assert (estado, c, p) == ("CANCELADA", "cancelou", "COBRANCA_INATIVA")
    # Nenhum pagamento sumiu: todo evento recebido está gravado.
    assert b.valor("SELECT count(*) FROM pagamento_evento") == "10"
    _invariantes_limpos(b)
