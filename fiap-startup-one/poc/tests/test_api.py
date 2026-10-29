"""API e serviço, contra Postgres real. Precisam de fastapi e psycopg instalados."""

import asyncio
import os

import pytest

pytest.importorskip("fastapi")
pytest.importorskip("psycopg_pool")

from fastapi.testclient import TestClient  # noqa: E402
from psycopg_pool import AsyncConnectionPool  # noqa: E402

from app.inscricao import servico  # noqa: E402
from app.inscricao.estados import Estado  # noqa: E402
from tests.conftest import URL  # noqa: E402

DADOS = """
INSERT INTO banca (nome) VALUES ('Banca Sintética');
INSERT INTO certame (banca_id, nome, taxa_centavos, inscricao_inicio, inscricao_fim)
VALUES (1, 'Certame Sintético', 8000, now() - interval '1 day', now() + interval '29 days');
INSERT INTO edital_versao (certame_id, versao, descricao) VALUES (1, 1, 'edital original');
INSERT INTO candidato (cpf, nome, email) VALUES ('00000000191', 'Um', 'um@exemplo.invalid');
"""


@pytest.fixture
def cliente(banco_vazio, monkeypatch):
    banco_vazio.script(DADOS)
    monkeypatch.setenv("CERTAMIS_DATABASE_URL", URL)
    from app.main import app

    with TestClient(app) as c:
        yield c


def test_fluxo_e_trilha_pela_api(cliente):
    h = {"X-Ator": "candidato:00000000191"}
    r = cliente.post("/inscricoes", json={"certame_id": 1, "candidato_id": 1}, headers=h)
    assert r.status_code == 201, r.text
    iid = r.json()["id"]
    assert r.json()["estado"] == "RASCUNHO"

    r = cliente.post(f"/inscricoes/{iid}/transicoes", headers=h,
                     json={"de": "RASCUNHO", "para": "AGUARDANDO_PAGAMENTO", "motivo": "finalizou"})
    assert r.status_code == 200, r.text
    assert r.json()["proximos"] == ["PAGA", "EXPIRADA"]

    trilha = cliente.get(f"/inscricoes/{iid}/trilha").json()
    assert [(t["de"], t["para"]) for t in trilha] == [
        (None, "RASCUNHO"), ("RASCUNHO", "AGUARDANDO_PAGAMENTO")]
    assert all(t["ator"] == "candidato:00000000191" and t["versao_edital"] == 1 for t in trilha)


def test_api_recusa_transicao_invalida_e_exige_ator(cliente):
    h = {"X-Ator": "candidato:00000000191"}
    iid = cliente.post("/inscricoes", json={"certame_id": 1, "candidato_id": 1}, headers=h).json()["id"]
    r = cliente.post(f"/inscricoes/{iid}/transicoes", headers=h,
                     json={"de": "RASCUNHO", "para": "PAGA", "motivo": "pulo"})
    assert r.status_code == 422
    r = cliente.post(f"/inscricoes/{iid}/transicoes",
                     json={"de": "RASCUNHO", "para": "CANCELADA", "motivo": "sem ator"})
    assert r.status_code == 422  # cabeçalho X-Ator ausente
    assert len(cliente.get(f"/inscricoes/{iid}/trilha").json()) == 1


def test_estado_desatualizado_da_409(cliente):
    h = {"X-Ator": "candidato:00000000191"}
    iid = cliente.post("/inscricoes", json={"certame_id": 1, "candidato_id": 1}, headers=h).json()["id"]
    corpo = {"de": "RASCUNHO", "para": "CANCELADA", "motivo": "desistiu"}
    assert cliente.post(f"/inscricoes/{iid}/transicoes", headers=h, json=corpo).status_code == 200
    assert cliente.post(f"/inscricoes/{iid}/transicoes", headers=h, json=corpo).status_code == 409
    assert cliente.get("/inscricoes/999999").status_code == 404


def test_mesma_transicao_concorrente_aplica_uma_vez(banco_vazio):
    """50 requisições simultâneas para a mesma transição: exatamente uma vence."""
    banco_vazio.script(DADOS)

    async def cenario():
        async with AsyncConnectionPool(URL, min_size=10, max_size=10, open=False) as pool:
            await pool.open(wait=True)
            async with pool.connection() as c:
                iid = await servico.criar(c, certame_id=1, candidato_id=1, ator="t")

            async def tentar():
                async with pool.connection() as c:
                    try:
                        await servico.transicionar(c, iid, de=Estado.RASCUNHO,
                                                   para=Estado.AGUARDANDO_PAGAMENTO,
                                                   ator="t", motivo="corrida")
                        return "ok"
                    except servico.ConflitoDeEstado:
                        return "conflito"

            res = await asyncio.gather(*(tentar() for _ in range(50)))
            async with pool.connection() as c:
                t = await servico.trilha(c, iid)
            return res, t

    res, trilha = asyncio.run(cenario())
    assert res.count("ok") == 1 and res.count("conflito") == 49
    assert len(trilha) == 2
