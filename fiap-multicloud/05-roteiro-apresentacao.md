# 5. Roteiro da apresentação

**Formato:** 1 apresentador · 4 a 6 min de defesa + 2 a 4 min de discussão.
**Duração alvo: 5 min 30 s** · **8 slides**

> **A fala palavra a palavra está em
> [`apresentacao/Lisura-multicloud-roteiro.pdf`](apresentacao/Lisura-multicloud-roteiro.pdf)** —
> miniatura do slide, texto a dizer e marcação, página por página.
> **Os slides para projetar:** [`apresentacao/Lisura-multicloud-slides.pdf`](apresentacao/Lisura-multicloud-slides.pdf)

**A tese, em uma frase:** *Não à arquitetura multicloud. Sim ao controle e ao backup fora do
provedor primário — porque multicloud cobre 1 das 7 causas reais de indisponibilidade.*

---

## A régua do tempo

| # | Slide | Início | Duração | Papel |
|---|---|---:|---:|---|
| 1 | Lisura — o produto, a nuvem e onde estamos | 0:00 | 55 s | Contexto e os cinco picos |
| 2 | A pergunta e a resposta | 0:55 | 30 s | **A tese, dita antes de defendida** |
| 3 | **O que realmente derruba o sistema** | 1:25 | 50 s | 🔑 **A defesa inteira** |
| 4 | Prós e contras | 2:15 | 50 s | Exigência do enunciado + dados de instituto |
| 5 | Mesmo dinheiro, proteção diferente | 3:05 | 45 s | Mata o "vocês economizaram" |
| 6 | A ressalva — o que sai da AWS | 3:50 | 30 s | Concede o que é multicloud de fato |
| 7 | A arquitetura alvo | 4:20 | 50 s | Diagrama |
| 8 | Fecho | 5:10 | 20 s | Gatilhos que invalidariam a decisão |
| | **Total** | | **5:30** | |

---

## Se estourar, corte nesta ordem

| Ordem | O que | Ganho |
|---|---|---|
| **1º** | No slide 7, dizer só "roxo, verde, azul" e apontar | −18 s |
| **2º** | No slide 1, o parágrafo "onde estamos" vira uma frase: *"hoje são 72 bancas, pós-breakeven, e o conselho perguntou o que acontece se a nuvem cair"* | −15 s |
| **3º** | No slide 4, ler só a linha ✅ dos prós e as duas 🔴 dos contras | −12 s |
| **nunca** | **Slide 2** (a tese) · **slide 3** (as 7 causas) · **slide 5** (a conta) · **slide 6** (a ressalva) | — |

Sem o 3, o "não" vira preguiça. Sem o 5, vira economia. Sem o 6, parece fuga da pergunta.

---

## Banco de perguntas para a discussão (2 a 4 min)

### Sobre a decisão

**"78% do mercado vai para multicloud. Vocês estão indo contra a maré."**
> Estamos, e o ônus da prova é nosso — por isso o dado está no nosso próprio slide. A diferença é
> que a maioria dessas empresas tem **carga distribuída e falha irreversível**. A nossa é
> **concentrada, agendada e reversível**. Multicloud é a resposta certa para muita gente; não é
> para este perfil de falha.

**"Vocês estão economizando e chamando isso de arquitetura."**
> Não. **As duas opções custam o mesmo:** R$ 1,66 milhão contra R$ 1,69 milhão por ano — está no
> slide 5. E com R$ 80 milhões em caixa, poderíamos pagar as duas. **Não fazemos porque
> compraríamos a proteção errada.**

**"E se a AWS cair justamente no último dia de inscrição?"**
> É o cenário que multicloud cobre, e é real. Três respostas. **Primeira:** o resultado, o gabarito
> e o cartão continuam no ar, porque são arquivos estáticos em CDN — o candidato ainda consulta.
> **Segunda:** a página de status está fora da AWS, e é por ela que se comunica formalmente a
> prorrogação. **Terceira, e a mais importante:** existe remédio. O prazo se prorroga. É caro em
> reputação, mas **ninguém perde o direito de concorrer** — e é exatamente isso que não seria
> verdade num sistema de pagamento.

**"Vocês não estão subestimando o dano reputacional?"**
> Provavelmente sim, e reconhecemos no documento. Valor esperado subestima cauda. Só que o
> argumento não é probabilístico — é de **eficácia**: mesmo aceitando que o dano é maior que a
> conta, multicloud continua cobrindo 1 das 7 causas. **Pagar mais pela cobertura errada não
> melhora com o dano sendo maior.**

**"Então vocês nunca fariam multicloud?"**
> Faríamos, e dissemos quando: **gatilho 2.** Quando a prova passar a ser aplicada online em
> escala — a Lei 14.965/2024 autoriza a partir de 2028 — **a falha deixa de ser reversível.** Não
> se prorroga uma prova em andamento. Naquele dia a resposta certa é a oposta desta, e o plano já
> prevê.

### Sobre a arquitetura

**"Por que não multirregião na AWS, que é mais simples que multicloud?"**
> A AWS opera **uma região no Brasil** — São Paulo, 3 AZs. O Rio de Janeiro é *Local Zone*, que
> depende da região-mãe e não isola falha. Mas o ponto é outro: **multirregião também não ataca as
> seis primeiras causas.** Ela morre pelo mesmo argumento, não pela geografia.

**"Escalonamento por calendário não é frágil? E se o edital mudar?"**
> Retificação de edital é rotina, e por isso o cronograma é **dado versionado do produto**, não
> configuração de infraestrutura — mudou a data, o provisionamento recalcula. E há uma rede de
> segurança: o autoescalonamento reativo continua ligado por baixo. **A agenda é o piso, não o
> teto.**

**"Publicar resultado como arquivo estático não quebra a personalização?"**
> A lista de classificação é pública e igual para todos — é o caso fácil. O que é individual, como
> a nota detalhada do candidato, continua dinâmico, mas é **uma fração do volume**: quase todo
> mundo abre a lista, poucos abrem o detalhe. Movemos o pico, não o produto inteiro.

**"Célula dedicada por certame grande não desperdiça capacidade?"**
> Ela é **alocada por evento e devolvida depois** — existe pelas semanas do certame, não pelo ano.
> E é vendável: a banca que roda um certame de 300 mil candidatos paga por isolamento garantido, e
> isso vira cláusula. **É a única linha do programa que se paga sozinha.**

**"Com 6 pessoas de plataforma, vocês conseguem operar isso?"**
> Sim — e é exatamente por isso que rejeitamos a segunda nuvem. O programa que escolhemos é feito
> de serviços gerenciados de um provedor só. **A segunda nuvem exigiria mais dois engenheiros, e
> contratar engenheiro de plataforma sênior no Brasil leva de 4 a 6 meses, não uma assinatura.**

**"Backup off-site em outro provedor não é multicloud pela metade?"**
> É — e nós dizemos isso no slide 6, de propósito. Pelo critério estrito do enunciado, **sim, é
> multicloud parcial.** A distinção que fazemos é entre **plano de dados** e **plano de controle**:
> a aplicação e o banco ficam na AWS; DNS, status e backup saem. É a porção que custa 3% do EBITDA
> e resolve o problema de *"não consigo nem avisar que caí"*.

### Dados, conformidade e operação

**"De onde vêm os números da Forrester?"**
> *State of Cloud Strategy Survey 2024*, da HashiCorp com a Forrester Consulting, junho de 2024,
> cerca de 1.200 profissionais e decisores em empresas de 1.000+ funcionários, global. Usamos: 8%
> de alta maturidade, 64% com lacuna de competências, 91% relatando desperdício, 78% em multicloud
> e 79% das maduras relatando melhora de segurança.

**"O 79% que diz que multicloud melhora a segurança não joga contra vocês?"**
> Joga, e por isso está no nosso documento. Só que o recorte é **"organizações de alta
> maturidade"** — os mesmos 8% do outro dado. **Nós não somos esses 8%**, e uma segunda nuvem
> operada sem maturidade aumenta a superfície em vez de melhorar a defesa.

**"Como vocês provam o ensaio de carga para um cliente?"**
> Relatório assinado, contra a mesma configuração que vai para produção, a 40× a média, anexado ao
> contrato antes de cada certame acima de 50 mil candidatos. **Nenhum concorrente anuncia isso** —
> e é a única coisa que responde à pergunta que a banca realmente faz, que é *"você aguenta o meu
> último dia?"*.

**"E a LGPD? Há dado sensível aí."**
> Há: laudo médico para atendimento especial e autodeclaração racial. Dois pontos do desenho
> respondem. **Primeiro:** somos operadora da inscrição — a banca é controladora — mas
> **controladores da conta do candidato**, o que exige política própria e canal de atendimento ao
> titular. **Segundo:** o documento guardado na conta não vai para a banca automaticamente; o
> candidato autoriza a cada inscrição. **O arquivo se reaproveita, o consentimento não.**

**"Vocês processam o pagamento da taxa?"**
> Não, e é decisão deliberada. **Taxa de inscrição é receita pública**, e há entendimento de
> tribunal de contas contra delegar a gestão desse dinheiro a empresa privada. Orquestramos a
> cobrança no provedor da própria banca e recebemos a confirmação. Isso remove uma camada inteira
> de exigência regulatória.

**"Qual o risco maior deste negócio?"**
> Não é técnico. É que **o mercado já está ocupado** — Gestor Editais, ProSeleta, Fábrica de
> Provas, ADSR. Nosso argumento não é "ninguém faz isso"; é **"todos fazem a inscrição, quase
> ninguém sobrevive ao dia do pico e ao recurso em massa"** — e é exatamente por isso que o
> programa de arquitetura é o produto, não o custo.
