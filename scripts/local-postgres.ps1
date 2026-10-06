<#
.SYNOPSIS
    Start, stop or inspect a portable PostgreSQL instance for local development
    without Docker (WSL/Docker Desktop is unusable on some Windows machines).

.DESCRIPTION
    Expects an extracted PostgreSQL Windows binary package, for example
    https://get.enterprisedb.com/postgresql/postgresql-16.6-1-windows-x64-binaries.zip
    which unpacks to <root>\pgsql\bin\pg_ctl.exe.

    `start` initialises the data directory on first use (trust authentication,
    local connections only), starts the server and creates the database.
    `migrate` applies migrations/*.up.sql in filename order, skipping files whose
    objects already exist, so it is safe to run repeatedly.
    `stop` shuts the server down.

.EXAMPLE
    powershell -File scripts\local-postgres.ps1 -PgRoot D:\pgtmp\pgsql -DataDir D:\pgdata -Action start
    powershell -File scripts\local-postgres.ps1 -PgRoot D:\pgtmp\pgsql -DataDir D:\pgdata -Action migrate
    powershell -File scripts\local-postgres.ps1 -PgRoot D:\pgtmp\pgsql -DataDir D:\pgdata -Action stop
#>
[CmdletBinding()]
param(
    [ValidateSet('start', 'stop', 'status', 'migrate')]
    [string]$Action = 'start',
    [string]$PgRoot = $env:PGROOT,
    [string]$DataDir = $env:PGDATA,
    [int]$Port = 5432,
    [string]$Database = 'saas',
    [string]$SuperUser = 'postgres',
    [string]$MigrationDir
)

$ErrorActionPreference = 'Stop'
# Keep NOTICE output out of the strings this script parses.
$env:PGOPTIONS = '-c client_min_messages=warning'
if (-not $PgRoot) { throw 'Provide -PgRoot (or set PGROOT), e.g. -PgRoot D:\pgtmp\pgsql' }
if (-not $DataDir) { throw 'Provide -DataDir (or set PGDATA), e.g. -DataDir D:\pgdata' }
if (-not $MigrationDir) { $MigrationDir = Join-Path $PSScriptRoot '..\migrations' }

$bin = Join-Path $PgRoot 'bin'
$pgCtl = Join-Path $bin 'pg_ctl.exe'
$pgIsReady = Join-Path $bin 'pg_isready.exe'
$psql = Join-Path $bin 'psql.exe'
$initdb = Join-Path $bin 'initdb.exe'
foreach ($tool in @($pgCtl, $pgIsReady, $psql, $initdb)) {
    if (-not (Test-Path $tool)) { throw "Missing $tool - point -PgRoot at an extracted PostgreSQL package" }
}

function Test-Ready {
    & $pgIsReady -h 127.0.0.1 -p $Port *> $null
    return ($LASTEXITCODE -eq 0)
}

function Wait-Ready([int]$Seconds = 60) {
    $deadline = (Get-Date).AddSeconds($Seconds)
    while ((Get-Date) -lt $deadline) {
        if (Test-Ready) { return $true }
        Start-Sleep -Seconds 2
    }
    return $false
}

function Invoke-Psql([string]$Query, [string]$Db) {
    $target = if ($Db) { @('-d', $Db) } else { @() }
    return (& $psql -U $SuperUser -h 127.0.0.1 -p $Port @target -tAc $Query 2>&1) -join ''
}

switch ($Action) {
    'status' {
        if (Test-Ready) { "postgres on 127.0.0.1:$Port is accepting connections" } else { "postgres on 127.0.0.1:$Port is NOT running" }
    }
    'start' {
        if (-not (Test-Path (Join-Path $DataDir 'PG_VERSION'))) {
            Write-Host "initialising $DataDir"
            & $initdb -D $DataDir -U $SuperUser -A trust -E UTF8 --locale=C | Out-Null
        }
        if (Test-Ready) {
            Write-Host "postgres already running on 127.0.0.1:$Port"
        }
        else {
            # Note: pg_ctl -w can block forever in some sandboxed shells, so start
            # without it and poll with pg_isready instead.
            & $pgCtl -D $DataDir -l (Join-Path $DataDir 'server.log') -o "-p $Port" start | Out-Null
            if (-not (Wait-Ready)) { throw "postgres did not become ready; see $(Join-Path $DataDir 'server.log')" }
            Write-Host "postgres started on 127.0.0.1:$Port"
        }
        if ((Invoke-Psql "SELECT 1 FROM pg_database WHERE datname='$Database'") -notmatch '1') {
            & $psql -U $SuperUser -h 127.0.0.1 -p $Port -c "CREATE DATABASE $Database" | Out-Null
            Write-Host "created database $Database"
        }
        Write-Host ''
        Write-Host 'Next: start the API with'
        Write-Host "  `$env:DATABASE_URL='postgres://$SuperUser@localhost:$Port/$Database?sslmode=disable'"
        Write-Host "  `$env:AUTH_TOKEN_SECRET='<32+ character random string>'"
        Write-Host '  go run ./cmd/api'
    }
    'migrate' {
        if (-not (Test-Ready)) { throw 'postgres is not running; run -Action start first' }
        $files = @(Get-ChildItem (Join-Path $MigrationDir '*.up.sql') | Sort-Object Name)
        Invoke-Psql 'CREATE TABLE IF NOT EXISTS schema_migrations (version text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())' $Database | Out-Null

        # A database that was migrated before this script existed has no version
        # rows yet: record the files as applied instead of re-running them.
        if ((Invoke-Psql "SELECT COALESCE(to_regclass('public.organizations')::text,'none')" $Database) -match 'organizations' -and
            (Invoke-Psql 'SELECT count(*) FROM schema_migrations' $Database) -match '^0$') {
            foreach ($file in $files) { Invoke-Psql "INSERT INTO schema_migrations (version) VALUES ('$($file.Name)') ON CONFLICT DO NOTHING" $Database | Out-Null }
            Write-Host 'existing schema detected: recorded all migration files as applied'
        }

        $applied = 0
        foreach ($file in $files) {
            $version = $file.Name
            if ((Invoke-Psql "SELECT 1 FROM schema_migrations WHERE version='$version'" $Database) -match '1') {
                Write-Host "-- $version (already applied)"
                continue
            }
            Write-Host "-- $version"
            & $psql -U $SuperUser -h 127.0.0.1 -p $Port -d $Database -v ON_ERROR_STOP=1 -q -f $file.FullName | Out-Null
            if ($LASTEXITCODE -ne 0) { throw "migration failed: $version" }
            Invoke-Psql "INSERT INTO schema_migrations (version) VALUES ('$version') ON CONFLICT DO NOTHING" $Database | Out-Null
            $applied++
        }
        $tables = Invoke-Psql "SELECT count(*) FROM information_schema.tables WHERE table_schema='public'" $Database
        Write-Host "applied $applied new migration file(s) out of $($files.Count); public tables: $tables"
    }
    'stop' {
        & $pgCtl -D $DataDir stop -m fast | Out-Null
        Write-Host 'postgres stopped'
    }
}
