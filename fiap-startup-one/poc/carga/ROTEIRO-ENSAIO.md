# Roteiro do ensaio de carga (local, Windows)

O entregável da POC é o relatório deste ensaio. Este roteiro diz como produzi-lo de forma
que outra pessoa consiga repetir e chegar perto do mesmo número.

## 0. Antes de tudo: duas decisões

| Decisão | Padrão no gerador | Status |
|---|---|---|
| **Fator de rajada** (quantas vezes a média, por 90 s a cada 10 min) | 3× | ⚠️ **a confirmar.** Vai para o relatório e para o pitch. Rodar com `-e FATOR_CONFIRMADO=1` só depois de confirmado |
| Probabilidades de comportamento (abandono 8%, recarga 15%, clique duplo 10%...) | ver topo de `ultimo-dia.js` | estimativa nossa, declarada no relatório |

## 1. Pré-requisitos

```powershell
python --version      # 3.12 ou mais novo
docker --version      # Docker Desktop
k6 version            # se faltar: winget install k6 --source winget
```

**A máquina importa.** Anote no relatório: modelo do processador, núcleos, memória. Feche o que
puder: navegador, IDE, sincronização de nuvem.

> **Limite honesto do ensaio local:** o k6, a aplicação, o worker e o Postgres disputam a mesma
> CPU. O ponto de quebra medido aqui é o desta máquina, não o da AWS. Se houver uma segunda
> máquina na mesma rede, rode o k6 nela com `-e BASE_URL=http://IP-DA-MAQUINA:8000`.

## 2. Ambiente limpo

```powershell
cd C:\projects\fiap-startup-one\fiap-startup-one\poc
docker compose down -v                 # apaga o banco anterior
docker compose up -d --wait
python -m venv .venv                   # só na primeira vez
.\.venv\Scripts\pip install -r requirements.txt
.\.venv\Scripts\python -m pytest -q -rs tests     # TUDO tem de passar antes do ensaio
docker compose down -v; docker compose up -d --wait  # os testes usam outro banco, mas limpe mesmo assim
.\.venv\Scripts\python -m app.migrar
.\.venv\Scripts\python -m app.semear 60000        # contas reaproveitáveis (metade das iterações)
mkdir relatorio\bruto -Force
```

## 3. Três terminais

**Terminal 1, a aplicação.** 4 processos, pool de 10 conexões cada: 40 conexões, abaixo das 100
do Postgres. Mudar estes números é decisão de capacidade e vai para o relatório.

```powershell
$env:CERTAMIS_POOL_MAX = "10"
.\.venv\Scripts\uvicorn app.main:app --host 0.0.0.0 --port 8000 --workers 4 --log-level warning
```

**Terminal 2, o worker.** Registra a cobrança no provedor, gera cartão, publica arquivo.

```powershell
.\.venv\Scripts\python -m app.worker
```

**Terminal 3, o gerador.** Ver passo 4.

Durante cada ensaio, num quarto terminal, grave o uso de recursos:

```powershell
docker stats --no-stream | Out-File -Append relatorio\bruto\docker-stats.txt   # repetir a cada poucos minutos
Get-Counter '\Processor(_Total)\% Processor Time' -SampleInterval 10 -MaxSamples 1080 |
  Export-Counter -Path relatorio\bruto\cpu.blg
```

## 4. A sequência de ensaios

Anote a hora de início de cada um (`Get-Date -Format o`). O verificador precisa dela.

| # | Ensaio | Comando | Duração |
|---|---|---|---|
| 1 | Fumaça | `k6 run -e PATAMAR=2000 -e DURACAO_MIN=10 carga\ultimo-dia.js` | 10 min |
| 2 | 4 mil/h | `k6 run -e PATAMAR=4000 -e DURACAO_MIN=30 -e CONTAS_SEMEADAS=60000 carga\ultimo-dia.js` | 30 min |
| 3 | 8 mil/h | idem com `PATAMAR=8000` | 30 min |
| 4 | **16,7 mil/h, H1** | `k6 run -e PATAMAR=16700 -e DURACAO_MIN=180 -e CONTAS_SEMEADAS=60000 carga\ultimo-dia.js` | **3 h** |
| 5 | **Subir até quebrar** | `k6 run -e MODO=escada carga\ultimo-dia.js` | ~80 min |

Entre um e outro: passo 2 de novo (ambiente limpo), para um ensaio não herdar o banco do anterior.

**Sempre suba até quebrar** (ensaio 5), mesmo que o 4 passe. Um teste que só passa não informa
nada; o número que não se sabe é o perigoso.

Se der tempo, repita o 4 com a fila de espera ligada, para medir o nível 1 do modo degradado:

```powershell
Invoke-RestMethod -Method Post http://localhost:8000/banca/operacao `
  -Headers @{"X-Ator-Banca"="plantao"} -ContentType "application/json" `
  -Body '{"nivel": 1, "taxa_por_s": 6}'
```

## 5. Depois de cada ensaio

```powershell
.\.venv\Scripts\python -m app.verificar_ensaio relatorio\bruto\k6-patamar-AAAA.json --desde 2026-11-03T14:00:00-03:00
```

Ele confere **no banco**, e não pela palavra do k6: inscrições perdidas (o k6 recebeu 200 ao
finalizar e o banco não tem a transição), cobranças duplicadas, pagamentos do k6 igual aos do
banco, e os invariantes da conciliação. Sai com código 1 se algum falhar.

## 6. O que vai para o relatório

Ver `relatorio/MODELO-RELATORIO.md`. Cada número tem de sair de um arquivo de
`relatorio/bruto/`, nunca de memória ou de tela.
