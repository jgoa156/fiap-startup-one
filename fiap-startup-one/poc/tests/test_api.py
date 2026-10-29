"""A API de ponta a ponta, contra Postgres real. Precisa de fastapi e psycopg instalados.

Roda o worker em linha (uma passada) onde o fluxo depende dele.
"""

import os
import tempfile
import time
import uuid

import pytest

pytest.importorskip("fastapi")
pytest.importorskip("psycopg_pool")

from tests.conftest import URL, Banco  # noqa: E402

PASTA = tempfile.mkdtemp(prefix="certamis-teste-")
os.environ.update({
    "CERTAMIS_DATABASE_URL": URL,
    "CERTAMIS_PASTA_ARQUIVOS": PASTA,
    "CERTAMIS_SCRYPT_N": str(2**10),  # rápido nos testes; o custo real é medido no ensaio
    "CERTAMIS_SANDBOX": "1",
})

from fastapi.testclient import TestClient  # noqa: E402

from app import cpf as cpf_mod  # noqa: E402
from app import worker  # noqa: E402
from app.config import carregar  # noqa: E402
from app.pagamento import provedor  # noqa: E402

BANCA = {"X-Ator-Banca": "analista-teste"}


@pytest.fixture
def api(banco_com_certame):
    from app.main import app

    with TestClient(app) as c:
        yield c


def _conta(api, seed: int) -> tuple[dict, str]:
    import random

    cpf = cpf_mod.sintetico(random.Random(seed))
    r = api.post("/candidatos", json={"cpf": cpf, "nome": f"Sintético {seed}",
                                     "email": f"c{seed}@exemplo.invalid", "senha": "senha-forte-1"})
    assert r.status_code == 201, r.text
    r = api.post("/sessoes", json={"cpf": cpf, "senha": "senha-forte-1"})
    assert r.status_code == 200, r.text
    return {"Authorization": f"Bearer {r.json()['token']}"}, cpf


def _inscricao_completa(api, h) -> int:
    r = api.post("/inscricoes", json={"certame_id": 1}, headers=h)
    assert r.status_code == 201, r.text
    iid = r.json()["id"]
    r = api.patch(f"/inscricoes/{iid}", headers=h, json={
        "cargo_id": 1, "cidade_id": 1, "modalidade_id": 1, "atendimento_especial": True,
        "aceitar_declaracoes": True})
    assert r.status_code == 200, r.text
    return iid


def test_pagante_do_cadastro_ao_cartao(api):
    h, _ = _conta(api, 1)
    assert api.get("/certames/1").json()["cargos"][0]["nome"] == "Analista"

    iid = _inscricao_completa(api, h)
    r = api.put(f"/inscricoes/{iid}/laudo", headers={**h, "Content-Type": "application/pdf"},
                content=b"%PDF-1.4 laudo sintetico")
    assert r.status_code == 201, r.text
    # Regressão: a gravação depois de uma leitura tem de persistir (pool em autocommit).
    assert Banco(URL).valor(f"SELECT count(*) FROM documento WHERE inscricao_id = {iid}") == "1"

    r = api.post(f"/inscricoes/{iid}/finalizar", headers=h)
    assert r.status_code == 200 and r.json()["estado"] == "AGUARDANDO_PAGAMENTO", r.text
    # Fora do rascunho, nada muda.
    assert api.patch(f"/inscricoes/{iid}", headers=h, json={"cargo_id": 1}).status_code == 409

    k1, k2 = str(uuid.uuid4()), str(uuid.uuid4())
    c1 = api.post(f"/inscricoes/{iid}/cobranca", headers={**h, "Idempotency-Key": k1}, json={"meio": "BOLETO"})
    c1b = api.post(f"/inscricoes/{iid}/cobranca", headers={**h, "Idempotency-Key": k1}, json={"meio": "BOLETO"})
    c2 = api.post(f"/inscricoes/{iid}/cobranca", headers={**h, "Idempotency-Key": k2}, json={"meio": "PIX"})
    assert c1.status_code == 200, c1.text
    assert c1.json()["id"] == c1b.json()["id"] == c2.json()["id"]

    # O registro no provedor acontece no worker, fora do request.
    assert api.get(f"/inscricoes/{iid}/cobranca", headers=h).json()["instrucoes"] is None
    worker.rodar(uma_vez=True)
    assert len(api.get(f"/inscricoes/{iid}/cobranca", headers=h).json()["instrucoes"]["linha_digitavel"]) == 47

    ref = c1.json()["referencia"]
    assert api.post(f"/sandbox/pagar/{ref}?evento_id=ev-a").json()["resultado"] == "APLICADO"
    assert api.post(f"/sandbox/pagar/{ref}?evento_id=ev-a").json()["resultado"] == "DUPLICADO"
    assert api.post(f"/sandbox/pagar/{ref}?evento_id=ev-b").json()["resultado"] == "COBRANCA_INATIVA"
    assert api.get(f"/inscricoes/{iid}", headers=h).json()["estado"] == "PAGA"
    assert api.post(f"/inscricoes/{iid}/cancelar", headers=h).status_code == 409

    for de, para in [("PAGA", "EM_ANALISE"), ("EM_ANALISE", "DEFERIDA"), ("DEFERIDA", "HOMOLOGADA")]:
        r = api.post(f"/banca/inscricoes/{iid}/transicoes", headers=BANCA,
                     json={"de": de, "para": para, "motivo": "teste de ponta a ponta"})
        assert r.status_code == 200, r.text

    assert api.get(f"/inscricoes/{iid}/cartao", headers=h).status_code == 202
    worker.rodar(uma_vez=True)
    r = api.get(f"/inscricoes/{iid}/cartao", headers=h)
    assert r.status_code == 200 and r.content.startswith(b"%PDF")

    assert api.post("/banca/certames/1/publicar-homologacao", headers=BANCA).status_code == 202
    worker.rodar(uma_vez=True)
    lista = api.get("/publico/certames/1/homologacao.json").json()
    assert [l["inscricao"] for l in lista["homologadas"]] == [iid]
    assert api.get(f"/publico/certames/1/situacao/{iid // 1000}.json").json()["situacao"][str(iid)] == "HOMOLOGADA"

    trilha = api.get(f"/inscricoes/{iid}/trilha", headers=h).json()
    assert [t["para"] for t in trilha] == ["RASCUNHO", "AGUARDANDO_PAGAMENTO", "PAGA", "EM_ANALISE",
                                           "DEFERIDA", "HOMOLOGADA"]
    assert trilha[2]["ator"] == "provedor:sandbox"
    assert api.get("/banca/conciliacao", headers=BANCA).json()[0]["detalhe"] == "ev-b"


def test_isencao_pela_triagem_e_pela_banca(api):
    h, _ = _conta(api, 2)
    r = api.post("/certames/1/isencao", headers=h, json={"hipotese": "CADUNICO", "nis": "12034567898"})
    assert r.status_code == 201 and r.json()["estado"] == "INDEFERIDA", r.text
    assert "dígito verificador" in r.json()["motivo"]

    h2, _ = _conta(api, 3)
    r = api.post("/certames/1/isencao", headers=h2, json={"hipotese": "CADUNICO", "nis": "12034567899"})
    assert r.json()["estado"] == "PENDENTE_ANALISE"
    r = api.post(f"/banca/isencoes/{r.json()['id']}/decisao", headers=BANCA,
                 json={"deferir": True, "motivo": "CadÚnico conferido"})
    assert r.json()["estado"] == "DEFERIDA"

    iid = _inscricao_completa(api, h2)
    assert api.post(f"/inscricoes/{iid}/finalizar", headers=h2).json()["estado"] == "ISENCAO_ATIVA"
    r = api.post(f"/inscricoes/{iid}/cobranca", headers={**h2, "Idempotency-Key": "chave-isento-1"},
                 json={"meio": "PIX"})
    assert r.status_code == 409 and r.json()["codigo"] == "CT010"


def test_reabrir_e_cancelar_persistem(api):
    h, _ = _conta(api, 8)
    iid = _inscricao_completa(api, h)
    api.post(f"/inscricoes/{iid}/finalizar", headers=h)
    api.post(f"/inscricoes/{iid}/cobranca", headers={**h, "Idempotency-Key": "chave-reabrir-1"},
             json={"meio": "PIX"})
    assert api.post(f"/inscricoes/{iid}/reabrir", headers=h).status_code == 200
    assert api.get(f"/inscricoes/{iid}", headers=h).json()["estado"] == "RASCUNHO"
    assert api.get(f"/inscricoes/{iid}/cobranca", headers=h).json()["status"] == "CANCELADA"
    api.post(f"/inscricoes/{iid}/finalizar", headers=h)
    assert api.post(f"/inscricoes/{iid}/cancelar", headers=h).status_code == 200
    assert api.get(f"/inscricoes/{iid}", headers=h).json()["estado"] == "CANCELADA"


def test_formulario_incompleto_e_dono_da_inscricao(api):
    h, _ = _conta(api, 4)
    r = api.post("/inscricoes", json={"certame_id": 1}, headers=h)
    iid = r.json()["id"]
    r = api.post(f"/inscricoes/{iid}/finalizar", headers=h)
    assert r.status_code == 422 and r.json()["codigo"] == "CT006"

    outro, _ = _conta(api, 5)
    assert api.get(f"/inscricoes/{iid}", headers=outro).status_code == 404
    assert api.get(f"/inscricoes/{iid}").status_code == 422  # sem Authorization
    assert api.get(f"/inscricoes/{iid}", headers={"Authorization": "Bearer x.y.z"}).status_code == 401


def test_conta_unica_por_cpf_e_cpf_invalido(api):
    _conta(api, 6)
    import random

    cpf = cpf_mod.sintetico(random.Random(6))
    r = api.post("/candidatos", json={"cpf": cpf, "nome": "Outro", "email": "outro@exemplo.invalid",
                                      "senha": "senha-forte-1"})
    assert r.status_code == 409
    r = api.post("/candidatos", json={"cpf": "52998224726", "nome": "Inválido",
                                      "email": "inv@exemplo.invalid", "senha": "senha-forte-1"})
    assert r.status_code == 422
    assert api.post("/sessoes", json={"cpf": cpf, "senha": "errada-123"}).status_code == 401


def test_webhook_exige_assinatura(api):
    corpo = provedor.evento_de_pagamento("CTM000000000999", 8000, "ev-x")
    assert api.post("/webhooks/pagamento/sandbox", content=corpo,
                    headers={"X-Assinatura": "errada"}).status_code == 401
    sig = provedor.assinar(carregar().segredo_provedor, corpo)
    r = api.post("/webhooks/pagamento/sandbox", content=corpo, headers={"X-Assinatura": sig})
    assert r.status_code == 200 and r.json()["resultado"] == "REFERENCIA_DESCONHECIDA"


def _esperar_nivel(api, nivel: int):
    for _ in range(30):
        if api.get("/status").json()["nivel"] == nivel:
            return
        time.sleep(0.1)
    raise AssertionError(f"nível {nivel} não chegou")


def test_modo_degradado_e_fila_de_espera(api):
    h, cpf = _conta(api, 7)
    try:
        # Nível 4: só estático. Escrita recusada com 503, status e arquivos públicos no ar.
        api.post("/banca/operacao", headers=BANCA, json={"nivel": 4, "mensagem": "teste"})
        _esperar_nivel(api, 4)
        r = api.post("/inscricoes", json={"certame_id": 1}, headers=h)
        assert r.status_code == 503 and r.json()["mensagem"] == "teste"
        assert api.get("/status").status_code == 200

        # Nível 1: login só com passe da fila. Quem já tem sessão segue.
        api.post("/banca/operacao", headers=BANCA, json={"nivel": 1, "taxa_por_s": 50})
        _esperar_nivel(api, 1)
        assert api.post("/sessoes", json={"cpf": cpf, "senha": "senha-forte-1"}).status_code == 429
        senha = api.post("/fila").json()["senha"]
        for _ in range(30):
            r = api.get(f"/fila/{senha}").json()
            if r["admitido"]:
                break
            time.sleep(0.2)
        assert r["admitido"], r
        r = api.post("/sessoes", json={"cpf": cpf, "senha": "senha-forte-1"},
                     headers={"X-Passe": r["passe"]})
        assert r.status_code == 200
        assert api.get(f"/fila/{senha}x").status_code == 400
    finally:
        api.post("/banca/operacao", headers=BANCA, json={"nivel": 0})
