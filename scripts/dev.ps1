param(
    [string]$Listen = '127.0.0.1:8080',
    [switch]$Version
)
$ErrorActionPreference = 'Stop'
Set-Location (Split-Path $PSScriptRoot -Parent)

$goArguments = @('run', './cmd/pikpak-rss-manager')
if ($Version) {
    & go @goArguments version
    exit $LASTEXITCODE
}

$env:APP_LISTEN = $Listen
if (-not $env:APP_ADMIN_PASSWORD -and -not $env:APP_ADMIN_PASSWORD_FILE) {
    # The program also reads an existing .env. Prompt only if that file does not
    # already declare the administrator password, without printing its values.
    $hasPassword = (Test-Path '.env') -and [bool](Select-String -Path '.env' -Pattern '^\s*APP_ADMIN_PASSWORD(?:_FILE)?\s*=\s*[^\s]' -Quiet)
    if (-not $hasPassword) {
        do {
            $privatePassword = Read-Host '管理密碼' -AsSecureString
            $env:APP_ADMIN_PASSWORD = [System.Net.NetworkCredential]::new('', $privatePassword).Password
            $privatePassword.Dispose()
            if (-not $env:APP_ADMIN_PASSWORD) { Write-Host '請輸入管理密碼。' }
        } while (-not $env:APP_ADMIN_PASSWORD)
    }
}
& go @goArguments
exit $LASTEXITCODE
