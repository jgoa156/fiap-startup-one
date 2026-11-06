# Constrói as três imagens (Go, front, k6) no CodeBuild e envia ao ECR.
#
#   powershell -ExecutionPolicy Bypass -File infra\scripts\publicar-imagens.ps1
#   (depois) terraform -chdir=infra\terraform apply -var tag_imagem=<a tag impressa>
#
# Pré-requisito: `terraform apply` já rodou uma vez (cria o bucket, os repositórios e o projeto do
# CodeBuild) e a AWS CLI tem credencial (`aws sts get-caller-identity` responde).
param(
    [string]$Tag = (Get-Date -Format 'yyyyMMdd-HHmmss')
)
$ErrorActionPreference = 'Stop'
$aws = (Get-Command aws -ErrorAction SilentlyContinue).Source
if (-not $aws) { $aws = 'C:\Users\rockb\tools\awscli\Amazon\AWSCLIV2\aws.exe' }
$tf = (Get-Command terraform -ErrorAction SilentlyContinue).Source
if (-not $tf) { $tf = 'C:\Users\rockb\tools\terraform\terraform.exe' }

$raiz = (Resolve-Path "$PSScriptRoot\..\..").Path   # fiap-startup-one
$dirTf = Join-Path $raiz 'infra\terraform'
$bucket = & $tf "-chdir=$dirTf" output -raw bucket
$projeto = & $tf "-chdir=$dirTf" output -raw projeto_codebuild

# Fonte enxuta: sem node_modules, .next, binários, dados locais nem estado do Terraform.
$palco = Join-Path $env:TEMP "lisura-fonte-$Tag"
if (Test-Path $palco) { Remove-Item $palco -Recurse -Force }
foreach ($pasta in 'poc', 'poc-web', 'carga', 'infra\imagens') {
    robocopy (Join-Path $raiz $pasta) (Join-Path $palco $pasta) /E /XD node_modules .next bin dados .terraform docs /NFL /NDL /NJH /NJS /NP | Out-Null
    if ($LASTEXITCODE -ge 8) { throw "robocopy $pasta falhou ($LASTEXITCODE)" }
}
Copy-Item (Join-Path $raiz 'infra\buildspec.yml') (Join-Path $palco 'infra\buildspec.yml')

# tar.exe (bsdtar, vem no Windows) grava o zip com "/" nos caminhos; o Compress-Archive do
# PowerShell 5.1 grava "\" e o Linux do CodeBuild não acharia as pastas.
$zip = Join-Path $env:TEMP "lisura-fonte-$Tag.zip"
& tar.exe -a -c -f $zip -C $palco .
if ($LASTEXITCODE -ne 0) { throw "tar falhou" }

& $aws s3 cp $zip "s3://$bucket/fonte/fonte.zip" --only-show-errors
if ($LASTEXITCODE -ne 0) { throw "envio da fonte falhou" }

$id = & $aws codebuild start-build --project-name $projeto --environment-variables-override "name=TAG,value=$Tag,type=PLAINTEXT" --query 'build.id' --output text
if ($LASTEXITCODE -ne 0) { throw "start-build falhou" }
"construção $id (tag $Tag)"
do {
    Start-Sleep -Seconds 20
    $estado = & $aws codebuild batch-get-builds --ids $id --query 'builds[0].buildStatus' --output text
    "  $estado"
} while ($estado -eq 'IN_PROGRESS')
if ($estado -ne 'SUCCEEDED') { throw "construção terminou em $estado (log no CloudWatch: /lisura-poc/codebuild)" }

Remove-Item $palco -Recurse -Force
Remove-Item $zip -Force
""
"Imagens publicadas com a tag $Tag. Agora:"
"  terraform -chdir=`"$dirTf`" apply -var tag_imagem=$Tag -var ligado=true"
