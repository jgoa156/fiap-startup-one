"""Módulos puros: sem banco e sem rede."""

import json
import random

import pytest

from app import cpf, operacao, pdf, senha, token
from app.armazenamento import DiscoLocal
from app.pagamento import provedor


# ------------------------------------------------------------------ CPF

def test_cpf_de_referencia():
    assert cpf.valido("52998224725")
    assert not cpf.valido("52998224726")
    assert not cpf.valido("11111111111") and not cpf.valido("123") and not cpf.valido("5299822472a")


def test_cpf_sintetico_sempre_valido():
    rng = random.Random(1)
    assert all(cpf.valido(cpf.sintetico(rng)) for _ in range(2000))


# ------------------------------------------------------------------ senha e token

def test_senha_confere_e_recusa():
    h = senha.gerar("correta-123", n=2**10)
    assert senha.confere("correta-123", h)
    assert not senha.confere("errada-123", h)
    assert not senha.confere("x", None) and not senha.confere("x", "lixo")
    assert senha.gerar("igual", n=2**10) != senha.gerar("igual", n=2**10)  # sal novo


def test_token_valido_adulterado_e_vencido():
    t = token.emitir("s", 42, validade_s=60, agora=1000)
    assert token.verificar("s", t, agora=1030) == 42
    with pytest.raises(token.TokenInvalido):
        token.verificar("s", t, agora=1061)
    with pytest.raises(token.TokenInvalido):
        token.verificar("outro-segredo", t, agora=1030)
    cid, exp, sig = t.split(".")
    with pytest.raises(token.TokenInvalido):
        token.verificar("s", f"43.{exp}.{sig}", agora=1030)  # trocou o candidato
    with pytest.raises(token.TokenInvalido):
        token.verificar("s", "nao-e-token", agora=1030)


# ------------------------------------------------------------------ fila de espera

def test_senha_e_passe_da_fila():
    s = operacao.senha_emitir("k", 77)
    assert operacao.senha_ler("k", s) == 77
    assert operacao.senha_ler("k", s.replace("77", "1")) is None
    assert operacao.senha_ler("outra", s) is None
    p = operacao.passe_emitir("k", 77, validade_s=10, agora=100)
    assert operacao.passe_valido("k", p, agora=105)
    assert not operacao.passe_valido("k", p, agora=111)
    assert not operacao.passe_valido("k", s, agora=105)  # senha não é passe


def test_posicao_e_espera():
    o = operacao.Operacao(pool=None)
    o.estado = operacao.Estado(admitido_ate=100, taxa_por_s=5)
    assert o.posicao(90) == (0, 0)
    assert o.posicao(150) == (50, 10)


# ------------------------------------------------------------------ PDF

def test_pdf_abre_e_tem_o_texto():
    pypdf = pytest.importorskip("pypdf")
    import io

    b = pdf.pagina([(18, "Cartão de confirmação"), (11, "Inscrição nº 7 (ação)")])
    texto = pypdf.PdfReader(io.BytesIO(b)).pages[0].extract_text()
    assert "Cartão de confirmação" in texto and "Inscrição nº 7 (ação)" in texto


# ------------------------------------------------------------------ armazenamento

def test_disco_local_deduplica_e_publica(tmp_path):
    arm = DiscoLocal(tmp_path)
    c1, h1 = arm.guardar("laudo", b"conteudo")
    c2, h2 = arm.guardar("laudo", b"conteudo")
    assert (c1, h1) == (c2, h2) and arm.ler(c1) == b"conteudo"
    assert len(list((tmp_path / "privado").rglob("*"))) == 3  # laudo/, xx/, arquivo
    arm.publicar("certames/1/homologacao.json", b"{}", "application/json")
    assert (tmp_path / "publico/certames/1/homologacao.json").read_bytes() == b"{}"
    assert not list(tmp_path.rglob("*.parcial"))


# ------------------------------------------------------------------ provedor

def test_sandbox_registra_de_forma_idempotente():
    sb = provedor.Sandbox()
    a = sb.registrar("CTM000000000001", "BOLETO", 8000, "2026-12-01")
    assert a == sb.registrar("CTM000000000001", "BOLETO", 8000, "2026-12-01")
    assert a != sb.registrar("CTM000000000002", "BOLETO", 8000, "2026-12-01")
    assert len(a["linha_digitavel"]) == 47


def test_webhook_assinado():
    corpo = provedor.evento_de_pagamento("CTM000000000001", 8000, "ev-1")
    assert json.loads(corpo)["evento_id"] == "ev-1"
    sig = provedor.assinar("seg", corpo)
    assert provedor.assinatura_ok("seg", corpo, sig)
    assert not provedor.assinatura_ok("seg", corpo + b" ", sig)
    assert not provedor.assinatura_ok("outro", corpo, sig)
    assert not provedor.assinatura_ok("seg", corpo, "")
