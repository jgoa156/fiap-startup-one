"""API da POC da Certamis.

Monta as rotas, o pool de conexões, o armazenamento e o laço do modo de operação.
Uso: uvicorn app.main:app
"""

import asyncio
import logging
from contextlib import asynccontextmanager
from pathlib import Path

from fastapi import Depends, FastAPI, Header, HTTPException, Request
from fastapi.responses import JSONResponse
from fastapi.staticfiles import StaticFiles
from psycopg import errors as pg_errors
from psycopg_pool import AsyncConnectionPool

from app import operacao as op
from app.api import banca, candidato, provedor
from app.api.base import conexao, config, erro_do_banco
from app.armazenamento import DiscoLocal
from app.config import SEGREDO_DEV, Config, carregar

log = logging.getLogger("certamis")


@asynccontextmanager
async def ciclo_de_vida(app: FastAPI):
    cfg = carregar()
    if cfg.segredo_token == SEGREDO_DEV:
        log.warning("usando segredo de desenvolvimento; defina CERTAMIS_SEGREDO_TOKEN fora do local")
    # autocommit: leitura solta não abre transação implícita. Toda escrita usa
    # conn.transaction() explícito. Sem isso, uma rota que lê antes de escrever deixaria
    # a escrita num savepoint, e o pool a desfaria ao receber a conexão de volta.
    pool = AsyncConnectionPool(cfg.database_url, min_size=cfg.pool_min, max_size=cfg.pool_max,
                               timeout=cfg.pool_timeout_s, open=False,
                               kwargs={"autocommit": True})
    await pool.open(wait=True)
    app.state.config = cfg
    app.state.pool = pool
    app.state.armazenamento = DiscoLocal(cfg.pasta_arquivos)
    app.state.operacao = op.Operacao(pool)
    await app.state.operacao.atualizar()
    laco = asyncio.create_task(app.state.operacao.laco())
    try:
        yield
    finally:
        laco.cancel()
        await pool.close()


app = FastAPI(title="Certamis POC", lifespan=ciclo_de_vida)
app.add_exception_handler(pg_errors.Error, erro_do_banco)

# Arquivos públicos (homologação, resultado): na AWS saem do S3 pela CDN, sem tocar
# na aplicação nem no banco. Localmente, saem daqui.
_publico = Path(carregar().pasta_arquivos) / "publico"
_publico.mkdir(parents=True, exist_ok=True)
app.mount("/publico", StaticFiles(directory=_publico), name="publico")

# O plantão precisa conseguir sair do modo degradado; o webhook não pode perder pagamento.
SEMPRE_LIBERADO = ("/saude", "/status", "/publico", "/fila", "/webhooks", "/banca/operacao")


@app.middleware("http")
async def modo_degradado(request: Request, chamar):
    """Níveis 4 e 5 de 04-arquitetura-alvo §4.5. O nível 1 (fila) é aplicado no login."""
    estado = request.app.state.operacao.estado
    caminho = request.url.path
    if estado.nivel >= op.SO_ESTATICO and not caminho.startswith(SEMPRE_LIBERADO):
        corpo = {
            "nivel": estado.nivel,
            "mensagem": estado.mensagem or "Serviço temporariamente restrito. Consultas públicas seguem disponíveis.",
            "status": "/status",
        }
        return JSONResponse(corpo, status_code=503, headers={"Retry-After": "60"})
    return await chamar(request)


@app.middleware("http")
async def fila_de_espera(request: Request, chamar):
    """Nível 1: só entra (faz login) quem tem passe da fila. Quem já entrou segue."""
    estado = request.app.state.operacao.estado
    if (estado.nivel == op.FILA and request.method == "POST" and request.url.path == "/sessoes"):
        cfg: Config = request.app.state.config
        passe = request.headers.get("x-passe", "")
        if not op.passe_valido(cfg.segredo_token, passe):
            return JSONResponse({"mensagem": "entre na fila de espera", "fila": "/fila"},
                                status_code=429, headers={"Retry-After": "5"})
    return await chamar(request)


@app.get("/saude")
async def saude(conn=Depends(conexao)):
    await conn.execute("SELECT 1")
    return {"ok": True}


@app.get("/status")
async def status(request: Request):
    """Sem banco: responde do estado em memória, atualizado a cada segundo."""
    e = request.app.state.operacao.estado
    return {"nivel": e.nivel, "mensagem": e.mensagem, "fila_admitido_ate": e.admitido_ate,
            "fila_taxa_por_s": e.taxa_por_s}


@app.post("/fila")
async def entrar_na_fila(request: Request, conn=Depends(conexao), cfg: Config = Depends(config)):
    cur = await conn.execute("SELECT nextval('fila_senha_seq')")
    (numero,) = await cur.fetchone()
    falta, espera = request.app.state.operacao.posicao(numero)
    return {"senha": op.senha_emitir(cfg.segredo_token, numero), "posicao": falta,
            "espera_estimada_s": round(espera)}


@app.get("/fila/{senha}")
async def consultar_fila(senha: str, request: Request, cfg: Config = Depends(config)):
    """Sem banco: a consulta de posição é a requisição mais repetida do pico."""
    numero = op.senha_ler(cfg.segredo_token, senha)
    if numero is None:
        raise HTTPException(400, "senha inválida")
    falta, espera = request.app.state.operacao.posicao(numero)
    if falta == 0:
        return {"admitido": True, "passe": op.passe_emitir(cfg.segredo_token, numero, cfg.admissao_ttl_s)}
    return {"admitido": False, "posicao": falta, "espera_estimada_s": round(espera)}


app.include_router(candidato.rotas)
app.include_router(banca.rotas)
app.include_router(provedor.rotas)
