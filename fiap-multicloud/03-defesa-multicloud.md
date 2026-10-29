# 3. A defesa

*(Atende ao item "Defesa da adoção (ou não adoção) de estratégia multicloud, incluindo prós e
contras baseados nos requisitos de negócio, considerando crescimento pós-breakeven ou nova
rodada de investimento".)*

---

## 3.1 A tese

> **Uma nuvem para operar, duas para sobreviver.**
>
> **A Certamis não deve operar em duas nuvens.** A razão não é custo. É que **uma segunda nuvem
> operando não resolve a falha que de fato nos derruba.** O que tira um sistema de inscrição do
> ar no último dia do prazo **não é o provedor cair.** É a nossa própria capacidade acabar. Uma
> segunda nuvem rodando o mesmo software mal dimensionado cai exatamente igual: replicamos o
> gargalo, não a capacidade.
>
> **E deve sobreviver em duas.** DNS, WAF, página de status, backup e a restauração de emergência
> ficam fora da AWS. Se a região inteira cair, o sistema volta no **Google Cloud, em São Paulo**,
> em cerca de 8 horas. Lá nada fica ligado: é uma segunda nuvem **a frio**, isto é, desligada até
> ser necessária, que só existe para sobreviver à única falha da tabela que não é nossa.

A pergunta "multicloud, sim ou não?" só tem resposta depois de outra: **qual falha estamos
comprando proteção contra, e ela é a falha que acontece?**

---

## 3.2 Como a pergunta é formulada aqui, e por que isso importa

> **O endurecimento da nuvem única já está feito e já está no orçamento.** Ele não é uma
> alternativa à segunda nuvem. É o piso.

Esta é a primeira coisa a esclarecer, porque a formulação errada da pergunta produz uma
comparação errada.

O programa de capacidade descrito em [`04-arquitetura-alvo.md`](04-arquitetura-alvo.md) custa
**R$ 900 mil/ano** e está dentro das linhas de infraestrutura e de P&D do
[plano de negócios](../fiap-startup-one/03-plano-de-negocios.md). Células, publicação estática,
geração assíncrona, fila de espera, ensaio de carga e restauração testada não são melhorias
opcionais: são o produto que vendemos.

**Ninguém escolheria uma segunda nuvem *em vez* de consertar o pool de conexões.** Quem fizesse
isso teria dois ambientes caindo pelo mesmo motivo. Portanto a pergunta do conselho não é "A ou
B". É:

> **Sobre o programa que já existe, vale acrescentar uma segunda nuvem sempre ligada?**

*O programa já inclui uma segunda nuvem desligada, só para restaurar (§3.10). A pergunta é se vale
operar nela.* Formulada assim, a decisão vira aritmética de custo marginal, e é o que §3.10 faz.

### O espectro de continuidade (taxonomia canônica de DR)

| | Estado na secundária | Tráfego | RTO | RPO |
|---|---|---|---|---|
| **1 · Backup & restore** | Apenas backup | 0% | 12 a 24 h | horas |
| **2 · Pilot light** | Dados replicados, computação desligada | 0% | 30 a 60 min | minutos |
| **3 · Warm standby** | Dados replicados, computação reduzida ligada | 0% | 15 min a 4 h | 15 min |
| **4 · Multi-site ativo-ativo** | Estado compartilhado | ~50% | ~30 s | ~0 |

**Ficamos no nível 1, com restauração de fato testada, e testada num segundo provedor.** O backup
fica no Google Cloud em São Paulo, e é lá que o sistema volta se a AWS cair. A faixa típica do
nível 1 é de 12 a 24 horas; a infraestrutura pronta para subir e o ensaio trimestral são o que nos
põem em 6 a 12. O nível 2 exigiria
replicação contínua entre nuvens, que é a parte cara do M2. Os níveis 3 e 4 são rejeitados.

### O que "multicloud" significaria aqui

| Camada | Multicloud faria sentido? |
|---|---|
| Aplicação de inscrição | ❌ Réplica ociosa do mesmo gargalo |
| Banco de dados | ❌ Replicação contínua entre nuvens, custo e latência, para cobrir a falha mais rara. O backup basta |
| **DNS e WAF** | ✅ **Sim**, e é barato |
| **Página de status** | ✅ **Sim.** Precisa sobreviver à queda que está anunciando |
| **Backup** | ✅ **Sim.** Backup no provedor que você quer sobreviver não é backup |
| **Restauração de emergência** | ✅ **Sim, a frio.** Nada fica ligado; o custo é o ensaio trimestral |

---

## 3.3 O argumento principal: multicloud protege contra a falha errada

Este é o argumento que decide, e ele não muda com o tamanho do caixa.

### O que realmente derruba um sistema de inscrição

Quando a notícia diz *"site do concurso fora do ar no último dia"*, a causa quase nunca é
indisponibilidade do provedor de nuvem. É uma destas:

| Causa | Dívida de origem | Multicloud resolve? |
|---|---|---|
| Pool de conexões do banco esgotado no pico | **D2** banco único, sem réplica de leitura | ❌ **Não.** A segunda nuvem tem o mesmo pool |
| Geração síncrona de boleto ou cartão consumindo toda a CPU | **D3** geração de PDF no request | ❌ **Não.** Mesmo código, mesmo gargalo |
| Todos entram, todos degradam, ninguém completa | **D4** sem fila de espera | ❌ **Não.** Falta admissão controlada |
| Capacidade nunca ensaiada para o pico | **D1** sem ensaio de carga | ❌ **Não.** Dobrar de nuvem não é ensaiar |
| Escalonamento reativo chegando tarde | **D8** autoescalonamento por métrica | ❌ **Não.** Chega tarde nas duas |
| Um certame gigante degradando os outros inquilinos | **D7** inquilino compartilhado | ❌ **Não.** Falta isolamento, não provedor |
| Ataque à aplicação: invasão ou negação de serviço | **D9** sem proteção de borda | ❌ **Não.** A segunda nuvem expõe a mesma aplicação |
| **Queda regional total do provedor** | nenhuma; é do fornecedor | ✅ **Sim**, e é a única |

> **De onde vem esta tabela, dito antes que perguntem.** Ela **não é uma estatística de mercado**,
> e não a apresentamos como tal. É a lista dos modos de falha **deste produto**, e cada linha
> corresponde a uma dívida técnica levantada e documentada em
> [`02-arquitetura-fase-startup.md` §2.3](02-arquitetura-fase-startup.md), antes de existir
> qualquer discussão sobre multicloud. **O que sustenta o argumento não é a contagem, é a
> origem:** sete destas oito falhas nascem de decisões nossas, e nenhuma delas muda de natureza
> por existir um segundo provedor.

> **A oitava linha foi acrescentada em 22/09/2026**, depois do caso FGV (abaixo). A arquitetura da
> fase startup não tinha WAF nem proteção contra negação de serviço, e essa lacuna virou a dívida
> D9 em §2.3. A proteção contra ela já estava em §3.11: WAF na borda, fora da AWS, e backup
> imutável no Google Cloud, fora do alcance de quem invadir a conta AWS.

### E os casos reais de 2026 confirmam a tabela

Nos três casos de falha em certame documentados em 2026 que encontramos, **nenhum foi queda do
provedor de nuvem**:

| Caso | Quando | O que falhou, segundo a fonte | Linha da tabela |
|---|---|---|---|
| **Alece** (banca Idecan) | mai e set/2026 | Instabilidade do sistema da banca, na isenção e depois no acesso ao resultado. Causa técnica não divulgada | Sistema da própria banca |
| **AgSUS** (Cebraspe), prova on-line remota | fev e mar/2026 | "Mensagens de erro do sistema, falha na conexão e restrição de acesso ao ambiente da prova, provocadas por mecanismos de controle de acesso do sistema de segurança" (retificação oficial). Uma aplicação foi anulada e remarcada | Configuração da própria banca |
| **FGV** | fev/2026 | Site fora do ar por dias após ataque hacker, afetando o CNU e inscrições | D9, ataque |

> **Três casos não são estatística, e não os apresentamos assim.** Mas são tudo o que a realidade
> recente mostra, e nenhum dos três teria sido evitado por uma segunda nuvem. Fontes em
> [`06-referencias.md` §6.4](06-referencias.md).

### E a falha tem remédio barato, embora não garantido

Nas fases que hoje concentram a carga, a falha de sistema tem um remédio praticado no setor:
reabrir ou prorrogar o prazo. O caso mais recente e documentado, o da **Alece em 2026**, mostra os
dois lados desse remédio:

- **Ele funcionou na fase de recurso.** Depois de falhas no acesso ao resultado preliminar, a
  banca reabriu o prazo de recurso por dois dias, sob cobrança formal da Assembleia, sem ação
  judicial.
- **Ele foi negado na fase de isenção.** O juiz da 13ª Vara da Fazenda Pública de Fortaleza
  decidiu "indeferir, por ora, a reabertura geral do período de isenção aos demais candidatos,
  inclusive, com fundamento nas alegadas instabilidades do sistema eletrônico".

Isso importa para a decisão de arquitetura:

- O remédio existe e é barato: reabrir um prazo custa dias, não o certame.
- **Mas não é garantido.** Depende da banca ou do juiz, e o candidato pode perder algo no
  caminho, como a isenção da taxa.
- O dano típico é **reputacional, operacional e às vezes financeiro** para o candidato.
  Raramente é a perda do direito de concorrer, porque isenção e recurso são fases, não o certame.

**Não estamos comprando proteção contra dano sem remédio.** Estamos comprando redução de
constrangimento, de custo de prorrogação e do risco de o remédio ser negado, e isso muda quanto a
proteção pode custar.

> **A correção desta premissa, dita antes que perguntem.** Versões anteriores deste trabalho
> afirmavam que "ninguém perde o direito de concorrer" e que "há registro de prazo estendido por
> decisão judicial". A primeira frase era forte demais, e a segunda não tinha caso nomeado. O caso
> Alece, lido na fonte, mostra um remédio praticado **e** um remédio negado. A premissa sobrevive
> mais modesta: **a falha tem remédio barato na maioria das vezes, não sempre.** O argumento
> principal desta defesa, o de eficácia, não depende dela.

> **O limite desta premissa.** Ela vale para isenção, inscrição, cartão, gabarito e resultado.
> **Numa prova sendo aplicada on-line, o remédio muda de ordem de grandeza:** deixa de ser reabrir
> um prazo e passa a ser anular e reaplicar a prova. Isso já aconteceu, na AgSUS em 2026. Esse é o
> gatilho 2 de §3.13.

### E o failover não exercitado é passivo, não ativo

Um caminho de failover entre nuvens que roda **uma vez a cada quatro anos**, operado por um time
de plataforma de cinco pessoas, não é uma proteção. É **um modo de falha novo**, que só será
descoberto no dia em que precisar funcionar. O detalhamento deste argumento, e o que ele **não**
afirma, está em C2 de §3.8. **A restauração a frio que adotamos não é esse caminho:** ela é o
próprio teste trimestral, executado quatro vezes por ano.

---

## 3.4 O argumento geográfico, e por que ele também não ajuda

A AWS opera **uma região no Brasil** (São Paulo, 3 AZs). O Rio de Janeiro é *Local Zone*, que
depende da região-mãe e não constitui isolamento de falha.

Então "multirregião AWS" também está fora, mas **por um motivo diferente do trabalho de
referência que estudamos**: não é que a segunda região não exista e por isso precisemos de outro
provedor. É que **nem multirregião nem multicloud atacam as sete primeiras falhas da tabela.**

> A restrição geográfica **não decide se vale operar em duas nuvens**: essa decisão morre antes,
> no argumento de eficácia. **Mas ela pesa na escolha do destino da restauração** (§3.10).
> Contratos com o setor público costumam exigir dado em território nacional, e essa é uma das
> razões para restaurar no Google Cloud em São Paulo, e não numa região da AWS no exterior. A
> exigência não foi verificada em contrato específico, e está declarada em §6.8.

---

## 3.5 O que mudou com o breakeven e a rodada

| # | Requisito de negócio novo | Consequência |
|---|---|---|
| **R1** | 45 bancas em vez de 3, com **certames simultâneos de bancas diferentes** | Isolamento entre inquilinos vira requisito, não refinamento |
| **R2** | Certames de até 300 mil candidatos | O pico de um certame grande chega a 40× a média daquele certame |
| **R3** | Contrato com cláusula de disponibilidade e multa | Disponibilidade vira obrigação contratual mensurável |
| **R4** | **Lei 14.965/2024**: normas gerais para concursos de todos os entes a partir de 2028, e prova a distância autorizada. **Prova on-line já praticada em 2026** (AgSUS) | Carga nova possível: aplicação de prova on-line, onde o remédio para falha é reaplicar a prova |
| **R5** | Exposição a ação judicial por falha | Trilha e capacidade comprovada viram defesa jurídica |
| **R6** | Rodada de R$ 30 mi | Capital deixa de ser a restrição, **e por isso o argumento não pode ser "é caro"** |

---

## 3.6 O que os institutos de mercado dizem

| Dado | Fonte | O que sustenta |
|---|---|---|
| **78%** gastam ou planejam gastar em multicloud | HashiCorp / **Forrester**, *State of Cloud Strategy Survey 2024* (~1.200 respondentes, empresas de 1.000+ funcionários) | Multicloud é o padrão, **o ônus da prova é nosso**, e por isso a defesa precisa ser específica |
| **Apenas 8%** são de alta maturidade em nuvem | HashiCorp / Forrester 2024 | Contra o multicloud ingênuo |
| **64%** não têm pessoal com expertise suficiente | HashiCorp / Forrester 2024 | **Decisivo para um time de plataforma de 5 pessoas** |
| Falta de competências é o maior contribuinte para desperdício; **91%** relatam desperdício | HashiCorp / Forrester 2024 | Duas nuvens amplificam o desperdício de quem já desperdiça |
| **79%** das organizações de alta maturidade dizem que multicloud melhorou a segurança | HashiCorp / Forrester 2024 | **Citado contra nós, honestamente.** O risco é de execução, e nós não somos de alta maturidade |
| **27%** do gasto em nuvem é desperdiçado; **84%** apontam custo como maior desafio | Flexera, *State of the Cloud Report 2025* | Governança fragmentada |
| **Mais de 92%** das grandes empresas operam em multicloud | Gartner | Normalização, e é exatamente por isso que dizer não exige argumento forte |
| Previsão de **pelo menos dois grandes incidentes de nuvem de vários dias** em 2026 | Forrester, *Predictions 2026* (Lee Sustar, nov/2025) | **Citado contra nós.** Reforça P8 e pressiona para cima a frequência estimada em §3.10. A réplica está em §3.10 |

> **Usamos o dado que joga contra nós de propósito.** Se 78% do mercado vai para multicloud e nós
> não vamos, a defesa tem que explicar por que somos o caso diferente, e o caso diferente é a
> tabela de §3.3.

---

## 3.7 Os prós, analisados um a um, com veredito

| # | Argumento | Veredito neste negócio |
|---|---|---|
| **P1** | **Descorrelação de risco com o cliente** | ⚠️ **Quase não se aplica.** Nossos usuários são candidatos em internet doméstica, e não compartilham nuvem conosco. A ressalva honesta é que a **banca** pode ter os próprios sistemas na AWS, e nesse caso uma queda regional derruba os dois lados ao mesmo tempo. Isso agrava a comunicação de crise, não a disponibilidade da inscrição |
| **P2** | **Exigência regulatória de continuidade** | ❌ **Não existe.** Não somos instituição financeira; nenhuma norma nos impõe DR multiprovedor |
| **P3** | **Evitar aprisionamento a fornecedor** | ⚠️ **Superestimado.** AWS e Google passaram a **dispensar a taxa de egresso para quem migra e encerra o uso**, em prazo definido, o que não é o mesmo que egresso livre no dia a dia. O aprisionamento real é de arquitetura e de competências, e se combate com Terraform |
| **P4** | **Poder de negociação comercial** | ⚠️ **Contraproducente aqui.** Fragmentar o compromisso reduz o desconto justamente no volume de pico, que é nosso maior item |
| **P5** | **Diversidade de fornecedor de IA** | ❌ **Não se aplica**, porque o produto é deliberadamente sem IA |
| **P6** | **Soberania na expansão internacional** | ❌ **Não se aplica**, porque não há expansão fora do Brasil no horizonte do plano |
| **P7** | **Arbitragem de custo entre provedores** | ❌ **Inválido.** A carga é interativa e síncrona, não lote interrompível. Não há o que arbitrar |
| **P8** | **Resiliência a falha de provedor** | ✅ **Válido, e é o único.** Cobre só a última linha da tabela de §3.3, e é por isso que o compramos a frio, pelo menor preço (§3.10) |

> **Um pró plenamente válido em oito.** O que a apresentação leva para a tela não é o placar, é
> **qual** pró sobrevive: exatamente o que corresponde à única falha que não é nossa.

---

## 3.8 Os contras, e quais deles o capital resolve

| # | Argumento | Peso | Some com capital? |
|---|---|---|---|
| **C1** | **Eficácia.** Multicloud não cobre 7 das 8 falhas da tabela, e as 7 têm origem documentada em dívida nossa | 🔴 **Decisivo** | **Não** |
| **C2** | **Memória operacional.** Um caminho que roda a cada 4 anos, num time de 5 pessoas, não cria competência | 🔴 **Decisivo** | **Não**, e a explicação está abaixo |
| **C3** | **Complexidade e talento.** 5 pessoas em plataforma; 64% do mercado sem expertise | 🟡 Alto | Parcialmente; contratar leva de 4 a 6 meses |
| **C4** | **Custo.** R$ 1,22 mi/ano, ou **36% do EBITDA** | 🟡 Confirmatório | Sim |
| **C5** | **Superfície de segurança.** Dado pessoal de milhões de candidatos, duplicado | 🟡 Alto | Sim, com execução |
| **C6** | **Menor denominador comum.** Perderíamos os serviços gerenciados que permitem operar com time pequeno | 🟡 Médio | Não |
| **C7** | **Governança de custo fragmentada** | 🟢 Baixo | Sim |

### C2 em detalhe, porque o argumento óbvio está errado

Seria fácil dizer que o failover não pode ser exercitado porque exige carga real. **Isso seria
falso, e a nossa própria proposta prova que é falso:** o programa M1 inclui um gerador de carga
sintética e ensaio antes de cada certame grande. A ferramenta para exercitar existe, e é nossa.

**O argumento correto é outro.** O que não se compra com ferramenta é **memória operacional**.
Num time de cinco pessoas, um procedimento executado uma vez a cada quatro anos:

- é escrito por alguém que provavelmente não estará lá quando for usado;
- compete por atenção com o calendário de editais, que nunca abre janela;
- envelhece em silêncio a cada mudança de esquema, de IAM e de rede;
- e será executado sob pressão, às duas da manhã, no dia em que o prazo fecha.

**Ensaio sintético prova que o caminho funcionava.** Não prova que as cinco pessoas de plantão
sabem operá-lo. É essa diferença que o capital não compra, e é por isso que C2 continua decisivo.

**E por que a restauração a frio no segundo provedor escapa deste argumento.** Ela não é um
caminho que roda uma vez a cada quatro anos: é o próprio teste trimestral de restauração (D5),
feito fora da produção, sem tocar tráfego real e sem disputar janela com o calendário de editais.
O plantão a executa quatro vezes por ano. O failover da segunda nuvem sempre ligada, o M2, é
outra coisa: desviar tráfego vivo para outra nuvem, e isso ninguém ensaia em produção no meio de
um prazo aberto.

**E por que ela também não cai no C6.** A operação continua usando os serviços gerenciados da AWS,
sem abrir mão de nada. Só o modo de recuperação usa o denominador comum, Kubernetes e PostgreSQL,
e ele é mais lento de propósito: existe para voltar ao ar, não para operar bem.

> **A última coluna é o coração da defesa.** Com a rodada em caixa, C3, C4, C5 e C7 desaparecem,
> porque são problemas de recurso. **C1 e C2 continuam exatamente onde estavam**, porque são
> propriedades do problema, não do orçamento. **E são eles que decidem.**

---

## 3.9 Os três modelos avaliados

| | **M0 · Como estávamos** | **M1 · Nuvem única endurecida** ✅ | **M1 + M2 · com segunda nuvem sempre ligada** |
|---|---|---|---|
| Continuidade | Backup sem restauração testada | **Backup restaurado a frio no 2º provedor, ensaiado todo trimestre** | Warm standby em 2º provedor |
| Cobertura de queda regional | ❌ só quando a AWS voltar | 🟡 6 a 12 h, voltando no 2º provedor | ✅ 4 h |
| **Cobertura das 7 falhas nossas** | ❌ | ✅ **Integral** | ✅ Integral, **herdada de M1** |
| Isolamento entre inquilinos | ❌ | ✅ Células | ✅ herdado |
| Capacidade de pico comprovada | ❌ | ✅ Ensaio contratual | ✅ herdado |
| Degradação graciosa | ❌ | ✅ Fila e estático | ✅ herdado |
| Controle, backup e restauração fora da AWS | ❌ | ✅ | ✅ |
| Custo/ano | Base | **R$ 900 mil** | **R$ 2,12 mi** |
| Veredito | Insuficiente | **Adotado** | **Rejeitado** |

> **Note a coluna da direita.** M2 não substitui nada de M1: ele **herda** tudo, porque sem M1 a
> segunda nuvem cai pelo mesmo motivo que a primeira. É por isso que a comparação correta é
> marginal, e não lado a lado.

---

## 3.10 A conta, feita na margem

**Linha de base (ano 5):** receita R$ 17,1 mi · EBITDA +R$ 3,4 mi · infraestrutura R$ 1,2 mi/ano ·
time de plataforma de 5 pessoas.

### O que já gastamos, e está no orçamento

| Investimento | Cobre qual falha | R$/ano |
|---|---|---:|
| **Arquitetura celular por porte de certame** | Certame gigante degradando os outros (D7) | 280 mil |
| **Ensaio de carga contratualizado** antes de cada certame grande | Capacidade nunca ensaiada (D1) | 120 mil |
| **Fila de espera na borda** | Todos entram e ninguém completa (D4) | 120 mil |
| **Controle, backup e restauração fora do provedor primário** | Ataque (D9) e queda regional, em cerca de 8 h | 170 mil |
| **Escalonamento dirigido pelo calendário do edital** | Reativo chega tarde (D8) | 60 mil |
| **Restauração testada trimestralmente** | Backup que nunca foi restaurado (D5) | 60 mil |
| **Publicação estática de gabarito e resultado em CDN** | Pico de leitura derrubando o banco (D2) | 50 mil |
| **Geração assíncrona de documento** (boleto, cartão, PDF) | CPU consumida por geração síncrona (D3) | 40 mil |
| **Congelamento de implantação por calendário** | Implantação ruim na véspera (D6) | 0 |
| **Total M1** | | **R$ 900 mil**, ou **26% do EBITDA** |

### O que a segunda nuvem sempre ligada custaria **em cima** disso

| Item | R$/ano |
|---|---:|
| 2 engenheiros de plataforma adicionais | 660 mil |
| *Landing zone* e replicação de dados entre nuvens | 240 mil |
| Capacidade de standby, com escala no pico | 200 mil |
| Ferramental duplicado: observabilidade, segurança, IaC | 120 mil |
| **Total M2, marginal** | **R$ 1,22 mi**, ou **36% do EBITDA** |

**Mais da metade do custo da segunda nuvem é gente**, num time que hoje tem cinco pessoas. Não é
uma assinatura que se cancela: é contratação de perfil sênior escasso, com 4 a 6 meses de ciclo.

### O que esse R$ 1,22 milhão compra

Uma única coisa: reduzir o RTO de **queda regional total** de cerca de 8 horas para 4 horas.

| | |
|---|---|
| Custo marginal | **R$ 1,22 mi/ano** |
| Frequência estimada da classe de falha | **1 evento a cada 4 anos** |
| Horas de indisponibilidade evitadas por evento | **4** |
| **Custo por hora de indisponibilidade evitada** | **R$ 1,22 milhão** |
| Natureza da falha | **Com remédio barato e praticado**, embora não garantido: reabrir o prazo |

> **R$ 1,22 milhão por hora evitada, numa falha que tem remédio.** É este número, e não o placar
> de prós e contras, que encerra a discussão. E ele só aparece quando a pergunta é formulada na
> margem.

### A premissa desta conta, e como ela foi resolvida

As "4 horas evitadas" supõem que, sob M1, uma queda total do sa-east-1 se resolve **restaurando o
backup em outro ambiente em cerca de 8 horas**. Sem esse destino, o RTO do M1 seria **a duração da
própria queda**, e a conta mudaria. Isso importa porque a Forrester prevê incidentes de vários
dias em 2026:

| Se o M1 não tivesse onde restaurar | Horas que M2 compraria | Custo por hora evitada |
|---|---:|---:|
| Queda de 24 h | 20 | R$ 244 mil |
| Queda de 72 h | 68 | R$ 72 mil |

**Por isso o destino da restauração faz parte do M1.** O backup é restaurado a frio no **Google
Cloud, região `southamerica-east1`, em São Paulo**, o mesmo provedor que já guarda o backup
imutável. Lá nada fica ligado: existe o backup, a infraestrutura como código pronta para subir e
a cota de máquinas pedida antecipadamente. Na restauração, o sistema sobe em modo de recuperação:
inscrição, pagamento e consulta funcionam; boleto e PDF são gerados por um processo no próprio
Kubernetes, mais lento, no lugar de SQS e Lambda. A publicação estática também é gravada lá, para
que gabarito e resultado continuem no ar mesmo sem o S3 de São Paulo.

| Destino avaliado | Cobre | Contra | Custo/ano |
|---|---|---|---:|
| Outra região da AWS, fora do Brasil | Queda da região | Dado do candidato sai do país; não cobre falha da AWS inteira nem conta comprometida | ~35 mil |
| **Google Cloud, São Paulo** ✅ | **Queda da região, falha da AWS inteira e conta comprometida** | Exige tornar o sistema portável, uma vez | **~80 mil** |

*A alternativa equivalente é o Azure, região Brazil South, também no estado de São Paulo. As duas
regiões estão verificadas na fonte (§6.6).*

**O custo da restauração a frio, estimativa nossa** com o mesmo custo por engenheiro da conta do
M2 (R$ 330 mil por ano):

| Item | R$/ano |
|---|---:|
| Tornar o sistema portável: cerca de 3 meses de trabalho, diluídos em 3 anos | 27 mil |
| Manter a infraestrutura como código do segundo provedor: 10% de um engenheiro | 33 mil |
| Máquinas ligadas nos 4 ensaios de restauração do ano, no máximo | 20 mil |
| **Total** | **80 mil** |

Ele cabe nas linhas do plano de negócios: R$ 60 mil são tempo do time de plataforma que já existe,
e R$ 20 mil são máquinas de teste. Esses R$ 20 mil somam-se aos R$ 60 mil da restauração testada
(D5), que já pagavam o teste trimestral quando ele era feito dentro da AWS; contamos as duas
linhas, o que superestima um pouco o custo. Num evento real, o sistema rodando no Google custa na ordem de
R$ 3 mil por dia, que é a infraestrutura anual dividida por 365.

| Queda regional de | Custo em 4 anos | Horas que a restauração a frio evita | Custo por hora evitada |
|---|---:|---:|---:|
| 24 h | R$ 320 mil | 16 | **R$ 20 mil** |
| 72 h | R$ 320 mil | 64 | **R$ 5 mil** |

**Uma régua para comparar os dois preços.** Por evento, a cada quatro anos, a segunda nuvem
desligada custa R$ 320 mil, menos que o contrato anual de uma única banca média (ACV de R$ 380 mil
no ano 5). A sempre ligada custaria R$ 4,88 milhões por evento, o contrato anual de quase 13
bancas, para comprar 4 horas.

> **A segunda nuvem desligada se paga com folga; a sempre ligada continua não se pagando.** E, com destino, o
> número de R$ 1,22 milhão por hora **deixa de depender da duração da queda**: o M2 compra sempre
> as mesmas 4 horas, de 8 para 4, seja a queda de um dia ou de três. A pergunta nunca foi
> "segunda nuvem, sim ou não", e sim **"segunda nuvem fazendo o quê"**.

> **Dois riscos declarados.** As 8 horas são estimativa até o primeiro ensaio medi-las, e o
> ensaio trimestral é o que as confirma ou derruba (gatilho 3). E, numa queda regional da AWS,
> muitas empresas tentam subir sistemas nas outras nuvens ao mesmo tempo: a cota de máquinas
> pedida antes, e comprovada em cada ensaio, é o que garante que haverá onde subir.

**E o EBITDA:** adotar M2 levaria o resultado de R$ 3,4 mi para **R$ 2,2 mi**, derrubando a
margem de 20% para 13%, para comprar quatro horas a cada quatro anos.

**Duas ressalvas de honestidade intelectual**, melhor ditas do que descobertas pela banca:

1. **A prorrogação não é indolor.** Custa reputação com a banca, e pode custar o contrato. O
   valor esperado subestima o dano de cauda.
2. **"Não podemos pagar" seria um argumento falso**, e não é o nosso. Com a rodada em caixa,
   **poderíamos pagar. Não fazemos porque compraríamos a proteção errada.**

> **A frequência de 1 evento a cada 4 anos é uma estimativa nossa, não um dado publicado.**
> Ela está declarada como estimativa porque o argumento não depende de precisão: mesmo dobrando a
> frequência para 1 evento a cada 2 anos, o custo cai para R$ 610 mil por hora evitada, e a
> conclusão não muda.

---

## 3.11 A ressalva: o que de fato sai do provedor primário

Rejeitar a segunda nuvem sempre ligada não significa aceitar ponto único em tudo.

| O que sai | Por quê | R$/ano |
|---|---|---:|
| **DNS e WAF** (provedor neutro) | Se a nuvem cai e o DNS está dentro dela, **não dá nem para apontar para uma página de status** | 60 mil |
| **Página de status** | Precisa sobreviver à queda que está anunciando. Hospedada fora, sempre | 8 mil |
| **Backup off-site no Google Cloud** | Backup no provedor que você quer sobreviver não é backup, é cópia | 22 mil |
| **Restauração a frio no Google Cloud, em São Paulo** | Se a AWS cair, é onde o sistema volta, em cerca de 8 h. Nada fica ligado: o ensaio trimestral é o custo | 80 mil |
| **Total** | | **R$ 170 mil, ou 5,0% do EBITDA** |

> **Isto é multicloud?** Pelo critério estrito do enunciado, que é usar mais de um provedor,
> **sim**, na camada de sobrevivência. E é exatamente por isso que a resposta honesta não é "não"
> puro, e sim: **uma nuvem para operar, duas para sobreviver.**

---

## 3.12 A decisão

**Adotamos M1**, com controle, backup e restauração fora da AWS (§3.11). **Rejeitamos acrescentar
M2.**

O critério que organizou tudo: **para cada real gasto em continuidade, qual classe de falha ele
cobre, e com que frequência essa classe acontece?** Sob esse critério, a segunda nuvem **sempre ligada**
perde não por ser cara, mas por encurtar em só 4 horas a única linha da tabela que não é nossa, a
R$ 1,22 milhão por hora. A segunda nuvem **desligada** passa pelo mesmo critério: R$ 80 mil por ano
cobrem a mesma linha a um custo entre R$ 5 mil e R$ 20 mil por hora evitada.

---

## 3.13 Gatilhos de reavaliação

A decisão tem validade condicional declarada. Qualquer um destes reabre a discussão:

1. Uma banca com **mais de 10% da receita** exigir contratualmente DR multiprovedor **com RTO
   abaixo de 8 horas**.
2. **A Certamis passar a aplicar prova on-line.** Não é hipótese de 2028: a Cebraspe aplicou prova
   on-line remota para a AgSUS em 2026, e ao menos uma aplicação foi anulada e remarcada por falha
   de sistema. **Numa prova sendo aplicada, o remédio deixa de ser reabrir um prazo e passa a ser
   anular e reaplicar a prova inteira**, com nova prova elaborada e nova data para toda a base.
3. Um **ensaio trimestral de restauração no segundo provedor passar de 8 horas**, ou um incidente
   real de queda regional com impacto acima de 8 horas durante encerramento de prazo.
4. Norma do setor público exigindo continuidade multiprovedor de fornecedor crítico.
5. A AWS anunciar uma **segunda região no Brasil**. Aí a restauração pode voltar para dentro da
   AWS, com o mesmo ferramental, e a escolha do destino muda.
6. O time de plataforma passar de **12 pessoas**, quando C2 e C3 deixam de morder.

Os gatilhos **2 e 3** são os mais interessantes para a discussão. O **3** é o único que depende
só de nós: o ensaio mede o RTO de verdade, e se passar de 8 horas o número de R$ 1,22 milhão por
hora deixa de valer. O **2 é o mais importante do trabalho inteiro**: se a prova passar a ser
aplicada on-line, a falha continua tendo remédio, mas um remédio de outra ordem de grandeza, e a
conta de §3.10 pode inverter.

> **Com uma ressalva que o próprio caso AgSUS ensina.** Lá a causa foi o controle de acesso da
> própria banca, não o provedor. Mesmo com prova on-line, o primeiro real continua indo para o
> endurecimento. A segunda nuvem sempre ligada só passa a se pagar para o risco residual do provedor, quando o
> remédio para cada hora de queda deixa de ser barato.

---

## 3.14 Resumo

| Pergunta | Resposta |
|---|---|
| Adotar multicloud? | **Uma nuvem para operar, duas para sobreviver:** DNS, WAF, status, backup e restauração a frio fora da AWS |
| E se a AWS cair inteira? | Voltamos no **Google Cloud, em São Paulo, em cerca de 8 horas**, ensaiado todo trimestre |
| Por que não operar em duas? | Operar em duas cobre só 1 das 8 falhas, e as outras 7 são dívidas nossas |
| É por custo? | **Não.** O endurecimento já está pago; a segunda nuvem sempre ligada custaria R$ 1,22 mi a mais |
| Quanto custa a hora comprada? | **R$ 1,22 milhão por hora de indisponibilidade evitada** |
| O capital resolve? | Resolve C3 a C7. **Não resolve C1 e C2** |
| O que muda a decisão? | Passarmos a aplicar prova on-line, que já existe no mercado: o gatilho 2 |
