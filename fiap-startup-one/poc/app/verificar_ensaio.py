"""Confere um ensaio no banco: o que o k6 diz que aconteceu, aconteceu?

Uso: python -m app.verificar_ensaio relatorio/bruto/k6-patamar-....json --desde 2026-11-03T14:00:00-03:00

"Zero inscrições perdidas" e "zero cobranças duplicadas" são as duas métricas que
viram processo judicial e administrativo. Elas não podem vir da palavra do
gerador de carga: são contadas aqui, no banco, depois do ensaio.

Perdida: o k6 recebeu 200 ao finalizar, mas o banco não tem a transição.
Duplicada: mais de uma cobrança viva, mais de uma transição para PAGA, ou mais
de um evento aplicado, para a mesma inscrição.
"""

import argparse
import json
import sys
from datetime import datetime
from pathlib import Path

from app.config import carregar


def _metrica(k6: dict, nome: str, campo: str = "count") -> float:
    m = k6.get("metrics", {}).get(nome)
    return float(m["values"].get(campo, 0)) if m else 0.0


def verificar(resumo_k6: dict, desde: str, url: str) -> dict:
    import psycopg

    k6 = resumo_k6["k6"]
    finalizadas_k6 = int(_metrica(k6, "finalizadas_ok"))
    aplicados_k6 = int(_metrica(k6, "pagamentos_aplicados"))
    concluidas = int(_metrica(k6, "inscricoes_concluidas"))
    falhas = int(_metrica(k6, "inscricoes_falhas"))
    abandonos = int(_metrica(k6, "inscricoes_abandonadas"))
    requisicoes = int(_metrica(k6, "http_reqs"))

    with psycopg.connect(url) as c:
        q = lambda sql, *a: c.execute(sql, (desde, *a)).fetchone()[0]
        finalizadas_db = q(
            "SELECT count(DISTINCT inscricao_id) FROM inscricao_transicao"
            " WHERE para = 'AGUARDANDO_PAGAMENTO' AND em >= %s")
        aplicados_db = q("SELECT count(*) FROM pagamento_evento WHERE resultado = 'APLICADO'"
                         " AND recebido_em >= %s")
        duplicados_evento = q("SELECT count(*) FROM pagamento_evento WHERE resultado = 'COBRANCA_INATIVA'"
                              " AND recebido_em >= %s")
        pagas_duas_vezes = q(
            "SELECT count(*) FROM (SELECT inscricao_id FROM inscricao_transicao"
            " WHERE para = 'PAGA' AND em >= %s GROUP BY inscricao_id HAVING count(*) > 1) x")
        vivas_duplicadas = c.execute(
            "SELECT count(*) FROM (SELECT inscricao_id FROM cobranca WHERE status IN ('PENDENTE', 'PAGA')"
            " GROUP BY inscricao_id HAVING count(*) > 1) x").fetchone()[0]
        invariantes = c.execute(
            "SELECT tipo, count(*) FROM conciliacao WHERE tipo <> 'EVENTO_NAO_APLICADO'"
            " GROUP BY tipo").fetchall()
        cobrancas = q("SELECT count(*) FROM cobranca WHERE criada_em >= %s")
        fila = c.execute("SELECT tipo, estado, quantidade FROM fila_resumo ORDER BY 1, 2").fetchall()
        conexoes_max = c.execute("SHOW max_connections").fetchone()[0]

    perdidas = max(0, finalizadas_k6 - finalizadas_db)
    tentativas = concluidas + falhas
    resultado = {
        "desde": desde,
        "k6": {"concluidas": concluidas, "falhas": falhas, "abandonos": abandonos,
               "finalizadas_ok": finalizadas_k6, "pagamentos_aplicados": aplicados_k6,
               "requisicoes": requisicoes,
               "requisicoes_por_inscricao_finalizada": round(requisicoes / finalizadas_k6, 1) if finalizadas_k6 else None,
               "p95_critico_ms": round(_metrica(k6, "http_req_duration{fase:critica}", "p(95)")),
               "taxa_conclusao": round(concluidas / tentativas, 4) if tentativas else None},
        "banco": {"finalizadas": finalizadas_db, "pagamentos_aplicados": aplicados_db,
                  "eventos_reenviados_recusados": duplicados_evento, "cobrancas_criadas": cobrancas,
                  "fila": [list(f) for f in fila], "max_connections": conexoes_max},
        "aceite": {
            "inscricoes_perdidas": perdidas,
            "cobrancas_duplicadas": pagas_duas_vezes + vivas_duplicadas,
            "pagamentos_k6_igual_banco": aplicados_k6 == aplicados_db,
            "invariantes_de_conciliacao": {t: n for t, n in invariantes},
        },
    }
    a = resultado["aceite"]
    a["passou_dinheiro_e_registro"] = (perdidas == 0 and a["cobrancas_duplicadas"] == 0
                                       and a["pagamentos_k6_igual_banco"] and not invariantes)
    return resultado


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("resumo_k6")
    ap.add_argument("--desde", required=True, help="início do ensaio, ISO 8601 com fuso")
    args = ap.parse_args()
    datetime.fromisoformat(args.desde)  # falha cedo se a data vier errada
    resumo = json.loads(Path(args.resumo_k6).read_text(encoding="utf-8"))
    r = verificar(resumo, args.desde, carregar().database_url)
    saida = Path(args.resumo_k6).with_name(Path(args.resumo_k6).stem + "-verificacao.json")
    saida.write_text(json.dumps(r, indent=2, ensure_ascii=False, default=str), encoding="utf-8")
    print(json.dumps(r["aceite"], indent=2, ensure_ascii=False))
    print("detalhe em", saida)
    return 0 if r["aceite"]["passou_dinheiro_e_registro"] else 1


if __name__ == "__main__":
    sys.exit(main())
