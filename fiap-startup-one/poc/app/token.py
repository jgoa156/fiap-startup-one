"""Token de sessão do candidato: HMAC-SHA256 sobre id e validade. Sem estado no servidor,
para o login não custar uma escrita no banco no pico."""

import base64
import hashlib
import hmac
import time


class TokenInvalido(Exception):
    pass


def _b64(b: bytes) -> str:
    return base64.urlsafe_b64encode(b).decode().rstrip("=")


def _assinatura(segredo: str, corpo: str) -> str:
    return _b64(hmac.new(segredo.encode(), corpo.encode(), hashlib.sha256).digest())


def emitir(segredo: str, candidato_id: int, validade_s: int = 4 * 3600, agora: float | None = None) -> str:
    expira = int((agora if agora is not None else time.time()) + validade_s)
    corpo = f"{candidato_id}.{expira}"
    return f"{corpo}.{_assinatura(segredo, corpo)}"


def verificar(segredo: str, token: str, agora: float | None = None) -> int:
    try:
        cid, expira, assinatura = token.split(".")
        corpo = f"{cid}.{expira}"
        if not hmac.compare_digest(assinatura, _assinatura(segredo, corpo)):
            raise TokenInvalido("assinatura")
        if int(expira) < (agora if agora is not None else time.time()):
            raise TokenInvalido("expirado")
        return int(cid)
    except ValueError as e:
        raise TokenInvalido("formato") from e
