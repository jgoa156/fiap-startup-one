# Certamis - arquitetura alvo. Gera o PNG usado no slide 7.
# Uso:  powershell -ExecutionPolicy Bypass -File render-arquitetura.ps1

Add-Type -AssemblyName System.Drawing
$S = 2; $W = 1440; $H = 1020
$bmp = New-Object System.Drawing.Bitmap ($W*$S), ($H*$S)
$g   = [System.Drawing.Graphics]::FromImage($bmp)
$g.SmoothingMode     = [System.Drawing.Drawing2D.SmoothingMode]::AntiAlias
$g.TextRenderingHint = [System.Drawing.Text.TextRenderingHint]::ClearTypeGridFit
$g.Clear([System.Drawing.Color]::White); $g.ScaleTransform($S,$S)

function C([string]$h){ [System.Drawing.ColorTranslator]::FromHtml($h) }
$IK="#1E1E1E"; $GY="#6C757D"; $LT="#F8F9FA"; $RL="#DEE2E6"
$VI="#6741D9"; $VIF="#EDE4FF"      # fora da nuvem primaria
$OR="#E8590C"; $ORF="#FFF3BF"      # AWS
$GR="#2F9E44"; $GRF="#D3F9D8"      # estatico
$BL="#1971C2"; $BLF="#D0EBFF"      # celulas
$RD="#C92A2A"; $RDF="#FFE3E3"      # celula dedicada

function RR($x,$y,$w,$h,$r){ $p=New-Object System.Drawing.Drawing2D.GraphicsPath; $d=$r*2
  $p.AddArc($x,$y,$d,$d,180,90); $p.AddArc($x+$w-$d,$y,$d,$d,270,90)
  $p.AddArc($x+$w-$d,$y+$h-$d,$d,$d,0,90); $p.AddArc($x,$y+$h-$d,$d,$d,90,90)
  $p.CloseFigure(); return $p }
function Box($x,$y,$w,$h,$s,$f,$wd,$r,$dash){ if(-not $r){$r=8}; $p=RR $x $y $w $h $r
  if($f){$b=New-Object System.Drawing.SolidBrush (C $f); $g.FillPath($b,$p); $b.Dispose()}
  if($s){$pen=New-Object System.Drawing.Pen ((C $s),$wd)
    if($dash){$pen.DashStyle=[System.Drawing.Drawing2D.DashStyle]::Dash}
    $g.DrawPath($pen,$p); $pen.Dispose()}; $p.Dispose() }
function Txt($x,$y,$sz,$c,$t,$b){ $st=if($b){[System.Drawing.FontStyle]::Bold}else{[System.Drawing.FontStyle]::Regular}
  $f=New-Object System.Drawing.Font("Segoe UI",$sz,$st,[System.Drawing.GraphicsUnit]::Pixel)
  $br=New-Object System.Drawing.SolidBrush (C $c); $g.DrawString($t,$f,$br,[single]$x,[single]$y); $f.Dispose(); $br.Dispose() }
function TxtC($cx,$y,$sz,$c,$t,$b){ $st=if($b){[System.Drawing.FontStyle]::Bold}else{[System.Drawing.FontStyle]::Regular}
  $f=New-Object System.Drawing.Font("Segoe UI",$sz,$st,[System.Drawing.GraphicsUnit]::Pixel)
  $s=$g.MeasureString($t,$f); $br=New-Object System.Drawing.SolidBrush (C $c)
  $g.DrawString($t,$f,$br,[single]($cx-$s.Width/2),[single]$y); $f.Dispose(); $br.Dispose() }
function Arr($x1,$y1,$x2,$y2,$c,$w){ $pen=New-Object System.Drawing.Pen ((C $c),$w)
  $g.DrawLine($pen,[single]$x1,[single]$y1,[single]$x2,[single]$y2)
  $a=[Math]::Atan2($y2-$y1,$x2-$x1); $L=12; $O=0.42
  $g.DrawLine($pen,[single]$x2,[single]$y2,[single]($x2-$L*[Math]::Cos($a-$O)),[single]($y2-$L*[Math]::Sin($a-$O)))
  $g.DrawLine($pen,[single]$x2,[single]$y2,[single]($x2-$L*[Math]::Cos($a+$O)),[single]($y2-$L*[Math]::Sin($a+$O))); $pen.Dispose() }

# ── titulo ──────────────────────────────────────────────────────────────
Txt 40 6 27 $IK "Certamis · arquitetura alvo: uma nuvem para operar, duas para sobreviver" $true
Txt 40 42 15 $GY "O calendário do edital é dado de entrada da infraestrutura. Roxo = fora da nuvem primária." $false

# ── candidato ───────────────────────────────────────────────────────────
TxtC 720 78 14 $GY "CANDIDATO" $true
Arr 720 98 720 118 $IK 2

# ── cloudflare ──────────────────────────────────────────────────────────
Box 40 120 1360 64 $VI $VIF 2.5
TxtC 720 132 18 "#5F3DC4" "CLOUDFLARE · DNS · WAF · FILA DE ESPERA" $true
TxtC 720 158 14 $IK "plano de controle neutro: se a AWS cai, o DNS continua de pé" $false
Txt 1180 128 12 $VI "FORA DA AWS" $true

Arr 720 186 720 208 $IK 2

# ── AWS ─────────────────────────────────────────────────────────────────
Box 40 210 1360 570 $OR $null 2.5 10 $true
Txt 62 222 18 $OR "AWS sa-east-1 · 3 zonas de disponibilidade" $true
Txt 62 248 13 $GY "região única. Rio de Janeiro é Local Zone, não é região, e não isola falha." $false

# publicacao estatica
Box 62 274 1316 92 $GR $GRF 2.5
Txt 84 286 17 "#2B8A3E" "CLOUDFRONT + S3 · PUBLICAÇÃO ESTÁTICA" $true
Txt 84 312 14 $IK "gabarito · cartão de confirmação · resultado e classificação · com cópia no Google Cloud, servida se a região cair" $false
Txt 84 334 14 "#2B8A3E" "Os 3 maiores picos são leitura de dado que não muda. Sobrevive à queda de TODO o backend." $true

# celulas
Txt 62 382 13 $BL "CÉLULAS · isolamento entre inquilinos" $true
$cx = 62
for($i=1; $i -le 3; $i++){
  Box $cx 402 318 78 $BL $BLF 2
  TxtC ($cx+159) 414 16 "#1864AB" ("Célula " + $i) $true
  TxtC ($cx+159) 438 13 $IK "EKS + Aurora" $false
  TxtC ($cx+159) 456 12 $GY "bancas pequenas" $false
  $cx += 330
}
Box 1050 402 328 78 $RD $RDF 2.5
TxtC 1214 412 16 $RD "CÉLULA DEDICADA" $true
TxtC 1214 436 13 $IK "certame acima de 100 mil candidatos" $false
TxtC 1214 454 12 $GY "alocada por evento, devolvida depois" $false

# assincrono + agenda
Box 62 500 650 92 $RL $LT 2
Txt 84 512 16 $IK "SQS + LAMBDA · GERAÇÃO ASSÍNCRONA" $true
Txt 84 538 14 $GY "boleto · cartão · PDF · e-mail" $false
Txt 84 560 14 $IK "nunca no request: era a D3, e derrubava a CPU no pico" $false

Box 728 500 650 92 $OR $ORF 2.5
Txt 750 512 16 $OR "SERVIÇO DE AGENDA" $true
Txt 750 538 14 $GY "lê o cronograma do edital" $false
Txt 750 560 14 $IK "provisiona 6 h antes do pico, devolve 6 h depois" $true

# modo degradado
Box 62 612 1316 78 $RL "#FFF9DB" 2
Txt 84 622 15 "#E8590C" "MODO DEGRADADO · automático, em 5 níveis" $true
Txt 84 646 13 $IK "1 fila de espera  →  2 desliga painel da banca  →  3 só escrita essencial  →  4 SÓ ESTÁTICO  →  5 página de status" $false
Txt 84 666 13 $GY "No nível 4, resultado e gabarito continuam publicados mesmo com a região inteira fora do ar." $false

Txt 62 710 15 $OR "Ensaio de carga a 40× a média DO CERTAME (16,7 mil inscrições/h num certame de 300 mil): relatório assinado, anexo ao contrato." $true
Txt 62 734 14 $GY "Congelamento de implantação enquanto houver janela crítica aberta em qualquer certame ativo." $false

# ── fora da AWS: backup e status ────────────────────────────────────────
Arr 400 782 400 810 $VI 2
Arr 1040 782 1040 810 $VI 2

Box 62 812 660 84 $VI $VIF 2.5
Txt 84 824 16 "#5F3DC4" "GOOGLE CLOUD, SÃO PAULO · backup e restauração a frio" $true
Txt 84 850 14 $IK "backup imutável · se a AWS cair, o sistema volta aqui em ~8 h" $false
Txt 84 872 13 $GY "nada fica ligado: a restauração é ensaiada todo trimestre" $false

Box 738 812 640 84 $VI $VIF 2.5
Txt 760 824 16 "#5F3DC4" "PÁGINA DE STATUS · provedor separado" $true
Txt 760 850 14 $IK "precisa sobreviver à queda que está anunciando" $false
Txt 760 872 13 $GY "é o canal formal para comunicar prorrogação de prazo" $false

# ── rodape ──────────────────────────────────────────────────────────────
Box 40 916 1360 68 $RL $LT 2
Txt 62 926 15 $IK "FORA DA AWS: DNS, WAF, página de status, backup e restauração a frio. R$ 170 mil/ano, 5% do EBITDA." $true
Txt 62 950 14 $GY "A operação NÃO sai da AWS: uma segunda nuvem ligada, rodando o mesmo software mal dimensionado, cai igual." $false

Txt 40 992 12 $GY "Roxo = fora da nuvem primária · Laranja = AWS · Verde = estático, sobrevive ao backend · Azul = célula · Vermelho = célula dedicada" $false

$dir = Split-Path -Parent $MyInvocation.MyCommand.Path
$png = Join-Path $dir "arquitetura-alvo.png"
$bmp.Save($png, [System.Drawing.Imaging.ImageFormat]::Png)
$g.Dispose(); $bmp.Dispose()
Write-Output ("PNG  {0}  ({1:N0} KB)" -f $png, ((Get-Item $png).Length/1KB))

