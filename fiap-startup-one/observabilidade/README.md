# Observabilidade — o sistema em tempo real, na máquina local

O que está aqui sobe o **Prometheus** (o coletor de séries temporais) e o **Grafana** (os painéis),
para ver a Lisura mudando enquanto um ensaio de carga roda. Na AWS o mesmo papel é do painel do
CloudWatch em [`../infra/terraform/observabilidade.tf`](../infra/terraform/observabilidade.tf).

```powershell
.\subir-observabilidade.ps1 -SegredoOperacao <o LISURA_OPERACAO_SEGREDO da API> -TokenKey <a LISURA_TOKEN_KEY da API>
```

| Programa | Endereço | O que faz |
|---|---|---|
| Prometheus 3.13.5 | 127.0.0.1:9090 | Raspa `GET /metrics` da API (8082) e do executor de ensaios (8083) a cada 5 s; recebe as métricas do k6 por remote write; guarda 7 dias |
| Grafana 13.2.4 | 127.0.0.1:3000 | O painel "Lisura — operação em tempo real"; só leitura, sem login, tema claro, embutível na página `/operacao` do front |
| Executor de ensaios | 127.0.0.1:8083 | `poc/cmd/ensaiador`: roda o k6 e a auditoria quando a página `/operacao` pede |

Os dois programas são binários nativos de Windows, com o SHA-256 conferido contra o publicado. O
Docker desta máquina só roda contêineres Windows, e as imagens oficiais são Linux.

## Os arquivos

| Arquivo | O que é |
|---|---|
| `prometheus.yml` | O que raspar. Uma segunda instância da API entra em `instancias.json`, sem reiniciar |
| `instancias.json` | Instâncias extras da API (file_sd); vazio por padrão |
| `grafana/grafana.ini` | Sem login, sem formulário de entrada, sem criar administrador, sem telemetria; embutível |
| `grafana/provisionamento/` | A fonte de dados (o Prometheus) e onde estão os painéis |
| `grafana/paineis/lisura.json` | **O painel**, versionado: instâncias, nível de degradação, tráfego e p95 por rota, fila (pendentes, espera, vazão), conexões do banco, CPU e memória da API, e o k6 (VUs, requisições, p95 visto pelo candidato) |

## As anotações

O painel marca sozinho, como uma linha vertical:
- o início e o fim de cada ensaio (`lisura_ensaio_rodando`, do executor);
- cada mudança do nível de degradação (`lisura_degradacao_nivel`);
- cada mudança no número de instâncias da API no ar (`lisura_instancia_info`).

## As métricas da API

Estão descritas em [`../poc/internal/metricas/metricas.go`](../poc/internal/metricas/metricas.go) e
em `poc/README.md`, "Documentação viva e observabilidade". A regra que importa: a rota entra no
rótulo pelo **padrão** (`GET /inscricoes/{id}`), nunca pela URL, senão cada inscrição viraria uma série.

## Limites

- Só a máquina local. Na AWS, o `/metrics` de cada tarefa fica dentro da VPC, e levá-lo a um
  Prometheus e Grafana gerenciados **ainda não existe**.
- Os dados e os logs ficam em `C:\Users\rockb\tools\observabilidade-dados`, fora do repositório.
