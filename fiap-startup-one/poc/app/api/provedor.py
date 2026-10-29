"""Webhook do provedor de pagamento, e o atalho do sandbox para o ensaio."""

import json

from fastapi import APIRouter, Depends, Header, HTTPException, Request
from psycopg.types.json import Jsonb

from app.api.base import conexao, config
from app.config import Config
from app.pagamento import provedor as prov

rotas = APIRouter()


async def _aplicar(conn, nome: str, corpo: bytes) -> dict:
    try:
        ev = json.loads(corpo)
        evento_id, ref, valor = ev["evento_id"], ev["referencia"], int(ev["valor_centavos"])
    except (ValueError, KeyError, TypeError) as e:
        raise HTTPException(400, "evento malformado") from e
    async with conn.transaction():
        cur = await conn.execute("SELECT confirmar_pagamento(%s, %s, %s, %s, %s)",
                                 (nome, evento_id, ref, valor, Jsonb(ev)))
        (resultado,) = await cur.fetchone()
    # 200 para tudo que foi gravado, inclusive divergência: o provedor não deve
    # reenviar um evento que já está registrado e na fila de conciliação.
    return {"evento_id": evento_id, "resultado": resultado}


@rotas.post("/webhooks/pagamento/{nome}")
async def webhook(nome: str, request: Request, conn=Depends(conexao),
                  cfg: Config = Depends(config), x_assinatura: str = Header("")):
    corpo = await request.body()
    if not prov.assinatura_ok(cfg.segredo_provedor, corpo, x_assinatura):
        raise HTTPException(401, "assinatura inválida")
    return await _aplicar(conn, nome, corpo)


@rotas.post("/sandbox/pagar/{referencia}")
async def sandbox_pagar(referencia: str, conn=Depends(conexao), cfg: Config = Depends(config),
                        evento_id: str | None = None):
    """Simula o candidato pagando e o provedor avisando. Só existe com CERTAMIS_SANDBOX=1.
    Passar evento_id repetido simula o reenvio do mesmo aviso."""
    if not cfg.sandbox:
        raise HTTPException(404)
    cur = await conn.execute("SELECT valor_centavos FROM cobranca WHERE referencia = %s",
                             (referencia,))
    row = await cur.fetchone()
    if row is None:
        raise HTTPException(404, "referência desconhecida")
    return await _aplicar(conn, "sandbox", prov.evento_de_pagamento(referencia, row[0], evento_id))
