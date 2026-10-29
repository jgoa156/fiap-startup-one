# POC da Certamis

Implementação da POC descrita em [`../04-escopo-poc.md`](../04-escopo-poc.md) e
[`../HANDOFF-POC.md`](../HANDOFF-POC.md). **O entregável é o relatório de ensaio de carga**,
não a tela.

## Decisões desta implementação (29/10/2026)

| Decisão | Valor |
|---|---|
| Prazo | **Menos de 4 semanas.** Plano comprimido abaixo |
| Linguagem | Python 3.13, FastAPI, psycopg 3 com pool explícito |
| Ambiente | **Tudo local** até haver conta AWS. Fila e armazenamento ficam atrás de interface, para trocar por SQS e S3 sem reescrever |
| Conta AWS | **Não existe ainda.** Sem ela, o custo por hora de pico é **estimado** pela tabela de preços, e o relatório diz isso |

## Plano comprimido

O cronograma original tem 8 semanas. Comprimido em 4 blocos, com a regra de corte do escopo:
**cai o painel da banca, nunca o ensaio.**

| Bloco | Semanas originais | Entrega | Status |
|---|---|---|---|
| 1 | 1 | Modelo do certame, máquina de estados, trilha de auditoria | ✅ |
| 2 | 2 e 3 | Fluxo de inscrição (CPF, cargo, cota, upload, isenção por regra) e pagamento em sandbox com idempotência | |
| 3 | 4 e 5 | Worker assíncrono (boleto, cartão), publicação estática do resultado, fila de espera, degradação níveis 1, 4 e 5 | |
| 4 | 6, 7 e 8 | Gerador k6 com perfil de último dia, ensaios a 4, 8 e 16,7 mil/h, subir até quebrar, relatório | |

Ordem de corte se faltar tempo: painel da banca (já fora), depois Terraform (só entra se houver
conta AWS), depois upload real de laudo (vira metadado). O ensaio não sai.

## Rodar no Windows (PowerShell)

Pré-requisitos: Python 3.12 ou mais novo e Docker Desktop. Para conferir:

```powershell
python --version; docker --version
```

Primeira vez:

```powershell
cd C:\projects\fiap-startup-one\fiap-startup-one\poc
python -m venv .venv
.\.venv\Scripts\pip install -r requirements.txt
docker compose up -d --wait
```

Testes (recriam o banco `certamis_test` do zero a cada teste):

```powershell
.\.venv\Scripts\python -m pytest -q -rs tests
```

Aplicação:

```powershell
.\.venv\Scripts\python -m app.migrar
.\.venv\Scripts\python -m app.semear 1000
.\.venv\Scripts\uvicorn app.main:app --port 8000
```

A documentação interativa fica em <http://localhost:8000/docs>.

## Semana 1: o que existe

**Máquina de estados** de [`02-produto` §5](../02-produto-features-e-fluxos.md), com 10 estados e
11 transições. A regra existe em dois lugares, de propósito:

- em [`app/inscricao/estados.py`](app/inscricao/estados.py), para a API responder rápido e com
  mensagem clara;
- **no banco**, em [`0002_inscricao.sql`](app/migrations/0002_inscricao.sql), que é a cópia que
  vale. Nenhum caminho de código, script de correção ou acesso manual muda o estado de uma
  inscrição sem passar por ela. Um teste garante que as duas listas são iguais.

**Trilha de auditoria**: toda transição, inclusive a criação, grava quem, quando, motivo e
**versão do edital vigente**. Quem grava é o trigger do banco, não a aplicação. A trilha não aceita
UPDATE, DELETE nem TRUNCATE, e inscrição não se apaga: cancelar é uma transição.

**Concorrência**: a transição é um UPDATE condicional ao estado esperado. Duas requisições para a
mesma transição não aplicam duas vezes; a segunda recebe 409.

| Código | Significado |
|---|---|
| CT001 | transição não permitida |
| CT002 | transição sem ator ou sem motivo |
| CT003 | tentativa de alterar ou apagar trilha ou inscrição |
| CT004 | dado da inscrição alterado fora do RASCUNHO |
| CT005 | certame sem edital publicado |

### O que foi verificado, e onde

| Teste | Onde rodou |
|---|---|
| `test_estados.py`: transições iguais ao diagrama, estados finais sem saída, todo estado alcançável | ✅ ambiente de desenvolvimento, Python 3.13 |
| `test_banco.py`: 15 verificações das regras do banco, mais a igualdade das tabelas | ✅ Postgres 16. Conferido também que o teste **acusa** um defeito plantado |
| `test_semear.py`: CPF sintético com dígitos válidos | ✅ |
| `test_api.py`: API, trilha pela API, 409 e **50 transições concorrentes com uma única vencedora** | ⚠️ **ainda não rodou**: o ambiente de desenvolvimento não tinha acesso ao PyPI. Roda no Windows com o comando acima |

### Pontos que o diagrama deixa em aberto

Implementado **exatamente** como o diagrama, sem inventar transição. Três coisas que ele não
prevê e que vão aparecer no bloco 2:

1. **Cancelar depois de finalizar.** Só existe `RASCUNHO -> CANCELADA`. Um candidato com boleto
   emitido que desiste fica em AGUARDANDO_PAGAMENTO até expirar.
2. **Recurso negado.** A inscrição fica em INDEFERIDA, que não é estado final no banco. Falta um
   estado para "indeferida em definitivo", ou aceitar INDEFERIDA como final depois do prazo.
3. **Voltar a editar.** Não há `AGUARDANDO_PAGAMENTO -> RASCUNHO`; quem errou o cargo depois de
   finalizar não corrige.

### Limite conhecido

Os triggers protegem contra a aplicação e contra scripts, não contra o dono das tabelas, que pode
desligá-los. Em produção, a aplicação conecta com um papel que não é dono do esquema. Na POC local
o usuário é o mesmo.
