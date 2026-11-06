# Gera os PDFs do deck e do roteiro a partir de deck-lisura.html.
#   powershell -ExecutionPolicy Bypass -File build-pdf.ps1

$ErrorActionPreference = "Stop"
$outDir = Split-Path -Parent $MyInvocation.MyCommand.Path   # ...\fiap-multicloud\apresentacao
$src    = Join-Path $outDir "deck-lisura.html"              # fonte dos slides, versionada aqui
$png    = Join-Path (Split-Path -Parent $outDir) "diagramas\arquitetura-alvo.png"
$work   = Join-Path $env:TEMP ("lisura-build-" + [Guid]::NewGuid().ToString("N").Substring(0,8))
New-Item -ItemType Directory -Path $work -Force | Out-Null

# ── diagrama embutido como data URI ─────────────────────────────────────
$b64 = [Convert]::ToBase64String([System.IO.File]::ReadAllBytes($png))
$html = [System.IO.File]::ReadAllText($src, (New-Object System.Text.UTF8Encoding $false))
$html = $html.Replace("{{DIAGRAMA}}", "data:image/png;base64,$b64")

$enc = New-Object System.Text.UTF8Encoding $true

# deck: 1280x720, uma pagina por slide
$deck = Join-Path $work "out-deck.html"
[System.IO.File]::WriteAllText($deck, $html, $enc)

# roteiro: A4, slide + fala + marcacao
$rot = Join-Path $work "out-roteiro.html"
$rotHtml = $html.Replace('data-theme="light"', 'data-theme="light" data-mode="roteiro"')
$rotHtml = $rotHtml.Replace('@page{size:1280px 720px;margin:0}', '@page{size:A4;margin:14mm 12mm 16mm}')
[System.IO.File]::WriteAllText($rot, $rotHtml, $enc)

Write-Output ("HTML: {0:N0} KB + {1:N0} KB" -f ((Get-Item $deck).Length/1KB), ((Get-Item $rot).Length/1KB))

# ── Edge headless -> PDF ────────────────────────────────────────────────
$edge = "C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe"
if (-not (Test-Path $edge)) { $edge = "C:\Program Files\Microsoft\Edge\Application\msedge.exe" }
$profile = Join-Path $env:TEMP ("edgepdf-" + [Guid]::NewGuid().ToString("N").Substring(0,8))

function ToPdf($inFile, $outFile){
  if (Test-Path $outFile) { Remove-Item $outFile -Force }
  $a = @("--headless=new","--disable-gpu","--no-first-run","--no-sandbox",
         "--user-data-dir=$profile","--no-pdf-header-footer",
         "--run-all-compositor-stages-before-draw","--virtual-time-budget=25000",
         "--print-to-pdf=$outFile", ("file:///" + $inFile.Replace('\','/')))
  $pr = Start-Process -FilePath $edge -ArgumentList $a -Wait -PassThru -NoNewWindow
  if (Test-Path $outFile) { Write-Output ("PDF  {0}  ({1:N0} KB)" -f (Split-Path -Leaf $outFile), ((Get-Item $outFile).Length/1KB)) }
  else { Write-Output ("FALHOU (exit {0}): {1}" -f $pr.ExitCode, $outFile) }
}

ToPdf $deck (Join-Path $outDir "Lisura-multicloud-slides.pdf")
ToPdf $rot  (Join-Path $outDir "Lisura-multicloud-roteiro.pdf")

Remove-Item -Recurse -Force $profile -ErrorAction SilentlyContinue
Remove-Item -Recurse -Force $work    -ErrorAction SilentlyContinue
