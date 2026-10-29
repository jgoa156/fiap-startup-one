"""Configuração por variável de ambiente. Nada de segredo no código de produção:
os padrões abaixo servem só para desenvolvimento local."""

import os
from dataclasses import dataclass

SEGREDO_DEV = "dev-apenas-local-nao-usar-em-producao"


@dataclass(frozen=True)
class Config:
    database_url: str
    # Pool de conexões explícito (dívida D2): o tamanho é decisão de capacidade,
    # medida no ensaio, e não um padrão escondido do driver.
    pool_min: int
    pool_max: int
    pool_timeout_s: float
    segredo_token: str
    segredo_provedor: str
    pasta_arquivos: str
    sandbox: bool
    scrypt_n: int
    # Fila de espera (dívida D4): quantas sessões ativas cabem ao mesmo tempo.
    admissao_max_ativos: int
    admissao_ttl_s: int


def carregar() -> Config:
    e = os.environ.get
    return Config(
        database_url=e("CERTAMIS_DATABASE_URL", "postgresql://certamis:certamis@localhost:5432/certamis"),
        pool_min=int(e("CERTAMIS_POOL_MIN", "2")),
        pool_max=int(e("CERTAMIS_POOL_MAX", "10")),
        pool_timeout_s=float(e("CERTAMIS_POOL_TIMEOUT_S", "5")),
        segredo_token=e("CERTAMIS_SEGREDO_TOKEN", SEGREDO_DEV),
        segredo_provedor=e("CERTAMIS_SEGREDO_PROVEDOR", SEGREDO_DEV + "-provedor"),
        pasta_arquivos=e("CERTAMIS_PASTA_ARQUIVOS", "./dados"),
        sandbox=e("CERTAMIS_SANDBOX", "1") == "1",
        scrypt_n=int(e("CERTAMIS_SCRYPT_N", str(2**14))),
        admissao_max_ativos=int(e("CERTAMIS_ADMISSAO_MAX_ATIVOS", "500")),
        admissao_ttl_s=int(e("CERTAMIS_ADMISSAO_TTL_S", "900")),
    )
