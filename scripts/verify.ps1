$ErrorActionPreference = 'Stop'
Set-Location (Split-Path $PSScriptRoot -Parent)
$unformatted = gofmt -l cmd internal tests
if ($unformatted) { throw 'Go source must be formatted with gofmt.' }
go test ./...
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
go vet ./...
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
go build -trimpath -o .local/pikpak-rss-manager.exe ./cmd/pikpak-rss-manager
exit $LASTEXITCODE
