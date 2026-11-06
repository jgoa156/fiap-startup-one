# Roda uma tarefa avulsa no ECS (seed, auditoria ou k6), espera terminar e imprime o log dela.
#
#   tarefa.ps1 seed
#   tarefa.ps1 k6 -Ambiente @{ CERTAME = '1'; MULT = '40'; DURACAO = '3h'; RODADA = '400001' }
#   tarefa.ps1 auditoria -Comando '/lisura-auditoria','-rodada','400001','-enviadas','104871','-pagas','93120'
#
# -Saida grava o log também num arquivo (UTF-8), uma mensagem por linha: é o que o relatório lê.
param(
    [Parameter(Mandatory)][ValidateSet('seed', 'auditoria', 'k6')][string]$Qual,
    [hashtable]$Ambiente = @{},
    [string[]]$Comando,
    [string]$Saida
)
$ErrorActionPreference = 'Stop'
$aws = (Get-Command aws -ErrorAction SilentlyContinue).Source
if (-not $aws) { $aws = 'C:\Users\rockb\tools\awscli\Amazon\AWSCLIV2\aws.exe' }
$tf = (Get-Command terraform -ErrorAction SilentlyContinue).Source
if (-not $tf) { $tf = 'C:\Users\rockb\tools\terraform\terraform.exe' }
$dirTf = (Resolve-Path "$PSScriptRoot\..\terraform").Path

$cluster = & $tf "-chdir=$dirTf" output -raw cluster
$rede = (& $tf "-chdir=$dirTf" output -json rede_tarefas) | ConvertFrom-Json
$familia = "lisura-poc-$Qual"

# Os parâmetros vão por arquivo (file://): passar JSON na linha de comando do PowerShell 5.1 para um
# programa nativo perde as aspas.
$override = @{ name = $Qual }
if ($Ambiente.Count) { $override.environment = @($Ambiente.GetEnumerator() | ForEach-Object { @{ name = $_.Key; value = [string]$_.Value } }) }
if ($Comando) { $override.command = $Comando }
$arqOverride = Join-Path $env:TEMP "lisura-override-$Qual.json"
@{ containerOverrides = @($override) } | ConvertTo-Json -Depth 6 | Set-Content -Encoding ascii $arqOverride
$arqRede = Join-Path $env:TEMP "lisura-rede.json"
@{ awsvpcConfiguration = @{ subnets = @($rede.subnets); securityGroups = @($rede.security_groups); assignPublicIp = 'ENABLED' } } |
    ConvertTo-Json -Depth 6 | Set-Content -Encoding ascii $arqRede

$arn = & $aws ecs run-task --cluster $cluster --task-definition $familia --launch-type FARGATE `
    --network-configuration "file://$arqRede" --overrides "file://$arqOverride" --query 'tasks[0].taskArn' --output text
if ($LASTEXITCODE -ne 0 -or $arn -eq 'None') { throw "run-task falhou" }
"tarefa $arn"
& $aws ecs wait tasks-stopped --cluster $cluster --tasks $arn
# `wait` desiste em 10 min; o k6 roda horas. Insiste até parar.
while ((& $aws ecs describe-tasks --cluster $cluster --tasks $arn --query 'tasks[0].lastStatus' --output text) -ne 'STOPPED') {
    Start-Sleep -Seconds 60
}
$codigo = & $aws ecs describe-tasks --cluster $cluster --tasks $arn --query 'tasks[0].containers[0].exitCode' --output text
$idTarefa = $arn.Split('/')[-1]
# O log vem em páginas (até 10 mil eventos ou 1 MB cada) e o resumo do k6 está no FIM: sem seguir as
# páginas, uma rodada de 3 h perderia justamente a linha RESUMO_DO_ENSAIO. Em JSON, não em texto: em
# texto o CLI emenda as mensagens com tabulação.
$linhas = New-Object System.Collections.Generic.List[string]
$token = $null
do {
    $a = @('logs', 'get-log-events', '--log-group-name', "/lisura-poc/$Qual", '--log-stream-name', "$Qual/$Qual/$idTarefa",
        '--start-from-head', '--output', 'json')
    if ($token) { $a += @('--next-token', $token) }
    $pagina = (& $aws @a) | Out-String | ConvertFrom-Json
    foreach ($e in $pagina.events) { $linhas.Add($e.message) }
    $anterior = $token
    $token = $pagina.nextForwardToken
} while ($pagina.events.Count -gt 0 -and $token -ne $anterior) # o fim é o mesmo token voltando
$linhas
if ($Saida) { [IO.File]::WriteAllLines($Saida, $linhas, (New-Object Text.UTF8Encoding $false)) }
"código de saída: $codigo"
