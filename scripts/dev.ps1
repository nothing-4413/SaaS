<#
.SYNOPSIS
    One-command local stack for machines where Docker Desktop/WSL is unavailable:
    portable PostgreSQL, migrations, then the API.

.DESCRIPTION
    Wraps scripts/local-postgres.ps1 (see there for -PgRoot/-DataDir) and starts
    cmd/api with the matching DATABASE_URL/AUTH_TOKEN_SECRET. `-Action check`
    prepares the database and prints the API command without starting anything;
    `-Action down` stops PostgreSQL.

    AUTH_TOKEN_SECRET defaults to an obvious development value - override it with
    -AuthTokenSecret when the API is reachable from anything but localhost.

.EXAMPLE
    powershell -File scripts\dev.ps1 -PgRoot D:\pgtmp\pgsql -DataDir D:\pgdata
    powershell -File scripts\dev.ps1 -PgRoot D:\pgtmp\pgsql -DataDir D:\pgdata -Action check
    powershell -File scripts\dev.ps1 -PgRoot D:\pgtmp\pgsql -DataDir D:\pgdata -Action down
#>
[CmdletBinding()]
param(
    [ValidateSet('up', 'check', 'down')]
    [string]$Action = 'up',
    [string]$PgRoot = $env:PGROOT,
    [string]$DataDir = $env:PGDATA,
    [int]$Port = 5432,
    [string]$Database = 'saas',
    [string]$SuperUser = 'postgres',
    [string]$HttpAddr = ':8080',
    [string]$AuthTokenSecret = 'local-development-secret-change-me-0123456789',
    [switch]$WithWorker
)

$ErrorActionPreference = 'Stop'
$pg = Join-Path $PSScriptRoot 'local-postgres.ps1'
if (-not (Test-Path $pg)) { throw "missing $pg" }

function Invoke-Pg([string]$PgAction) {
    & $pg -PgRoot $PgRoot -DataDir $DataDir -Port $Port -Database $Database -SuperUser $SuperUser -Action $PgAction
}
if ($Action -eq 'down') {
    Invoke-Pg 'stop'
    exit $LASTEXITCODE
}

Invoke-Pg 'start'
Invoke-Pg 'migrate'

$env:DATABASE_URL = "postgres://$SuperUser@localhost:$Port/${Database}?sslmode=disable"
$env:AUTH_TOKEN_SECRET = $AuthTokenSecret
$env:HTTP_ADDR = $HttpAddr
$env:APP_ENV = 'development'
$port = if ($HttpAddr -like ':*') { $HttpAddr.Substring(1) } else { '8080' }
$env:APP_PUBLIC_URL = "http://localhost:$port"

if ($Action -eq 'check') {
    Write-Host ''
    Write-Host 'database ready; start the API yourself with:'
    Write-Host "  DATABASE_URL='$env:DATABASE_URL'"
    Write-Host "  AUTH_TOKEN_SECRET='<same value>'"
    Write-Host '  go run ./cmd/api'
    exit 0
}

if ($WithWorker) {
    # The worker sends notifications over SMTP; without a mail server it logs
    # delivery failures and retries, which is expected on a local machine.
    $worker = Start-Process -FilePath 'go' -ArgumentList @('run', './cmd/worker') -PassThru -NoNewWindow
    Write-Host "worker running in the background (pid $($worker.Id))"
}

Write-Host "api starting on $HttpAddr (Ctrl+C to stop; PostgreSQL keeps running until -Action down)"
& go run ./cmd/api
