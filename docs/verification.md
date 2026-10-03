# Verification

## v1.0.0 republication (2026-10-04)

The existing v1.0.0 tag was updated to functional commit `2a66034` after selected backfill/replacement, repeat downloads, dark mode and new naming defaults passed local verification. The published v1.0.0 and latest images both passed anonymous checks on linux/amd64 and linux/arm64.

| Validation | Evidence |
|---|---|
| Linux race/vet/build, localization, non-root container startup/persistence | [CI](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37155134834) |
| main/latest amd64/arm64 publishing and manifest | [Main publishing](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37155134838) |
| Reissued v1.0.0 amd64/arm64 publishing and manifest | [Tag publishing](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37155173674) |
| Anonymous latest pulls and container persistence on both architectures | [Latest smoke](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37155321003) |
| Anonymous v1.0.0 pulls and container persistence on both architectures | [Release smoke](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37155344169) |

The release tag/image were replaced as requested; pull again even when already using v1.0.0. Back up the complete data volume before the schema-3 upgrade. Real PikPak replacement operations were not run, and mocked tests do not establish real cloud behavior. No VPS was deployed. Credentials remained local; staged files and the release tree passed secret-exclusion checks.

## Persistent dark mode (2026-10-04)

Passed locally: `scripts/verify.ps1`, JavaScript theme/backfill syntax and localization checks, `git diff --check`, and the built executable reporting `v1.0.0`. Theme preference applies before styles load, follows the system until explicitly selected, persists across page reloads/language changes, and synchronizes across tabs.

Headless Edge used only isolated mock account and first-run data directories. Both languages and all five management views passed at 1280, 390 and 320 pixels without horizontal overflow or JavaScript errors. System-theme changes, saved light/dark toggles, logout/login, setup, subscription forms, folder browsing and the backfill dialog passed. Visually inspected fixture screenshots: [dark overview](screenshots/v100-dark-overview.png), [mobile login](screenshots/v100-dark-login.png). No real PAT or PikPak operation was used.

## Selected backfill and repeat downloads (2026-10-04)

Passed locally: `go test ./...`, `go vet ./...`, JavaScript syntax/localization checks, `git diff --check`, and a `-buildvcs=false -trimpath` application build reporting `v1.0.0`. The standard verification script also passed with a nonfatal shared module-cache metadata permission warning; the explicit build used the workspace-local cache.

Regression coverage includes version-2 migration retaining encrypted jobs, staging/task IDs and partial file actions; repeat resource downloads; atomic/idempotent selection batches; authentication/CSRF; subscription, account and expiry checks; preview without baseline/queue/cloud mutations; one feed snapshot and distinct torrent read; v2 filenames, metadata-size protection and mismatched provenance; and recovery after an uncertain backup move. Ordinary collisions and same-name folders still require review.

Headless Edge used `.local/v100-backfill-ui-data` with a mocked account only. Both languages passed the 72-filename list, empty selection, cancelled/accepted confirmation, selected-only queue insertion, repeat source downloads, new naming defaults, and 1280/390/320-pixel layouts without overflow or JavaScript errors. The desktop fixture screenshot was visually inspected. No real PikPak operation was performed; container/race/publishing validation runs in Actions after all requested changes are complete.

## New subscription replacement defaults (2026-10-04)

New subscriptions default to `\[(\d+)\]` and `S01E$1`. Existing saved rules, including empty replacements, retain their values. Passed JavaScript syntax and localization checks.

## v1.0.0 release (2026-10-04)

Passed locally: `scripts/verify.ps1`, `git diff --check` and the built executable reporting `v1.0.0`. This release includes automatic manual-task link/name detection, administrator password changes, the shared brand icon and unused-file/code cleanup.

Release commit `d807d2f` passed GitHub Actions:

| Validation | Evidence |
|---|---|
| Linux race/vet/build, localization, non-root Docker startup/persistence | [CI](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37153145948) |
| main/latest amd64/arm64 publishing and manifest | [Main publishing](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37153145983) |
| v1.0.0 amd64/arm64 publishing and manifest | [Tag publishing](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37153147857) |
| Anonymous v1.0.0 pulls and container persistence on both architectures | [Release smoke](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37153359325) |
| Anonymous latest pulls and container persistence on both architectures | [Latest smoke](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37153360876) |

The development machine has no Docker; container checks ran in Actions. Real PikPak operations were not rerun, and no VPS was deployed. Local secret-exclusion checks covered staged files and the release tree before publication; credentials remain local.

## Unused-file and code cleanup (2026-10-04)

Removed 19 unreferenced screenshots (791,031 bytes), an unused RSS fixture, unused Web constructor configuration, an unused JavaScript regex constant, redundant empty-set initialization and CSS for retired UI elements. Screenshots referenced below, migrations, legacy naming rules and the cursor API remain available. `go mod tidy -diff` found no dependency changes.

Passed locally with Go 1.27.1: `scripts/verify.ps1` (formatting, `go test ./...`, `go vet ./...`, build, JavaScript syntax and localization) and `git diff --check`. Go used a workspace-local cache; the build emitted a nonfatal module-cache metadata permission warning.

Headless Edge used only `.local/release-ui-data` with a mocked account. Both languages and five management views passed at 1280, 390 and 320 pixels without horizontal overflow or browser errors. Reading 72 torrent filenames, regex previews and the folder-create dialog passed. Desktop/mobile fixture screenshots were visually inspected. No real PikPak operation was performed; container verification runs in Actions.

## Administrator password changes (2026-10-04)

Passed locally: Go 1.27.1 `go test ./...`, `go vet ./...`, application build, JavaScript syntax checks, `node scripts/test-i18n.cjs` and `git diff --check`. Go used a workspace-local build cache. The build succeeded with a nonfatal module-cache metadata permission warning.

Regression tests cover authenticated/CSRF-protected changes, empty/mismatched/wrong passwords, invalid JSON, shared authentication throttling, serialized Argon2 operations, storage failure preserving the verifier and sessions, stale database updates, all-session revocation, cookie expiry/CSRF rotation, old-password rejection and reopening the database with the new password. One-character, Unicode, long and whitespace-containing passwords remain supported. Application settings remain intact; persisted files contain no tested plaintext passwords.

Headless Edge used only an isolated mocked account and `.local/password-ui-data`. Traditional Chinese and English settings, inline errors, confirmation mismatch without a request, input clearing, password changes, revocation in two browser contexts and subsequent login passed. Desktop (1280×900) and mobile (390×844) fixture screenshots were visually inspected; no horizontal overflow or browser errors occurred. The sidebar and favicon reference the same SVG. No real PikPak operation or Docker validation was performed for this change.

## Shared favicon and brand icon (2026-10-04)

The favicon and the setup, login and sidebar brand marks share one SVG with a white P and northeast arrow on purple. Passed locally: `go test ./internal/web`, `node scripts/test-i18n.cjs` and `git diff --check`. Go used a workspace-local build cache because the default cache was inaccessible.

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
