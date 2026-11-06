# FIAP Startup One — Lisura

**Lisura** — plataforma que conduz um processo seletivo público de ponta a ponta, do edital à
homologação do resultado. Vendida para **bancas organizadoras**.

> *O nome vem de "lisura do certame" — o termo que a administração pública usa para a integridade
> do processo. É o que o produto protege.*

---

## Para apresentar

| Arquivo | O que é |
|---|---|
| **[`apresentacao/Lisura-pitch.pdf`](apresentacao/Lisura-pitch.pdf)** | **O pitch para investidores** — 10 slides, 16:9 (960×540 pt) |
| [`apresentacao/previews/`](apresentacao/previews/) | Os 10 slides em PNG, para revisar sem abrir o PDF |
| [`apresentacao/pitch-lisura.html`](apresentacao/pitch-lisura.html) | A **fonte** dos slides — edite aqui |
| [`apresentacao/build-pitch.ps1`](apresentacao/build-pitch.ps1) | Regera o PDF (Edge headless — sem Python nem Node) |

**A tese em uma frase:** um mercado inteiro é obrigado por lei a trocar de plataforma até 2028, e
o critério de escolha vai ser exatamente aquilo que ninguém resolve — **aguentar o último dia.**

---

## Documentos

| Arquivo | Conteúdo |
|---|---|
| [01-ideacao.md](01-ideacao.md) | A rodada de ideação que selecionou esta ideia — com os **seis motivos legítimos** para uma nuvem ser necessária, e as nove ideias verificadas contra eles |
| [02-produto-features-e-fluxos.md](02-produto-features-e-fluxos.md) | **O produto** — 9 módulos, fluxogramas em ASCII, máquina de estados da inscrição, recurso em escala sem IA, e as três decisões fechadas |
| [03-plano-de-negocios.md](03-plano-de-negocios.md) | **O plano** — mercado, preço com sensibilidade, unit economics, projeções de 5 anos, concorrência verificada, riscos |
| [04-escopo-poc.md](04-escopo-poc.md) | **A POC** — 1 pessoa, 8 semanas, ~US$ 100. O entregável é um relatório de ensaio de carga, não uma tela |

---

## A POC (prova de conceito)

| Pasta | O que é |
|---|---|
| [`poc/`](poc/README.md) | **A API em Go** (PostgreSQL 17, sem ORM): inscrição, isenção, pagamento Pix e boleto com PSP simulado, devolução, fila de tarefas no Postgres, publicação estática, sala de espera e modo degradado, trilha de auditoria encadeada com âncora num cofre, resultado e recurso, ensalamento, cronograma, prestação de contas, exportação para o órgão, avisos por e-mail (SES), métricas. **O estado de cada parte, com os testes e as medidas, está no README dela** |
| [`poc-web/`](poc-web/README.md) | **O portal do candidato** em Next.js + Ant Design, e as páginas de documentação viva e operação |
| [`infra/`](infra/README.md) | **Terraform da AWS** (ECS Fargate, Aurora Serverless v2, CloudFront, S3, SES, CloudWatch), imagens e scripts do ensaio. Validado, **não aplicado** (falta a credencial) |
| [`carga/`](carga/README.md) | **O gerador de carga k6** e o que "40×" quer dizer em número |
| [`observabilidade/`](observabilidade/README.md) | **Prometheus e Grafana** locais, com o painel do sistema em tempo real |

**Documentação viva** (com a API e o front no ar): `/docs` (a entrada), `/docs/api`, `/design`,
`/docs/infra` e `/operacao`. Cada página nasce do código ou do que está rodando, e um teste quebra a
suíte quando ela fica para trás.

**Estado em 06/11/2026:** 187 testes Go (199 com subtestes) passando, também com o detector de
corrida. O ensaio local a 640× passa em tudo (sem inscrição perdida, sem cobrança dupla). **O H1, 40×
por 3 h na AWS, ainda não foi ensaiado**: a semana 7 depende da credencial da AWS e do aceite do
custo.

---

## O essencial

**O problema** é uma manchete anual: *"site do concurso fora do ar no último dia de inscrição"*.
Quando acontece, chega ao Judiciário — há registro de prazo estendido por decisão judicial.

**Por que exige nuvem:** um certame tem **cinco picos**, não um. E os três maiores — cartão de
confirmação, gabarito e resultado — são acessados pela **base inteira ao mesmo tempo**. A carga é
40× a média, e a data está publicada em edital com meses de antecedência.

**Por que agora:** a **Lei nº 14.965/2024** autoriza prova a distância e endurece as regras da
banca. Vigência em **1º/01/2028**, com aplicação antecipada permitida. Todo o mercado precisa se
replataformar.

**Como ganha dinheiro:** assinatura anual pequena + **valor por inscrição processada**. Alinha com
o modelo da própria banca e concentra a receita nos mesmos dias em que o custo se concentra.
**Vender para a banca evita licitação** — o órgão licita a banca; a banca contrata fornecedor como
empresa privada.

**A variável que decide tudo:** o preço por inscrição. **Abaixo de R$ 2,15 o modelo não fecha.**
Está marcada como não verificada no plano, de propósito.

**O que não escondemos:** o TAM da categoria é modesto (~R$ 90 mi), e **o mercado já está
ocupado** — Gestor Editais, ProSeleta, Fábrica de Provas, ADSR. O argumento não é "ninguém faz
isso"; é *"todos fazem a inscrição, quase ninguém sobrevive ao dia do pico e ao recurso em
massa"*.

---

## O trabalho de multicloud

A defesa de arquitetura está em [`../fiap-multicloud/`](../fiap-multicloud/) e conclui **não** —
porque multicloud cobre 1 das 7 causas reais de indisponibilidade deste produto.
