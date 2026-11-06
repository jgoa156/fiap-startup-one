# 1. A empresa e o contexto

*(Atende ao item "Breve descrição do projeto/empresa e sua área de atuação".)*

---

## 1.1 O que é a Lisura

**Lisura** é a plataforma que conduz um processo seletivo público de ponta a ponta — do edital à
homologação do resultado. Vendida para **bancas organizadoras**, que são as empresas contratadas
por prefeituras, autarquias, universidades e hospitais para realizar concursos e vestibulares.

*O nome vem de "lisura do certame" — o termo que a administração pública usa para a integridade
do processo. É o que o produto protege.*

| | |
|---|---|
| **Setor** | Software B2B para o setor de certames públicos |
| **Cliente** | Banca organizadora de porte médio — 5 a 50 certames por ano |
| **Usuário final** | O candidato, que se inscreve, paga, recorre e consulta resultado |
| **Modelo** | Assinatura anual + **valor por inscrição processada** (o segundo domina) |
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
software de negócio: **a carga do ano cabe em poucas horas, e essas horas têm data marcada em
edital.**

### Os cinco picos de um certame

| # | Pico | Quem acessa | Por que dói |
|---|---|---|---|
| **1** | Encerramento do prazo de **isenção** | Quem pede isenção | Prazo anterior ao da inscrição, já concentrado |
| **2** | Encerramento do prazo de **inscrição** | Quem ainda não se inscreveu | **Escrita pesada** — cadastro, upload, cobrança |
| **3** | Liberação do **cartão de confirmação** | **Toda a base** | Leitura simultânea de todos os inscritos |
| **4** | **Gabarito preliminar** + janela de recurso | **Toda a base** | Leitura em massa seguida de escrita em massa |
| **5** | **Resultado e classificação** | **Toda a base** | O maior de todos |

> **Correção importante sobre o perfil:** o pico de inscrição não é o pior. Nele, só acessa quem
> ainda não se inscreveu. Nos picos 3, 4 e 5 acessa **a base inteira, ao mesmo tempo** — todo
> mundo quer saber onde vai fazer a prova e quanto tirou.

### E o pico é conhecido com meses de antecedência

O cronograma está publicado no edital. **Sabemos a data e a hora em que precisaremos de 40 vezes
a capacidade média.** Isso é raro, e é a característica que organiza toda a decisão de
arquitetura — inclusive a de multicloud.

### A consequência da falha

Existe registro de **prazo de inscrição estendido por decisão judicial** por indisponibilidade do
site. A falha é pública, é notícia, e chega ao Judiciário — mas **é reversível**, e essa
reversibilidade é dado de entrada da decisão de continuidade.

---

## 1.3 Por que agora — a janela regulatória

A **Lei nº 14.965/2024**, sancionada em 9 de setembro de 2024, é a Lei Geral dos Concursos
Públicos. Ela:

- autoriza **prova total ou parcialmente a distância**;
- endurece as regras de escolha e conduta da banca organizadora;
- **entra em vigor em 1º de janeiro de 2028**, com aplicação antecipada permitida se o edital
  expressamente previr.

> **Todo o mercado precisa se replataformar até 2028** — e quem quiser antecipar precisa, já, de
> um sistema que atenda à lei nova. É a janela que justifica o *timing* da empresa.

---

## 1.4 O mercado

| | Faixa | Status |
|---|---|---|
| Inscrições pagas por ano no Brasil | 20 a 40 milhões | ⚠️ não verificado |
| Taxa média paga pelo candidato | ~R$ 80 | ⚠️ âncora: R$ 69,33 documentado |
| Bancas de porte médio | 80 a 200 | ⚠️ não verificado |
| **TAM da camada de plataforma** | **~R$ 90 mi/ano** | Modesto — crescimento por adjacência |

**O mercado está ocupado:** Gestor Editais (Grupo ACT), ProSeleta, Fábrica de Provas, ADSR, MH
Informática, Instituto Avalia. O argumento não é "ninguém faz isso" — é **"todos fazem a
inscrição; quase ninguém sobrevive ao dia do pico e ao recurso em massa"**.

Análise completa em
[`../fiap-startup-one/03-plano-de-negocios.md`](../fiap-startup-one/03-plano-de-negocios.md).

---

## 1.5 Onde a empresa está — a premissa deste trabalho

**A Lisura não é mais uma startup.** O trabalho parte do ano 5:

| Indicador | Ano 1 | **Ano 5** |
|---|---:|---:|
| Bancas clientes | 6 | **72** |
| Inscrições processadas por ano | 0,12 mi | **7,7 mi** |
| Receita (ARR) | R$ 0,6 mi | **R$ 27,4 mi** |
| EBITDA | −R$ 3,8 mi | **+R$ 5,3 mi (19%)** |
| Headcount | 14 | **68** |
| Time de plataforma | 1 | **6** |
| Infraestrutura | R$ 0,2 mi | **R$ 3,2 mi/ano** |

**Breakeven no 4º trimestre do ano 4. Rodada de R$ 80 milhões no mês 46.**

É esse o gatilho da discussão: com caixa e com 72 bancas dependendo da plataforma, o conselho
perguntou o que acontece se a nuvem cair — e se não deveríamos estar em duas.

> **A resposta deste trabalho é que a pergunta do conselho está mal formulada** — e o documento
> [`03-defesa-multicloud.md`](03-defesa-multicloud.md) explica por quê.
