[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$terraformRoot = Join-Path $repoRoot 'infra/terraform/gcp/local'
$localConfigJSON = & terraform "-chdir=$terraformRoot" output -json local_recaptcha_config
if ($LASTEXITCODE -ne 0) { throw 'Could not read the local reCAPTCHA Terraform output.' }
$localConfig = ($localConfigJSON -join "`n") | ConvertFrom-Json
if (-not $localConfig.site_key -or -not $localConfig.api_key -or -not $localConfig.project_id) {
    throw 'Apply the local reCAPTCHA resources before configuring local development.'
}

function Set-LocalEnvironmentValues {
    param([string]$Path, [hashtable]$Values)
    $content = if (Test-Path -LiteralPath $Path) { [IO.File]::ReadAllText($Path) } else { '' }
    foreach ($key in $Values.Keys) {
        $line = "$key=$($Values[$key])"
        $pattern = "(?m)^$([regex]::Escape($key))=.*$"
        if ([regex]::IsMatch($content, $pattern)) {
            $content = [regex]::Replace($content, $pattern, [System.Text.RegularExpressions.MatchEvaluator]{ param($match) $line })
        }
        else {
            $content = $content.TrimEnd() + "`n$line`n"
        }
    }
    [IO.File]::WriteAllText($Path, $content, [Text.UTF8Encoding]::new($false))
}

Set-LocalEnvironmentValues -Path (Join-Path $repoRoot '.env') -Values @{
    LOCAL_PHOTO_RECAPTCHA_PROJECT_ID = $localConfig.project_id
    LOCAL_PHOTO_RECAPTCHA_SITE_KEY = $localConfig.site_key
    LOCAL_PHOTO_RECAPTCHA_API_KEY = $localConfig.api_key
}
Set-LocalEnvironmentValues -Path (Join-Path $repoRoot '../armada.nu/apps/photos/.env.local') -Values @{
    NEXT_PUBLIC_LOCAL_RECAPTCHA_SITE_KEY = $localConfig.site_key
}
Write-Host 'Configured the local backend and photo app with the dedicated Terraform test key.'
