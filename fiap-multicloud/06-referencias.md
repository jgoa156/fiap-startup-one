# 6. Referências

*Cada referência está mapeada ao dado exato que sustenta. O critério de avaliação pede
"referências de institutos de mercado (Forrester ou outros) com dados e pesquisas" — esta é a
lista, e nada é citado na apresentação sem constar aqui.*

---

## 6.1 Institutos de mercado

| Fonte | Dado usado | Onde aparece |
|---|---|---|
| **HashiCorp / Forrester Consulting** — *State of Cloud Strategy Survey 2024* (jun/2024, ~1.200 profissionais e decisores, empresas de 1.000+ funcionários, global) | **78%** gastam ou planejam gastar em multicloud | Slide 4 · §3.6 |
| idem | **8%** são de alta maturidade em nuvem | Slide 4 · §3.6 · §3.8 (C3) |
| idem | **64%** não têm pessoal com expertise suficiente | Slide 4 · §3.8 (C3) |
| idem | **91%** relatam desperdício; falta de competências é o maior contribuinte | §3.6 |
| idem | **79%** das organizações de alta maturidade dizem que multicloud melhorou a segurança | §3.6 — **citado contra a nossa própria tese**, de propósito |
| **Flexera** — *State of the Cloud Report 2025* | **27%** do gasto em nuvem é desperdiçado; **84%** apontam custo como maior desafio | §3.6 · §3.8 (C7) |
| **Gartner** | **Mais de 92%** das grandes empresas operam em multicloud | Slide 4 · §3.6 |

> **Por que citamos o dado que joga contra nós.** Se 78% do mercado vai para multicloud e nós não
> vamos, a defesa tem que explicar por que somos o caso diferente. Omitir o número enfraqueceria
> mais do que citá-lo.

**Links:**
- [HashiCorp State of Cloud Strategy Survey 2024](https://www.hashicorp.com/en/state-of-the-cloud)
- [Análise da HashiCorp sobre a edição 2024](https://www.hashicorp.com/en/blog/hashicorp-state-of-cloud-strategy-survey-2024-cloud-maturity)

---

## 6.2 Legislação e normas

| Norma | O que estabelece | Onde sustenta |
|---|---|---|
| **Lei nº 14.965, de 09/09/2024** — Lei Geral dos Concursos Públicos | Autoriza prova total ou parcialmente a distância; endurece regras da banca. **Vigência em 1º/01/2028**, com aplicação antecipada se o edital previr | §1.3 (por que agora) · **gatilho 2** do slide 8 — o cenário que inverteria a decisão |
| **Lei nº 9.504/1997, art. 73, V** — Lei das Eleições | Veda **nomeação** de aprovados nos 3 meses anteriores ao pleito até a posse. **Não veda realizar o certame** | Correção do risco de ciclo eleitoral no plano de negócios |
| **LGPD — Lei nº 13.709/2018** | Base para o tratamento de laudo médico e autodeclaração racial; papel duplo controlador/operador | §3.11 · banco de perguntas |

**Links:**
- [Lei nº 14.965/2024 — texto](https://www.jusbrasil.com.br/legislacao/2719968211/lei-14965-24) · [análise das mudanças](https://www.jusbrasil.com.br/artigos/principais-mudancas-nos-concursos-publicos-com-a-lei-14965-2024/2754141665)
- [Concurso público em ano eleitoral — regras de nomeação](https://www.tecconcursos.com.br/blog/noticias/concurso-publico-em-ano-eleitoral/)

---

## 6.3 Entendimento de controle — taxa de inscrição

| Fonte | O que sustenta |
|---|---|
| **TCE-MG** — consulta sobre taxas de inscrição em concursos | Taxa de inscrição é **receita pública**; a remuneração da organizadora pode ser fixa ou variável conforme o número de inscritos, desde que prevista em edital e contrato |
| **MPRJ** — *A contratação de empresas para a realização de concursos públicos* | Existência do modelo **"custo zero"**, em que a organizadora assume os custos e fica com a arrecadação das taxas |

> **Sustenta a decisão de produto de não processar o pagamento.** Orquestramos a cobrança no
> provedor da própria banca. Isso deixou de ser simplificação de engenharia e virou requisito.

**Links:**
- [TCE-MG responde consulta sobre taxas de inscrição](https://www.tce.mg.gov.br/TCE-responde-consulta-sobre-taxas-de-inscricao-em-concursos-publicos.html/Noticia/1111620521)
- [MPRJ — contratação de empresas para realização de concursos](https://www.mprj.mp.br/documents/20184/6380157/rita_tourinho.pdf/)

---

## 6.4 Evidência da falha — e da sua reversibilidade

| Fonte | O que sustenta |
|---|---|
| Análise jurídica sobre **indisponibilidade do sistema de inscrição e perda de prazo** | Que a indisponibilidade no fim do prazo é fenômeno recorrente **e que o remédio praticado é a prorrogação, inclusive por decisão judicial** |

> **É a referência mais importante da defesa**, porque sustenta a premissa de §3.3: a falha aqui é
> **reversível**, e isso muda quanto a proteção pode custar. Sem ela, o argumento vira opinião.

**Link:** [O que fazer quando o sistema de inscrição fica indisponível e o candidato perde o prazo](https://fabioportela.com/concursopublico/blog/o-que-fazer-quando-o-sistema-de-inscricao-no-concurso-publico-fica-indisponivel-e-o-concurseiro-perde-o-prazo-de-inscricao/)

---

## 6.5 Concorrência verificada

| Concorrente | Relevância |
|---|---|
| **Gestor Editais** (Grupo ACT) | **Cobra sobre o total de inscrições do certame** — valida o modelo de preço |
| **ProSeleta** | Gestão de todas as fases, da inscrição ao resultado |
| **Fábrica de Provas** | Avaliação, inscrição, aplicação e correção; menciona conformidade com LGPD |
| **ADSR** | ERP para concursos, com processamento de isenções |
| **MH Informática** | Correção eletrônica de cartão-resposta |
| **Instituto Avalia** | Prova online e gestão pública |

**Links:** [Gestor Editais](https://gestoreditais.com.br/) · [ProSeleta](https://www.proseleta.com.br/) · [Fábrica de Provas](https://www.fabricadeprovas.com.br/) · [ADSR](https://www.adsr.com.br/sistema-concursos/) · [MH Informática](https://mhinformatica.inf.br/concursos-publicos) · [Instituto Avalia](https://avalia.org.br/atuacao)

---

## 6.6 Soluções de mercado citadas na arquitetura

*O critério de avaliação pede "citando exemplos de soluções de mercado". Cada componente do
desenho nomeia produto e fornecedor.*

| Camada | Produto | Fornecedor |
|---|---|---|
| DNS, WAF, fila de espera | **Cloudflare** (DNS, WAF, Waiting Room) | Cloudflare — **fora da nuvem primária** |
| CDN e publicação estática | **Amazon CloudFront + S3** | AWS |
| Orquestração de contêiner | **Amazon EKS + Karpenter** | AWS |
| Banco de dados | **Amazon Aurora PostgreSQL Serverless v2** | AWS |
| Fila e execução assíncrona | **Amazon SQS + AWS Lambda** | AWS |
| Infraestrutura como código | **Terraform** | HashiCorp |
| CI/CD | **GitHub Actions** | GitHub — externo |
| Página de status | **Statuspage** ou equivalente | **Fora da nuvem primária** |
| Backup off-site | Armazenamento de objetos em **segundo provedor**, com trava de imutabilidade | **Fora da nuvem primária** |
| Ensaio de carga | **k6** ou **Gatling** | Grafana Labs / Gatling Corp |

---

## 6.7 A reverificar na véspera

| # | Item | Por que importa |
|---|---|---|
| 1 | **A AWS anunciou uma segunda região no Brasil?** *(verificado em set/2026: não — São Paulo é a única; Rio é Local Zone)* | É o **gatilho 5**. Se mudou, o slide 8 muda |
| 2 | Saiu edição 2025 ou 2026 do *State of Cloud Strategy Survey*? *(verificado: 2024 continua sendo a mais recente amplamente divulgada)* | Dado mais novo fortalece o slide 4 |
| 3 | Houve caso público recente de queda de sistema de inscrição? | Aumenta muito a força do slide 3 se for recente |
| 4 | Algum edital já invocou a aplicação antecipada da Lei 14.965/2024? | Anteciparia o **gatilho 2** |

> **Itens 1 e 2 foram verificados em 2 de setembro de 2026 e seguem válidos.** Os itens 3 e 4 são
> oportunidade, não risco: se houver novidade, ela reforça a defesa.

## 6.8 Reverificação de 29 de outubro de 2026

*Feita depois da apresentação. Nenhum resultado muda a tese; o item 3 traz dois casos novos.*

| # | Resultado | Efeito na defesa |
|---|---|---|
| 1 | **Não.** São Paulo (sa-east-1) segue a única região AWS no Brasil. A expansão anunciada é de borda (edge location em Fortaleza) e a nova região sul-americana planejada é a do **Chile**, não do Brasil | Gatilho 5 **não disparou** |
| 2 | **Não saiu edição nova feita pela Forrester.** A página do *State of Cloud Strategy Survey* continua sendo a de 2024 (Forrester Consulting, ~1.200 respondentes, 79% com multicloud ou planejando). A HashiCorp publicou o *2025 Cloud Complexity Report* (1.100+ líderes de TI; 52% citam complexidade como desafio principal), mas **não nomeia o instituto** e não traz número de adoção multicloud | Slide 4 segue com o dado de 2024. O relatório de 2025 serve no máximo como apoio ao argumento de complexidade, **sem atribuir à Forrester** |
| 3 | **Sim, dois casos de 2026** — ver abaixo | Reforça o slide 3, com ressalva sobre a causa |
| 4 | **Não encontrado.** Nenhum edital localizado invoca a aplicação antecipada da Lei 14.965/2024. A prova a distância ainda depende de regulamentação do Executivo. *Armadilha:* edital da Bahia (EPPGG 2026) cita a "Lei estadual nº 14.965/**2025**" — é lei estadual de número parecido, **não** a lei federal | Gatilho 2 **não foi antecipado** |

**Os dois casos do item 3:**

- **Enem 2026 — 27/10/2026.** Na liberação do **cartão de confirmação** (local de prova), a Página do
  Participante do Inep ficou lenta e inacessível. Congestionamento é a explicação provável, mas
  **a causa não foi confirmada oficialmente**. É exatamente o **pico 3** da Lisura — a base inteira
  acessando numa data publicada com antecedência. *Ressalva:* é o Inep operando sistema próprio,
  não uma banca.
- **FGV — 20 a 23/02/2026.** Site e sistemas da banca instáveis por cerca de 3 dias; inscrições do
  Exame Nacional dos Cartórios não puderam ser concluídas e prazos de recurso (CPRM, Nova Iguaçu)
  venceram com o site fora do ar. **Causa não confirmada** — a FGV negou oficialmente ataque, e uma
  coluna de O Globo falou em sistemas derrubados. Prorrogação apenas cogitada, não confirmada.

> **Como usar com honestidade:** os dois casos provam que **a queda acontece e é pública** — não
> provam qual das 7 causas a provocou. Nenhum dos dois foi atribuído a queda regional de provedor,
> o que é coerente com a tese, mas **não a demonstra**. Não apresentar como evidência de causa.

Fontes: [Lightsail em novas regiões (AWS, jun/2026)](https://aws.amazon.com/about-aws/whats-new/2026/06/amazon-lightsail-aws-regions/) ·
[Edge location em Fortaleza (DCD)](https://www.datacenterdynamics.com/en/news/aws-to-launch-edge-location-in-fortaleza-brazil/) ·
[State of Cloud Strategy Survey 2024](https://www.hashicorp.com/en/state-of-the-cloud) ·
[2025 Cloud Complexity Report](https://www.hashicorp.com/en/cloud-complexity-report) ·
[Inep fora do ar no local de prova do Enem 2026 (TechTudo)](https://www.techtudo.com.br/noticias/2026/10/inep-fora-do-ar-site-congestiona-apos-divulgacao-do-local-de-prova-do-enem-2026-edsoftwares.ghtml) ·
[FGV estabiliza após instabilidade (Folha Dirigida/QConcursos)](https://folha.qconcursos.com/n/fgv-cai-e-candidatos-de-concursos-ficam-sem-acesso) ·
[Lei 14.965/2024 (Planalto)](https://www.planalto.gov.br/ccivil_03/_ato2023-2026/2024/lei/l14965.htm) ·
[Nova lei dos concursos (Estado de Minas, jul/2026)](https://www.em.com.br/emfoco/2026/07/08/nova-lei-dos-concursos-tem-data-para-comecar-e-pode-liberar-provas-online/)
