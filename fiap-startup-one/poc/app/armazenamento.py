"""Armazenamento de arquivos atrás de interface: disco local na POC, S3 depois.

A chave é derivada do hash, então reenviar o mesmo arquivo não duplica bytes, e
o hash gravado no banco prova anos depois qual arquivo foi analisado.
"""

import hashlib
from pathlib import Path
from typing import Protocol


class Armazenamento(Protocol):
    def guardar(self, prefixo: str, dados: bytes) -> tuple[str, str]:
        """Devolve (chave, sha256)."""
        ...

    def ler(self, chave: str) -> bytes: ...

    def publicar(self, caminho: str, dados: bytes, tipo: str) -> None:
        """Grava um arquivo público (resultado estático), servido pela CDN."""
        ...


class DiscoLocal:
    def __init__(self, raiz: str | Path):
        self.raiz = Path(raiz)
        (self.raiz / "privado").mkdir(parents=True, exist_ok=True)
        (self.raiz / "publico").mkdir(parents=True, exist_ok=True)

    def guardar(self, prefixo: str, dados: bytes) -> tuple[str, str]:
        sha = hashlib.sha256(dados).hexdigest()
        chave = f"{prefixo}/{sha[:2]}/{sha}"
        destino = self.raiz / "privado" / chave
        if not destino.exists():
            destino.parent.mkdir(parents=True, exist_ok=True)
            temp = destino.with_suffix(".parcial")
            temp.write_bytes(dados)
            temp.replace(destino)  # atômico: ninguém lê arquivo pela metade
        return chave, sha

    def ler(self, chave: str) -> bytes:
        return (self.raiz / "privado" / chave).read_bytes()

    def publicar(self, caminho: str, dados: bytes, tipo: str) -> None:
        destino = self.raiz / "publico" / caminho
        destino.parent.mkdir(parents=True, exist_ok=True)
        temp = destino.with_name(destino.name + ".parcial")
        temp.write_bytes(dados)
        temp.replace(destino)
