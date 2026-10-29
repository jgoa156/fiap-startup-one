# Relatório de ensaio de carga: Certamis POC

*Modelo. Cada campo `⟨…⟩` é preenchido a partir de um arquivo de `relatorio/bruto/`, e a coluna
"de onde" diz qual. Nenhum número entra de memória ou de tela.*

## 1. A pergunta

> **H1: o sistema absorve 16,7 mil inscrições por hora, sustentadas por 3 horas, sem perder
> inscrição, sem cobrar duas vezes e sem estourar o custo previsto de infraestrutura.**

16,7 mil por hora é 40× a média **do certame de referência** (300 mil candidatos em 30 dias,
417 por hora), e não da empresa. Resultado negativo também é entregável, se disser onde e por quê
quebrou.

## 2. Resultado em uma tabela

| Métrica | Alvo | Medido | De onde |
|---|---|---|---|
| Inscrições perdidas | zero | ⟨…⟩ | `*-verificacao.json` → `aceite.inscricoes_perdidas` |
| Cobranças duplicadas sob repetição e reenvio | zero | ⟨…⟩ | `*-verificacao.json` → `aceite.cobrancas_duplicadas` |
| p95 no pico | < 2 s | ⟨…⟩ ms | `*-verificacao.json` → `k6.p95_critico_ms` |
| Conclusão do fluxo completo | > 98% | ⟨…⟩ | `k6.taxa_conclusao` (abandono deliberado fica fora) |
| **Ponto de quebra** | medido | ⟨…⟩ inscrições/h | ensaio escada: primeiro degrau com p95 > 2 s ou conclusão < 98% |
| Custo por hora de pico | medido | **estimado**: ⟨…⟩ US$ · ⟨…⟩ R$ | `carga/custo.py` com a configuração do §5 |

**Veredito:** ⟨passou / não passou⟩. ⟨Uma frase: onde quebrou e qual foi o gargalo.⟩

## 3. Como o ensaio foi feito

| | |
|---|---|
| Máquina | ⟨processador, núcleos, memória⟩. k6, aplicação, worker e banco **na mesma máquina** ⟨ou não⟩ |
| Aplicação | ⟨N⟩ processos uvicorn, pool de ⟨N⟩ conexões cada |
| Banco | Postgres 16 em contêiner, ⟨limites, se houver⟩ |
| Dado | 100% sintético. ⟨N⟩ contas semeadas |
| Perfil | ⟨patamar⟩ por hora em média; rajadas de 90 s a **⟨F⟩×** a cada 10 min. Fator 3× confirmado em 29/10/2026 (`perfil.fator_rajada_confirmado`) |
| Planejado pelo perfil | ⟨N⟩ inscrições em ⟨h⟩ h (`perfil.planejado` no JSON do k6) |
| Comportamento simulado | abandono ⟨8%⟩, recarga ⟨15%⟩, clique duplo na cobrança ⟨10%⟩, outra aba ⟨3%⟩, reenvio do aviso de pagamento ⟨5%⟩. **Estimativas nossas** |

### As duas unidades

| | |
|---|---|
| Inscrições finalizadas por hora | ⟨…⟩ |
| Requisições por inscrição finalizada | ⟨…⟩ (`k6.requisicoes_por_inscricao_finalizada`) |
| Requisições por segundo no pico da rajada | ⟨…⟩ |

## 4. Curva de quebra (ensaio escada)

| Degrau (inscrições/h) | Requisições/s | p95 crítico | Conclusão | CPU app | CPU banco | Observação |
|---|---|---|---|---|---|---|
| 4.000 | | | | | | |
| 8.000 | | | | | | |
| 16.700 | | | | | | |
| 25.000 | | | | | | |
| 33.400 | | | | | | |
| 50.000 | | | | | | |

**Gargalo identificado:** ⟨pool de conexões, CPU do login, CPU do banco, fila do worker...⟩.
**Evidência:** ⟨qual métrica mostrou⟩.

### Antes e depois da correção (semana 8)

| | Antes | Depois |
|---|---|---|
| Ponto de quebra | | |
| p95 a 16,7 mil/h | | |
| O que mudou | | |

## 5. Custo

**É estimativa, não medição**, porque o ensaio rodou local: ainda não há conta AWS. Preços
on-demand de São Paulo conferidos em 29/10/2026, câmbio de R$ 5,3738; fontes em `carga/custo.py`.

| Configuração equivalente na AWS | US$/h | R$/h | R$ por inscrição no pico |
|---|---|---|---|
| ⟨N⟩ × Fargate ⟨…⟩ + RDS ⟨…⟩ | | | |

O número de tarefas sai do ensaio: CPU consumida por inscrição aqui, convertida em vCPU.
Fora da conta, declarado: balanceador, transferência, CDN, monitoramento e armazenamento.

> **O que este número diz, e o que não diz.** Uma configuração de exemplo (4 tarefas de 1 vCPU
> mais um RDS pequeno) custa uns R$ 1,83 por hora, ou **R$ 0,0001 por inscrição** a 16,7 mil
> por hora: **0,004% do preço** de R$ 3,00. A infraestrutura do pico não é o que decide o piso
> de R$ 2,29 do plano de negócios; o piso depende de pessoas e de vendas. A POC mostra que o
> custo do pico **não ameaça** o preço. **Não mostra que o preço está certo**, e o pitch não pode
> sugerir que mostra.

## 6. O que este ensaio NÃO prova

- **Não testa a restauração no Google Cloud.** O gatilho 3 do trabalho de multicloud (restaurar
  em até 8 h) continua estimativa.
- **Não mede a AWS.** Mede esta máquina. A ordem de grandeza da capacidade se transfere; o número
  exato, não.
- **Não usa provedor de pagamento real.** O sandbox é simulado e reproduz o que importa
  (referência nossa e aviso repetido), não a latência de um provedor real.
- **Não usa a fila de espera da borda.** Na arquitetura-alvo ela fica no Cloudflare; aqui fica na
  aplicação, com o mesmo contrato.
- ⟨Outros limites que aparecerem.⟩

## 7. Reprodução

`carga/ROTEIRO-ENSAIO.md`, com os arquivos brutos em `relatorio/bruto/`.
