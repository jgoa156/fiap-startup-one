"""Modo degradado e fila de espera.

O estado (nível e portão) mora no banco, mas é lido no máximo uma vez por segundo
por instância: no pico, consultar a posição na fila não pode custar uma ida ao
banco por requisição, senão a fila vira a carga que ela deveria conter.
"""

import asyncio
import base64
import hashlib
import hmac
import logging
import time
from dataclasses import dataclass

log = logging.getLogger("certamis.operacao")

NORMAL, FILA, SO_ESTATICO, STATUS = 0, 1, 4, 5
TRAVA_PORTAO = 0x706F7274  # "port"


@dataclass
class Estado:
    nivel: int = NORMAL
    mensagem: str = ""
    admitido_ate: int = 0
    taxa_por_s: float = 5.0
    lido_em: float = 0.0


def _assinar(segredo: str, corpo: str) -> str:
    mac = hmac.new(segredo.encode(), corpo.encode(), hashlib.sha256).digest()
    return base64.urlsafe_b64encode(mac[:16]).decode().rstrip("=")


def senha_emitir(segredo: str, numero: int) -> str:
    corpo = f"s.{numero}"
    return f"{corpo}.{_assinar(segredo, corpo)}"


def senha_ler(segredo: str, senha: str) -> int | None:
    try:
        tipo, numero, assinatura = senha.split(".")
        if tipo != "s" or not hmac.compare_digest(assinatura, _assinar(segredo, f"s.{numero}")):
            return None
        return int(numero)
    except ValueError:
        return None


def passe_emitir(segredo: str, numero: int, validade_s: int, agora: float | None = None) -> str:
    expira = int((agora if agora is not None else time.time()) + validade_s)
    corpo = f"p.{numero}.{expira}"
    return f"{corpo}.{_assinar(segredo, corpo)}"


def passe_valido(segredo: str, passe: str, agora: float | None = None) -> bool:
    try:
        tipo, numero, expira, assinatura = passe.split(".")
        corpo = f"p.{numero}.{expira}"
        return (tipo == "p" and hmac.compare_digest(assinatura, _assinar(segredo, corpo))
                and int(expira) >= (agora if agora is not None else time.time()))
    except ValueError:
        return False


class Operacao:
    def __init__(self, pool):
        self.pool = pool
        self.estado = Estado()

    async def atualizar(self) -> None:
        async with self.pool.connection() as conn:
            async with conn.transaction():
                cur = await conn.execute("SELECT pg_try_advisory_xact_lock(%s)", (TRAVA_PORTAO,))
                (dono,) = await cur.fetchone()
                if dono:
                    await conn.execute("SELECT avancar_portao()")
                cur = await conn.execute(
                    "SELECT m.nivel, m.mensagem, p.admitido_ate, p.taxa_por_s"
                    " FROM modo_operacao m, portao p")
                nivel, msg, adm, taxa = await cur.fetchone()
        self.estado = Estado(nivel, msg, adm, float(taxa), time.monotonic())

    async def laco(self, intervalo_s: float = 1.0) -> None:
        while True:
            try:
                await self.atualizar()
            except Exception:  # o laço não pode morrer: sem ele o portão para
                log.exception("falha ao atualizar o modo de operação")
            await asyncio.sleep(intervalo_s)

    def posicao(self, numero: int) -> tuple[int, float]:
        falta = max(0, numero - self.estado.admitido_ate)
        return falta, falta / self.estado.taxa_por_s
