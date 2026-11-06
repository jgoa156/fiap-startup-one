param([string]$Api = 'http://localhost:8080/api',[Parameter(Mandatory)][long]$Certame)
# Percorre o fluxo do candidato contra a API rodando: conta, inscrição, laudo, finalização.
$ErrorActionPreference = 'Stop'

function Chamar($Metodo, $Caminho, $Corpo, $Token) {
    $h = @{}
    if ($Token) { $h.Authorization = "Bearer $Token" }
    $json = if ($null -ne $Corpo) { $Corpo | ConvertTo-Json -Compress } else { $null }
    Invoke-RestMethod -Method $Metodo -Uri "$Api$Caminho" -Headers $h -ContentType 'application/json; charset=utf-8' -Body $json
}

# CPF sintético válido a partir de uma base aleatória.
$base = '{0:D9}' -f (Get-Random -Minimum 1 -Maximum 999999999)
$d = [int[]]($base.ToCharArray() | ForEach-Object { [string]$_ })
foreach ($n in 10, 11) { $s = 0; for ($i = 0; $i -lt $n - 1; $i++) { $s += $d[$i] * ($n - $i) }; $d += ($s * 10 % 11) % 10 }
$cpf = -join $d

$pub = Chamar GET "/certames/$Certame"
"Certame: $($pub.nome) · $($pub.cargos.Count) cargos · edital v$($pub.edital_versao)"

$conta = Chamar POST '/contas' @{ cpf = $cpf; nome = 'Candidata Demo'; email = 'demo@exemplo.test' }
$t = $conta.token
"Conta criada: CPF $cpf (id $($conta.id))"
Chamar PUT '/conta' @{ nome = 'Candidata Demo'; email = 'demo@exemplo.test'; data_nascimento = '1994-07-20'; municipio = 'Campinas'; uf = 'SP' } $t | Out-Null

$ins = (Chamar POST "/certames/$Certame/inscricoes" @{ cargo_id = $pub.cargos[0].id } $t).id
"Inscrição $ins aberta em RASCUNHO"
Chamar PUT "/inscricoes/$ins/cidade" @{ cidade_id = $pub.cidades_prova[0].id } $t | Out-Null
Chamar PUT "/inscricoes/$ins/cotas" @{ cotas = @('pcd') } $t | Out-Null

try { Chamar POST "/inscricoes/$ins/finalizar" $null $t | Out-Null }
catch { "Finalizar antes da hora: $($_.ErrorDetails.Message)" }

$pdf = Join-Path $env:TEMP 'laudo-demo.pdf'
[IO.File]::WriteAllText($pdf, "%PDF-1.4`n1 0 obj << /Type /Catalog >> endobj`ntrailer << >>`n%%EOF`n")
$up = curl.exe -s -X POST "$Api/conta/documentos" -H "Authorization: Bearer $t" -F tipo=laudo -F "arquivo=@$pdf;type=application/pdf" | ConvertFrom-Json
"Laudo enviado para a carteira: documento $($up.id)"
Chamar PUT "/inscricoes/$ins/atendimento" @{ necessidades = @('ledor'); laudo_documento_id = $up.id } $t | Out-Null
Chamar PUT "/inscricoes/$ins/declaracoes" @{ aceito = $true } $t | Out-Null

$fim = Chamar POST "/inscricoes/$ins/finalizar" $null $t
"Finalizada: $($fim.estado)"
$fim2 = Chamar POST "/inscricoes/$ins/finalizar" $null $t
"Segundo clique: $($fim2.estado), reenvio=$($fim2.reenvio)"
