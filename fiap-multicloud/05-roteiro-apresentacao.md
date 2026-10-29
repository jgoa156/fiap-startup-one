# 5. Roteiro da apresentação

**Formato:** 1 apresentador · 4 a 6 min de defesa mais 2 a 4 min de discussão.
**Duração alvo: 5 min 45 s** · **capa mais 8 slides** · **fala medida: 822 palavras**, ou 5 min 28 s
a 150 palavras por minuto e 4 min 58 s a 165. Cabe na janela de 4 a 6 minutos nos dois ritmos.

> **Por que a medição importa.** Até 23/09/2026 a fala tinha cerca de 1.490 palavras, perto de
> 10 minutos, e a "régua de 5 min 30 s" era só a divisão do tempo por slide, nunca uma medição do
> texto. Foi reescrita dentro do orçamento de cada slide. O que saiu da fala está no banco de
> perguntas abaixo. **Se reescrever qualquer fala, meça de novo:** 2,5 palavras por segundo.

> **A fala palavra a palavra está em
> [`apresentacao/Certamis-multicloud-roteiro.pdf`](apresentacao/Certamis-multicloud-roteiro.pdf)**,
> com miniatura do slide, texto a dizer e marcação, página por página.
> **Os slides para projetar:** [`apresentacao/Certamis-multicloud-slides.pdf`](apresentacao/Certamis-multicloud-slides.pdf)

**A tese, em uma frase:** *Uma nuvem para operar, duas para sobreviver. Operar em duas não
resolve sete das oito falhas que derrubam este produto, que são nossas, e a segunda nuvem sempre
ligada custaria R$ 1,22 milhão por hora de indisponibilidade evitada. Para a oitava, a queda do
provedor, basta a segunda nuvem desligada, a frio: R$ 80 mil por ano.*

---

## A régua do tempo

| # | Slide | Início | Duração | Papel |
|---|---|---:|---:|---|
| 0 | **Capa:** a pergunta e os quatro passos | 0:00 | 15 s | Nome, a pergunta do trabalho, o caminho. **Não antecipa a resposta** |
| 1 | Certamis: o produto, a nuvem e onde estamos | 0:15 | 55 s | Contexto e os cinco picos |
| 2 | A pergunta certa | 1:10 | 30 s | **A tese, e a reformulação da pergunta** |
| 3 | **O que realmente derruba o sistema** | 1:40 | 50 s | 🔑 **A defesa inteira** |
| 4 | Prós e contras | 2:30 | 50 s | Exigência do enunciado e dados de instituto |
| 5 | A conta, feita na margem | 3:20 | 45 s | **R$ 1,22 mi por hora evitada** |
| 6 | A ressalva, o que sai da AWS | 4:05 | 30 s | Concede o que é multicloud de fato |
| 7 | A arquitetura alvo | 4:35 | 50 s | Diagrama |
| 8 | Fecho | 5:25 | 20 s | Gatilhos que invalidariam a decisão |
| | **Total** | | **5:45** | |

**A fala da capa**, palavra a palavra (36 palavras, cerca de 15 s):

> *Boa noite. Sou [seu nome], e apresento a estratégia de nuvem da Certamis, a empresa do Startup
> One. A pergunta é uma só: ela deve adotar multicloud? Respondo em quatro passos, que estão na
> tela.*

---

## Se estourar, corte nesta ordem

| Ordem | O que | Ganho |
|---|---|---|
| **1º** | No slide 7, dizer só "roxo, verde, azul" e apontar | −18 s |
| **2º** | No slide 1, o parágrafo "onde estamos" vira uma frase: *"hoje são 45 bancas, pós-breakeven, e o conselho perguntou o que acontece se a nuvem cair"* | −15 s |
| **3º** | No slide 4, ler só a linha ✅ dos prós e as duas 🔴 dos contras | −12 s |
| **nunca** | **Slide 2** (a pergunta certa) · **slide 3** (as falhas) · **slide 5** (a conta) · **slide 6** (a ressalva) | |

Sem o 3, o "não" vira preguiça. Sem o 5, vira palpite. Sem o 2, a banca compara coisas erradas.
Sem o 6, parece fuga da pergunta.

---

## Banco de perguntas para a discussão (2 a 4 min)

### Sobre a formulação da pergunta

**"Vocês compararam coisas diferentes. O endurecimento não é alternativa à segunda nuvem."**
> **Correto, e é exatamente por isso que não comparamos.** O endurecimento já está no orçamento,
> custa R$ 900 mil por ano e é o piso: sem ele, a segunda nuvem cai pelo mesmo motivo que a
> primeira. A pergunta que respondemos é a marginal: **sobre o que já existe, vale acrescentar
> R$ 1,22 milhão por ano?** E a resposta é não, porque esse valor compra quatro horas de RTO num
> evento a cada quatro anos.

**"A tabela das falhas é de vocês. Se listassem falha de AZ e de plano de controle, a proporção seria outra."**
> É nossa, sim, e dizemos isso no documento. **Ela não é estatística de mercado, é a lista de
> modos de falha deste produto**, e cada linha corresponde a uma dívida técnica que levantamos em
> §2.3. O argumento não se apoia na contagem, se apoia na origem: **sete dessas oito falhas nascem
> de decisões nossas**, e nenhuma muda de natureza por existir um segundo provedor. E a realidade
> recente é consistente com ela: nos três casos de falha em certame documentados em 2026, Alece,
> AgSUS e FGV, **nenhum foi queda do provedor de nuvem.**

**"A lista original não tinha ataque. A FGV caiu por ataque em fevereiro."**
> Não tinha, e corrigimos. A arquitetura da fase startup não tinha WAF nem proteção contra negação
> de serviço, e isso virou a dívida D9 e a oitava linha da tabela. **A proteção contra ela já
> estava no nosso plano:** WAF na borda, fora da AWS, e backup imutável no Google Cloud. Uma
> segunda nuvem não ajudaria, porque exporia a mesma aplicação ao mesmo ataque.

### Sobre a decisão

**"78% do mercado vai para multicloud. Vocês estão indo contra a maré."**
> Estamos, e o ônus da prova é nosso, por isso o dado está no nosso próprio slide. A diferença é
> que a maioria dessas empresas tem **carga distribuída e falha sem remédio barato**. A nossa é
> **concentrada, agendada e quase sempre remediável reabrindo um prazo**. Multicloud é a resposta
> certa para muita gente; não é para este perfil de falha.

**"Vocês estão economizando e chamando isso de arquitetura."**
> Não, e o número que prova isso é o custo por hora. **A segunda nuvem sempre ligada custaria R$ 1,22 milhão por
> ano e compraria quatro horas de RTO num evento a cada quatro anos: R$ 1,22 milhão por hora de
> indisponibilidade evitada.** Poderíamos pagar, porque a rodada está em caixa. **Não pagamos
> porque o preço por hora é absurdo para uma falha que quase sempre se resolve reabrindo um prazo.**

**"E se a AWS cair justamente no último dia de inscrição?"**
> É o cenário que multicloud cobre, e é real. Três respostas. **Primeira:** o resultado, o gabarito
> e o cartão continuam no ar, porque são arquivos estáticos em CDN, e o candidato ainda consulta.
> **Segunda:** o sistema volta no Google Cloud, em São Paulo, em cerca de oito horas, pelo mesmo
> procedimento que ensaiamos todo trimestre, e a página de status, fora da AWS, avisa a previsão.
> **Terceira:** para as horas em que ficamos fora, existe remédio, e ele é barato, que é reabrir o
> prazo. **Mas não vou dizer que é garantido**, porque não é: no caso Alece, em 2026, a banca reabriu o prazo de
> recurso depois de uma falha, e na mesma Alece o juiz negou reabrir a isenção pedida com base na
> instabilidade. O remédio depende da banca ou do juiz. É por isso que investimos primeiro em não
> cair, que é o que o endurecimento faz.

**"Vocês não estão subestimando o dano reputacional?"**
> Provavelmente sim, e reconhecemos no documento. Valor esperado subestima cauda. Só que o
> argumento não é probabilístico, é de **eficácia**: mesmo aceitando que o dano é maior que a
> conta, a segunda nuvem continua cobrindo uma falha de oito. **Pagar mais pela cobertura errada
> não melhora com o dano sendo maior.**

**"De onde vem esse 1 evento a cada 4 anos?"**
> É estimativa nossa, e está declarada como estimativa. **O argumento não depende da precisão:**
> mesmo dobrando a frequência, para um evento a cada dois anos, o custo cai para R$ 610 mil por
> hora evitada, e a conclusão não muda. Mesmo com um evento **a cada três meses**, seriam uns
> R$ 76 mil por hora, e com essa frequência já estaríamos discutindo trocar de provedor, não somar
> um.

**"Então vocês nunca fariam multicloud?"**
> Já fazemos, na camada de sobrevivência: controle, backup e restauração ficam fora da AWS. O que
> não fazemos é **operar** em duas nuvens. Isso faríamos, e dissemos quando:
> **gatilho 2, se passarmos a aplicar prova on-line.** E isso não é
> hipótese de 2028: a Cebraspe aplicou prova on-line remota para a AgSUS em 2026, e ao menos uma
> aplicação foi anulada e remarcada por falha de sistema. **Numa prova em andamento, o remédio
> deixa de ser reabrir um prazo e passa a ser reaplicar a prova inteira.** O custo de cada hora de
> queda sobe de ordem de grandeza, e a conta pode inverter. Com uma ressalva que o próprio caso
> ensina: na AgSUS a causa foi o controle de acesso da banca, não o provedor. Mesmo ali, o
> primeiro real vai para o endurecimento.

### Sobre a arquitetura

**"Vocês dizem que não dá para exercitar o failover, mas estão construindo um gerador de carga."**
> Boa. **Nós não dizemos isso**, justamente porque seria falso. O gerador existe e exercitaria o
> caminho. O que ele não produz é **memória operacional**: num time de cinco pessoas, um
> procedimento que roda uma vez a cada quatro anos é escrito por quem não estará lá quando for
> usado, envelhece a cada mudança de esquema e de IAM, e será executado sob pressão de madrugada.
> **Ensaio prova que o caminho funcionava. Não prova que o plantão sabe operá-lo.** E a
> restauração no Google não cai nisso: ela é o próprio teste trimestral, que o plantão executa
> quatro vezes por ano.

**"Por que não multirregião na AWS, que é mais simples que multicloud?"**
> A AWS opera **uma região no Brasil**, São Paulo, com 3 AZs. O Rio de Janeiro é *Local Zone*, que
> depende da região-mãe e não isola falha. Mas o ponto é outro: **multirregião também não ataca as
> sete primeiras falhas.** Ela morre pelo mesmo argumento, não pela geografia. E para restaurar
> preferimos o Google em São Paulo a uma região da AWS no exterior: mantém o dado no país e
> sobrevive a uma falha da AWS inteira.

**"Escalonamento por calendário não é frágil? E se o edital mudar?"**
> Retificação de edital é rotina, e por isso o cronograma é **dado versionado do produto**, não
> configuração de infraestrutura. Mudou a data, o provisionamento recalcula. E há uma rede de
> segurança: o autoescalonamento reativo continua ligado por baixo. **A agenda é o piso, não o
> teto.**

**"Publicar resultado como arquivo estático não quebra a personalização?"**
> A lista de classificação é pública e igual para todos, então é o caso fácil. O que é individual,
> como a nota detalhada do candidato, continua dinâmico, mas é **uma fração do volume**: quase
> todo mundo abre a lista, poucos abrem o detalhe. Movemos o pico, não o produto inteiro.

**"Célula dedicada por certame grande não desperdiça capacidade?"**
> Ela é **alocada por evento e devolvida depois**, então existe pelas semanas do certame, não pelo
> ano. E é vendável: a banca que roda um certame de 300 mil candidatos paga por isolamento
> garantido, e isso vira cláusula. **É a única linha do programa que se paga sozinha.**

**"Com 5 pessoas de plataforma, vocês conseguem operar isso?"**
> Sim, e é exatamente por isso que rejeitamos a segunda nuvem sempre ligada. A operação usa
> serviços gerenciados de um provedor só, e a restauração no Google usa o mesmo Kubernetes e o
> mesmo PostgreSQL, dentro do time atual. **A nuvem sempre ligada exigiria mais dois engenheiros,
> que sozinhos são R$ 660 mil dos R$ 1,22 milhão, e contratar engenheiro de plataforma sênior no
> Brasil leva de 4 a 6 meses, não uma assinatura.**

**"O que exatamente significa 40× o pico?"**
> **40 vezes a média do próprio certame, não a média anual da empresa.** Num certame de 300 mil
> candidatos com janela de 30 dias, a média é de 417 inscrições por hora, e o ensaio prova
> **16,7 mil por hora**, cerca de 4,6 por segundo, sustentadas por três horas. A média anual da
> empresa inteira é de 550 por hora, então o pico de um único certame grande é trinta vezes a
> operação normal.

**"Backup e restauração em outro provedor não é multicloud pela metade?"**
> É, e nós dizemos isso no slide 6, de propósito. Pelo critério estrito do enunciado, **sim, é
> multicloud**, só que na camada de sobrevivência. A distinção que fazemos é entre **operar** e
> **sobreviver**: a operação fica na AWS; DNS, status, backup e a restauração de emergência saem.
> É a porção que custa 5% do EBITDA e resolve dois problemas: *"não consigo nem avisar que caí"* e
> *"não tenho onde voltar"*.

**"Por que o Google Cloud, e não outra região da AWS?"**
> Avaliamos as duas. Outra região da AWS custaria menos, uns R$ 35 mil por ano contra R$ 80 mil,
> e usaria o mesmo ferramental. Mas tiraria o dado dos candidatos do Brasil, e não cobriria uma
> falha da AWS inteira nem uma conta comprometida por ataque. O Google Cloud tem região em São
> Paulo e já guarda o nosso backup. **Pagamos R$ 45 mil a mais por ano para cobrir mais falhas e
> manter o dado no país.** O Azure, com região no estado de São Paulo, seria equivalente.

**"Se a segunda nuvem desligada se paga, por que não a sempre ligada?"**
> Porque as duas compram horas muito diferentes. A desligada custa R$ 80 mil por ano e tira o
> sistema de uma queda longa em cerca de oito horas: numa queda de três dias, são 64 horas
> evitadas, a uns R$ 5 mil por hora. A sempre ligada custaria R$ 1,22 milhão a mais e só
> encurtaria essas oito horas para quatro. Por evento, a desligada custa menos que o contrato
> anual de uma banca média; a ligada, o de quase 13 bancas. **A diferença de preço não é erro de
> conta: é o argumento.**

### Dados, conformidade e operação

**"De onde vêm os números da Forrester?"**
> *State of Cloud Strategy Survey 2024*, da HashiCorp com a Forrester Consulting, junho de 2024,
> cerca de 1.200 profissionais e decisores em empresas de 1.000+ funcionários, global. Usamos: 8%
> de alta maturidade, 64% com lacuna de competências, 91% relatando desperdício, 78% em multicloud
> e 79% das maduras relatando melhora de segurança.

**"O 79% que diz que multicloud melhora a segurança não joga contra vocês?"**
> Joga, e por isso está no nosso documento. Só que o recorte é **"organizações de alta
> maturidade"**, os mesmos 8% do outro dado. **Nós não somos esses 8%**, e uma segunda nuvem
> operada sem maturidade aumenta a superfície em vez de melhorar a defesa.

**"A premissa de que a falha tem remédio tem fonte primária?"**
> Tem, e ela corta nos dois sentidos, por isso a citamos inteira. É o **concurso da Alece**, no
> Ceará, em 2026. Em maio, candidatos não conseguiram pedir isenção por instabilidade no site. O
> Ministério Público entrou com ação civil pública, e o juiz da 13ª Vara da Fazenda Pública de
> Fortaleza **negou** reabrir o período de isenção com base na instabilidade. Em setembro, uma
> segunda falha, no acesso ao resultado, levou a própria banca a **reabrir** o prazo de recurso,
> sem ação judicial. **Um remédio negado e um concedido, no mesmo certame.** Por isso dizemos que
> o remédio é praticado, mas não garantido.

**"Então a premissa de vocês ficou mais fraca?"**
> Ficou mais modesta, e mais verdadeira. Antes dizíamos que ninguém perde o direito de concorrer,
> e isso era forte demais. O que sustentamos agora é que a falha em fase de prazo **quase sempre**
> tem um remédio barato. **E o argumento que decide não depende dela:** sete das oito falhas são
> nossas, e a segunda nuvem não resolve nenhuma delas, com ou sem remédio.

**"A Forrester prevê pelo menos dois grandes incidentes de nuvem de vários dias em 2026. Isso não derruba o argumento de vocês?"**
> Não, e foi essa previsão que nos fez fechar o furo que ela exporia. A previsão está no
> *Predictions 2026* da Forrester. Se nós não tivéssemos onde restaurar, uma queda de três dias
> faria a segunda nuvem sempre ligada comprar 68 horas, e o custo cairia para uns R$ 72 mil por hora. Por
> isso a restauração a frio no Google Cloud faz parte do plano. **Com ela, voltamos em cerca de
> oito horas qualquer que seja a duração da queda**, e a nuvem sempre ligada compra sempre as mesmas
> quatro horas, a R$ 1,22 milhão cada.

**"E se a região ficar 72 horas fora, onde vocês restauram?"**
> No Google Cloud, região de São Paulo, onde o backup já fica. Nada fica ligado lá: há o backup, a
> infraestrutura como código e a cota de máquinas pedida antes. O sistema sobe em modo de
> recuperação, com inscrição, pagamento e consulta, e boleto e PDF gerados mais devagar. **É o
> mesmo procedimento do ensaio trimestral**, então o plantão já o executou quatro vezes no ano.

**"E se todo mundo correr para o Google ao mesmo tempo e faltar máquina?"**
> É um risco real, e está declarado. Pedimos a cota de máquinas antecipadamente, e cada ensaio
> trimestral comprova que ela existe. **E as oito horas são estimativa até o primeiro ensaio
> medi-las:** se um ensaio passar de oito horas, é o gatilho 3, e a conta é refeita.

**"Como vocês provam o ensaio de carga para um cliente?"**
> Relatório assinado, contra a mesma configuração que vai para produção, anexado ao contrato antes
> de cada certame acima de 50 mil candidatos. **Nenhum concorrente anuncia isso**, e é a única
> coisa que responde à pergunta que a banca realmente faz, que é *"você aguenta o meu último
> dia?"*.

**"E a LGPD? Há dado sensível aí."**
> Há: laudo médico para atendimento especial e autodeclaração racial. Dois pontos do desenho
> respondem. **Primeiro:** somos operadora da inscrição, porque a banca é controladora, mas
> **controladores da conta do candidato**, o que exige política própria e canal de atendimento ao
> titular. **Segundo:** o documento guardado na conta não vai para a banca automaticamente; o
> candidato autoriza a cada inscrição. **O arquivo se reaproveita, o consentimento não.**

**"Vocês processam o pagamento da taxa?"**
> Não, e é decisão deliberada. **Taxa de inscrição é receita pública**, e há entendimento de
> tribunal de contas contra delegar a gestão desse dinheiro a empresa privada. Orquestramos a
> cobrança no provedor da própria banca e recebemos a confirmação. Isso remove uma camada inteira
> de exigência regulatória.

**"Qual o risco maior deste negócio?"**
> Não é técnico. É que **o mercado já está ocupado**, com Gestor Editais, ProSeleta, Fábrica de
> Provas e ADSR. Nosso argumento não é "ninguém faz isso". É **"todos fazem a inscrição, quase
> ninguém sobrevive ao dia do pico e ao recurso em massa"**, e é exatamente por isso que o
> programa de arquitetura é o produto, não o custo.
