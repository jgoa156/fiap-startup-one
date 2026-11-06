# Continuar o trabalho — Lisura (FIAP)

*Documento de retomada. Foi escrito para quem chega frio — outra sessão, outro dia, outro
assistente. Tudo que é preciso saber para continuar sem refazer decisão já tomada está aqui.*

**Última atualização:** 5 de novembro de 2026, à noite (semana 6 da POC + picos 4 e 5 + âncora da
trilha e prioridade na fila + ensaio local acima de 640×, no back-end).

---

## 1. O que é isto

Dois trabalhos de pós-graduação da FIAP, ligados entre si:

| Trabalho | O que pede | Pasta |
|---|---|---|
| **Startup One** | Conceber uma startup do zero, com plano de negócios, para apresentar a investidores. **Precisa ser implementada** como POC | `fiap-startup-one/` |
| **Soluções Multicloud** | Defender a **adoção ou não adoção** de estratégia multicloud para a empresa do Startup One | `fiap-multicloud/` |

**Premissa do trabalho de multicloud:** a empresa **não é mais uma startup** — está consolidada,
pós-breakeven, discutindo evolução de arquitetura por causa de crescimento ou nova rodada.

**Formato da apresentação de multicloud:** **1 apresentador**, 4 a 6 minutos de defesa + 2 a 4
minutos de discussão.

**Critérios de avaliação explícitos:** tempo usado (4–6 min) · referências de institutos de
mercado (Forrester ou outros) com dados e pesquisas · descrição das arquiteturas atual e alvo
**citando exemplos de soluções de mercado**.

---

## 2. Estado atual

> **Está pronto para apresentar.** Slides, roteiro e pitch estão em PDF e revisados.

| Entregável | Onde | Status |
|---|---|---|
| Slides do multicloud | `fiap-multicloud/apresentacao/Lisura-multicloud-slides.pdf` | ✅ 8 slides, 16:9, 5min30 |
| Roteiro do apresentador | `fiap-multicloud/apresentacao/Lisura-multicloud-roteiro.pdf` | ✅ fala palavra a palavra |
| Pitch do Startup One | `fiap-startup-one/apresentacao/Lisura-pitch.pdf` | ✅ 10 slides |
| Previews em PNG | `*/apresentacao/previews/` | ✅ um PNG por slide |
| Diagrama da arquitetura | `fiap-multicloud/diagramas/arquitetura-alvo.png` | ✅ 2880×2040 |
| Documentação dos dois trabalhos | `fiap-*/0*.md` | ✅ completa |

### O que falta

1. ~~Ensaiar com cronômetro~~ — **a apresentação de multicloud já foi feita** (confirmado pelo
   usuário em 29/10/2026).
2. **Implementar a POC** — escopo em `fiap-startup-one/04-escopo-poc.md`. **É a única pendência
   aberta.** Duas pastas: **`fiap-startup-one/poc/`** (API Go, banco, PSP simulado — README com
   estado, rotas e como rodar) e **`fiap-startup-one/poc-web/`** (front Next).
   - **Feito:** semanas 1 e 2 (29/10) e o **front em Next.js 16 + React 19 + Ant Design 6**
     (04/11), no layout do Inclusa com a paleta Grafite: entrar, início, certames, certame,
     inscrição em 7 passos com salvamento automático, Pix/boleto, histórico. **Percorrido de ponta
     a ponta no navegador** (até "Paga"), desktop e celular. 20 testes Go contra o Postgres real.
   - **Semana 3 (pagamento) concluída em 04/11:** 12 testes de pagamento contra o PSP simulado
     (cobrança concorrente, resposta perdida, aviso repetido/forjado/perdido, Pix + boleto, boleto
     pago 2×, pagamento depois da expiração, margem de compensação, valor divergente), dois
     conferidos contra o defeito. **32 testes Go no total.** Tabela em `poc/README.md`.
   - **Semana 4 concluída em 04/11:** fila de tarefas no Postgres (papel do SQS + Lambda),
     cobrança assíncrona (o PSP saiu do caminho do clique), publicação estática em `/publico`
     (dados do certame, relação sem dado pessoal, cartão de confirmação). **Provado com o banco
     desligado:** API 500, cartão/relação/certame 200. 43 testes Go.
   - **Semana 5 concluída em 04/11:** `internal/degradacao` — sala de espera com senha em cookie
     assinado e admissão por taxa, e 3 níveis (1 sala de espera · 2 só o essencial · 3 só
     consulta), trocados por `POST /api/operacao` com `LISURA_OPERACAO_SEGREDO`. O aviso do
     prestador passa em todo nível. Front: tela "Muita gente acessando agora" (posição + tempo,
     entra sozinha) e "Modo somente consulta" (lista os cartões guardados no aparelho). **52
     testes Go.** Os 3 níveis percorridos no navegador. De quebra, dois defeitos visuais antigos
     corrigidos: fundos de alerta escuros (derivados do grafite) e o ✓ invisível nos passos
     concluídos (classe mudou no antd 6).
   - **Semana 6 concluída em 05/11 — mas o ambiente NÃO foi aplicado na AWS:** esta máquina não
     tem credencial AWS, e configurá-la é do usuário. Feito: `fiap-startup-one/infra/` (Terraform
     validado: VPC, Aurora PG 17 Serverless v2, S3 + CloudFront, balanceador, ECS Fargate, ECR,
     CodeBuild, Secrets Manager; Dockerfiles; `scripts/publicar-imagens.ps1` e `scripts/tarefa.ps1`
     para seed/k6/auditoria) e `fiap-startup-one/carga/ultimo-dia.js` (k6, perfil do último dia;
     40× = 35.160 inscrições/h, derivado de 7,7 mi/ano). A sala de espera foi para o Postgres;
     publicação e documentos vão para o S3; `cmd/auditoria` confere no banco o que o k6 viu.
     **62 testes Go.** Validação local a 40× (tempo de pensar a 5%): 0 falhas, banco = k6, 0
     cobranças duplicadas, sala de espera com fila de até 56 s sem falha. Defeitos achados e
     corrigidos: o servidor local **listava a pasta de cartões** (vazava o endereço de todos — desde
     a semana 4), senha do banco no log de erro, migração concorrente, cookie sem `Secure` atrás do
     CloudFront, leitura falhando no Windows durante a republicação. Desvios da arquitetura alvo
     (ECS em vez de EKS etc.) com o porquê em `infra/README.md`.
   - **05/11, sem AWS (o usuário pediu para seguir desenvolvendo): "subir até quebrar" na máquina
     local.** Dois gargalos reais achados e corrigidos, com medição antes/depois: (1) a conciliação
     fazia uma transação por pagamento do extrato a cada minuto em cada instância — 13.299
     transações/min com o sistema parado, agora 559, e só uma instância concilia por rodada;
     (2) faltava índice em `cobranca.inscricao_id` — 94% do tempo do banco; a 640× o p95 crítico foi
     de 4.672 ms para 25,7 ms e as jornadas concluídas de 33,5% para 99,7% (migração 00006).
     Depois, a fila: (3) a relação de inscritos era refeita ~115 vezes/min — agora agrupada em
     janelas de 30 s, 2/min (migração 00007; a primeira tentativa falhou porque a mudança durante a
     execução furava a janela); a fila passou a medir espera × execução por tipo e a logar a cada
     minuto; (4) o cliente HTTP do PSP descartava conexões (531 conexões novas para 640 chamadas,
     agora 32). A 640× local: 18.776 inscrições em 3 min, p95 crítico 13 ms, auditoria limpa; as
     ~200 falhas restantes eram conexões que não abriam [**CORRIGIDO à tarde:** a causa era o PSP simulado
     antigo, que gastava as portas; com o PSP novo, o 640× passa limpo; ver abaixo] — subir mais localmente mede a
     máquina, não o sistema. **68 testes Go.** O PSP simulado em execução ainda é o binário antigo
     (reiniciá-lo apaga os pagamentos da memória e gera divergências falsas na conciliação); o novo
     está em `poc\bin\psp-simulado-novo.exe`. O Postgres local agora carrega `pg_stat_statements` (`postgresql.conf` do
     `pgdata-lisura`). Para medir carga localmente, o front tem de rodar em modo de produção
     (`poc-web\.next\standalone\server.js` na porta 8090, depois de `next build` e da cópia de
     `.next\static`): o `next dev` não aguenta carga e falseia o resultado.
   - **05/11, depois (o usuário pediu "prossiga com o desenvolvimento, especialmente back-end e
     testes"): picos 4 e 5 — gabarito, recurso e resultado** (`poc/internal/resultado`, migrações
     00008 e 00009, `cmd/banca`). A banca entrega as respostas lidas (a leitura física é dela); a
     plataforma corrige, aplica nota mínima, cotas (Lei 15.142/2025: 30%, fração ≥ 0,5 sobe, só com
     2+ vagas; Decreto 9.508/2018: 5%, fração sobe — **conferidos no Planalto em 05/11**, viram
     parâmetro de cada cota porque município segue lei própria) e desempate (Estatuto da Pessoa Idosa
     art. 27: idade primeiro, entre quem tem 60+), classifica e grava cada cálculo como **versão
     imutável**. Publica pela fila `gabarito.json`, `resultado.json` (listas sem dado pessoal) e o
     resultado individual em 256 fatias cuja chave deriva do código do cartão — no pico 5 a tela não
     precisa de API nem de banco. Recurso contra o gabarito: um por questão, fundamento de lista,
     agrupado por questão para a banca, uma decisão responde a todos, o definitivo só sai com tudo
     decidido. Rotas novas: `GET /inscricoes/{id}/resultado`, `GET`/`POST /inscricoes/{id}/recursos`.
     Também: `TestCadaMigracaoDesfazERefaz` (cada migração desfeita e refeita, schema comparado);
     **trilha encadeada por SHA-256** (migração 00010; selo impresso no cartão; `auditoria -trilhas`
     verifica tudo — 205.471 trilhas, 0 quebradas, em 0,96 s, depois de encadear os 560.925 eventos
     existentes em ~80 s); **`go test -race` rodando** (llvm-mingw em `C:\Users\rockb\tools\llvm-mingw`,
     `CGO_ENABLED=1`, `CC=clang`; conferido com corrida plantada). **94 testes Go, todos passando,
     também com `-race`.** Medido localmente com 200 mil inscrições: cálculo 9,9 s, recálculo após
     anular uma questão 10,2 s, 256 fatias de 143 KB (19 KB com gzip); **pico 5** (`carga/pico-resultado.js`,
     1.000/s por 2 min): 139.999 aberturas, 0 falhas, banco 12,7 transações/s parado e 9,3 durante —
     o pico não chega ao banco. Certame 6 do banco local é essa massa (ids 211822–411821).
     **Front do resultado e do recurso NÃO foi feito** (só back-end e testes).
   - **05/11, à tarde (o usuário escolheu "mais back-end"):** três coisas.
     - **Âncora da trilha fora do banco** (`poc/internal/ancora`): de hora em hora, o último evento
       e o resumo de cada inscrição vão para um cofre que grava uma vez e não sobrescreve. O cofre
       é o bucket S3 com bloqueio de objeto em `infra/terraform/cofre.tf` (GOVERNANCE 1 dia no
       ensaio, COMPLIANCE em produção); na máquina local, a pasta `C:\Users\rockb\tools\lisura-cofre`.
       Um recibo com o resumo sai em `/publico/certames/{id}/ancoras/`. `auditoria -ancoras` confere
       o banco contra o cofre, e `banca ancorar` ancora na hora.
     - **Defeito achado ao testar a âncora:** ler o cofre depois de fotografar o banco gerava âncora
       mais velha que a anterior. Com a ordem antiga, o teste dá 8 alarmes falsos em 22 âncoras.
     - **Fila com prioridade** (migração 00011): urgente (Pix), normal, lote (republicação em massa
       da homologação). Provado contra o defeito: sem a prioridade, o trabalhador pega
       `[lote lote lote]` antes do Pix.
     - **Medido com 100 mil inscrições** (certame 8 local):

       | Etapa | Resultado |
       |---|---|
       | Âncora | 0,65 s e 7,9 MB com 300 mil eventos; 1,8 s com 600 mil |
       | Conferência | 200 mil pontos em 2,2 s, 0 problemas |
       | Homologação em lote | 82,7 s, ~1.200 inscrições/s |
       | Fila sem prioridade | 100 mil cartões em ~4 min, pico de 75.877 tarefas vencidas; a relação ficou 1 min 40 s atrás deles |

     - **Conciliação descartada como gargalo:** 158.327 pagamentos no extrato conferidos em 0,55 s
       por minuto.
     - **Fila priorizada, medida de novo** (certame 9, outras 100 mil): a relação esperou 1 a 59 ms,
       antes 1 min 40 s; os cartões de lote saíram no mesmo ritmo.
     - **Âncora incremental:** uma completa por dia; entre elas, só o que mudou. Na API, a âncora
       nº 2 do certame 5 levou 800 de 206.265 inscrições. Recibo que faltou sai na rodada seguinte.
     - **Gerador do relatório do H1** (`poc/cmd/relatorio` + `internal/relatorio`). Lê um manifesto
       com o log do k6 e o JSON da auditoria de cada patamar e escreve o relatório: veredito por
       critério, H1 só com 40× por 3 h, ponto de quebra, custo/h contra US$ 4,44 (04-escopo §5).
       **Validado com rodada local real**: 1.744 inscrições, k6 = banco, 0 duplicadas com 97 cliques
       duplos; o relatório recusou o H1 por durar 2 min. Exemplo em `carga/exemplo-relatorio-local.md`.
     - **Armadilha do ensaio:** os 2 últimos dígitos da `RODADA` escolhem 1 de 99 faixas de CPF.
       Repetir uma faixa faz a rodada inteira falhar ("CPF já existe"). O banco local já usou 22.
     - **Os testes Go passam todos, também com `-race`** (a contagem está em `poc/README.md`).
   - **05/11, à noite ("mais back-end" de novo):** acima de 640×, localmente. Detalhes e números
     em `poc/README.md`, seção "6. Acima de 640×".
     - **Orquestrador do ensaio** (`infra/scripts/ensaio.ps1`): roda patamar a patamar, com o k6, a
       auditoria daquela rodada, o manifesto e o relatório. Para no primeiro patamar que falha,
       que é o ponto de quebra, e escolhe sozinho uma faixa de CPF livre.
     - **PSP simulado com diário** (`-diario`): sobrevive a reinício sem perder cobranças nem o
       extrato.
     - **640× passa limpo**: 27.415 inscrições, 100%, 0 falhas, 0 reembolsos.
     - **1.280× falha em jornadas concluídas** (74% com 4 trabalhadores, 47% com 16), com a CPU da
       máquina a 99–100%, medida por um amostrador em Go. O k6 come um terço da máquina, e mais
       trabalhadores pioram. **É limite desta máquina, não do sistema**: só a AWS responde. Mesmo
       quebrando, zero inscrições perdidas e zero cobranças duplicadas.
     - **Defeitos achados a 1.280× e corrigidos:**
       - o PSP simulado aceitava pagar duas vezes o mesmo Pix (8 casos), e agora recusa com 409;
       - o k6 recarregava o "pagar no banco" (128 boletos pagos 2 ou 3 vezes), e agora não recarrega;
       - a conciliação mais larga era de 48 h, menor que a margem do boleto (3 dias úteis + fim de
         semana), e o boleto pago com o aviso perdido expiraria. Agora as janelas são escalonadas:
         10 min a cada minuto, 2 h a cada 10, 48 h a cada hora, 7 dias por dia;
       - o k6 tinha travado por uma regressão minha (propriedade nova num objeto do Go), já corrigida.
     - **Auditoria:** duas conferências novas, "cobrada duas vezes" (dois pagamentos confirmados na
       mesma inscrição) e "pago a mais sem reembolso". O relatório mostra os reembolsos.
     - **Suspeita errada, registrada:** a busca binária no extrato do PSP não tirou os picos de CPU
       dele. A causa não foi achada; o PSP é simulado e na AWS roda numa tarefa própria.
     - **Medir CPU nesta máquina:** o PowerShell leva ~20 s por `Get-Process`, e o WMI de
       desempenho trava. Use o amostrador em Go do scratchpad da sessão, ou escreva outro com
       `GetSystemTimes`/`GetProcessTimes`.
     - **Em execução (processos independentes):**
       - a API `poc\bin\lisura.exe`, com 4 trabalhadores, as janelas novas e a âncora a cada 10 min;
       - o PSP novo, com `-diario C:\Users\rockb\tools\lisura-psp-diario.jsonl -repetir 2 -perder 0.02`.

       A última rodada usou a faixa de CPF 27; o orquestrador acha a próxima livre sozinho.
   - **05/11, mais tarde ("prossiga initerruptamente"): o dinheiro a mais agora volta.** Detalhes em
     `poc/README.md`, "A devolução do que entrou a mais" e "Reembolso em massa".
     - **Devolução** (migração 00012): uma por pagamento, nunca duas (chave única no pagamento; id no
       prestador = `dev-` + id do pagamento, num `PUT` idempotente como na API Pix do BC). O Pix volta
       para quem pagou; o boleto, por Pix para a chave CPF do candidato, com o titular conferido pelo
       prestador. Sem chave CPF: `AGUARDANDO_CHAVE`. A rotina retoma, a cada 10 min, a devolução cuja
       tarefa a fila abandonou.
     - **Medido:** a migração criou 906 devoluções do que já tinha entrado a mais; 868 saíram em
       0,48 s. As 38 recusadas são Pix de antes de o PSP ter diário (a conversão do PSP antigo levou os
       pagamentos e não as cobranças), um artefato local que a auditoria acusa.
     - **Reembolso em massa** (migração 00013, doc 02 §C): `banca cancelar -certame N -analista X
       -motivo CODIGO`. Fecha o certame a inscrição e cobrança, cancela toda inscrição viva pela
       máquina de estados (evento `CANCELAR_CERTAME`, inclusive a homologada) e devolve todo
       pagamento confirmado em lote. Retomável sem repetir.
     - **Medido no certame 10**, criado para isto, depois de um 640× real: 28.831 inscrições canceladas
       e 24.885 pagamentos devolvidos em 41,9 s, auditoria com 0 problemas. O certame 5 ficou intacto.
     - **Auditoria:** "pago a mais" passou a conferir a devolução, não a marca, e ganhou quatro
       conferências: devolução de pagamento válido, acima do pago e recusada, e certame cancelado com
       inscrição viva ou pagamento retido.
     - **Não provado por teste:** a recusa de cobrança em certame cancelado. A inscrição cancelada já
       recusa sozinha, e a conferência só cobre o instante entre marcar o certame e cancelar cada
       inscrição.
     - **Rota da chave Pix (feita, a pedido do usuário):**
       `PUT /api/inscricoes/{id}/devolucoes/{dev}/chave`.
       - O formato é conferido e normalizado como o DICT guarda (CPF com dígitos verificadores,
         e-mail, `+55` com DDD, chave aleatória). CNPJ é recusado.
       - Só vale enquanto a devolução espera chave; depois de pedida ou concluída, não muda.
       - O prestador confere o titular. Chave de outra pessoa ou inexistente volta a esperar, com
         `recusa`, e nada sai.
       - O PSP simulado ganhou um DICT mínimo (`RegistrarChave`, `SemChaveCPF`).
       - A prova contra o defeito mostrou que a devolução concluída também é barrada pela restrição da
         tabela, e que o caso "pedida e ainda não concluída" não tinha teste. Agora tem.
     - **Front das devoluções e da chave Pix: feito em 06/11** (ver abaixo).
   - **05 para 06/11 ("mais back-end"): prestação de contas** (`poc/internal/arrecadacao`, migrações
     00014 e 00015). Detalhes em `poc/README.md`, "Prestação de contas".
     - **Sem repasse de dinheiro:** o dinheiro cai na conta da banca, e nós orquestramos (03-plano §3).
       O que a banca e o órgão recebem é a conta.
     - **Os comandos:**
       - `banca arrecadacao -certame N [-json]`: por dia, cargo e cidade, com a prova
         "entrou − saiu − a devolver − em divergência = taxas válidas". Sai com 1 se não fecha.
       - `banca fechar -certame N -analista X`: fecha cada dia terminado, imutável (gatilho da
         trilha), com o SHA-256 dos ids e valores do dia. A API também fecha a cada hora.
       - `banca conferir -certame N`: a igualdade, os dias fechados refeitos e o extrato do
         prestador, pagamentos e devoluções, nos dois sentidos. Sai com 1 se acha algo.
     - **O dia é o do registro,** não o do pagamento, como na data de crédito do extrato. O boleto
       informado tarde não muda dia fechado.
     - **Medido:** certame 5, 343.756 pagamentos, R$ 34.385.700,00, a conta fecha, conferência
       contra o extrato inteiro do prestador em 2,7 s, 0 problemas. Certame 10, cancelado: entrou =
       saiu, 0 taxas válidas. O 04/11 do certame 5 foi fechado (1.495 pagamentos, 367 ms).
     - **Furo achado antes de rodar e corrigido:** a devolução do prestador era reconhecida pela
       nossa linha de devolução e sumia da conferência se a linha fosse apagada. Agora é reconhecida
       pelo pagamento.
   - **06/11 ("keep doing back-end as much as possible"): ensalamento** (`poc/internal/ensalamento`,
     migração 00016; doc 02 §7 F e §8, "ensalamento simples por capacidade" é do MVP). Detalhes em
     `poc/README.md`, "Ensalamento".
     - **Os comandos:**
       - `banca locais -certame N -arquivo locais.csv`. O cabeçalho do CSV é
         `cidade;local;endereco;amamentacao;sala;capacidade;tipo;acessivel`, e o tipo é comum,
         especial ou individual.
       - `banca ensalar -certame N -analista X -motivo CODIGO`.
       - `banca mapa`, `banca listas -pasta P` e `banca conferir-ensalamento`.
     - **As regras:**
       - ledor ou transcritor vão para sala individual; tempo adicional ou prova ampliada, para sala
         especial; quem precisa de fácil acesso, para sala acessível; a lactante, para um local com
         amamentação;
       - os mais restritos escolhem primeiro, depois a ordem é cargo e nome;
       - faltou lugar, ninguém é alocado, e o erro diz o que falta e onde;
       - reensalar mantém quem tinha sala.
     - **O ato publicado** é imutável, com SHA-256. O banco recusa sala lotada por gatilho. O cartão
       mostra local, endereço e sala.
     - **Fila:** ganhou `EnfileirarMuitos`, várias tarefas numa instrução só.
     - **Medido no certame 11** (criado para isto, 150 mil homologadas):
       - ensalar em 14,3 s;
       - conferência com 0 problemas;
       - 150 mil cartões republicados em 5 min 14 s, sem falha.
     - **Limite conhecido:** quem entra por recurso só disputa o lugar que sobrou. Ninguém é
       remanejado.
   - **06/11: cronograma e retificação** (`poc/internal/cronograma`; doc 02 §7 A, "mudou uma data,
     o sistema mostra o que quebra").
     - **8 regras entre fases**, cada uma só quando o certame tem as duas fases:
       - a isenção fecha 5 dias antes do fim do pagamento;
       - o pagamento cobre o fim da inscrição;
       - a homologação abre 3 dias depois do pagamento (a margem do boleto);
       - o ensalamento vem depois da homologação;
       - a prova é 7 dias depois do ensalamento;
       - o recurso vem depois da prova e dura 2 dias;
       - o resultado vem depois do recurso.
     - **`banca cronograma -certame N`** mostra as fases e o que quebra.
     - **`banca retificar -certame N -fase F -abre|-fecha AAAA-MM-DDTHH:MM -analista X -motivo Y
       [-simular]`:**
       - recusa fase encerrada, início de fase aberta e fim no passado;
       - recusa só o que a mudança quebra; o que já estava quebrado sai em `ja_quebradas`;
       - grava uma versão nova do edital com o antes, o depois e o motivo, e republica o certame;
       - com `-simular`, mostra o que quebraria e quem seria alcançado, sem gravar.
     - **7 testes, quatro conferidos contra o defeito.**
   - **06/11: exportação para o órgão** (`poc/internal/exportacao`). Detalhes em `poc/README.md`,
     "Exportação para o órgão e a auditoria".
     - `banca exportar -certame N -pasta P` grava CSVs por assunto (pelo `COPY`, numa transação
       REPEATABLE READ), as âncoras do cofre e um manifesto com SHA-256 e selo. Sem dado pessoal.
     - `banca verificar-exportacao -pasta P` confere sem banco: arquivos, corrente da trilha, selos,
       âncoras e dias fechados refeitos.
     - **Defeito achado e corrigido:** o resumo do dia fechado dependia da colação do banco
       (`ORDER BY` de texto). No Aurora, en_US.UTF-8, sairia outro resumo. Agora usa `COLLATE "C"`.
   - **06/11: avisos ao candidato** (`poc/internal/notificacao`, migração 00017). Detalhes em
     `poc/README.md`, "Avisos ao candidato".
     - Um e-mail por evento que importa, nascido na mesma transação do evento, com prioridade de
       lote; chave do envio = id do evento; prova imutável com SHA-256 do texto.
     - Provedor: caixa de saída local (`LISURA_EMAIL_CAIXA`), `LISURA_EMAIL=descarte` no ensaio AWS.
       SES ainda não existe.
     - **A/B a 640×** (mesmo código, com e sem o aviso): custo dentro do ruído. Uma rodada de cada:
       o processo morreu na terceira.
   - **06/11: relação em fatias** (`poc/internal/publicacao`, migrações 00018 e 00019).
     - A relação inteira (3,0 s, 32,7 MB por republicação no certame 5) virou fatias de 10 mil
       números com um índice. A mudança de uma inscrição refaz só a fatia dela: 50 ms em média.
     - O índice deduzido das inscrições custava 271 ms: o planejador ignorou o índice
       `(certame_id, id)`. Agora as fatias ficam anotadas em `relacao_fatia`, na transação da
       inscrição, e o índice sai em 0,054 ms.
     - **Defeito do Windows achado pela suíte:** a leitura do arquivo estático falhava com acesso
       negado no instante da troca (11 de 90 rodadas sob carga). A abertura repete por ~10 ms; 90 de
       90 limpas.
     - **Defender:** bloqueia executável de teste do Go em `%TEMP%`. Compile em `poc\bin`.
     - **Suíte:** 166 testes (178 com subtestes), verdes também com `-race`.
     - **640× depois das fatias, passou em tudo:** 25.554 inscrições, p95 150 ms, 100% concluídas,
       CPU 80%. De manhã: 19.591 e 269 ms. É uma rodada só, sem A/B.
     - **Ambiente:** o front de carga é o Next de produção em
       `poc-web\.next\standalone\server.js`, com `PORT=8090`, `HOSTNAME=127.0.0.1` (com
       `localhost` escuta só em IPv6 e o k6 leva recusa) e `LISURA_API=http://localhost:8082`. Um
       reinício da máquina o derruba.
   - **06/11, à tarde ("prossiga"):**
     - **SES** (`poc/internal/notificacao/ses.go`). Detalhes em `poc/README.md`, "O SES".
       - Lê a cota da conta na subida e espaça os envios nela.
       - Separa a recusa definitiva (`MessageRejected`) da passageira.
       - O SES não tem chave de idempotência: a garantia vira "pelo menos um aviso", provado por teste.
       - No Terraform, `email_de` liga o SES. Vazia, o ensaio fica em `descarte`.
       - **Falta:** tratar os *bounces* (devoluções tardias, via conjunto de configuração), a
         identidade com DKIM na conta, e testar contra a AWS de verdade.
       - **Antes de produção:** a 40× são ~39 avisos/s. Conferir a cota da conta e pedir aumento.
     - **Cobrança no meio do cancelamento**: o furo de teste registrado em 05/11 foi fechado
       (`TestCobrarNoMeioDoCancelamento`, conferido contra o defeito).
     - **Dependências: corrigido um erro meu.** O SDK da AWS tinha entrado em 05/11 com versões de
       04/11, contra a regra das 2 semanas. Foi rebaixado para a leva de 22/10. Os 131 módulos têm 2
       semanas ou mais, e `go mod verify` passa.
     - **Suíte:** 171 testes (183 com subtestes), verdes também com `-race`.
   - **06/11, fim da tarde ("prossiga initerruptamente com tudo que não precisa ser perguntado"):**
     - **Devolução e reclamação dos avisos** (migração 00020, `notificacao/sns.go` e
       `ocorrencias.go`, `infra/terraform/ses.tf`). Detalhes em `poc/README.md`, "Depois de sair".
       - O SES publica no SNS, e o SNS entrega em `POST /api/ses/eventos` (com `LISURA_SES_TOPICO`).
       - A assinatura é conferida. A ocorrência fica imutável. Devolução definitiva ou reclamação
         suprimem o endereço; a temporária, não.
       - A exportação leva `avisos_ocorrencias.csv`, sem endereço nem diagnóstico.
     - **Front** (`poc-web/src/app/(area)/inscricoes/[id]/`):
       - `Devolucoes.tsx`: o andamento e a chave Pix, com a recusa só quando o candidato informou
         uma chave.
       - `Avisos.tsx`: cada e-mail com a prova (SHA-256). O que voltou pede um e-mail novo ali mesmo.
     - **Percorrido no navegador** com uma conta de teste (CPF 99499123938, certame 3, inscrição
       1123248): chave inválida, chave inexistente, chave certa até "Devolvido"; aviso devolvido
       (ocorrência gravada direto no banco, porque o SNS não alcança esta máquina) até o e-mail
       novo, e o alerta some.
     - **PSP simulado:** `PUT /sandbox/dict/{chave}` e `DELETE /sandbox/dict/cpf/{cpf}`. Ficam só na
       memória: um reinício do PSP os perde.
     - **Suíte:** 177 testes (189 com subtestes), verdes também com `-race`.
     - **Ambiente:** o front de carga sobe com `HOSTNAME=127.0.0.1` (já no `subir-servicos.ps1` do
       scratchpad). Para recompilar: parar só o pid do front, `npm run build`, e `robocopy` de
       `.next\static` e `public` para `.next\standalone`.
   - **06/11, noite: documentação viva e observação em tempo real** (pedido do usuário: páginas da
     API, do front com todos os componentes, da infraestrutura, uma que dispare testes de estresse,
     e ver o sistema mudando, "como Kibana e Grafana"). Detalhes em `poc/README.md`,
     "Documentação viva e observabilidade".
     - **Subir:** `observabilidade\subir-observabilidade.ps1 -SegredoOperacao operacao-local
       -TokenKey chave-local-da-poc` (Prometheus 9090, Grafana 3000, executor 8083). O front:
       `subir-front.ps1` do scratchpad, ou `npm run design` + `npm run build` + copiar
       `.next\static` e `public` para `.next\standalone`.
     - **Páginas no front (8090):** `/docs`, `/docs/api` (Scalar, do `openapi.yaml` embutido na API),
       `/design` (Storybook estático em `public/design`), `/docs/infra` (gerado do Terraform por
       `cmd/infradoc`), `/operacao` (dispara ensaios, troca o nível, embute o Grafana).
     - **Métricas:** `GET /metrics` na API (`internal/metricas`); o Grafana é só leitura, sem login, e
       o painel é arquivo do repositório (`observabilidade/grafana/paineis/lisura.json`).
     - **Na AWS:** painel do CloudWatch em `infra/terraform/observabilidade.tf` (não aplicado).
     - **Achados:** o antd estava em 6.6.8 (26/10, contra a regra das 2 semanas) → fixado em 6.6.7, e
       o front reinstalado com `npm install --before=2026-10-23T20:00Z`. O Next 16.4.3 (23/10) passa
       por horas.
   - **06/11, fechamento (o usuário parou o desenvolvimento e pediu: compilar, documentar, subir):**
     - **Compilação final:** 187 testes Go (199 com subtestes), verdes também com `-race`; o front
       passa em tipos, lint, Storybook e build de produção.
     - **`/operacao` não está conferida:** quebrou no navegador (o executor mandava `null` no
       histórico vazio); a correção está no código, com teste, mas a página não foi aberta de novo e
       nenhum ensaio foi disparado por ela. É a primeira coisa a conferir na próxima sessão.
     - **O repositório agora é a fonte:** `C:\projects\fiap-startup-one` (GitHub
       `jgoa156/fiap-startup-one`). Até 06/11 o trabalho em Go/Next vivia só nesta pasta do Desktop,
       e o repositório tinha parado em 29/10 com a POC antiga em Python. Ele foi sincronizado com o
       Desktop (sem `node_modules`, `.next`, `bin`, `dados`, Storybook compilado, estado do Terraform
       e resultados de ensaio) e enviado. **Daqui em diante, trabalhar no repositório.**
     - **Processos parados:** API, PSP, executor, Prometheus, Grafana, front e o Postgres local. Para
       voltar: `poc\scripts\db-up.ps1`, depois a API, o PSP e o front (poc/README.md, "Como rodar"),
       e `observabilidade\subir-observabilidade.ps1`.
     - **ClaudeWork:** nada sobre a Lisura lá (só material da Idera); nada a integrar.
   - **Próximo: semana 7 — precisa do usuário:** configurar a credencial AWS (`aws configure sso`
     ou chave), aceitar o custo, e então `terraform apply` → `publicar-imagens.ps1` → ensaio 10×,
     20×, 40× e acima até quebrar. Passo a passo em `infra/README.md`. A conciliação, que era a
     suspeita, foi medida localmente e não é gargalo (158 mil pagamentos em 0,55 s).
   - **Ambiente (04/11):** o Git Bash desta máquina falha com `add_item ... errno 1` (usar
     PowerShell ou as ferramentas de arquivo). Serviços: Postgres `scripts\db-up.ps1`; API
     `poc\bin\lisura.exe` (8082) com `LISURA_PUBLICO=C:\Users\rockb\tools\lisura-publico`,
     `LISURA_TOKEN_KEY=chave-local-da-poc`, `LISURA_SANDBOX=1`,
     `LISURA_OPERACAO_SEGREDO=operacao-local`; PSP `poc\bin\psp-simulado.exe` (8081); front
     `npm run dev -- -p 8080` em `poc-web`.
   - **Rodar:** 3 processos (API :8082, PSP :8081, Next :8080) — passo a passo em `poc/README.md`.
   - **Front: REACT COM NEXT.JS (decisão do usuário em 30/10/2026).** O portal em HTML no servidor
     + htmx foi **abortado** pelo usuário ("não existe nenhuma estilização e vários itens estão
     incorretos") e o pacote `internal/web` foi **removido em 04/11**.
   - **Referência de front: o projeto Inclusa (`C:\projects\inclusa`), lido em 30/10/2026.** O
     usuário quer "escolhas similares, especialmente no front". O que o Inclusa decidiu (Q-16,
     doc 17): **Ant Design** tematizado por `ConfigProvider` com **raios e sombras do HeroUI**
     (protótipo em `mockups/antd-heroui-sidenav/index.html`); Nunito; altura de controle 44 px;
     botões e campos em pílula; cartões com sombra suave; lateral em cartão flutuante que recolhe
     para ícones; barra superior com busca `Ctrl+K`, ajuda `?` e notificações; ProComponents
     (`StepsForm`, `CheckCard`, `ProTable`) e antd-mobile no celular. Princípios de UX (doc 01):
     escolher em vez de digitar (cartões, chips e segmentados no lugar de select/radio/checkbox),
     uma ação primária por tela, rascunho automático, bloqueio que explica o que falta, ajuda `?`
     em toda tela, WCAG 2.2 AA, alvos ≥ 44 px, cor nunca sozinha. O Inclusa **não** escolhe
     framework (não tem código); o Next é decisão do usuário para a Lisura.
   - **Layout igual ao do Inclusa, cores diferentes — paleta 5 · Grafite escolhida em 30/10/2026**
     (ação `#2F3B4C`, marcador laranja `#FF7A1A`). Tokens em
     `fiap-startup-one/poc/docs/paletas/tokens-lisura.css`; as 8 opções e os contrastes em
     `poc/docs/paletas/README.md`; inventário do que aproveitar do Inclusa em
     `poc/docs/front-a-partir-do-inclusa.md`.
   - O usuário quer cada termo técnico explicado em uma linha — reagiu mal a "pgx, goose, HMAC"
     soltos. Também quer que o trabalho siga sem parar quando diz "prossiga".
   - **Stack: Go 1.27 + PostgreSQL 17** (pgx, goose). Justificativa em `poc/README.md`.
   - **O usuário exige que a stack seja escolhida pelo problema, não pelo que está instalado**, e
     autorizou instalar o que for preciso. Instalados: Go 1.27, PostgreSQL 17, Node 24 LTS e (05/11,
     com checksum/assinatura conferidos) Terraform 1.17.1, k6 2.3.0, AWS CLI 2.39.4 — em
     `C:\Users\rockb\tools\terraform\`, `k6\`, `awscli\Amazon\AWSCLIV2\`. Docker não está instalado
     (as imagens são construídas pelo CodeBuild, na AWS).
   - Ferramentas em `C:\Users\rockb\tools\` (sem admin). O daemon `docker` desta máquina é de
     contêineres **Windows** — não serve para o Postgres nem para imagens Linux.
3. ~~Reverificar 4 itens~~ — **feito em 29/10/2026**, resultado em `fiap-multicloud/06-referencias.md`
   §6.8. Nenhum gatilho disparou; dois casos novos de queda (Enem 2026 e FGV), ambos com causa não
   confirmada.

---

## 3. A empresa: Lisura

**Plataforma que conduz um processo seletivo público de ponta a ponta** — do edital à homologação
do resultado. Vendida para **bancas organizadoras** (as empresas contratadas por prefeituras,
autarquias e universidades para realizar concursos e vestibulares).

*O nome vem de "lisura do certame", termo da administração pública para a integridade do processo.
**É escolha minha, não do usuário — pode ser trocado.** Se trocar, são poucos lugares e os PDFs
se regeram num comando (ver §8).*

### Por que exige nuvem

Um certame tem **cinco picos**, não um:

| # | Pico | Quem acessa |
|---|---|---|
| 1 | Encerramento da **isenção** | quem pede isenção |
| 2 | Encerramento da **inscrição** | quem ainda não se inscreveu — escrita pesada |
| 3 | **Cartão de confirmação** | **a base inteira** |
| 4 | **Gabarito** + recurso | **a base inteira** |
| 5 | **Resultado** e classificação | **a base inteira — o maior** |

**A carga é 40× a média, e a data está publicada no edital com meses de antecedência.** Essa
combinação — extrema, concentrada e *agendada* — é o que organiza toda a decisão de arquitetura.

### Por que agora

**Lei nº 14.965/2024** (sancionada em 09/09/2024), a Lei Geral dos Concursos Públicos: autoriza
prova total ou parcialmente a distância, endurece as regras da banca, e **entra em vigor em
1º/01/2028** com aplicação antecipada permitida. **Todo o mercado precisa se replataformar.**

### Modelo de receita

Assinatura anual pequena (R$ 48–84 mil) **+ valor por inscrição processada (R$ 3,00)** — o segundo
domina. **Vender para a banca evita licitação:** o órgão licita a banca; a banca contrata
fornecedor como empresa privada.

---

## 4. Os números canônicos

> **Use estes. Se algum documento divergir, o documento está errado.**

### Ano 5 (a premissa do trabalho de multicloud)

| | |
|---|---|
| Bancas clientes | **72** |
| Inscrições processadas/ano | **7,7 milhões** |
| Receita (ARR) | **R$ 27,4 mi** |
| EBITDA | **+R$ 5,3 mi (19%)** |
| Headcount | 68 · **time de plataforma: 6** |
| Infraestrutura | R$ 3,2 mi/ano |
| Breakeven | **4º trimestre do ano 4** |
| Rodada | **R$ 80 mi no mês 46** |

### Projeções

| R$ mi | Ano 1 | Ano 2 | Ano 3 | Ano 4 | Ano 5 |
|---|---:|---:|---:|---:|---:|
| Bancas | 6 | 13 | 28 | 48 | 72 |
| Inscrições (mi) | 0,12 | 0,69 | 2,05 | 4,30 | 7,70 |
| **ARR** | **0,6** | **2,9** | **7,8** | **15,8** | **27,4** |
| **EBITDA** | −3,8 | −5,0 | −3,9 | −0,6 | **+5,3** |

### Unit economics (ano 3)

ACV **R$ 280 mil** · CAC **R$ 65 mil** · retorno de CAC **3,7 meses** · perda 6%/ano ·
LTV/CAC **~15×** · margem bruta 76%

### Mercado

TAM **R$ 90 mi** · SAM **R$ 36 mi** · SOM ano 5 **R$ 27 mi**
**O TAM é modesto de propósito — e isso é dito no pitch.** Crescimento por adjacência.

### Custo dos dois modelos de continuidade

| | R$/ano | Cobre |
|---|---:|---|
| **M2 · segunda nuvem (warm standby)** — rejeitado | **1,66 mi** | 1 das 7 causas |
| **M1 · nuvem única endurecida** — adotado | **1,69 mi** | as 7 |
| dos quais: plano de controle fora da AWS | 162 mil | 3% do EBITDA |

### Financiamento

Pre-seed R$ 1,5 mi (M0) · **Seed R$ 10 mi (M8)** · Série A R$ 35 mi (M26) · Série B R$ 80 mi (M46)

---

## 5. A tese do trabalho de multicloud

> **NÃO à arquitetura multicloud. SIM ao controle e ao backup fora do provedor primário.**

### O argumento central (é o slide 3, e é a defesa inteira)

Quando a notícia diz *"site do concurso fora do ar no último dia"*, a causa é uma de sete:

| Causa | Multicloud resolve? |
|---|---|
| Pool de conexões esgotado com 30× a carga | não |
| Geração síncrona de boleto/cartão comendo a CPU | não |
| Consulta sem índice que só aparece com 200 mil linhas | não |
| Capacidade nunca ensaiada para o pico | não |
| Implantação ruim na véspera do prazo | não |
| Um certame gigante degradando os outros inquilinos | não |
| **Queda regional total do provedor** | **sim** — e é a mais rara |

**Multicloud cobre uma.** Uma segunda nuvem rodando o mesmo software mal dimensionado esgota o
mesmo pool. **Replica o gargalo, não a capacidade.**

### Os três argumentos de apoio

1. **A falha é reversível.** Há registro de prazo de inscrição **estendido por decisão judicial**
   por indisponibilidade do site. Ninguém perde o direito de concorrer. Isso muda quanto a
   proteção pode custar. *(É a premissa mais importante da defesa — está verificada.)*
2. **Não é custo.** M1 e M2 custam praticamente o mesmo (R$ 1,69 mi vs R$ 1,66 mi). A escolha é
   sobre o que o dinheiro compra. **Nunca apresente isso como economia.**
3. **O failover não exercitado é passivo.** Um caminho que roda 1× a cada 4 anos, operado por 6
   pessoas, é um modo de falha novo — não uma proteção.

### A ressalva (slide 6) — não pode ser cortada

DNS + WAF (60 mil) · página de status (12 mil) · backup off-site em segundo provedor (90 mil) =
**R$ 162 mil/ano**. E **conceder que "pelo critério estrito isto é multicloud"** é o que impede a
banca de dizer que a pergunta não foi respondida.

### Prós e contras

**1 pró válido de 8** (só P8, resiliência a falha de provedor). Contras decisivos: **eficácia** e
**failover não exercitado** — os dois que o dinheiro não resolve.

### Os gatilhos que invalidariam a decisão

- **Gatilho 2 — o mais importante do trabalho:** se a prova online entrar em escala (Lei
  14.965/2024, 2028), **a falha deixa de ser reversível** — não se prorroga prova em andamento — e
  a resposta certa passa a ser a oposta.
- **Gatilho 5:** se a AWS abrir uma segunda região no Brasil.

---

## 6. Decisões fechadas — não relitigar

Estas foram decididas **pelo usuário** ao longo do trabalho. Um assistente novo não deve reabri-las.

| Decisão | Valor |
|---|---|
| Modelo de negócio | **B2B** — vender para a banca, não para o órgão |
| IA no produto | **Nenhuma.** Descartado explicitamente. O agrupamento de recursos é por **fundamento estruturado**, não por modelo |
| Escopo da plataforma | **Vai até o resultado final**, não para na homologação |
| Logística física | **Fica com a banca.** Nós paramos no dado |
| Conta do candidato | **Única por CPF**, viva entre bancas e certames |
| MVP | **Só prova objetiva** — sem discursiva, títulos ou TAF |
| Adjacência | Fica como **nota**, não entra no plano principal |
| Pagamento | **Orquestramos, não processamos** — taxa de inscrição é receita pública |
| Multicloud | **Não**, com a ressalva do plano de controle |
| POC | **1 pessoa, 2 meses** |
| Deck do Startup One | **Sim**, é necessário |

---

## 7. O que está verificado e o que NÃO está

> **Regra do projeto: premissa não verificada é marcada, nunca embutida.** Foi assumir premissa
> inventada que derrubou uma versão anterior inteira do trabalho.

### ✅ Verificado (set/2026, com fonte)

| Fato | Fonte |
|---|---|
| **Lei 14.965/2024** existe, autoriza prova a distância, vigência 1º/01/2028 | Jusbrasil, análises jurídicas |
| **O mercado está ocupado** — Gestor Editais (Grupo ACT), ProSeleta, Fábrica de Provas, ADSR, MH Informática, Instituto Avalia | sites dos fornecedores |
| **Gestor Editais já cobra por inscrição** — valida o modelo de preço | site do fornecedor |
| **Taxa de inscrição é receita pública**; há entendimento contra delegar sua gestão a empresa privada | TCE-MG, MPRJ |
| Existe **modelo "custo zero"** — a banca assume custos e fica com as taxas | MPRJ |
| **Taxa de R$ 69,33 por candidato** documentada | MPRJ |
| **Prazo estendido por decisão judicial** por site fora do ar | análise jurídica especializada |
| **Ano eleitoral veda nomeação, não o certame** (Lei 9.504/97 art. 73, V) | TecConcursos e outros |
| **AWS tem uma região no Brasil** (São Paulo, 3 AZs). Rio é Local Zone | AWS |
| **HashiCorp/Forrester 2024** é a edição mais recente amplamente divulgada | HashiCorp |

### ⚠️ NÃO verificado — tratado como variável

| Número | Faixa usada | Impacto |
|---|---|---|
| **Preço por inscrição praticado** | R$ 1,50 a 5,00 (base R$ 3,00) | **Decide o negócio. Abaixo de R$ 2,15 o modelo não fecha** |
| Inscrições pagas/ano no Brasil | 20 a 40 milhões | Define o TAM |
| Taxa média paga pelo candidato | ~R$ 80 | Contextualiza o preço |
| Bancas de porte médio existentes | 80 a 200 | Define o SAM |
| Concentração da inscrição nas últimas 48 h | 80–90% | **Sustenta o argumento de nuvem inteiro, e não tem fonte** |

---

## 8. Como regerar os artefatos

Não há Python nem Node nesta máquina. Tudo roda em **PowerShell + Edge headless**.

| O quê | Comando | Fonte |
|---|---|---|
| **Diagrama** | `powershell -ExecutionPolicy Bypass -File fiap-multicloud\diagramas\render-arquitetura.ps1` | o próprio `.ps1` (System.Drawing) |
| **Slides + roteiro (PDF)** | `powershell -ExecutionPolicy Bypass -File fiap-multicloud\apresentacao\build-pdf.ps1` | `fiap-multicloud\apresentacao\deck-lisura.html` |
| **Pitch (PDF)** | `powershell -ExecutionPolicy Bypass -File fiap-startup-one\apresentacao\build-pitch.ps1` | `fiap-startup-one\apresentacao\pitch-lisura.html` |

> **Tudo é autossuficiente.** As fontes HTML dos slides e do pitch estão versionadas ao lado dos
> PDFs, e os dois scripts resolvem os caminhos a partir da própria pasta — funcionam de qualquer
> lugar, em qualquer sessão. Editar slide = editar o HTML e rodar o script. *(Verificado em
> 18/09/2026: os três PDFs foram regerados a partir destas fontes.)*

**Armadilhas conhecidas de PowerShell nesta máquina:**
- Scripts `.ps1` com acento precisam ser gravados em **UTF-8 com BOM**, senão são lidos como ANSI
  e os acentos quebram o parser.
- `Start-Process -Wait` é necessário para o Edge headless — chamar `& msedge` direto retorna antes
  de terminar e o PDF não sai.
- Evitar `.Replace('\','/')` inline em comandos — use `-replace '\\','/'` numa variável antes.

---

## 9. Histórico — o que foi descartado e por quê

*Importante para não repetir caminho já percorrido.*

| Ideia | Por que caiu |
|---|---|
| **Sentinela** — antifraude consorciado para Pix | **Outra equipe apresentou a mesma solução.** Todo o material foi apagado |
| 30 ideias em música (ensaio remoto, tablatura, stems…) | Descartadas quando o usuário excluiu IA aplicada |
| Detecção de música gerada por IA | Era a recomendação da rodada de IA/GPU — caiu junto |
| Transcrição para tablatura | Analisada a fundo; POC bom, negócio mediano |

**Aprendizados que custaram caro:**

1. **Verificar concorrentes ANTES de desenvolver a ideia.** A colisão da Sentinela aconteceu por
   não ter feito isso.
2. **Não inventar restrição para fazer o argumento fechar.** A defesa da Sentinela apoiava-se numa
   cláusula contratual que o assistente havia escrito — e desmoronou quando questionada.
3. **Conceder o ponto forte do adversário antes que ele o use.** Foi o que salvou o slide 6.

---

## 10. Como o usuário quer trabalhar

*Observado ao longo do projeto. Vale respeitar.*

- **Quer honestidade brutal.** Pediu explicitamente avaliação crítica e reagiu mal a otimismo
  infundado. Quando perguntou *"essa proposta é viável?"*, a resposta útil foi a que listou os
  quatro riscos reais, não a que vendeu a ideia.
- **Detecta inconsistência numérica e cobra.** Já apontou que os textos tinham números
  divergentes. **Confira a aritmética antes de entregar** — somas de tabela, percentuais, contagens
  de itens ("8 prós" tem que ter 8 linhas).
- **Não gosta de jargão não explicado.** Reagiu com força a "célula" usada sem definição.
- **Quer ver antes de finalizar.** Pediu previews. Gerar PNG de cada slide é o caminho.
- **Corta escopo com decisão.** Respondeu 14 perguntas de uma vez, em uma linha cada. **Perguntas
  curtas e numeradas funcionam; perguntas longas, não.**
- **Idioma: português.**

---

## 11. Índice dos arquivos

```
fiap-startup-one/
  README.md                          resumo e índice
  01-ideacao.md                      as 9 ideias avaliadas e por que a Lisura ganhou
  02-produto-features-e-fluxos.md    9 módulos, fluxogramas, máquina de estados
  03-plano-de-negocios.md            mercado, preço, projeções, riscos, verificações
  04-escopo-poc.md                   1 pessoa, 8 semanas, ~US$ 100
  apresentacao/
    Lisura-pitch.pdf                 10 slides
    previews/pitch-01..10.png

fiap-multicloud/
  README.md                          tese, mapa das exigências, status
  01-empresa-e-contexto.md           empresa, 5 picos, Lei 14.965, números do ano 5
  02-arquitetura-fase-startup.md     arquitetura AWS inicial e as 8 dívidas
  03-defesa-multicloud.md            ★ O NÚCLEO — 7 causas, prós/contras, 3 modelos, conta
  04-arquitetura-alvo.md             células, agenda, publicação estática, modo degradado
  05-roteiro-apresentacao.md         régua de tempo, ordem de corte, 17 perguntas
  06-referencias.md                  institutos, leis, concorrência, soluções de mercado
  apresentacao/
    Lisura-multicloud-slides.pdf     8 slides
    Lisura-multicloud-roteiro.pdf    fala palavra a palavra
    previews/slide-01..08.png
    deck-lisura.html                 ★ fonte dos slides e do roteiro
    build-pdf.ps1
  diagramas/
    arquitetura-alvo.png / .jpg
    render-arquitetura.ps1
```

---

## 12. Se você for retomar agora, comece por aqui

1. Leia `fiap-multicloud/03-defesa-multicloud.md` — é o núcleo, e o resto deriva dele.
2. Abra `fiap-multicloud/apresentacao/previews/` — vê os 8 slides em 30 segundos.
3. A apresentação já foi feita. **A pendência agora é a POC** — comece por
   `fiap-startup-one/04-escopo-poc.md`, semana 1.
