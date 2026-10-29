"""Aplica as migrações SQL de app/migrations em ordem, cada uma numa transação.

Uso: python -m app.migrar
"""

import sys
from pathlib import Path

import psycopg

from app.config import carregar

PASTA = Path(__file__).parent / "migrations"
# Trava de sessão para que duas instâncias subindo juntas não migrem em paralelo.
TRAVA = 0x6365727461  # "certa"


def migrar(database_url: str) -> list[str]:
    aplicadas: list[str] = []
    with psycopg.connect(database_url, autocommit=True) as conn:
        conn.execute("SELECT pg_advisory_lock(%s)", (TRAVA,))
        try:
            conn.execute(
                "CREATE TABLE IF NOT EXISTS schema_migracao ("
                " nome text PRIMARY KEY, aplicada_em timestamptz NOT NULL DEFAULT now())"
            )
            feitas = {r[0] for r in conn.execute("SELECT nome FROM schema_migracao")}
            for arquivo in sorted(PASTA.glob("*.sql")):
                if arquivo.name in feitas:
                    continue
                with conn.transaction():
                    conn.execute(arquivo.read_text(encoding="utf-8"))
                    conn.execute(
                        "INSERT INTO schema_migracao (nome) VALUES (%s)", (arquivo.name,)
                    )
                aplicadas.append(arquivo.name)
        finally:
            conn.execute("SELECT pg_advisory_unlock(%s)", (TRAVA,))
    return aplicadas


if __name__ == "__main__":
    url = sys.argv[1] if len(sys.argv) > 1 else carregar().database_url
    feitas = migrar(url)
    print("aplicadas:", ", ".join(feitas) if feitas else "nenhuma, banco em dia")
