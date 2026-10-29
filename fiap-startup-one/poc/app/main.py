"""API da POC. Semana 1: inscrição, transições e trilha.

Ainda sem autenticação: o ator vem no cabeçalho X-Ator. Autenticação entra
junto com o cadastro por CPF, na semana 2.
"""

from contextlib import asynccontextmanager
from datetime import datetime

from fastapi import Depends, FastAPI, Header, HTTPException, Request
from psycopg_pool import AsyncConnectionPool
from pydantic import BaseModel, Field

from app.config import carregar
from app.inscricao import servico
from app.inscricao.estados import Estado, TransicaoInvalida, proximos


@asynccontextmanager
async def ciclo_de_vida(app: FastAPI):
    cfg = carregar()
    pool = AsyncConnectionPool(
        cfg.database_url,
        min_size=cfg.pool_min,
        max_size=cfg.pool_max,
        timeout=cfg.pool_timeout_s,
        open=False,
    )
    await pool.open(wait=True)
    app.state.pool = pool
    try:
        yield
    finally:
        await pool.close()


app = FastAPI(title="Certamis POC", lifespan=ciclo_de_vida)


async def conexao(request: Request):
    async with request.app.state.pool.connection() as conn:
        yield conn


def ator(x_ator: str = Header(..., min_length=1)) -> str:
    return x_ator


class NovaInscricao(BaseModel):
    certame_id: int
    candidato_id: int


class NovaTransicao(BaseModel):
    de: Estado
    para: Estado
    motivo: str = Field(min_length=1)


class TransicaoSaida(BaseModel):
    de: Estado | None
    para: Estado
    ator: str
    motivo: str
    versao_edital: int
    em: datetime


class InscricaoSaida(BaseModel):
    id: int
    estado: Estado
    proximos: list[Estado]


@app.get("/saude")
async def saude(conn=Depends(conexao)):
    await conn.execute("SELECT 1")
    return {"ok": True}


@app.post("/inscricoes", status_code=201, response_model=InscricaoSaida)
async def criar_inscricao(corpo: NovaInscricao, conn=Depends(conexao), quem: str = Depends(ator)):
    try:
        iid = await servico.criar(
            conn, certame_id=corpo.certame_id, candidato_id=corpo.candidato_id, ator=quem
        )
    except servico.EditalNaoPublicado as e:
        raise HTTPException(409, str(e)) from e
    return InscricaoSaida(id=iid, estado=Estado.RASCUNHO, proximos=proximos(Estado.RASCUNHO))


@app.get("/inscricoes/{iid}", response_model=InscricaoSaida)
async def ver_inscricao(iid: int, conn=Depends(conexao)):
    try:
        estado = await servico.estado_atual(conn, iid)
    except servico.InscricaoNaoEncontrada as e:
        raise HTTPException(404, "inscrição não encontrada") from e
    return InscricaoSaida(id=iid, estado=estado, proximos=proximos(estado))


@app.post("/inscricoes/{iid}/transicoes", response_model=InscricaoSaida)
async def transicionar(
    iid: int, corpo: NovaTransicao, conn=Depends(conexao), quem: str = Depends(ator)
):
    try:
        await servico.transicionar(
            conn, iid, de=corpo.de, para=corpo.para, ator=quem, motivo=corpo.motivo
        )
    except servico.InscricaoNaoEncontrada as e:
        raise HTTPException(404, "inscrição não encontrada") from e
    except TransicaoInvalida as e:
        raise HTTPException(422, str(e)) from e
    except servico.ConflitoDeEstado as e:
        raise HTTPException(409, str(e)) from e
    return InscricaoSaida(id=iid, estado=corpo.para, proximos=proximos(corpo.para))


@app.get("/inscricoes/{iid}/trilha", response_model=list[TransicaoSaida])
async def ver_trilha(iid: int, conn=Depends(conexao)):
    try:
        return [TransicaoSaida(**t.__dict__) for t in await servico.trilha(conn, iid)]
    except servico.InscricaoNaoEncontrada as e:
        raise HTTPException(404, "inscrição não encontrada") from e
