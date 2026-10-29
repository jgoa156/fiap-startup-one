"""CPF: validação e geração sintética. Dado de candidato na POC é sempre sintético."""

import random


def _digitos(base: list[int]) -> list[int]:
    for n in (10, 11):
        s = sum(d * p for d, p in zip(base, range(n, 1, -1)))
        base = base + [0 if s % 11 < 2 else 11 - s % 11]
    return base


def valido(cpf: str) -> bool:
    if len(cpf) != 11 or not cpf.isdigit() or len(set(cpf)) == 1:
        return False
    d = [int(x) for x in cpf]
    return _digitos(d[:9]) == d


def sintetico(rng: random.Random) -> str:
    """CPF com dígitos válidos, ao acaso. Pode coincidir com um real; nunca sai do ambiente."""
    while True:
        cpf = "".join(map(str, _digitos([rng.randint(0, 9) for _ in range(9)])))
        if valido(cpf):
            return cpf


def por_indice(n: int, prefixo: int = 8) -> str:
    """O mesmo CPF que o gerador de carga calcula para o índice n (carga/ultimo-dia.js).
    Prefixo 8 para contas semeadas, 9 para contas criadas durante o ensaio."""
    base = [int(x) for x in str(prefixo * 100_000_000 + n % 99_999_999).zfill(9)]
    return "".join(map(str, _digitos(base)))
