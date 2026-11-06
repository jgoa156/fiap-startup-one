# Infraestrutura da POC — o ambiente do ensaio (semana 6)

Tudo o que o ensaio de carga precisa, em código: **Terraform** para a AWS, **Dockerfiles** construídos
pelo **CodeBuild** e o **gerador de carga k6** em [`../carga`](../carga). Subir, medir e devolver:
o ambiente inteiro nasce com um `apply` e some com um `destroy`.

> **Estado em 06/11/2026: escrito e validado localmente (`terraform validate`), NÃO aplicado na AWS.**
> A página **/docs/infra** do front mostra este Terraform lido do repositório: cada recurso com o
> comentário que o explica, quem usa quem, variáveis e saídas (gerada por `poc/cmd/infradoc`; um teste
> quebra a suíte se ela ficar para trás de um `.tf`).
>
> **Até 04/11:** Esta máquina não tem
> credencial AWS configurada, e quem configura credencial e aceita o custo é o dono da conta. O que
> foi conferido: `terraform validate` passa; o código novo da aplicação (sala de espera no Postgres,
> S3) tem testes; o k6 rodou contra o ambiente local e a auditoria conferiu o resultado (ver
> `../poc/README.md`, semana 6). **O que só a AWS vai mostrar:** se o `apply` sobe sem erro, se as
> imagens constroem no CodeBuild e se as tarefas ficam saudáveis.

## O desenho

```
candidato / k6 ──► CloudFront ─┬─ /publico/*      ──► S3 (cartão, relação, certame) — com cache
                               ├─ /_next/static/* ──► balanceador ──► front Next — com cache
                               ├─ /api/*          ──► balanceador ──► API Go (N tarefas) ──► Aurora
                               └─ resto           ──► balanceador ──► front Next      │
                                                                       │        S3 (documentos)
                                         API ◄── aviso de pagamento ── PSP simulado (DNS interno)
```

| Arquivo | O que cria |
|---|---|
| `terraform/rede.tf` | VPC em 2 zonas; sub-redes públicas (balanceador, tarefas) e privadas sem saída (banco); grupos de segurança |
| `terraform/banco.tf` | **Aurora PostgreSQL 17 Serverless v2**, senha criada e guardada pelo próprio RDS, Performance Insights ligado |
| `terraform/armazenamento.tf` | Bucket com `publico/` (só o CloudFront lê), `documentos/` (só a API) e `fonte/` (o CodeBuild) |
| `terraform/cdn.tf` | **CloudFront**: a porta única, com as quatro rotas do desenho |
| `terraform/balanceador.tf` | Balanceador que só aceita o CloudFront (lista de IPs da AWS **e** cabeçalho secreto) |
| `terraform/ecs.tf` | **ECS Fargate**: serviços API, front e PSP simulado; tarefas avulsas seed, auditoria e k6; papéis IAM mínimos |
| `terraform/imagens.tf` | Repositórios ECR e o projeto CodeBuild |
| `terraform/segredos.tf` | Chave dos tokens, segredo de operação e do PSP, gerados e entregues pelo Secrets Manager |
| `terraform/cofre.tf` | Bucket com **bloqueio de objeto** para as âncoras da trilha (a API escreve, ninguém sobrescreve) |
| `terraform/ses.tf` | Só com `email_de`: conjunto de configuração do **SES**, tópico **SNS** dos eventos (devolução, reclamação, entrega) e a inscrição da API nele (`POST /api/ses/eventos`, só com `ligado=true`) |
| `terraform/observabilidade.tf` | Painel do **CloudWatch** para acompanhar o ensaio: tarefas no ar por serviço (a escala), CPU e memória, requisições e 5xx no balanceador, p95 com a linha do H1, ACU e conexões do Aurora. Saída `painel_operacao` |
| `imagens/*.Dockerfile` | Go (4 binários, base distroless), front Next (modo standalone), k6 (binário oficial, checksum conferido) |
| `scripts/publicar-imagens.ps1` | Empacota a fonte, constrói no CodeBuild, publica no ECR |
| `scripts/tarefa.ps1` | Roda seed, k6 ou auditoria no ECS e traz o log |

## Onde a POC difere da arquitetura alvo, e por quê

A regra do escopo é *"a configuração do ensaio é a mesma que iria para produção"*. Ela vale para
**esta** configuração: é a que a POC levaria ao primeiro cliente. Ela **não** é a arquitetura alvo
do ano 5, e as diferenças são estas:

| Arquitetura alvo | POC | Por quê | O que isso faz com o resultado |
|---|---|---|---|
| EKS + Karpenter | **ECS Fargate** | O EKS cobra o plano de controle por hora e exige manifestos, controlador de ingresso e Karpenter: semanas de trabalho para uma pessoa. A hipótese H1 é sobre banco, conexões, fila e publicação estática, não sobre o orquestrador | O tempo para subir tarefas novas é o do Fargate (~1 min), não o de um *node pool* pré-aquecido. Por isso o ensaio sobe a capacidade **antes**, como a agenda faria |
| Cloudflare Waiting Room | Sala de espera **na API**, estado no Postgres | Sem domínio nem conta Cloudflare | A fila consome um pouco da API e do banco (um `nextval` por visitante novo). Na borda, não consumiria nada |
| SQS + Lambda | Fila no Postgres, trabalhadores dentro da API | Já provada na semana 4; uma peça a menos | **Risco real que o ensaio mede:** a fila disputa conexões e CPU do mesmo banco que a inscrição |
| Células | Uma célula | Multi-tenant está fora do escopo | Nenhum, para um certame só |
| HTTPS até a origem | CloudFront → balanceador em **HTTP** | Sem domínio próprio não há certificado no balanceador. Mitigado: o balanceador só aceita IPs do CloudFront **e** um cabeçalho secreto; os dados são sintéticos | Nenhum no desempenho. Em produção: domínio + certificado ACM no balanceador |
| Tarefas em sub-rede privada com NAT | Tarefas em sub-rede **pública** com IP próprio e grupo de segurança fechado | NAT Gateway cobra por hora mesmo parado | Nenhum no desempenho; o banco continua sem saída para a internet |
| Escalonamento por agenda | Capacidade definida à mão, por variável | Na semana 7 o ensaio sobe a capacidade antes, à mão; automatizar é a onda 2 | — |
| TLS do banco conferido | `sslmode=require` (cifra, não confere o certificado) | O certificado do RDS não é de autoridade pública; conferir exige embutir o pacote da AWS na imagem | Nenhum no desempenho. Pendente para produção |

## Como subir (quem tem a credencial)

Uma vez: configurar a AWS CLI com uma credencial da conta (`aws configure sso` ou chave de acesso)
e conferir com `aws sts get-caller-identity`. Ferramentas instaladas em `C:\Users\rockb\tools\`
(`terraform\`, `awscli\Amazon\AWSCLIV2\`, `k6\`).

```powershell
cd infra\terraform
terraform init
terraform apply                                    # 1ª vez: rede, banco, bucket, ECR, CodeBuild (tarefas = 0)
..\scripts\publicar-imagens.ps1                    # imprime a tag
terraform apply -var tag_imagem=<tag> -var ligado=true -var banco_acu_min=8
..\scripts\tarefa.ps1 seed                         # imprime o certame_id
..\scripts\tarefa.ps1 k6 -Ambiente @{ CERTAME='<id>'; MULT='10'; DURACAO='15m'; RODADA='100001' }
..\scripts\tarefa.ps1 auditoria -Comando '/lisura-auditoria','-rodada','100001','-enviadas','<n>','-pagas','<n>'
terraform apply -var tag_imagem=<tag> -var ligado=false   # tarefas a zero, banco pausa sozinho
terraform destroy                                  # entre ensaios: o balanceador e o CloudFront cobram parados
```

O k6 imprime, no fim, a linha `PARA_AUDITAR` com o comando da auditoria já preenchido.

**Acompanhar em tempo real:** abrir a saída `painel_operacao` (CloudWatch). As métricas da própria
aplicação (fila, sala de espera, conexões do pool) estão em `GET /metrics` de cada tarefa, dentro da
VPC. Levá-las a um Prometheus e Grafana gerenciados ainda não está aqui; na máquina local, o
Prometheus e o Grafana de `../observabilidade` fazem esse papel.

**E-mail de verdade:** `-var email_de=avisos@<domínio>` troca o provedor de avisos para o SES. O
remetente (ou o domínio, com DKIM) precisa estar verificado no SES da região. Antes de produção,
confira a cota da conta: a 40× são ~39 avisos por segundo.

**O ensaio inteiro num comando** (`scripts/ensaio.ps1`), no lugar dos passos de k6 e auditoria
acima. Ele roda os patamares em ordem. Em cada um:
- roda o k6 e guarda o log;
- roda a auditoria daquela rodada;
- anota a janela de horário (UTC);
- escreve o manifesto e refaz o relatório.

Ele para no primeiro patamar que falhar, que é o ponto de quebra. Com `-AteQuebrar`, depois da lista
continua dobrando a carga até falhar ou chegar ao `-Teto`. Cada patamar usa uma faixa de CPF
própria.

```powershell
..\scripts\ensaio.ps1 -Onde aws -Certame <id> -Patamares 10,20,40 -Duracao 3h -AteQuebrar `
  -Titulo 'Ensaio do H1 — AWS sa-east-1' -Ambiente '<tarefas, ACUs, versão>'
```

Os arquivos vão para `carga\resultados\ensaio-<data>\`. O custo de cada patamar fica em 0 no
manifesto: preencha `custo_usd` com o Cost Explorer, usando a janela que está no relatório, e rode
`poc\bin\relatorio.exe -manifesto <pasta>\manifesto.json -saida <pasta>\relatorio.md` de novo.
**Validado só na máquina local** (`-Onde local`); o caminho da AWS usa o `tarefa.ps1`, que não
rodou contra a AWS real.

**Dois defeitos do `tarefa.ps1` corrigidos antes de rodar** (05/11, achados lendo o código):
- **Log cortado:** o log vinha só na primeira página, até 10 mil eventos ou 1 MB. Num k6 de 3 h,
  a linha `RESUMO_DO_ENSAIO`, que sai no fim, ficaria de fora.
- **Linhas emendadas:** em `--output text` o CLI emendava as mensagens com tabulação.

Agora o log segue as páginas, vem em JSON e pode ser gravado com `-Saida`.

**Custo.** O que cobra por hora com tudo ligado: tarefas Fargate (API, front, PSP, k6), Aurora
(por ACU), balanceador, Container Insights e Performance Insights. Com `ligado=false` sobram o
balanceador, o armazenamento e o mínimo do Aurora (0 = pausado); por isso a recomendação é
`destroy` entre ensaios. O custo medido de cada patamar é entregável da semana 7 (Cost Explorer com
a etiqueta `Projeto=lisura-poc`, que todo recurso recebe).

**O estado do Terraform é sensível**: guarda os segredos gerados. Fica nesta máquina
(`terraform.tfstate`, fora do controle de versão pelo `.gitignore`). Para mais de uma pessoa operar,
movê-lo para um bucket S3 com trava.
