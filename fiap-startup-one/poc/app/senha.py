"""Hash de senha com scrypt (biblioteca padrão).

O custo é parâmetro de capacidade: no pico, o login é CPU pura. O ensaio mede
quanto ele pesa, e o valor de n vai para o relatório.
"""

import base64
import hashlib
import hmac
import os


def gerar(senha: str, n: int = 2**14, r: int = 8, p: int = 1) -> str:
    sal = os.urandom(16)
    h = hashlib.scrypt(senha.encode(), salt=sal, n=n, r=r, p=p, dklen=32)
    b64 = lambda b: base64.b64encode(b).decode()
    return f"scrypt${n}${r}${p}${b64(sal)}${b64(h)}"


def confere(senha: str, guardado: str | None) -> bool:
    if not guardado:
        return False
    try:
        alg, n, r, p, sal, h = guardado.split("$")
        if alg != "scrypt":
            return False
        calc = hashlib.scrypt(senha.encode(), salt=base64.b64decode(sal),
                              n=int(n), r=int(r), p=int(p), dklen=32)
        return hmac.compare_digest(calc, base64.b64decode(h))
    except ValueError:
        return False
