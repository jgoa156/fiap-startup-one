"""Operações sobre a inscrição.

Toda mudança de estado passa por `transicionar`, que:
  1. valida a transição na máquina de estados do Python (erro rápido e claro);
  2. define ator e motivo no contexto da transação, que o trigger do banco exige;
  3. faz UPDATE condicional ao estado esperado (compare-and-set), então duas
     requisições concorrentes não aplicam a mesma transição duas vezes.
O banco valida de novo e grava a trilha. A aplicação nunca escreve na trilha.
"""

from dataclasses import dataclass
from datetime import datetime

from psycopg import AsyncConnection, errors
from psycopg.rows import dict_row

from app.inscricao.estados import Estado, TransicaoInvalida, validar


class InscricaoNaoEncontrada(Exception):
    pass


class ConflitoDeEstado(Exception):
    """O estado mudou entre a leitura e a escrita (outra requisição chegou antes)."""

    def __init__(self, esperado: Estado, atual: Estado):
        self.esperado, self.atual = esperado, atual
        super().__init__(f"estado esperado {esperado}, mas está {atual}")


class EditalNaoPublicado(Exception):
    pass


class DadosCongelados(Exception):
    pass


@dataclass(frozen=True)
class Transicao:
    de: Estado | None
    para: Estado
    ator: str
    motivo: str
    versao_edital: int
    em: datetime


async def _contexto(conn: AsyncConnection, ator: str, motivo: str) -> None:
    if not ator.strip() or not motivo.strip():
        raise ValueError("ator e motivo são obrigatórios")
    await conn.execute(
        "SELECT set_config('certamis.ator', %s, true), set_config('certamis.motivo', %s, true)",
        (ator, motivo),
    )


async def criar(
    conn: AsyncConnection,
    *,
    certame_id: int,
    candidato_id: int,
    ator: str,
    motivo: str = "inscrição iniciada",
) -> int:
    async with conn.transaction():
        await _contexto(conn, ator, motivo)
        try:
            cur = await conn.execute(
                "INSERT INTO inscricao (certame_id, candidato_id) VALUES (%s, %s) RETURNING id",
                (certame_id, candidato_id),
            )
        except errors.DatabaseError as e:
            raise _traduzir(e) from e
        row = await cur.fetchone()
        assert row is not None
        return row[0]


async def transicionar(
    conn: AsyncConnection,
    inscricao_id: int,
    *,
    de: Estado,
    para: Estado,
    ator: str,
    motivo: str,
) -> None:
    validar(de, para)
    async with conn.transaction():
        await _contexto(conn, ator, motivo)
        try:
            cur = await conn.execute(
                "UPDATE inscricao SET estado = %s WHERE id = %s AND estado = %s RETURNING id",
                (para.value, inscricao_id, de.value),
            )
        except errors.DatabaseError as e:
            raise _traduzir(e) from e
        if await cur.fetchone() is None:
            atual = await estado_atual(conn, inscricao_id)
            raise ConflitoDeEstado(de, atual)


async def estado_atual(conn: AsyncConnection, inscricao_id: int) -> Estado:
    cur = await conn.execute("SELECT estado FROM inscricao WHERE id = %s", (inscricao_id,))
    row = await cur.fetchone()
    if row is None:
        raise InscricaoNaoEncontrada(inscricao_id)
    return Estado(row[0])


async def trilha(conn: AsyncConnection, inscricao_id: int) -> list[Transicao]:
    await estado_atual(conn, inscricao_id)  # 404 se não existe
    async with conn.cursor(row_factory=dict_row) as cur:
        await cur.execute(
            "SELECT de, para, ator, motivo, versao_edital, em FROM inscricao_transicao"
            " WHERE inscricao_id = %s ORDER BY id",
            (inscricao_id,),
        )
        rows = await cur.fetchall()
    return [
        Transicao(
            de=Estado(r["de"]) if r["de"] else None,
            para=Estado(r["para"]),
            ator=r["ator"],
            motivo=r["motivo"],
            versao_edital=r["versao_edital"],
            em=r["em"],
        )
        for r in rows
    ]


def _traduzir(e: Exception) -> Exception:
    """Converte os erros CTxxx do banco em exceções do domínio."""
    sqlstate = getattr(e, "sqlstate", None)
    diag = getattr(e, "diag", None)
    msg = getattr(diag, "message_primary", None) or str(e)
    if sqlstate == "CT001":
        return TransicaoInvalidaNoBanco(msg)
    if sqlstate == "CT004":
        return DadosCongelados(msg)
    if sqlstate == "CT005":
        return EditalNaoPublicado(msg)
    return e


class TransicaoInvalidaNoBanco(TransicaoInvalida):
    """O banco recusou uma transição que o Python aceitou: as tabelas divergiram."""

    def __init__(self, msg: str):
        Exception.__init__(self, msg)
