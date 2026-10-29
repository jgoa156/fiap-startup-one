"""Estimativa do custo por hora de pico na AWS, para uma configuração dada.

Uso: python carga/custo.py --tarefas 4 --vcpu 1 --gb 2 --arm --rds db.t4g.small

É ESTIMATIVA, não medição: a POC roda local porque ainda não há conta AWS. O
número de tarefas vem do ensaio local (CPU por inscrição); o preço, da tabela
abaixo, com fonte e data. Quando houver conta, um ensaio curto lá substitui
esta conta pela fatura real.

Fora da conta, declarado: balanceador (ALB), transferência de dados, CloudFront,
CloudWatch, armazenamento do RDS e S3. Na hora de pico de uma POC são pequenos
perto de computação e banco, mas não são zero.
"""

import argparse

# Preços on-demand, USD, sa-east-1 (São Paulo). Conferidos em 29/10/2026.
PRECOS = {
    # vmprices.com/serverless/fargate, "checked 29 October 2026". Os valores de
    # us-east-1 da mesma página batem com o preço oficial da AWS (0,04048/vCPU-h).
    "fargate_x86_vcpu_h": 0.0696,
    "fargate_x86_gb_h": 0.0076,
    "fargate_arm_vcpu_h": 0.0557,
    "fargate_arm_gb_h": 0.00612,
    # RDS PostgreSQL, instância (computação) apenas.
    # db.t4g.small: sparecores.com, 0,069/h single-AZ e 0,137/h multi-AZ; bytebase.com, US$ 50/mês.
    "rds_db.t4g.small": 0.069,
    "rds_db.t4g.small_multiaz": 0.137,
    # db.m7g.large: bytebase.com, US$ 161/mês (atualizado em 26/10/2026), ÷ 730 h.
    "rds_db.m7g.large": 161 / 730,
}
DOLAR = 5.3738          # USD/BRL comercial, 29/10/2026 13h11 (renovainvest.com.br)
ALVO_POR_HORA = 16_700  # H1
PRECO_INSCRICAO = 3.00  # R$, base do plano de negócios (não verificado)


def custo_hora(tarefas: int, vcpu: float, gb: float, arm: bool, rds: str) -> dict:
    arq = "arm" if arm else "x86"
    fargate = tarefas * (vcpu * PRECOS[f"fargate_{arq}_vcpu_h"] + gb * PRECOS[f"fargate_{arq}_gb_h"])
    banco = PRECOS[f"rds_{rds}"]
    total_usd = fargate + banco
    total_brl = total_usd * DOLAR
    por_inscricao = total_brl / ALVO_POR_HORA
    return {
        "configuracao": f"{tarefas} × Fargate {arq} {vcpu} vCPU/{gb} GB + RDS {rds}",
        "fargate_usd_h": round(fargate, 4),
        "rds_usd_h": round(banco, 4),
        "total_usd_h": round(total_usd, 4),
        "total_brl_h": round(total_brl, 2),
        "brl_por_inscricao_no_pico": round(por_inscricao, 5),
        "fracao_do_preco_por_inscricao": f"{por_inscricao / PRECO_INSCRICAO:.3%}",
    }


if __name__ == "__main__":
    ap = argparse.ArgumentParser()
    ap.add_argument("--tarefas", type=int, default=4)
    ap.add_argument("--vcpu", type=float, default=1)
    ap.add_argument("--gb", type=float, default=2)
    ap.add_argument("--arm", action="store_true")
    ap.add_argument("--rds", default="db.t4g.small")
    a = ap.parse_args()
    for k, v in custo_hora(a.tarefas, a.vcpu, a.gb, a.arm, a.rds).items():
        print(f"{k:32} {v}")
