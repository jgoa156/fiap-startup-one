"""Rotas do candidato: conta, sessão, inscrição, laudo, isenção e cobrança.

É o caminho crítico do pico 2. Cada rota faz o mínimo de idas ao banco, e nenhuma
gera PDF ou fala com o provedor de pagamento dentro do request (dívida D3).
"""

import asyncio
from datetime import datetime

from fastapi import APIRouter, Depends, Header, HTTPException, Request, Response
from psycopg import errors as pg_errors
from psycopg.rows import dict_row
from psycopg.types.json import Jsonb
from pydantic import BaseModel, Field

from app import cpf as cpf_mod
from app import senha, token
from app.api.base import (armazenamento, candidato_atual, conexao, config, contexto,
                          inscricao_do_candidato)
from app.config import Config
from app.inscricao import servico as insc_servico
from app.inscricao.estados import Estado, TransicaoInvalida, proximos

rotas = APIRouter()

TAMANHO_MAX = 10 * 1024 * 1024
TIPOS_ACEITOS = {"application/pdf", "image/jpeg", "image/png"}


# ------------------------------------------------------------------ conta e sessão

class NovoCandidato(BaseModel):
    cpf: str = Field(pattern=r"^\d{11}$")
    nome: str = Field(min_length=3, max_length=200)
    email: str = Field(pattern=r"^[^@\s]+@[^@\s]+\.[^@\s]+$", max_length=200)
    senha: str = Field(min_length=8, max_length=200)


class Credencial(BaseModel):
    cpf: str = Field(pattern=r"^\d{11}$")
    senha: str


@rotas.post("/candidatos", status_code=201)
async def cadastrar(corpo: NovoCandidato, conn=Depends(conexao), cfg: Config = Depends(config)):
    if not cpf_mod.valido(corpo.cpf):
        raise HTTPException(422, "CPF inválido")
    # scrypt é CPU pura: fora do loop de eventos, para não travar as outras requisições.
    h = await asyncio.to_thread(senha.gerar, corpo.senha, cfg.scrypt_n)
    try:
        async with conn.transaction():
            cur = await conn.execute(
                "INSERT INTO candidato (cpf, nome, email, senha_hash) VALUES (%s, %s, %s, %s)"
                " RETURNING id",
                (corpo.cpf, corpo.nome, corpo.email, h),
            )
            (cid,) = await cur.fetchone()
    except pg_errors.UniqueViolation as e:
        # Conta é única por CPF (decisão fechada). Não diz se foi o CPF ou o e-mail.
        raise HTTPException(409, "já existe conta com este CPF ou e-mail") from e
    return {"id": cid}


@rotas.post("/sessoes")
async def entrar(corpo: Credencial, conn=Depends(conexao), cfg: Config = Depends(config)):
    cur = await conn.execute("SELECT id, senha_hash FROM candidato WHERE cpf = %s", (corpo.cpf,))
    row = await cur.fetchone()
    if row is None or not await asyncio.to_thread(senha.confere, corpo.senha, row[1]):
        raise HTTPException(401, "CPF ou senha incorretos")
    return {"token": token.emitir(cfg.segredo_token, row[0])}


# ------------------------------------------------------------------ certame (leitura)

@rotas.get("/certames/{certame_id}")
async def ver_certame(certame_id: int, conn=Depends(conexao)):
    """Na produção isto é arquivo estático na CDN; aqui fica no banco para a POC."""
    async with conn.cursor(row_factory=dict_row) as cur:
        await cur.execute(
            "SELECT id, nome, taxa_centavos, isencao_inicio, isencao_fim, inscricao_inicio,"
            " inscricao_fim, pagamento_fim, edital_versao_vigente(id) AS versao_edital"
            " FROM certame WHERE id = %s", (certame_id,))
        certame = await cur.fetchone()
        if certame is None:
            raise HTTPException(404, "certame não encontrado")
        for tabela, colunas, chave in [
            ("cargo", "id, nome, vagas", "cargos"),
            ("cidade_prova", "id, nome", "cidades"),
            ("modalidade", "id, codigo, nome", "modalidades"),
            ("hipotese_isencao", "codigo, nome, exige_nis, exige_comprovante", "hipoteses_isencao"),
        ]:
            await cur.execute(f"SELECT {colunas} FROM {tabela} WHERE certame_id = %s", (certame_id,))
            certame[chave] = await cur.fetchall()
    return certame


# ------------------------------------------------------------------ inscrição

class InscricaoSaida(BaseModel):
    id: int
    certame_id: int
    estado: Estado
    proximos: list[Estado]
    cargo_id: int | None = None
    cidade_id: int | None = None
    modalidade_id: int | None = None
    atendimento_especial: bool = False
    declaracoes_aceitas_em: datetime | None = None


def _saida(d: dict) -> InscricaoSaida:
    e = Estado(d["estado"])
    return InscricaoSaida(**{**d, "estado": e, "proximos": proximos(e)})


class NovaInscricao(BaseModel):
    certame_id: int


class DadosInscricao(BaseModel):
    cargo_id: int | None = None
    cidade_id: int | None = None
    modalidade_id: int | None = None
    atendimento_especial: bool | None = None
    aceitar_declaracoes: bool | None = None


@rotas.post("/inscricoes", status_code=201, response_model=InscricaoSaida)
async def iniciar(corpo: NovaInscricao, conn=Depends(conexao), cid: int = Depends(candidato_atual)):
    try:
        iid = await insc_servico.criar(conn, certame_id=corpo.certame_id, candidato_id=cid,
                                       ator=f"candidato:{cid}")
    except insc_servico.EditalNaoPublicado as e:
        raise HTTPException(409, str(e)) from e
    return _saida(await inscricao_do_candidato(conn, iid, cid))


@rotas.get("/inscricoes/{iid}", response_model=InscricaoSaida)
async def ver(iid: int, conn=Depends(conexao), cid: int = Depends(candidato_atual)):
    return _saida(await inscricao_do_candidato(conn, iid, cid))


@rotas.patch("/inscricoes/{iid}", response_model=InscricaoSaida)
async def preencher(iid: int, corpo: DadosInscricao, conn=Depends(conexao),
                    cid: int = Depends(candidato_atual)):
    """Salva a cada passo do formulário. Fora do RASCUNHO o banco recusa (CT004)."""
    campos = corpo.model_dump(exclude_none=True)
    sets, valores = [], []
    for c in ("cargo_id", "cidade_id", "modalidade_id", "atendimento_especial"):
        if c in campos:
            sets.append(f"{c} = %s")
            valores.append(campos[c])
    if campos.get("aceitar_declaracoes"):
        sets.append("declaracoes_aceitas_em = coalesce(declaracoes_aceitas_em, now())")
    elif campos.get("aceitar_declaracoes") is False:
        sets.append("declaracoes_aceitas_em = NULL")
    async with conn.transaction():
        await inscricao_do_candidato(conn, iid, cid)
        if sets:
            await conn.execute(f"UPDATE inscricao SET {', '.join(sets)} WHERE id = %s",
                               (*valores, iid))
    return _saida(await inscricao_do_candidato(conn, iid, cid))


@rotas.put("/inscricoes/{iid}/laudo", status_code=201)
async def enviar_laudo(iid: int, request: Request, conn=Depends(conexao),
                       cid: int = Depends(candidato_atual), arm=Depends(armazenamento),
                       content_type: str = Header(...),
                       x_nome_arquivo: str = Header("laudo.pdf", max_length=200)):
    """Corpo cru do arquivo. O laudo pertence a esta inscrição e só a ela."""
    if content_type not in TIPOS_ACEITOS:
        raise HTTPException(415, "envie PDF, JPEG ou PNG")
    dados = await request.body()
    if not 0 < len(dados) <= TAMANHO_MAX:
        raise HTTPException(413, "arquivo vazio ou maior que 10 MB")
    insc = await inscricao_do_candidato(conn, iid, cid)
    if insc["estado"] != Estado.RASCUNHO:
        raise HTTPException(409, "laudo só é enviado com a inscrição em rascunho")
    chave, sha = await asyncio.to_thread(arm.guardar, "laudo", dados)
    async with conn.transaction():
        cur = await conn.execute(
            "INSERT INTO documento (candidato_id, inscricao_id, tipo, nome_original, tamanho_bytes,"
            " sha256, chave) VALUES (%s, %s, 'LAUDO', %s, %s, %s, %s) RETURNING id",
            (cid, iid, x_nome_arquivo, len(dados), sha, chave),
        )
        (did,) = await cur.fetchone()
    return {"documento_id": did, "sha256": sha}


@rotas.post("/inscricoes/{iid}/finalizar", response_model=InscricaoSaida)
async def finalizar(iid: int, conn=Depends(conexao), cid: int = Depends(candidato_atual)):
    """Isento vai para ISENCAO_ATIVA, pagante para AGUARDANDO_PAGAMENTO. Quem decide é
    a isenção deferida no banco; o banco também confere se o formulário está completo."""
    async with conn.transaction():
        insc = await inscricao_do_candidato(conn, iid, cid)
        if insc["estado"] != Estado.RASCUNHO:
            raise HTTPException(409, f"inscrição está em {insc['estado']}, não em RASCUNHO")
        cur = await conn.execute(
            "SELECT EXISTS (SELECT 1 FROM pedido_isencao WHERE certame_id = %s"
            " AND candidato_id = %s AND estado = 'DEFERIDA')", (insc["certame_id"], cid))
        (isento,) = await cur.fetchone()
        para = Estado.ISENCAO_ATIVA if isento else Estado.AGUARDANDO_PAGAMENTO
        await contexto(conn, f"candidato:{cid}", "finalizou o formulário")
        cur = await conn.execute(
            "UPDATE inscricao SET estado = %s WHERE id = %s AND estado = 'RASCUNHO' RETURNING id",
            (para.value, iid))
        if await cur.fetchone() is None:
            raise HTTPException(409, "a inscrição mudou de estado durante a operação")
    return _saida(await inscricao_do_candidato(conn, iid, cid))


async def _mover(conn, iid: int, cid: int, de: Estado, para: Estado, motivo: str):
    await inscricao_do_candidato(conn, iid, cid)
    try:
        await insc_servico.transicionar(conn, iid, de=de, para=para,
                                        ator=f"candidato:{cid}", motivo=motivo)
    except (insc_servico.ConflitoDeEstado, TransicaoInvalida) as e:
        raise HTTPException(409, str(e)) from e
    return _saida(await inscricao_do_candidato(conn, iid, cid))


@rotas.post("/inscricoes/{iid}/reabrir", response_model=InscricaoSaida)
async def reabrir(iid: int, conn=Depends(conexao), cid: int = Depends(candidato_atual)):
    """Volta ao rascunho para corrigir, antes de pagar. A cobrança pendente é cancelada."""
    return await _mover(conn, iid, cid, Estado.AGUARDANDO_PAGAMENTO, Estado.RASCUNHO,
                        "voltou ao rascunho para corrigir")


@rotas.post("/inscricoes/{iid}/cancelar", response_model=InscricaoSaida)
async def cancelar(iid: int, conn=Depends(conexao), cid: int = Depends(candidato_atual)):
    insc = await inscricao_do_candidato(conn, iid, cid)
    return await _mover(conn, iid, cid, Estado(insc["estado"]), Estado.CANCELADA,
                        "candidato desistiu")


@rotas.get("/inscricoes/{iid}/trilha")
async def trilha(iid: int, conn=Depends(conexao), cid: int = Depends(candidato_atual)):
    await inscricao_do_candidato(conn, iid, cid)
    return [t.__dict__ for t in await insc_servico.trilha(conn, iid)]


@rotas.get("/inscricoes/{iid}/cartao")
async def cartao(iid: int, conn=Depends(conexao), cid: int = Depends(candidato_atual),
                 arm=Depends(armazenamento)):
    """O cartão é gerado pelo worker na homologação. Na AWS isto vira URL assinada do S3."""
    await inscricao_do_candidato(conn, iid, cid)
    cur = await conn.execute("SELECT cartao_chave FROM inscricao WHERE id = %s", (iid,))
    (chave,) = await cur.fetchone()
    if not chave:
        return Response(status_code=202, content=b'{"pronto": false}', media_type="application/json")
    dados = await asyncio.to_thread(arm.ler, chave)
    return Response(content=dados, media_type="application/pdf")


@rotas.get("/inscricoes/{iid}/cobranca")
async def ver_cobranca(iid: int, conn=Depends(conexao), cid: int = Depends(candidato_atual)):
    await inscricao_do_candidato(conn, iid, cid)
    async with conn.cursor(row_factory=dict_row) as cur:
        await cur.execute(
            "SELECT id, referencia, meio, valor_centavos, status, vence_em, instrucoes, registrada_em"
            " FROM cobranca WHERE inscricao_id = %s ORDER BY id DESC LIMIT 1", (iid,))
        c = await cur.fetchone()
    if c is None:
        raise HTTPException(404, "sem cobrança")
    return c


# ------------------------------------------------------------------ cobrança

class PedidoCobranca(BaseModel):
    meio: str = Field(pattern="^(PIX|BOLETO|CARTAO)$")


@rotas.post("/inscricoes/{iid}/cobranca")
async def cobranca(iid: int, corpo: PedidoCobranca, conn=Depends(conexao),
                   cid: int = Depends(candidato_atual),
                   idempotency_key: str = Header(..., min_length=8, max_length=200)):
    """Repetir com a mesma Idempotency-Key devolve a mesma cobrança. Repetir com outra,
    enquanto houver uma pendente, também: nunca existem duas vivas. Quem registra a
    cobrança no provedor é o worker, fora do request."""
    await inscricao_do_candidato(conn, iid, cid)
    async with conn.transaction():
        async with conn.cursor(row_factory=dict_row) as cur:
            await cur.execute(
                "SELECT id, referencia, meio, valor_centavos, status, vence_em"
                " FROM gerar_cobranca(%s, %s, %s)", (iid, idempotency_key, corpo.meio))
            c = await cur.fetchone()
        # Enfileira o registro no provedor (boleto, QR do Pix). Idempotente pela referência.
        await conn.execute(
            "INSERT INTO tarefa (tipo, chave, dados) VALUES ('registrar_cobranca', %s, %s)"
            " ON CONFLICT (tipo, chave) DO NOTHING",
            (c["referencia"], Jsonb({"cobranca_id": c["id"]})))
    return c


# ------------------------------------------------------------------ isenção

class PedidoIsencao(BaseModel):
    hipotese: str
    nis: str | None = Field(default=None, pattern=r"^\d{11}$")
    comprovante_id: int | None = None


@rotas.put("/certames/{certame_id}/isencao/comprovante", status_code=201)
async def enviar_comprovante(certame_id: int, request: Request, conn=Depends(conexao),
                             cid: int = Depends(candidato_atual), arm=Depends(armazenamento),
                             content_type: str = Header(...),
                             x_nome_arquivo: str = Header("comprovante.pdf", max_length=200)):
    if content_type not in TIPOS_ACEITOS:
        raise HTTPException(415, "envie PDF, JPEG ou PNG")
    dados = await request.body()
    if not 0 < len(dados) <= TAMANHO_MAX:
        raise HTTPException(413, "arquivo vazio ou maior que 10 MB")
    chave, sha = await asyncio.to_thread(arm.guardar, "isencao", dados)
    async with conn.transaction():
        cur = await conn.execute(
            "INSERT INTO documento (candidato_id, tipo, nome_original, tamanho_bytes, sha256, chave)"
            " VALUES (%s, 'COMPROVANTE_ISENCAO', %s, %s, %s, %s) RETURNING id",
            (cid, x_nome_arquivo, len(dados), sha, chave))
        (did,) = await cur.fetchone()
    return {"documento_id": did, "sha256": sha}


@rotas.post("/certames/{certame_id}/isencao", status_code=201)
async def pedir_isencao(certame_id: int, corpo: PedidoIsencao, conn=Depends(conexao),
                        cid: int = Depends(candidato_atual)):
    """A triagem por regra roda no banco, na entrada: a resposta já diz se faltou algo."""
    async with conn.transaction():
        await contexto(conn, "sistema:triagem", "triagem automática na entrada do pedido")
        await conn.execute(
            "INSERT INTO pedido_isencao (certame_id, candidato_id, hipotese, nis, comprovante_id)"
            " VALUES (%s, %s, %s, %s, %s)",
            (certame_id, cid, corpo.hipotese, corpo.nis, corpo.comprovante_id))
    return await ver_isencao(certame_id, conn, cid)


@rotas.get("/certames/{certame_id}/isencao")
async def ver_isencao(certame_id: int, conn=Depends(conexao), cid: int = Depends(candidato_atual)):
    async with conn.cursor(row_factory=dict_row) as cur:
        await cur.execute(
            "SELECT p.id, p.hipotese, p.estado, p.motivo_codigo, m.texto AS motivo"
            " FROM pedido_isencao p LEFT JOIN motivo_isencao m ON m.codigo = p.motivo_codigo"
            " WHERE p.certame_id = %s AND p.candidato_id = %s", (certame_id, cid))
        pedido = await cur.fetchone()
    if pedido is None:
        raise HTTPException(404, "nenhum pedido de isenção neste certame")
    return pedido
