# Sobe o PostgreSQL local da POC (cria o cluster na primeira vez).
# Só escuta em localhost, autenticação trust: não há senha a guardar.
$ErrorActionPreference = 'Stop'
$bin  = if ($env:PG_BIN)  { $env:PG_BIN }  else { 'C:\Users\rockb\tools\pgsql\bin' }
$data = if ($env:PG_DATA) { $env:PG_DATA } else { 'C:\Users\rockb\tools\pgdata-lisura' }
$port = 5432

if (-not (Test-Path "$data\PG_VERSION")) {
    & "$bin\initdb.exe" -D $data -U postgres -A trust -E UTF8 --locale=C | Out-Host
    Add-Content "$data\postgresql.conf" "`nlisten_addresses = 'localhost'`nport = $port`nmax_connections = 200`n"
}

& "$bin\pg_ctl.exe" -D $data status *> $null
if ($LASTEXITCODE -ne 0) {
    # O postgres herda os handles do console: pipe ou Start-Process -Wait (que espera os
    # descendentes) travariam aqui. Espera só o pg_ctl.
    $p = Start-Process -FilePath "$bin\pg_ctl.exe" -ArgumentList '-D', "`"$data`"", '-l', "`"$data\server.log`"", '-w', 'start' -NoNewWindow -PassThru
    $p.WaitForExit()
}

foreach ($db in 'lisura', 'lisura_test') {
    $exists = & "$bin\psql.exe" -h localhost -p $port -U postgres -tAc "select 1 from pg_database where datname = '$db'"
    if ($exists -ne '1') { & "$bin\createdb.exe" -h localhost -p $port -U postgres $db }
}
"PostgreSQL pronto em localhost:$port (bancos lisura e lisura_test)"
