"""Worker: consome a fila de tarefas fora do request (dívida D3).

Uso: python -m app.worker [--uma-vez]

Toda tarefa é idempotente, porque a entrega é "pelo menos uma vez": se o worker
morrer no meio, a trava vence e outro worker refaz.
"""

import json
import logging
import sys
import time
from datetime import datetime, timezone

import psycopg
from psycopg.rows import dict_row
from psycopg.types.json import Jsonb

from app import pdf
from app.armazenamento import Armazenamento, DiscoLocal
from app.config import carregar
from app.pagamento.provedor import Provedor, Sandbox

log = logging.getLogger("certamis.worker")


class FilaPostgres:
    """A fila local. Na AWS, a mesma interface sobre SQS."""

    def __init__(self, conn: psycopg.Connection):
        self.conn = conn

    def pegar(self, tipos: list[str], n: int, trava_s: int) -> list[dict]:
        with self.conn.cursor(row_factory=dict_row) as cur:
            cur.execute("SELECT id, tipo, chave, dados, tentativas FROM pegar_tarefas(%s, %s, %s)",
                        (tipos, n, trava_s))
            linhas = cur.fetchall()
        self.conn.commit()
        return linhas

    def concluir(self, tid: int) -> None:
        self.conn.execute("SELECT concluir_tarefa(%s)", (tid,))
        self.conn.commit()

    def falhar(self, tid: int, erro: str) -> None:
        self.conn.execute("SELECT falhar_tarefa(%s, %s)", (tid, erro))
        self.conn.commit()


# ------------------------------------------------------------------ tarefas

def registrar_cobranca(conn, arm: Armazenamento, prov: Provedor, dados: dict) -> None:
    with conn.cursor(row_factory=dict_row) as cur:
        cur.execute("SELECT id, referencia, meio, valor_centavos, vence_em, status, registrada_em"
                    " FROM cobranca WHERE id = %s", (dados["cobranca_id"],))
        c = cur.fetchone()
    if c is None or c["status"] != "PENDENTE" or c["registrada_em"] is not None:
        return  # já feito, ou não precisa mais: idempotente
    instrucoes = prov.registrar(c["referencia"], c["meio"], c["valor_centavos"],
                                c["vence_em"].isoformat())
    conn.execute("UPDATE cobranca SET instrucoes = %s, registrada_em = now()"
                 " WHERE id = %s AND registrada_em IS NULL", (Jsonb(instrucoes), c["id"]))
    conn.commit()


def gerar_cartao(conn, arm: Armazenamento, prov: Provedor, dados: dict) -> None:
    with conn.cursor(row_factory=dict_row) as cur:
        cur.execute(
            "SELECT i.id, i.estado, i.cartao_chave, ca.nome AS candidato, ca.cpf, ce.nome AS certame,"
            "       cg.nome AS cargo, ci.nome AS cidade, m.nome AS modalidade"
            "  FROM inscricao i JOIN candidato ca ON ca.id = i.candidato_id"
            "  JOIN certame ce ON ce.id = i.certame_id JOIN cargo cg ON cg.id = i.cargo_id"
            "  JOIN cidade_prova ci ON ci.id = i.cidade_id JOIN modalidade m ON m.id = i.modalidade_id"
            " WHERE i.id = %s", (dados["inscricao_id"],))
        i = cur.fetchone()
    if i is None or i["estado"] != "HOMOLOGADA" or i["cartao_chave"]:
        return
    cpf_mascarado = f"***.{i['cpf'][3:6]}.{i['cpf'][6:9]}-**"
    conteudo = pdf.pagina([
        (18, "Cartão de confirmação de inscrição"),
        (12, i["certame"]),
        (11, ""),
        (11, f"Inscrição nº {i['id']}"),
        (11, f"Candidato: {i['candidato']}  ·  CPF {cpf_mascarado}"),
        (11, f"Cargo: {i['cargo']}"),
        (11, f"Modalidade: {i['modalidade']}"),
        (11, f"Cidade de prova: {i['cidade']}"),
        (11, ""),
        (9, "Local e sala serão informados no ensalamento. Documento gerado pela POC Certamis."),
    ])
    chave, _ = arm.guardar("cartao", conteudo)
    conn.execute("UPDATE inscricao SET cartao_chave = %s WHERE id = %s AND cartao_chave IS NULL",
                 (chave, i["id"]))
    conn.commit()


def publicar_homologacao(conn, arm: Armazenamento, prov: Provedor, dados: dict) -> None:
    """Publica a situação de todas as inscrições como arquivo. A consulta do candidato
    no pico de leitura vai ao arquivo, não ao banco."""
    certame_id = int(dados["certame_id"])
    with conn.cursor(row_factory=dict_row) as cur:
        cur.execute(
            "SELECT i.id AS inscricao, ca.nome, i.cargo_id, i.modalidade_id, i.estado"
            "  FROM inscricao i JOIN candidato ca ON ca.id = i.candidato_id"
            " WHERE i.certame_id = %s AND i.estado IN"
            "       ('HOMOLOGADA', 'INDEFERIDA', 'INDEFERIDA_DEFINITIVA', 'DEFERIDA', 'EM_ANALISE')"
            " ORDER BY i.id", (certame_id,))
        linhas = cur.fetchall()
    gerado = datetime.now(timezone.utc).isoformat()
    lista = [l for l in linhas if l["estado"] == "HOMOLOGADA"]
    arm.publicar(f"certames/{certame_id}/homologacao.json",
                 json.dumps({"certame": certame_id, "gerado_em": gerado, "homologadas": lista},
                            ensure_ascii=False).encode(), "application/json")
    # Fatias de 1.000 inscrições: o candidato baixa um arquivo pequeno, não a lista inteira.
    fatias: dict[int, dict] = {}
    for l in linhas:
        fatias.setdefault(l["inscricao"] // 1000, {})[str(l["inscricao"])] = l["estado"]
    for n, mapa in fatias.items():
        arm.publicar(f"certames/{certame_id}/situacao/{n}.json",
                     json.dumps({"gerado_em": gerado, "situacao": mapa}).encode(), "application/json")


TAREFAS = {
    "registrar_cobranca": registrar_cobranca,
    "gerar_cartao": gerar_cartao,
    "publicar_homologacao": publicar_homologacao,
}


def rodar(uma_vez: bool = False, lote: int = 20, trava_s: int = 60) -> int:
    cfg = carregar()
    arm = DiscoLocal(cfg.pasta_arquivos)
    prov = Sandbox()
    feitas = 0
    ultima_expiracao = 0.0
    with psycopg.connect(cfg.database_url) as conn:
        fila = FilaPostgres(conn)
        while True:
            if time.monotonic() - ultima_expiracao > 60:
                n = conn.execute("SELECT expirar_pagamentos_vencidos()").fetchone()[0]
                conn.commit()
                if n:
                    log.info("expiradas %d inscrições com prazo de pagamento vencido", n)
                ultima_expiracao = time.monotonic()
            tarefas = fila.pegar(list(TAREFAS), lote, trava_s)
            for t in tarefas:
                try:
                    TAREFAS[t["tipo"]](conn, arm, prov, t["dados"])
                    fila.concluir(t["id"])
                    feitas += 1
                except Exception as e:  # registra, recua e segue: uma tarefa ruim não para a fila
                    conn.rollback()
                    log.exception("tarefa %s %s falhou", t["tipo"], t["chave"])
                    fila.falhar(t["id"], f"{type(e).__name__}: {e}")
            if uma_vez and not tarefas:
                return feitas
            if not tarefas:
                time.sleep(0.5)


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(name)s %(message)s")
    print("tarefas feitas:", rodar(uma_vez="--uma-vez" in sys.argv))
