"""Gera um certame sintético para desenvolvimento. Dado de candidato é sempre falso.

Uso: python -m app.semear [quantidade_de_candidatos]
"""

import random
import sys

from app.config import carregar


def cpf_sintetico(rng: random.Random) -> str:
    """CPF com dígitos verificadores válidos, gerado ao acaso. Não é de ninguém de propósito,
    mas pode coincidir com um real; por isso nunca sai deste ambiente."""
    base = [rng.randint(0, 9) for _ in range(9)]
    for n in (10, 11):
        s = sum(d * p for d, p in zip(base, range(n, 1, -1)))
        base.append(0 if s % 11 < 2 else 11 - s % 11)
    return "".join(map(str, base))


def semear(url: str, n: int, semente: int = 42) -> None:
    import psycopg

    rng = random.Random(semente)
    with psycopg.connect(url) as c:
        banca = c.execute("INSERT INTO banca (nome) VALUES ('Banca Sintética') RETURNING id").fetchone()[0]
        certame = c.execute(
            "INSERT INTO certame (banca_id, nome, taxa_centavos, isencao_inicio, isencao_fim,"
            " inscricao_inicio, inscricao_fim) VALUES (%s, 'Prefeitura Sintética 2026', 8000,"
            " now() - interval '10 days', now() - interval '3 days',"
            " now() - interval '2 days', now() + interval '28 days') RETURNING id",
            (banca,),
        ).fetchone()[0]
        c.execute("INSERT INTO edital_versao (certame_id, versao, descricao)"
                  " VALUES (%s, 1, 'edital original')", (certame,))
        for nome, vagas in [("Agente Administrativo", 40), ("Professor", 25), ("Analista", 10)]:
            c.execute("INSERT INTO cargo (certame_id, nome, vagas) VALUES (%s, %s, %s)",
                      (certame, nome, vagas))
        for cidade in ["Sede", "Distrito Norte"]:
            c.execute("INSERT INTO cidade_prova (certame_id, nome) VALUES (%s, %s)", (certame, cidade))
        for cod, nome in [("AMPLA", "Ampla concorrência"), ("PCD", "Pessoa com deficiência"),
                          ("COTA_RACIAL", "Reserva racial")]:
            c.execute("INSERT INTO modalidade (certame_id, codigo, nome) VALUES (%s, %s, %s)",
                      (certame, cod, nome))
        cpfs = {cpf_sintetico(rng) for _ in range(n)}
        with c.cursor() as cur:
            with cur.copy("COPY candidato (cpf, nome, email) FROM STDIN") as cp:
                for i, cpf in enumerate(sorted(cpfs)):
                    cp.write_row((cpf, f"Candidato Sintético {i}", f"c{i}@exemplo.invalid"))
    print(f"certame {certame} criado com {len(cpfs)} candidatos sintéticos")


if __name__ == "__main__":
    semear(carregar().database_url, int(sys.argv[1]) if len(sys.argv) > 1 else 1000)
