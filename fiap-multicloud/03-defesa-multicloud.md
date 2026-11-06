# 3. A defesa

*(Atende ao item "Defesa da adoção (ou não adoção) de estratégia multicloud, incluindo prós e
contras baseados nos requisitos de negócio, considerando crescimento pós-breakeven ou nova
rodada de investimento".)*

---

## 3.1 A tese

> **Não. A Lisura não deve adotar multicloud** — e a razão não é custo, é que **multicloud não
> resolve a falha que de fato nos derruba.**
>
> O que tira um sistema de inscrição do ar no último dia do prazo **não é o provedor cair.** É a
> nossa própria capacidade acabar. Uma segunda nuvem rodando o mesmo software mal dimensionado
> cai exatamente igual — replicamos o gargalo, não a capacidade.
>
> **Com uma ressalva que adotamos:** DNS, WAF, página de status e backup saem do provedor
> primário. Isso não é arquitetura multicloud — é não ter ponto único de falha no **controle** e
> na **última linha de defesa**.

A pergunta "multicloud, sim ou não?" só tem resposta depois de outra: **qual falha estamos
comprando proteção contra, e ela é a falha que acontece?**

---

## 3.2 O que exatamente estamos comparando

### O espectro de continuidade (taxonomia canônica de DR)

| | Estado na secundária | Tráfego | RTO | RPO |
|---|---|---|---|---|
| **1 · Backup & restore** | Apenas backup | 0% | 12–24 h | horas |
| **2 · Pilot light** | Dados replicados, computação desligada | 0% | 30–60 min | minutos |
| **3 · Warm standby** | Dados replicados, computação reduzida ligada | 0% | 15 min–4 h | 15 min |
| **4 · Multi-site ativo-ativo** | Estado compartilhado | ~50% | ~30 s | ~0 |

**Hoje estamos no nível 1. A proposta mantém o nível 1 — com restauração de fato testada — e
investe o dinheiro em outro lugar.** Os níveis 3 e 4 são rejeitados.

### O que "multicloud" significaria aqui

| Camada | Multicloud faria sentido? |
|---|---|
| Aplicação de inscrição | ❌ Réplica ociosa do mesmo gargalo |
| Banco de dados | ❌ Replicação entre nuvens, custo e latência, para cobrir a falha mais rara |
| **DNS e WAF** | ✅ **Sim** — e é barato |
| **Página de status** | ✅ **Sim** — precisa sobreviver à queda que está anunciando |
| **Backup** | ✅ **Sim** — backup no provedor que você quer sobreviver não é backup |

---

## 3.3 O argumento principal — multicloud protege contra a falha errada

Este é o argumento que decide, e ele não muda com o tamanho do caixa.

### O que realmente derruba um sistema de inscrição

Quando a notícia diz *"site do concurso fora do ar no último dia"*, a causa quase nunca é
indisponibilidade do provedor de nuvem. É uma destas:

| Causa real | Multicloud resolve? |
|---|---|
| Pool de conexões do banco esgotado com 30× a carga normal | ❌ **Não.** A segunda nuvem tem o mesmo pool |
| Geração síncrona de boleto ou cartão consumindo toda a CPU | ❌ **Não.** Mesmo código, mesmo gargalo |
| Consulta sem índice que só aparece com 200 mil linhas | ❌ **Não** |
| Capacidade nunca ensaiada para o pico | ❌ **Não.** Dobrar de nuvem não é ensaiar |
| Implantação ruim na véspera do prazo | ❌ **Não.** Implanta nas duas |
| Um certame gigante degradando os outros inquilinos | ❌ **Não.** Falta isolamento, não provedor |
| **Queda regional total do provedor** | ✅ **Sim** — e é a **última** da lista em frequência |

> **A conclusão desconfortável para quem vende multicloud:** das sete causas que tiram este
> produto do ar, **multicloud cobre uma** — e é justamente a mais rara.

### E a falha aqui é reversível

Existe registro de prazo de inscrição **estendido por decisão judicial** por indisponibilidade do
site. Isso importa para a decisão de arquitetura mais do que parece:

- Não há dinheiro perdido de forma irrecuperável.
- Não há direito extinto: o prazo se prorroga.
- O dano é **reputacional e operacional**, não terminal.

**Não estamos comprando proteção contra dano irreversível.** Estamos comprando redução de
constrangimento e de custo de prorrogação — e isso muda quanto a proteção pode custar.

### E o failover não exercitado é passivo, não ativo

Um caminho de failover entre nuvens que roda **uma vez a cada quatro anos**, operado por um time
de plataforma de seis pessoas, não é uma proteção — é **um modo de falha novo**, que só será
descoberto no dia em que precisar funcionar.

---

## 3.4 O argumento geográfico — e por que ele também não ajuda

A AWS opera **uma região no Brasil** (São Paulo, 3 AZs). O Rio de Janeiro é *Local Zone*, que
depende da região-mãe e não constitui isolamento de falha.

Então "multirregião AWS" também está fora — mas **por um motivo diferente do trabalho de
referência que estudamos**: não é que a segunda região não exista e por isso precisemos de outro
provedor. É que **nem multirregião nem multicloud atacam as sete causas da tabela acima.**

> A restrição geográfica existe — contratos com o setor público exigem dado em território
> nacional — mas ela é **irrelevante para esta decisão**, porque a decisão morre antes, no
> argumento de eficácia.

---

## 3.5 O que mudou com o breakeven e a rodada

| # | Requisito de negócio novo | Consequência |
|---|---|---|
| **R1** | 72 bancas em vez de 6 — **certames simultâneos de bancas diferentes** | Isolamento entre inquilinos vira requisito, não refinamento |
| **R2** | Certames de até 500 mil candidatos | O pico deixa de ser 10× e passa a ser 40× |
| **R3** | Contrato com cláusula de disponibilidade e multa | Disponibilidade vira obrigação contratual mensurável |
| **R4** | **Lei 14.965/2024** — prova a distância, vigência em 2028 | Carga nova: aplicação de prova online, com janela ainda mais rígida |
| **R5** | Exposição a ação judicial por falha | Trilha e capacidade comprovada viram defesa jurídica |
| **R6** | Rodada de R$ 80 mi | Capital deixa de ser a restrição — **e por isso o argumento não pode ser "é caro"** |

---

## 3.6 O que os institutos de mercado dizem

| Dado | Fonte | O que sustenta |
|---|---|---|
| **78%** gastam ou planejam gastar em multicloud | HashiCorp / **Forrester**, *State of Cloud Strategy Survey 2024* (~1.200 respondentes, empresas de 1.000+ funcionários) | Multicloud é o padrão — **o ônus da prova é nosso**, e por isso a defesa precisa ser específica |
| **Apenas 8%** são de alta maturidade em nuvem | HashiCorp / Forrester 2024 | Contra o multicloud ingênuo |
| **64%** não têm pessoal com expertise suficiente | HashiCorp / Forrester 2024 | **Decisivo para um time de plataforma de 6 pessoas** |
| Falta de competências é o maior contribuinte para desperdício; **91%** relatam desperdício | HashiCorp / Forrester 2024 | Duas nuvens amplificam o desperdício de quem já desperdiça |
| **79%** das organizações de alta maturidade dizem que multicloud melhorou a segurança | HashiCorp / Forrester 2024 | **Citado contra nós, honestamente** — o risco é de execução. Só que nós não somos de alta maturidade |
| **27%** do gasto em nuvem é desperdiçado; **84%** apontam custo como maior desafio | Flexera, *State of the Cloud Report 2025* | Governança fragmentada |
| **Mais de 92%** das grandes empresas operam em multicloud | Gartner | Normalização — e é exatamente por isso que dizer não exige argumento forte |

> **Usamos o dado que joga contra nós de propósito.** Se 78% do mercado vai para multicloud e nós
> não vamos, a defesa tem que explicar por que somos o caso diferente — e o caso diferente é a
> tabela das sete causas de §3.3.

---

## 3.7 Os prós — analisados um a um, com veredito

| # | Argumento | Veredito neste negócio |
|---|---|---|
| **P1** | **Descorrelação de risco com o cliente** | ❌ **Não se aplica.** Nossos clientes são bancas, e nossos usuários são candidatos em internet doméstica. Ninguém compartilha nuvem conosco |
| **P2** | **Exigência regulatória de continuidade** | ❌ **Não existe.** Não somos instituição financeira; nenhuma norma nos impõe DR multiprovedor |
| **P3** | **Evitar aprisionamento a fornecedor** | ⚠️ **Superestimado.** AWS e Google removeram a taxa de egresso de saída em 2024. O aprisionamento real é de arquitetura e de competências, e se combate com Terraform |
| **P4** | **Poder de negociação comercial** | ⚠️ **Contraproducente aqui.** Fragmentar o compromisso reduz o desconto justamente no volume de pico, que é nosso maior item |
| **P5** | **Diversidade de fornecedor de IA** | ❌ **Não se aplica** — o produto é deliberadamente sem IA |
| **P6** | **Soberania na expansão internacional** | ❌ **Não se aplica** — não há expansão fora do Brasil no horizonte do plano |
| **P7** | **Arbitragem de custo entre provedores** | ❌ **Inválido.** A carga é interativa e síncrona, não lote interrompível. Não há o que arbitrar |
| **P8** | **Resiliência a falha de provedor** | ✅ **Válido — e é o único.** Mas cobre a última causa da lista de §3.3 |

> **Um pró válido de oito.** Este é o número que a apresentação leva para a tela, porque ele é a
> defesa inteira em um dado.

---

## 3.8 Os contras — e quais deles o capital resolve

| # | Argumento | Peso | Some com capital? |
|---|---|---|---|
| **C1** | **Eficácia.** Multicloud não cobre 6 das 7 causas reais de indisponibilidade | 🔴 **Decisivo** | **Não** |
| **C2** | **Failover não exercitado.** Um caminho que roda a cada 4 anos é fonte de falha | 🔴 **Decisivo** | **Não** — exercitar exige carga real, que só existe no pico |
| **C3** | **Complexidade e talento.** 6 pessoas em plataforma; 64% do mercado sem expertise | 🟡 Alto | Parcialmente — contratar leva de 4 a 6 meses |
| **C4** | **Custo.** R$ 1,66 mi/ano = **31% do EBITDA** | 🟡 Confirmatório | Sim |
| **C5** | **Superfície de segurança.** Dado pessoal de milhões de candidatos, duplicado | 🟡 Alto | Sim, com execução |
| **C6** | **Menor denominador comum.** Perderíamos os serviços gerenciados que permitem operar com time pequeno | 🟡 Médio | Não |
| **C7** | **Governança de custo fragmentada** | 🟢 Baixo | Sim |

> **A última coluna é o coração da defesa.** Com R$ 80 milhões em caixa, C3, C4, C5 e C7
> desaparecem — são problemas de recurso. **C1 e C2 continuam exatamente onde estavam**, porque
> são propriedades do problema, não do orçamento. **E são eles que decidem.**

---

## 3.9 Os três modelos avaliados

| | **M0 · Nuvem única, como está** | **M1 · Nuvem única endurecida** ✅ | **M2 · Multicloud warm standby** |
|---|---|---|---|
| Continuidade | Backup sem restauração testada | **Backup com restauração trimestral testada** | Warm standby em 2º provedor |
| Cobertura de queda regional | ❌ 12–24 h | ⚠️ 6–12 h | ✅ 4 h |
| **Cobertura das 6 causas reais** | ❌ | ✅ **Integral** | ❌ **Nenhuma** |
| Isolamento entre inquilinos | ❌ | ✅ Células | ❌ |
| Capacidade de pico comprovada | ❌ | ✅ Ensaio contratual | ❌ |
| Degradação graciosa | ❌ | ✅ Fila + estático | ❌ |
| Plano de controle neutro | ❌ | ✅ | ✅ |
| Custo/ano | Base | **+R$ 1,69 mi** | **+R$ 1,66 mi** |
| Veredito | Insuficiente | **Recomendado** | **Ineficaz**, antes de ser caro |

**O achado que sustenta a apresentação:** M1 e M2 custam **praticamente o mesmo**. Não é uma
escolha entre proteção e economia — é uma escolha entre **duas proteções pelo mesmo preço**, e
uma delas cobre a falha que acontece.

---

## 3.10 Os números — mesmo dinheiro, proteção diferente

**Linha de base (ano 5):** receita R$ 27,4 mi · EBITDA +R$ 5,3 mi · infraestrutura R$ 3,2 mi/ano.

### M2 — o que a segunda nuvem custaria

| Item | R$/ano |
|---|---:|
| *Landing zone* e replicação de dados entre nuvens | 380 mil |
| Capacidade de standby, com escala no pico | 320 mil |
| Ferramental duplicado — observabilidade, segurança, IaC | 240 mil |
| 2 engenheiros de plataforma adicionais | 720 mil |
| **Total** | **R$ 1,66 mi** — **31% do EBITDA** |

### M1 — o mesmo dinheiro, aplicado no que derruba o sistema

| Investimento | Cobre qual causa | R$/ano |
|---|---|---:|
| **Arquitetura celular por porte de banca** | Certame gigante degradando os outros | 620 mil |
| **Ensaio de carga contratualizado** — 40× provado antes de cada certame grande | Capacidade nunca ensaiada | 280 mil |
| **Escalonamento dirigido pelo calendário do edital** | Reativo chega tarde num pico de 90 s | 240 mil |
| **Fila de espera na borda** | Todos entram e ninguém completa | 180 mil |
| **Publicação estática de gabarito e resultado em CDN** | Pico de leitura derrubando o banco | 90 mil |
| **Geração assíncrona de documento** (boleto, cartão, PDF) | CPU consumida por geração síncrona | 140 mil |
| **Restauração testada trimestralmente** | Backup que nunca foi restaurado | 120 mil |
| **Congelamento de implantação por calendário** | Implantação ruim na véspera | 0 |
| **Plano de controle e backup fora do provedor primário** | Queda regional — *parcialmente* | 162 mil |
| **Total** | | **R$ 1,69 mi** |

> **R$ 1,69 mi contra R$ 1,66 mi.** A diferença não é de preço — é de **o que o dinheiro compra.**

### E o que M2 compraria a mais

Apenas reduzir o RTO de queda regional total de ~8 h para 4 h. Frequência estimada dessa classe
de falha: **1 evento a cada 4 anos**. Probabilidade de coincidir com uma janela crítica de
encerramento de prazo: material, porque com dezenas de certames quase sempre há um prazo
correndo — **mas o remédio existe e é a prorrogação.**

**Duas ressalvas de honestidade intelectual**, melhor ditas do que descobertas pela banca:

1. **A prorrogação não é indolor.** Custa reputação com a banca, e pode custar o contrato. O
   valor esperado subestima o dano de cauda.
2. **Com R$ 80 milhões em caixa, "não podemos pagar" seria um argumento falso** — e não é o
   nosso. **Poderíamos pagar. Não o fazemos porque compraríamos a proteção errada.**

---

## 3.11 A ressalva — o que de fato sai do provedor primário

Rejeitar multicloud como arquitetura não significa aceitar ponto único em tudo.

| O que sai | Por quê | R$/ano |
|---|---|---:|
| **DNS e WAF** (provedor neutro) | Se a nuvem cai e o DNS está dentro dela, **não dá nem para apontar para uma página de status** | 60 mil |
| **Página de status** | Precisa sobreviver à queda que está anunciando. Hospedada fora, sempre | 12 mil |
| **Backup off-site em segundo provedor** | Backup no provedor que você quer sobreviver não é backup — é cópia | 90 mil |
| **Total** | | **R$ 162 mil — 3% do EBITDA** |

> **Isto é multicloud?** Pelo critério estrito do enunciado — usar mais de um provedor — sim,
> parcialmente. E é exatamente por isso que a resposta honesta não é "não" puro, e sim:
> **não à arquitetura multicloud; sim ao controle e ao backup fora do provedor primário.**

---

## 3.12 A decisão

**Rejeitamos M2.** Adotamos **M1**, com o plano de controle neutro de §3.11.

O critério que organizou tudo: **para cada real gasto em continuidade, qual classe de falha ele
cobre, e com que frequência essa classe acontece?** Sob esse critério, multicloud perde não por
ser caro, mas por cobrir a coluna errada da tabela.

---

## 3.13 Gatilhos de reavaliação

A decisão tem validade condicional declarada. Qualquer um destes reabre a discussão:

1. Uma banca com **> 10% da receita** exigir DR multiprovedor contratualmente.
2. **Aplicação de prova online** (Lei 14.965/2024) entrar em produção em escala — a prova
   acontecendo *ao vivo* torna a falha irreversível, e **o argumento de §3.3 cai inteiro**.
3. Incidente real de queda regional com **impacto acima de 8 horas** durante encerramento de prazo.
4. Norma do setor público exigindo continuidade multiprovedor de fornecedor crítico.
5. A AWS anunciar uma **segunda região no Brasil** — barateia tanto o DR que a conta muda.
6. O time de plataforma passar de **15 pessoas** — C3 deixa de morder.

Os gatilhos **2 e 5** são os mais interessantes para a discussão. O **2 é o mais importante do
trabalho inteiro**: no dia em que a prova é aplicada online, a falha deixa de ter remédio — não
se prorroga uma prova em andamento — e a resposta correta passa a ser a oposta desta.

---

## 3.14 Resumo

| Pergunta | Resposta |
|---|---|
| Adotar multicloud? | **Não como arquitetura. Sim para DNS, WAF, status e backup** |
| Por quê? | Cobre 1 das 7 causas reais de indisponibilidade |
| É por custo? | **Não.** M1 e M2 custam o mesmo |
| O capital resolve? | Resolve C3 a C7. **Não resolve C1 e C2** |
| O que muda a decisão? | Prova online em escala — gatilho 2 |
