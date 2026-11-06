# Sobe, na máquina local, o Prometheus (127.0.0.1:9090), o Grafana (127.0.0.1:3000) e o executor de
# ensaios (127.0.0.1:8083), cada um como processo independente. A API e o front sobem à parte (ver
# poc/README.md, "Como rodar"). Rodar de novo reinicia os três.
#
#   -Ferramentas  onde estão o Prometheus e o Grafana extraídos (padrão C:\Users\rockb\tools)
#   -Dados        onde ficam as séries do Prometheus e o banco do Grafana (fora do repositório)
#
# Versões: Prometheus 3.13.5 (21/10/2026) e Grafana 13.2.4 (13/10/2026), SHA-256 conferidos com os
# publicados (sha256sums.txt do Prometheus; API de downloads do grafana.com).
param(
    [string]$Ferramentas = 'C:\Users\rockb\tools',
    [string]$Dados = 'C:\Users\rockb\tools\observabilidade-dados',
    [string]$SegredoOperacao = $env:LISURA_OPERACAO_SEGREDO,
    [string]$TokenKey = $env:LISURA_TOKEN_KEY
)
$ErrorActionPreference = 'Stop'
$aqui = $PSScriptRoot
$raiz = (Resolve-Path "$aqui\..").Path
$prom = Join-Path $Ferramentas 'prometheus-3.13.5.windows-amd64\prometheus.exe'
$grafanaCasa = Get-ChildItem (Join-Path $Ferramentas 'grafana-13.2.4-pacote') -Directory | Select-Object -First 1 -ExpandProperty FullName
if (-not (Test-Path $prom)) { throw "Prometheus não encontrado em $prom" }
if (-not $grafanaCasa) { throw "Grafana não encontrado em $Ferramentas\grafana-13.2.4-pacote" }
New-Item -ItemType Directory -Force "$Dados\prometheus", "$Dados\grafana", "$Dados\logs" | Out-Null

# Pelo nome do processo (prometheus, grafana, ensaiador): o WMI desta máquina trava, e os nomes são
# só destes programas.
function Parar([string]$nome, [string]$caminho) {
    Get-Process -Name ([IO.Path]::GetFileNameWithoutExtension($nome)) -ErrorAction SilentlyContinue |
        Stop-Process -Force -Confirm:$false -ErrorAction SilentlyContinue
}
function Esperar([string]$url) {
    for ($i = 0; $i -lt 60; $i++) {
        try { Invoke-WebRequest -UseBasicParsing -TimeoutSec 2 $url | Out-Null; return $true } catch { Start-Sleep 1 }
    }
    return $false
}

# Prometheus: remote write ligado (o k6 escreve aqui), 7 dias de séries.
Parar 'prometheus.exe' $prom
Start-Process -FilePath $prom -WorkingDirectory $aqui -WindowStyle Hidden `
    -RedirectStandardError "$Dados\logs\prometheus.log" -RedirectStandardOutput "$Dados\logs\prometheus.out" `
    -ArgumentList "--config.file=`"$aqui\prometheus.yml`"", "--storage.tsdb.path=`"$Dados\prometheus`"",
        '--web.listen-address=127.0.0.1:9090', '--web.enable-remote-write-receiver', '--storage.tsdb.retention.time=7d' | Out-Null
"Prometheus: $(if (Esperar 'http://127.0.0.1:9090/-/ready') { 'no ar em http://127.0.0.1:9090' } else { 'NÃO subiu (ver ' + $Dados + '\logs\prometheus.log)' })"

# Grafana: só leitura, sem login, painéis do repositório.
$grafana = Join-Path $grafanaCasa 'bin\grafana.exe'
Parar 'grafana.exe' $grafana
$env:GF_PATHS_DATA = "$Dados\grafana"
$env:GF_PATHS_LOGS = "$Dados\logs"
$env:GF_PATHS_PROVISIONING = "$aqui\grafana\provisionamento"
$env:GF_DASHBOARDS_DEFAULT_HOME_DASHBOARD_PATH = "$aqui\grafana\paineis\lisura.json"
$env:PAINEIS = "$aqui\grafana\paineis"
Start-Process -FilePath $grafana -WorkingDirectory $grafanaCasa -WindowStyle Hidden `
    -RedirectStandardError "$Dados\logs\grafana.err" -RedirectStandardOutput "$Dados\logs\grafana.out" `
    -ArgumentList 'server', "--homepath=`"$grafanaCasa`"", "--config=`"$aqui\grafana\grafana.ini`"" | Out-Null
"Grafana: $(if (Esperar 'http://127.0.0.1:3000/api/health') { 'no ar em http://127.0.0.1:3000' } else { 'NÃO subiu (ver ' + $Dados + '\logs\grafana.err)' })"

# Executor de ensaios: compilado do código atual.
$poc = Join-Path $raiz 'poc'
$go = (Get-Command go -ErrorAction SilentlyContinue).Source
if (-not $go) { $go = Join-Path $Ferramentas 'go\bin\go.exe' }
Push-Location $poc
& $go build -o bin\ensaiador.exe ./cmd/ensaiador
Pop-Location
if ($LASTEXITCODE -ne 0) { throw 'go build do ensaiador falhou' }
$ens = Join-Path $poc 'bin\ensaiador.exe'
Parar 'ensaiador.exe' $ens
$env:ENSAIADOR_ROTEIROS = Join-Path $raiz 'carga'
$env:LISURA_OPERACAO_SEGREDO = $SegredoOperacao
$env:LISURA_TOKEN_KEY = $TokenKey
Start-Process -FilePath $ens -WorkingDirectory $poc -WindowStyle Hidden `
    -RedirectStandardError "$Dados\logs\ensaiador.log" -RedirectStandardOutput "$Dados\logs\ensaiador.out" | Out-Null
"Executor de ensaios: $(if (Esperar 'http://127.0.0.1:8083/estado') { 'no ar em http://127.0.0.1:8083' } else { 'NÃO subiu (ver ' + $Dados + '\logs\ensaiador.log)' })"
