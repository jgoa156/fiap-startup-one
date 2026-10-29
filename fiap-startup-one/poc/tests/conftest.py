"""Infra de teste: um banco descartável, recriado e migrado a cada sessão.

Usa psycopg quando está instalado. Sem ele, cai para o binário psql, o que
permite rodar os testes do banco mesmo sem as dependências Python da API.
"""

import importlib.util
import os
import shutil
import subprocess
from pathlib import Path
from urllib.parse import urlparse

import pytest

RAIZ = Path(__file__).resolve().parent.parent
MIGRACOES = RAIZ / "app" / "migrations"
URL = os.environ.get(
    "CERTAMIS_TEST_DATABASE_URL",
    "postgresql://certamis:certamis@localhost:5432/certamis_test",
)
TEM_PSYCOPG = importlib.util.find_spec("psycopg") is not None


def _url_admin(url: str) -> str:
    return urlparse(url)._replace(path="/postgres").geturl()


def _nome_banco(url: str) -> str:
    return urlparse(url).path.lstrip("/")


class Banco:
    """Executa SQL no banco de teste por psycopg ou por psql."""

    def __init__(self, url: str):
        self.url = url

    def script(self, sql: str) -> None:
        if TEM_PSYCOPG:
            import psycopg

            # Comandos de barra invertida são do psql, não do servidor.
            limpo = "\n".join(l for l in sql.splitlines() if not l.lstrip().startswith("\\"))
            with psycopg.connect(self.url, autocommit=True) as c:
                c.execute(limpo)
        else:
            subprocess.run(
                ["psql", self.url, "-X", "-q", "-v", "ON_ERROR_STOP=1", "-1"],
                input=sql, text=True, check=True, capture_output=True,
            )

    def linhas(self, sql: str) -> list[tuple]:
        if TEM_PSYCOPG:
            import psycopg

            with psycopg.connect(self.url) as c:
                return [tuple(r) for r in c.execute(sql).fetchall()]
        saida = subprocess.run(
            ["psql", self.url, "-X", "-q", "-At", "-F", "\t", "-c", sql],
            text=True, check=True, capture_output=True,
        ).stdout
        return [tuple(l.split("\t")) for l in saida.splitlines() if l]


def _recriar(url: str) -> None:
    nome = _nome_banco(url)
    assert nome.endswith("_test"), "por segurança, o banco de teste precisa terminar em _test"
    cmds = [f'DROP DATABASE IF EXISTS "{nome}" WITH (FORCE)', f'CREATE DATABASE "{nome}"']
    if TEM_PSYCOPG:
        import psycopg

        with psycopg.connect(_url_admin(url), autocommit=True) as c:
            for cmd in cmds:
                c.execute(cmd)
    else:
        if shutil.which("psql") is None:
            pytest.skip("sem psycopg e sem psql: não há como falar com o Postgres")
        for cmd in cmds:
            subprocess.run(["psql", _url_admin(url), "-X", "-q", "-c", cmd],
                           check=True, capture_output=True)


@pytest.fixture
def banco_vazio() -> Banco:
    """Banco recriado e migrado do zero, para cada teste que pede."""
    try:
        _recriar(URL)
    except Exception as e:  # Postgres fora do ar
        pytest.skip(f"Postgres de teste indisponível em {URL}: {e}")
    b = Banco(URL)
    for arq in sorted(MIGRACOES.glob("*.sql")):
        b.script(arq.read_text(encoding="utf-8"))
    return b
