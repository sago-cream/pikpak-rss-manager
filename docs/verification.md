# Verification

## Manual tasks without a name field (2026-10-04)

Passed locally: Go 1.27.1 `go test ./...`, `go vet ./...`, application build with `-buildvcs=false`, JavaScript syntax/localization checks and `git diff --check`.

Manual tasks accept an omitted name. Display names use Magnet `dn`, resolved torrent metadata, or the final HTTP URL path segment (host for a root URL); missing, oversized or unsafe names fall back to the resource key. Names persist in the job/rule snapshot without enabling renaming. Explicit names from existing API clients remain supported and validated. Tests cover unnamed authenticated/CSRF-protected requests, v1/v2 resources, Unicode filenames, query exclusion, invalid names, deduplication and original filename preservation.

Headless Edge with the isolated mock account in `.local/unnamed-tasks-ui-data` passed Magnet, HTTP and torrent task creation without `name` or `source_type` fields, generated names in the task list, duplicate rejection, Chinese/English dialogs and switching back to subscriptions with their name field still required. Desktop and 390×844 mobile layouts have no horizontal overflow or browser errors. Visually inspected fixture screenshots: [desktop](screenshots/unnamed-task-desktop.png), [mobile](screenshots/unnamed-task-mobile.png). No real PikPak operation was performed.

## Login and setup language-selector layout (2026-10-04)

Passed locally: Go 1.27.1 `go test ./...`, `go vet ./...`, application build with `-buildvcs=false`, localization checks and `git diff --check`.

Headless Edge used an isolated mock account (`.local/auth-header-ui-data`) and a separate first-run data directory (`.local/auth-header-setup-data`). Traditional Chinese and English login/setup layouts passed at 1280×900, 390×844 and 320×568: the selector shares a row with the logo and aligns with the form's right edge, with no overlap or horizontal overflow. Persisted language switching, setup, login, translated wrong-password errors and logout passed without browser errors. Visually inspected fixture screenshots: [desktop](screenshots/login-language-desktop.png), [mobile](screenshots/login-language-mobile.png). No real PAT or PikPak operation was used.

## Automatic manual-task source detection (2026-10-04)

Passed locally: Go 1.27.1 `go test ./...`, `go vet ./...`, application build with `-buildvcs=false`, JavaScript syntax/localization checks and `git diff --check`. VCS stamping was disabled because the elevated build user differs from the checkout owner.

Regression tests cover automatic v1/v2 Magnet normalization, `.torrent` paths with mixed case and query strings, direct HTTP/HTTPS URLs, legacy explicit types, invalid sources, private-network metadata policy and deduplication across automatic/legacy requests. General HTTP/HTTPS links are queued without fetching their content; extensionless torrent URLs use the direct URL path unless an API client explicitly requests torrent resolution.

Headless Edge with an isolated mocked account and `.local/auto-links-ui-data` passed creating Magnet, HTTP and torrent tasks without a type selector or `source_type` request field, duplicate rejection, and a 390×844 viewport without horizontal overflow or browser errors. No real PikPak operation was performed.

## Current change (2026-10-04)

Passed locally: Go 1.27.1 `go test ./...`, `go vet ./...`, CGO-free build, JavaScript syntax checks, localization checks and `git diff --check`. The executable reports v0.5.0.

Headless Edge used an isolated mocked account and `.local/v050-ui-data-final`, without reading a real PAT or calling PikPak. Passed: English login, persisted language switching, unchanged Chinese subscription names, task creation with a selected folder, duplicate-source error, task details, 72 torrent samples and live filename warnings. At 1280×900 and 390×844, the regex textarea cannot resize, the form scrolls inside the rounded dialog, and there is no horizontal overflow or browser error. Fixture screenshots: [desktop](screenshots/v050-desktop.png), [mobile](screenshots/v050-mobile.png).

An additional isolated first-run browser test passed English setup, automatic login, logout, translated wrong-password errors and switching back to Traditional Chinese.

Functional commit `7893767` passed GitHub Actions:

| Validation | Evidence |
|---|---|
| Linux race/vet/build, localization, Docker startup/persistence | [CI](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37147066885) |
| main/latest amd64/arm64 publishing and manifest | [Main publishing](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37147066868) |
| v0.5.0 amd64/arm64 publishing and manifest | [Tag publishing](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37147207385) |
| Anonymous latest pulls and container persistence on both architectures | [Latest smoke](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37147258471) |
| Anonymous v0.5.0 pulls and container persistence on both architectures | [Release smoke](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37147403033) |

This development machine has no Docker; container checks ran in Actions. Real PikPak operations have not been rerun, and no VPS was deployed. Mocked tests do not prove real PikPak operations.

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
