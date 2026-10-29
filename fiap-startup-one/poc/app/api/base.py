"""Peças comuns da API: conexão, autenticação, contexto de auditoria e erros do banco."""

from fastapi import Depends, Header, HTTPException, Request
from fastapi.responses import JSONResponse
from psycopg import AsyncConnection
from psycopg import errors as pg_errors

from app import token
from app.config import Config

# SQLSTATE próprio (classe CT) -> status HTTP. As regras moram no banco; a API só traduz.
HTTP_POR_SQLSTATE = {
    "CT001": 409,  # transição não permitida no estado atual
    "CT002": 400,  # sem ator ou motivo
    "CT003": 409,  # trilha imutável
    "CT004": 409,  # dado congelado fora do rascunho
    "CT005": 409,  # certame sem edital
    "CT006": 422,  # inscrição incompleta
    "CT007": 409,  # fora do prazo
    "CT008": 409,  # isenção incoerente
    "CT009": 409,  # transição de isenção não permitida
    "CT010": 409,  # inscrição não aguarda pagamento
    "CT011": 409,  # já paga
    "23505": 409,  # violação de unicidade
    "23503": 422,  # referência inexistente (cargo de outro certame, etc.)
}


async def erro_do_banco(_: Request, e: pg_errors.Error) -> JSONResponse:
    status = HTTP_POR_SQLSTATE.get(e.sqlstate or "")
    if status is None:
        raise e
    msg = (e.diag.message_primary if e.diag else None) or str(e)
    return JSONResponse({"codigo": e.sqlstate, "mensagem": msg}, status_code=status)


def config(request: Request) -> Config:
    return request.app.state.config


async def conexao(request: Request):
    async with request.app.state.pool.connection() as conn:
        yield conn


def armazenamento(request: Request):
    return request.app.state.armazenamento


def candidato_atual(
    authorization: str = Header(...), cfg: Config = Depends(config)
) -> int:
    esquema, _, valor = authorization.partition(" ")
    if esquema.lower() != "bearer" or not valor:
        raise HTTPException(401, "token ausente")
    try:
        return token.verificar(cfg.segredo_token, valor)
    except token.TokenInvalido as e:
        raise HTTPException(401, "token inválido") from e


def ator_banca(x_ator_banca: str = Header(..., min_length=1)) -> str:
    """Na POC a banca não tem login: o painel da banca está fora do escopo. O ator ainda
    é obrigatório, porque vai para a trilha."""
    return f"banca:{x_ator_banca}"


async def contexto(conn: AsyncConnection, ator: str, motivo: str) -> None:
    """Define quem e por quê, para os triggers de auditoria. Vale até o fim da transação."""
    await conn.execute(
        "SELECT set_config('certamis.ator', %s, true), set_config('certamis.motivo', %s, true)",
        (ator, motivo),
    )


async def inscricao_do_candidato(conn: AsyncConnection, iid: int, cid: int) -> dict:
    cur = await conn.execute(
        "SELECT id, certame_id, estado, cargo_id, cidade_id, modalidade_id,"
        " atendimento_especial, declaracoes_aceitas_em FROM inscricao"
        " WHERE id = %s AND candidato_id = %s",
        (iid, cid),
    )
    row = await cur.fetchone()
    if row is None:
        # 404 também para inscrição de outro candidato: não revela que existe.
        raise HTTPException(404, "inscrição não encontrada")
    cols = [d.name for d in cur.description]
    return dict(zip(cols, row))
