# POC Lisura

Escopo e cronograma: [`../04-escopo-poc.md`](../04-escopo-poc.md). Entregável final: o relatório
do ensaio de carga que testa a hipótese **H1** (40× a carga média por 3 h, sem perder inscrição,
sem cobrar duas vezes, dentro do custo previsto).

Duas pastas: **`poc/`** (esta — API em Go, banco, PSP simulado) e **[`../poc-web/`](../poc-web/README.md)**
(o front em Next.js).

## Estado

| Semana | O quê | Status |
|---|---|---|
| **1** | Modelo de dados do certame · máquina de estados · trilha de auditoria | ✅ 29/10/2026 |
| **2** | Fluxo de inscrição ponta a ponta · conta, cargo, cota, upload, isenção | ✅ 29/10/2026 |
| **front** | **Portal do candidato em Next.js** — layout do Inclusa, paleta Grafite (fora do cronograma original) | ✅ 04/11/2026 — fluxo completo percorrido no navegador |
| **3** | Pagamento em sandbox, idempotência, conciliação | ✅ 04/11/2026 — 12 testes de pagamento, dois conferidos contra o defeito |
| **4** | Geração assíncrona · publicação estática | ✅ 04/11/2026 — fila, cobrança assíncrona, cartão e relação estáticos; 11 testes novos |
| **5** | Fila de espera · modo degradado | ✅ 04/11/2026 — sala de espera e 3 níveis, 9 testes novos; os 3 níveis percorridos no navegador |
| **6** | IaC · gerador de carga | ✅ 05/11/2026 — Terraform validado (não aplicado: sem credencial AWS), k6 + auditoria rodados localmente; 10 testes novos; defeitos achados e corrigidos (um, só no Windows local, atenuado) |
| **picos 4 e 5** | Gabarito, recurso por questão, resultado (nota, cotas, desempate, classificação) e publicação estática; trilha encadeada; migrações testadas nos dois sentidos; `-race` | ✅ 05/11/2026 — 26 testes novos; **sem front** |
| **trilha, fila e relatório** | Âncora da trilha (completa por dia, incremental por hora) num cofre que não sobrescreve (S3 com bloqueio de objeto); prioridade na fila; homologação em lote medida com 100 mil; **gerador do relatório do H1** (`cmd/relatorio`), validado com uma rodada local real | ✅ 05/11/2026 — 21 testes novos, quatro conferidos contra o defeito |
| **acima de 640×** | Orquestrador do ensaio (`infra/scripts/ensaio.ps1`); PSP simulado que sobrevive a reinício (diário); 1.280× local com CPU medida; Pix pago uma vez só; conciliação em janelas escalonadas até 7 dias; auditoria com "cobrada duas vezes" e "pago a mais sem reembolso" | ✅ 05/11/2026 — 640× passa limpo; 1.280× é limite **desta máquina** (CPU a 100%); 5 testes novos, todos conferidos contra o defeito |
| **devolução** | O que entrou a mais volta, um pagamento por vez: Pix para quem pagou, boleto para a chave CPF do candidato (titular conferido); nunca duas vezes; tarefa abandonada é retomada; auditoria com 4 conferências novas | ✅ 05/11/2026 — 906 devoluções retroativas no banco local, 868 concluídas em 0,48 s; 7 testes novos, quatro conferidos contra o defeito |
| **chave Pix da devolução** | O candidato sem chave CPF informa outra chave dele; formato conferido como o DICT guarda; titular conferido pelo prestador; a chave não muda depois de pedida | ✅ 05/11/2026 — 3 testes novos, um deles conferido contra o defeito duas vezes, e o teste do prestador ampliado |
| **exportação** | Pacote do certame para o órgão e a auditoria (CSV por assunto, âncoras do cofre, manifesto com SHA-256 e selo, sem dado pessoal); conferência sem banco: arquivos, corrente, selos, âncoras e dias fechados refeitos | ✅ 06/11/2026 — 5 testes que adulteram e refazem o manifesto, quatro conferidos contra o defeito; um defeito de colação achado e corrigido no fechamento do dia |
| **avisos ao candidato** | E-mail a cada mudança de estado que importa, na mesma transação do evento, com prioridade de lote; um por evento, nunca dois; prova imutável com SHA-256 do texto; endereço recusado registrado; caixa de saída local e descarte no ensaio | ✅ 06/11/2026 — 6 testes novos, três conferidos contra o defeito; medição sob carga abaixo |
| **cronograma** | 8 regras de dependência entre fases (da isenção ao resultado, inclusive a margem de compensação do boleto); retificação que recusa o que quebra, vira versão nova do edital e republica o certame; simulação sem gravar | ✅ 06/11/2026 — 7 testes novos, quatro conferidos contra o defeito |
| **ensalamento** | Locais e salas por CSV; cada homologada numa sala da sua cidade, com capacidade e atendimento especial (individual, especial, acessível, amamentação); reensalar mantém quem tinha sala; ato publicado com SHA-256; o banco recusa sala lotada; cartão com local e sala; mapa e listas de presença | ✅ 06/11/2026 — 150 mil alocadas em 14,3 s, 0 problemas, 150 mil cartões republicados em 5 min 14 s sem falha; 8 testes novos, quatro conferidos contra o defeito |
| **prestação de contas** | Arrecadação por dia, cargo e cidade; a conta provada (entrou − saiu − a devolver − em divergência = taxas válidas); conferência contra o extrato do prestador nos dois sentidos; dia fechado imutável, com resumo SHA-256 | ✅ 05/11/2026 — certame 5: 343.756 pagamentos, R$ 34,4 mi, a conta fecha, 0 problemas em 2,7 s; certame cancelado: entrou = saiu; 5 testes novos, quatro conferidos contra o defeito |
| **reembolso em massa** | A banca anula o certame: portas fechadas, toda inscrição viva cancelada com trilha (inclusive a homologada), todo pagamento confirmado devolvido em lote; retomável sem repetir | ✅ 05/11/2026 — 28.831 inscrições canceladas e 24.885 pagamentos devolvidos em 41,9 s, auditoria limpa; 3 testes novos, um conferido contra o defeito duas vezes |
| **SES** | Os avisos pelo Amazon SES: a taxa espaçada na cota real da conta, lida na subida; recusa definitiva separada da passageira; a chave em cabeçalho e etiqueta; no Terraform, `email_de` liga o SES com duas permissões; a garantia vira "pelo menos um aviso", dita e provada por teste | ✅ 06/11/2026 — 4 testes contra um SES falso local, três conferidos contra o defeito; **não verificado contra a AWS de verdade** |
| **documentação viva e observabilidade** | `/docs/api` (Scalar sobre o `openapi.yaml` conferido contra as rotas), `/design` (Storybook com fundamentos, Ant Design com o tema, componentes e telas em todos os estados), `/docs/infra` (gerada do Terraform), `/operacao` (dispara ensaios e mostra o Grafana); `/metrics` na API; Prometheus e Grafana locais; painel do CloudWatch no Terraform | ⚠️ 06/11/2026 — 10 testes novos, seis conferidos contra o defeito; `/docs`, `/docs/api`, `/design` e `/docs/infra` percorridas no navegador. **`/operacao` quebrou no navegador** (histórico vazio vinha `null`); corrigida no código com teste, **não conferida de novo**, e nenhum ensaio foi disparado por ela. O desenvolvimento foi interrompido pelo usuário nesse ponto |
| **devolução de e-mail e telas** | Devolução e reclamação dos avisos pelo SNS, com a assinatura conferida; endereço suprimido; o aviso que voltou na área do candidato, que troca o e-mail ali; a exportação leva as ocorrências sem o endereço. No front: devoluções com a chave Pix, e avisos com a prova. No PSP simulado: DICT de teste por HTTP | ✅ 06/11/2026 — 6 testes novos e 2 ampliados, seis conferidos contra o defeito; as telas percorridas no navegador (chave inválida, chave inexistente, chave certa até "Devolvido"; aviso devolvido até o e-mail novo) |
| **relação em fatias** | A relação de inscritos sai em fatias de 10 mil números, com um índice; a mudança de uma inscrição refaz só a fatia dela; as fatias ficam anotadas numa tabela (migração 00019), e o índice não depende do planejador; a leitura da página estática no Windows não falha mais na troca do arquivo | ✅ 06/11/2026 — certame 5: fatia em 50 ms em média (antes 3,0 s a relação inteira, 32,7 MB), índice em 0,054 ms (antes 271 ms); 3 testes novos ou ampliados, todos conferidos contra o defeito |
| 7 | Ensaio 10× / 20× / 40× | — |
| 8 | Relatório | — |

**Testes Go: 187** (199 com os subtestes), todos passando, também com o detector de corrida (06/11/2026, compilação final). O front passa na checagem de tipos, no lint, no build do Storybook e no build de produção.

**Dependências:** todo módulo tem pelo menos 2 semanas de publicado e é conferido contra o checksum
público do Go (`go mod verify`: "all modules verified"; o mais novo é de 22/10/2026). Em 06/11 o SDK
da AWS **não cumpria isso**: o núcleo, `config`, `credentials` e `s3` tinham entrado em 05/11 com as
versões de 04/11, um dia depois de publicadas. Eles foram rebaixados para a leva de 22/10, e o S3, o
cofre e a exportação continuaram passando. Sem banco a suíte **falha** (não pula em silêncio);
para pular de propósito, `LISURA_SEM_BANCO=1`.

Outras pastas: **[`../infra`](../infra/README.md)** (Terraform, imagens, scripts de operação) e
**[`../carga`](../carga/README.md)** (o roteiro do k6 e o que "40×" quer dizer em número).

## A stack, e por que esta

| Camada | Tecnologia |
|---|---|
| API | **Go 1.27**, `net/http` da biblioteca padrão |
| Banco de dados | **PostgreSQL 17** (Aurora PostgreSQL na produção) |
| Acesso ao banco · migrações | **pgx v5** · **goose v3** — SQL escrito à mão, sem ORM |
| Front | **Next.js 16 + React 19 + Ant Design 6** — ver [`../poc-web`](../poc-web/README.md) |
| Sessão | Token assinado (HMAC-SHA256): cabeçalho `Authorization` para scripts e ensaio de carga; cookie `HttpOnly` + `SameSite=Lax` para o front |
| Pagamento | PSP simulado (`cmd/psp-simulado`) falando HTTP, com os defeitos de um real |
| Arquivos enviados | Disco local na POC; S3 na arquitetura alvo, pela mesma interface |

### Por que Go

A escolha foi feita contra o problema, não contra o que estava instalado. Os critérios saem de H1
e da arquitetura alvo (`../../fiap-multicloud/04-arquitetura-alvo.md`):

| Critério | Por que importa aqui | Go | .NET 10 | Java 21 | Node | Python |
|---|---|---|---|---|---|---|
| Partida a frio em Lambda | A geração assíncrona roda em **SQS + Lambda** | ✅ dezenas de ms | ⚠️ boa só com AOT | ❌ sem SnapStart | ✅ | ✅ |
| Memória por pod | **Custo por hora de pico** é métrica do ensaio | ✅ | ✅ | ⚠️ | ⚠️ | ❌ |
| Imagem e partida do contêiner | Escalonamento **por agenda** no EKS | ✅ binário estático | ⚠️ | ⚠️ | ⚠️ | ⚠️ |
| Vazão de I/O por núcleo | 40× com o mesmo hardware | ✅ | ✅ | ✅ | ⚠️ | ❌ |
| SQL explícito | Provar "zero cobrança duplicada" exige ver a transação | ✅ pgx | ✅ Npgsql | ✅ | ✅ | ✅ |
| 1 pessoa, 8 semanas | Linguagem pequena, compilação rápida | ✅ | ✅ | ⚠️ | ✅ | ✅ |

**Honestidade sobre a margem:** Go e .NET ficam próximos; Go vence pela partida a frio em Lambda e
pelo binário pequeno. Nada disso decide H1 — **quem decide H1 é o desenho do banco, o pool de
conexões, a geração assíncrona e a publicação estática.**

### Por que um PSP simulado, e não o sandbox de um real

1. Criar conta e lidar com chave de API de um prestador real é coisa que o usuário faz, não o assistente.
2. O ensaio de carga dispara milhares de pagamentos por minuto; sandbox real tem limite de taxa e
   termos de uso que proíbem isso. O simulador aguenta e **provoca de propósito** os defeitos que
   importam: aviso repetido, aviso perdido, aviso atrasado, resposta perdida depois de criar.

O cliente fala com ele por HTTP como falaria com um prestador real (`internal/psp/cliente.go`);
trocar é escrever só esse adaptador.

## Como rodar

São três processos. Em cada um, a partir de `poc/`:

```powershell
powershell -ExecutionPolicy Bypass -File scripts\db-up.ps1   # Postgres local (cria na 1ª vez)
go run ./cmd/seed                                            # um certame sintético
```

```powershell
$env:LISURA_TOKEN_KEY = 'qualquer-chave-local'; $env:LISURA_SANDBOX = '1'
$env:LISURA_OPERACAO_SEGREDO = 'operacao-local'              # libera a troca de nível (semana 5)
go run ./cmd/lisura                                          # API em :8082
```

```powershell
go run ./cmd/psp-simulado -diario dados\psp-diario.jsonl      # PSP de mentira em :8081
```

Com `-diario`, o PSP simulado grava cada cobrança e cada pagamento num arquivo e o relê ao subir,
como um prestador real que reinicia sem esquecer nada. Sem ele, reiniciar apagava as cobranças, e o
candidato não conseguia mais pagar. Apagava também o extrato, e a conciliação passava a acusar
divergências falsas. **Conferido contra o defeito:** sem anotar a cobrança no diário, os dois testes
de reinício falham. **Limite na AWS:** o disco de uma tarefa Fargate não sobrevive à troca da
tarefa, então lá o diário só vale enquanto a mesma tarefa estiver viva.

E o front, a partir de `../poc-web/`: `npm run dev -- -p 8080` → **http://localhost:8080**.

Testes: `go test ./...` (cada pacote recria o próprio schema em `lisura_test`).

**Documentação viva e observação em tempo real** (desde 06/11; ver a seção "Documentação viva e
observabilidade"):

```powershell
..\observabilidade\subir-observabilidade.ps1 -SegredoOperacao operacao-local -TokenKey qualquer-chave-local
```

Sobe o Prometheus (127.0.0.1:9090), o Grafana (127.0.0.1:3000) e o executor de ensaios
(127.0.0.1:8083). Depois, no front de produção: **/docs** (a entrada), **/docs/api**, **/design**,
**/docs/infra** e **/operacao**.

## A API (em `/api`)

| Rota | Para quê |
|---|---|
| `POST /contas` · `POST /sessoes` · `POST /sessoes/sair` · `GET /eu` | Conta única por CPF, entrar, sair, quem sou eu |
| `PUT /conta` · `GET`/`POST /conta/documentos` | Dados pessoais · carteira de documentos (laudo, comprovante) |
| `GET /certames` · `GET /certames/{id}` | Certames abertos · dados públicos (cargos, cidades, cotas, cronograma) |
| `GET`/`POST /certames/{id}/isencao` | Meu pedido de isenção · pedir, com triagem por regra |
| `GET /inscricoes` · `POST /certames/{id}/inscricoes` | Minhas inscrições · abrir (idempotente) |
| `GET /inscricoes/{id}` · `/pendencias` · `/trilha` | Revisão · o que falta · histórico |
| `PUT /inscricoes/{id}/cargo` · `/cidade` · `/cotas` · `/atendimento` · `/declaracoes` | Um passo por chamada, salvo na hora |
| `POST /inscricoes/{id}/finalizar` | Encaminha para pagamento ou isenção ativa |
| `GET`/`POST /inscricoes/{id}/pagamento` | Cobranças · gerar Pix ou boleto (duplo clique devolve a mesma) |
| `POST /inscricoes/{id}/pagamento/{cobranca}/simular` | Só com `LISURA_SANDBOX`: o candidato pagando no banco. Pix já pago: **409** |
| `GET /inscricoes/{id}/devolucoes` | O que entrou a mais e o caminho de volta de cada pagamento (pendente, aguardando chave, concluída, recusada); com `recusa` quando a última chave não serviu |
| `GET /inscricoes/{id}/avisos` | Os e-mails mandados sobre a inscrição: evento, assunto, quando, se foi entregue e o SHA-256 que prova o texto |
| `PUT /inscricoes/{id}/devolucoes/{devolucao}/chave` | A chave Pix para onde volta o boleto pago a mais, quando o CPF não é chave: **202** (sai pela fila); **422** chave mal escrita ou de CNPJ; **409** a devolução não espera chave |
| `GET /inscricoes/{id}/resultado` | Onde está o resultado individual: a fatia estática e a chave da linha (no pico, o front deriva a chave do cartão e nem chama) |
| `GET`/`POST /inscricoes/{id}/recursos` | Meus recursos contra o gabarito, com a decisão de cada um · recorrer (questão, fundamento, argumento) |
| `POST /psp/aviso` | Aviso do prestador, **só com assinatura válida**; repetido responde 200 sem reaplicar |
| `GET /situacao` | Nível de degradação atual (público, nunca entra na fila) |
| `GET`/`POST /operacao` | Ver ou trocar o nível — só com `Authorization: Operacao <LISURA_OPERACAO_SEGREDO>` |

Erros com código estável: **401** sem sessão · **403** escrita com cookie vinda de outro site ·
**404** inexistente *ou de outro candidato* · **409** fora do prazo, já finalizada, CPF repetido ·
**413/415** arquivo grande ou de formato errado · **422** dado inválido ou incompleto (lista `faltando`)
· **502** prestador de pagamento fora do ar · **503** sistema sob carga: `{erro, degradado}` e, na sala
de espera, `fila: {posicao, espera_segundos}` com `Retry-After`.

Ferramentas instaladas sem administrador em `C:\Users\rockb\tools\` (`go\`, `pgsql\`, `node\`, dados
em `pgdata-lisura\`). O Postgres escuta só em `localhost:5432`, com autenticação `trust`.

## O que está provado por teste

| Garantia | Teste |
|---|---|
| Só transições do diagrama; estado e trilha mudam juntos | `TestProximo`, `TestFluxoCompletoGravaTrilha`, `TestTransicaoInvalidaNaoGravaNada` |
| Aviso de pagamento repetido não confirma duas vezes (50 simultâneos) | `TestConfirmacoesConcorrentesSoUmaVale` — **conferido contra o defeito**: sem o `FOR UPDATE`, 12 transições onde só 1 vale |
| Pagamento × vencimento no mesmo instante: só um vence | `TestConfirmacaoContraExpiracao` |
| Duplo clique não duplica inscrição (50 simultâneos) | `TestAbrirConcorrenteNaoDuplica` |
| Só o ator certo decide; indeferir exige motivo; trilha não aceita alteração pela aplicação | `TestAtorEMotivoSaoExigidos`, `TestTrilhaEImutavel` |
| Decisão registra a versão do edital em vigor; sem edital não há inscrição | `TestDecisaoRegistraVersao…`, `TestSemEditalPublicado…` |
| Fluxo completo por HTTP; finalizar diz o que falta; retirar o aceite volta a faltar | `TestInscricaoPontaAPonta` |
| Laudo exigido por cota PcD ou atendimento, compartilhado **só com esta inscrição** | idem |
| Isenção deferida dispensa pagamento; NIS de outro CPF é indeferido por regra | `TestIsencaoDeferidaDispensaPagamento` |
| Um candidato não alcança inscrição nem laudo de outro; cargo/cidade/cota de outro certame são recusados | `TestUmCandidatoNaoAlcancaOOutro` |
| Prazo encerrado trava tudo; arquivo validado pelo conteúdo, até 5 MB | `TestPrazoEncerradoTrava`, `TestUploadConfereConteudo` |
| Cookie de sessão `HttpOnly`/`SameSite=Lax`; escrita com cookie vinda de outro site é recusada | `TestSessaoPorCookie` |

### Pagamento — "nenhuma cobrança duplicada sob repetição" (semana 3)

| Defeito provocado no PSP simulado | O que acontece | Teste |
|---|---|---|
| 50 cliques simultâneos em "Pagar com Pix" | 1 cobrança aqui, 1 no prestador | `TestCobrancaConcorrenteNaoDuplica` — **conferido contra o defeito**: com chave de idempotência aleatória, 40 cobranças no prestador |
| Prestador cria a cobrança e a resposta se perde (2×) | O cliente repete com a mesma chave; 1 cobrança | `TestRespostaPerdidaNaoCriaSegundaCobranca` |
| O mesmo aviso chega 50× ao mesmo tempo | 1 confirmação, 49 "já processado" | `TestAvisoRepetidoConfirmaUmaVez` — **conferido contra o defeito**: sem a deduplicação, a repetição vira "pagamento em duplicidade" e um reembolso falso |
| O aviso nunca chega | A conciliação acha no extrato e confirma; rodar de novo não reaplica | `TestAvisoPerdidoRecuperadoPelaConciliacao` |
| Candidato paga o Pix **e** o boleto | 1 confirmação; o segundo pagamento fica marcado para reembolso | `TestPixEBoletoPagosViramUmaConfirmacaoEUmReembolso` |
| Boleto pago duas vezes | Cobrança paga uma vez, reembolso marcado nela | `TestBoletoPagoDuasVezes` |
| Pagamento depois de a inscrição expirar | Inscrição continua expirada; pagamento vira reembolso | `TestPagamentoDepoisDaExpiracao` |
| Boleto pago no último dia, informado depois | Não expira dentro da margem de 3 dias; confirma | `TestBoletoNaMargemDeCompensacaoNaoExpira` |
| Valor pago diferente do cobrado | Não confirma; divergência registrada | `TestValorDivergenteNaoConfirma` |
| Confirmado aqui, ausente do extrato | Divergência registrada | `TestConfirmadoSemRegistroNoPrestador` |
| Pagamento de referência desconhecida | Divergência registrada, nada muda | `TestPagamentoDeCobrancaDesconhecida` |
| Fluxo HTTP inteiro com cada aviso chegando 5× ao mesmo tempo; aviso com assinatura falsa | "Paga" com 1 confirmação; aviso forjado recusado com 401 e não registrado | `TestPagamentoPontaAPontaComAvisosRepetidos` (`internal/api`) |

#### A devolução do que entrou a mais (migração 00012, 05/11 à noite)

Até aqui o sistema **marcava** o reembolso e ninguém devolvia nada. Agora, o pagamento classificado
como a mais (pago em duplicidade ou depois da expiração) cria, na mesma transação, uma **devolução**
e a tarefa que a pede ao prestador.

- **Uma devolução por pagamento,** não por cobrança. O boleto pago três vezes tem dois pagamentos a
  devolver. A chave única no pagamento é o espelho de "nunca cobrar duas vezes": **nenhum pagamento
  é devolvido duas vezes.**
- **O id da devolução no prestador** é `dev-` mais o id do pagamento, e vai na URL
  (`PUT /devolucoes/{id}`), como na API Pix do Banco Central. Repetir devolve a mesma devolução.
- **O Pix volta para quem pagou,** pelo próprio pagamento, sem chave.
- **O boleto não diz quem pagou.** Volta por Pix para a chave CPF do candidato, e o prestador confere
  que o titular da chave é esse CPF. Quem tomar a conta não desvia a devolução para outra pessoa. Se
  o CPF não é chave Pix, a devolução fica `AGUARDANDO_CHAVE`: não é recusa, e nenhum dinheiro sai.
- **Recusa definitiva** do prestador (valor acima do pago, chave de outro titular) não é repetida. A
  devolução fica `RECUSADA`, e a auditoria acusa.
- **A fila abandona uma tarefa depois de 5 tentativas,** cerca de meio minuto de prestador fora do
  ar. A devolução não morre ali: a cada 10 min, `RetomarDevolucoes` reenfileira toda devolução
  `PENDENTE` sem tarefa a caminho.
- **O candidato vê as devoluções** em `GET /api/inscricoes/{id}/devolucoes` e na página da inscrição
  (desde 06/11). Quando falta chave, a tela pede uma e diz por que a anterior não serviu, só se ele
  informou alguma (`chave_informada`): a recusa da chave CPF, tentada sozinha, não é dele.
- **Quando o CPF não é chave Pix,** o candidato informa outra chave dele em
  `PUT /api/inscricoes/{id}/devolucoes/{devolucao}/chave`.
  - **Formato conferido antes de ir ao prestador** (`NormalizarChavePix`): CPF com os dígitos
    verificadores; e-mail; telefone `+55`, DDD e número; chave aleatória. Tudo é guardado como o
    DICT, o diretório de chaves Pix do Banco Central, guarda.
  - **Chave de CNPJ é recusada:** é de empresa, nunca do candidato.
  - **A chave só é aceita enquanto a devolução espera chave.** Pedida ao prestador ou concluída, não
    muda mais, e quem tomar a conta não redireciona uma devolução a caminho.
  - **A proteção de fato é o prestador,** que só paga se o titular da chave for o CPF da conta.
    Chave de outra pessoa ou que não existe faz a devolução voltar a esperar, com `recusa` dizendo
    por quê, e nenhum dinheiro sai. Antes, a chave de outro titular deixava a devolução `RECUSADA`,
    morta.

| Defeito provocado | O que acontece | Teste |
|---|---|---|
| Pix e boleto pagos; a conciliação passa 3 vezes | 1 devolução, do boleto, para a chave CPF; a inscrição continua PAGA | `TestPagoEmDobroVolta` — **conferido contra o defeito**: sem criar a devolução em `aplicar`, falha |
| Pix pago, aviso perdido, inscrição expira | Volta para quem pagou, sem chave; a inscrição continua expirada | `TestPixDepoisDaExpiracaoVoltaParaQuemPagou` — idem |
| O prestador devolve e a resposta se perde | A fila repete; o prestador reconhece o id; 1 devolução | `TestRespostaPerdidaNaoDevolveDuasVezes` — **conferido contra o defeito**: com id novo a cada tentativa, falha |
| O mesmo pagamento a mais chega 20 vezes ao mesmo tempo, por aviso e conciliação | 1 devolução, 1 tarefa | `TestPagamentoAMaisRepetidoUmaDevolucao` |
| Boleto a mais, e o CPF não é chave Pix | `AGUARDANDO_CHAVE`; nada sai | `TestSemChaveCPFAguardaChave` |
| Tarefa abandonada depois das tentativas | A rotina reenfileira; 1 devolução | `TestDevolucaoAbandonadaVoltaParaAFila` — **conferido contra o defeito**: com a rotina sem retomar, falha |
| No prestador: 20 pedidos com o mesmo id, outro id para o mesmo Pix, o mesmo id com outro valor, boleto sem chave, CPF sem chave, chave de outro CPF, e-mail de outro titular, e-mail que não é chave, resposta perdida, reinício | 1 devolução; recusa com o motivo certo; os totais sobrevivem ao reinício | `TestDevolucaoNoPrestador` (`internal/psp`) |
| Chave Pix: CPF com e sem pontuação, e-mail em maiúsculas, telefone com espaços e parênteses, aleatória; e 11 formatos errados, CNPJ entre eles | Normalizadas como o DICT guarda; as erradas recusadas | `TestChavePixNormalizada` |
| CPF sem chave; o candidato informa a chave de outra pessoa, depois a dele em maiúsculas; tenta trocar com a devolução a caminho e depois de concluída; outro candidato e um CNPJ tentam informar | De outra pessoa: volta a esperar, `recusa` = `TITULAR_DIFERENTE`, nada sai. A dela: 1 devolução. A caminho ou concluída: não muda. Outro candidato: não encontrada. CNPJ: dado inválido | `TestCandidatoInformaChaveEDinheiroVolta` — **conferido contra o defeito** duas vezes: aceitando chave fora da espera, a troca passa com a devolução a caminho; com a chave de outro titular dando `RECUSADA`, a devolução morre |
| O mesmo caminho pelo HTTP | 404 para outro candidato, 422 para CNPJ, 202 e a devolução concluída com a chave dela, 409 depois | `TestChavePixPelaAPI` (`internal/api`) |

**A prova contra o defeito achou uma segunda linha de defesa e um buraco no teste.** Sem a
conferência de estado, a troca de chave numa devolução **concluída** foi barrada mesmo assim, pela
restrição da tabela (concluída ⇔ id no prestador e data). Já a devolução **pedida e ainda não
concluída** não tinha teste, e é a que só a conferência protege. O caso entrou no teste.

**Uma descoberta da prova contra o defeito:** com id novo a cada tentativa, o prestador recusou a
segunda devolução (`VALOR_ACIMA_DO_PAGO`), e o dinheiro não saiu duas vezes. O limite do prestador é
uma segunda linha de defesa, mas a primeira tem de ser nossa: a devolução ficaria `RECUSADA` em vez de
concluída.

**Na auditoria:** pago a mais **sem devolução**, devolução de pagamento que confirmou a inscrição,
devolução acima do pago e devolução recusada. O relatório mostra as devoluções por estado.

**Medido no banco local** (05/11, 21h20). A migração criou as devoluções de tudo o que já tinha
entrado a mais: 906 pagamentos, 617 de Pix e 289 de boleto. A primeira rodada da rotina, 1 min
depois de a API subir, mandou todas à fila:

| Resultado | Quantas |
|---|---|
| Concluídas | **868**, todas em 0,48 s |
| `RECUSADA`, `CHAVE_OBRIGATORIA` | 38, todas de Pix |

As 38 recusadas são **todas** de pagamentos entre 01h10 e 02h14, antes de o PSP simulado ter diário,
criado às 15h23. A migração do PSP antigo para o diário (`trocar-psp.ps1`) levou os pagamentos, mas
não as cobranças. Sem a cobrança, o prestador não sabe que aquele pagamento foi Pix e pede chave.
Todo Pix devolvido com sucesso é de depois das 15h41. É artefato daquela migração local, e o sistema
fez o que devia: recusa definitiva fica para alguém olhar, e a auditoria acusa.

#### Reembolso em massa: o certame anulado (migração 00013)

A banca cancela o certame (`banca cancelar -certame N -analista ana -motivo ANULADO_POR_DECISAO_JUDICIAL`,
com o motivo em código padronizado, como todo motivo da trilha). Em três passos, que fecham as portas
antes de varrer:

1. **O certame é marcado e as cobranças pendentes expiram.** Daí em diante não entra inscrição, passo
   de formulário, isenção nem cobrança nova (`ErrCertameCancelado`, **409**). O Pix que já estava
   gerado ainda pode ser pago no banco; esse pagamento cai em "pago depois da expiração", que já
   devolve.
2. **Toda inscrição viva vai a `CANCELADA`** pela máquina de estados (evento `CANCELAR_CERTAME`), com
   a analista e o motivo na trilha e o cartão republicado. Inclui a `HOMOLOGADA`: um certame anulado
   depois da prova (fraude achada depois) também cancela. A anulação é a única saída da homologação;
   `EXPIRADA` e `CANCELADA` continuam sem saída.
3. **Todo pagamento confirmado ganha uma devolução**, motivo `CERTAME_CANCELADO`, em lotes de 1.000 e
   com prioridade de lote na fila. O Pix volta para quem pagou e o boleto para a chave CPF, como na
   devolução individual. Depois do passo 2 nenhum pagamento é confirmado, então o passo 3 alcança
   todos.

Chamar de novo continua de onde parou e não repete nada: a transição já aplicada é pulada, e a
devolução tem chave única no pagamento.

| O que se prova | Teste |
|---|---|
| Certame com inscrição em rascunho, aguardando o Pix, paga por Pix e paga por boleto: 4 canceladas com trilha (analista e motivo), 2 devoluções concluídas; chamar de novo não faz nada; o Pix gerado antes e pago depois também volta; inscrição nova e cobrança são recusadas | `TestCancelarCertameDevolveTudoUmaVez` — **conferido contra o defeito** duas vezes: sem o passo 3, 0 devoluções em vez de 2; com o certame cancelado aceitando inscrição, a nova entra |
| Motivo vazio ou em texto livre é recusado, e nada muda | `TestCancelarCertameExigeMotivoPadronizado` |
| A anulação alcança toda inscrição viva e nenhuma morta, exige motivo e é decisão da banca | `TestCancelarCertameAlcancaTodaInscricaoViva` (`internal/inscricao`) |
| O instante entre o passo 1 e o passo 2: o certame já marcado, a inscrição ainda aguardando pagamento; a cobrança é recusada e nenhuma nasce | `TestCobrarNoMeioDoCancelamento` (06/11) — **conferido contra o defeito**: sem a conferência do certame em `Cobrar`, a cobrança nasce |

Até 06/11, a recusa pelo certame cancelado não estava provada: o teste de cima cobra numa inscrição
já cancelada, que recusa sozinha. O teste novo para no meio do cancelamento, que é o único instante
que a conferência protege.

**Na auditoria:** em certame cancelado, inscrição que continua valendo e pagamento confirmado sem
devolução. A devolução de um pagamento que confirmou a inscrição deixa de ser acusada quando o
motivo é o cancelamento do certame.

**Medido na máquina local** (05/11, 22h). O certame 10 foi criado para isto, e o 5, o do ensaio,
ficou intacto. Primeiro, uma rodada real de 640× por 2 min: 28.831 inscrições, 24.885 pagas, 2.839
aguardando pagamento e 1.107 em rascunho. Depois, `banca cancelar`:

| Medida | Resultado |
|---|---|
| Inscrições canceladas | 28.831, todas, com trilha |
| Devoluções criadas | 24.885, uma por pagamento confirmado |
| Duração do comando | **41,7 s** |
| Da anulação à última devolução concluída no prestador | **41,9 s**: a fila (4 trabalhadores) devolveu enquanto os lotes eram criados |
| Auditoria da rodada depois | **0 problemas**; 24.885 devoluções concluídas, 0 inscrições vivas, 0 pagamentos retidos |

Dá ~600 devoluções por segundo. Num certame de 200 mil pagantes, a conta linear daria ~6 min. Essa
conta **não foi medida**.

### Ensalamento (`internal/ensalamento`, migração 00016)

Cada inscrição homologada ganha uma sala na cidade de prova que escolheu (02-produto §7 F;
"ensalamento simples por capacidade" é do MVP, §8). Antes, o ciclo pulava da homologação para o
resultado sem ninguém ter sala.

**Os locais e as salas** vêm de um CSV da banca (`banca locais -certame N -arquivo locais.csv`):

- o **local** é da cidade de prova e pode ter espaço para amamentação;
- a **sala** tem capacidade, pode ser acessível e tem um tipo: **comum**; **especial** (tempo
  adicional, prova ampliada: o grupo fica mais tempo, com fiscal próprio); **individual** (ledor,
  transcritor: um candidato e quem o atende).

Linha errada recusa o arquivo inteiro, com o número da linha.

**A regra** (`banca ensalar -certame N -analista X -motivo CODIGO`):

| Atendimento pedido | Sala |
|---|---|
| Ledor ou transcritor | Individual |
| Tempo adicional ou prova ampliada | Especial |
| Sala de fácil acesso | Acessível |
| Lactante | Num local com amamentação |
| Nenhum | Comum |

- **Os mais restritos escolhem primeiro.** Dentro de cada grupo, a ordem é cargo e nome, e a sala é a
  primeira com lugar, na ordem de local e nome. O mesmo dado dá sempre a mesma alocação.
- **Faltou lugar, ninguém é alocado.** O erro diz quantos lugares de que tipo faltam em que cidade.
- **Ensalar de novo mantém quem já tinha sala válida.** Quem entrou por recurso fica com o lugar que
  sobrou, e quem saiu libera o seu. Sem mudança, nenhuma versão nova.
- **Cada ensalamento é um ato publicado,** imutável, com o SHA-256 da alocação inteira.
- **O banco recusa sala lotada,** venha a alocação de onde vier: é a última linha de defesa, como
  na cobrança.
- **O cartão de confirmação passa a mostrar local, endereço e sala.** Já prometia isso desde o pico 4.
  Os cartões de quem ganhou ou perdeu sala são republicados pela fila, com prioridade de lote,
  enfileirados numa instrução só (`fila.EnfileirarMuitos`).
- **Para a banca:** `banca mapa` (ocupação por local), `banca listas` (uma lista de presença por
  sala, com CPF mascarado e coluna de assinatura) e `banca conferir-ensalamento`.

**A conferência verifica:**

- homologada sem sala, e sala para quem não está homologada;
- sala em outra cidade;
- sala acima da capacidade;
- atendimento especial não atendido, com a mesma regra da alocação;
- alocação que não é a do último ato publicado.

| O que se prova | Teste |
|---|---|
| Cada atendimento na sala certa, os restritos primeiro, os demais em ordem de nome; mapa e listas | `TestEnsalaComCapacidadeEAtendimento` — **conferido contra o defeito**: sem os restritos primeiro, o candidato que precisa de sala acessível fica sem lugar |
| Dois ledores para uma sala individual: ninguém alocado, e o erro diz o que falta e onde | `TestFaltaLugarNaoAlocaNinguem` |
| Uma saída e uma entrada por recurso: quem tinha sala continua nela, a nova fica com o lugar que vagou; sem mudança, sem versão nova | `TestReensalarMantemQuemTinhaSala` — **conferido contra o defeito**: refeito do zero, 0 mantidas e 7 novas |
| O banco recusa a segunda pessoa na sala individual | `TestBancoRecusaSalaLotada` — **conferido contra o defeito**: sem a condição no gatilho, aceita |
| Ledor movido para a sala especial e a de tempo adicional sem sala, por fora: as três coisas acusadas | `TestConferirAcusaAlocacaoMexida` — **conferido contra o defeito**: sem comparar o resumo, a alocação mexida passa |
| O cartão mostra local, endereço e sala, e deixa de avisar que eles ainda vão sair | `TestCartaoMostraASala` |
| CSV com cidade de outro certame, sala individual com dois, tipo inventado, cabeçalho errado | `TestImportarLocaisRecusaLinhaErrada` |
| Enfileirar muitas: a pendente promovida, a concluída reaberta, a repetida na mesma chamada vira uma, 1.003 tarefas | `TestEnfileirarMuitosComoUmPorUm` (`internal/fila`) |

**Dois testes foram ajustados antes da prova contra o defeito, porque não provavam nada.** Com os
nomes da primeira versão, o candidato que precisa de sala acessível vinha primeiro em ordem
alfabética e ganharia a sala com ou sem a regra. Ele virou "Zé Acessível". E a saída escolhida no
reensalamento dava a mesma alocação refeita do zero. Agora sai alguém da sala 102, e refazer do zero
empurraria outra candidata. Uma terceira prova foi refeita: o primeiro defeito plantado no gatilho
quebrava a migração inteira, e o teste falhava pelo motivo errado.

**Medido na máquina local** (06/11). O certame 11 foi criado para isto: 150 mil inscrições
homologadas, em 3 cargos e 3 cidades, com atendimento especial na proporção de um certame real (1%
ledor, 2% tempo adicional, 1% sala acessível, 1% lactante, 1% prova ampliada em sala acessível). Os
locais: 40 escolas por cidade, cada uma com 30 salas comuns de 42 lugares, 2 especiais de 20 e 14
individuais.

| Etapa | Resultado |
|---|---|
| Importar locais e salas | 120 locais e 5.520 salas em 1,6 s |
| `banca ensalar` | **150.000 alocadas em 14,3 s**, numa transação, com os 150 mil cartões enfileirados |
| `banca conferir-ensalamento` | **0 problemas**, 2,7 s |
| Ensalar de novo | "nada mudou", 1,1 s, sem versão nova |
| `banca listas` | 5.155 listas de presença (as salas com alguém) em 8,4 s |
| `banca mapa` | 157.680 lugares, 150.000 alocados, 7.680 livres |
| Cartões republicados pela fila (4 trabalhadores) | 150.000 em **5 min 14 s** (~480 por segundo), **0 falhas**; 200 de 200 cartões da amostra com local e sala |

**Limite conhecido:** quem entra depois por recurso disputa só o lugar que sobrou. Se o candidato
novo precisa de sala acessível e as acessíveis estão tomadas por quem não precisa, o ensalamento
acusa a falta em vez de remanejar quem já tinha sala. É a troca consciente por ninguém mudar de sala
depois de publicada; a banca resolve abrindo outra sala.

### Exportação para o órgão e a auditoria (`internal/exportacao`)

"Exportação para órgão e auditoria" (02-produto §7 H). `banca exportar -certame N -pasta P` grava um
pacote:

- **um CSV por assunto:** edital e versões, cronograma, inscrições, trilha, pagamentos,
  devoluções, dias fechados, avisos, alocação e versões do ensalamento;
- **todas as âncoras do certame, copiadas do cofre;**
- **um manifesto** com o SHA-256 de cada arquivo e um **selo** do conjunto. Quem recebe o pacote
  guarda só o selo.

As consultas rodam numa transação só, REPEATABLE READ: os arquivos são da mesma fotografia do banco,
gravados direto do `COPY` do Postgres, com o SHA-256 calculado no caminho. **Dado pessoal fica
fora:** CPF mascarado, sem nome, sem e-mail. O número da inscrição liga tudo.

`banca verificar-exportacao -pasta P` é o que o auditor roda, **sem banco**, só com a pasta:

1. cada arquivo bate com o manifesto, e o manifesto com o selo; nada a mais nem a menos na pasta;
2. a corrente da trilha refeita evento por evento, com a mesma conta do banco
   (`inscricao.ResumoDoEvento`, a mesma função da auditoria, para as duas nunca divergirem);
3. o selo de cada inscrição é o último evento da trilha dela;
4. cada ponto de cada âncora do cofre está na trilha exportada, com o mesmo resumo;
5. cada dia fechado é refeito dos pagamentos e das devoluções exportados: totais e resumo.

**Os testes adulteram como um adulterador competente:** mexem num arquivo e refazem o manifesto
inteiro (cada SHA-256 e o selo), para o pacote voltar a bater por fora.

| Adulteração, com o manifesto refeito | Quem acusa | Teste |
|---|---|---|
| Nenhuma | — (tudo confere; nenhum nome, e-mail ou CPF inteiro no pacote) | `TestPacoteIntegroConfere` — **conferido contra o defeito**: com o auditor ordenando a linha inteira em vez de tipo e id, o pacote íntegro dá alarme falso no dia fechado (o teste usa de propósito um id que é prefixo de outro) |
| Quem decidiu um evento, trocado na trilha | A corrente refeita do arquivo | `TestTrilhaAlteradaEReseladaEAcusada` — **conferido contra o defeito**: sem refazer a corrente, passa |
| A trilha reescrita inteira, com a corrente e os selos refeitos para ficarem coerentes | **Só a âncora do cofre**, fora do alcance de quem reescreveu | `TestTrilhaReescritaCoerenteEAcusadaPelaAncora` — **conferido contra o defeito**: sem conferir a âncora, passa |
| Um pagamento de dia fechado apagado | O dia fechado refeito | `TestPagamentoSumidoEAcusadoPeloDiaFechado` — **conferido contra o defeito**: sem refazer o dia, passa |
| Arquivo a mais, a menos, manifesto mexido sem refazer o selo | O manifesto e o selo | `TestArquivosEManifesto` |

**O limite, dito sem rodeio:** quem controla o pacote inteiro pode trocar também as âncoras copiadas
nele. A defesa contra isso fica fora do pacote: o recibo de cada âncora publicado em `/publico`, a
própria âncora no cofre (S3 com bloqueio de objeto) e o selo impresso no cartão de cada candidato.

**Um defeito achado na prestação de contas ao escrever esta conferência.** O resumo do dia fechado
ordenava os ids com `ORDER BY` de texto, e a ordem do texto depende da **colação** do banco. O banco
local é `C`, byte a byte. O Aurora vem com en_US.UTF-8, em que o "_" dos ids conta diferente. Na AWS,
o mesmo dia teria outro resumo, e o auditor, que refaz byte a byte, acusaria dias fechados que
ninguém mexeu. O fechamento passou a ordenar com `COLLATE "C"`. No banco local o resultado é o
mesmo de antes, e os dias já fechados continuam batendo. O mesmo cuidado entrou no auditor: ordenar
a linha inteira (`P|id|valor`) daria outra ordem que ordenar por tipo e id, porque "abc|" vem depois
de "abc_d".

### Avisos ao candidato (`internal/notificacao`, migração 00017)

"Notificação por e-mail e mensagem sobre cada mudança de estado" (02-produto §7 B). Ela pesa na
H1. Em produção, cada envio, pagamento e homologação manda um e-mail, e a 40× isso dá da ordem de
140 mil por hora, pela mesma fila do Pix. Um ensaio sem os avisos mediria um sistema mais leve do
que o real (04-escopo §8: "a configuração do ensaio é a mesma que iria para produção").

**Os eventos que avisam:** inscrição enviada (com o prazo do pagamento), enviada com isenção,
pagamento confirmado, expirada, indeferida (com o motivo), recurso provido, homologada e certame
cancelado (com o caminho da devolução). Criar o rascunho, iniciar a análise e o deferimento no meio
da homologação em lote não avisam: não pedem nada do candidato nem mudam o que ele pode fazer.

- **O aviso nasce na mesma transação do evento da trilha,** como tarefa da fila, a partir do único
  lugar por onde passa toda transição. Não há evento sem aviso nem aviso sem evento.
- **Com prioridade de lote:** o e-mail nunca passa na frente do Pix nem do cartão. No pico ele
  espera e sai quando o pico escoa.
- **Um aviso por evento, nunca dois.** A chave do envio é o id do evento, e o provedor que recebe a
  mesma chave devolve o mesmo envio. É o mesmo desenho da cobrança e da devolução. **Com o SES, não:**
  ele não tem essa chave, e a garantia vira "pelo menos um" (abaixo).
- **A prova do aviso não muda depois:** destinatário, assunto, SHA-256 do assunto e do texto, e
  quando saiu. É o que sustenta um prazo de recurso. O endereço recusado pelo provedor também fica
  registrado, com o motivo, e a fila não insiste.
- **O texto não leva o CPF.** O candidato vê seus avisos em `GET /api/inscricoes/{id}/avisos`.
- **O provedor:**
  - na máquina local, uma caixa de saída num arquivo (`LISURA_EMAIL_CAIXA`), que sobrevive a
    reinício;
  - no ensaio na AWS, `LISURA_EMAIL=descarte` (no `ecs.tf`): aceita e conta, e os avisos passam
    pela fila como em produção;
  - em produção, `LISURA_EMAIL=ses`, o Amazon SES (desde 06/11, abaixo).

**O SES** (`notificacao.SES`; no Terraform, `email_de` preenchida liga o SES e dá ao papel da API
só `ses:SendEmail` no remetente e `ses:GetAccount`):

- **O que muda na garantia, dito sem rodeio:** o SES não tem chave de idempotência (a chave que faz
  o provedor devolver o mesmo envio em vez de mandar outro). Se a mensagem sai e a resposta se
  perde, a fila repete, e a candidata recebe **duas cópias**. Com o SES, "um aviso por evento" vira
  "pelo menos um". Para um aviso que abre prazo de recurso, chegar duas vezes é melhor do que não
  chegar.
- **A prova continua uma só:** a tabela é única por evento. Cada cópia leva a chave no cabeçalho
  `X-Lisura-Chave` e na etiqueta `chave` do SES, para o atendimento reconhecer a repetida.
- **A cota:** o SES limita envios por segundo e por dia, por conta e região. Na subida, o servidor lê
  a cota real da conta (`GetAccount`) e espaça os envios nela, em vez de bater no limite e depender
  das repetições da fila. `LISURA_SES_TAXA` só baixa a taxa. Sem acesso à conta, o servidor não
  sobe: o problema aparece na subida, não no primeiro aviso encalhado.
- **A conta que precisa ser feita antes de produção:** a 40×, os avisos são da ordem de 140 mil
  por hora, ~39 por segundo. A cota de uma conta recém-saída do sandbox (o modo de teste do SES, que
  só entrega para endereços verificados) é definida pela AWS e costuma ficar abaixo disso. Pedir o
  aumento é passo de implantação, e o log da subida mostra a cota e se a conta ainda está no sandbox.
  Com a cota baixa, os avisos não se perdem: esperam na fila com prioridade de lote, como no pico.
- **A recusa:** `MessageRejected` (no sandbox, o destinatário não verificado) é definitiva e fica
  registrada. Vazão, conta pausada e falha do serviço são passageiras, e a fila repete. O endereço
  que só devolve depois (*bounce*) não aparece no envio: chega por um conjunto de configuração
  (`LISURA_SES_CONJUNTO`) e é tratado abaixo, em "Depois de sair".
- **Não verificado contra a AWS de verdade:** os testes usam um SES falso local, que responde como
  a API v2. A identidade do remetente (domínio com DKIM, a assinatura que prova ao destinatário que
  o domínio mandou o e-mail) é configurada na conta e não está no Terraform.

| O que se prova | Teste |
|---|---|
| O SES recebe exatamente o aviso: remetente, destinatário, assunto e texto em UTF-8, o conjunto de configuração, a chave no cabeçalho e na etiqueta | `TestSESMandaOQueOAvisoDiz` — **conferido contra o defeito**: sem o cabeçalho da chave, falha |
| `MessageRejected` é definitiva; vazão, conta pausada e falha do serviço, passageiras | `TestSESRecusaEDefinitivaVazaoNao` — **conferido contra o defeito**: com a recusa tratada como passageira, falha |
| 10 envios ao mesmo tempo a 20/s saem espaçados | `TestSESEspacaOsEnvios` — **conferido contra o defeito**: sem o espaçamento, os 10 saíram em 1,6 ms |
| O limite, provado: a resposta se perde, a fila repete, 2 cópias com a mesma chave e 1 prova | `TestSESRespostaPerdidaAvisaPeloMenosUmaVez` |

**Depois de sair: devolução e reclamação** (`notificacao.ReceberEventosSES`, migração 00020; no
Terraform, `ses.tf`). O SES publica num tópico do SNS a devolução (*bounce*: o servidor do
destinatário recusou), a reclamação (o destinatário marcou como spam) e a entrega. O SNS entrega por
HTTPS em `POST /api/ses/eventos`, fora da sala de espera.

- **A assinatura é conferida como a AWS documenta.** Só vale mensagem do nosso tópico. O certificado
  só é baixado de `https://sns.<região>.amazonaws.com`, e a assinatura RSA da mensagem canônica tem
  de conferir (SHA1 na versão 1, SHA256 na 2). Sem isso, qualquer um mandaria uma "devolução" falsa e
  calaria os avisos de um candidato. O que não confere recebe 403 e não muda nada.
- **Cada evento vira uma ocorrência imutável**, ligada ao aviso pelo id do envio no SES. A mesma
  mensagem entregue de novo (o SNS repete até ouvir 200) é a mesma linha.
- **Devolução definitiva ou reclamação suprimem o endereço.** Os próximos avisos não são mandados e
  ficam registrados como não enviados, com o porquê. Insistir derruba a reputação do remetente, e o
  SES pode pausar a conta inteira. A devolução temporária (caixa cheia, servidor fora) só fica
  registrada.
- **O candidato vê:** na área dele, o aviso aparece como "Voltou" ou "Não enviado", e a tela pede um
  e-mail novo, que não está suprimido. Tudo o que os avisos dizem também está na página.
- **A exportação para o órgão** leva as ocorrências (`avisos_ocorrencias.csv`) sem o endereço e sem
  o diagnóstico do servidor de e-mail, que costuma repetir o endereço: só o tipo e a classe.
- **Na AWS, não verificado.** A inscrição da API no tópico é confirmada pela própria API, então só
  existe com ela no ar (`var.ligado`). Se a confirmação chegar antes de a API responder, o `apply`
  falha no prazo e tem de rodar de novo.

| O que se prova | Teste |
|---|---|
| Só passa mensagem do nosso tópico, assinada pela chave do certificado vindo do SNS (versões 1 e 2); texto alterado, outro tópico, outra chave, certificado de fora do SNS ou sem HTTPS: recusados, e o de fora nem é baixado | `TestSNSSoAceitaONossoTopicoAssinado` — **conferido contra o defeito**: aceitando certificado de qualquer host, baixa e confia |
| Devolução definitiva: registrada uma vez (mesmo entregue duas), endereço suprimido sem importar maiúsculas, o candidato vê que voltou, e o aviso seguinte não sai | `TestDevolucaoSuprimeEOCandidatoVe` — **conferido contra o defeito**: sem conferir a supressão, o aviso seguinte sai |
| Devolução temporária: registrada, não suprime | `TestDevolucaoTemporariaNaoSuprime` — **conferido contra o defeito**: suprimindo também a temporária, o aviso seguinte não sai. A primeira tentativa de plantar foi barrada pela restrição do banco (motivo só `DEVOLVIDA` ou `RECLAMACAO`); a segunda passou pelo banco e o teste acusou |
| Devolução forjada (assinada por outra chave): 403, nada registrado, nada suprimido | `TestDevolucaoForjadaNaoCalaNinguem` — **conferido contra o defeito**: sem conferir a assinatura, a forjada é aceita |
| A inscrição no tópico é confirmada pelo `SubscribeURL` do SNS; um de fora não é visitado | `TestInscricaoNoTopicoConfirmada` |
| O pacote para o órgão leva a devolução do aviso, e nenhum arquivo leva nome ou e-mail | `TestPacoteIntegroConfere` (`internal/exportacao`) — **conferido contra o defeito**: com o diagnóstico inteiro, o e-mail vaza no pacote |
| Do envio à homologação: avisa o envio (com o prazo), o pagamento e a homologação, e não avisa a análise nem o deferimento; cada aviso uma vez, mesmo pedido de novo; a prova confere com o texto que saiu; nenhum texto leva o CPF | `TestCadaEventoQueImportaAvisaUmaVez` — **conferido contra o defeito**: com o evento sem enfileirar o aviso, nenhum sai |
| A resposta do provedor se perde depois do envio: um e-mail, não dois | `TestRespostaPerdidaNaoAvisaDuasVezes` — **conferido contra o defeito**: com chave nova a cada tentativa, 2 mensagens |
| Endereço recusado: registrado com o motivo, e a fila não insiste | `TestEnderecoRecusadoFicaRegistrado` — **conferido contra o defeito**: tratado como falha passageira, nada fica registrado |
| O banco recusa alterar e apagar a prova | `TestAvisoNaoMuda` |
| A caixa de saída: a mesma chave é o mesmo envio, também depois de reabrir | `TestCaixaSobreviveAReinicio` |
| Pela API: os dois avisos da candidata, com o resumo; outra candidata não vê nada | `TestAvisosPelaAPI` (`internal/api`) |

**Um defeito de montagem achado ao compilar:** o teste importava a inscrição, e a inscrição agora
importa a notificação. O ciclo só existia no teste, e o teste passou a ser um pacote externo
(`notificacao_test`).

**Medido sob carga** (06/11, 640× local, 2 min, 4 trabalhadores). A primeira rodada com os avisos
mandou **28.990 avisos**:

- 15.256 de envio e 13.734 de pagamento;
- todos entregues, **0 falhas**, e nenhum evento ficou sem aviso;
- no pico os avisos **esperaram**: 2 min 48 s em média, e nenhum rodou no pico. Saíram depois, o
  último 3 min 36 s depois do evento.

A mesma rodada veio bem pior que o 640× da noite anterior:

| | Inscrições | p95 crítico | Cartão (p95) |
|---|---|---|---|
| Noite anterior, sem avisos | 27.415 | 62,8 ms | 19,5 s |
| Primeira rodada com avisos | 15.256 | 381 ms | 116 s |

**Não dava para atribuir isso aos avisos sem prova.** Entre uma rodada e outra, o banco tinha crescido
e a máquina era outra. Então veio uma **comparação controlada**: o mesmo código e a mesma hora, um
binário sem o aviso enfileirado na transição (o mesmo trecho plantado da prova contra o defeito), e
rodadas alternadas.

| 640×, 06/11 de manhã | Sem avisos | Com avisos |
|---|---|---|
| Inscrições | 19.313 | 19.591 |
| p95 das requisições críticas | 218,7 ms | 268,9 ms |
| Jornadas concluídas | 100% | 99,99% (1 falha) |
| CPU da máquina no patamar | 88% | 87% |

- **O custo dos avisos está dentro do ruído:** as mesmas inscrições e a mesma CPU. A diferença de
  p95, 50 ms, é menor que a variação entre rodadas iguais nesta noite (de 62 a 381 ms).
- **O que piorou o 640× entre a noite e a manhã é outra coisa,** que ainda não foi medida.
- **O limite:** uma rodada de cada. Eram quatro, alternadas, e o processo morreu no começo da
  terceira; a causa não ficou clara. Isso basta para descartar um efeito grande, não para medir um
  pequeno.

**Depois da relação em fatias** (06/11 à tarde, 640× local, 2 min, 4 trabalhadores, com os avisos):

| | Manhã, com avisos (A/B) | Tarde, com avisos e fatias |
|---|---|---|
| Inscrições | 19.591 | **25.554** |
| p95 das requisições críticas | 268,9 ms | **150,3 ms** |
| Jornadas concluídas | 99,99% | **100%** |
| p95 até o Pix / até o cartão | — | 1,7 s / 39,8 s |
| CPU da máquina no patamar | 87% | 80% |

- **A rodada passou em tudo:** 0 perdidas, 0 duplicadas, 26.574 trilhas verificadas sem quebra,
  0 divergências da conciliação, auditoria sem problemas.
- **Na fila, durante a rodada:** a relação foi republicada poucas vezes por minuto, a ~50 ms cada,
  e antes cada uma custava 3 s. Os cartões esperaram até 44 s com 4 trabalhadores, atrás do Pix,
  que tem prioridade. Os 34.565 avisos esperaram até 2 min 34 s e saíram todos depois do pico, 0
  falhas.
- **O que isto não prova:** é uma rodada, não uma comparação A/B. É coerente com a relação ter
  sido a maior parte da piora da manhã, mas a medida não isola isso.
- **Duas rodadas perdidas antes, por ambiente:** o reinício da máquina tinha derrubado o front
  de produção (porta 8090), e depois o Next subiu escutando só em `[::1]` (IPv6), enquanto o k6
  disca `127.0.0.1`. As duas deram 0 inscrições e 28.831 falhas do k6, e não entram em nenhuma
  conta. O front local de carga sobe com `HOSTNAME=127.0.0.1`.

### Cronograma e retificação (`internal/cronograma`)

"Mudou uma data, o sistema mostra o que quebra" (02-produto §7 A). As regras tornam explícito o que
o resto do sistema já supõe. Cada uma só vale quando o certame tem as duas fases:

| Regra | Por quê |
|---|---|
| A isenção fecha ao menos 5 dias antes do fim do pagamento | Quem tem o pedido negado precisa de tempo para pagar |
| O pagamento abre até o fim da inscrição e não fecha antes dele | Quem finaliza no último minuto tem de conseguir pagar |
| A homologação abre ao menos 3 dias depois do fim do pagamento | É a `MargemCompensacao`: o boleto pago no último dia ainda está sendo informado, e homologar antes contradiz a conciliação |
| O ensalamento abre depois do fim da homologação | Só a inscrição homologada ganha sala |
| A prova é ao menos 7 dias depois do fim do ensalamento | O cartão com o local sai com antecedência |
| O recurso contra o gabarito abre depois da prova e dura ao menos 2 dias | Prazo mínimo de recurso |
| O resultado abre depois do fim do recurso | O resultado reflete os recursos |

`banca cronograma -certame N` mostra as fases e o que quebra.

`banca retificar -certame N -fase F -fecha … -motivo … -analista … [-simular]` muda uma janela:

- **A fase encerrada não muda,** o início da fase aberta não muda, e o fim não vai para o passado.
- **A retificação que quebra uma regra é recusada inteira,** com a regra e o porquê.
- **Só se recusa o que a mudança quebra.** O que já estava quebrado antes aparece em
  `ja_quebradas` e não impede corrigir outra coisa.
- **A que passa vira uma versão nova do edital,** com o antes, o depois, o motivo e quem. A versão
  antiga não muda, e toda decisão da trilha continua dizendo sob qual versão foi tomada. A página
  pública do certame é republicada.
- **`-simular` mostra o que quebraria** e quem a mudança alcança (rascunhos, quem ainda não pagou,
  homologados), sem gravar nada.

| O que se prova | Teste |
|---|---|
| Um cronograma de PSS completo não quebra nada | `TestCronogramaValidoNaoQuebra` |
| Cada uma das 8 regras e a fase que fecha antes de abrir, quebradas sozinhas, são acusadas pelo nome, e só elas | `TestCadaRegraQuebraSozinha` — **conferido contra o defeito**: com a homologação sem a margem do boleto, a regra deixa de acusar |
| A regra entre duas fases não vale quando o certame não tem uma delas | `TestFaseQueNaoExisteNaoQuebra` |
| A prorrogação grava a janela, publica a versão 2 com o motivo e diz quem alcança | `TestRetificarProrrogaEPublicaVersao` |
| A que quebra é recusada sem gravar nada; a simulação mostra o mesmo | `TestRetificacaoQueQuebraNaoGrava` — **conferido contra o defeito**: sem a recusa, a janela muda |
| O que já estava quebrado não impede outra retificação | `TestJaQuebradoNaoImpedeOutraRetificacao` — **conferido contra o defeito**: recusando tudo o que quebra, falha |
| Fase encerrada, início de fase aberta, fim no passado, fase que o certame não tem, fase inventada, sem motivo: todos recusados, nenhuma versão publicada | `TestLimitesDaRetificacao` — **conferido contra o defeito**: sem a conferência de fase encerrada, falha |

**Um defeito de desenho achado antes de testar:** a primeira versão recusava a retificação se o
cronograma tivesse **qualquer** quebra, inclusive uma anterior a ela. Um certame com o cronograma já
quebrado nunca mais poderia ser retificado, nem para corrigir a própria quebra por outra fase.

### Prestação de contas (`internal/arrecadacao`, migrações 00014 e 00015)

O dinheiro não passa pelo Lisura: cai na conta da banca, no prestador dela (03-plano §3, "nós
orquestramos o pagamento, não processamos"). Então não há repasse de dinheiro nosso. O que a banca e
o órgão precisam (02-produto §C) é a **conta exata**:

- **Por dia, cargo e cidade de prova:** o que entrou, o que saiu, o saldo e as taxas válidas.
- **A prova de que a conta fecha.**
- **A conferência contra o extrato do prestador.**
- **Dias fechados que não mudam depois.**

**A conta que tem de fechar, centavo por centavo.** Cada pagamento do certame é confirmado, a mais
(duplicidade ou depois da expiração) ou com valor divergente. Cada devolução é de um pagamento a mais
ou de um confirmado de certame cancelado. Então:

> entrou − saiu − a devolver − em divergência = **taxas válidas** (confirmados sem devolução, contados à parte)

Se a igualdade falha, há dinheiro sem explicação, e o relatório diz "a conta não fecha" e por quanto.

**O dia de um pagamento é o dia em que o sistema o registrou,** não o dia em que o candidato pagou.
O boleto pago na sexta e informado na terça entra na terça, como num extrato bancário pela data de
crédito. Assim, um dia fechado só muda por defeito ou adulteração. A devolução entra no dia em que o
prestador a concluiu. Tudo no horário de Brasília, pelo relógio do banco.

**O fechamento do dia** é a fotografia da conta de um dia terminado. Grava os totais e o SHA-256 dos
ids e valores do dia, em ordem: o resumo muda se qualquer pagamento ou devolução do dia mudar, sumir
ou aparecer, mesmo com os totais iguais. O banco recusa alterar ou apagar um fechamento (o mesmo
gatilho da trilha). A API fecha os dias terminados a cada hora, e `banca fechar` fecha na hora.
Fechar de novo não regrava.

**A conferência** (`banca conferir -certame N`, sai com 1 se achar algo) verifica:

- a igualdade;
- cada dia fechado, refeito da base agora;
- cada pagamento e cada devolução concluída daqui contra o extrato do prestador, nos dois sentidos:
  só aqui, só lá ou com valor diferente.

O prestador ganhou o extrato de devoluções (`GET /devolucoes`).

| O que se prova | Teste |
|---|---|
| Certame com cada destino do dinheiro (confirmado; pago em dobro e devolvido; valor divergente; pago em dobro esperando chave): cada total certo, a conta fecha, cada recorte soma o total, e o extrato confere | `TestAContaFecha` — **conferido contra o defeito**: tirando "a devolver" da igualdade, a conta deixa de fechar |
| A devolução apagada daqui: a conta não fecha, e a devolução do prestador aparece sem registro | `TestContaQueNaoFechaEAcusada` — **conferido contra o defeito** (abaixo) |
| Dia de hoje não fecha; fechar de novo não regrava; o banco recusa alterar e apagar; o boleto pago ontem e informado hoje não mexe no dia fechado; empurrar um pagamento para o dia fechado é acusado | `TestDiaFechadoNaoMuda` — **conferido contra o defeito**: com o dia contado pela data do pagamento, falha |
| Pagamento só aqui (aviso forjado ou linha inserida à mão), só lá (nem aviso nem conciliação chegaram) e com outro valor lá: os três acusados | `TestConferirContraOPrestador` — **conferido contra o defeito**: sem comparar o valor, o terceiro passa |
| Centavos em reais (`R$ 1.234.567,89`, negativos) | `TestBRL` |

**Um furo achado ao reler a conferência, antes de rodá-la.** A primeira versão reconhecia a devolução
do prestador como "deste certame" pela **nossa** linha de devolução. Com a linha apagada daqui, a
devolução do prestador passava por "de outro certame" e sumia da conferência, justo o caso que mais
precisa aparecer. Agora quem diz de que certame é a devolução é o **pagamento**. A prova contra o
defeito confirmou: com a regra antiga, só a igualdade acusava; a devolução do prestador não
aparecia.

**Medido no banco local** (05/11, 23h45):

| Certame | Entrou | Saiu | A devolver | Taxas válidas | A conta fecha? | Gerar · conferir |
|---|---|---|---|---|---|---|
| 5 (o do ensaio) | 343.756 pagamentos, R$ 34.385.700,00 | 868, R$ 86.480,00 | 38, R$ 3.880,00 (as recusadas) | 342.850, R$ 34.295.340,00 | **sim** | 0,77 s · 2,7 s, contra o extrato inteiro do prestador, **0 problemas** |
| 10 (cancelado) | 24.885, R$ 2.491.620,00 | 24.885, R$ 2.491.620,00 | 0 | **0** | **sim** | 0,26 s · 0,62 s, **0 problemas** |

No certame cancelado, tudo o que entrou saiu, e não sobrou taxa válida nenhuma: é o que a anulação
promete, agora provado pela conta.

**O fechamento no certame 5.** Quase toda a movimentação local é de 05/11, que ainda não tinha
terminado. O 04/11 tinha 1.495 pagamentos das primeiras rodadas:

| Etapa | Resultado |
|---|---|
| `banca fechar` | 04/11 fechado em **367 ms**: 1.495 pagamentos, R$ 148.880,00, resumo `be424258…` |
| Fechar de novo | Nada regravado |
| `banca conferir` | O dia fechado refeito da base e igual; **0 problemas**; 2,7 s |

### Fila e publicação estática — "o pico de leitura sai do banco" (semana 4)

**Fila** (`internal/fila`): tabela `tarefa` com `FOR UPDATE SKIP LOCKED` — o papel do SQS + Lambda
da arquitetura alvo (o Docker desta máquina só roda contêineres Windows e não há Java, então os
emuladores de SQS não rodam aqui). A tarefa nasce **na mesma transação** da mudança de estado
(não se perde), é **reservada** numa transação curta e executada fora dela (nenhuma conexão do
banco fica presa esperando o prestador), repete com espera crescente (1 s, 2 s, 4 s…) e vira
`FALHOU` depois de 5 tentativas.

**Cobrança assíncrona:** "Pagar com Pix" grava cobrança + tarefa e responde já; o trabalhador chama
o prestador. No pico, um prestador lento atrasa o aparecimento do Pix na tela, não a requisição.

**Publicação estática** (`internal/publicacao`, servida em `/publico` — S3 + CloudFront na produção):

| Arquivo | Quando é gerado | Quem lê |
|---|---|---|
| `certames/{id}.json` — dados públicos | criação do certame, subida do servidor | a página do certame no front (cai para a API se faltar) |
| `certames/{id}/relacao.json` — o **índice** da relação: as fatias que têm inscrição enviada e onde está cada uma | cada fatia republicada (a fatia nova entra) | quem confere a relação |
| `certames/{id}/relacao/{fatia}.json` — uma fatia de 10 mil números de inscrição: número, cargo e situação, **sem dado pessoal** | cada mudança de inscrição **naquela fatia** (mil mudanças viram uma republicação: a tarefa é deduplicada por fatia) | quem confere a relação |
| `cartoes/{código}.html` — cartão de confirmação para imprimir | quando a inscrição passa a valer (paga ou isenta) | o candidato, por um endereço com código que não se deduz do número |

| Garantia | Teste |
|---|---|
| 8 trabalhadores disputando a mesma tarefa: roda uma vez | `TestUmaTarefaUmTrabalhador` — **conferido contra o defeito**: sem a reserva, 8 execuções |
| A mesma tarefa enfileirada 10× enquanto espera: uma linha | `TestEnfileirarDeduplica` |
| Falha repetida: espera crescente, depois `FALHOU` com o erro anotado | `TestFalhaRepeteEDesiste` |
| Mudança chega enquanto a tarefa roda: roda de novo, não conclui com dado velho | `TestReenfileiradaDuranteExecucaoRodaDeNovo` — **conferido contra o defeito**: sem a versão, 1 execução onde deviam ser 2 |
| Tarefa de tipo desconhecido falha com motivo, não some | `TestTipoDesconhecido` |
| "Pagar" responde em menos de 1 s com prestador levando 2 s; o Pix chega pela fila | `TestCobrancaNaoEsperaOPrestador` |
| 50 cliques + 4 trabalhadores: 1 tarefa, 1 cobrança no prestador | `TestCobrancaConcorrenteNaoDuplica` |
| Prestador fora do ar além das tentativas do cliente: tarefa espera, depois sai — uma cobrança | `TestPrestadorForaDoArATarefaRepete` |
| Cartão só depois de valer; nome e CPF **mascarado**, nunca inteiro | `TestCartaoSoDepoisDeValer` |
| Relação sem CPF, nome ou e-mail | `TestRelacaoSemDadoPessoal` |
| A mudança de uma inscrição refaz só a fatia dela; o índice aponta as fatias de todas | `TestMudancaRefazSoAFatiaDela` — **conferido contra o defeito** duas vezes: com toda inscrição na mesma fatia, e sem anotar a fatia nova (o índice sai vazio) |
| Fatia sem anotação (inscrição de antes da anotação existir) entra no índice na subida do servidor | `TestRepublicarTodosAnotaAFatiaQueFalta` — **conferido contra o defeito**: sem a anotação da subida, o índice sai vazio |
| Código do cartão não se deduz do número | `TestTokenNaoSeDeduzDoNumero` |
| Arquivo servido com cache público; inexistente dá 404 | `TestServirArquivoPublicado` |
| Ponta a ponta: Pix pela fila → pago → cartão publicado e aberto **sem sessão**; outro candidato recebe 404 | `TestPagamentoPontaAPontaComAvisosRepetidos` |

**A relação em fatias, medida (06/11/2026, certame 5, 491 mil inscrições, 54 fatias):**

| | Antes | Agora |
|---|---|---|
| Uma republicação | a relação inteira: 3,0 s e 32,7 MB | uma fatia: 50 ms em média, 273 ms no máximo (120 tarefas, 0 falhas); a maior tem 662 KB |
| O índice (`relacao.json`, 3,1 KB) | deduzido das inscrições: 271 ms | lido de `relacao_fatia`: 0,054 ms |

- **Por que uma tabela, e não um índice no banco:** a migração 00018 criou `(certame_id, id)` para
  o índice pular de fatia em fatia. O planejador não o usou e preferiu a chave primária, que é só
  por id. Cada salto atravessava as inscrições dos outros certames intercaladas ali (10.510 linhas
  descartadas por salto). Reescrita com comparação de linha, a consulta piorou para 8,6 s.
- **Agora:** a fatia é anotada na mesma transação da inscrição que a abre. A subida do servidor
  anota as que faltam. A 00019 tira o índice que não servia. As 120 fatias anotadas no banco local
  batem com as 120 deduzidas das inscrições.
- **Defeito achado pela suíte, corrigido:** com a máquina ocupada, a leitura do arquivo estático
  ainda falhava no instante da troca. O Windows devolve acesso negado enquanto o arquivo antigo
  termina de ser apagado. Com 6 cópias do teste ao mesmo tempo, 11 de 90 rodadas tinham 1 ou 2
  erros. A abertura agora repete esse erro por até ~10 ms, e as 90 rodadas passaram sem erro. Só
  afeta a máquina local: no S3 a troca é atômica.

**Provado com o banco desligado (04/11/2026, servidor real, pelo navegador):** com o Postgres parado,
`/api/certames/3`, `/api/eu` e `/api/inscricoes` responderam **500**, e o cartão de confirmação, a
relação e os dados do certame responderam **200** com `Cache-Control: public, max-age=60`. É o
"modo degradado nível 4" da arquitetura alvo: o que a base inteira consulta no pico continua no ar
sem backend.

**Verificado no navegador:** conta nova → inscrição → "Pagar com Pix" mostra "Gerando o seu Pix…"
e o Pix chega pela fila → simular pagamento → **Paga** → "Abrir cartão" abre a página estática com
nome e CPF mascarado.

Os testes acharam um defeito real: o simulador numerava cobranças a partir de `cob_000001` a cada
início, e reiniciá-lo colidia com cobranças antigas. Agora os identificadores levam um prefixo
aleatório por execução, como os de um prestador real.

### Sala de espera e modo degradado — "sob carga, fila honesta em vez de erro" (semana 5)

`internal/degradacao` é um filtro na frente de toda a API (na produção, a sala de espera fica na
borda, no Cloudflare Waiting Room). Os 5 níveis da arquitetura alvo (`fiap-multicloud/04` §4.5)
viram 3 demonstráveis:

| Nível na POC | O que o candidato vê | Nível alvo |
|---|---|---|
| **0** normal | tudo | 0 |
| **1** sala de espera | quem já entrou segue; quem chega recebe uma **senha**, a posição e o tempo estimado, e a tela entra sozinha quando chega a vez | 1 |
| **2** só o essencial | lista de certames, isenção, histórico e carteira de documentos dizem "temporariamente desligada"; entrar, inscrever, pagar e ver a inscrição seguem | 2 e 3 |
| **3** só consulta | a API recusa tudo; o **cartão** (estático) abre, e a tela lista os cartões já vistos naquele aparelho | 4 |

Em **todos** os níveis passam o aviso do prestador (`/psp/aviso`: pagamento que entrou não pode se
perder) e `/saude`. A fila admite **por taxa**: as primeiras `folga` senhas entram já e depois
`taxa_por_segundo`. A senha e a admissão vão em cookies **assinados** (HMAC) — recarregar não perde a
vez e ninguém fabrica uma senha menor. A admissão vence após 15 min **sem uso** (renova a cada pedido:
quem está no meio do formulário não é expulso).

| Garantia | Teste |
|---|---|
| Nível 0 não muda nada nem grava cookie | `TestNormalNaoMexeEmNada` |
| Ordem de chegada; posição e espera certas; recarregar mantém a senha; a fila anda com o relógio; admitido não é barrado no meio | `TestSalaDeEsperaAdmiteNaOrdemDeChegada` |
| Admissão parada há 16 min volta para a fila | `TestAdmissaoVencePorFaltaDeUso` |
| Senha e admissão forjadas (outra chave, tipo trocado, assinatura lixo) não furam a fila | `TestSenhaFalsificadaNaoFuraFila` |
| Sala fechada e reaberta: quem já tinha senha entra antes dos novos | `TestSalaReabertaNaoFazQuemEsperavaPerderAVez` |
| Aviso do prestador e saúde passam nos níveis 1, 2 e 3 | `TestAvisoDoPrestadorNuncaEsperaNemCai` |
| Nível 2: 5 rotas desligadas, 8 críticas seguem (o envio do laudo é crítico) | `TestSoEssencialDesligaOQueNaoECritico` |
| Nível 3: a API recusa tudo | `TestSoEstaticoRecusaTudoNaAPI` |
| Trocar o nível exige o segredo; sem segredo configurado, ninguém troca | `TestOperacaoExigeSegredo` |

**Percorrido no navegador (04/11/2026, servidor real):** nível 1 com 40 senhas na frente → a tela
mostra **41º, cerca de 3 minutos**; recarregar → **39º** (a fila andou, a senha ficou); a operação
libera a fila → a tela entra sozinha no Início. Nível 2 → Início e lista de certames explicam o
desligamento. Inscrição paga → nível 3 → `/api/eu` **503**, cartão **200**, a tela "Modo somente
consulta" lista o cartão; nível 0 → a tela volta sozinha à inscrição.

### Ambiente reproduzível e gerador de carga (semana 6)

O ambiente da AWS está em código ([`../infra`](../infra/README.md)) e o gerador de carga em
[`../carga`](../carga/README.md). **O Terraform não foi aplicado**: esta máquina não tem credencial
AWS, e quem configura a credencial e aceita o custo é o dono da conta. `terraform validate` passa.

Para rodar em várias instâncias atrás de um balanceador, a aplicação mudou:

| Mudança | Teste |
|---|---|
| **A sala de espera mora no Postgres** (nível, taxa, folga e numeração das senhas); cada instância relê a cada 1 s, e só a emissão de senha nova toca o banco | `TestDuasInstanciasUmaFila`, `TestSenhasConcorrentesNaoRepetem` (200 ao mesmo tempo, nenhuma repetida), `TestPostgresReaberturaGuardaABase` |
| **Publicação e documentos no S3** quando `LISURA_S3_BUCKET` está definido; o bucket é conferido na subida (um HEAD não distingue "bucket errado" de "ainda não publicado") | `TestS3GravaComTipoECache`, `TestS3VerificarRecusaBucketInexistente` (S3 falso dentro do teste, `gofakes3`) |
| **Auditoria pós-ensaio** (`cmd/auditoria`): confere no banco o que o k6 viu — inscrições e pagamentos perdidos, cobrança duplicada, pagamento aplicado duas vezes, PAGA sem recebimento | `TestRodadaLimpaNaoAcusaNada`, `TestAcusaCadaDefeito` (os 4 defeitos plantados, cada um acusado) |

**Defeitos achados nesta semana (todos corrigidos):**

1. **O servidor local listava a pasta de cartões.** `GET /publico/cartoes/` devolvia 200 com o
   endereço de todos os cartões (2.994 links), o que anulava o código secreto do endereço criado na
   semana 4. Vinha do `http.Dir` do Go. Agora `arquivo.Pasta` não abre pastas: 404.
   `TestPastaNaoListaDiretorios` e `TestServirArquivoPublicado` foram **conferidos contra o defeito**:
   com o `http.Dir` de volta, a pasta é listada (200) e o teste falha. Só valia para o servidor local;
   na nuvem o CloudFront não tem permissão de listar o bucket.
2. **A senha do banco ia para o log** se a conexão falhasse (a mensagem trazia a URL inteira). Agora
   o log mostra só host, porta e banco.
3. **Várias instâncias subindo juntas migrariam o banco ao mesmo tempo.** Agora a migração usa uma
   trava consultiva do Postgres (goose `SessionLocker`): uma migra, as outras esperam.
4. **Atrás do CloudFront, os cookies sairiam sem `Secure`**: o TLS termina antes do Go, e o código
   olhava só `r.TLS`. Agora também lê `CloudFront-Forwarded-Proto` e `X-Forwarded-Proto`.
5. **No Windows, a leitura de um arquivo falhava enquanto a fila o republicava** (o k6 viu dois 404
   no JSON do certame). Medido isoladamente: 3.076 leituras com erro. Com a abertura que permite
   `FILE_SHARE_DELETE`, 1. Agora a leitura local abre assim (`TestLeituraNaoFalhaDuranteRepublicacao`:
   zero erros em ~500 mil leituras). **Continua:** a gravação por cima de um arquivo aberto ainda
   falha no Windows (83 de 100 com 8 leitores sem pausa); ela tenta de novo por 200 ms e, se não
   der, a tarefa da fila repete com espera. No Linux e no S3 a troca é atômica.
6. **No gerador de carga** (defeitos meus, não do sistema): CPFs repetidos entre rodadas, que fizeram
   o candidato novo cair em inscrições já finalizadas (725 falhas, todas recusadas corretamente pelo
   sistema com 409); e o atendimento especial sem o laudo que o formulário exige (70 falhas, recusa
   correta com 422).

**Validação local** (05/11/2026; 40× por 1 a 2 min, tempo de pensar a 5%; **não é resultado do
ensaio**, prova só que roteiro, API e auditoria funcionam juntos):

| Rodada | O que testou | k6 | Auditoria |
|---|---|---|---|
| 401003 | Caminho completo | 1.730 enviadas, 1.555 pagas, 90 cliques duplos, **0 falhas**, 100% concluídas, p95 crítico 3 ms | banco = k6; 0 cobranças repetidas, 0 pagamentos aplicados duas vezes (com o PSP mandando cada aviso **2×** e perdendo 2%) |
| 401004 | Sala de espera ligada (admite 5/s, chegam 9,8/s) | 1.175 enviadas, 1.048 pagas, **0 falhas**, espera na fila p95 **56 s**; 1.216 senhas, todas admitidas | banco = k6; nada acusado |

### Subir até quebrar, na máquina local (05/11/2026, antes do ensaio na AWS)

Sem a AWS, o mesmo método do escopo foi aplicado localmente: subir a carga até quebrar, achar o que
quebra, corrigir, medir de novo. **O número local não é o resultado do H1**: tudo roda numa máquina
só (12 núcleos, 32 GB), com banco local e tempo de pensar a 5%. O que ele dá é o gargalo do
**código**, que a nuvem também pagaria.

**1. Falso gargalo: o servidor de desenvolvimento do Next.** O primeiro degrau a 160× deu p95 geral
de 30 s, com as requisições críticas da API em 3 ms: estourou só a página `/entrar`, servida pelo
`next dev`, que compila sob demanda. Com o front em modo de produção (o servidor autônomo que vai
na imagem), 160× passou limpo: **6.942 inscrições, 0 falhas, p95 crítico 9,5 ms**.

**2. Gargalo real: a conciliação.** A cada minuto, cada instância baixava 48 h de extrato do
prestador e abria **uma transação por pagamento**, mesmo os já registrados pelo aviso. Medido com o
sistema parado (`xact_commit` do Postgres, 2 min):

| | Transações por minuto sem carga | Uma rodada de conciliação |
|---|---|---|
| Antes | **13.299** (≈ o tamanho do extrato: 12,7 mil pagamentos) | 1 transação por pagamento, crescendo o dia inteiro |
| Depois | **559** (o resto é a fila e a sala de espera consultando o estado) | 12.750 pagamentos conferidos em **~50 ms** |

Num dia de 105 mil pagamentos, o "antes" seriam ~105 mil transações por minuto **por instância**,
disputando o banco com as inscrições — e a rodada não caberia no minuto, emendando uma na outra.
Correção: uma consulta só separa os pagamentos que faltam, e só eles passam pelo registro; uma
trava consultiva do Postgres faz **uma** instância conciliar por rodada. Testes:
`TestConciliacaoSoRegistraOQueFalta` (**conferido contra o defeito**: sem a correção, registra 2
onde faltava 1) e `TestConciliacaoUmaInstanciaPorVez`. A duração de cada rodada agora vai para o log.

**3. Gargalo real que quebrou o sistema: falta de índice.** 320× passou limpo (13.894 inscrições em
2 min, p95 crítico 11,8 ms); 640× quebrou, com o Postgres a 4,6 núcleos e a API quase ociosa (0,9),
esperando conexão. Ligado o `pg_stat_statements` (registra o tempo de cada consulta), o culpado
apareceu — e **não era o suspeito** (a republicação da relação de inscritos: 1,8%):

| Consulta | Chamadas | Média | Do tempo do banco |
|---|---|---|---|
| Cobranças de uma inscrição (a tela pergunta "o Pix chegou?" a cada 1,5 s) | 202.718 | 9,2 ms | **94,3%** |
| Relação de inscritos | 525 | 67 ms | 1,8% |

O plano mostrou leitura da tabela inteira: 58.859 linhas descartadas para achar 1. Os índices de
`cobranca` eram parciais (só `PENDENTE`, só `PAGA`) e esta consulta não filtra por estado; o custo
crescia com cada inscrição. Correção: migração `00006` cria `cobranca (inscricao_id, id)`
(`CONCURRENTLY`, sem travar a tabela). A consulta foi de **13 ms para 0,16 ms**. Comparação justa a
640×, mesma máquina e mesmo k6, só o índice mudando:

| 640× | Sem o índice | Com o índice |
|---|---|---|
| p95 das requisições críticas | **4.672 ms** | **25,7 ms** |
| Jornadas concluídas | 33,5% | 99,7% |
| Falhas | 5.621 | 32 |
| Recargas dos candidatos (a espiral: lento → recarrega → mais lento) | 66.908 | 0 |

As 11 chaves estrangeiras sem índice foram listadas; só esta estava no caminho crítico (as outras
consultas ficaram todas abaixo de 0,3 ms), e índice tem custo em cada gravação.

**4. A fila de tarefas** (05/11). Uma rodada a 640× mostrou o Pix em 11 s (p95). Para saber se era
espera (falta de trabalhador) ou execução (tarefa lenta), a fila passou a medir os dois, por tipo, e
a publicar no log a cada minuto. **A primeira leitura estava errada:** os 11 s vinham do acúmulo de
tarefas deixado pela rodada quebrada anterior; numa rodada limpa, o Pix sai em 1,7 s. O que a
medição mostrou a 640×, por minuto:

| Tarefa | Quantas | Espera na fila | Execução |
|---|---|---|---|
| Publicar cartão | ~14.600 | 0,5 s (máx 1,7 s) | 6 ms |
| Criar cobrança no prestador | ~7.800 | 0,5 s | 2–3 ms |
| Publicar certame e relação | **~115** | 0,2 s | **250–290 ms**, crescendo com o certame |

A espera é capacidade (4 trabalhadores por instância, ~67% ocupados): ajuste de
`LISURA_TRABALHADORES`, não defeito. O defeito era a **relação de inscritos**, refeita do zero ~2
vezes por segundo, relendo todas as inscrições do certame, para um arquivo que o CDN guarda 60 s.
Correção: as republicações da relação são agrupadas em janelas de 30 s (`fila.EnfileirarDepois`).

**A primeira versão da correção não funcionou** (medido: continuou ~100 por minuto). Quando chegava
mudança *durante* a republicação (~300 ms), a fila mandava rodar de novo "já", furando a janela; sob
carga sempre chega, então a relação rodava em laço. O teste novo travou exatamente assim, num laço
sem fim. A janela agora mora na tarefa (migração `00007`) e a nova rodada respeita. Medido a 640×:

| | Antes | Correção que falhou | Correção final |
|---|---|---|---|
| Republicações da relação por minuto | ~115 | ~100 | **2** |
| Pix aparece (p95) | 1,7 s | 4,5 s | **1,6 s** |
| Cartão publicado (p95) | 3,5 s | 11,3 s | **3,4 s** |

Testes: `TestEnfileirarDepoisAgrupaSemAdiar` (**conferido contra o defeito**: se mudança nova adiasse
a janela, a relação nunca sairia sob mudança contínua), `TestMudancaDuranteExecucaoRespeitaAJanela`
(o que travava) e `TestMedidasSeparamEsperaDeExecucao`.

**5. O cliente HTTP do prestador descartava conexões.** Um 500 isolado em "pagar no banco" era a API
sem porta livre para falar com o PSP. O transporte padrão do Go guarda só 2 conexões ociosas por
destino, e o cliente fechava a resposta sem ler o corpo até o fim, o que também impede o
reaproveitamento. Com um prestador real, por HTTPS, seria um aperto de mão TLS por cobrança.
Correção: `psp.NovoHTTP` (128 ociosas por destino) e leitura do resto do corpo antes de fechar.
`TestClienteReaproveitaConexoes`, **conferido contra o defeito**: 640 chamadas abriram **531**
conexões com o transporte padrão e **32** com o corrigido. Na carga, depois da correção, zero
conexões da API para o PSP descartadas em 20 amostras (não há medição de antes em carga: o primeiro
medidor era lento demais e não coletou nada).

**Onde parou, a 640×:** 18.776 inscrições e 16.760 pagamentos em 3 min (90 s de subida, 60 s no patamar, 30 s de descida), p95 crítico 13 ms,
98,8% das jornadas concluídas; auditoria limpa. As ~200 falhas restantes eram conexões que nem
chegavam a abrir, no clique duplo. Elas foram atribuídas, aqui, ao k6 sem portas locais.

> **Corrigido em 05/11, à tarde: a causa era outra.** A máquina ficava, sim, sem portas, mas quem
> as gastava era o **PSP simulado em execução**, um binário anterior à correção 5. O cliente dele
> que manda os avisos abria uma conexão por aviso. Na curva refeita pelo orquestrador, o 640× com o
> PSP antigo deu:
> - 97,35% das jornadas concluídas e 661 falhas, todas de conexão nova;
> - ~1.640 pagamentos por minuto confirmados pela conciliação, cerca de 21%, porque o aviso não
>   chegou (o esperado eram os 2% perdidos de propósito);
> - cartão em 35 s no p95.
>
> Com o PSP novo, nos mesmos parâmetros (`-repetir 2 -perder 0.02`), o 640× deu **100% concluídas,
> 0 falhas**, p95 crítico 14,8 ms, cartão em 5 s no p95 e ~2% confirmados pela conciliação.
> "Subir mais aqui mediria o Windows" estava errado. Os patamares acima de 640× estão logo abaixo.

**6. Acima de 640×: aqui, a máquina acaba antes do sistema** (05/11, à noite). A 1.280× local a
rodada falha em jornadas concluídas. O que separa "limite da máquina" de "limite do sistema" é a CPU,
medida a cada 5 s por um amostrador em Go (`GetSystemTimes` e `GetProcessTimes`). O PowerShell desta
máquina leva ~20 s por `Get-Process`, e o contador de desempenho do WMI travou, então nenhum dos dois
media nada. O amostrador foi conferido com carga conhecida: 3 núcleos girando deram 25%, o esperado.

| 1.280×, 2 min | 4 trabalhadores da fila | 16 trabalhadores |
|---|---|---|
| CPU da máquina no patamar | **99–100%** | **99–100%** |
| k6 · API · Postgres · PSP (% da máquina) | ~30 · ~18 · ~17 · picos de 40 | ~30 · ~16 · ~16 · picos de 40 |
| Jornadas concluídas | 74,35% | 47,15% |
| Cartão publicado (p95) | 127 s | 107 s |
| Inscrições perdidas · cobranças duplicadas | 0 · 0 | 0 · 0 |

A máquina tem 12 núcleos, e ~17–20% deles já estão ocupados parada (os servidores do Next e o resto
do Windows). A 1.280× o próprio k6 come um terço dela. Mais trabalhadores na fila, com a CPU no teto,
pioraram: disputam os mesmos núcleos com o resto. **Aqui, número acima de 640× mede esta máquina, não
o Lisura**: quem responde é a rodada na AWS, com o k6 em outra máquina. Mesmo quebrando, nenhuma rodada
perdeu inscrição ou cobrou duas vezes.

O 1.280× achou três defeitos reais, corrigidos:

- **O PSP simulado aceitava pagar duas vezes o mesmo Pix.** O Pix com identificador (o do QR
  dinâmico) passa a "concluído" no primeiro pagamento, e o banco de quem paga recusa o segundo. No
  simulador, não: 8 inscrições tiveram o mesmo Pix pago duas vezes, com 1,4 a 4 s entre um pagamento
  e outro. O sistema fez o certo: confirmou uma vez e marcou o reembolso. Mas o ensaio media um
  prestador que não existe. Agora o segundo pagamento recebe 409, "este Pix já foi pago". O boleto
  continua aceitando dois pagamentos, como na vida real. Teste: `TestPixSoSePagaUmaVez`,
  **conferido contra o defeito**: sem a trava, os 20 pagamentos simultâneos passaram.
- **O gerador de carga recarregava o "pagar no banco".** É o app do banco, não a página do Lisura.
  Com a trava do Pix, sobraram 128 boletos pagos 2 ou 3 vezes (309 pagamentos). Agora esse passo não
  recarrega.
- **A conciliação não alcançava o boleto informado tarde.** O sistema espera até 3 dias úteis pela
  compensação do boleto (`MargemCompensacao`), mas a conciliação mais larga olhava 48 h para trás.
  Num boleto pago na quinta, com o aviso perdido, o pagamento nunca entrava numa conciliação, e a
  inscrição expiraria paga. As janelas agora são escalonadas (`pagamento.JanelaDaConciliacao`):

  | A cada | Janela |
  |---|---|
  | minuto | 10 min |
  | 10 min | 2 h |
  | hora | 48 h |
  | dia | 7 dias |

  O escalonamento também barateia o minuto. Com 2 h a cada minuto, cada rodada custava o volume de
  2 h: a 1.280×, 43 mil pagamentos, e 7 a 17 s por rodada. `TestJanelasDaConciliacaoCobremTudo`
  confere, para qualquer minuto de partida, que nenhum pagamento escapa:
  - um pagamento de 9 min é conferido na mesma rodada;
  - um de 90 min, em menos de 10 rodadas;
  - um de 30 h, em menos de 60;
  - um de 5 dias, em menos de um dia.

  **Conferido contra o defeito:** com a varredura de volta a 48 h, o teste falha.

**O 640× de novo, com as três correções:**

| Medida | Resultado |
|---|---|
| Inscrições | 27.415 |
| Jornadas concluídas | **100%** |
| Falhas | 0 |
| Perdidas · duplicadas | 0 · 0 |
| Reembolsos | 0 |
| p95 das requisições críticas | 62,8 ms |
| Pix aparece (p95) | 1,6 s |
| Cartão publicado (p95) | 19,5 s |
| CPU da máquina no patamar | média 73%, máximo 99% |

- **O cartão é o tempo mais longo.** Com 4 trabalhadores, a espera de `publicar_cartao` chegou a
  22 s no pico.
- **A conciliação de 10 min** levou de 54 ms a 0,8 s; uma vez, no pico, 4,8 s.
- **O p95 de 14,8 ms do 640× da tarde** não tem CPU medida ao lado, então os dois p95 não se
  comparam. Os dois estão a mais de 30× do limite de 2 s.

Na mesma investigação, a auditoria ganhou duas conferências do "sem cobrar duas vezes":

- **Inscrição com dois pagamentos confirmados:** o sistema ficaria com o dinheiro duas vezes.
- **Pagamento a mais sem reembolso marcado.**

O relatório agora mostra os reembolsos, que não reprovam a rodada, mas quem lê precisa ver. Duas
cobranças PAGA na mesma inscrição já eram barradas pelo índice `cobranca_uma_paga_por_inscricao`.

**Uma suspeita errada, registrada:** os picos de CPU do PSP simulado pareciam vir do extrato, que
varria todos os 272 mil pagamentos com a trava a cada conciliação. A varredura virou busca binária
(`TestExtratoDevolveAJanelaPedida`, conferido contra o defeito), mas o extrato de 2 h só foi de
3,3 s para 2,8 s, e os picos continuaram. A medida passa pelo PowerShell, que nesta máquina é lento,
então não é exata, mas basta para dizer que o custo não estava na varredura. **A causa dos picos não
foi achada.** O PSP é simulado, e na AWS ele roda numa tarefa própria.

**Defeitos do gerador de carga achados nestes degraus** (corrigidos): a conta de usuários virtuais
ignorava o tempo de pensar comprimido e, a 640×, pré-alocava 28 mil para ~3 mil simultâneos; o
disparo de todos juntos estourava a fila de conexões. A primeira medição de 640× sem índice rodou com
esse defeito; por isso a comparação acima foi refeita com o k6 corrigido.

### Gabarito, recurso e resultado — os picos 4 e 5 (05/11/2026, fora do cronograma original)

O escopo põe "publicação estática **de resultado**" dentro e "correção real de cartão-resposta" fora.
`internal/resultado` faz o meio: a banca entrega as respostas lidas pela leitora óptica (a leitura
física fica com ela), e a plataforma corrige, aplica nota mínima, cotas e desempate, classifica, guarda
cada cálculo como versão imutável e publica estático. O painel da banca continua fora: as operações
são o `cmd/banca` (homologar em lote, gabarito, folhas, folhas sintéticas, calcular).

**Regras, conferidas no texto do Planalto em 05/11/2026** (num certame municipal vale a lei do
município, por isso percentual, arredondamento e mínimo de vagas são parâmetros de cada cota):

| Regra | Fonte | No código |
|---|---|---|
| Pessoas pretas, pardas, indígenas e quilombolas: 30%, só com 2 vagas ou mais; fração ≥ 0,5 sobe, < 0,5 desce | Lei 15.142/2025, arts. 1º e 5º | cota `ppiq`: 30, `meio`, mínimo 2 |
| Cotista concorre também na ampla; quem entra pela ampla não ocupa vaga reservada; reservada sem cotista volta para a ampla | Lei 15.142/2025, arts. 7º e 8º | `Calcular`, passos 1 a 3 |
| Pessoa com deficiência: no mínimo 5%, qualquer fração sobe; vaga sem aprovado pode ir para a ampla | Decreto 9.508/2018, art. 1º §§ 1º, 3º e 5º | cota `pcd`: 5, `cima`, mínimo 5 (assim a 1ª vaga reservada é 20% de 5) |
| Primeiro desempate: idade, preferindo o mais velho, entre quem tem 60 anos ou mais | Estatuto da Pessoa Idosa, art. 27, parágrafo único, e art. 1º | antes de qualquer critério do edital |
| Questão anulada vale para todos os presentes | prática dos editais (não é lei) | `*` no gabarito |

As notas são inteiras, em **centésimos de ponto**: nota de concurso não pode depender de arredondamento
de ponto flutuante. Empate que os critérios do edital não desfazem é **marcado** (`empate: true`), não
escondido: a lei manda a sorteio, e a ordem por número de inscrição é só para o arquivo ser
reprodutível.

**Publicação** (pela fila, como o cartão): `certames/{id}/gabarito.json` (versão em vigor, anuladas
marcadas, histórico), `certames/{id}/resultado.json` (listas de ampla e de cada cota, só aprovados,
por número de inscrição e nota, e o histórico de versões) e o resultado individual em **256 fatias**
`certames/{id}/resultados/{xx}.json`. A chave da linha de cada candidato é um resumo de mão única do
código do cartão, que o front já guarda no aparelho: no pico 5 a tela calcula a chave e baixa a
fatia, **sem API e sem banco**. Quem baixa todas as fatias vê chaves, nota e situação, mas não chega
ao cartão (nome) de ninguém. Um arquivo por candidato seriam 200 mil gravações por recálculo; um
arquivo só, dezenas de MB por candidato.

**Recurso contra o gabarito** (02-produto §6): um por questão por inscrição, com fundamento escolhido
de uma lista (é o que permite agrupar), dentro da janela `gabarito` do cronograma. A banca vê uma
linha por questão (quantos recursos, por fundamento, que letra pediram), decide uma vez (mantém,
altera, anula) e cada recorrente recebe essa decisão como resposta. O definitivo só sai com toda
questão recorrida decidida, e trava o preliminar.

| Garantia | Teste |
|---|---|
| Anulada vale para todos os presentes, marcaram o que marcaram; branco e rasura valem zero; sem folha é ausente | `TestCorrecao` |
| Reserva de vagas de 1 a 41 vagas, com as duas regras de arredondamento e o mínimo de vagas | `TestReservaDeVagas` |
| Nota mínima elimina; no limite, passa | `TestNotaMinimaElimina` |
| Idoso antes de todos (incluindo quem faz 60 no dia), mais velho entre idosos, depois disciplina e idade; empate que sobra é marcado | `TestDesempate`, `TestSemNascimentoNaoDesempataPorIdade` |
| Cotista que entra pela ampla não ocupa vaga reservada | `TestCotistaNaAmplaNaoOcupaVagaReservada` — **conferido contra o defeito** (este e o ponta a ponta falham) |
| Vaga reservada sem cotista aprovado volta para a ampla | `TestVagaReservadaSemCotistaVoltaParaAAmpla` |
| Ponta a ponta: folhas, gabarito, cálculo, publicação; anulação no definitivo recalcula todos; a versão 1 fica intacta no banco; nada de nome, CPF ou e-mail nos arquivos | `TestResultadoPontaAPonta` |
| Folha que não confere é recusada com motivo (6 motivos), o resto entra; reimportar substitui | `TestImportarFolhasRecusaOQueNaoConfere` |
| Gabarito confere tamanho e letras, exige motivo, não muda depois de gravado; a prova fica fixa depois do 1º gabarito | `TestGabaritoConfereAProvaEFixaAProva` |
| Homologação em lote pela máquina de estados, com o analista na trilha; rodar de novo não faz nada | `TestHomologarLotePelaMaquinaDeEstados` |
| 5 cálculos simultâneos: 5 versões distintas, cada uma inteira; resultado gravado não muda | `TestCalculosConcorrentesGanhamVersoesDistintas` |
| 60 recursos simultâneos em 2 questões viram 2 linhas para a banca; cada recorrente recebe a decisão; o definitivo espera todas as decisões e fecha o preliminar | `TestMuitosRecursosUmaDecisaoPorQuestao` |
| Recurso fora da janela, de outro candidato, de inscrição não homologada, de questão inexistente, sem argumento, com fundamento inventado: recusado; um por questão; decisão não muda | `TestRecursoRecusado` |
| 6 consolidações simultâneas: um definitivo só | `TestConsolidacoesConcorrentesUmDefinitivo` — **conferido contra o defeito**: sem a trava do cargo, 5 colisões. A 1ª versão do teste passava mesmo sem a trava (as 6 não chegavam a se sobrepor); agora uma transação do teste segura todas no ponto de conflito |
| Pela API: a rota do resultado só responde ao dono; a fatia abre sem sessão | `TestResultadoPeloEnderecoEstatico`, `TestRecursoPelaAPI` |

**Medido na máquina local (05/11/2026), certame com 200 mil inscrições homologadas** (inseridas direto
no banco: o caminho até a homologação já tem os próprios testes e o ensaio de inscrição; 30% ppiq, 5%
pcd, nascimentos de 1950 a 2006), 20 questões, 12% de ausentes nas folhas sintéticas (`banca sintetico`):

| Etapa | Tempo |
|---|---|
| Importar 175.878 folhas lidas | 8,3 s |
| Calcular o resultado preliminar (corrigir, cotas, desempate, classificar, gravar a versão) | **9,9 s** |
| Anular a questão 7 no definitivo e **recalcular todo mundo** (a versão 1 fica) | **10,2 s** |
| Publicar pela fila: 256 fatias (143 KB em média, **19 KB com gzip**, que o CloudFront aplica) + listas (9 MB) | as fatias saem em ~1 s |

O recálculo em massa — o "efeito dominó" da anulação — cabe em segundos, não em horas: não é ele que
segura o resultado. Cada versão ocupa ~42 MB no banco (210 bytes por inscrição).

**Pico 5 local** (`carga/pico-resultado.js`, 1.000 candidatos por segundo abrindo o próprio resultado,
2 min de patamar): **139.999 aberturas, zero falhas, toda linha achada, p95 0,6 ms**. O banco fez
**12,7 transações/s parado, 9,3 durante o pico e 9,3 depois** — o pico não chega ao banco. (O p95 é
do servidor de arquivos local; na nuvem quem responde é o CloudFront. O que o número local prova é a
parte do banco.)

De quebra, a medição achou um falso tempo de fila: a tarefa de publicação aparecia com 10 s de
"espera" porque nascia com `now()`, que no Postgres é a hora em que a transação começou — e o cálculo é
uma transação de 10 s. Agora a tarefa nasce com `clock_timestamp()`.

### Migrações: a rota de fuga testada

`TestCadaMigracaoDesfazERefaz` aplica, desfaz e reaplica cada migração, uma por vez, e compara o
schema inteiro (colunas, restrições, índices, gatilhos, funções) com o de antes e o de depois.
**Conferido contra o defeito:** sem o `DROP COLUMN necessidades` no `Down` da 00002, o teste aponta
a coluna que sobrou. Desfazer uma implantação ruim na véspera do prazo é uma das 7 causas de queda;
rota de fuga que nunca rodou não é rota.

### Detector de corrida (`go test -race`)

O detector de corrida do Go acusa dois goroutines mexendo na mesma memória sem trava; precisa de
compilador C, que esta máquina não tinha. Instalado o **llvm-mingw 20261020** (o compilador que os
próprios servidores de build do Go usam no Windows), versão com mais de duas semanas de publicada e
SHA-256 conferido contra o registrado pelo GitHub. **A suíte inteira passou com `-race`, zero
corridas** (05/11). Para o zero valer, o detector foi conferido com uma corrida plantada (dois
goroutines somando no mesmo inteiro): acusou `DATA RACE`.

### Trilha encadeada (migração 00010)

Cada evento da trilha guarda SHA-256(resumo do evento anterior da mesma inscrição ‖ conteúdo). O
resumo é calculado por gatilho (quem grava não escolhe o próprio) e **refeito em Go** na verificação
(quem adulterasse o banco poderia adulterar uma função do banco também). O cartão de confirmação
imprime o **selo** (16 dígitos do resumo do último evento), e a auditoria pós-ensaio confere as
correntes da rodada.

| O que um superusuário faz | O que acontece | Teste |
|---|---|---|
| Desliga a imutabilidade e troca o motivo de um indeferimento | A corrente quebra exatamente naquele evento | `TestAlterarEventoAntigoQuebraACorrente` |
| Apaga um evento do meio | Quebra no evento seguinte | `TestApagarEventoQuebraACorrente` |
| Reescreve a corrente inteira, recalculando cada resumo | A verificação interna passa (é consistente) — **só o selo do cartão denuncia** | `TestReescreverTudoSoOSeloDenuncia` |
| Grava um evento com um resumo escolhido | O gatilho substitui | `TestResumoGravadoEOCalculado` |
| Adultera a trilha de uma inscrição de uma rodada do ensaio | A auditoria acusa `TRILHA ADULTERADA` | `TestAcusaCadaDefeito` |

No banco de desenvolvimento, a migração encadeou os **560.925 eventos** que já existiam (~80 s, uma
vez só), e `auditoria -trilhas` refez em Go a corrente de **205.471 trilhas em 0,96 s: nenhuma
quebrada** — a conta do gatilho e a da verificação batem também sobre dado real, não só no teste.

### Homologação em lote e prioridade na fila (migração 00011)

Medido com 100 mil inscrições pagas, cada uma com trilha (certame 8 local, 05/11):
- **Homologação:** `banca homologar` levou **82,7 s** (~1.200 inscrições/s). Cada inscrição passou
  por análise, deferimento e homologação na máquina de estados: 300 mil eventos novos.
- **O efeito na fila:** os 100 mil cartões republicados levaram ~4 min, sem nenhuma falha, com pico
  de **75.877 tarefas vencidas** esperando trabalhador.
- **O problema:** a fila não tinha prioridade. A republicação da relação de inscritos esperou
  **1 min 40 s** atrás dos cartões, e um Pix pedido naquela hora esperaria o mesmo, com o candidato
  olhando a tela.

A correção é a prioridade por tarefa, escolhida por quem enfileira, porque o mesmo cartão é normal
quando o candidato acabou de pagar e lote quando a homologação republica 100 mil:

| Prioridade | Quem usa |
|---|---|
| Urgente | Criar a cobrança no prestador (o Pix) |
| Normal | Cartão de quem mexeu na inscrição agora, relação, gabarito, resultado |
| Lote | Cartões republicados pela homologação em lote |

Reenfileirar uma tarefa pendente com prioridade mais alta a promove; com mais baixa, não a rebaixa.
**Conferido contra o defeito:** com a ordem antiga (só por data), `TestUrgenteENormalPassamNaFrenteDoLote`
falha com `[lote lote lote]`.

**Medido de novo, com a API nova** (certame 9, outras 100 mil inscrições, 05/11):

| | Sem prioridade (certame 8) | Com prioridade (certame 9) |
|---|---|---|
| Espera da relação de inscritos | 1 min 40 s | **1 a 59 ms**, com até 78.964 cartões de lote na fila |
| 100 mil cartões republicados | ~4 min, 0 falhas | ~4 min (255 s), 0 falhas |
| Homologação | 82,7 s | 72,3 s |

O lote não ficou mais rápido, e nem era a ideia: ele só deixou de passar na frente.

### Âncora da trilha fora do banco (`internal/ancora`)

A corrente sozinha não pega quem a reescreve inteira; o selo do cartão pega, mas só para quem
guardou o cartão. A **âncora** é a mesma testemunha para todos:
- **Quando:** a cada hora (`LISURA_ANCORA_INTERVALO`), ou por `banca ancorar` no fim de uma fase.
- **O que grava:** o último evento e o resumo de cada inscrição do certame (completa, uma por dia)
  ou só das que mudaram (incremental, entre elas), num arquivo de texto simples que dá para
  conferir sem esta aplicação. Cada âncora traz o resumo da anterior.
- **Onde:** num **cofre** que não aceita alteração, o bucket S3 `cofre` com bloqueio de objeto
  (`infra/terraform/cofre.tf`); na máquina local, uma pasta que só imita a regra.
- **O recibo:** cabeçalho e resumo SHA-256 da âncora, publicado no site estático
  (`/publico/certames/{id}/ancoras/NNNNNN.txt`), onde qualquer um pode guardá-lo.
- **Sem mudança, sem âncora:** se a contagem de eventos não mudou desde a última, nada é gravado.

`auditoria -ancoras` confere o banco contra todas as âncoras. Como o resumo de um evento cobre a
corrente inteira até ele, conferir o ponto ancorado confere o caminho, desde que a corrente também
confira por dentro (`auditoria -trilhas`).

| O que alguém faz | O que a conferência acusa | Teste |
|---|---|---|
| Reescreve a corrente inteira de uma inscrição, consistente por dentro | `TRILHA_DIFERENTE_DA_ANCORA` | `TestReescritaDepoisDaAncoraEDenunciada` |
| Reescreve e deixa o sistema ancorar de novo, para "lavar" | `RESUMO_MUDOU_ENTRE_ANCORAS`: a âncora velha continua no cofre | `TestReancorarNaoLavaAReescrita` |
| Apaga a trilha de uma inscrição ancorada | `EVENTO_ANCORADO_SUMIU` | `TestApagarInscricaoAncoradaEDenunciado` |
| Enfia uma inscrição com data anterior à âncora | `INSCRICAO_FORA_DA_ANCORA` (folga de 10 min); a mesma inscrição com data real passa | `TestInscricaoEnfiadaNoPassadoEDenunciada` |
| Troca ou apaga uma âncora antiga (só possível no cofre local) | `CADEIA_DE_ANCORAS_QUEBRADA`, `ANCORA_FALTANDO` | `TestCofreMexidoEDenunciado` |
| Duas gravações na mesma chave do cofre, inclusive 8 ao mesmo tempo | Uma vence; as outras recebem `ErrJaExiste` (disco por link; S3 por `If-None-Match: *`) | `TestCofreDiscoNaoSobrescreve`, `TestCofreS3NaoSobrescreve` |
| Várias instâncias ancorando enquanto a trilha anda, sem trava | Nenhuma âncora sai de uma fotografia mais velha que a anterior | `TestAncorasDisputadasNuncaAndamParaTras` |
| Reescreve uma inscrição que não mudou desde a completa | A incremental seguinte a leva; `RESUMO_MUDOU_ENTRE_ANCORAS` | `TestIncrementalSoLevaOQueMudou` |
| Âncora gravada sem recibo | O recibo sai na rodada seguinte | `TestReciboQueFaltouSaiNaProximaRodada` |

Medido com as 100 mil inscrições do certame 8:

| Etapa | Resultado |
|---|---|
| Âncora com 300 mil eventos | 0,65 s, 7,9 MB (79 bytes por inscrição) |
| Âncora com 600 mil eventos, depois da homologação | 1,8 s |
| Âncora sem mudança | 0,5 s, nada gravado |
| `auditoria -ancoras`, 2 âncoras | 200 mil pontos conferidos em 2,2 s, 0 problemas |
| `auditoria -trilhas`, o banco todo | 305.471 trilhas em 3,6 s, 0 quebradas |

Na API em execução, a rotina ancorou o certame 5, que concentra as rodadas de carga (205.465
inscrições, 560.911 eventos), em **0,89 s**.

**Âncora incremental.** Uma completa com 200 mil inscrições tem ~16 MB. Uma por hora em 60 dias de
inscrição dariam ~23 GB por certame no cofre, quase tudo repetido. Por isso só sai uma completa por
dia (`CompletaCada`). Entre elas, a âncora leva só as inscrições cujo ponto mudou, e a conferência
soma tudo. Uma inscrição reescrita sem evento novo também muda de resumo e entra na incremental.
**Conferido contra o defeito:** comparando só o número do evento, a reescrita ficava de fora da
incremental, e dois testes falham. Mesmo assim a conferência ainda acusou a reescrita pelo ponto
antigo, que continua sendo conferido contra o banco.

**Recibo que faltou:** a âncora gravada pela `banca` (que não publica) ou com o site fora do ar
ganha o recibo na rodada seguinte da rotina. Isso cobre as âncoras desde a última completa; recibo
anterior a ela que tenha faltado não volta (é o caso da âncora 1 do certame 8 local).

**Defeito achado ao escrever o teste de concorrência:** a primeira versão lia a última âncora do
cofre *depois* de fotografar o banco. Uma instância que fotografasse antes de outra e lesse o cofre
depois dela gravaria uma âncora mais velha que a anterior, e a conferência acusaria falsamente
"corrente andou para trás". A trava entre instâncias escondia isso. Agora a leitura vem antes da
fotografia; a trava só poupa trabalho.

### Documentação viva e observabilidade (06/11/2026)

"Viva" quer dizer: cada página nasce do que roda ou do código, e um teste quebra a suíte quando ela
fica para trás.

| Página | O que mostra | De onde vem | O que a mantém viva |
|---|---|---|---|
| `/docs` | A entrada, com o que mantém cada página viva | o front | — |
| `/docs/api` | Todas as rotas de `/api`, corpo, respostas, erros, e o botão para chamar cada uma | o Scalar, a partir de `GET /api/openapi.yaml` (`internal/api/openapi.yaml`, embutido no binário) | `TestOpenAPICobreTodasAsRotas`: as rotas vêm de uma tabela (`rotas()`), e rota sem documentação, ou documentação de rota que não existe, quebra |
| `/design` | O sistema de design: cores com o contraste de cada uma, tipografia, raios, sombras, toque; o Ant Design com o tema; cada componente do portal e cada bloco das telas em todos os estados | o Storybook estático (`npm run design` grava em `public/design`); a API é simulada por história | as histórias importam os componentes reais: mudou a assinatura, o build quebra |
| `/docs/infra` | O caminho de um pedido, cada peça local e na AWS, e o Terraform: recursos com o comentário que os explica, quem usa quem (diagrama), variáveis e saídas | `cmd/infradoc` lê `infra/terraform/*.tf` e grava `poc-web/public/docs/infra.json` | `TestInfraDocAtual` (o JSON publicado é o que o gerador produz agora) e `TestInfraDocTemTodoRecurso` |
| `/operacao` | Dispara ensaios (último dia, estresse até quebrar, pico do resultado), muda o nível de degradação, e embute os painéis do Grafana | `cmd/ensaiador` (executor do k6, só local), Prometheus e Grafana | os testes do executor |

**As métricas** (`internal/metricas`, `GET /metrics` fora de `/api`): requisições e tempo por rota,
pelo **padrão** da rota e nunca pela URL (com o número da inscrição no rótulo, cada inscrição viraria
uma série e o Prometheus estouraria a memória); tarefas da fila por tipo e resultado, espera e
execução; pendentes e vencidas, lidas do banco a cada raspagem pelo índice parcial (só as pendentes);
trabalhadores de pé e rodando; nível da sala de espera e senhas; conexões do pool; CPU, memória e
goroutines; e `lisura_instancia_info`, uma série por processo, que faz o painel mostrar o servidor
subindo, caindo ou trocando de versão.

**O Grafana** sobe só leitura e sem login (não há senha de administrador para vazar). O painel é um
arquivo do repositório (`observabilidade/grafana/paineis/lisura.json`) com anotações automáticas no
início e no fim de cada ensaio, em cada mudança do nível de degradação e quando muda o número de
instâncias. O k6 escreve no Prometheus por remote write (`--out experimental-prometheus-rw`).

**O executor de ensaios** só escuta em 127.0.0.1, roda um ensaio de cada vez, e só aceita pedido com
o cabeçalho `X-Lisura-Painel` e a origem do front. Outro site aberto no navegador não consegue
disparar um ensaio, porque o cabeçalho exige uma pré-verificação CORS que ele não autoriza. O segredo
de operação fica no executor, não na tela. A escada dobra o multiplicador até um patamar quebrar (o
k6 estourando um limite, ou a auditoria acusando algo) ou até o teto.

**Na AWS:** o painel do CloudWatch (`infra/terraform/observabilidade.tf`, saída `painel_operacao`)
mostra tarefas no ar por serviço, CPU e memória, requisições e 5xx no balanceador, o p95 com a linha
do H1 e a capacidade do Aurora. As métricas da aplicação ficam em `/metrics` dentro da VPC. Levá-las
a um Prometheus e Grafana gerenciados **ainda não existe**, e nada disso foi aplicado.

| O que se prova | Teste |
|---|---|
| Duas inscrições diferentes são a mesma série; o que não casa com rota é uma série só; a URL nunca vira rótulo | `TestRequisicaoMedidaPeloPadraoDaRota` — **conferido contra o defeito**: rotulando pela URL, falha |
| Cada tarefa da fila conta por tipo e resultado, e entra nos histogramas | `TestTarefaDaFilaMedida` |
| Toda rota do servidor está documentada, e toda documentada existe | `TestOpenAPICobreTodasAsRotas` — **conferido contra o defeito**: uma rota nova sem documentação quebra |
| A escada dobra até o primeiro patamar que quebra, cada um numa faixa de CPF e auditado com o que o k6 contou | `TestEscadaParaNoPrimeiroPatamarQueQuebra` — **conferido contra o defeito**: sem subir, falha |
| A auditoria acusar algo quebra o patamar, mesmo com o k6 dentro dos limites | `TestAuditoriaComProblemaQuebraOPatamar` — **conferido contra o defeito**: olhando só o k6, falha |
| Um ensaio de cada vez; parar encerra o k6 sem auditar o meio-patamar | `TestUmEnsaioDeCadaVezEPararEncerra` |
| Pedido inválido nem começa | `TestPedidoInvalidoNemComeca` |
| Só o painel dispara: sem o cabeçalho, de outro site ou de outra porta, 403 | `TestSoOPainelDispara` — **conferido contra o defeito**: sem conferir a origem, outro site dispara |
| A página de infraestrutura é a do Terraform de hoje, com todo recurso | `TestInfraDocAtual`, `TestInfraDocTemTodoRecurso` — **conferido contra o defeito**: com um leitor de blocos que perde recursos, falha |

Os testes do executor usam um k6 falso: o próprio binário do teste, chamado de novo, imprime o
`PARA_AUDITAR` e sai com o código pedido. Assim a escada, a parada e a auditoria se provam sem carga
de verdade.

**Versões instaladas** (todas com pelo menos 2 semanas, checksums conferidos): Prometheus 3.13.5 e
Grafana 13.2.4 (binários nativos para Windows: o Docker desta máquina só roda contêineres Windows),
`client_golang` 1.25.0, Scalar 0.9.82, Mermaid 12.2.0, Storybook 10.6.3. O front foi reinstalado com
`npm install --before=2026-10-23T20:00Z`, que limita também as dependências indiretas.

**Achado ao instalar:** o antd do projeto estava em 6.6.8, publicado em 26/10, contra a regra das
2 semanas. Foi fixado em 6.6.7 (20/10).

## Decisões e limites conhecidos

- **A sala de espera passou para o Postgres na semana 6.** Na borda (desenho alvo), ela não
  consumiria nada do backend; aqui, um `nextval` por visitante novo.
- **A fila de tarefas consulta o banco mesmo vazia** (parte das 559 transações por minuto sem
  carga). É pouco por instância; com muitas instâncias, `LISTEN/NOTIFY` do Postgres acordaria os
  trabalhadores só quando houvesse tarefa.
- **Pedidos simultâneos de quem ainda não tem senha recebem senhas diferentes**; o navegador fica com a
  última e as outras viram "senhas fantasmas", que atrasam a fila em 1/taxa segundo cada. No uso real
  a primeira chamada é uma só (`/eu`). No teste saíram 42 senhas para 41 visitantes; a causa
  provável (não confirmada) é o `next dev`, cujo modo estrito do React faz essa chamada em dobro.
- **O nível muda à mão** (`POST /operacao`). Na produção sobe sozinho por métrica (latência, conexões
  do banco); o ensaio de carga vai acioná-lo pela mesma rota.
- **A rota de operação passa pelo repasse do Next** e fica alcançável de fora, protegida só pelo
  segredo. Na produção ela fica na rede interna.
- **Não verificado no navegador:** a sala de espera na tela de **entrar** (mesmo componente; o cookie
  de admissão de 15 min do teste impedia voltar à fila sem apagar cookies `HttpOnly`).

- **Login sem senha (`POST /sessoes`) é um substituto da POC**, aceitável só porque todos os
  candidatos são sintéticos. Em produção, gov.br.
- **Formato inválido é erro de formulário (422), não indeferimento.** A triagem por regra só fecha o
  que consegue sozinha (NIS de outro CPF); o resto vai para análise humana.
- **Os passos do formulário não entram na trilha** — só as transições de estado.
- **Desvio do diagrama (02-produto §5):** recurso provido volta para `DEFERIDA`; `HOMOLOGAR` é um
  evento só, na lista definitiva.
- **Cancelar só a partir de `RASCUNHO`**, como no diagrama.
- **A trilha é imutável contra a aplicação e, desde 05/11, encadeada contra o superusuário**
  (migração 00010): ele ainda consegue desligar o gatilho, mas alterar ou apagar um evento quebra a
  corrente, e reescrever a corrente inteira não reproduz o selo impresso no cartão. Desde 05/11 a
  trilha também é **ancorada fora do banco** (ver acima). O que a âncora não pega: adulteração
  entre um evento e a âncora seguinte (até 1 h), e quem tem ao mesmo tempo o banco e a permissão de
  furar a retenção do cofre — no modo `GOVERNANCE` do ensaio, essa permissão existe; no `COMPLIANCE`
  de produção, não existe para ninguém. O bloqueio de objeto **não foi testado contra a AWS real**
  (só contra um S3 falso, que não implementa retenção).
- **Proteção da escrita por cookie usa `Sec-Fetch-Site`** (todo navegador atual envia), não `Origin`:
  pelo repasse do Next, o `Host` que chega ao Go não é o do navegador.
- **Migrações de reversão testadas e `go test -race` rodando** desde 05/11 (ver acima). O `-race`
  exige compilador C: `C:\Users\rockb\tools\llvm-mingw`, com `CGO_ENABLED=1` e `CC=clang`.
- O portal em HTML no servidor (`internal/web`) foi **removido em 04/11/2026** por decisão do
  usuário; o front é o Next.
