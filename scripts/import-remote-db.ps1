[CmdletBinding()]
param(
    [string]$EnvFile = ".env",
    [switch]$KeepDump
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Read-DotEnv {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Path
    )

    if (-not (Test-Path $Path)) {
        throw "Env file not found: $Path"
    }

    $values = @{}

    foreach ($line in Get-Content -Path $Path) {
        $trimmed = $line.Trim()

        if ([string]::IsNullOrWhiteSpace($trimmed) -or $trimmed.StartsWith("#")) {
            continue
        }

        $parts = $trimmed -split '=', 2
        if ($parts.Count -ne 2) {
            continue
        }

        $key = $parts[0].Trim()
        $value = $parts[1]

        if (($value.StartsWith('"') -and $value.EndsWith('"')) -or ($value.StartsWith("'") -and $value.EndsWith("'"))) {
            $value = $value.Substring(1, $value.Length - 2)
        }

        $values[$key] = $value
    }

    return $values
}

function Get-RequiredValue {
    param(
        [hashtable]$Config,
        [string]$Key
    )

    if (-not $Config.ContainsKey($Key) -or [string]::IsNullOrWhiteSpace($Config[$Key])) {
        throw "Missing required value '$Key' in env file."
    }

    return $Config[$Key]
}

function Invoke-Docker {
    param(
        [string[]]$Arguments
    )

    & docker @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "Docker command failed: docker $($Arguments -join ' ')"
    }
}

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$envPath = if ([System.IO.Path]::IsPathRooted($EnvFile)) { $EnvFile } else { Join-Path $repoRoot $EnvFile }
$config = Read-DotEnv -Path $envPath

$sourceHost = Get-RequiredValue -Config $config -Key "SOURCE_DB_HOST"
$sourcePort = if ($config.ContainsKey("SOURCE_DB_PORT") -and -not [string]::IsNullOrWhiteSpace($config["SOURCE_DB_PORT"])) { $config["SOURCE_DB_PORT"] } else { "5432" }
$sourceUser = Get-RequiredValue -Config $config -Key "SOURCE_DB_USER"
$sourcePassword = Get-RequiredValue -Config $config -Key "SOURCE_DB_PASSWORD"
$sourceName = Get-RequiredValue -Config $config -Key "SOURCE_DB_NAME"
$sourceSslMode = if ($config.ContainsKey("SOURCE_DB_SSLMODE") -and -not [string]::IsNullOrWhiteSpace($config["SOURCE_DB_SSLMODE"])) { $config["SOURCE_DB_SSLMODE"] } else { "require" }
$sourceToolsImage = if ($config.ContainsKey("SOURCE_DB_TOOLS_IMAGE") -and -not [string]::IsNullOrWhiteSpace($config["SOURCE_DB_TOOLS_IMAGE"])) { $config["SOURCE_DB_TOOLS_IMAGE"] } else { "postgres:17" }

$localContainer = if ($config.ContainsKey("LOCAL_DB_CONTAINER") -and -not [string]::IsNullOrWhiteSpace($config["LOCAL_DB_CONTAINER"])) { $config["LOCAL_DB_CONTAINER"] } else { "armadacms-postgres" }
$localPort = if ($config.ContainsKey("DB_PORT") -and -not [string]::IsNullOrWhiteSpace($config["DB_PORT"])) { $config["DB_PORT"] } else { "5432" }
$localUser = Get-RequiredValue -Config $config -Key "DB_USER"
$localPassword = Get-RequiredValue -Config $config -Key "DB_PASSWORD"
$localName = Get-RequiredValue -Config $config -Key "DB_NAME"

$runningState = (& docker inspect -f "{{.State.Running}}" $localContainer 2>$null)
if ($LASTEXITCODE -ne 0 -or $runningState.Trim() -ne "true") {
    throw "Local Postgres container '$localContainer' is not running. Start it first with: docker compose -f docker-compose.db.yml up -d"
}

$tempDir = Join-Path ([System.IO.Path]::GetTempPath()) ("armadacms-db-clone-" + [guid]::NewGuid().ToString("N"))
$null = New-Item -ItemType Directory -Path $tempDir -Force
$dumpFileName = "remote-clone.sql"
$dumpFilePath = Join-Path $tempDir $dumpFileName
$containerDumpPath = "/tmp/$dumpFileName"

Write-Host "Dumping remote database..." -ForegroundColor Cyan
$dumpArgs = @(
    'run',
    '--rm',
    '-e', "PGPASSWORD=$sourcePassword",
    '-e', "PGSSLMODE=$sourceSslMode",
    '-v', ($tempDir + ':/dump'),
    $sourceToolsImage,
    'pg_dump',
    '--host', $sourceHost,
    '--port', $sourcePort,
    '--username', $sourceUser,
    '--dbname', $sourceName,
    '--encoding', 'UTF8',
    '--clean',
    '--if-exists',
    '--no-owner',
    '--no-privileges',
    '--file', "/dump/$dumpFileName"
)
Invoke-Docker -Arguments $dumpArgs

Write-Host "Copying dump into local Postgres container..." -ForegroundColor Cyan
Invoke-Docker -Arguments @('cp', $dumpFilePath, ($localContainer + ':' + $containerDumpPath))

Write-Host "Resetting local database '$localName'..." -ForegroundColor Cyan
Invoke-Docker -Arguments @(
    'exec',
    '-e', "PGPASSWORD=$localPassword",
    $localContainer,
    'psql',
    '--host', 'localhost',
    '--port', $localPort,
    '--username', $localUser,
    '--dbname', 'postgres',
    '-v', 'ON_ERROR_STOP=1',
    '-c', "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '$localName' AND pid <> pg_backend_pid();"
)
Invoke-Docker -Arguments @(
    'exec',
    '-e', "PGPASSWORD=$localPassword",
    $localContainer,
    'psql',
    '--host', 'localhost',
    '--port', $localPort,
    '--username', $localUser,
    '--dbname', 'postgres',
    '-v', 'ON_ERROR_STOP=1',
    '-c', ('DROP DATABASE IF EXISTS "{0}";' -f $localName)
)
Invoke-Docker -Arguments @(
    'exec',
    '-e', "PGPASSWORD=$localPassword",
    $localContainer,
    'psql',
    '--host', 'localhost',
    '--port', $localPort,
    '--username', $localUser,
    '--dbname', 'postgres',
    '-v', 'ON_ERROR_STOP=1',
    '-c', ('CREATE DATABASE "{0}";' -f $localName)
)

Write-Host "Importing dump into local database '$localName'..." -ForegroundColor Cyan
Invoke-Docker -Arguments @(
    'exec',
    '-e', "PGPASSWORD=$localPassword",
    $localContainer,
    'psql',
    '--host', 'localhost',
    '--port', $localPort,
    '--username', $localUser,
    '--dbname', $localName,
    '-v', 'ON_ERROR_STOP=1',
    '-f', $containerDumpPath
)

Invoke-Docker -Arguments @('exec', $localContainer, 'rm', '-f', $containerDumpPath)

if (-not $KeepDump) {
    Remove-Item -Path $tempDir -Recurse -Force
    Write-Host "Temporary dump removed." -ForegroundColor DarkGray
}
else {
    Write-Host "Temporary dump kept at: $dumpFilePath" -ForegroundColor Yellow
}

Write-Host "Remote clone completed successfully." -ForegroundColor Green
Write-Host "You can now start the backend against your local database." -ForegroundColor Green
