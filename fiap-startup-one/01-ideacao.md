# 1. Ideação — SaaS sem IA, com necessidade real de nuvem

*A rodada de ideação que selecionou a Lisura. Critério: SaaS que precise de nuvem **por
construção**, sem IA aplicada, e com POC viável para uma pessoa.*

---

## O filtro, corrigido em dois pontos

### 1. "POC viável" mudou de significado

Se a implementação é feita com o Claude, **escrever código deixa de ser o gargalo.** CRUD,
integração, IaC, teste de carga, simulador de tráfego — tudo isso vira barato. O que continua
caro é o que **não se resolve escrevendo código**:

| Ainda é bloqueio | Deixou de ser bloqueio |
|---|---|
| Credencial de API de terceiro que exige contrato | Volume de código |
| Parceiro que precisa dizer sim | Complexidade de arquitetura |
| Hardware, sensor, dispositivo | Infraestrutura, IaC, pipeline |
| Dado real que não é público | Teste de carga e simulação |
| Homologação regulatória | Frontend, painel, relatório |

**Todas as ideias abaixo foram verificadas contra essa coluna da esquerda.**

### 2. Sem IA, a nuvem precisa ser necessária por outro motivo

Existem seis motivos legítimos, e nenhum deles é "escala" genérica:

| Motivo | O que caracteriza |
|---|---|
| **E · Elasticidade extrema** | A carga do ano cabe em dois dias. Capacidade fixa é ruína ou queda |
| **D · Disponibilidade como produto** | Se cai, o cliente perde dinheiro **naquele minuto** |
| **X · Independência da infraestrutura do cliente** | Só funciona se estiver **fora** do ambiente do cliente |
| **C · Consistência sob concorrência** | Muita gente disputando o mesmo recurso, e ordem importa |
| **B · Lote periódico massivo** | Fecha o mês e processa o mundo; no resto, nada |
| **G · Gravidade e retenção de dado** | Volume, durabilidade e prazo legal de guarda |

---

## As ideias

### 1. Plataforma de inscrição para processos seletivos ⭐ recomendada

Concurso público, vestibular, residência médica, processo seletivo de rede de ensino. A
instituição publica o edital; a plataforma cuida de inscrição, isenção, pagamento, recurso,
convocação e resultado.

| Filtro | Verificação |
|---|---|
| **Necessidade real** | ✅ Não precisa ser argumentada. **Sistema de inscrição fora do ar no último dia de prazo é notícia todo ano no Brasil.** A banca já viveu isso |
| **Por que nuvem** | **E, no estado mais puro que existe.** Algo em torno de 80 a 90% das inscrições chegam nas últimas 48 horas do prazo, e boa parte nas últimas 3. **Fora dessas horas, o sistema não faz nada.** Nenhum outro perfil de carga é tão extremo |
| **Rentabilidade** | ✅ Taxa por candidato. Um concurso grande passa de 500 mil inscritos; a alguns reais por inscrição, **um único evento paga um ano de operação.** Receita concentrada e previsível pelo calendário de editais |
| **POC com Claude** | ✅ **Excelente.** Construir o fluxo é trivial; o entregável de verdade é o **teste de carga** — provar que absorve 40× a média sem perder inscrição nem cobrar duas vezes. Único externo: sandbox de pagamento, que é público |
| **Incumbentes** | ⚠️ Cebraspe, FGV, Vunesp, IBFC e Quadrix operam sistema próprio. Existe espaço em banca média e em rede privada de ensino — **verificar antes de fechar** |
| **Debate multicloud** | ⭐ **O melhor de todos.** *"Pagamos uma segunda nuvem para 48 horas por ano?"* — e a resposta é defensável nos dois sentidos, com números |

**A tensão arquitetural que sustenta o trabalho inteiro:** o sistema precisa ser **impecável
durante dois dias e ocioso no resto do ano**. Isso é o argumento de nuvem em estado puro, e torna
a pergunta de continuidade genuinamente difícil — porque a janela em que a falha importa é curta,
conhecida com antecedência e absolutamente inegociável.

---

### 2. Leilão online ao vivo ⭐ alternativa

Judicial, veículos, imóveis, agropecuário. A casa de leilão publica o lote; a plataforma conduz o
pregão ao vivo, com lance, prorrogação automática e homologação.

| Filtro | Verificação |
|---|---|
| **Necessidade real** | ✅ Mercado grande no Brasil, e a tecnologia da maioria das casas é ruim |
| **Por que nuvem** | **E + C juntos.** O *sniping* concentra lances nos últimos segundos, e **a ordem dos lances é juridicamente relevante** — não dá para perder nem reordenar. Fan-out em tempo real para todos os assistentes |
| **Rentabilidade** | ✅ Comissão sobre valor arrematado, ou SaaS para a casa. Ticket alto |
| **POC com Claude** | ✅ **Excelente, e com hipótese falseável:** *sob 10 mil lances concorrentes nos últimos 5 segundos, nenhum lance é perdido, invertido ou duplicado, e todos os espectadores veem o mesmo estado.* Isso se **prova**, não se demonstra |
| **Incumbentes** | ⚠️ Superbid, Sold, Freitas e outros. Concentrado no topo, fragmentado embaixo |
| **Debate multicloud** | ✅ Bom — mas o argumento de consistência é o mais batido em defesa de multicloud |

---

### 3. Monitoramento de disponibilidade de serviços críticos brasileiros

SEFAZ de cada estado, Pix/SPI, Receita, Correios, bureaus, adquirentes. Um painel neutro que
responde à pergunta que todo time de ERP e fintech faz várias vezes por semana: **"caiu para todo
mundo ou é só a gente?"**

| Filtro | Verificação |
|---|---|
| **Necessidade real** | ✅ Instabilidade de SEFAZ é rotina e para faturamento de empresa inteira. Hoje se descobre por grupo de WhatsApp |
| **Por que nuvem** | **X + D.** Precisa estar **fora** da infraestrutura de quem monitora — um monitor hospedado junto do cliente cai junto com ele. E precisa medir de vários pontos para distinguir queda do serviço de problema de rede do cliente |
| **Rentabilidade** | ⚠️ **É o furo.** Assinatura de R$ 500 a 5 mil por mês, jogo de volume. Risco real de ser **funcionalidade, não empresa** |
| **POC com Claude** | ✅ **Excelente.** SEFAZ publica webservice de status; o resto é sondagem, histórico e alerta. Nenhuma credencial de terceiro |
| **Incumbentes** | ✅ Pouco atendido de forma neutra |

---

### 4. Auditoria de fatura de frete e conciliação de transporte

Embarcador recebe fatura da transportadora e paga sem conferir contra a tabela contratada, o peso
cubado e o prazo. A diferença encontrada é dinheiro de volta.

| Filtro | Verificação |
|---|---|
| **Necessidade real** | ✅ Conferência manual ou inexistente na maioria das empresas médias |
| **Por que nuvem** | **B.** Lote pesado no fechamento, ocioso no resto. Menos dramático que a #1 |
| **Rentabilidade** | ✅ **O melhor modelo da lista:** percentual do que se recupera. O cliente não decide se "vale a pena" — ele paga com o dinheiro achado |
| **POC com Claude** | ✅ Boa. Faturas sintéticas e tabela de frete; o motor de divergência é regra, não modelo |
| **Incumbentes** | ⚠️ Existem, mas o mercado médio é mal atendido |

---

### 5. Conciliação de meios de pagamento

Casar vendas do ERP com a liquidação de adquirentes, Pix e boleto; achar taxa cobrada a maior,
antecipação errada e venda não liquidada.

| Filtro | Verificação |
|---|---|
| **Necessidade real** | ✅ Alta, e cresce com o número de meios de pagamento |
| **Por que nuvem** | **B + G.** Arquivos diários grandes, anos de retenção |
| **Rentabilidade** | ✅ Boa, por volume transacionado |
| **POC com Claude** | ✅ Boa, com arquivos sintéticos |
| **Incumbentes** | 🔴 **Concorrido.** Concil e outros já ocupam. Diferenciação é o problema |

---

### 6. Monitoramento contínuo de regularidade de fornecedor

CND, FGTS, trabalhista, CADIN, sanções, situação cadastral — verificados **continuamente** para
toda a base de fornecedores, com alerta quando alguém perde a regularidade.

| Filtro | Verificação |
|---|---|
| **Necessidade real** | ✅ Hoje é planilha e conferência manual na assinatura do contrato. Fornecedor irregular é exposição jurídica |
| **Por que nuvem** | **X + E.** Consulta contínua em escala, fora do ambiente do cliente |
| **Rentabilidade** | ✅ Por fornecedor monitorado. Empresa com 5 mil fornecedores paga bem |
| **POC com Claude** | 🔴 **Aqui está o bloqueio.** Depende de raspagem de portais públicos, com captcha e mudança de layout. **É frágil por natureza e não se resolve escrevendo código melhor** |
| **Incumbentes** | ⚠️ Linkana e afins |

---

### 7. Arquivo e retenção de documentos fiscais eletrônicos

Guardar, indexar e disponibilizar XML de NF-e e NFS-e pelo prazo legal.

| Filtro | Verificação |
|---|---|
| **Por que nuvem** | **G.** Durabilidade, camadas de armazenamento, retenção de 5 anos |
| **Rentabilidade** | ⚠️ Commodity. Preço por documento cai sempre |
| **POC com Claude** | ✅ Fácil demais — e é sintoma: **fácil de construir costuma significar fácil de copiar** |
| **Incumbentes** | 🔴 Arquivei e outros já resolveram |

---

### 8. Monitoramento sintético de jornada de pagamento

Robô que compra de verdade no seu e-commerce, de vários pontos, várias vezes por hora, e avisa
quando o checkout quebra — antes do cliente reclamar.

| Filtro | Verificação |
|---|---|
| **Por que nuvem** | **X**, e de forma incontestável: um monitor dentro da sua infraestrutura não detecta a queda da sua infraestrutura |
| **Rentabilidade** | ⚠️ Média |
| **POC com Claude** | ✅ Muito boa |
| **Incumbentes** | 🔴 Datadog, Checkly, Pingdom. **O nicho brasileiro — Pix, boleto, antifraude — é o único ângulo** |

---

### 9. Sincronização de repertório para palco *(continuidade com o tema de música)*

Cifra, setlist, ordem e anotação sincronizadas entre todos os músicos ao vivo — funcionando com
o wi-fi ruim que toda casa de show tem.

| Filtro | Verificação |
|---|---|
| **Necessidade real** | ✅ Equipe de louvor e banda cover, mercado grande no Brasil |
| **Por que nuvem** | ⚠️ Fan-out em tempo real e resolução de conflito — **mas o requisito de verdade é funcionar offline**, o que reduz o papel da nuvem |
| **Rentabilidade** | 🔴 B2C de músico. Disposição a pagar baixa |
| **POC com Claude** | ✅ Boa |
| **Incumbentes** | 🔴 OnSong, Cifra Club, Planning Center |

---

## Consolidado

*(POC vale dobrado; "nuvem" mede necessidade estrutural, não conveniência)*

| # | Ideia | POC×2 | Nuvem | Mercado | Rentab. | Difer. | **Σ** |
|---|---|:-:|:-:|:-:|:-:|:-:|:-:|
| **1** | **Inscrição em processos seletivos** | 10 | 5 | 4 | 5 | 3 | **27** |
| 2 | Leilão online ao vivo | 10 | 5 | 4 | 4 | 3 | **26** |
| 4 | Auditoria de fatura de frete | 8 | 3 | 4 | 5 | 4 | **24** |
| 3 | Disponibilidade de serviços BR | 10 | 5 | 3 | 2 | 4 | **24** |
| 5 | Conciliação de pagamentos | 8 | 4 | 4 | 4 | 2 | **22** |
| 8 | Monitoramento sintético | 8 | 5 | 3 | 3 | 2 | **21** |
| 7 | Retenção de documento fiscal | 8 | 4 | 3 | 2 | 1 | **18** |
| 9 | Repertório de palco | 8 | 2 | 3 | 2 | 2 | **17** |
| 6 | Regularidade de fornecedor | 4 | 4 | 4 | 4 | 3 | **19** ⚠️ POC frágil |

---

## Recomendação: **#1 — inscrição em processos seletivos**

Três motivos, e o terceiro é o que decide.

**A necessidade de nuvem não precisa ser argumentada — precisa só ser lembrada.** Sistema de
inscrição caindo no último dia de prazo é notícia recorrente no Brasil. Toda a plateia já tentou
se inscrever em alguma coisa às 23h do último dia. Você não gasta um segundo do pitch convencendo
ninguém de que o problema existe — e **esse é o custo escondido de quase toda ideia de startup**:
metade do pitch vai embora só provando que a dor é real.

**A rentabilidade é aritmética, não projeção.** Taxa por candidato, um edital grande passa de 500
mil inscritos, o calendário de editais é público. Não depende de adoção viral nem de efeito de
rede — depende de ganhar contrato de banca.

**E a POC é a única da lista cujo entregável principal é uma prova, não uma tela.** Construir o
fluxo com o Claude é rápido; o resultado que importa é o **teste de carga**:

> **H1 — o sistema absorve 40 vezes a carga média por três horas, sem perder inscrição, sem
> cobrar duas vezes e sem exceder o custo previsto de infraestrutura.**

Isso se mede. Passa ou não passa. E o resultado negativo — "quebrou em 22×, e o gargalo foi este"
— também é entregável, o que é exatamente a qualidade que faltava nas ideias das rodadas
anteriores.

### E o trabalho de multicloud fica genuinamente interessante

A pergunta vira: **vale uma segunda nuvem para 48 horas por ano?**

| Defesa | Argumento |
|---|---|
| **Sim** | A janela é curta, conhecida e inegociável. Cair no último dia do prazo é falha irrecuperável — candidato perde o direito de concorrer, e há consequência jurídica, não só reputacional. O custo de 48 horas de espera quente é pequeno perto disso |
| **Não** | Fora da janela o sistema é ocioso; multi-AZ com autoescalonamento pré-aquecido resolve, e a complexidade de dois provedores custa mais em pessoal do que compra em disponibilidade. O dinheiro rende mais em ensaio de carga e degradação graciosa |

**As duas se sustentam com números**, o que é exatamente o que o enunciado pede — e o argumento é
incomum, porque aqui **a restrição é temporal, não geográfica.**

### Alternativa: **#2, leilão ao vivo**

Se preferir um problema onde a corretude sob concorrência é o coração — a ordem dos lances tem
consequência jurídica, e isso se prova sob carga. Ligeiramente mais concorrido.

---

## O que verificar antes de fechar

1. **Quem já vende sistema de inscrição para banca média** e para rede privada de ensino. Foi a
   ausência desse levantamento que faz uma ideia inteira colidir com a de outra equipe.
2. **Ordem de grandeza da taxa por candidato** praticada no mercado.
3. **Concentração real da curva de inscrição** nas últimas 48 h — número que sustenta o slide
   inteiro e precisa de fonte.
4. **Sandbox de meio de pagamento** disponível sem contrato, para a POC.
