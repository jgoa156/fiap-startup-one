# 1. A empresa e o contexto

*(Atende ao item "Breve descrição do projeto/empresa e sua área de atuação".)*

---

## 1.1 O que é a Certamis

**Certamis** é a plataforma que conduz um processo seletivo público de ponta a ponta, do edital à
homologação do resultado. Vendida para **bancas organizadoras**, que são as empresas contratadas
por prefeituras, autarquias, universidades e hospitais para realizar concursos e vestibulares.

*O nome vem de "certame", como a administração pública chama o processo seletivo público. É o
objeto inteiro do produto.*

| | |
|---|---|
| **Setor** | Software B2B para o setor de certames públicos |
| **Cliente** | Banca organizadora de porte médio, de 5 a 50 certames por ano |
| **Usuário final** | O candidato, que se inscreve, paga, recorre e consulta resultado |
| **Modelo** | Assinatura anual mais **valor por inscrição processada** (o segundo domina) |
| **Fundada** | Ano 1 do plano, com um processo seletivo simplificado de prefeitura |

### O que a plataforma faz

Oito fases encadeadas, cada uma com prazo legal e cada uma recorrível: configuração do certame,
publicação do edital, **isenção de taxa**, **inscrição**, homologação, **ensalamento**,
**gabarito e recurso**, **resultado e classificação**.

Detalhe completo de funcionalidades e fluxos em
[`../fiap-startup-one/02-produto-features-e-fluxos.md`](../fiap-startup-one/02-produto-features-e-fluxos.md).

---

## 1.2 Por que este negócio exige nuvem

Não por "escalabilidade" genérica. Por um perfil de carga que é o mais extremo que existe em
software de negócio: **a carga de um certame se concentra em poucas horas, e essas horas têm data
marcada em edital.**

### Os cinco picos de um certame

| # | Pico | Quem acessa | Por que dói |
|---|---|---|---|
| **1** | Encerramento do prazo de **isenção** | Quem pede isenção | Prazo anterior ao da inscrição, já concentrado |
| **2** | Encerramento do prazo de **inscrição** | Quem ainda não se inscreveu | **Escrita pesada**: cadastro, upload, cobrança |
| **3** | Liberação do **cartão de confirmação** | **Toda a base** | Leitura simultânea de todos os inscritos |
| **4** | **Gabarito preliminar** e janela de recurso | **Toda a base** | Leitura em massa seguida de escrita em massa |
| **5** | **Resultado e classificação** | **Toda a base** | O maior de todos |

> **Correção importante sobre o perfil:** o pico de inscrição não é o pior em número de acessos.
> Nele, só acessa quem ainda não se inscreveu. Nos picos 3, 4 e 5 acessa **a base inteira, ao
> mesmo tempo**. Mas o pico 2 continua sendo o mais perigoso, porque é **escrita**, e escrita não
> se resolve com CDN.

### O que "40× o pico" quer dizer exatamente

Este número aparece no contrato e no ensaio de carga, então precisa ser definido sem ambiguidade.

> **É 40× a média do próprio certame, não a média anual da empresa.**

A média anual é uma abstração sem uso operacional: ela mistura dezenas de certames em fases
diferentes. O que a infraestrutura precisa aguentar é o pico **de um certame concreto** contra a
média **daquele certame**.

| | Certame de 100 mil candidatos | Certame de 300 mil candidatos |
|---|---:|---:|
| Janela de inscrição | 30 dias | 30 dias |
| Média do certame | 139 inscrições/h | 417 inscrições/h |
| **Pico a 40×** | **5,6 mil/h** | **16,7 mil/h**, ou 4,6 por segundo |

Para contexto, a média anual da empresa inteira no ano 5 é de cerca de **550 inscrições por
hora**. O pico de um único certame grande é, sozinho, trinta vezes a operação média da empresa.

**O pico de leitura é maior em requisições**, porque no dia do resultado a base inteira consulta
ao mesmo tempo. Mas ele é servido de CDN como arquivo estático, e por isso não é o que dimensiona
o banco de dados. Ver [`04-arquitetura-alvo.md` §4.2](04-arquitetura-alvo.md).

### E o pico é conhecido com meses de antecedência

O cronograma está publicado no edital. **Sabemos a data e a hora em que precisaremos de quarenta
vezes a capacidade média daquele certame.** Isso é raro, e é a característica que organiza toda a
decisão de arquitetura, inclusive a de multicloud.

### A consequência da falha, e o limite dessa afirmação

A falha é pública, é notícia, e chega ao Judiciário. Em 2026 houve pelo menos três casos
documentados: a Alece teve instabilidade na isenção e no recurso, a prova on-line da AgSUS foi
anulada e remarcada, e o site da FGV ficou dias fora do ar após um ataque. Casos e fontes em
[`06-referencias.md` §6.4](06-referencias.md).

**Nas fases de prazo, a falha tem remédio praticado, mas não garantido.** Na Alece, a banca reabriu
o prazo de recurso depois da falha; na mesma Alece, o juiz negou a reabertura da isenção pedida com
base na instabilidade. Esse remédio, quando vem, é barato: reabrir um prazo custa dias, não o
certame. Isso é dado de entrada da decisão de continuidade.

> **Numa prova sendo aplicada on-line, o remédio muda de ordem de grandeza.** Deixa de ser reabrir
> um prazo e passa a ser anular e reaplicar a prova inteira, como aconteceu na AgSUS. É por isso
> que a prova on-line aparece como gatilho de reavaliação em
> [`03-defesa-multicloud.md`](03-defesa-multicloud.md) §3.13. A distinção entre remédio barato e
> remédio caro é o eixo da defesa inteira.

---

## 1.3 Por que agora: a janela regulatória

A **Lei nº 14.965/2024**, sancionada em 9 de setembro de 2024, estabelece **normas gerais sobre
concurso público**. Pelo texto:

- vale para os concursos **de todos os entes**, com exceções (magistratura, Ministério Público,
  Defensoria, carreiras militares); estados, DF e municípios podem, **em alternativa**, editar
  normas próprias (art. 13, §2º);
- **entra em vigor em 1º de janeiro de 2028**, e cada concurso pode antecipar sua aplicação no ato
  de abertura (art. 13);
- **autoriza prova total ou parcialmente a distância** (art. 8º), como faculdade que depende de
  regulamentação.

> **O que a janela é, sem exagero para nenhum lado.** A lei não obriga ninguém a trocar de sistema
> nem a fazer prova on-line. O que ela faz é **mudar as regras do concurso para quase todo o setor
> público em 2028**, e dar base legal expressa à prova a distância, que **já está sendo praticada em
> 2026** (a AgSUS aplicou prova on-line remota). Sistemas construídos para as regras antigas terão
> de se adequar, e a banca que fizer isso vai escolher fornecedor. É a janela que justifica o
> *timing* da empresa.
>
> *Versões anteriores diziam "todo o mercado precisa se replataformar", e uma correção de
> 22/09/2026 disse "obrigatória só para concursos federais". As duas estavam erradas; o texto da lei
> está em [`06-referencias.md` §6.2](06-referencias.md).*

---

## 1.4 O mercado

| | Faixa | Status |
|---|---|---|
| Inscrições pagas por ano no Brasil | 20 a 40 milhões | ⚠️ não verificado |
| Taxa média paga pelo candidato | ~R$ 80 | ⚠️ âncora: R$ 69,33 documentado |
| Bancas de porte médio e pequeno | 80 a 200 | ⚠️ não verificado, **e o plano depende disso** |
| **TAM da camada de plataforma** | **~R$ 90 mi/ano** | Modesto, com crescimento por adjacência |

O plano assume **cerca de 150 bancas** no segmento, e as 45 clientes do ano 5 são 30% dele. A
premissa está declarada, com sensibilidade, em
[`../fiap-startup-one/03-plano-de-negocios.md` §2](../fiap-startup-one/03-plano-de-negocios.md).

**O mercado está ocupado:** Gestor Editais (Grupo ACT), ProSeleta, Fábrica de Provas, ADSR, MH
Informática, Instituto Avalia. O argumento não é "ninguém faz isso". É **"todos fazem a
inscrição; quase ninguém sobrevive ao dia do pico e ao recurso em massa"**.

---

## 1.5 Onde a empresa está: a premissa deste trabalho

**A Certamis não é mais uma startup.** O trabalho parte do ano 5:

| Indicador | Ano 1 | **Ano 5** |
|---|---:|---:|
| Bancas clientes | 3 | **45** |
| Inscrições processadas por ano | 0,10 mi | **4,80 mi** |
| Receita (ARR) | R$ 0,5 mi | **R$ 17,1 mi** |
| EBITDA | −R$ 2,4 mi | **+R$ 3,4 mi (20%)** |
| Headcount | 9 | **44** |
| Time de plataforma | 1 | **5** |
| Infraestrutura | R$ 0,1 mi | **R$ 1,2 mi/ano** |

**Breakeven no 4º trimestre do ano 4. Rodada de R$ 30 milhões no mês 46.**

> **A infraestrutura de R$ 1,2 mi já inclui o programa de capacidade** descrito em
> [`04-arquitetura-alvo.md`](04-arquitetura-alvo.md). Células, publicação estática, geração
> assíncrona e fila de espera não são um orçamento à parte esperando aprovação: estão pagos, e
> são o produto.

É esse o gatilho da discussão: com caixa e com 45 bancas dependendo da plataforma, o conselho
perguntou o que acontece se a nuvem cair, e se não deveríamos estar em duas.

> **A resposta deste trabalho é que a pergunta do conselho está mal formulada**, e o documento
> [`03-defesa-multicloud.md`](03-defesa-multicloud.md) explica por quê.
