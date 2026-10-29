# Gera os PDFs e os previews de slides e roteiro a partir de deck.html.
#   powershell -ExecutionPolicy Bypass -File build-pdf.ps1
#
# O HTML fonte mora AQUI, no repositorio. Nao voltar a deixa-lo em pasta
# temporaria: foi assim que a fonte do deck anterior se perdeu.

$ErrorActionPreference = "Stop"
$here    = Split-Path -Parent $MyInvocation.MyCommand.Path
$src     = Join-Path $here "deck.html"
$outDir  = $here
$prevDir = Join-Path $here "previews"
$png     = Join-Path (Split-Path -Parent $here) 'diagramas\arquitetura-alvo.png'
$work    = Join-Path $env:TEMP ("deckbuild-" + [Guid]::NewGuid().ToString("N").Substring(0,8))
New-Item -ItemType Directory -Force -Path $work    | Out-Null
New-Item -ItemType Directory -Force -Path $prevDir | Out-Null

$html = [System.IO.File]::ReadAllText($src, (New-Object System.Text.UTF8Encoding $false))

# ---- diagrama embutido como data URI --------------------------------------
if (Test-Path $png) {
  $b64  = [Convert]::ToBase64String([System.IO.File]::ReadAllBytes($png))
  $html = $html.Replace("{{DIAGRAMA}}", "data:image/png;base64,$b64")
} else {
  throw "Diagrama nao encontrado: $png"
}

$enc = New-Object System.Text.UTF8Encoding $true

# ---- variante slides: 1280x720, uma pagina por slide ----------------------
$deck = Join-Path $work "out-deck.html"
[System.IO.File]::WriteAllText($deck, $html, $enc)

# ---- variante roteiro: A4, miniatura do slide + fala + marcacao -----------
$rot = Join-Path $work "out-roteiro.html"
$rotHtml = $html.Replace('data-theme="light"', 'data-theme="light" data-mode="roteiro"')
$rotHtml = $rotHtml.Replace('@page{size:1280px 720px;margin:0}', '@page{size:A4;margin:13mm 12mm 14mm}')
[System.IO.File]::WriteAllText($rot, $rotHtml, $enc)

# ---- navegador Chromium ---------------------------------------------------
# Edge saiu desta maquina em set/2026; o Chrome ficou. Qualquer um dos dois serve.
$browser = $null
foreach ($c in @(
  "C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe",
  "C:\Program Files\Microsoft\Edge\Application\msedge.exe",
  "$env:LOCALAPPDATA\Microsoft\Edge\Application\msedge.exe",
  "C:\Program Files\Google\Chrome\Application\chrome.exe",
  "C:\Program Files (x86)\Google\Chrome\Application\chrome.exe",
  "$env:LOCALAPPDATA\Google\Chrome\Application\chrome.exe")) {
  if (Test-Path $c) { $browser = $c; break }
}
if (-not $browser) { throw "Nenhum navegador Chromium (Edge ou Chrome) encontrado." }
Write-Output ("Navegador: " + $browser)

function ToPdf($inFile, $outFile) {
  if (Test-Path $outFile) { Remove-Item $outFile -Force }
  $prof = Join-Path $env:TEMP ("chpdf-" + [Guid]::NewGuid().ToString("N").Substring(0,8))
  $uri  = "file:///" + ($inFile -replace '\\','/')
  $a = @("--headless=new","--disable-gpu","--no-first-run","--no-default-browser-check",
         "--no-sandbox","--disable-extensions","--user-data-dir=$prof",
         "--no-pdf-header-footer","--run-all-compositor-stages-before-draw",
         "--virtual-time-budget=30000","--print-to-pdf=$outFile", $uri)

  # NAO usar -Wait aqui: o Chrome headless deixa processos filhos vivos, e
  # -Wait espera a arvore inteira e trava o script. Espera-se o ARQUIVO
  # aparecer e parar de crescer.
  $pr = Start-Process -FilePath $browser -ArgumentList $a -PassThru -NoNewWindow
  $last = -1; $stable = 0; $waited = 0
  while ($waited -lt 180) {
    Start-Sleep -Seconds 1
    $waited++
    if (Test-Path $outFile) {
      $sz = (Get-Item $outFile).Length
      if ($sz -gt 0 -and $sz -eq $last) { $stable++ } else { $stable = 0 }
      $last = $sz
      if ($stable -ge 3) { break }
    }
  }
  & cmd /c "taskkill /PID $($pr.Id) /T /F >nul 2>&1"
  Start-Sleep -Milliseconds 400
  Remove-Item -Recurse -Force $prof -ErrorAction SilentlyContinue

  if ((Test-Path $outFile) -and ((Get-Item $outFile).Length -gt 0)) {
    Write-Output ("PDF  {0}  ({1:N0} KB)" -f (Split-Path -Leaf $outFile), ((Get-Item $outFile).Length/1KB))
  } else {
    throw ("FALHOU: " + $outFile)
  }
}

ToPdf $deck (Join-Path $outDir "Certamis-multicloud-slides.pdf")
ToPdf $rot  (Join-Path $outDir "Certamis-multicloud-roteiro.pdf")

# ---- previews PNG, uma imagem por pagina ----------------------------------
$py = $null
foreach ($c in @("$env:LOCALAPPDATA\Microsoft\WindowsApps\python.exe",
                 "$env:LOCALAPPDATA\Programs\Python\Python313\python.exe","python.exe")) {
  if (Get-Command $c -ErrorAction SilentlyContinue) { $py = $c; break }
}
if ($py) {
  $script = Join-Path $work "prev.py"
  $code = @'
import sys, pymupdf
pdf, outdir, prefix = sys.argv[1], sys.argv[2], sys.argv[3]
d = pymupdf.open(pdf)
for i, p in enumerate(d):
    p.get_pixmap(dpi=96).save("%s/%s-%02d.png" % (outdir, prefix, i + 1))
print("previews gerados:", len(d))
'@
  [System.IO.File]::WriteAllText($script, $code, (New-Object System.Text.UTF8Encoding $false))
  # sem 2>&1: no PowerShell 5.1 isso transforma stderr de exe nativo em erro fatal
  $ErrorActionPreference = "Continue"
  try { & $py -m pip install pymupdf --quiet | Out-Null } catch { }
  $ErrorActionPreference = "Stop"
  Get-ChildItem $prevDir -Filter "slide-*.png" -ErrorAction SilentlyContinue | Remove-Item -Force
  & $py $script (Join-Path $outDir "Certamis-multicloud-slides.pdf") $prevDir "slide"
} else {
  Write-Output "Python nao encontrado: previews PNG nao foram gerados."
}

Remove-Item -Recurse -Force $work -ErrorAction SilentlyContinue
Write-Output "Pronto."
