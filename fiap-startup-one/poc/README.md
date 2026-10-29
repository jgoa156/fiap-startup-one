# POC da Certamis

Implementação da POC descrita em [`../04-escopo-poc.md`](../04-escopo-poc.md) e
[`../HANDOFF-POC.md`](../HANDOFF-POC.md). **O entregável é o relatório de ensaio de carga**,
não a tela.

## Onde estamos (29/10/2026)

| Bloco | Semanas originais | Entrega | Código | Ensaio |
|---|---|---|---|---|
| 1 | 1 | Máquina de estados, trilha de auditoria | ✅ | |
| 2 | 2 e 3 | Inscrição, isenção por regra, pagamento idempotente | ✅ | |
| 3 | 4 e 5 | Worker, cartão em PDF, publicação estática, fila de espera, modo degradado | ✅ | |
| 4 | 6, 7 e 8 | Gerador k6, verificador, roteiro, modelo do relatório, estimativa de custo | ✅ | ⏳ **não rodou**: roda na sua máquina, ver [`carga/ROTEIRO-ENSAIO.md`](carga/ROTEIRO-ENSAIO.md) |

**O que falta para a POC estar entregue**, em ordem:

1. Rodar a suíte inteira no Windows (inclui `test_api.py`, que ainda não executou em lugar nenhum).
2. **Confirmar o fator de rajada** (padrão 3×). Vai para o relatório e para o pitch.
3. Rodar os ensaios do roteiro, subir até quebrar, corrigir o gargalo, repetir.
4. Preencher [`relatorio/MODELO-RELATORIO.md`](relatorio/MODELO-RELATORIO.md) a partir dos arquivos brutos.
5. Levar o resultado para o pitch.

## Decisões desta implementação

| Decisão | Valor |
|---|---|
| Prazo | Menos de 4 semanas: as 8 do escopo comprimidas em 4 blocos. Regra de corte: cai o painel da banca, nunca o ensaio |
| Linguagem | Python 3.13, FastAPI, psycopg 3 com pool explícito |
| Ambiente | **Tudo local** até haver conta AWS. Fila (Postgres em vez de SQS), arquivos (disco em vez de S3) e provedor de pagamento (sandbox simulado) ficam atrás de interface |
| Custo por hora de pico | **Estimado** pela tabela de preços de São Paulo ([`carga/custo.py`](carga/custo.py)), porque não há conta AWS. Declarado no relatório |
| Onde moram as regras | **No banco.** Transição de estado, prazos, formulário completo, triagem de isenção, cobrança única e pagamento idempotente são triggers e funções do Postgres. A API só traduz o erro |
| Terraform | **Fora**, primeiro corte depois do painel. Só faz sentido com conta AWS |

## Rodar no Windows (PowerShell)

Pré-requisitos: Python 3.12 ou mais novo, Docker Desktop e, para o ensaio, k6.

```powershell
cd C:\projects\fiap-startup-one\fiap-startup-one\poc
python -m venv .venv
.\.venv\Scripts\pip install -r requirements.txt
docker compose up -d --wait
.\.venv\Scripts\python -m pytest -q -rs tests      # recria certamis_test a cada teste
```

Aplicação, worker e dados sintéticos:

```powershell
.\.venv\Scripts\python -m app.migrar
.\.venv\Scripts\python -m app.semear 1000
.\.venv\Scripts\uvicorn app.main:app --port 8000     # documentação em http://localhost:8000/docs
.\.venv\Scripts\python -m app.worker                 # noutro terminal
```

## O que existe

### Bloco 1: estado e trilha

Máquina de estados de [`02-produto` §5](../02-produto-features-e-fluxos.md) com 11 estados e 14
transições: 11 do diagrama e 3 decididas em 29/10/2026 (cancelar antes de pagar, voltar ao
rascunho antes de pagar, recurso negado como estado final). Toda transição, inclusive a criação,
grava quem, quando, motivo e versão do edital vigente, e quem grava é o trigger. A trilha não
aceita UPDATE, DELETE nem TRUNCATE.

### Bloco 2: o caminho do candidato

- **Conta única por CPF**, com dígito verificado e senha em scrypt. Login devolve token assinado,
  sem estado no servidor, para não custar escrita no pico.
- **Formulário salvo a cada passo.** Fora do RASCUNHO nenhum dado muda (CT004). Finalizar exige
  cargo, cidade, modalidade e declarações (CT006), e prazo aberto (CT007).
- **Laudo e comprovante** guardados com SHA-256. O laudo pertence a uma inscrição só (LGPD).
- **Isenção com triagem por regra, sem IA.** Na entrada do pedido o banco confere NIS (dígito
  verificador), comprovante obrigatório e dono do comprovante. O que falta é indeferido na hora,
  com motivo padronizado; o resto vai para análise humana. Isento finaliza como isento e não
  paga; quem não tem isenção deferida não finaliza como isento (CT008).
- **Pagamento: orquestramos, não processamos.** Duas garantias no banco:
  1. no máximo **uma cobrança viva** por inscrição (índice único parcial). Repetir com a mesma
     `Idempotency-Key` devolve a mesma; com outra, enquanto há uma pendente, também;
  2. cada **evento do provedor aplicado no máximo uma vez** (chave única no id do evento).

  Pagamento que chega para cobrança cancelada **não some**: fica gravado e aparece na
  conciliação, para estorno pela banca. Cancelar e confirmar travam na mesma ordem (inscrição,
  depois cobrança), o que impede deadlock entre os dois.
- **Expiração**: prazo de pagamento vencido expira inscrição e cobrança, com trilha.

### Bloco 3: fora do request e degradação

- **Fila de tarefas no Postgres** com `SKIP LOCKED`, entrega pelo menos uma vez, recuo
  exponencial e recuperação de tarefa travada por worker que morreu.
- **Worker**: registra a cobrança no provedor (o candidato espera o QR na tela, não no request),
  gera o **cartão de confirmação em PDF** na homologação e **publica a homologação como arquivo**
  estático, em fatias de mil inscrições. A consulta de situação no pico de leitura vai ao
  arquivo, não ao banco.
- **Fila de espera** (nível 1): senha numerada e assinada; o portão anda a uma taxa fixa e não
  acumula crédito com a fila vazia. Consultar a posição **não toca no banco**.
- **Modo degradado** níveis 4 (só estático) e 5 (página de status), mudado pelo plantão e lido
  por cada instância em até 1 segundo.

### Bloco 4: o ensaio

- [`carga/ultimo-dia.js`](carga/ultimo-dia.js): k6 com o perfil do último dia. A unidade é a
  **inscrição completa** (cadastro ou login, formulário em três passos, laudo em 10%, finalizar,
  cobrança, espera do QR, aviso de pagamento, consulta), com abandono, recarga, clique duplo,
  aba duplicada e reenvio do aviso de pagamento **de propósito**. Rajadas de 90 s a cada 10 min,
  com a média da hora mantida no patamar: **conferido por script** que o perfil de 16,7 mil
  planeja 16.656 por hora (por segundo dava 18.368, por isso as taxas são por minuto). Modo
  `escada` para subir até quebrar.
- [`app/verificar_ensaio.py`](app/verificar_ensaio.py): confere no banco, depois do ensaio,
  inscrições perdidas, cobranças duplicadas e pagamentos do k6 contra os do banco.
- [`carga/custo.py`](carga/custo.py): custo por hora de pico estimado, com preços datados e
  fonte de cada um.

> **Achado ao montar a estimativa de custo.** Uma configuração de exemplo (4 tarefas Fargate de
> 1 vCPU e um RDS pequeno) custa uns **R$ 1,83 por hora**: R$ 0,0001 por inscrição a 16,7 mil
> por hora, 0,004% do preço de R$ 3,00. O custo do pico **não ameaça** o preço, mas também
> **não o valida**: o piso de R$ 2,29 depende de pessoas e de vendas. O pitch não pode sugerir
> que a POC provou o preço.

## O que foi verificado, e onde

O ambiente em que este código foi escrito **não tinha acesso ao PyPI** (bloqueio de rede), então
FastAPI e psycopg não puderam ser instalados lá. Tudo que não depende deles rodou de verdade.

| Teste | O que prova | Onde rodou |
|---|---|---|
| `test_banco.py` (bloco 1, 18 verificações) | transições, trilha, imutabilidade | ✅ Postgres 16 |
| `test_banco.py` (bloco 2, 22 verificações) | prazos, formulário, triagem de isenção, cobrança única, evento idempotente, pagamento tardio, expiração | ✅ Postgres 16 |
| `test_banco.py` (bloco 3, 9 verificações) | fila de tarefas, portão da fila de espera | ✅ Postgres 16 |
| `test_concorrencia.py` | **conexões paralelas reais**: 30 cobranças simultâneas viram uma; 30 avisos iguais aplicam um; 30 avisos diferentes aplicam um; cancelamento disputando com pagamento, sem deadlock e sem pagamento perdido | ✅ Postgres 16. Os dois desfechos da disputa aparecem |
| `test_estados.py`, `test_unidades.py` | máquina de estados, CPF (comparado com o k6 em 2.000 índices), senha, token, fila, PDF, armazenamento, provedor | ✅ Python 3.13 |
| Defeitos plantados | 4 mutações nas regras do banco; o teste acusou as 4 | ✅ |
| Verificação estática (pyright) | nomes, atributos, chamadas | ✅ 0 erros |
| `test_api.py` | a API de ponta a ponta: pagante do cadastro ao cartão, isento, dono da inscrição, webhook assinado, modo degradado, fila de espera, persistência depois de leitura | ⚠️ **não rodou ainda**. Roda no Windows |

## Limites conhecidos

- Os triggers protegem contra a aplicação e contra scripts, não contra o dono das tabelas. Em
  produção, a aplicação conecta com um papel que não é dono do esquema.
- A banca não tem login: o painel da banca está fora do escopo. As rotas `/banca` exigem só o
  cabeçalho `X-Ator-Banca`, que vai para a trilha. **Não expor isto fora da máquina local.**
- A fila de espera fica na aplicação; na arquitetura-alvo ela fica na borda (Cloudflare).
- O provedor de pagamento é simulado. A latência de um provedor real não está no ensaio.
