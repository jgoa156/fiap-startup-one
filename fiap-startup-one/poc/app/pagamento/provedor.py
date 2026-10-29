"""Provedor de pagamento: interface e o simulador de sandbox.

Orquestramos, não processamos. Na produção, cada banca tem conta no provedor dela
e o dinheiro nunca passa por nós. Na POC não há conta em provedor real: o
simulador faz o papel dele, com as duas coisas que importam para o ensaio:
registrar a cobrança de forma idempotente pela nossa referência, e avisar o
pagamento por webhook assinado, às vezes repetido, como provedores reais fazem.
"""

import hashlib
import hmac
import json
import random
import uuid
from typing import Protocol


def assinar(segredo: str, corpo: bytes) -> str:
    return hmac.new(segredo.encode(), corpo, hashlib.sha256).hexdigest()


def assinatura_ok(segredo: str, corpo: bytes, assinatura: str) -> bool:
    return hmac.compare_digest(assinar(segredo, corpo), assinatura or "")


class Provedor(Protocol):
    nome: str

    def registrar(self, referencia: str, meio: str, valor_centavos: int, vence_em: str) -> dict:
        """Registra a cobrança no provedor. Repetir com a mesma referência devolve o mesmo
        registro: a referência é o txid do Pix ou o "seu número" do boleto."""
        ...


class Sandbox:
    nome = "sandbox"

    def registrar(self, referencia: str, meio: str, valor_centavos: int, vence_em: str) -> dict:
        # Determinístico pela referência: registrar duas vezes dá o mesmo resultado.
        semente = int(hashlib.sha256(referencia.encode()).hexdigest()[:12], 16)
        rng = random.Random(semente)
        if meio == "PIX":
            return {"pix_copia_e_cola": f"00020126SANDBOX{referencia}{valor_centavos:010d}"}
        if meio == "BOLETO":
            linha = "".join(str(rng.randint(0, 9)) for _ in range(47))
            return {"linha_digitavel": linha, "vencimento": vence_em}
        return {"url_pagamento": f"https://sandbox.invalid/cartao/{referencia}"}


def evento_de_pagamento(referencia: str, valor_centavos: int, evento_id: str | None = None) -> bytes:
    """Corpo do webhook que o provedor mandaria. O id do evento é do provedor."""
    return json.dumps({
        "evento_id": evento_id or f"sbx-{uuid.uuid4()}",
        "tipo": "pagamento.confirmado",
        "referencia": referencia,
        "valor_centavos": valor_centavos,
    }).encode()
