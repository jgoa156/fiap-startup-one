# Gera o PDF do pitch a partir de pitch-lisura.html (Edge headless, sem Python nem Node).
#   powershell -ExecutionPolicy Bypass -File build-pitch.ps1

$ErrorActionPreference = "Stop"
$outDir = Split-Path -Parent $MyInvocation.MyCommand.Path   # ...\fiap-startup-one\apresentacao
$src    = Join-Path $outDir "pitch-lisura.html"
$out    = Join-Path $outDir "Lisura-pitch.pdf"

if (-not (Test-Path $src)) { throw "Fonte nao encontrada: $src" }

$edge = "C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe"
if (-not (Test-Path $edge)) { $edge = "C:\Program Files\Microsoft\Edge\Application\msedge.exe" }
$profile = Join-Path $env:TEMP ("edgepdf-" + [Guid]::NewGuid().ToString("N").Substring(0,8))

if (Test-Path $out) { Remove-Item $out -Force }
$srcUrl = "file:///" + ($src -replace '\\','/')
$a = @("--headless=new","--disable-gpu","--no-first-run","--no-sandbox",
       "--user-data-dir=$profile","--no-pdf-header-footer",
       "--run-all-compositor-stages-before-draw","--virtual-time-budget=25000",
       "--print-to-pdf=$out", $srcUrl)
$pr = Start-Process -FilePath $edge -ArgumentList $a -Wait -PassThru -NoNewWindow

if (Test-Path $out) { Write-Output ("PDF  {0}  ({1:N0} KB)" -f (Split-Path -Leaf $out), ((Get-Item $out).Length/1KB)) }
else { Write-Output ("FALHOU (exit {0}): {1}" -f $pr.ExitCode, $out) }

Remove-Item -Recurse -Force $profile -ErrorAction SilentlyContinue