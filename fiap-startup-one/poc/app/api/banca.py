"""Rotas da banca. O painel da banca está fora do escopo da POC (é o primeiro corte);
estas rotas existem porque o ensaio e os testes precisam movimentar a análise."""

from fastapi import APIRouter, Depends, HTTPException
from psycopg.rows import dict_row
from psycopg.types.json import Jsonb
from pydantic import BaseModel, Field

from app.api.base import ator_banca, conexao, contexto
from app.inscricao import servico as insc_servico
from app.inscricao.estados import Estado, TransicaoInvalida, proximos

rotas = APIRouter(prefix="/banca")


class Decisao(BaseModel):
    deferir: bool
    motivo: str = Field(min_length=3)


class Transicao(BaseModel):
    de: Estado
    para: Estado
    motivo: str = Field(min_length=3)


@rotas.post("/isencoes/{pedido_id}/decisao")
async def decidir_isencao(pedido_id: int, corpo: Decisao, conn=Depends(conexao),
                          ator: str = Depends(ator_banca)):
    async with conn.transaction():
        await contexto(conn, ator, corpo.motivo)
        if corpo.deferir:
            cur = await conn.execute(
                "UPDATE pedido_isencao SET estado = 'DEFERIDA' WHERE id = %s RETURNING estado",
                (pedido_id,))
        else:
            cur = await conn.execute(
                "UPDATE pedido_isencao SET estado = 'INDEFERIDA', motivo_codigo = 'ANALISE_HUMANA'"
                " WHERE id = %s RETURNING estado", (pedido_id,))
        row = await cur.fetchone()
    if row is None:
        raise HTTPException(404, "pedido não encontrado")
    return {"id": pedido_id, "estado": row[0]}


@rotas.post("/inscricoes/{iid}/transicoes")
async def transicionar(iid: int, corpo: Transicao, conn=Depends(conexao),
                       ator: str = Depends(ator_banca)):
    try:
        await insc_servico.transicionar(conn, iid, de=corpo.de, para=corpo.para,
                                        ator=ator, motivo=corpo.motivo)
    except insc_servico.InscricaoNaoEncontrada as e:
        raise HTTPException(404, "inscrição não encontrada") from e
    except TransicaoInvalida as e:
        raise HTTPException(422, str(e)) from e
    except insc_servico.ConflitoDeEstado as e:
        raise HTTPException(409, str(e)) from e
    return {"id": iid, "estado": corpo.para, "proximos": proximos(corpo.para)}


@rotas.get("/inscricoes/{iid}/trilha")
async def trilha(iid: int, conn=Depends(conexao), _: str = Depends(ator_banca)):
    try:
        return [t.__dict__ for t in await insc_servico.trilha(conn, iid)]
    except insc_servico.InscricaoNaoEncontrada as e:
        raise HTTPException(404, "inscrição não encontrada") from e


@rotas.get("/conciliacao")
async def conciliacao(conn=Depends(conexao), _: str = Depends(ator_banca)):
    async with conn.cursor(row_factory=dict_row) as cur:
        await cur.execute("SELECT * FROM conciliacao ORDER BY em")
        return await cur.fetchall()


class Operacao(BaseModel):
    nivel: int = Field(description="0 normal, 1 fila de espera, 4 só estático, 5 página de status")
    mensagem: str = ""
    taxa_por_s: float | None = Field(default=None, gt=0)


@rotas.post("/operacao")
async def mudar_operacao(corpo: Operacao, conn=Depends(conexao), ator: str = Depends(ator_banca)):
    """O plantão muda o nível. Cada instância lê a mudança em até 1 segundo."""
    if corpo.nivel not in (0, 1, 4, 5):
        raise HTTPException(422, "nível deve ser 0, 1, 4 ou 5")
    async with conn.transaction():
        await conn.execute("UPDATE modo_operacao SET nivel = %s, mensagem = %s,"
                           " atualizado_em = now(), atualizado_por = %s",
                           (corpo.nivel, corpo.mensagem, ator))
        if corpo.taxa_por_s:
            await conn.execute("UPDATE portao SET taxa_por_s = %s", (corpo.taxa_por_s,))
    return corpo


@rotas.post("/certames/{certame_id}/publicar-homologacao", status_code=202)
async def publicar_homologacao(certame_id: int, conn=Depends(conexao),
                               ator: str = Depends(ator_banca)):
    async with conn.transaction():
        cur = await conn.execute(
            "INSERT INTO tarefa (tipo, chave, dados) VALUES ('publicar_homologacao',"
            " %s || ':' || clock_timestamp()::text, %s) RETURNING id",
            (str(certame_id), Jsonb({"certame_id": certame_id, "pedido_por": ator})))
        (tid,) = await cur.fetchone()
    return {"tarefa": tid, "arquivo": f"/publico/certames/{certame_id}/homologacao.json"}
