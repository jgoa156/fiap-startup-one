"""Infra de teste: um banco descartável, recriado e migrado a cada teste que pede.

Fala com o Postgres por psycopg quando ele está instalado. Sem ele, usa o
binário psql. Os arquivos .sql de teste usam recursos do psql (\\gset, :var),
então rodam pelo psql local ou, se não houver, pelo psql de dentro do
contêiner do docker-compose.
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


def _comando_psql(url: str) -> list[str]:
    if shutil.which("psql"):
        return ["psql", url]
    if shutil.which("docker"):
        u = urlparse(url)
        return ["docker", "compose", "-f", str(RAIZ / "docker-compose.yml"), "exec", "-T",
                "postgres", "psql", "-U", u.username or "certamis", "-d", _nome_banco(url)]
    pytest.skip("sem psql e sem docker: não há como rodar os testes .sql")


class ErroBanco(Exception):
    def __init__(self, sqlstate: str | None, mensagem: str):
        self.sqlstate, self.mensagem = sqlstate, mensagem
        super().__init__(f"{sqlstate}: {mensagem}")


class Banco:
    def __init__(self, url: str):
        self.url = url

    def script(self, sql: str) -> None:
        """SQL puro, sem comandos do psql (migrações)."""
        if TEM_PSYCOPG:
            import psycopg

            with psycopg.connect(self.url, autocommit=True) as c:
                c.execute(sql)
        else:
            self._psql(sql, um_so=True)

    def arquivo(self, caminho: Path) -> str:
        """Arquivo de teste .sql, que pode usar \\gset e :variáveis."""
        return self._psql(caminho.read_text(encoding="utf-8"), um_so=False)

    def _psql(self, sql: str, um_so: bool) -> str:
        cmd = _comando_psql(self.url) + ["-X", "-q", "-v", "ON_ERROR_STOP=1"]
        if um_so:
            cmd.append("-1")
        r = subprocess.run(cmd, input=sql, text=True, capture_output=True, encoding="utf-8")
        if r.returncode != 0:
            raise AssertionError(r.stderr.strip())
        return r.stdout

    def linhas(self, sql: str) -> list[tuple]:
        """Executa numa conexão nova e devolve as linhas. Levanta ErroBanco com o SQLSTATE."""
        if TEM_PSYCOPG:
            import psycopg

            try:
                with psycopg.connect(self.url, autocommit=True) as c:
                    cur = c.execute(sql)
                    return [tuple(map(_texto, r)) for r in cur.fetchall()] if cur.description else []
            except psycopg.Error as e:
                raise ErroBanco(e.sqlstate, str(e)) from e
        cmd = _comando_psql(self.url) + ["-X", "-q", "-At", "-F", "\t",
                                         "-v", "ON_ERROR_STOP=1", "-v", "VERBOSITY=verbose"]
        r = subprocess.run(cmd, input=sql, text=True, capture_output=True, encoding="utf-8")
        if r.returncode != 0:
            # Formato verboso: "ERROR:  CT001: mensagem"
            linha = next((l for l in r.stderr.splitlines() if l.startswith("ERROR:")), r.stderr)
            partes = linha.removeprefix("ERROR:").strip().split(":", 1)
            raise ErroBanco(partes[0].strip() if len(partes) == 2 else None, linha)
        return [tuple(l.split("\t")) for l in r.stdout.splitlines() if l]

    def valor(self, sql: str) -> str:
        return self.linhas(sql)[0][0]


def _texto(v) -> str:
    return "" if v is None else str(v)


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


CERTAME_ABERTO = """
INSERT INTO banca (nome) VALUES ('Banca Sintética');
INSERT INTO certame (banca_id, nome, taxa_centavos, isencao_inicio, isencao_fim,
                     inscricao_inicio, inscricao_fim)
VALUES (1, 'Certame Sintético', 8000, now() - interval '2 days', now() + interval '5 days',
        now() - interval '1 day', now() + interval '29 days');
INSERT INTO edital_versao (certame_id, versao, descricao) VALUES (1, 1, 'edital original');
INSERT INTO cargo (certame_id, nome, vagas) VALUES (1, 'Analista', 10);
INSERT INTO cidade_prova (certame_id, nome) VALUES (1, 'Sede');
INSERT INTO modalidade (certame_id, codigo, nome) VALUES (1, 'AMPLA', 'Ampla concorrência');
INSERT INTO hipotese_isencao (certame_id, codigo, nome, exige_nis, exige_comprovante)
VALUES (1, 'CADUNICO', 'Inscrito no CadÚnico', true, false);
"""


@pytest.fixture
def banco_com_certame(banco_vazio) -> Banco:
    banco_vazio.script(CERTAME_ABERTO)
    return banco_vazio
