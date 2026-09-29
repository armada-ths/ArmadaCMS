[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$supabaseVersion = "2.118.0"
$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path

Push-Location $repoRoot
try {
    & docker compose -f docker-compose.dev.yml down --remove-orphans
    if ($LASTEXITCODE -ne 0) {
        throw "Docker Compose failed with exit code $LASTEXITCODE."
    }

    & pnpx --yes "supabase@$supabaseVersion" stop
    if ($LASTEXITCODE -ne 0) {
        throw "Supabase CLI failed to stop the local services."
    }
}
finally {
    Pop-Location
}
