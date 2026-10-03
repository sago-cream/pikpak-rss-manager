# Verification

## Current change (2026-10-04)

Passed locally: Go 1.27.1 `go test ./...`, `go vet ./...`, CGO-free build, JavaScript syntax checks, localization checks and `git diff --check`. The executable reports v0.5.0.

Headless Edge used an isolated mocked account and `.local/v050-ui-data-final`, without reading a real PAT or calling PikPak. Passed: English login, persisted language switching, unchanged Chinese subscription names, task creation with a selected folder, duplicate-source error, task details, 72 torrent samples and live filename warnings. At 1280×900 and 390×844, the regex textarea cannot resize, the form scrolls inside the rounded dialog, and there is no horizontal overflow or browser error. Fixture screenshots: [desktop](screenshots/v050-desktop.png), [mobile](screenshots/v050-mobile.png).

Docker and Linux race validation remain for GitHub Actions; this development machine has no Docker. No v0.5.0 image has been published in this change, and real PikPak operations have not been rerun. Mocked tests do not prove real PikPak operations.

## Reproduce

```sh
go test ./...
go vet ./...
go build ./cmd/pikpak-rss-manager
node --check internal/web/static/app.js
node --check internal/web/static/i18n.js
node scripts/test-i18n.cjs
```

Tests cover RSS/Atom and v1/v2 torrents, network/metadata limits, preview provenance, per-account deduplication, baselines, legacy naming, independent file rules, staging, uncertain submissions, restart/partial action recovery, authorization/quota/rate failures, folder paging/collisions/account isolation, authentication and CSRF.

Manual-task regressions cover authentication/CSRF, normalized infohash deduplication shared with feed jobs, direct URLs, private-network torrent policy, stale destinations/root selections, account switches, encrypted persistence, original filenames and restart recovery with one submission. Localization checks cover templates, message parameters and preservation of user content.

## Prior external validation

Real PikPak tests on 2026-10-03 used only `_pikpak-rss-manager-test/run-20261002T201936-d8752c` and a public-domain Alice fixture (163,783 bytes). MCP authorization, folder creation, HTTP offline download, rename/move and local deduplication passed. The Magnet did not finish within 90 seconds; torrent availability/instant download was not proven. Existing account content was preserved, with no permanent deletion. A separate folder-only run passed paging/selection/creation and persistence without downloads.

Opt-in tests require an administrator/PAT already configured in a local data directory. No PAT is sent to CI:

```powershell
$env:PIKPAK_LIVE_FOLDERS_TEST='1'
go test ./tests/integration -run TestLiveFolderBrowsingAndCreation -count=1 -v
Remove-Item Env:PIKPAK_LIVE_FOLDERS_TEST
```

For download/rename/move tests use PIKPAK_LIVE_TEST=1 and TestLivePikPak; these create small tasks inside a new isolated test namespace. Optional data override: PIKPAK_LIVE_DATA_DIR.

Previously published v0.4.3 images passed Linux race/vet/container persistence, both architectures and anonymous pulls:

- [CI](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37144695003)
- [Tag publishing](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37144695286)
- [Anonymous amd64/arm64 smoke](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37144895903)

These results apply to v0.4.3, not to unpublished changes or a VPS deployment. OAuth research did not include client registration, consent or refresh-token testing.
