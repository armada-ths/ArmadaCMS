[CmdletBinding()]
param(
    [switch]$Build
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$supabaseVersion = "2.118.0"
$excludedServices = @(
    "gotrue",
    "realtime",
    "imgproxy",
    "mailpit",
    "postgrest",
    "postgres-meta",
    "studio",
    "edge-runtime",
    "logflare",
    "vector",
    "supavisor"
)

function Invoke-Supabase {
    param([string[]]$Arguments)

    $null = & pnpx --yes "supabase@$supabaseVersion" @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "Supabase CLI failed: supabase $($Arguments -join ' ')"
    }
}

function Get-StatusValue {
    param(
        [string[]]$StatusLines,
        [string[]]$Names
    )

    foreach ($name in $Names) {
        $line = $StatusLines | Where-Object { $_ -match "^$([regex]::Escape($name))=" } | Select-Object -First 1
        if ($null -ne $line) {
            return ($line -split '=', 2)[1].Trim().Trim('"')
        }
    }

    throw "Supabase status did not contain any of: $($Names -join ', ')"
}

if (-not (Get-Command pnpx -ErrorAction SilentlyContinue)) {
    throw "pnpx is required. Install pnpm before starting the development environment."
}
if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    throw "Docker is required. Start Docker Desktop before starting the development environment."
}

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
Push-Location $repoRoot
try {
    Invoke-Supabase -Arguments @("start", "--exclude", ($excludedServices -join ','))

    $statusLines = & pnpx --yes "supabase@$supabaseVersion" status -o env
    if ($LASTEXITCODE -ne 0) {
        throw "Unable to read local Supabase status."
    }

    $env:LOCAL_SUPABASE_S3_ACCESS_KEY = Get-StatusValue -StatusLines $statusLines -Names @("S3_PROTOCOL_ACCESS_KEY_ID")
    $env:LOCAL_SUPABASE_S3_SECRET_KEY = Get-StatusValue -StatusLines $statusLines -Names @("S3_PROTOCOL_ACCESS_KEY_SECRET")
    $env:LOCAL_SUPABASE_S3_REGION = Get-StatusValue -StatusLines $statusLines -Names @("S3_PROTOCOL_REGION")

    $composeArguments = @("compose", "-f", "docker-compose.dev.yml", "up", "--remove-orphans")
    if ($Build) {
        $composeArguments += "--build"
    }

    & docker @composeArguments
    if ($LASTEXITCODE -ne 0) {
        throw "Docker Compose failed with exit code $LASTEXITCODE."
    }
}
finally {
    try {
        & (Join-Path $PSScriptRoot "dev-down.ps1")
        if ($LASTEXITCODE -ne 0) {
            Write-Warning "Development environment cleanup exited with code $LASTEXITCODE."
        }
    }
    catch {
        Write-Warning "Development environment cleanup failed: $($_.Exception.Message)"
    }

    Remove-Item Env:LOCAL_SUPABASE_S3_ACCESS_KEY -ErrorAction SilentlyContinue
    Remove-Item Env:LOCAL_SUPABASE_S3_SECRET_KEY -ErrorAction SilentlyContinue
    Remove-Item Env:LOCAL_SUPABASE_S3_REGION -ErrorAction SilentlyContinue
    Pop-Location
}
