# 6. Referências

*Cada referência está mapeada ao dado exato que sustenta. O critério de avaliação pede
"referências de institutos de mercado (Forrester ou outros) com dados e pesquisas". Esta é a
lista, e nada é citado na apresentação sem constar aqui.*

> **Regra desta página, desde 22/09/2026:** todo fato abaixo foi conferido na fonte aberta, com o
> trecho literal. Resumo de buscador não conta como fonte. Uma rodada anterior que não seguiu essa
> regra gravou cinco erros nesta página, registrados em §6.9.

---

## 6.1 Institutos de mercado

| Fonte | Dado usado | Onde aparece |
|---|---|---|
| **HashiCorp / Forrester Consulting**, *State of Cloud Strategy Survey 2024* (jun/2024, ~1.200 profissionais e decisores, empresas de 1.000+ funcionários, global) | **78%** gastam ou planejam gastar em multicloud | Slide 4 · §3.6 |
| idem | **8%** são de alta maturidade em nuvem | Slide 4 · §3.6 · §3.8 (C3) |
| idem | **64%** não têm pessoal com expertise suficiente | Slide 4 · §3.8 (C3) |
| idem | **91%** relatam desperdício; falta de competências é o maior contribuinte | §3.6 |
| idem | **79%** das organizações de alta maturidade dizem que multicloud melhorou a segurança | §3.6, **citado contra a nossa própria tese**, de propósito |
| **Flexera**, *State of the Cloud Report 2025* | **27%** do gasto em nuvem é desperdiçado; **84%** apontam custo como maior desafio | §3.6 · §3.8 (C7) |
| **Gartner** | **Mais de 92%** das grandes empresas operam em multicloud | Slide 4 · §3.6 |
| **Forrester**, *Predictions 2026: Cloud Outages, Private AI On Private Clouds, And The Rise Of The Neoclouds* (blog, Lee Sustar, 17/11/2025) | "At least two major multiday outages in 2026", atribuídos ao desvio de investimento dos hiperescaladores para data centers de IA | §3.6 · §3.10, **citado contra a nossa tese** |

> **Por que citamos o dado que joga contra nós.** Se 78% do mercado vai para multicloud e nós não
> vamos, a defesa tem que explicar por que somos o caso diferente. Omitir o número enfraqueceria
> mais do que citá-lo.

> **A previsão da Forrester é o dado mais forte contra esta defesa**, e está no documento por isso.
> Ela não afeta o argumento de eficácia, mas afetaria a conta do custo por hora: se as quedas
> ficam mais longas, a segunda nuvem compra mais horas **quando o M1 não tem onde restaurar**. Foi
> por isso que o M1 ganhou um destino de restauração a frio no Google Cloud (§3.10): com ele, a
> segunda nuvem sempre ligada compra sempre as mesmas 4 horas, qualquer que seja a duração da queda.
>
> **O que a Forrester não diz:** em nenhum dos textos abertos há recomendação de "desenhar para
> 72 horas". O relatório *The State Of Cloud Resilience, 2026* (Brent Ellis, 21/01/2026) existe,
> mas é fechado; seu resumo público só diz que resiliência virou prioridade e cita failover,
> backup, SRE, observabilidade e estratégias híbridas. **Não citamos nada dele além disso.**

**Links:**
- [HashiCorp State of Cloud Strategy Survey 2024](https://www.hashicorp.com/en/state-of-the-cloud)
- [Análise da HashiCorp sobre a edição 2024](https://www.hashicorp.com/en/blog/hashicorp-state-of-cloud-strategy-survey-2024-cloud-maturity)
- [Forrester, Predictions 2026: Cloud Outages...](https://www.forrester.com/blogs/predictions-2026-cloud-outages-private-ai-on-private-clouds-and-the-rise-of-the-neoclouds/)
- [Forrester, The State Of Cloud Resilience, 2026 (resumo público)](https://www.forrester.com/report/the-state-of-cloud-resilience-2025/RES190878)

---

## 6.2 Legislação e normas

| Norma | O que estabelece, no texto | Onde sustenta |
|---|---|---|
| **Lei nº 14.965, de 09/09/2024**, normas gerais de concursos públicos | **Art. 1º:** "estabelece normas gerais sobre concurso público para provimento de cargos e empregos públicos". Exclui magistratura, Ministério Público, Defensoria, carreiras militares e estatais não dependentes (§3º) | §1.3 (por que agora) |
| idem, **art. 13, §2º** | "Alternativamente à observância das normas desta Lei, os Estados, o Distrito Federal e os Municípios podem optar por editar normas próprias, observados os princípios constitucionais da administração pública" | §1.3 |
| idem, **art. 13, caput** | Vigência em **1º de janeiro do quarto ano após a publicação**, ou seja 1º/01/2028, "podendo sua aplicação ser antecipada pelo ato que autorizar a abertura de cada concurso público". Não se aplica a concursos autorizados antes disso (§1º) | §1.3 |
| idem, **art. 7º** | Conteúdo mínimo obrigatório do edital, incluindo "a forma e o prazo para interposição de recursos" (inciso XIV). Arts. 4º a 6º tratam da comissão e da instituição organizadora | Pitch, slide 4 |
| idem, **art. 8º** | O concurso "**poderá** ser realizado total ou parcialmente à distância, de forma online ou por plataforma eletrônica com acesso individual seguro e em ambiente controlado". Parágrafo único: "depende de regulamentação", com consulta pública prévia | **Gatilho 2** de §3.13 |
| **Lei nº 9.504/1997, art. 73, V**, Lei das Eleições | Veda **nomeação** de aprovados nos 3 meses anteriores ao pleito até a posse. **Não veda realizar o certame** | Correção do risco de ciclo eleitoral no plano de negócios |
| **LGPD, Lei nº 13.709/2018** | Base para o tratamento de laudo médico e autodeclaração racial; papel duplo controlador e operador | §3.11 · banco de perguntas |

> **Como ler a Lei 14.965/2024, sem exagero para nenhum lado.**
> - Ela vale **para os concursos de todos os entes**, não só da União. Estados, DF e municípios
>   podem, em alternativa, editar normas próprias.
> - Ela **não obriga ninguém a trocar de sistema nem a fazer prova on-line.** A prova a distância
>   é faculdade ("poderá") e depende de regulamentação.
> - O que ela faz é **mudar as regras do concurso** a partir de 2028. Sistemas antigos terão de se
>   adequar a regras novas, e a prova on-line passa a ter base legal expressa.
>
> Por isso as frases "todo o mercado precisa se replataformar" (versão original) e "obrigatória só
> para concursos federais" (uma correção errada de 22/09/2026) estão **ambas erradas** e foram
> substituídas em todos os documentos.

**Links:**
- [Lei nº 14.965/2024, texto na Câmara dos Deputados](https://www2.camara.leg.br/legin/fed/lei/2024/lei-14965-9-setembro-2024-796212-publicacaooriginal-172961-pl.html) · [Planalto](https://www.planalto.gov.br/ccivil_03/_ato2023-2026/2024/lei/l14965.htm) · [Normas Legais](https://www.normaslegais.com.br/legislacao/lei-14965-2024.htm)
- [Concurso público em ano eleitoral, regras de nomeação](https://www.tecconcursos.com.br/blog/noticias/concurso-publico-em-ano-eleitoral/)

---

## 6.3 Entendimento de controle sobre a taxa de inscrição

| Fonte | O que sustenta |
|---|---|
| **TCE-MG**, consulta sobre taxas de inscrição em concursos | Taxa de inscrição é **receita pública**; a remuneração da organizadora pode ser fixa ou variável conforme o número de inscritos, desde que prevista em edital e contrato |
| **MPRJ**, *A contratação de empresas para a realização de concursos públicos* | Existência do modelo **"custo zero"**, em que a organizadora assume os custos e fica com a arrecadação das taxas |

> **Sustenta a decisão de produto de não processar o pagamento.** Orquestramos a cobrança no
> provedor da própria banca. Isso deixou de ser simplificação de engenharia e virou requisito.

**Links:**
- [TCE-MG responde consulta sobre taxas de inscrição](https://www.tce.mg.gov.br/TCE-responde-consulta-sobre-taxas-de-inscricao-em-concursos-publicos.html/Noticia/1111620521)
- [MPRJ, contratação de empresas para realização de concursos](https://www.mprj.mp.br/documents/20184/6380157/rita_tourinho.pdf/)

---

## 6.4 Falhas reais em certames, e o que aconteceu depois

*Os três casos documentados de 2026 que encontramos. São a base de §3.3 ("nenhum foi queda do
provedor") e da premissa de remédio.*

### Alece 2026 (Assembleia Legislativa do Ceará, banca Idecan)

| Fase | O que aconteceu | Remédio | Fonte |
|---|---|---|---|
| **Isenção**, mai/2026 | O MPCE "recebeu várias manifestações narrando que candidatos não conseguiram efetivar o seu pedido de isenção e inscrição em razão da instabilidade no site", nos dias 21 e 22 de maio | **Negado.** Em ação civil pública do MPCE, o juiz João Everardo Matos Biermann, da 13ª Vara da Fazenda Pública de Fortaleza, decidiu "indeferir, por ora, a reabertura geral do período de isenção aos demais candidatos, inclusive, com fundamento nas alegadas instabilidades do sistema eletrônico". A reanálise de isenções que ele determinou foi por **outro motivo**, o critério de isenção para egressos de universidade pública, e a data da prova foi mantida | Diário do Nordeste, 11/08/2026 |
| **Recurso**, set/2026 | "Dificuldades de acesso a dados no sistema, além de falhas na divulgação preliminar da prova" | **Concedido, por via administrativa.** Sob ofício da Assembleia, a Idecan reabriu o prazo de recurso em 16 e 17 de setembro | O Povo, 15/09/2026 |

> **O que o caso prova, e o que não prova.** Prova que a falha de sistema em fase de prazo tem
> remédio praticado (o recurso foi reaberto), **e que esse remédio não é garantido** (a isenção
> não foi). Não prova "prorrogação de prazo de inscrição por decisão judicial": nesse caso, a
> via judicial negou justamente o pedido fundado na instabilidade. A causa técnica das falhas
> não foi divulgada.

### AgSUS 2026 (Agência Brasileira de Apoio à Gestão do SUS, banca Cebraspe)

| O que aconteceu | Remédio | Fonte |
|---|---|---|
| Prova objetiva **on-line e remota**, feita em casa com "câmera e microfone" e "inteligência artificial e fiscais remotos para monitoramento". A retificação oficial registra "mensagens de erro do sistema, falha na conexão e restrição de acesso ao ambiente da prova, provocadas por mecanismos de controle de acesso do sistema de segurança". Há relatos de candidatos que perderam 16 de 40 questões | **A aplicação de 22/02/2026 foi anulada** e remarcada para 15/03/2026. Relatos de falhas também em 08/03/2026 | Direção Concursos; Correio Braziliense, mar/2026 |

> **Por que este caso muda o gatilho 2.** A prova on-line não é cenário de 2028: já é aplicada em
> 2026. E a falha durante a prova **teve remédio**, mas um remédio de outra ordem de grandeza:
> anular e reaplicar a prova inteira, não reabrir um prazo. E a causa foi da própria banca.

### FGV, fevereiro de 2026

| O que aconteceu | Remédio | Fonte |
|---|---|---|
| Site de concursos fora do ar desde a sexta-feira anterior a 23/02/2026, após "um ataque hacker"; afetou o CNU (manifestação de interesse e resultados) e inscrições de outros concursos. O tipo de ataque não foi divulgado | Link alternativo temporário. Não há registro de prorrogação formal na fonte | Acheconcursos, 23/02/2026 |

> **Por que este caso muda a tabela de §3.3.** Ataque é uma causa de "site do concurso fora do ar"
> que a tabela original não tinha. Virou a oitava linha, ligada à dívida D9.

**Links:**
- [Diário do Nordeste: Justiça mantém prova do concurso da Alece, mas impõe medidas ao Idecan](https://diariodonordeste.verdesmares.com.br/papo-carreira/justica-mantem-prova-do-concurso-da-alece-para-domingo-16-mas-impoe-medidas-ao-idecan-1.3783150)
- [O Povo: Concurso Alece tem prazo de recursos reaberto pela Idecan](https://www.opovo.com.br/noticias/economia/2026/09/15/concurso-alece-tem-prazo-de-recursos-reaberto-pela-idecan.html)
- [Direção Concursos: Concurso AgSUS anula prova objetiva on-line](https://www.direcaoconcursos.com.br/noticias/concurso-agsus-provas-anuladas)
- [Correio Braziliense: AgSUS, candidatos relatam falhas durante prova on-line](https://www.correiobraziliense.com.br/euestudante/concursos/2026/03/7372414-agsus-candidatos-relatam-falhas-em-sistema-durante-prova-on-line.html)
- [Acheconcursos: site da FGV segue fora do ar](https://www.acheconcursos.com.br/noticias/site-da-fgv-segue-fora-do-ar-hoje-23-02-candidatos-reclamam-88152)

### Fonte genérica anterior, mantida como contexto

| Fonte | O que sustenta |
|---|---|
| Análise jurídica sobre indisponibilidade do sistema de inscrição e perda de prazo | Que o candidato prejudicado pode buscar mandado de segurança e liminar. **Não cita caso concreto**, e por isso não é usada como prova de que o remédio acontece |

**Link:** [O que fazer quando o sistema de inscrição fica indisponível e o candidato perde o prazo](https://fabioportela.com/concursopublico/blog/o-que-fazer-quando-o-sistema-de-inscricao-no-concurso-publico-fica-indisponivel-e-o-concurseiro-perde-o-prazo-de-inscricao/)

---

## 6.5 Concorrência verificada

| Concorrente | Relevância |
|---|---|
| **Gestor Editais** (Grupo ACT) | **Cobra sobre o total de inscrições do certame**, o que valida o modelo de preço |
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
| DNS, WAF, fila de espera | **Cloudflare** (DNS, WAF, Waiting Room) | Cloudflare, **fora da nuvem primária** |
| CDN e publicação estática | **Amazon CloudFront + S3** | AWS |
| Orquestração de contêiner | **Amazon EKS + Karpenter** | AWS |
| Banco de dados | **Amazon Aurora PostgreSQL Serverless v2** | AWS |
| Fila e execução assíncrona | **Amazon SQS + AWS Lambda** | AWS |
| Infraestrutura como código | **Terraform** | HashiCorp |
| CI/CD | **GitHub Actions** | GitHub, externo |
| Página de status | **Statuspage** ou equivalente | **Fora da nuvem primária** |
| Backup off-site | Armazenamento de objetos no **Google Cloud**, região `southamerica-east1`, com trava de imutabilidade | Google, **fora da nuvem primária** |
| Restauração a frio | **GKE e Cloud SQL para PostgreSQL**, criados só no ensaio ou na emergência. Alternativa equivalente: **Azure**, região Brazil South | Google, **fora da nuvem primária** |
| Ensaio de carga | **k6** ou **Gatling** | Grafana Labs / Gatling Corp |

**Regiões do segundo provedor, conferidas na fonte em 23/09/2026:**

| Provedor | O que a fonte diz |
|---|---|
| **Google Cloud** | Região **`southamerica-east1`, em São Paulo** |
| **Microsoft Azure** | Região **Brazil South** (`brazilsouth`), localização física "Sao Paulo State", **com zonas de disponibilidade**. A Brazil Southeast, no Rio, tem acesso restrito |

- [Google Cloud: locations](https://cloud.google.com/about/locations)
- [Microsoft Learn: List of Azure regions](https://learn.microsoft.com/en-us/azure/reliability/regions-list)

---

## 6.7 Reverificação da véspera, feita em 22/09/2026

| # | Item | Resultado, conferido na fonte |
|---|---|---|
| 1 | A AWS anunciou uma segunda região no Brasil? | **Não.** São Paulo continua a única; Rio é Local Zone. Gatilho 5 não disparou |
| 2 | Saiu edição 2025 ou 2026 do *State of Cloud Strategy Survey*? | **Não.** A de 2024 segue a mais recente. Entrou, em compensação, a previsão da Forrester para 2026 (§6.1) |
| 3 | Houve caso público recente de queda em certame? | **Sim, três, todos de 2026:** Alece, AgSUS e FGV (§6.4). Nenhum foi queda do provedor de nuvem |
| 4 | Algum edital invocou a aplicação antecipada da Lei 14.965/2024? | Nenhum encontrado. Mas a prova on-line **já é praticada** (AgSUS), e a leitura do texto da lei corrigiu o escopo (§6.2) |
| 5 | Existe decisão judicial primária sobre indisponibilidade? | **Existe, e é desfavorável:** no caso Alece, o juiz negou a reabertura da isenção fundada em instabilidade. O remédio que aconteceu foi administrativo. A premissa de §3.3 foi reescrita de acordo |
| 6 | Existe registro de queda total do sa-east-1? | **Não encontrado.** Ver abaixo os dois incidentes AWS recentes que existem, e por que nenhum serve de fonte para a frequência de §3.10 |

### Incidentes AWS recentes, e o que eles provam de fato

| Incidente | O que a fonte diz | O que isso prova para nós |
|---|---|---|
| **us-east-1, 20/10/2025** | Problema no "processo de resolução de DNS" do DynamoDB, "cerca de 15 horas" de instabilidade; "iFood, Mercado Livre e PicPay no Brasil" entre os afetados | Que uma falha do provedor, **fora do Brasil**, afeta empresas brasileiras. **A fonte não diz por qual mecanismo**: carga hospedada em us-east-1, dependência de serviços globais ou de fornecedores SaaS. Não afirmamos nenhum deles |
| **me-central-1 (Emirados), 01/03/2026** | Data center atingido por "objects that struck the data center, creating sparks and fire"; a AWS confirmou depois "drone strikes". As três zonas da região foram afetadas: duas por perda de energia e a terceira com erros de API e falha ao iniciar instâncias | Que uma **perda de múltiplas zonas numa região** acontece de fato. Mas a causa foi um ataque militar em zona de guerra, sem relação com o Brasil e **sem evidência de impacto em empresas brasileiras** |

> **Nenhum dos dois é fonte para a frequência de "1 queda regional do sa-east-1 a cada 4 anos"**,
> que continua sendo estimativa nossa (§6.8). O primeiro reforça, por outro caminho, o motivo de
> tirar DNS, WAF, status e backup da AWS inteira, e não só da região.

**Links:**
- [Unicast: AWS us-east-1, o apagão de 20 de outubro](https://unicast.com.br/posts/aws-us-east1-o-apagao-que-deixou-o-brasil-e-o-mundo-na-emergencia/)
- [Data Center Knowledge: AWS Middle East outage after data center hit by unidentified objects](https://www.datacenterknowledge.com/outages/aws-middle-east-outage-after-data-center-hit-by-unidentified-objects)

---

## 6.8 O que este trabalho afirma sem fonte externa, e por quê

*Declarado aqui para que ninguém precise descobrir na arguição.*

| Afirmação | Onde | Status |
|---|---|---|
| **A tabela das oito falhas** de §3.3 | Slide 3, o argumento central | **Taxonomia nossa.** Não é estatística de mercado e não é apresentada como tal. Cada uma das sete falhas internas corresponde a uma dívida técnica documentada em §2.3. O que sustenta o argumento é a **origem** das falhas, não a contagem. Os três casos reais de 2026 (§6.4) são consistentes com ela |
| **Frequência de 1 queda regional a cada 4 anos** | §3.10, base do custo por hora | **Estimativa nossa.** O argumento é robusto à frequência: dobrando-a, o custo por hora cai pela metade e a conclusão não muda. A previsão da Forrester (§6.1) empurra a estimativa para cima |
| **RTO de 6 a 12 h sob M1 e de 4 h sob M2** | §3.9 e §3.10 | **Estimativa de engenharia**, não medida em exercício real. O ensaio trimestral de restauração no Google Cloud é o que a confirma ou derruba; acima de 8 horas, dispara o gatilho 3 |
| **Exigência de dado em território nacional** nos contratos públicos | §3.4 e §3.10 | **Não verificada em contrato específico.** Por isso a escolha do Google não depende só dela: ele também sobrevive a uma falha da AWS inteira e a uma conta AWS comprometida |
| **Custo da restauração a frio, R$ 80 mil/ano, e da alternativa na AWS, ~R$ 35 mil** | §3.10 e §3.11 | **Estimativa nossa**, feita com o mesmo custo por engenheiro da conta do M2 (R$ 330 mil/ano): 3 meses de portabilidade diluídos em 3 anos, 10% de um engenheiro e até R$ 20 mil de máquinas de ensaio. O destino foi **decidido em 23/09/2026**: Google Cloud em São Paulo |
| **Preço por inscrição de R$ 3,00** | Plano de negócios | **Não verificado**, tratado como variável com sensibilidade. Ver [`../fiap-startup-one/03-plano-de-negocios.md` §4](../fiap-startup-one/03-plano-de-negocios.md) |

> **Por que declarar isto.** Um trabalho que apresenta taxonomia própria como se fosse dado de
> mercado perde a discussão no momento em que alguém pede a fonte. Declarar antes transforma a
> fragilidade em método.

---

## 6.9 Erros registrados, para não se repetirem

*Uma rodada de verificação em 22/09/2026 usou resumos do buscador como se fossem fontes e gravou
os erros abaixo. Todos foram corrigidos no mesmo dia, depois da leitura das fontes.*

| O que foi gravado | O que a fonte diz |
|---|---|
| Lei 14.965/2024 "obrigatória só para concursos federais" | Vale para todos os entes; estados, DF e municípios podem, em alternativa, editar normas próprias (art. 13, §2º) |
| No caso Alece, o juiz "determinou a reanálise das isenções negadas por instabilidade" | O juiz **indeferiu** a reabertura fundada em instabilidade. A reanálise foi pelo critério de universidade pública |
| Erros "HTTP 503" no caso Alece | Não aparecem na matéria consultada |
| A Forrester "recomenda desenhar para 72 horas" | Não está em nenhum texto aberto da Forrester |
| Os incidentes AWS afetaram o Brasil "por dependência de plano de controle" | A fonte não descreve o mecanismo; o incidente dos Emirados não tem impacto documentado no Brasil |
| *(23/09/2026)* O Google Cloud tem "uma única região na América do Sul" | Veio do resumo automático da página de regiões; uma segunda leitura da mesma página não confirmou. Ficou só o que a fonte mostrou: `southamerica-east1` em São Paulo |
