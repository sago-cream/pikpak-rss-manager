param([string]$Listen = '127.0.0.1:8080')
$ErrorActionPreference = 'Stop'
Set-Location (Split-Path $PSScriptRoot -Parent)
$env:APP_LISTEN = $Listen
if (-not $env:APP_ADMIN_PASSWORD -and -not $env:APP_ADMIN_PASSWORD_FILE) {
    # The program also reads an existing .env. Prompt only if that file does not
    # already declare the administrator password, without printing its values.
    $hasPassword = (Test-Path '.env') -and [bool](Select-String -Path '.env' -Pattern '^\s*APP_ADMIN_PASSWORD(?:_FILE)?\s*=\s*[^\s]' -Quiet)
    if (-not $hasPassword) {
        $privatePassword = Read-Host '管理密碼（至少 12 字元）' -AsSecureString
        $env:APP_ADMIN_PASSWORD = [System.Net.NetworkCredential]::new('', $privatePassword).Password
    }
}
go run ./cmd/pikpak-rss-manager
exit $LASTEXITCODE
