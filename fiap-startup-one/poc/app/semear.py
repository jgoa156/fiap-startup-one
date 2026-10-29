"""Gera um certame sintético e contas para o ensaio. Dado de candidato é sempre falso.

Uso: python -m app.semear [contas]

As contas semeadas têm CPF calculado pelo índice (app.cpf.por_indice), a mesma
fórmula do gerador de carga, e todas a mesma senha, para o k6 poder entrar com
elas sem precisar de lista. O hash é calculado uma vez só: é dado sintético.
"""

import sys

from app import cpf, senha
from app.config import carregar

SENHA_SEMEADA = "senha-sintetica-1"


def semear(url: str, contas: int, scrypt_n: int) -> int:
    import psycopg

    with psycopg.connect(url) as c:
        banca = c.execute("INSERT INTO banca (nome) VALUES ('Banca Sintética') RETURNING id").fetchone()[0]
        certame = c.execute(
            "INSERT INTO certame (banca_id, nome, taxa_centavos, isencao_inicio, isencao_fim,"
            " inscricao_inicio, inscricao_fim) VALUES (%s, 'Prefeitura Sintética 2026', 8000,"
            " now() - interval '1 day', now() + interval '2 days',"
            " now() - interval '1 day', now() + interval '29 days') RETURNING id",
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
        for cod, nome, nis, comp in [("CADUNICO", "Inscrito no CadÚnico", True, False),
                                     ("DOADOR_MEDULA", "Doador de medula óssea", False, True)]:
            c.execute("INSERT INTO hipotese_isencao (certame_id, codigo, nome, exige_nis,"
                      " exige_comprovante) VALUES (%s, %s, %s, %s, %s)", (certame, cod, nome, nis, comp))
        h = senha.gerar(SENHA_SEMEADA, n=scrypt_n)
        with c.cursor() as cur:
            with cur.copy("COPY candidato (cpf, nome, email, senha_hash) FROM STDIN") as cp:
                for i in range(contas):
                    cp.write_row((cpf.por_indice(i, 8), f"Semeado {i}", f"semeado-{i}@exemplo.invalid", h))
    print(f"certame {certame} criado; {contas} contas semeadas com a senha '{SENHA_SEMEADA}'")
    return certame


if __name__ == "__main__":
    cfg = carregar()
    semear(cfg.database_url, int(sys.argv[1]) if len(sys.argv) > 1 else 1000, cfg.scrypt_n)
