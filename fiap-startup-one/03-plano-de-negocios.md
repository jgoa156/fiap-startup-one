# 3. Plano de negócios

*Plataforma B2B de inscrição e gestão de processos seletivos, vendida para bancas organizadoras.
Produto e fluxos em [02-produto-features-e-fluxos.md](02-produto-features-e-fluxos.md).*

> **Aviso de honestidade metodológica.** Três números deste plano **não foram verificados** e estão
> marcados como `⚠ NV`. O principal deles — o preço por inscrição — decide se este é um negócio de
> R$ 16 milhões ou de R$ 43 milhões. Ele é tratado como **variável**, com faixa e sensibilidade, e
> não como premissa. Assumir premissa não verificada é o erro mais caro que um plano pode conter.

---

## 1. Resumo

| | |
|---|---|
| **O que é** | Plataforma que conduz um processo seletivo de ponta a ponta — do edital à homologação do resultado |
| **Cliente** | Banca organizadora de porte médio: 5 a 50 certames por ano |
| **Por que agora** | **Lei 14.965/2024** — Lei Geral dos Concursos Públicos, sancionada em 09/09/2024. Autoriza prova total ou parcialmente a distância e endurece as regras de escolha e conduta da banca. **Vigência em 1º/01/2028, com aplicação antecipada se o edital previr.** Todo o mercado precisa se replataformar até lá, e quem adotar antes precisa de sistema que já atenda |
| **Como cobra** | Assinatura anual pequena + **valor por inscrição processada** (o segundo domina) |
| **Vantagem comercial** | **Vender para a banca evita licitação** — o órgão licita a banca; a banca contrata fornecedor como empresa privada |
| **Fosso** | Sistema de registro do certame + conta única do candidato, que atravessa bancas |
| **Meta** | R$ 27 mi de ARR no ano 5, breakeven no 4º trimestre do ano 4 |

---

## 2. O mercado

### O que dá para afirmar

Existem, por ano no Brasil, milhares de certames: concurso municipal, estadual e federal, vestibular
de instituição privada, residência médica, processo seletivo simplificado, prova de conselho de
classe. Cada um cobra taxa por candidato, e a arrecadação da taxa é a receita da banca.

### O que não dá para afirmar sem verificar

| Número | Faixa usada | Status |
|---|---|---|
| Inscrições pagas por ano no Brasil, em todos os certames | 20 a 40 milhões | `⚠ NV` |
| Taxa média paga pelo candidato | R$ 80 | `⚠ NV` |
| Quantas bancas de porte médio existem | 80 a 200 | `⚠ NV` |

### TAM, SAM e SOM — e a conclusão desconfortável

| | Cálculo | Valor |
|---|---|---|
| **TAM** | 30 mi de inscrições × R$ 3 | **R$ 90 mi/ano** |
| **SAM** | Bancas médias e pequenas, ~40% do volume | **R$ 36 mi/ano** |
| **SOM (ano 5)** | ~72 bancas, 7,7 mi de inscrições | **R$ 27 mi de ARR** |

> **O TAM é modesto, e isso precisa ser dito antes que perguntem.** Mesmo dominando o segmento
> alvo, a categoria isolada não passa de algumas dezenas de milhões. **A empresa cresce por
> adjacência, não por participação**: certificação profissional, prova de conselho de classe,
> seleção corporativa em massa e LatAm — todos usam o mesmo motor de "muita gente, prazo curto,
> resultado contestável".

---

## 3. Modelo de receita

### A decisão que mantém o negócio simples

> **Nós orquestramos o pagamento. Não processamos o pagamento.**

O boleto e o Pix são emitidos pelo provedor **da própria banca**; nós disparamos a cobrança e
recebemos a confirmação. Três consequências, todas boas:

1. A receita fica limpa — nada de repasse inflando o faturamento.
2. Ficamos fora do fluxo financeiro, o que **elimina uma camada inteira de exigência regulatória**.
3. A banca mantém a relação bancária que já tem, o que remove uma objeção na venda.

### As duas linhas

| Linha | Valor | Papel |
|---|---|---|
| **Assinatura anual da plataforma** | R$ 48 a 84 mil `⚠ NV` | Cobre a operação de base. Previsível |
| **Valor por inscrição processada** | **R$ 1,50 a 5,00** `⚠ NV` | **É o negócio.** Alinha com o modelo da banca, que também cobra por candidato |

**E há um efeito que vale registrar:** a receita por inscrição se concentra **exatamente nos dias em
que o custo de infraestrutura se concentra**. Receita e custo pulsam juntos, o que é raro e é uma
das razões pelas quais a margem se sustenta em modelo elástico.

---

## 4. A variável que decide tudo

**V1 — preço por inscrição processada.** Contextualizando: sobre uma taxa média de R$ 80 pagos pelo
candidato, R$ 3,00 representam **3,75% da taxa**.

### Sensibilidade no ano 5 — 72 bancas, 7,7 milhões de inscrições

| V1 | Receita por inscrição | Assinaturas | **Receita total** | EBITDA | Veredito |
|---:|---:|---:|---:|---:|---|
| **R$ 1,50** | 11,6 | 4,3 | **R$ 15,9 mi** | **−R$ 5,0 mi** | ❌ **Não fecha** com esta estrutura |
| **R$ 2,15** | 16,6 | 4,3 | **R$ 20,9 mi** | **≈ 0** | ⚠️ Piso de viabilidade |
| **R$ 3,00** | 23,1 | 4,3 | **R$ 27,4 mi** | **+R$ 5,3 mi** | ✅ Cenário base |
| **R$ 5,00** | 38,5 | 4,3 | **R$ 42,8 mi** | **+R$ 19,9 mi** | ✅ Excedente reinvestido |

> **O achado:** abaixo de **R$ 2,15 por inscrição** o negócio não fecha com 72 clientes e esta
> estrutura de custo. Ou o preço sobe, ou a base de clientes precisa ser maior, ou o time precisa
> ser menor. **Descobrir o preço praticado é a tarefa número um** — antes de escrever mais uma
> linha de plano.

---

## 5. Unit economics (base: ano 3)

| Métrica | Valor | Como |
|---|---|---|
| ACV médio | **R$ 280 mil** | R$ 60 mil de assinatura + 73 mil inscrições × R$ 3 |
| Margem bruta | **76%** | Infra elástica + suporte |
| CAC | **R$ 65 mil** | Ciclo de 3 a 6 meses, **sem pregão**, venda direta a sócio ou diretor |
| **Retorno de CAC** | **3,7 meses** | CAC ÷ (ACV × margem ÷ 12) |
| Perda de cliente | **6% ao ano** | Sistema de registro do certame; ninguém troca no meio |
| LTV (5 anos, com desconto) | ~R$ 1,0 mi | Truncado em 5 anos de propósito — a 6% de perda o cálculo infinito estoura e engana |
| **LTV / CAC** | **~15x** | |

> **A economia unitária é excelente e não é o gargalo.** O gargalo é o tamanho do mercado — o
> inverso do padrão, em que a economia é apertada e o mercado é enorme.

**E há uma vantagem estrutural que merece registro:** aqui **não existe partida a
frio**. O produto entrega valor integral ao primeiro cliente, sozinho. A conta única do candidato
acrescenta valor com escala, mas não é condição para funcionar.

---

## 6. Projeções (R$ milhões)

| | Ano 1 | Ano 2 | Ano 3 | Ano 4 | Ano 5 |
|---|---:|---:|---:|---:|---:|
| Bancas clientes (fim do ano) | 4 | 13 | 28 | 48 | 72 |
| Inscrições processadas (milhões) | 0,12 | 0,69 | 2,05 | 4,30 | 7,70 |
| ACV médio (R$ mil) | 150 | 220 | 280 | 330 | 380 |
| **Receita (ARR)** | **0,6** | **2,9** | **7,8** | **15,8** | **27,4** |
| Custo dos serviços | 0,2 | 0,8 | 1,9 | 3,3 | 5,2 |
| **Margem bruta** | **67%** | **72%** | **76%** | **79%** | **81%** |
| P&D / Engenharia | 2,6 | 4,0 | 5,2 | 6,6 | 8,2 |
| Vendas e marketing | 0,9 | 2,0 | 3,0 | 4,2 | 5,6 |
| G&A / jurídico | 0,7 | 1,1 | 1,6 | 2,3 | 3,1 |
| **EBITDA** | **−3,8** | **−5,0** | **−3,9** | **−0,6** | **+5,3** |
| Margem EBITDA | — | — | — | −4% | **+19%** |
| Headcount | 14 | 26 | 38 | 52 | 68 |

**Breakeven: 4º trimestre do ano 4.** É esse marco, somado à rodada do ano 4, que constitui a
premissa do trabalho de Soluções Multicloud.

**Por que P&D é enxuto:** não há time de ciência de dados, não há
plataforma de ML, não há retreino. É engenharia de produto e de plataforma — mais barata e mais
fácil de contratar.

---

## 7. Concorrência — verificada

**O mercado existe e está ocupado.** Foi a verificação nº 2, e a resposta é sim.

| Concorrente | O que é |
|---|---|
| **Gestor Editais** (Grupo ACT) | Plataforma de concursos e processos seletivos. **Cobra sobre o total de inscrições que o certame alcançar** |
| **ProSeleta** | Gestão de todas as fases, da inscrição ao resultado final |
| **Fábrica de Provas** | Plataforma de avaliação — instituições de ensino, seletivos, certificação e concursos. Integra inscrição, aplicação e correção |
| **ADSR** | ERP para concursos: inscrição automatizada, validação, processamento de isenções |
| **MH Informática** | Correção eletrônica de cartão-resposta, exportação para gestão pública |
| **Instituto Avalia**, **Gestão de Concursos** | Operam prova online e gestão pública |
| **Sistemas institucionais** (UNICAMP, FCPC/UFC) | Construídos em casa por fundações universitárias |

### O que isso muda — e não é fatal

1. **O modelo de preço está validado.** O Gestor Editais **já cobra por inscrição**. Não estamos
   inventando um modelo; estamos entrando num mercado que já precifica assim.
2. **A tese "banca média opera com planilha" está errada.** Ela opera com **um destes**. A venda
   deixa de ser evangelização e vira substituição — ciclo mais curto, objeção diferente.
3. **A diferenciação precisa mudar de lugar.** Não é "existir": é fazer melhor o que dói. As
   apostas passam a ser **recurso agrupado por fundamento estruturado**, **recálculo versionado
   após anulação de questão**, **trilha que sobrevive a ação judicial anos depois** e
   **capacidade de pico comprovada por ensaio de carga contratual** — que é justamente o que
   nenhum concorrente anuncia.

> **Correção honesta ao pitch:** o argumento deixa de ser "ninguém faz isso" e passa a ser
> **"todos fazem a inscrição; quase ninguém sobrevive ao dia do pico e ao recurso em massa"**.
> É um argumento mais estreito e mais verdadeiro.

---

## 8. Entrada no mercado

| Fase | Quando | O quê |
|---|---|---|
| **1 · Primeiro certame** | M0–M6 | Um processo seletivo simplificado de prefeitura ou vestibular de faculdade privada. Poucos milhares de candidatos. **Ponta a ponta.** Preço simbólico em troca de estudo de caso |
| **2 · Bancas médias** | M6–M24 | Venda direta. O estudo de caso do ciclo fechado é a peça |
| **3 · Expansão dentro do cliente** | M18+ | A banca migra certames maiores para a plataforma. **É aqui que o ACV cresce** — mais do que pela entrada de novos logos |
| **4 · Adjacência** | Ano 4+ | Certificação profissional, conselho de classe, seleção corporativa. Mesmo motor, TAM novo |

**O ativo de marketing é o dado.** Publicar, a cada temporada, um relatório do comportamento de
inscrição — concentração no último dia, horário de pico, taxa de isenção deferida, tempo médio de
julgamento de recurso. Ninguém publica isso, todo diretor de banca quer ler, e é a única peça que
gera entrada sem prospecção fria.

---

## 9. Riscos

| Risco | Prob. | Impacto | Mitigação |
|---|:-:|:-:|---|
| **Preço por inscrição abaixo de R$ 2,15** | Média | **Crítico** | É a variável que decide o negócio. **Verificar antes de qualquer outra coisa.** Se o mercado pagar menos, o plano muda de estrutura, não de detalhe |
| **Falha durante o certame → anulação** | Baixa | **Crítico** | Anular um certame por falha de sistema é dano de escala imprevisível: candidato perde direito, órgão perde o contrato, há ação coletiva. Teto de responsabilidade contratual, seguro de RC, e **ensaio de carga contratualizado antes de cada certame grande** |
| **Concentração de receita por evento** | Alta | Alto | Um certame grande pode ser 30% da receita do ano. Diversificar por banca e por calendário |
| **Sazonalidade de caixa** | Alta | Médio | Receita segue calendário de editais, custo é mensal. Assinatura anual antecipada existe para isso |
| ~~**Ciclo eleitoral**~~ | — | — | **Verificado e rebaixado.** A Lei 9.504/97, art. 73, V, veda a **nomeação** nos três meses anteriores ao pleito — não a realização do certame. Editais são publicados e provas aplicadas normalmente em ano eleitoral. O efeito sobre a nossa receita é indireto e pequeno |
| **TAM modesto** | Alta | Médio | Conhecido desde o início. A resposta é adjacência, e ela precisa estar no plano do ano 3, não do ano 5 |
| **Banca perde a licitação do órgão** | Média | Médio | Perda indireta que não depende da nossa performance. Contrato plurianual com a banca, não por certame |
| **A banca constrói em casa** | Média | Alto | Conformidade, auditoria e pico não se amortizam em 20 certames por ano. É o argumento de comprar em vez de construir, e precisa estar no material de venda |

---

## 10. Financiamento

| Rodada | Momento | Valor | Uso |
|---|---|---:|---|
| Pre-seed | M0 | R$ 1,5 mi | MVP e primeiro certame ponta a ponta |
| **Seed** | M8 | **R$ 10 mi** | Engenharia, conformidade, primeiras 12 bancas |
| Série A | M26 | R$ 35 mi | Escala de vendas, ensalamento e correção em produção |
| **Série B** | M46 | **R$ 80 mi** | Adjacência e LatAm — **e é aqui que entra a decisão de arquitetura** |

---

## 11. A premissa que o trabalho de multicloud herda

No ano 5 a empresa é:

| | |
|---|---|
| Clientes | **72 bancas organizadoras** |
| Receita | **R$ 27,4 mi de ARR**, EBITDA +R$ 5,3 mi |
| Volume | **7,7 milhões de inscrições por ano** |
| Perfil de carga | **Cinco picos por certame**, dezenas de certames em paralelo, cada pico com data conhecida e inegociável |
| Consequência de falha | Candidato perde o direito de concorrer. **Anulação de certame, ação coletiva, perda do contrato pela banca** |

**A pergunta do conselho, e do trabalho:** *vale uma segunda nuvem para as janelas críticas?*

O que torna a pergunta genuinamente difícil:

- **A restrição é temporal, não geográfica.** Sabemos exatamente quando a falha importa — o que
  favorece o **não**, porque dá para pré-aquecer capacidade em vez de manter réplica.
- **Mas a janela é inegociável.** Não existe "tenta de novo amanhã" quando o prazo de inscrição
  encerra à meia-noite — o que favorece o **sim**.
- **E a carga é ociosa a maior parte do ano**, o que torna o custo de uma segunda nuvem
  desproporcional ao uso — de novo o **não**.

**As três forças não apontam para o mesmo lado, e é por isso que a defesa vale a pena escrever.**

---

## 12. Resultado das verificações

| # | Pergunta | Status |
|---|---|---|
| 1 | Preço por inscrição praticado | ⚠️ **Continua não verificado.** Nenhum fornecedor publica tabela. **Âncoras encontradas:** taxa documentada de **R$ 69,33 por candidato** destinada a custear os serviços da banca; e o **Gestor Editais cobra sobre o total de inscrições**, o que valida o modelo mas não o valor |
| 2 | Quem já vende para banca média | ✅ **Verificado — e o mercado está ocupado.** Ver §7 |
| 3 | Concentração da inscrição nas últimas 48 h | ⚠️ **Sem número público.** Mas o **fenômeno é documentado**: há registro de sistema fora do ar no fim do prazo e de **prazo de inscrição estendido por decisão judicial** por indisponibilidade do site |
| 4 | Ciclo eleitoral | ✅ **Verificado, e o risco era menor do que eu supus** — a vedação é de nomeação, não de certame |

### Dois achados que não estavam na lista e mudam decisões

**1 · O modelo "custo zero" é comum e define a economia do cliente.** Em boa parte das
contratações, a banca assume todos os custos do certame e **fica com a arrecadação das taxas de
inscrição**. Isso é ótimo para nós: o cliente tem receita variável por candidato, exatamente como
o nosso preço — **os incentivos se alinham sem negociação.**

**2 · Taxa de inscrição é receita pública, e há entendimento de controle contra delegar a gestão
desse dinheiro a empresa privada.** Isso **valida juridicamente** a decisão de §3: **orquestramos o
pagamento, não o processamos.** Deixa de ser simplificação de engenharia e passa a ser requisito.

### Fontes

- [Lei nº 14.965/2024 — Lei Geral dos Concursos Públicos](https://www.jusbrasil.com.br/legislacao/2719968211/lei-14965-24) · [análise das mudanças](https://www.jusbrasil.com.br/artigos/principais-mudancas-nos-concursos-publicos-com-a-lei-14965-2024/2754141665)
- [Gestor Editais](https://gestoreditais.com.br/) · [ProSeleta](https://www.proseleta.com.br/) · [Fábrica de Provas](https://www.fabricadeprovas.com.br/) · [ADSR](https://www.adsr.com.br/sistema-concursos/) · [MH Informática](https://mhinformatica.inf.br/concursos-publicos) · [Instituto Avalia](https://avalia.org.br/atuacao)
- [TCE-MG — consulta sobre taxas de inscrição em concursos](https://www.tce.mg.gov.br/TCE-responde-consulta-sobre-taxas-de-inscricao-em-concursos-publicos.html/Noticia/1111620521) · [MPRJ — contratação de empresas para realização de concursos](https://www.mprj.mp.br/documents/20184/6380157/rita_tourinho.pdf/)
- [Indisponibilidade do sistema e perda do prazo de inscrição](https://fabioportela.com/concursopublico/blog/o-que-fazer-quando-o-sistema-de-inscricao-no-concurso-publico-fica-indisponivel-e-o-concurseiro-perde-o-prazo-de-inscricao/)
- [Concurso público em ano eleitoral — regras de nomeação](https://www.tecconcursos.com.br/blog/noticias/concurso-publico-em-ano-eleitoral/)
