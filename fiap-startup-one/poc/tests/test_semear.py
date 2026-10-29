import random

from app.semear import cpf_sintetico


def _dv_ok(cpf: str) -> bool:
    d = [int(x) for x in cpf]
    for n in (9, 10):
        s = sum(d[i] * (n + 1 - i) for i in range(n))
        if d[n] != (0 if s % 11 < 2 else 11 - s % 11):
            return False
    return True


def test_verificador_de_referencia():
    assert _dv_ok("52998224725") and not _dv_ok("52998224726")


def test_cpf_sintetico_tem_digitos_validos():
    rng = random.Random(1)
    assert all(_dv_ok(cpf_sintetico(rng)) for _ in range(2000))
