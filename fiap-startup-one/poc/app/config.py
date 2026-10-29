"""Configuração por variável de ambiente. Nada de segredo no código."""

import os
from dataclasses import dataclass


@dataclass(frozen=True)
class Config:
    database_url: str
    # Pool de conexões explícito (dívida D2): o tamanho é decisão de capacidade,
    # medida no ensaio, e não um padrão escondido do driver.
    pool_min: int
    pool_max: int
    pool_timeout_s: float


def carregar() -> Config:
    return Config(
        database_url=os.environ.get(
            "CERTAMIS_DATABASE_URL",
            "postgresql://certamis:certamis@localhost:5432/certamis",
        ),
        pool_min=int(os.environ.get("CERTAMIS_POOL_MIN", "2")),
        pool_max=int(os.environ.get("CERTAMIS_POOL_MAX", "10")),
        pool_timeout_s=float(os.environ.get("CERTAMIS_POOL_TIMEOUT_S", "5")),
    )
