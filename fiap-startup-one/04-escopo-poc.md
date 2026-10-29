# 4. Escopo da POC: 1 pessoa, 2 meses

*Dimensionado para **1 pessoa em 8 semanas**, construindo com o Claude.*

---

## 1. A regra que organiza o escopo

> **O entregável da POC não é uma tela. É um relatório de ensaio de carga.**

Construir o fluxo de inscrição com o Claude é rápido, e por isso **não é** o resultado. O
resultado é a resposta a uma pergunta que o mercado inteiro evita:

> **H1: o sistema absorve 16,7 mil inscrições por hora, sustentadas por 3 horas, sem perder
> inscrição, sem cobrar duas vezes e sem estourar o custo previsto de infraestrutura.**

### Por que a hipótese é um número absoluto, e não um múltiplo

A formulação anterior era *"40× a carga média"*, e ela **não é testável como escrita**: 40× a
média de quê? A média anual da empresa mistura dezenas de certames em fases diferentes, e 40×
dela seria um alvo muito menor que o pico real. A média de uma POC pequena seria menor ainda, e
provar 40× dela não diria nada sobre um certame de verdade.

> **O múltiplo é uma abstração; o alvo do contrato é um número.**

| | |
|---|---|
| Certame de referência | 300 mil candidatos, janela de inscrição de 30 dias |
| Média daquele certame | 417 inscrições/h |
| **Alvo do ensaio, a 40× a média do certame** | **16,7 mil inscrições/h**, ou 4,6 por segundo |
| Duração sustentada | 3 horas |

**Assim a POC prova algo sobre produção, e não sobre si mesma.** O ambiente da POC é pequeno; o
alvo de carga, não.

É falseável. Passa ou não passa. E **o resultado negativo também é entregável**: *"quebrou em 9 mil por hora,
e o gargalo foi o pool de conexões"* é uma conclusão de engenharia, muito mais útil que uma demo
que sempre funciona.

---

## 2. O que entra e o que fica de fora

| No escopo | Fora |
|---|---|
| Certame com **prova objetiva** apenas | Discursiva, títulos, TAF, prática |
| Inscrição, pagamento simulado, isenção por regra | Ensalamento otimizado, heteroidentificação |
| **Publicação estática de resultado** | Correção real de cartão-resposta |
| **Fila de espera** e geração assíncrona | Painel completo da banca |
| **Ensaio de carga a 16,7 mil inscrições/h** | Multi-tenant completo com células |
| Máquina de estados da inscrição, com trilha | Recurso com agrupamento (só o modelo de dados) |

**O corte é por profundidade, não por largura:** um certame pequeno inteiro, com o caminho crítico
endurecido, porque é o caminho crítico que a hipótese testa.

---

## 3. Cronograma de 8 semanas

| Semana | O quê | Sai disso |
|---|---|---|
| **1** | Modelo de dados do certame · máquina de estados da inscrição · trilha de auditoria | Esquema e migrações |
| **2** | Fluxo de inscrição ponta a ponta · cadastro, cargo, cota, upload | Aplicação funcionando |
| **3** | Pagamento em **sandbox**, com idempotência e conciliação | Nenhuma cobrança duplicada sob repetição |
| **4** | **Geração assíncrona** (boleto, cartão) · **publicação estática** de resultado em CDN | O pico de leitura sai do banco |
| **5** | **Fila de espera** com admissão controlada · modo degradado em 3 níveis | Degradação graciosa demonstrável |
| **6** | Infraestrutura como código · **gerador de carga sintética** (perfil de último dia) | Ambiente reproduzível |
| **7** | **Ensaio de carga** a 4, 8 e 16,7 mil inscrições/h · medição de custo por patamar | Curva de quebra |
| **8** | Relatório, análise do gargalo, e repetição após a correção | **O entregável** |

> **Os 3 níveis de degradação da POC são os níveis 1, 4 e 5** do modo degradado completo descrito
> em [`../fiap-multicloud/04-arquitetura-alvo.md` §4.5](../fiap-multicloud/04-arquitetura-alvo.md):
> fila de espera, só estático e página de status. Os níveis 2 e 3 dependem de módulos que estão
> fora do escopo da POC.

---

## 4. O que precisa existir fora do código

Esta é a lista que decide se a POC é viável, e **está vazia de bloqueios**, que era o critério.

| Dependência | Status |
|---|---|
| Conta AWS | ✅ Já existe. Crédito inicial cobre o uso |
| Sandbox de meio de pagamento | ✅ Público, sem contrato |
| Dado de candidato | ✅ **Sintético**, gerado e nunca real. Evita LGPD por completo |
| Parceiro, banca ou órgão | ✅ **Não é necessário** |
| Hardware ou dispositivo | ✅ Não é necessário |
| GPU | ✅ Não é necessário, porque **não há IA no produto** |

---

## 5. Custo

| Item | Estimativa |
|---|---|
| Ambiente de aplicação, 2 meses, escalado a zero fora dos ensaios | ~US$ 25 |
| Banco gerenciado, instância pequena | ~US$ 30 |
| Armazenamento e CDN | ~US$ 5 |
| **Os ensaios de carga**, 3 h × 3 patamares, capacidade alta e devolvida | ~US$ 40 |
| Gerador de carga (k6 open source) | US$ 0 |
| **Total** | **~US$ 100** |

**O crédito inicial de uma conta AWS nova cobre.** O item mais caro é justamente o que produz o
resultado, e é por isso que o escalonamento por agenda entra já na POC: **subir capacidade, medir
e devolver** é o comportamento que o produto precisa ter em produção.

---

## 6. Métricas do ensaio

| Métrica | Alvo | Por que importa |
|---|---|---|
| **Inscrições perdidas** no alvo | **zero** | É a falha que vira ação judicial |
| **Cobranças duplicadas** sob repetição e reenvio | **zero** | É a falha que vira processo administrativo |
| p95 de resposta no pico | menos de 2 s | Acima disso o candidato recarrega e piora a carga |
| Taxa de conclusão do fluxo completo | acima de 98% | Entrar não basta; tem que concluir |
| **Ponto de quebra** | medido | Se não quebrar em 16,7 mil/h, subir até quebrar. **O número que não se sabe é o perigoso** |
| Custo por hora de pico | medido | Alimenta o preço por inscrição, que é a variável aberta do plano |

---

## 7. Como isso vira material de venda

O relatório da POC é **exatamente o artefato que o produto promete ao cliente**: ensaio de carga
assinado, contra a configuração que vai para produção, anexado ao contrato.

> A POC não é um protótipo que será jogado fora. **É a primeira instância da entrega que
> diferencia a empresa**, e, se H1 falhar, é a primeira instância da honestidade que a diferencia
> também.

---

## 8. O risco da POC

| Risco | Mitigação |
|---|---|
| Carga sintética não parece carga real | Modelar o perfil do último dia: chegada concentrada, recarregamento, abandono e retorno, não uma rampa uniforme |
| "Passou no alvo" sem ter chegado ao limite | **Subir até quebrar, sempre.** Um teste que só passa não informa nada |
| Otimizar para o teste em vez do produto | A configuração do ensaio é **a mesma** que iria para produção. Nenhum ajuste exclusivo do teste |
| 8 semanas para 1 pessoa é apertado | Semanas 1 a 5 são construção acelerada com o Claude. **Se algo cair, cai o painel da banca, nunca o ensaio** |
