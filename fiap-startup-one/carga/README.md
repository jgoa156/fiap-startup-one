# Gerador de carga — o último dia de inscrição

[`ultimo-dia.js`](ultimo-dia.js) é o roteiro do **k6** (ferramenta de ensaio de carga, código aberto)
que testa o H1. Cada iteração é **um candidato inteiro**, não uma requisição solta.

## O que "40×" quer dizer aqui

| | Valor | De onde vem |
|---|---|---|
| Inscrições por ano (ano 5) | 7,7 milhões | `03-plano-de-negocios.md` |
| **Carga média** | **879 inscrições por hora** | 7.700.000 ÷ 8.760 h |
| 10× · 20× · 40× | 8.790 · 17.580 · **35.160 por hora** | `MULT` |
| 40× por 3 h | ~105 mil inscrições | o último dia inteiro de um certame grande |

Cada inscrição faz 15 requisições à API (conta, quem sou eu, dados, abrir, 4 passos, revisão,
finalizar, pagar, ver a cobrança, pagar no banco, a confirmação e o cartão), mais as consultas
repetidas enquanto espera o Pix, a confirmação e o cartão, mais as leituras estáticas.

## O perfil (04-escopo-poc §8: "não uma rampa uniforme")

| Comportamento | Como está no roteiro |
|---|---|
| Chegada concentrada | Sobe de 10% a 100% da taxa em **90 s** |
| Rajadas | A cada 30 min, **2 min com o dobro** (`RAJADAS=0` desliga) |
| Tempo de pensar | 2 a 30 s entre passos; 15 a 60 s pagando no app do banco (`PENSAR` multiplica) |
| Recarregamento | Resposta > 2 s ou erro → **metade recarrega** e pede de novo. Se "criar conta" recarregado volta "CPF já existe", a primeira tentativa deu certo sem a resposta chegar: o candidato entra na conta (`POST /api/sessoes`). Até 05/11 isso contava como falha: 338 jornadas a 1.280× local. **"Pagar no banco" não recarrega**: é o app do banco, e recarregar a página do Lisura não paga de novo (até 05/11 recarregava: 128 boletos pagos 2 ou 3 vezes a 1.280× local) |
| Clique duplo em "Pagar" | **5%**, as duas requisições ao mesmo tempo |
| Abandono e retorno | **12%** param no meio; **60%** desses voltam 2 a 8 min depois e continuam o rascunho |
| Não paga | **10%** geram o Pix e não pagam (não conta como falha) |
| Pix ou boleto | 85% · 15% |
| Sala de espera | 503 com fila → espera o `Retry-After` e tenta de novo, com a mesma senha |
| Aviso repetido e perdido | No PSP simulado do ambiente: cada aviso vai 2×, 2% não chegam (a conciliação recupera) |

## O que sai

- **Limites do H1** (o k6 sai com código ≠ 0 se algum estourar): p95 das requisições críticas
  < 2 s; jornadas concluídas > 98%; zero falhas.
- **Contadores para a auditoria:** `inscricoes_enviadas` (o candidato viu "enviada") e
  `pagamentos_vistos` (viu "Paga"). O banco tem de ter pelo menos isso, e a auditoria
  (`lisura-auditoria`) confere, junto com cobranças duplicadas e pagamentos sem lastro.
- Tempos até o Pix aparecer, até o pagamento ser confirmado e até o cartão ser publicado, e o
  tempo na fila.

## O relatório (`lisura-relatorio`)

O relatório do ensaio sai dos arquivos de cada rodada, sem número digitado à mão. Cada patamar
entra no manifesto com:
- o log do k6, com a linha `RESUMO_DO_ENSAIO`, ou o arquivo `SAIDA`;
- o JSON da auditoria daquela rodada;
- o custo do patamar na fatura (Cost Explorer), quando houver.

```powershell
lisura-relatorio -manifesto ensaio.json -saida relatorio.md
```

```json
{
  "titulo": "Ensaio do H1 — AWS sa-east-1",
  "ambiente": "ECS Fargate …, Aurora Serverless v2 …",
  "orcamento_usd_hora": 4.44,
  "patamares": [
    { "k6": "k6-10x.log", "auditoria": "auditoria-10x.json", "custo_usd": 3.10 },
    { "k6": "k6-40x.log", "auditoria": "auditoria-40x.json", "custo_usd": 11.80 }
  ]
}
```

O orçamento padrão de **US$ 4,44 por hora de patamar** vem do 04-escopo-poc §5: os ensaios custam
~US$ 40 para 3 patamares de 3 h, e 40 ÷ 9 h dá 4,44.

O que o relatório julga, por patamar:

| Critério | Alvo |
|---|---|
| Inscrições perdidas (o k6 viu, o banco não tem; idem pagamentos) | zero |
| Cobranças duplicadas ou pagamento aplicado duas vezes | zero |
| p95 das requisições críticas | < 2 s |
| Jornadas concluídas | > 98% |
| Problemas da auditoria | nenhum |
| k6 e auditoria da mesma rodada | iguais |
| Custo por hora, se informado | ≤ orçamento |

O veredito e o que sai em volta dele:
- **O H1 passa** só se um patamar de 40× ou mais, rodado por 3 h ou mais, passar em tudo.
- **Ponto de quebra:** o menor patamar que falhou. Se nenhum falhou, o relatório diz que o limite
  não foi alcançado e que é preciso subir mais.
- **Curva não monótona:** se o relatório falha abaixo de 40× e passa acima, ele avisa.
- **Código de saída:** 1 quando o H1 não passa. O relatório é escrito do mesmo jeito.

## Pela página de operação (desde 06/11/2026)

A página **/operacao** do front dispara os mesmos roteiros por um botão, pelo executor
`poc/cmd/ensaiador` (só na máquina local, em 127.0.0.1:8083):
- **Último dia**: `ultimo-dia.js` num multiplicador e numa duração; no fim, a auditoria da rodada;
- **Estresse: subir até quebrar**: dobra o multiplicador a cada patamar até o k6 estourar um limite
  ou a auditoria acusar algo, ou até o teto;
- **Pico do resultado**: `pico-resultado.js` com a faixa de inscrições homologadas lida do banco.

Cada patamar escolhe sozinho uma faixa de CPF livre. O k6 manda as métricas ao Prometheus local por
remote write (`--out experimental-prometheus-rw`), e a página embute os painéis do Grafana com o
sistema em tempo real. Para subir o executor, o Prometheus e o Grafana:
`..\observabilidade\subir-observabilidade.ps1`.

**Estado em 06/11:** o executor tem testes (escada, auditoria, parada, um ensaio por vez, só o
painel dispara). A página quebrou no navegador com o histórico vazio (o executor mandava `null`). A
correção está no código, com teste, mas **a página corrigida não foi conferida no navegador**, e
nenhum ensaio foi disparado por ela ainda.

## Rodar na máquina local (só para validar o roteiro)

```powershell
# API (8082), PSP simulado (8081) e front (8080) no ar; certame semeado (bin\lisura-seed.exe)
$env:BASE_URL='http://localhost:8082'; $env:FRONT_URL='http://localhost:8080'
$env:CERTAME='5'; $env:MULT='40'; $env:DURACAO='2m'; $env:PENSAR='0.05'; $env:RAJADAS='0'; $env:RODADA='401001'
k6 run ultimo-dia.js
```

## Pico 5 — o resultado (`pico-resultado.js`)

A base inteira abre o resultado ao mesmo tempo. Cada chegada é um candidato que já tem o código do
cartão no aparelho: calcula a chave e baixa a fatia estática (`/publico/certames/{id}/resultados/{xx}.json`).
**Nenhuma requisição vai à API.** O roteiro confere que a fatia chega e que a linha do candidato está
nela; o que ele quer provar se mede de fora: o contador de transações do banco (`xact_commit`) não
pode subir durante o pico além do que sobe parado.

O k6 recebe a `LISURA_TOKEN_KEY` para recalcular o código do cartão de cada inscrição — atalho do
disparador, para não precisar de 200 mil cartões guardados. No aparelho, o código vem do cartão.

```powershell
$env:PUBLICO_URL='http://localhost:8082'; $env:CERTAME='…'; $env:ID_MIN='…'; $env:ID_MAX='…'
$env:TOKEN_KEY='chave-local-da-poc'; $env:TAXA='1000'; $env:DURACAO='2m'
k6 run pico-resultado.js
```

Rodado localmente em 05/11/2026 (certame 6, 200 mil inscrições, `TAXA=1000`): 139.999 aberturas,
zero falhas, toda linha achada, p95 0,6 ms; banco a 12,7 transações/s parado e 9,3 durante o pico.

**Cada rodada precisa de uma faixa de CPF livre.** Os 2 últimos dígitos da `RODADA` escolhem uma de
99 faixas de 10 milhões de CPFs. Repetir a faixa de uma rodada anterior faz cada candidato cair numa
conta que já existe, e o k6 para a jornada com "CPF já existe (RODADA repetida?)". Isso aconteceu
na validação local de 05/11: 1.802 jornadas, nenhuma inscrição. No banco novo da AWS sobram as 99
faixas; no banco local, o roteiro de validação escolhe a primeira faixa sem nenhum candidato.

`PENSAR=0.05` comprime o tempo de pensar a 5%: a mesma chegada por minuto, com jornadas 20× mais
curtas, logo menos candidatos simultâneos do que no ensaio real. **Número local não é resultado do
ensaio**: só a rodada na AWS, com `PENSAR=1`, responde ao H1.
