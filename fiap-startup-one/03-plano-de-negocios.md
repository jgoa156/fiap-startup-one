# 3. Plano de negócios

*Plataforma B2B de inscrição e gestão de processos seletivos, vendida para bancas organizadoras.
Produto e fluxos em [02-produto-features-e-fluxos.md](02-produto-features-e-fluxos.md).*

> **Aviso de honestidade metodológica.** Quatro números deste plano **não foram verificados** e
> estão marcados como `⚠ NV`. O principal deles, o preço por inscrição, decide se este é um negócio
> de R$ 10 milhões ou de R$ 27 milhões. Ele é tratado como **variável**, com faixa e sensibilidade,
> e não como premissa. Assumir premissa não verificada é o erro mais caro que um plano pode conter.

---

## 1. Resumo

| | |
|---|---|
| **O que é** | Plataforma que conduz um processo seletivo de ponta a ponta, do edital à homologação do resultado |
| **Cliente** | Banca organizadora de porte médio: 5 a 50 certames por ano |
| **Por que agora** | **Lei 14.965/2024**, sancionada em 09/09/2024, estabelece normas gerais para os concursos de **todos os entes** a partir de 1º/01/2028 (estados, DF e municípios podem, em alternativa, editar normas próprias) e autoriza prova total ou parcialmente a distância. A lei **não obriga ninguém a trocar de sistema**, mas muda as regras que o sistema precisa cumprir, e a prova on-line **já é praticada em 2026** (AgSUS). Quem se adequar vai escolher fornecedor |
| **Como cobra** | Assinatura anual pequena mais **valor por inscrição processada** (o segundo domina) |
| **Vantagem comercial** | **Vender para a banca evita licitação.** O órgão licita a banca; a banca contrata fornecedor como empresa privada |
| **Fosso** | Sistema de registro do certame e conta única do candidato, que atravessa bancas |
| **Meta** | R$ 17 mi de ARR no ano 5, breakeven no 4º trimestre do ano 4 |

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
| Quantas bancas de porte médio e pequeno existem | 80 a 200 | `⚠ NV` |
| Concentração da inscrição nas últimas 48 h do prazo | 80 a 90% | `⚠ NV` |

### TAM, SAM e SOM

| | Cálculo | Valor |
|---|---|---|
| **TAM** | 30 mi de inscrições × R$ 3 | **R$ 90 mi/ano** |
| **SAM** | Bancas médias e pequenas, ~40% do volume | **R$ 36 mi/ano** |
| **SOM (ano 5)** | 45 bancas, 4,8 mi de inscrições | **R$ 17,1 mi de ARR** |

### A premissa de participação, dita explicitamente

O plano assume que o segmento de bancas médias e pequenas tem **cerca de 150 empresas**, o ponto
médio da faixa não verificada. Sob essa premissa, **45 clientes no ano 5 são 30% do segmento**, o
que é liderança de mercado sem ser implausível.

**A faixa inteira, e o que ela faz com o plano:**

| Se o segmento tiver | 45 clientes seriam | Leitura |
|---:|---:|---|
| 80 bancas | **56%** | Implausível com seis concorrentes estabelecidos. O plano teria que parar perto de 25 bancas, e o ARR do ano 5 cairia para cerca de R$ 9,5 mi |
| 150 bancas | **30%** | Premissa do plano |
| 200 bancas | **22%** | Confortável |

> **Isto é uma dependência, não um detalhe.** Contar quantas bancas de porte médio existem no
> Brasil é a segunda verificação mais urgente do plano, atrás só do preço por inscrição.

> **O TAM é modesto, e isso precisa ser dito antes que perguntem.** Mesmo dominando o segmento
> alvo, a categoria isolada não passa de algumas dezenas de milhões. **A empresa cresce por
> adjacência, não por participação:** certificação profissional, prova de conselho de classe,
> seleção corporativa em massa e LatAm, todos usando o mesmo motor de "muita gente, prazo curto,
> resultado contestável".

---

## 3. Modelo de receita

### A decisão que mantém o negócio simples

> **Nós orquestramos o pagamento. Não processamos o pagamento.**

O boleto e o Pix são emitidos pelo provedor **da própria banca**; nós disparamos a cobrança e
recebemos a confirmação. Três consequências, todas boas:

1. A receita fica limpa, sem repasse inflando o faturamento.
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

**V1, o preço por inscrição processada.** Contextualizando: sobre uma taxa média de R$ 80 paga pelo
candidato, R$ 3,00 representam **3,75% da taxa**.

### Sensibilidade no ano 5, com 45 bancas e 4,8 milhões de inscrições

O volume não muda entre os cenários, só o preço. Como o custo total do ano 5 é **R$ 13,7 mi** e
não depende de V1, **cada real a mais ou a menos de receita vai inteiro para o EBITDA.**

| V1 | Receita por inscrição | Assinaturas | **Receita total** | EBITDA | Veredito |
|---:|---:|---:|---:|---:|---|
| **R$ 1,50** | 7,2 | 2,7 | **R$ 9,9 mi** | **−R$ 3,8 mi** | ❌ **Não fecha** com esta estrutura |
| **R$ 2,29** | 11,0 | 2,7 | **R$ 13,7 mi** | **≈ 0** | ⚠️ **Piso de viabilidade** |
| **R$ 3,00** | 14,4 | 2,7 | **R$ 17,1 mi** | **+R$ 3,4 mi** | ✅ Cenário base |
| **R$ 5,00** | 24,0 | 2,7 | **R$ 26,7 mi** | **+R$ 13,0 mi** | ✅ Excedente reinvestido |

> **O achado:** abaixo de **R$ 2,29 por inscrição** o negócio não fecha com 45 clientes e esta
> estrutura de custo. R$ 2,29 são **2,9% de uma taxa de R$ 80**. Ou o preço sobe, ou a base de
> clientes precisa ser maior, ou o time precisa ser menor. **Descobrir o preço praticado é a
> tarefa número um**, antes de escrever mais uma linha de plano.

---

## 5. Unit economics (base: ano 3)

> **O CAC aqui não é estimado, é derivado.** Ele é a linha inteira de Vendas e Marketing dividida
> pelas adições brutas de clientes daquele ano, incluindo liderança comercial, marketing e o custo
> de repor a perda. Um CAC calculado só sobre o vendedor daria um número muito melhor e muito
> menos verdadeiro.

| Métrica | Valor | Como |
|---|---|---|
| ACV médio | **R$ 245 mil** | R$ 60 mil de assinatura mais 62 mil inscrições × R$ 3 |
| Margem bruta | **77%** | Infra elástica e suporte |
| **CAC** | **R$ 158 mil** | S&M de R$ 1,5 mi ÷ 9,5 adições brutas |
| **Retorno de CAC** | **10 meses** | CAC ÷ (ACV × margem ÷ 12) |
| Perda de cliente | **6% ao ano** | Sistema de registro do certame; ninguém troca no meio |
| LTV (5 anos, com desconto) | ~R$ 710 mil | Truncado em 5 anos de propósito: a 6% de perda, o cálculo infinito estoura e engana |
| **LTV / CAC** | **~4,5×** | |

> **A economia unitária é boa e não é o gargalo.** Retorno de CAC em 10 meses e LTV/CAC de 4,5×
> são números saudáveis de SaaS B2B, e são conservadores porque o CAC inclui a estrutura comercial
> inteira, não apenas o vendedor. O gargalo é o tamanho do mercado, o inverso do padrão, em que a
> economia é apertada e o mercado é enorme.

**E há uma vantagem estrutural que merece registro:** aqui **não existe partida a frio**. O produto
entrega valor integral ao primeiro cliente, sozinho. A conta única do candidato acrescenta valor
com escala, mas não é condição para funcionar.

---

## 6. Projeções (R$ milhões)

| | Ano 1 | Ano 2 | Ano 3 | Ano 4 | Ano 5 |
|---|---:|---:|---:|---:|---:|
| Bancas clientes (fim do ano) | 3 | 9 | 18 | 31 | 45 |
| Adições brutas no ano | 3,0 | 6,2 | 9,5 | 14,1 | 15,9 |
| Inscrições processadas (milhões) | 0,10 | 0,41 | 1,11 | 2,64 | 4,80 |
| ACV médio (R$ mil) | 160 | 195 | 245 | 315 | 380 |
| **Receita (ARR)** | **0,5** | **1,8** | **4,4** | **9,8** | **17,1** |
| Custo dos serviços | 0,2 | 0,5 | 1,0 | 1,8 | 3,0 |
| *dos quais infraestrutura* | *0,1* | *0,2* | *0,5* | *0,8* | *1,2* |
| **Margem bruta** | **60%** | **72%** | **77%** | **82%** | **82%** |
| P&D / Engenharia | 1,8 | 2,7 | 3,6 | 4,6 | 5,6 |
| Vendas e marketing | 0,5 | 0,9 | 1,5 | 2,1 | 2,7 |
| G&A / jurídico | 0,4 | 0,7 | 1,2 | 1,8 | 2,4 |
| **EBITDA** | **−2,4** | **−3,0** | **−2,9** | **−0,5** | **+3,4** |
| Margem EBITDA | | | | −5% | **+20%** |
| Headcount | 9 | 17 | 26 | 35 | 44 |
| *dos quais plataforma* | *1* | *2* | *3* | *4* | *5* |

> **Conferência da linha de receita:** ARR = bancas × ACV médio. Ano 3: 18 × 245 mil = 4,4 mi.
> Ano 5: 45 × 380 mil = 17,1 mi. E, pelo outro lado, ano 5 = 4,80 mi de inscrições × R$ 3,00
> (14,4) mais 45 assinaturas de R$ 60 mil (2,7) = 17,1 mi.

**Breakeven: 4º trimestre do ano 4.** É esse marco, somado à rodada do ano 4, que constitui a
premissa do trabalho de Soluções Multicloud.

**Por que P&D é enxuto:** não há time de ciência de dados, não há plataforma de ML, não há
retreino. É engenharia de produto e de plataforma, mais barata e mais fácil de contratar.

### O que já está dentro deste orçamento

O programa de arquitetura descrito em
[`../fiap-multicloud/04-arquitetura-alvo.md`](../fiap-multicloud/04-arquitetura-alvo.md), que custa
**R$ 900 mil/ano**, **não é um acréscimo a estas linhas: ele já está dentro delas.** Células,
publicação estática, geração assíncrona, fila de espera, DNS, WAF e backup fora da AWS estão dentro
da infraestrutura de R$ 1,2 mi; o ensaio de carga e a restauração testada no segundo provedor
estão dentro do P&D e do time de plataforma de 5 pessoas.

> **Isso é deliberado, e é a decisão mais importante deste plano.** Capacidade comprovada é o
> produto que vendemos, não uma melhoria opcional. Se estivesse fora do orçamento, seria a
> primeira coisa a ser cortada num trimestre ruim, e é a única que não pode ser.

---

## 7. Concorrência, verificada

**O mercado existe e está ocupado.** Foi a verificação nº 2, e a resposta é sim.

| Concorrente | O que é |
|---|---|
| **Gestor Editais** (Grupo ACT) | Plataforma de concursos e processos seletivos. **Cobra sobre o total de inscrições que o certame alcançar** |
| **ProSeleta** | Gestão de todas as fases, da inscrição ao resultado final |
| **Fábrica de Provas** | Plataforma de avaliação para instituições de ensino, seletivos, certificação e concursos. Integra inscrição, aplicação e correção |
| **ADSR** | ERP para concursos: inscrição automatizada, validação, processamento de isenções |
| **MH Informática** | Correção eletrônica de cartão-resposta, exportação para gestão pública |
| **Instituto Avalia**, **Gestão de Concursos** | Operam prova online e gestão pública |
| **Sistemas institucionais** (UNICAMP, FCPC/UFC) | Construídos em casa por fundações universitárias |

### O que isso muda, e não é fatal

1. **O modelo de preço está validado.** O Gestor Editais **já cobra por inscrição**. Não estamos
   inventando um modelo; estamos entrando num mercado que já precifica assim.
2. **A tese "banca média opera com planilha" está errada.** Ela opera com **um destes**. A venda
   deixa de ser evangelização e vira substituição: ciclo mais curto, objeção diferente.
3. **A diferenciação precisa mudar de lugar.** Não é "existir", é fazer melhor o que dói. As
   apostas passam a ser **recurso agrupado por fundamento estruturado**, **recálculo versionado
   após anulação de questão**, **trilha que sobrevive a ação judicial anos depois** e
   **capacidade de pico comprovada por ensaio de carga contratual**, que é justamente o que
   nenhum concorrente anuncia.

> **Correção honesta ao pitch:** o argumento deixa de ser "ninguém faz isso" e passa a ser
> **"todos fazem a inscrição; quase ninguém sobrevive ao dia do pico e ao recurso em massa"**.
> É um argumento mais estreito e mais verdadeiro.

---

## 8. Entrada no mercado

| Fase | Quando | O quê |
|---|---|---|
| **1 · Primeiro certame** | M0 a M6 | Um processo seletivo simplificado de prefeitura ou vestibular de faculdade privada. Poucos milhares de candidatos. **Ponta a ponta.** Preço simbólico em troca de estudo de caso |
| **2 · Bancas médias** | M6 a M24 | Venda direta. O estudo de caso do ciclo fechado é a peça |
| **3 · Expansão dentro do cliente** | M18+ | A banca migra certames maiores para a plataforma. **É aqui que o ACV cresce**, mais do que pela entrada de novos logos |
| **4 · Adjacência** | Ano 4+ | Certificação profissional, conselho de classe, seleção corporativa. Mesmo motor, TAM novo |

**O ativo de marketing é o dado.** Publicar, a cada temporada, um relatório do comportamento de
inscrição: concentração no último dia, horário de pico, taxa de isenção deferida, tempo médio de
julgamento de recurso. Ninguém publica isso, todo diretor de banca quer ler, e é a única peça que
gera entrada sem prospecção fria.

---

## 9. Riscos

| Risco | Prob. | Impacto | Mitigação |
|---|:-:|:-:|---|
| **Preço por inscrição abaixo de R$ 2,29** | Média | **Crítico** | É a variável que decide o negócio. **Verificar antes de qualquer outra coisa.** Se o mercado pagar menos, o plano muda de estrutura, não de detalhe |
| **O segmento ser menor que 150 bancas** | Média | **Crítico** | Se forem 80, o teto realista cai para cerca de 25 clientes e o ARR do ano 5 para cerca de R$ 9,5 mi. O negócio continua existindo, mas deixa de comportar a estrutura de custo projetada |
| **Falha durante uma prova on-line, se entrarmos nesse mercado** | Baixa | **Crítico** | Hoje a Certamis não aplica prova. Se passar a aplicar on-line, a falha tem **remédio de outra ordem de grandeza**: anular e reaplicar a prova, como na AgSUS em 2026, com nova data para toda a base, desgaste com o órgão e risco de ação coletiva. Teto de responsabilidade contratual, seguro de RC, **ensaio de carga contratualizado**, e a decisão de multicloud é reaberta (gatilho 2) |
| **Indisponibilidade durante um prazo** (isenção, inscrição, recurso) | Média | Médio | **Remédio barato, mas não garantido.** O remédio praticado é reabrir o prazo. No caso Alece (2026) a banca reabriu o recurso, e o juiz negou reabrir a isenção com base na instabilidade. Custa reputação e pode custar o contrato. É a distinção que organiza a defesa de arquitetura em [`../fiap-multicloud/03-defesa-multicloud.md`](../fiap-multicloud/03-defesa-multicloud.md) |
| **Concentração de receita por evento** | Alta | Alto | Pesa nos primeiros anos: um certame de 300 mil candidatos a R$ 3 são R$ 0,9 mi, metade da receita do ano 2 e 5% da do ano 5. Diversificar por banca e por calendário |
| **Sazonalidade de caixa** | Alta | Médio | Receita segue calendário de editais, custo é mensal. Assinatura anual antecipada existe para isso |
| ~~**Ciclo eleitoral**~~ | | | **Verificado e rebaixado.** A Lei 9.504/97, art. 73, V, veda a **nomeação** nos três meses anteriores ao pleito, não a realização do certame. Editais são publicados e provas aplicadas normalmente em ano eleitoral |
| **TAM modesto** | Alta | Médio | Conhecido desde o início. A resposta é adjacência, e ela precisa estar no plano do ano 3, não do ano 5 |
| **Banca perde a licitação do órgão** | Média | Médio | Perda indireta que não depende da nossa performance. Contrato plurianual com a banca, não por certame |
| **A banca constrói em casa** | Média | Alto | Conformidade, auditoria e pico não se amortizam em 20 certames por ano. É o argumento de comprar em vez de construir, e precisa estar no material de venda |

---

## 10. Financiamento

Queima acumulada até o breakeven: **R$ 8,8 milhões**. As rodadas cobrem isso com folga e financiam
crescimento, não sobrevivência.

| Rodada | Momento | Valor | Uso |
|---|---|---:|---|
| Pre-seed | M0 | R$ 1,5 mi | MVP e primeiro certame ponta a ponta |
| **Seed** | M8 | **R$ 8 mi** | Engenharia, conformidade, primeiras 9 bancas |
| Série A | M28 | R$ 15 mi | Escala de vendas, ensalamento e correção em produção |
| **Série B** | M46 | **R$ 30 mi** | Adjacência e LatAm, **e é aqui que entra a decisão de arquitetura** |

> **Por que a Série B é de R$ 30 mi e não maior.** Contra um ARR de R$ 17,1 mi, são 1,8× a
> receita, um múltiplo normal. Contra o TAM de R$ 90 mi da categoria, é um terço do mercado
> inteiro, e por isso a rodada só se justifica se financiar **adjacência**, que é TAM novo. Uma
> rodada maior que esta não teria onde ser aplicada dentro da categoria, e um investidor que faça
> a divisão vai perguntar exatamente isso.

---

## 11. A premissa que o trabalho de multicloud herda

No ano 5 a empresa é:

| | |
|---|---|
| Clientes | **45 bancas organizadoras** |
| Receita | **R$ 17,1 mi de ARR**, EBITDA +R$ 3,4 mi (20%) |
| Volume | **4,8 milhões de inscrições por ano** |
| Time de plataforma | **5 pessoas**, de 44 no total |
| Infraestrutura | **R$ 1,2 mi/ano**, já incluindo o programa de capacidade |
| Perfil de carga | **Cinco picos por certame**, dezenas de certames em paralelo, cada pico com data conhecida e inegociável |
| Consequência de falha | **Depende da fase, e essa distinção é o eixo da defesa.** Nas fases de prazo (isenção, inscrição, cartão, gabarito, resultado) a falha tem **remédio barato, embora não garantido**: reabrir o prazo ou republicar. Numa **prova sendo aplicada on-line**, o remédio é **anular e reaplicar a prova**, como aconteceu na AgSUS em 2026, e custa uma ordem de grandeza a mais |

**A pergunta do conselho, e do trabalho:** *o programa de capacidade já está no orçamento. Vale
acrescentar uma segunda nuvem em cima dele?*

O que torna a pergunta genuinamente difícil:

- **A restrição é temporal, não geográfica.** Sabemos exatamente quando a falha importa, o que
  favorece o **não**, porque dá para pré-aquecer capacidade em vez de manter réplica.
- **Mas a janela é inegociável.** Não existe "tenta de novo amanhã" quando o prazo de inscrição
  encerra à meia-noite, o que favorece o **sim**.
- **E a carga é ociosa a maior parte do ano**, o que torna o custo de uma segunda nuvem
  desproporcional ao uso, de novo o **não**.

**As três forças não apontam para o mesmo lado, e é por isso que a defesa vale a pena escrever.**
A resposta a que ela chegou: **uma nuvem para operar, duas para sobreviver.** Controle, backup e
restauração a frio ficam fora da AWS, por R$ 170 mil por ano dentro do orçamento acima; a segunda
nuvem sempre ligada foi rejeitada.

---

## 12. Resultado das verificações

| # | Pergunta | Status |
|---|---|---|
| 1 | Preço por inscrição praticado | ⚠️ **Continua não verificado.** Nenhum fornecedor publica tabela. **Âncoras encontradas:** taxa documentada de **R$ 69,33 por candidato** destinada a custear os serviços da banca; e o **Gestor Editais cobra sobre o total de inscrições**, o que valida o modelo mas não o valor |
| 2 | Quem já vende para banca média | ✅ **Verificado, e o mercado está ocupado.** Ver §7 |
| 3 | Quantas bancas de porte médio existem | ⚠️ **Não verificado, e o plano depende disso.** Ver §2. A premissa de 150 é o ponto médio de uma faixa que ninguém publicou |
| 4 | Concentração da inscrição nas últimas 48 h | ⚠️ **Sem número público.** Mas o **fenômeno de queda em certame é documentado**, com três casos em 2026: Alece (isenção e recurso), FGV (inscrições) e AgSUS (durante a prova on-line). Ver [`../fiap-multicloud/06-referencias.md` §6.4](../fiap-multicloud/06-referencias.md) |
| 5 | Ciclo eleitoral | ✅ **Verificado, e o risco era menor do que a suposição inicial:** a vedação é de nomeação, não de certame |

### Dois achados que não estavam na lista e mudam decisões

**1 · O modelo "custo zero" é comum e define a economia do cliente.** Em boa parte das
contratações, a banca assume todos os custos do certame e **fica com a arrecadação das taxas de
inscrição**. Isso é bom para nós: o cliente tem receita variável por candidato, exatamente como
o nosso preço, e **os incentivos se alinham sem negociação.**

**2 · Taxa de inscrição é receita pública, e há entendimento de controle contra delegar a gestão
desse dinheiro a empresa privada.** Isso **valida juridicamente** a decisão de §3: **orquestramos o
pagamento, não o processamos.** Deixa de ser simplificação de engenharia e passa a ser requisito.

### Fontes

- [Lei nº 14.965/2024, Lei Geral dos Concursos Públicos](https://www.jusbrasil.com.br/legislacao/2719968211/lei-14965-24) · [análise das mudanças](https://www.jusbrasil.com.br/artigos/principais-mudancas-nos-concursos-publicos-com-a-lei-14965-2024/2754141665)
- [Gestor Editais](https://gestoreditais.com.br/) · [ProSeleta](https://www.proseleta.com.br/) · [Fábrica de Provas](https://www.fabricadeprovas.com.br/) · [ADSR](https://www.adsr.com.br/sistema-concursos/) · [MH Informática](https://mhinformatica.inf.br/concursos-publicos) · [Instituto Avalia](https://avalia.org.br/atuacao)
- [TCE-MG, consulta sobre taxas de inscrição em concursos](https://www.tce.mg.gov.br/TCE-responde-consulta-sobre-taxas-de-inscricao-em-concursos-publicos.html/Noticia/1111620521) · [MPRJ, contratação de empresas para realização de concursos](https://www.mprj.mp.br/documents/20184/6380157/rita_tourinho.pdf/)
- [Indisponibilidade do sistema e perda do prazo de inscrição](https://fabioportela.com/concursopublico/blog/o-que-fazer-quando-o-sistema-de-inscricao-no-concurso-publico-fica-indisponivel-e-o-concurseiro-perde-o-prazo-de-inscricao/)
- [Concurso público em ano eleitoral, regras de nomeação](https://www.tecconcursos.com.br/blog/noticias/concurso-publico-em-ano-eleitoral/)
