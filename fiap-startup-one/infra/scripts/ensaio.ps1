# Roda o ensaio inteiro, patamar por patamar, até o relatório: k6 → auditoria daquela rodada →
# manifesto → lisura-relatorio. Para no primeiro patamar que falhar (é o ponto de quebra); com
# -AteQuebrar, depois da lista continua dobrando a carga até falhar ou chegar ao -Teto, porque um
# ensaio que só passa não informa o limite (04-escopo-poc §8).
#
#   # na AWS (depois do terraform apply e do seed):
#   ensaio.ps1 -Onde aws -Certame 1 -Patamares 10,20,40 -Duracao 3h -AteQuebrar
#   # na máquina local (validação da cadeia; número local não é resultado):
#   ensaio.ps1 -Onde local -Certame 5 -Patamares 40 -Duracao 2m -Pensar 0.05 -Rajadas 0 -AteQuebrar -Teto 640
#
# Cada patamar usa uma faixa de CPF própria (os 2 últimos dígitos da RODADA, 1 de 99): na AWS, a
# partir de -PrimeiraFaixa (banco novo: 1); na máquina local, as primeiras faixas sem nenhum candidato.
# Tudo vai para -Pasta: k6-<N>x.log, auditoria-<N>x.json, manifesto.json, relatorio.md, veredito.json.
# O custo de cada patamar fica em 0 no manifesto: preencha custo_usd com o Cost Explorer (a janela de
# cada rodada está no manifesto e no relatório) e rode lisura-relatorio de novo.
param(
    [ValidateSet('local', 'aws')][string]$Onde = 'local',
    [Parameter(Mandatory)][string]$Certame,
    [double[]]$Patamares = @(10, 20, 40),
    [string]$Duracao = '3h',
    [double]$Pensar = 1,
    [string]$Rajadas = '1',
    [switch]$AteQuebrar,
    [double]$Teto = 1280,
    [int]$PrimeiraFaixa = 1,
    [double]$Orcamento = 4.44, # US$/h por patamar: 04-escopo-poc §5 (US$ 40 para 3 patamares de 3 h)
    [string]$Pasta,
    [string]$Titulo = 'Ensaio do H1',
    [string]$Ambiente = '',
    [string]$BaseUrl = 'http://localhost:8082', # só local
    [string]$FrontUrl = 'http://localhost:8090' # só local: o Next de produção (o `next dev` falseia a medida)
)
$ErrorActionPreference = 'Stop'
$raiz = (Resolve-Path "$PSScriptRoot\..\..").Path
$poc = Join-Path $raiz 'poc'
if (-not $Pasta) { $Pasta = Join-Path $raiz ("carga\resultados\ensaio-" + (Get-Date).ToString('yyyyMMdd-HHmm')) }
New-Item -ItemType Directory -Force $Pasta | Out-Null
$Pasta = (Resolve-Path $Pasta).Path # o .NET resolve caminho relativo pela pasta do processo, não a do PowerShell
$utf8 = New-Object Text.UTF8Encoding $false

# Programa externo pelo cmd, com saída e erro em arquivo: no PowerShell 5.1, com
# ErrorActionPreference = Stop, qualquer linha na saída de erro de um programa redirecionada vira erro
# fatal — e a auditoria e o relatório escrevem log nela.
function Externo([string]$exe, [string[]]$argumentos, [string]$saida) {
    $linha = ($argumentos | ForEach-Object { "`"$_`"" }) -join ' '
    cmd /c "`"`"$exe`" $linha > `"$saida`" 2> `"$saida.err`"`""
    return $LASTEXITCODE
}

# O relatório roda sempre aqui; a auditoria, aqui (local) ou no ECS (aws).
$go = (Get-Command go -ErrorAction SilentlyContinue).Source
if (-not $go) { $go = 'C:\Users\rockb\tools\go\bin\go.exe' }
Push-Location $poc
& $go build -o bin\ ./cmd/relatorio ./cmd/auditoria
Pop-Location
if ($LASTEXITCODE -ne 0) { throw 'go build falhou' }
$relatorio = Join-Path $poc 'bin\relatorio.exe'
$auditoriaLocal = Join-Path $poc 'bin\auditoria.exe'
$k6 = (Get-Command k6 -ErrorAction SilentlyContinue).Source
if (-not $k6) { $k6 = 'C:\Users\rockb\tools\k6\k6.exe' }

function FaixasLivres {
    if ($Onde -eq 'aws') { return @($PrimeiraFaixa..99) }
    $psql = (Get-Command psql -ErrorAction SilentlyContinue).Source
    if (-not $psql) { $psql = 'C:\Users\rockb\tools\pgsql\bin\psql.exe' }
    $usadas = & $psql -h localhost -U postgres -d lisura -At -c 'SELECT DISTINCT left(cpf, 2) FROM candidato'
    return @(1..99 | Where-Object { $usadas -notcontains $_.ToString('00') })
}
$faixas = FaixasLivres

$manifesto = [ordered]@{ titulo = $Titulo; ambiente = $Ambiente; orcamento_usd_hora = $Orcamento; patamares = @() }
$fila = New-Object System.Collections.Generic.Queue[double]
$Patamares | ForEach-Object { $fila.Enqueue($_) }
$i = 0
$ultimo = 0
while ($fila.Count -gt 0) {
    $mult = $fila.Dequeue()
    if ($i -ge $faixas.Count) { throw 'acabaram as faixas de CPF livres (99 por banco)' }
    $faixa = $faixas[$i]; $i++
    $rodada = '{0}{1:D2}' -f [int][math]::Round($mult), $faixa
    $nome = "$($mult)x"
    $logK6 = Join-Path $Pasta "k6-$nome.log"
    $jsonAud = Join-Path $Pasta "auditoria-$nome.json"
    Write-Host "=== $nome · rodada $rodada · faixa de CPF $($faixa.ToString('00')) · $Duracao"

    $inicio = (Get-Date).ToUniversalTime()
    $amb = @{ CERTAME = $Certame; MULT = [string]$mult; DURACAO = $Duracao; RODADA = $rodada; PENSAR = [string]$Pensar; RAJADAS = $Rajadas }
    if ($Onde -eq 'local') {
        foreach ($k in $amb.Keys) { Set-Item "env:$k" $amb[$k] }
        $env:BASE_URL = $BaseUrl; $env:FRONT_URL = $FrontUrl
        Push-Location (Join-Path $raiz 'carga')
        # cmd /c também guarda os bytes como o k6 escreveu (o redirecionamento do PowerShell 5.1 grava
        # UTF-16). O k6 escreve o resumo na saída e os avisos no erro: os dois vão para o mesmo log.
        cmd /c "`"`"$k6`" run --quiet ultimo-dia.js > `"$logK6`" 2>&1`""
        Pop-Location
    } else {
        & "$PSScriptRoot\tarefa.ps1" k6 -Ambiente $amb -Saida $logK6 | Out-Null
    }
    $fim = (Get-Date).ToUniversalTime()

    $m = Select-String -Path $logK6 -Pattern 'PARA_AUDITAR lisura-auditoria (-rodada \S+ -enviadas \d+ -pagas \d+)' | Select-Object -First 1
    if (-not $m) { throw "o k6 do patamar $nome não imprimiu PARA_AUDITAR (ver $logK6)" }
    $argsAud = $m.Matches[0].Groups[1].Value -split '\s+'
    if ($Onde -eq 'local') {
        Externo $auditoriaLocal $argsAud $jsonAud | Out-Null # sai com 1 se achou problema: o relatório julga
    } else {
        & "$PSScriptRoot\tarefa.ps1" auditoria -Comando (@('/lisura-auditoria') + $argsAud) -Saida $jsonAud | Out-Null
    }

    $manifesto.patamares += [ordered]@{
        k6 = (Split-Path $logK6 -Leaf); auditoria = (Split-Path $jsonAud -Leaf); custo_usd = 0
        inicio = $inicio.ToString('yyyy-MM-ddTHH:mm:ssZ'); fim = $fim.ToString('yyyy-MM-ddTHH:mm:ssZ')
        notas = "Rodada $rodada, faixa de CPF $($faixa.ToString('00')), PENSAR=$Pensar, RAJADAS=$Rajadas."
    }
    $arqManifesto = Join-Path $Pasta 'manifesto.json'
    [IO.File]::WriteAllText($arqManifesto, ($manifesto | ConvertTo-Json -Depth 5), $utf8)
    $arqVeredito = Join-Path $Pasta 'veredito.json'
    Remove-Item $arqVeredito -ErrorAction SilentlyContinue # o do patamar anterior não pode passar por este
    Externo $relatorio @('-manifesto', $arqManifesto, '-saida', (Join-Path $Pasta 'relatorio.md'), '-veredito', $arqVeredito) `
        (Join-Path $Pasta 'relatorio.log') | Out-Null
    if (-not (Test-Path $arqVeredito)) { throw "o relatório não saiu (ver $(Join-Path $Pasta 'relatorio.log.err'))" }
    $v =[IO.File]::ReadAllText($arqVeredito) | ConvertFrom-Json
    $este = $v.patamares | Where-Object { $_.rodada -eq $rodada } | Select-Object -First 1
    if (-not $este.passou) {
        Write-Host "=== $nome FALHOU ($($este.falhou -join '; ')): é o ponto de quebra. Parando."
        break
    }
    Write-Host "=== $nome passou"
    $ultimo = $mult
    if ($fila.Count -eq 0 -and $AteQuebrar -and $mult * 2 -le $Teto) { $fila.Enqueue($mult * 2) }
}
if ($fila.Count -eq 0 -and $ultimo -gt 0 -and $AteQuebrar -and $ultimo * 2 -gt $Teto) {
    Write-Host "=== chegou ao teto ($Teto×) sem quebrar"
}
$v = [IO.File]::ReadAllText((Join-Path $Pasta 'veredito.json')) | ConvertFrom-Json
Write-Host "=== H1: $(if ($v.h1) { 'PASSOU' } else { 'NÃO PASSOU' }) — $($v.porque)"
Write-Host "=== relatório: $(Join-Path $Pasta 'relatorio.md')"
exit $(if ($v.h1) { 0 } else { 1 }) # como o lisura-relatorio: 1 = o H1 não passou
