# 2. Produto: funcionalidades e fluxos

*Plataforma de inscrição e gestão de processos seletivos. Documento de produto: **nenhuma decisão
de nuvem aqui.** Arquitetura vem depois, e vem depois de propósito.*

---

## 1. Quem é quem

| Ator | Papel | É o cliente? |
|---|---|---|
| **Banca organizadora** | Contratada para conduzir o certame. Configura, analisa, julga recursos, publica | ✅ **Sim, é quem paga e quem opera** |
| **Órgão / instituição contratante** | Prefeitura, autarquia, universidade, hospital. Contrata a banca, define vagas, homologa | Influencia a compra, não opera |
| **Candidato** | Se inscreve, paga, recorre, faz prova | ❌ Usuário, não cliente |
| **Analista da banca** | Julga isenção, laudo, recurso | Operador interno |
| **Banca examinadora** | Professores que julgam recurso de questão | Operador externo, acesso temporário |
| **Auditoria** | TCU, Ministério Público, Judiciário | Consome trilha, não opera |

> **Decisão de posicionamento:** o cliente é a **banca**, não o órgão. Banca média e pequena opera
> hoje com um fornecedor do mercado ou com sistema próprio envelhecido (o levantamento está no
> plano de negócios, §7). As grandes têm sistema interno e não compram.

---

## 2. O ciclo de vida de um certame

Não é um formulário. São **oito fases encadeadas**, cada uma com prazo legal e cada uma
recorrível.

```
┌──────────────────────────────────────────────────────────────────────────┐
│  1. CONFIGURAÇÃO          banca monta o certame, órgão aprova            │
│         │                                                                │
│         ▼                                                                │
│  2. PUBLICAÇÃO DO EDITAL  a partir daqui tudo é público e versionado     │
│         │                                                                │
│         ▼                                                                │
│  3. ISENÇÃO DE TAXA       prazo próprio, ANTES da inscrição   ◄── PICO 1 │
│         │                 análise → resultado → recurso                  │
│         ▼                                                                │
│  4. INSCRIÇÃO             cadastro, cargo, cota, pagamento    ◄── PICO 2 │
│         │                                                                │
│         ▼                                                                │
│  5. HOMOLOGAÇÃO           lista preliminar → recurso → definitiva        │
│         │                 análise de laudo e atendimento especial        │
│         ▼                                                                │
│  6. ENSALAMENTO           candidato → cidade → local → sala              │
│         │                 cartão de confirmação                ◄── PICO 3│
│         ▼                                                                │
│  7. APLICAÇÃO E GABARITO  prova, gabarito preliminar           ◄── PICO 4│
│         │                 recurso de questão → gabarito definitivo       │
│         ▼                                                                │
│  8. RESULTADO             nota, cotas, desempate, classificação ◄─ PICO 5│
│                           recurso → resultado final → homologação        │
└──────────────────────────────────────────────────────────────────────────┘
```

> **Observação de domínio, não de arquitetura:** existem **cinco picos**, não um. A primeira
> versão deste documento supunha que só a inscrição concentrava carga, e isso está errado.
> Gabarito e resultado geram acesso simultâneo de **toda a base**, não só de quem ainda não se
> inscreveu. Isso muda o dimensionamento e é insumo para o documento de arquitetura, não
> conclusão dele.

---

## 3. Fluxo do candidato

```
                         ┌─────────────────┐
                         │  Entra no site  │
                         └────────┬────────┘
                                  ▼
                    ┌─────────────────────────┐
                    │  Já tem conta (CPF)?    │
                    └────┬───────────────┬────┘
                     não │               │ sim
                         ▼               ▼
              ┌──────────────────┐   ┌─────────┐
              │ Cria conta única │   │  Login  │
              │ CPF · e-mail     │   └────┬────┘
              │ verificação      │        │
              └────────┬─────────┘        │
                       └────────┬─────────┘
                                ▼
                    ┌───────────────────────┐
                    │  Escolhe o certame    │
                    └───────────┬───────────┘
                                ▼
                    ┌───────────────────────┐
                    │  Quer pedir isenção?  │
                    └────┬─────────────┬────┘
                     sim │             │ não
                         ▼             │
          ┌──────────────────────────┐ │
          │  FLUXO DE ISENÇÃO        │ │
          │  (seção 4 · prazo        │ │
          │   próprio e anterior)    │ │
          └──────────┬───────────────┘ │
                     │ deferida        │
                     └────────┬────────┘
                              ▼
              ╔═══════════════════════════════╗
              ║   FORMULÁRIO DE INSCRIÇÃO     ║
              ║   (salva a cada passo)        ║
              ╠═══════════════════════════════╣
              ║ 1. Dados pessoais e endereço  ║
              ║ 2. Cargo / vaga               ║
              ║ 3. Cidade de prova            ║
              ║ 4. Cota (PcD, racial, outras) ║
              ║ 5. Atendimento especial       ║
              ║    + upload de laudo          ║
              ║ 6. Declarações e aceite       ║
              ║ 7. Revisão                    ║
              ╚═══════════════╤═══════════════╝
                              ▼
                    ┌───────────────────────┐
                    │  Isento?              │
                    └────┬─────────────┬────┘
                     sim │             │ não
                         │             ▼
                         │   ┌───────────────────────┐
                         │   │  PAGAMENTO            │
                         │   │  Pix · boleto · cartão│
                         │   └───────────┬───────────┘
                         │               │
                         │               ▼
                         │   ┌───────────────────────┐
                         │   │ Aguarda confirmação   │
                         │   │ Pix: segundos         │
                         │   │ Boleto: até 3 dias    │
                         │   │ Cartão: minutos       │
                         │   └───────────┬───────────┘
                         └───────┬───────┘
                                 ▼
                    ┌────────────────────────┐
                    │  INSCRIÇÃO REGISTRADA  │
                    └────────────┬───────────┘
                                 ▼
                    ┌────────────────────────┐
                    │  Análise pela banca    │
                    │  laudo, cota, dados    │
                    └────┬──────────────┬────┘
                deferida │              │ indeferida
                         │              ▼
                         │   ┌──────────────────────┐
                         │   │  Recurso (prazo)     │
                         │   └──────┬───────────┬───┘
                         │   provido│           │negado
                         └──────┬───┘           ▼
                                │          ┌─────────┐
                                ▼          │  FIM    │
                    ┌────────────────────┐ └─────────┘
                    │  HOMOLOGADA        │
                    └─────────┬──────────┘
                              ▼
       ┌──────────────────────────────────────────────┐
       │  ÁREA DO CANDIDATO · daqui até o fim         │
       ├──────────────────────────────────────────────┤
       │  · Cartão de confirmação (local e sala)      │
       │  · Gabarito preliminar                       │
       │  · Recurso de questão                        │
       │  · Gabarito definitivo                       │
       │  · Nota e classificação                      │
       │  · Recurso de resultado                      │
       │  · Convocação                                │
       └──────────────────────────────────────────────┘
```

### A decisão de produto que está escondida aí

**A conta é do candidato, não do certame.** Ele cria uma vez, com CPF, e reaproveita em todos os
certames de todas as bancas que usam a plataforma: dados, documentos, laudo e histórico.

Isso tem três consequências:

1. **Fricção despenca** na segunda inscrição em diante, e candidato de concurso se inscreve em
   dezenas por ano.
2. **Vira efeito de rede pelo lado do candidato**, que é raro num produto vendido para a banca:
   quanto mais bancas usam, mais valiosa a conta; quanto mais candidatos têm conta, mais atraente
   para a próxima banca.
3. **Cria um problema de LGPD que precisa ser resolvido no desenho, não depois:** o laudo médico
   é dado sensível e **não pode vazar de um certame para outro sem consentimento específico.** A
   conta é única; o compartilhamento é por ato do candidato, a cada inscrição.

---

## 4. Fluxo de isenção de taxa

Sub-fluxo próprio, com prazo **anterior** ao da inscrição. É onde a operação da banca mais sofre.

```
   Candidato solicita isenção
             │
             ▼
   ┌─────────────────────────────────┐
   │  Escolhe a hipótese legal       │
   │  · CadÚnico / NIS               │
   │  · Doador de medula             │
   │  · Doador de sangue             │
   │  · Desempregado                 │
   │  · Outras previstas no edital   │
   └──────────────┬──────────────────┘
                  ▼
   ┌─────────────────────────────────┐
   │  Envia comprovação              │
   └──────────────┬──────────────────┘
                  ▼
   ┌─────────────────────────────────┐
   │  FILA DE ANÁLISE                │
   │  · triagem automática por regra │
   │  · análise humana no que sobra  │
   └───────┬─────────────────┬───────┘
      defere                 indefere
           │                 │
           │                 ▼
           │      ┌──────────────────────┐
           │      │ Resultado preliminar │
           │      │ + motivo padronizado │
           │      └──────────┬───────────┘
           │                 ▼
           │      ┌──────────────────────┐
           │      │  Recurso (prazo)     │
           │      └──┬────────────────┬──┘
           │  provido│                │negado
           └────┬────┘                ▼
                │            ┌──────────────────────┐
                ▼            │ Paga a taxa normal   │
   ┌─────────────────────┐   │ ou desiste           │
   │ ISENTO · segue para │   └──────────────────────┘
   │ inscrição sem pagar │
   └─────────────────────┘
```

**Onde está a dor real da banca:** o volume de pedidos de isenção é alto e a análise é manual.
A **triagem automática por regra** (não por IA, e sim por conferência de campo e de documento
obrigatório) é o que transforma uma semana de trabalho em um dia. É a funcionalidade que fecha a
venda.

---

## 5. Máquina de estados da inscrição

Estado explícito é o que permite auditar anos depois, e concurso é judicializado por padrão.

```
                         ┌────────────┐
                         │  RASCUNHO  │
                         └─────┬──────┘
                               │ finaliza
              ┌────────────────┼────────────────┐
       isento │                │ paga           │ cancela
              ▼                ▼                ▼
   ┌────────────────┐  ┌───────────────┐  ┌───────────┐
   │ ISENCAO_ATIVA  │  │  AGUARDANDO   │  │ CANCELADA │
   └───────┬────────┘  │   PAGAMENTO   │  └───────────┘
           │           └───┬───────┬───┘
           │      confirma │       │ vence o prazo
           │               ▼       ▼
           │        ┌──────────┐ ┌──────────┐
           │        │   PAGA   │ │ EXPIRADA │
           │        └────┬─────┘ └──────────┘
           └─────────────┤
                         ▼
              ┌─────────────────────┐
              │     EM_ANALISE      │
              └───┬─────────────┬───┘
          defere  │             │  indefere
                  ▼             ▼
       ┌──────────────┐  ┌──────────────┐
       │  DEFERIDA    │  │ INDEFERIDA   │
       └──────┬───────┘  └──────┬───────┘
              │                 │ recurso provido
              │◄────────────────┘
              ▼
       ┌──────────────┐
       │  HOMOLOGADA  │  ── estado final da fase de inscrição
       └──────────────┘
```

**Regra que não pode ser quebrada:** toda transição registra quem, quando, por qual motivo e sob
qual versão do edital. **Retificação de edital é rotina**, e uma decisão tomada sob a versão 2 não
pode ser reavaliada com a régua da versão 5.

---

## 6. Fluxo de recurso de questão, e o efeito dominó

O recurso mais perigoso, porque **anular uma questão recalcula a nota de todo mundo.**

```
   Gabarito preliminar publicado
             │
             ▼
   ┌─────────────────────────────┐
   │  Janela de recurso aberta   │  ◄── milhares de recursos
   │  1 recurso por questão      │      em poucas horas
   └──────────────┬──────────────┘
                  ▼
   ┌─────────────────────────────┐
   │  AGRUPAMENTO POR QUESTÃO    │
   │  500 recursos sobre a Q42   │
   │  = 1 análise, 500 respostas │
   └──────────────┬──────────────┘
                  ▼
   ┌─────────────────────────────┐
   │  Banca examinadora julga    │
   └───┬──────────┬──────────┬───┘
       │          │          │
   mantém    altera      anula
    gabarito  resposta   questão
       │          │          │
       └────┬─────┴──────────┘
            ▼
   ┌─────────────────────────────┐
   │  GABARITO DEFINITIVO        │
   └──────────────┬──────────────┘
                  ▼
   ┌─────────────────────────────┐
   │  RECÁLCULO EM MASSA         │
   │  toda a base reprocessada   │
   │  nota → cota → desempate    │
   │  → classificação            │
   └──────────────┬──────────────┘
                  ▼
   ┌─────────────────────────────┐
   │  Resultado publicado        │  ◄── toda a base acessa ao mesmo tempo
   └─────────────────────────────┘
```

**O agrupamento por questão é a funcionalidade que vale dinheiro.** Sem ele, a banca lê 500 textos
sobre a mesma questão. Com ele, lê um resumo, decide uma vez e a resposta padronizada vai para os
500, com registro individual, que é o que a lei exige.

---

## 7. Funcionalidades por módulo

### A · Configuração do certame *(banca)*
- Editor estruturado de edital: cargos, vagas, requisitos, escolaridade, taxa por nível
- **Cronograma com dependência entre fases.** Mudou uma data, o sistema mostra o que quebra
- Regras de cota e reserva legal, com percentual e arredondamento configuráveis
- Critérios de pontuação, peso por disciplina, nota de corte, desempate ordenado
- Cidades, locais e capacidade
- **Versionamento e retificação de edital**, com diferença visível entre versões
- Ambiente de simulação antes de publicar

### B · Portal do candidato
- Conta única por CPF, reaproveitável entre certames e bancas
- Inscrição em etapas, com salvamento automático
- Upload de documento com validação de formato e tamanho
- **Acessibilidade WCAG**, que não é opcional: parte do público é PcD por definição
- Área do candidato com linha do tempo do certame
- Notificação por e-mail e mensagem sobre cada mudança de estado
- Segunda via de boleto, alteração de dados dentro do prazo

### C · Pagamento e financeiro *(orquestrado no provedor da banca; o dinheiro não passa por nós)*
- Pix, boleto e cartão
- Baixa automática e conciliação
- **Reembolso em massa**, porque certame suspenso ou anulado acontece
- Prestação de contas da arrecadação para banca e órgão
- Relatório de arrecadação por cargo, cidade e dia

### D · Análise e deferimento *(banca)*
- Fila de isenção com **triagem automática por regra**
- Fila de laudo e atendimento especial
- Deferimento em lote com filtro
- Motivo padronizado, porque texto livre é o que gera nulidade
- Heteroidentificação: agendamento, registro e resultado

### E · Recursos
- Recurso de isenção, inscrição, gabarito, nota e resultado
- **Agrupamento por questão** com resposta padronizada
- Distribuição para examinador com prazo e acompanhamento
- Publicação em bloco e notificação individual

### F · Ensalamento e logística
- Alocação candidato → cidade → local → sala, respeitando capacidade e necessidade especial
- Cartão de confirmação
- Lista de presença, etiqueta, ata e malote
- Mapa de ocupação por local

### G · Correção e resultado
- Importação de folha de resposta
- Cálculo com peso, corte e cota
- **Recálculo em massa** com histórico versionado
- Listas por cargo, cota e cidade
- Convocação e homologação

### H · Transparência e auditoria
- Publicação de atos com carimbo de data e versão
- **Trilha imutável**: quem, quando, o quê, sob qual versão do edital
- Exportação para órgão e auditoria
- Painel público de status do certame

### I · Atendimento
- Chamados vinculados à inscrição
- Base de respostas por fase
- Painel interno de fila e tempo de resposta

---

## 8. O que entra no MVP

Não dá para vender meio certame, porque a banca precisa do ciclo fechado. A saída é **escopo por
complexidade de certame**, não por módulo.

| No MVP | Fora do MVP |
|---|---|
| Certame com prova objetiva apenas | Discursiva, títulos, TAF, prática, psicotécnico |
| Cotas legais padrão | Regra municipal exótica |
| Isenção com triagem por regra | Integração automática com base externa |
| Ensalamento simples por capacidade | Otimização por deslocamento |
| Recurso com agrupamento | Distribuição automática por examinador |
| Trilha de auditoria | Painel público de transparência |

**Alvo do primeiro cliente:** processo seletivo simplificado de prefeitura, ou vestibular de
faculdade privada. Milhares de candidatos, não centenas de milhares. Ciclo curto, risco baixo, e
**um certame inteiro de ponta a ponta**, que é o que vira estudo de caso para vender o próximo.

---

## 9. Análise de recurso em escala, sem IA

### A inversão que resolve o problema

Chegam 500 recursos sobre a questão 42. A pergunta óbvia é como interpretar 500 textos livres.
**A pergunta certa é por que aceitamos texto livre.**

> **Não interprete a entrada. Estruture a entrada.**

O recurso deixa de ser uma caixa de texto e passa a ser um formulário com **fundamento obrigatório
escolhido de uma lista fechada**, derivada do próprio edital:

```
   ┌──────────────────────────────────────────────────┐
   │  RECURSO CONTRA A QUESTÃO 42                     │
   ├──────────────────────────────────────────────────┤
   │  Fundamento  (obrigatório, escolha uma)          │
   │   ( ) Enunciado ambíguo ou incompleto            │
   │   (•) Mais de uma alternativa correta            │
   │   ( ) Nenhuma alternativa correta                │
   │   ( ) Conteúdo fora do programa do edital        │
   │   ( ) Divergência de bibliografia                │
   │   ( ) Erro material · digitação, gráfico, imagem │
   │   ( ) Outro                        ◄── vai para  │
   │                                        leitura   │
   │                                        individual│
   │                                                  │
   │  Alternativa que considera correta:  [ C ]       │
   │  Bibliografia que sustenta:          [_______]   │
   │  Argumentação:                       [_______]   │
   └──────────────────────────────────────────────────┘
```

A partir daí o agrupamento é **exato**, não semelhante:

```
   500 recursos sobre a questão 42
              │
              ▼
   ┌──────────────────────────────────────────┐
   │ [1] TRIAGEM POR REGRA                    │
   │     prazo · legitimidade · formato       │
   │     a questão existe · campos completos  │
   └──────────────────┬───────────────────────┘
                      ▼
   ┌──────────────────────────────────────────┐
   │ [2] AGRUPAMENTO EXATO                    │
   │     chave = questão + fundamento +       │
   │             alternativa apontada         │
   │                                          │
   │     300 → ambíguo                        │
   │     150 → mais de uma correta, aponta C  │
   │      38 → bibliografia divergente        │
   │      12 → OUTRO  ◄── lidos um a um       │
   └──────────────────┬───────────────────────┘
                      ▼
   ┌──────────────────────────────────────────┐
   │ [3] DETECÇÃO DE CÓPIA                    │
   │     similaridade de texto clássica       │
   │     (n-gramas + MinHash) revela os 280   │
   │     recursos idênticos vindos do mesmo   │
   │     cursinho · 1 leitura, não 280        │
   └──────────────────┬───────────────────────┘
                      ▼
   ╔══════════════════════════════════════════╗
   ║ [4] DECISÃO · BANCA EXAMINADORA          ║
   ║     lê 4 grupos + 12 únicos = 16 textos  ║
   ║     mantém · altera · anula              ║
   ╚══════════════════╤═══════════════════════╝
                      ▼
   ┌──────────────────────────────────────────┐
   │ [5] RESPOSTA POR MODELO                  │
   │     um modelo por (fundamento, decisão)  │
   │     escrito uma vez, aplicado a todos    │
   │     do grupo, com registro individual    │
   └──────────────────┬───────────────────────┘
                      ▼
   ┌──────────────────────────────────────────┐
   │ [6] PUBLICAÇÃO + notificação individual  │
   └──────────────────────────────────────────┘
```

**Resultado: o examinador lê 16 textos em vez de 500.** Sem modelo de linguagem, sem GPU, sem
fornecedor de IA no caminho crítico.

### Por que isto é melhor do que usar IA aqui, e não só mais barato

| | Agrupamento estruturado | Agrupamento por IA |
|---|---|---|
| **LGPD art. 20** | Nada é decidido por máquina; a máquina só ordena | Cria direito a revisão de decisão automatizada |
| **Auditoria** | "Foi agrupado porque o candidato marcou o fundamento X", uma **explicação de uma linha** | "O modelo julgou parecido", indefensável em juízo |
| **Erro** | Se o candidato marca errado, o erro é dele e está registrado | Erro de agrupamento é nosso, e é invisível |
| **Transparência no edital** | Nada a declarar | Convém declarar o uso de apoio automatizado |
| **Dependência** | Nenhuma | Provedor externo no dia mais crítico do certame |
| **Custo no pico** | Zero marginal | Por token, exatamente quando o volume explode |

**O risco de enterrar o recurso que está certo desaparece.** Numa clusterização por semelhança, o
único candidato que aponta um erro factual real vira ruído estatístico. Aqui ele marca "outro",
ou marca um fundamento e escreve algo diferente do grupo, o que a detecção de cópia expõe, e
**cai numa fila que é lida uma a uma, por construção.**

### A mesma inversão vale para a isenção

Hipótese legal escolhida de lista fechada → **documento obrigatório específico daquela hipótese** →
conferência por regra (campo presente, data dentro da validade, número no formato correto) →
análise humana só no que a regra não fecha.

Não é reconhecimento de documento. É **formulário bem desenhado**, que é o que a administração
pública faz há décadas quando faz bem.

### O custo honesto desta escolha

O candidato precisa classificar o próprio argumento, e parte vai classificar errado. Três
mitigações, todas de produto:

1. **"Outro" é sempre uma opção visível**, e cai na fila de leitura individual.
2. **Fundamento marcado errado não prejudica o mérito.** Está no edital, e o examinador pode
   reclassificar antes de julgar.
3. **O formulário explica cada fundamento com uma frase**, não com jargão jurídico.

---

## 10. As três decisões, fechadas, e o que cada uma implica

### Decisão 1 · A plataforma vai até o resultado final

Não para na homologação. Ensalamento, correção, classificação e convocação entram.

| Implicação | Consequência |
|---|---|
| **Vira sistema de registro do certame** | O custo de troca deixa de ser o de um formulário e passa a ser o do certame inteiro. Ninguém troca de fornecedor no meio |
| **Ticket sobe** | O valor por inscrição passa a cobrir o certame inteiro, não só o formulário |
| **A exposição jurídica sobe junto** | Erro de classificação é ação judicial, não chamado de suporte. Recálculo versionado deixa de ser recurso e vira obrigação |
| **Fronteira a definir** | Vamos até o **dado**. Malote, impressão e logística física ficam com a banca, porque é operação, não software, e destrói a margem |

### Decisão 2 · Conta única do candidato

Uma conta por CPF, viva entre certames e entre bancas.

**A consequência que muda tudo:** a plataforma passa a ser multi-inquilino em **dois eixos ao
mesmo tempo**. A banca é o inquilino, e o candidato é uma identidade que **atravessa
inquilinos**. Isso é incomum e precisa ser desenhado, não improvisado.

E cria um papel duplo diante da LGPD que precisa estar certo desde o começo:

| Dado | Controlador | Nosso papel |
|---|---|---|
| Inscrição num certame | **A banca** | **Operadora**, tratamos por conta dela |
| A conta em si: cadastro, documentos, histórico | **Nós** | **Controladores** |

> **Regra que decorre disso:** o laudo médico guardado na conta **não vai para a banca
> automaticamente.** O candidato autoriza o compartilhamento **a cada inscrição**, e a autorização
> é registrada. A conveniência é reaproveitar o arquivo; o consentimento não se reaproveita.

Na prática o candidato ganha uma **carteira de documentos** e uma linha do tempo de todos os
certames em que está inscrito, e isso é, sozinho, motivo para criar conta antes mesmo de haver
inscrição aberta.

### Decisão 3 · Vende-se para a banca

E aqui está a vantagem comercial que a rodada de ideação não tinha identificado:

> **Vender para a banca evita licitação.** O órgão licita a banca; a banca contrata seus
> fornecedores como qualquer empresa privada. Compramos um ciclo de venda B2B normal em vez de
> pregão eletrônico.

| Aspecto | Definição |
|---|---|
| **Perfil ideal** | Banca de porte médio, 5 a 50 certames por ano, com sistema próprio envelhecido ou fornecedor que não aguenta o pico |
| **Quem assina** | Sócio ou diretor de operações. Empresa pequena, decisão rápida |
| **O que trava** | O edital de licitação **do órgão** às vezes impõe requisitos de sistema à banca, e precisamos atender antes que ela concorra |
| **Modelo de preço** | Assinatura anual pequena mais **valor por inscrição processada**, e o segundo domina |
| **Por que assim** | Alinha com o modelo da própria banca, que cobra taxa por candidato. **E concentra nossa receita exatamente nos dias em que o custo se concentra** |
| **Entrada** | Um processo seletivo simplificado, ciclo curto, poucos milhares de candidatos. Ele vira o estudo de caso que vende o próximo |

---

## 11. O que ficou aberto

1. **Preço por inscrição.** Falta o número praticado no mercado. É a variável que define o
   negócio inteiro e não dá para estimar por analogia. Tratada com faixa e sensibilidade em
   [03-plano-de-negocios.md](03-plano-de-negocios.md), **explicitamente marcada como não
   verificada**.
2. **Modelo de contrato para a conta única.** Se o candidato é titular e nós somos controladores
   da conta, isso precisa de política própria e de canal de atendimento ao titular. É custo fixo.
3. ~~**Quem mais vende isto para banca de porte médio.**~~ ✅ **Verificado, e o mercado está
   ocupado:** Gestor Editais, ProSeleta, Fábrica de Provas, ADSR, MH Informática e Instituto
   Avalia. O levantamento está em [03-plano-de-negocios.md §7](03-plano-de-negocios.md), e o
   efeito foi mudar a diferenciação de lugar: o argumento deixou de ser "ninguém faz isso" e
   passou a ser "todos fazem a inscrição, quase ninguém sobrevive ao dia do pico".
