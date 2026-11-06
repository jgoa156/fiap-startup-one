# Ensaio LOCAL de validacao da cadeia (nao e o resultado do H1)

**Ambiente:** maquina local Windows, API + PSP simulado + Next de producao, Postgres 17 local; tempo de pensar a 5%

> **H1** — o sistema absorve 40 vezes a carga média por 3 horas, sem perder inscrição, sem cobrar
> duas vezes e sem estourar o custo previsto de infraestrutura.

**H1 NÃO PASSOU**: a rodada 6131101 passou a 40×, mas durou 2m; o H1 pede 3 h.

**Ponto de quebra: não alcançado.** A maior carga ensaiada (40×) passou em tudo. Um teste que só passa não informa o limite: é preciso subir mais (04-escopo-poc §8).

## A curva

| Patamar | Rodada | Duração | Inscrições enviadas | p95 crítico | Concluídas | Perdidas | Duplicadas | Falhas do k6 | Custo/h | Custo/inscrição | Resultado |
|---:|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---|
| 40× | 6131101 | 2m | 1.744 | 2,3 ms | 100,00% | 0 | 0 | 0 | — | — | passou |

## 40× — rodada 6131101

| Critério | Medido | Alvo | |
|---|---:|---|---|
| Inscrições enviadas na rodada | 1744 | mais que zero | ✅ |
| Inscrições perdidas | 0 | zero | ✅ |
| Cobranças duplicadas | 0 | zero | ✅ |
| p95 das requisições críticas | 2,3 ms | < 2 s | ✅ |
| Jornadas concluídas | 100,00% | > 98% | ✅ |
| Problemas da auditoria | 0 | nenhum | ✅ |
| k6 e auditoria da mesma rodada | 6131101 · 6131101 | iguais | ✅ |

- O k6 viu **1.744 inscrições enviadas** e **1.563 pagamentos**; o banco tem 1.744 e 1.563.
- 35.085 requisições; p95 geral 2,2 ms; recarregamentos 0; cliques duplos em "Pagar" 97.
- p95 até o Pix aparecer 1,5 s; até o cartão 3,0 s; espera na sala de espera 0,0 s.
- Trilhas verificadas 1.802, quebradas 0; divergências da conciliação 0.
- Rodada curta (2m) so para validar o relatorio.

