# Continuar o trabalho: Certamis (FIAP)

*Documento de retomada. Foi escrito para quem chega frio: outra sessão, outro dia, outro
assistente. Tudo que é preciso saber para continuar sem refazer decisão já tomada está aqui.*

**Última atualização:** 23 de setembro de 2026.

---

## 1. O que é isto

Dois trabalhos de pós-graduação da FIAP, ligados entre si:

| Trabalho | O que pede | Pasta |
|---|---|---|
| **Startup One** | Conceber uma startup do zero, com plano de negócios, para apresentar a investidores. **Precisa ser implementada** como POC | `fiap-startup-one/` |
| **Soluções Multicloud** | Defender a **adoção ou não adoção** de estratégia multicloud para a empresa do Startup One | `fiap-multicloud/` |

**Premissa do trabalho de multicloud:** a empresa **não é mais uma startup**. Está consolidada,
pós-breakeven, discutindo evolução de arquitetura por causa de crescimento e de nova rodada.

**Formato da apresentação de multicloud:** **1 apresentador**, 4 a 6 minutos de defesa mais 2 a 4
minutos de discussão.

**Critérios de avaliação explícitos:** tempo usado (4 a 6 min) · referências de institutos de
mercado (Forrester ou outros) com dados e pesquisas · descrição das arquiteturas atual e alvo
**citando exemplos de soluções de mercado**.

---

## 2. Estado atual

> **Está pronto para apresentar.** Slides, roteiro e pitch estão em PDF, revisados numericamente
> (15/09), revisados **adversarialmente nos argumentos** (22/09) e com a **véspera reverificada
> nas fontes primárias** (23/09), o que corrigiu cinco erros de uma rodada anterior e mudou os
> slides 3, 5 e 8 do multicloud e os slides 1, 2, 4 e 10 do pitch. Ver §14. **Em 23/09 a tese
> mudou** para "uma nuvem para operar, duas para sobreviver", com restauração a frio no Google
> Cloud. Ver §15.

| Entregável | Onde | Status |
|---|---|---|
| Slides do multicloud | `fiap-multicloud/apresentacao/Certamis-multicloud-slides.pdf` | ✅ capa mais 8 slides, 16:9, fala de 822 palavras (~5min28) |
| Roteiro do apresentador | `fiap-multicloud/apresentacao/Certamis-multicloud-roteiro.pdf` | ✅ fala palavra a palavra |
| Pitch do Startup One | `fiap-startup-one/apresentacao/Certamis-pitch.pdf` | ✅ 10 slides |
| Previews em PNG | `*/apresentacao/previews/` | ✅ um PNG por slide |
| Diagrama da arquitetura | `fiap-multicloud/diagramas/arquitetura-alvo.png` | ✅ 2880×2040 |
| Documentação dos dois trabalhos | `fiap-*/0*.md` | ✅ completa |

### O que falta

1. **Ensaiar com cronômetro, 3×.** O tempo é critério de avaliação e a fala mede ~5min28 de
   uma janela de 4 a 6. A ordem de corte está em `fiap-multicloud/05-roteiro-apresentacao.md`.
2. **Implementar a POC.** Escopo em `fiap-startup-one/04-escopo-poc.md`. 1 pessoa, 8 semanas.
3. **Renomear a pasta raiz** para `FIAP-Certamis`, com o VS Code fechado, usando `Rename-Item`
   no PowerShell. O caminho atual ainda tem o nome antigo da empresa. A memória do assistente já foi
   copiada para o nome novo.

---

## 3. A empresa: Certamis

**Plataforma que conduz um processo seletivo público de ponta a ponta**, do edital à homologação
do resultado. Vendida para **bancas organizadoras**, que são as empresas contratadas por
prefeituras, autarquias e universidades para realizar concursos e vestibulares.

*O nome vem de "certame", como a administração pública chama o processo seletivo público.*

> **Sobre o nome.** A empresa teve outro nome até 15/09/2026, e o usuário pediu para abandoná-lo.
> Tudo foi renomeado para **Certamis**, e o nome anterior não aparece mais em lugar nenhum do
> projeto. Checagem de colisão feita em set/2026: não existe nenhuma "Certamis" no ramo. Há um
> *Instituto Certame* (treinamento em licitações) e um portal *O Certame* (notícias de concursos),
> ambos em ramos distintos.

### Por que exige nuvem

Um certame tem **cinco picos**, não um:

| # | Pico | Quem acessa | Tipo |
|---|---|---|---|
| 1 | Encerramento da **isenção** | quem pede isenção | escrita |
| 2 | Encerramento da **inscrição** | quem ainda não se inscreveu | **escrita, e é o que derruba** |
| 3 | **Cartão de confirmação** | a base inteira | leitura, sai do CDN |
| 4 | **Gabarito** e recurso | a base inteira | leitura, depois escrita |
| 5 | **Resultado** e classificação | a base inteira | leitura, sai do CDN |

> **Cuidado com esta parte, porque a versão anterior errava.** Os picos 3, 4 e 5 são maiores em
> requisições, mas são leitura de dado que não muda e **saem inteiros do CDN**. O pico que
> dimensiona o banco é o **2**, o da inscrição, porque é escrita.

### O "40×", definido sem ambiguidade

> **É 40× a média do PRÓPRIO CERTAME, não a média anual da empresa.**

| | |
|---|---|
| Certame de referência | 300 mil candidatos, janela de 30 dias |
| Média daquele certame | 417 inscrições/h |
| **Pico a 40×, e alvo do contrato** | **16,7 mil inscrições/h**, ou 4,6 por segundo |
| Média anual da empresa inteira | 548 inscrições/h |

A versão anterior dizia "40× a média" sem dizer de quê, e isso brigava com a frase "a carga do ano
cabe em poucas horas", que implicaria um múltiplo muito maior. **Se alguém reescrever isso, tem
que manter o denominador explícito.**

### Por que agora

**Lei nº 14.965/2024** (sancionada em 09/09/2024), normas gerais de concursos públicos. Pelo texto
(art. 1º, 8º e 13): vale para os concursos **de todos os entes** a partir de **1º/01/2028**, com
exceções (magistratura, MP, Defensoria, militares); estados, DF e municípios podem, **em
alternativa**, editar normas próprias; e a prova a distância é **facultativa** e depende de
regulamentação. **A lei não obriga ninguém a trocar de sistema nem a fazer prova on-line.** Muda as
regras que o sistema precisa cumprir, e a prova on-line **já é praticada em 2026** (AgSUS).

> ⚠️ **Não escrever nenhuma destas duas frases, ambas já estiveram nos documentos e ambas estão
> erradas:** "todo o mercado precisa se replataformar até 2028" e "a lei é obrigatória só para
> concursos federais". Ver §14.

### Modelo de receita

Assinatura anual pequena (R$ 48 a 84 mil) **mais valor por inscrição processada (R$ 3,00)**, e o
segundo domina. **Vender para a banca evita licitação:** o órgão licita a banca; a banca contrata
fornecedor como empresa privada.

---

## 4. Os números canônicos

> **Use estes. Se algum documento divergir, o documento está errado.**
> **Estes números foram redimensionados em 22/09/2026.** A versão anterior era três vezes maior e
> não sobrevivia a checagem de plausibilidade. Ver §10.

### Ano 5 (a premissa do trabalho de multicloud)

| | |
|---|---|
| Bancas clientes | **45** |
| Inscrições processadas/ano | **4,8 milhões** |
| Receita (ARR) | **R$ 17,1 mi** |
| EBITDA | **+R$ 3,4 mi (20%)** |
| Custo total do ano 5 | **R$ 13,7 mi** |
| Headcount | 44 · **time de plataforma: 5** |
| Infraestrutura | **R$ 1,2 mi/ano, já incluindo o programa de capacidade** |
| Breakeven | **4º trimestre do ano 4**, com R$ 8,8 mi de queima acumulada |
| Rodada | **R$ 30 mi no mês 46** |

### Projeções

| R$ mi | Ano 1 | Ano 2 | Ano 3 | Ano 4 | Ano 5 |
|---|---:|---:|---:|---:|---:|
| Bancas | 3 | 9 | 18 | 31 | 45 |
| Adições brutas | 3,0 | 6,2 | 9,5 | 14,1 | 15,9 |
| Inscrições (mi) | 0,10 | 0,41 | 1,11 | 2,64 | 4,80 |
| ACV médio (R$ mil) | 160 | 195 | 245 | 315 | 380 |
| **ARR** | **0,5** | **1,8** | **4,4** | **9,8** | **17,1** |
| **EBITDA** | −2,4 | −3,0 | −2,9 | −0,5 | **+3,4** |
| Headcount | 9 | 17 | 26 | 35 | 44 |

*ARR fecha como bancas × ACV médio nos cinco anos, e o ano 5 também fecha pelo outro lado:
4,80 mi × R$ 3,00 mais 45 × R$ 60 mil.*

### Unit economics (ano 3)

ACV **R$ 245 mil** · margem bruta **77%** · **CAC R$ 158 mil** · retorno de CAC **10 meses** ·
perda 6%/ano · LTV **~R$ 710 mil** · LTV/CAC **~4,5×**

> **O CAC é DERIVADO, não estimado:** é a linha inteira de Vendas e Marketing dividida pelas
> adições brutas. A versão anterior afirmava R$ 65 mil, mas o próprio P&L implicava R$ 200 mil.
> **Não voltar a declarar um CAC que não saia da tabela.**

### Mercado

TAM **R$ 90 mi** · SAM **R$ 36 mi** · SOM ano 5 **R$ 17,1 mi**
**Premissa declarada: o segmento tem cerca de 150 bancas, e 45 clientes são 30% dele.** Se forem
80, seriam 56% e o plano pararia em ~25 bancas. Contar o segmento é a segunda verificação mais
urgente, atrás só do preço.

### Continuidade: a conta é MARGINAL

| | R$/ano | Cobre |
|---|---:|---|
| **M1 · nuvem única endurecida**, adotado e **já no orçamento** | **900 mil** (26% do EBITDA) | as 7 falhas nossas, e volta da 8ª em ~8 h |
| **M2 · segunda nuvem sempre ligada**, rejeitado, **incremento sobre M1** | **1,22 mi** (36% do EBITDA) | só encurta a 8ª de 8 h para 4 h |
| dos quais, dentro de M1: controle, backup e restauração fora da AWS | 170 mil | 5,0% do EBITDA |
| dos quais, dentro desses 170: restauração a frio no Google Cloud | 80 mil | estimativa nossa |

> **M1 e M2 NÃO são alternativas, e apresentá-los lado a lado é o erro que a revisão de 22/09
> corrigiu.** Sem M1, a segunda nuvem cai pelo mesmo motivo que a primeira. A pergunta é: sobre o
> que já existe, vale somar M2?
>
> **O número que encerra a discussão:** M2 compra 4 horas de RTO num evento estimado a cada 4
> anos, ou seja **R$ 1,22 milhão por hora de indisponibilidade evitada**, numa falha que quase
> sempre se resolve reabrindo um prazo.
>
> **A premissa desta conta, resolvida em 23/09:** as 4 horas supõem que o M1 restaura em outro
> ambiente em cerca de 8 h. O destino é o **Google Cloud, `southamerica-east1`, São Paulo**, a
> frio. Sem ele, a 72 h o custo cairia para uns R$ 72 mil por hora; com ele, o M2 compra sempre as
> mesmas 4 horas. A restauração a frio custa ~R$ 20 mil por hora evitada numa queda de 24 h e ~R$ 5
> mil numa de 72 h. Ver `03-defesa-multicloud.md` §3.10 e §15 abaixo.

### Financiamento

Pre-seed R$ 1,5 mi (M0) · **Seed R$ 8 mi (M8)** · Série A R$ 15 mi (M28) · Série B R$ 30 mi (M46)

---

## 5. A tese do trabalho de multicloud

> **Uma nuvem para operar, duas para sobreviver.** Não operar em duas nuvens; ter DNS, WAF,
> status, backup e restauração a frio fora da AWS.

### O argumento central (é o slide 3, e é a defesa inteira)

| Falha | Dívida de origem | Multicloud resolve? |
|---|---|---|
| Pool de conexões esgotado no pico | D2 | não |
| Geração síncrona de boleto/cartão comendo a CPU | D3 | não |
| Todos entram, todos degradam, ninguém completa | D4 | não |
| Capacidade nunca ensaiada para o pico | D1 | não |
| Escalonamento reativo chegando tarde | D8 | não |
| Um certame gigante degradando os outros inquilinos | D7 | não |
| Ataque à aplicação: invasão ou negação de serviço | D9 | não |
| **Queda regional total do provedor** | nenhuma | **sim** |

*A linha de ataque (D9) entrou em 22/09/2026, depois do caso FGV. A proteção contra ela é o WAF na
borda, que já estava na ressalva.*

> **A evidência mais forte do trabalho:** nos três casos de falha em certame documentados em 2026
> (Alece, AgSUS, FGV), **nenhum foi queda do provedor de nuvem.** Está no slide 3.

> **A coluna do meio é a defesa contra a objeção mais óbvia.** A tabela é taxonomia nossa, não
> estatística de mercado, e isso está declarado em §6.8 das referências. O que a sustenta é que
> cada linha aponta para uma **dívida técnica documentada em §2.3 antes de existir discussão de
> multicloud**. O argumento se apoia na origem das falhas, não na contagem. **Não voltar a usar
> "1 de 8" como se fosse dado medido.**

### Os três argumentos de apoio

1. **A falha em fase de prazo tem remédio barato, mas não garantido.** No caso Alece (2026), a
   banca reabriu o prazo de recurso depois de uma falha, **e o juiz negou** reabrir a isenção pedida
   com base na instabilidade. *(Não escrever "ninguém perde o direito de concorrer" nem "há
   registro de prazo estendido por decisão judicial": a primeira é forte demais e a segunda não tem
   caso nomeado.)*
2. **Não é economia.** M1 já está pago; M2 custaria R$ 1,22 mi a mais, a R$ 1,22 mi por hora. E
   a segunda nuvem **a frio**, que se paga, foi adotada.
3. **Memória operacional.** Um caminho que roda 1× a cada 4 anos, num time de 5 pessoas, não cria
   competência de plantão.

> ⚠️ **O argumento 3 tem uma armadilha, e a revisão de 22/09 a corrigiu.** A versão anterior dizia
> que o failover "não pode ser exercitado porque exige carga real". **Isso é falso, e o próprio M1
> prova que é falso**, porque inclui um gerador de carga sintética. O argumento que sobrevive é
> sobre **memória operacional**, não sobre impossibilidade de teste. Se alguém reescrever C2, não
> voltar à versão errada.

### A ressalva (slide 6), que não pode ser cortada

DNS e WAF (60 mil) · página de status (8 mil) · backup no Google Cloud (22 mil) · restauração a
frio no Google Cloud (80 mil) = **R$ 170 mil/ano**. E **conceder que "pelo critério estrito isto é
multicloud"**, na camada de sobrevivência, é o que impede a banca de dizer que a pergunta não foi
respondida.

### Os gatilhos que invalidariam a decisão

- **Gatilho 2, o mais importante:** se a Certamis passar a aplicar prova on-line. **Não é cenário
  de 2028:** a AgSUS (Cebraspe) já aplicou prova on-line remota em 2026, e uma aplicação foi anulada
  e remarcada. Numa prova em andamento **o remédio passa a ser reaplicar a prova**, uma ordem de
  grandeza mais caro que reabrir um prazo, e a conta pode inverter. Não escrever que "a falha deixa
  de ter remédio": na AgSUS ela teve, só que caro. E lá a causa foi da própria banca.
- **Gatilho 3, o que depende só de nós:** se o ensaio trimestral de restauração no Google Cloud
  passar de 8 horas, o número de R$ 1,22 milhão por hora deixa de valer. Substituiu o gatilho 5
  (segunda região AWS no Brasil) no slide 8; o 5 continua no documento.

---

## 6. Decisões fechadas, não relitigar

| Decisão | Valor |
|---|---|
| Modelo de negócio | **B2B**, vender para a banca, não para o órgão |
| IA no produto | **Nenhuma.** O agrupamento de recursos é por **fundamento estruturado** |
| Escopo da plataforma | **Vai até o resultado final** |
| Logística física | **Fica com a banca.** Nós paramos no dado |
| Conta do candidato | **Única por CPF**, viva entre bancas e certames |
| MVP | **Só prova objetiva** |
| Pagamento | **Orquestramos, não processamos.** Taxa de inscrição é receita pública |
| Multicloud | **Uma nuvem para operar, duas para sobreviver.** Segunda nuvem só a frio, no Google Cloud em São Paulo (23/09/2026). **Não voltar à segunda nuvem ligada, nem aumentar custo em milhões** |
| POC | **1 pessoa, 2 meses** |
| Nome da empresa | **Certamis**, adotado em 15/09/2026 |
| Travessões nos textos | **Não usar.** Vale para documentação, slides e conversa |
| Escala da empresa | **Redimensionada em 22/09/2026** para escala de startup. Não voltar aos números antigos |
| O programa de capacidade | **Está dentro do orçamento**, não é um extra a aprovar |

---

## 7. O que está verificado e o que NÃO está

> **Regra do projeto: premissa não verificada é marcada, nunca embutida.**

### ✅ Verificado (set/2026, com fonte)

| Fato | Fonte |
|---|---|
| **Lei 14.965/2024**: normas gerais para todos os entes, opção de norma própria (art. 13, §2º), vigência 1º/01/2028, prova a distância facultativa (art. 8º) | Texto da lei, Câmara e Normas Legais |
| **Três falhas de certame em 2026**, nenhuma do provedor: Alece, AgSUS, FGV | Diário do Nordeste, O Povo, Direção Concursos, Correio Braziliense, Acheconcursos |
| **Previsão da Forrester** de ao menos dois incidentes de nuvem de vários dias em 2026 | Forrester, *Predictions 2026*, Lee Sustar, 17/11/2025 |
| **O mercado está ocupado:** Gestor Editais, ProSeleta, Fábrica de Provas, ADSR, MH Informática, Instituto Avalia | sites dos fornecedores |
| **Gestor Editais já cobra por inscrição** | site do fornecedor |
| **Taxa de inscrição é receita pública** | TCE-MG, MPRJ |
| Modelo **"custo zero"** | MPRJ |
| **Taxa de R$ 69,33 por candidato** | MPRJ |
| **Ano eleitoral veda nomeação, não o certame** | TecConcursos |
| **AWS tem uma região no Brasil**; Rio é Local Zone | AWS |
| **Google Cloud tem região em São Paulo** (`southamerica-east1`); **Azure tem Brazil South**, no estado de São Paulo, com zonas | Google Cloud, Microsoft Learn (23/09/2026) |
| **HashiCorp/Forrester 2024** é a edição mais recente | HashiCorp |

### ⚠️ NÃO verificado

| Número | Faixa usada | Impacto |
|---|---|---|
| **Preço por inscrição** | R$ 1,50 a 5,00 (base 3,00) | **Decide o negócio. Abaixo de R$ 2,29 não fecha** |
| **Quantas bancas médias existem** | 80 a 200 (plano assume 150) | **Decide se 45 clientes é plausível ou fantasia** |
| Inscrições pagas/ano no Brasil | 20 a 40 milhões | Define o TAM |
| Taxa média paga pelo candidato | ~R$ 80 | Contextualiza o preço |
| Concentração nas últimas 48 h | 80 a 90% | Sustenta o argumento de nuvem |

### ⚠️ Afirmado sem fonte externa, e declarado como tal

Está tudo em **`fiap-multicloud/06-referencias.md` §6.8**, que existe exatamente para isso: a
taxonomia das 8 falhas, a frequência de 1 queda regional a cada 4 anos, os RTOs de 6 a 12 h e 4 h,
e o custo de R$ 80 mil da restauração a frio. **Declarar antes transforma fragilidade em método.**

---

## 8. Como regerar os artefatos

**Python 3.13 existe nesta máquina** (`C:\Users\rockb\AppData\Local\Microsoft\WindowsApps\python.exe`),
com `pip` funcionando. Node não há. **O Edge foi removido; o Chrome está instalado** e os scripts
detectam qualquer um dos dois.

| O quê | Comando |
|---|---|
| **Diagrama** | `powershell -ExecutionPolicy Bypass -File fiap-multicloud\diagramas\render-arquitetura.ps1` |
| **Slides e roteiro do multicloud** | `powershell -ExecutionPolicy Bypass -File fiap-multicloud\apresentacao\build-pdf.ps1` |
| **Pitch do Startup One** | `powershell -ExecutionPolicy Bypass -File fiap-startup-one\apresentacao\build-pdf.ps1` |

Cada `build-pdf.ps1` gera os PDFs **e** os previews PNG, e é idempotente. **Rode o do diagrama
antes do deck do multicloud**, porque o diagrama entra embutido no slide 7.

> **O HTML fonte mora dentro do repositório** (`deck.html` e `pitch.html`). A versão anterior
> guardava o deck numa pasta temporária de sessão, que evaporou e obrigou a reconstruir tudo a
> partir dos previews. **Não repetir isso.**

**Armadilhas conhecidas desta máquina:**

- Scripts `.ps1` com acento precisam ser gravados em **UTF-8 com BOM**, senão são lidos como ANSI
  e os acentos quebram o parser.
- **Não usar `Start-Process -Wait` com Chrome headless.** O Chrome deixa processos filhos vivos e
  o `-Wait` espera a árvore inteira, travando o script. Já deixou 21 processos órfãos. Os scripts
  esperam o **arquivo PDF parar de crescer** e depois matam a árvore com `taskkill /T /F`.
- **Não usar `2>&1` em executável nativo no PowerShell 5.1.** Transforma stderr em erro fatal
  mesmo quando o programa retorna 0.
- **Heredoc de bash com markdown grande quebra.** Para reescrever documento inteiro, usar a
  ferramenta de escrita de arquivo ou um script Python, não `cat <<EOF`.

---

## 9. Histórico: o que foi descartado e por quê

| Ideia | Por que caiu |
|---|---|
| **Sentinela**, antifraude consorciado para Pix | **Outra equipe apresentou a mesma solução** |
| 30 ideias em música | Descartadas quando o usuário excluiu IA aplicada |
| Detecção de música gerada por IA | Era a recomendação da rodada de IA/GPU |
| Transcrição para tablatura | POC bom, negócio mediano |

**Aprendizados que custaram caro:**

1. **Verificar concorrentes ANTES de desenvolver a ideia.**
2. **Não inventar restrição para fazer o argumento fechar.**
3. **Conceder o ponto forte do adversário antes que ele o use.** Foi o que salvou o slide 6.
4. **Conferir a soma de toda tabela de dinheiro.** A revisão de 15/09 achou um total errado que
   estava em cinco lugares.
5. **Conferir se a COMPARAÇÃO é válida, não só se as contas fecham.** A revisão de 15/09 validou
   a aritmética de uma comparação que era inválida na premissa. Ver §10.

---

## 10. As duas revisões, e o que cada uma achou

### 15/09/2026, revisão numérica

Conferiu se as contas fechavam. Achou um total de R$ 1,69 mi que somava R$ 1,83 mi, um piso de
preço errado, "6 bancas" onde o P&L exigia 4, e uma contradição sobre reversibilidade entre os
dois trabalhos. **Corrigiu tudo, mas não questionou se a comparação central fazia sentido.**

### 22/09/2026, revisão adversarial dos argumentos

| # | O que estava errado | Correção |
|---|---|---|
| 1 | **O slide da conta comparava M1 e M2 como alternativas.** Mas a própria tabela dizia que M2 não cobre nenhuma das 6 falhas internas, então ninguém escolheria M2 *em vez* de M1. A comparação era enviesada e indefensável | M1 entrou no orçamento e virou o piso; M2 virou **incremento**. Métrica nova: **R$ 1,22 mi por hora de indisponibilidade evitada** |
| 2 | **"1 de 7" era taxonomia própria apresentada como dado.** Com outra lista seria "3 de 9" | Cada falha amarrada a uma dívida D1 a D8 documentada antes; §6.8 declara o que é nosso |
| 3 | **C2 dizia que failover não pode ser exercitado**, mas o M1 constrói um gerador de carga | Reescrito para **memória operacional** num time de 5 pessoas |
| 4 | **CAC de R$ 65 mil contra R$ 200 mil implícitos no P&L.** Quebrava payback e LTV/CAC no pitch | CAC passou a ser **derivado** de S&M ÷ adições brutas: R$ 158 mil, payback 10 meses, LTV/CAC 4,5× |
| 5 | **O programa de capacidade não cabia no P&L**: infra 3,2 mi + programa 1,83 mi contra CPV de 5,2 mi | Infra redimensionada para R$ 1,2 mi e o programa **dentro** dela e do P&D |
| 6 | **"40× a média" brigava com "o ano cabe em poucas horas"** e o denominador não estava definido | Passou a ser 40× a média **do certame**, com o absoluto de 16,7 mil/h ao lado |
| 7 | **72 bancas eram 90% do extremo pessimista do mercado** | Caiu para 45, com a premissa de 150 bancas declarada e com sensibilidade |
| 8 | **Série B de R$ 80 mi era 88% do TAM** | Caiu para R$ 30 mi, 1,8× ARR |
| 9 | Egresso grátis afirmado sem a condição de saída integral; P1 negado rápido demais | Ambos matizados |
| 10 | Reversibilidade apoiada em fonte secundária, sem dizer | Declarado, e virou item 5 da lista de reverificação |

**O que foi conferido e continua certo:** a estrutura dos cinco picos, o argumento de eficácia, a
distinção entre falha reversível e irreversível, a ressalva do plano de controle, os dados de
instituto, e a concorrência verificada.

---

## 11. Como o usuário quer trabalhar

- **Quer honestidade brutal.** Pediu explicitamente avaliação crítica e reagiu mal a otimismo
  infundado.
- **Detecta inconsistência numérica e cobra.** Conferir aritmética antes de entregar, por script,
  não a olho. E conferir se a **comparação** é válida, não só se a soma fecha.
- **Não gosta de jargão não explicado.**
- **Quer ver antes de finalizar.** Os scripts de build já geram os previews automaticamente.
- **Corta escopo com decisão.** Perguntas curtas e numeradas funcionam.
- **Não usa travessões.**
- **Idioma: português.**

---

## 12. Índice dos arquivos

```
fiap-startup-one/
  README.md                          resumo e índice
  01-ideacao.md                      as 9 ideias avaliadas e por que a Certamis ganhou
  02-produto-features-e-fluxos.md    9 módulos, fluxogramas, máquina de estados
  03-plano-de-negocios.md            mercado, preço, projeções, riscos, verificações
  04-escopo-poc.md                   1 pessoa, 8 semanas, ~US$ 100
  apresentacao/
    Certamis-pitch.pdf               10 slides
    pitch.html                       ★ FONTE do pitch
    build-pdf.ps1                    regera PDF e previews
    previews/pitch-01..10.png

fiap-multicloud/
  README.md                          tese, mapa das exigências, status
  01-empresa-e-contexto.md           empresa, 5 picos, a definição do 40×, números do ano 5
  02-arquitetura-fase-startup.md     arquitetura AWS inicial e as 9 dívidas
  03-defesa-multicloud.md            ★ O NÚCLEO: a pergunta marginal, as 8 falhas, a conta
  04-arquitetura-alvo.md             células, agenda, publicação estática, modo degradado
  05-roteiro-apresentacao.md         fala medida, ordem de corte, 30 perguntas
  06-referencias.md                  fontes conferidas no trecho; §6.8 sem fonte; §6.9 erros
  apresentacao/
    Certamis-multicloud-slides.pdf   capa mais 8 slides
    Certamis-multicloud-roteiro.pdf  fala palavra a palavra
    deck.html                        ★ FONTE dos slides E do roteiro
    build-pdf.ps1                    regera os dois PDFs e os previews
    previews/slide-01..09.png        slide-01 é a capa
  diagramas/
    arquitetura-alvo.png
    render-arquitetura.ps1
```

---

## 13. Se você for retomar agora, comece por aqui

1. Leia `fiap-multicloud/03-defesa-multicloud.md`, sobretudo **§3.2**, que é a reformulação da
   pergunta e o que segura a defesa inteira.
2. Abra `fiap-multicloud/apresentacao/previews/`. Vê a capa e os 8 slides em 30 segundos.
3. Ensaie com cronômetro. **É a única pendência que afeta a nota.**

---

## 14. A reverificação da véspera (22/09/2026), e a correção dela

*Houve duas rodadas no mesmo dia. A primeira usou os resumos do buscador como se fossem fontes e
gravou cinco erros. A segunda leu as fontes, desfez os erros e achou dois fatos que mudaram a
defesa. O registro completo está em `fiap-multicloud/06-referencias.md` §6.7 e §6.9.*

### Os erros da primeira rodada, já desfeitos

| O que foi gravado | O que a fonte diz |
|---|---|
| Lei 14.965/2024 "obrigatória só para concursos federais" | Vale para todos os entes; estados, DF e municípios podem, em alternativa, editar normas próprias |
| Juiz do caso Alece "mandou reanalisar isenções negadas por instabilidade" | O juiz **negou** a reabertura fundada em instabilidade; a reanálise foi por outro critério |
| "Erros HTTP 503" no caso Alece | Não estão na fonte |
| A Forrester "recomenda desenhar para 72 h" | Não está em nenhum texto aberto da Forrester |
| Incidentes AWS afetaram o Brasil "por dependência de plano de controle" | A fonte não diz o mecanismo; o incidente dos Emirados não afetou o Brasil |

> **Lição, gravada também na memória do assistente:** resumo de buscador não é fonte. Todo fato
> que vai para um entregável precisa do trecho literal da fonte aberta.

### O que a segunda rodada mudou na defesa

| Achado | Efeito |
|---|---|
| **Três falhas de certame em 2026** (Alece, AgSUS, FGV), **nenhuma do provedor** | Reforça o argumento central. Entrou no slide 3 |
| **FGV caiu por ataque**, causa que não estava na tabela | Oitava linha na tabela, dívida **D9** (sem proteção de borda). A defesa já estava na ressalva: WAF e backup imutável |
| **Alece: remédio concedido no recurso e negado na isenção** | A premissa de reversibilidade virou "remédio barato, mas não garantido". Slides 3 e 5, §3.3, roteiro |
| **AgSUS: prova on-line remota já aplicada em 2026, anulada e reaplicada** | Gatilho 2 reescrito: não é 2028, e o remédio não some, fica caro. Slide 8, §3.13, roteiro |
| **Lei 14.965/2024 lida no texto** | "Por que agora" reescrito: muda as regras para todos, não obriga troca de sistema. Pitch slides 1, 2, 4 e 10 |
| **A conta do custo por hora supõe um destino de restauração** | Tabela de 8, 24 e 72 horas em §3.10. **Resolvida em §15** |

### O que ficou pendente, de verdade

- ~~Decidir o destino da restauração sob M1.~~ **Decidido em 23/09/2026:** Google Cloud em São
  Paulo, a frio. Ver §15.
- O ensaio com cronômetro, que continua sendo a pendência que afeta a nota.

### Achado final desta rodada: a fala era o dobro do tempo permitido

Medida por contagem de palavras, a fala palavra a palavra tinha **cerca de 1.490 palavras, perto de
10 minutos** a 150 palavras por minuto, contra uma janela de 4 a 6. O slide 8 tinha 154 palavras
para 20 segundos. A "régua de 5min30" sempre foi a divisão do tempo por slide, **nunca uma medição
do texto**, e os documentos a chamavam de "cronometrada".

**Corrigido:** as oito falas foram reescritas dentro do orçamento de cada slide, somando **784
palavras (5min14 a 150 palavras/min, 4min45 a 165)**, e depois 786 com a mudança de §15. O conteúdo cortado já estava no banco de
perguntas. **Regra:** qualquer fala reescrita tem de ser medida de novo, a 2,5 palavras por
segundo.

---

## 15. A mudança de tese (23/09/2026): uma nuvem para operar, duas para sobreviver

**Como aconteceu.** A conta de R$ 1,22 milhão por hora supunha que o M1 restaurava o backup em
algum lugar em ~8 h, e nenhum documento dizia onde. Duas opções foram avaliadas:

| | A · outra região AWS, fora do Brasil | **B · Google Cloud, São Paulo** ✅ |
|---|---|---|
| Queda da região | cobre | cobre |
| Falha da AWS inteira, ou conta comprometida (D9) | **não cobre** | cobre |
| Dado do candidato fora do Brasil | **sim** | não |
| Esforço | baixo, mesmo código | médio: SQS e Lambda viram processo no Kubernetes em modo de recuperação |
| Custo | ~R$ 35 mil/ano | **~R$ 80 mil/ano** |

**O argumento que tinha sido usado contra a B estava errado.** O C2 (memória operacional) não
distingue A de B: as duas são o próprio teste trimestral de restauração, que o plantão executa
quatro vezes por ano. O C2 só pesa contra a segunda nuvem **ligada**, cujo failover desvia tráfego
vivo e não se ensaia em produção.

**Os números novos, conferidos por script:**

| Item | Antes | Agora |
|---|---:|---:|
| M1 | 820 mil (24% do EBITDA) | **900 mil (26%)** |
| Controle, backup e restauração fora da AWS | 90 mil (2,6%) | **170 mil (5,0%)** |
| Onda 3 da migração | 270 mil | **350 mil** |
| M1 + M2 | 2,04 mi | **2,12 mi** |
| M2 marginal, custo por hora | 1,22 mi | 1,22 mi, **agora independente da duração da queda** |
| Restauração a frio, custo por hora evitada | | **~20 mil (queda de 24 h) · ~5 mil (72 h)** |

**O plano de negócios não mudou de número.** Os R$ 80 mil cabem nas linhas existentes: R$ 60 mil são
tempo do time de plataforma que já existe (P&D) e R$ 20 mil são máquinas de ensaio. Se fossem
contados como custo novo, o EBITDA do ano 5 iria de R$ 3,40 para R$ 3,32 mi.

**O que mudou nos entregáveis:** slides 2, 5, 6, 7 e 8 e as falas deles; o diagrama; §3.1, §3.8,
§3.9, §3.10, §3.11, §3.12, §3.13 e §3.14 da defesa; §4.3, §4.5, §4.6 e §4.7 da arquitetura; três
perguntas novas no banco; §6.1, §6.6 e §6.8 das referências; o plano de negócios §6.

**Um furo antigo, fechado junto:** o nível 4 do modo degradado dizia que gabarito e resultado
continuavam no ar "com a região inteira fora", mas eles moravam no S3 de São Paulo, que cai junto.
Agora a publicação estática também é gravada no Google Cloud e servida pelo Cloudflare.

**Riscos declarados:** as 8 horas são estimativa até o primeiro ensaio medir (gatilho 3); e numa
queda regional pode faltar máquina nas outras nuvens, por isso a cota é pedida antes e comprovada em
cada ensaio.

---

## 16. Revisão completa depois da mudança de tese (23/09/2026)

Todos os arquivos relidos do início. O que foi achado e corrigido:

| Onde | O problema | Correção |
|---|---|---|
| Defesa §3.2 | A pergunta "vale acrescentar uma segunda nuvem?" ficou ambígua, porque o M1 já tem uma | Virou "segunda nuvem **sempre ligada**", com nota explicando a desligada |
| Defesa §3.3 e §3.8 | "Failover que roda a cada 4 anos não é proteção" parecia valer também para a restauração no Google | Dito que ela é o teste trimestral; e explicado por que ela não cai no C6 |
| Defesa §3.4 | Dizia que a restrição geográfica era "irrelevante", mas ela é uma das razões para escolher o Google | Reescrito: irrelevante para operar em duas, relevante para o destino. Exigência declarada em §6.8 como não verificada |
| Defesa §3.10 | "A nuvem desligada se paga" sem régua | Régua: por evento, a desligada custa menos que o contrato de uma banca; a ligada, o de quase 13 |
| Defesa §3.10 | Possível dupla contagem entre os R$ 20 mil de máquinas e os R$ 60 mil da restauração testada | Declarado: as duas linhas foram contadas, o que superestima o custo |
| Todos | "Quente/fria" nos documentos, "ligada/desligada" nos slides | Padronizado em "sempre ligada" e "desligada"; "a frio" definido |
| Referências §6.6 | A tabela de soluções estava quebrada (linha do k6 solta) e dizia que o Google tem "uma única região na América do Sul", tirado de resumo automático | Tabela consertada; afirmação retirada e registrada em §6.9 |
| Roteiro | "Um evento a cada três meses para a conta virar" não dizia virar contra o quê | Trocado pela conta: seriam uns R$ 76 mil por hora |
| Plano de negócios §9 | Chamava falha durante a prova de "sem remédio", o que contradiz o caso AgSUS; e a Certamis nem aplica prova hoje | Reescrito como risco condicional, com remédio de outra ordem de grandeza |
| Plano de negócios §9 | "Um certame grande pode ser 20% da receita", sem base | Conta: R$ 0,9 mi, metade do ano 2 e 5% do ano 5 |
| Plano de negócios §2 e §9 | 25 bancas × R$ 380 mil são R$ 9,5 mi, não "R$ 9 mi" | Corrigido |
| Plano de negócios §12 | Chamava a AgSUS de queda "em fase de prazo"; foi durante a prova | Corrigido |
| Produto | "Banca opera com planilha" (o plano §7 diz que isso está errado); "repasse" de pagamento (não processamos); "cobra-se por fase" (o preço é por inscrição) | Os três corrigidos |
| Ideação | "Um evento paga um ano de operação" (R$ 0,9 mi contra custo de R$ 13,7 mi); "500 mil inscritos" contra 300 mil no resto | Corrigidos |
| Pitch slide 2 | "Por que acontece" culpava só a carga, mas dos três casos citados um foi ataque e outro configuração | Reescrito: carga não ensaiada e falta de proteção; a falha foi da banca, não da nuvem |
| Pitch slide 10 e README | "O que ninguém resolve" contradizia "quase ninguém sobrevive" do slide 8 | "Quase ninguém resolve" |
