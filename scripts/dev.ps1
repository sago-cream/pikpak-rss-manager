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
Write-Host "開啟 http://$Listen，首次使用請在網頁完成初始化。"
& go @goArguments
exit $LASTEXITCODE
