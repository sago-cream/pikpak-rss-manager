# Verification

## Task header controls (2026-10-04)

Moved **Clear completed tasks** between **Refresh** and **New task** in the offline-task header. The clearing button is hidden on other pages; all three controls remain available on narrow task screens. Refresh reloads the four read-only dashboard APIs; it does not check RSS or resume tasks. The existing 15-second automatic refresh remains unchanged.

Passed locally: `make verify` with Go 1.27.1 and `GOFLAGS=-buildvcs=false`, plus `git diff --check`. Headless Edge used an isolated mocked account and checked Traditional Chinese/English, header order and visibility, navigation to all other pages, 1280/390/320-pixel widths, GET-only refresh behavior and absence of overflow/browser errors. Fixture screenshots were visually inspected: [desktop](screenshots/v111-task-header-desktop.png), [mobile](screenshots/v111-task-header-mobile.png). No real PikPak operation was performed.

Release decision: published [v1.1.1](https://github.com/wade00754/pikpak-rss-manager/releases/tag/v1.1.1) for this user-visible layout fix, without changing schema 4 or existing tags. Source commit `91841de` passed:

| Validation | Evidence |
|---|---|
| Linux race/vet/build, UI checks, Docker startup/non-root/persistence | [CI](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37201895188) |
| latest/main amd64/arm64 publishing and manifest | [Main publishing](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37201895229) |
| v1.1.1 amd64/arm64 publishing and manifest | [Tag publishing](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37201896847) |
| Anonymous v1.1.1 pulls, Compose persistence and manifests on both architectures | [Release smoke](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37202115749) |
| Anonymous latest pulls, Compose persistence and manifests on both architectures | [Latest smoke](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37202118197) |

The local service was restarted; `/healthz` reports `v1.1.1` and `ok`. No VPS was deployed. This documentation evidence uses `[skip ci]`; the application source and published images above received the full checks.

## v1.1.0 release validation (2026-10-04)

Source/release commit `0945359` passed GitHub Actions:

| Validation | Evidence |
|---|---|
| Linux race/vet/build, UI localization, Docker startup/authentication/non-root/persistence | [CI](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37199416561) |
| latest/main linux/amd64 and linux/arm64 publication and manifest | [Main publishing](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37199416500) |
| v1.1.0 linux/amd64 and linux/arm64 publication and manifest | [Tag publishing](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37199417887) |
| Anonymous v1.1.0 pulls, Compose startup/persistence and manifests on both architectures | [Release smoke](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37199615837) |
| Anonymous latest pulls, Compose startup/persistence and manifests on both architectures | [Latest smoke](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37199632315) |

The new [v1.1.0 release](https://github.com/wade00754/pikpak-rss-manager/releases/tag/v1.1.0) retains existing tags. The Windows service at localhost:8080 was restarted after a complete local data backup; `/healthz` reports `v1.1.0` and `ok`. No production task was canceled or resubmitted, and no VPS was deployed. Documentation evidence uses `[skip ci]`; the application source and published images above received the full checks.

## Task deletion and completed-record clearing (2026-10-04)

Delete buttons on task rows and details cancel incomplete PikPak downloads before deleting local records. Completed clearing removes all completed records, including those outside the latest-200 view. Cloud files, feed baselines and event history are retained. SQLite schema 4 keeps deleted-request IDs so a repeated backfill confirmation cannot recreate a removed task. Cancellation/account errors preserve records; unknown submissions without IDs require manual cancellation before explicit record deletion. Deletion is serialized with worker submissions/file actions and account switches.

Passed locally: `make verify` with Go 1.27.1 and `GOFLAGS=-buildvcs=false`. Tests cover official MCP `task_rm` arguments with `delete-files:false`, single cancellation attempts, rollback, 205-record clearing, request tombstones after restart, auth/CSRF, account changes, missing remote tasks, unknown submissions and concurrent submission/deletion.

Headless Edge used only an isolated mocked account/data directory. Traditional Chinese/English controls, dismissed confirmation, active cancellation, completed-only clearing, explicit unknown-submission removal and task-detail deletion passed at 1280×900 and 390×844 without browser errors or horizontal overflow. Fixture screenshots were visually inspected: [desktop](screenshots/v110-task-actions-desktop.png), [mobile](screenshots/v110-task-actions-mobile.png).

Real MCP validation in `_pikpak-rss-manager-test/download-20261004T112602-c18672` removed the completed public-domain Alice test task through `task_rm` with `delete-files:false`. Reading its file afterward confirmed the same ID, parent and 163,783-byte size, without Trash. This proves file-preserving task removal for that test, not cancellation of the production stalled source. No production task was canceled or resubmitted.

Release decision: published v1.1.0 for these user-visible features and the Magnet compatibility fix. Existing tags remain unchanged. Linux race, Docker and anonymous multiarch checks passed as recorded above.

## Canonical Magnet submission (2026-10-04)

The newly reported production task existed in PikPak with status `running`, 0% progress and message `Saving`. Its infohash matched both the persisted resource key and the public RSS torrent metadata. This does not establish why that source remained stalled.

Magnet generation now puts the selected normalized infohash first with literal URN separators. Display names, trackers, source parameters and hybrid v1/v2 topics retain their decoded values. Legacy queued links are normalized before their first submission; active tasks retain their IDs without another submission.

Passed locally: `make verify` with Go 1.27.1 and `GOFLAGS=-buildvcs=false`. Regression coverage includes parameter preservation, hybrid/v2 links, idempotent normalization, generated torrent links and one submission for legacy queued jobs. Real validation under `_pikpak-rss-manager-test/download-20261004T112602-c18672` downloaded the 163,783-byte public-domain Alice fixture at 100%. No production task was resubmitted, and no credentials were printed or uploaded.

Release decision: included this user-visible compatibility fix in the new v1.1.0 release with task deletion. Existing tags remain unchanged. Docker/race/public-image checks passed as recorded above.

## Makefile and Go container tests (2026-10-04)

The root Makefile now supplies the shared development/CI commands. PowerShell wrappers were removed, the Python container smoke and architecture checks were replaced with opt-in Go tests, and localization tests moved to `tests/web`. README documents the prerequisites and targets. AGENTS requires documentation updates, immediate Conventional Commits, GitHub uploads, workflow verification and an explicit release decision for each implementation task.

Passed locally on Windows with GNU Make 4.4.1 and Go 1.27.1: `make help`, dry-run verification/container commands, `make verify` (formatting, `go test ./...`, `go vet ./...`, JavaScript syntax/localization and application build), isolated smoke-harness tests and `git diff --check`. The executable reports `v1.0.0`. Go used a workspace-local build cache; the successful build emitted a nonfatal shared module-cache metadata permission warning.

The smoke harness ran the actual Web handlers against an isolated temporary SQLite database with simulated Docker commands. It checked setup/CSRF, login, regex preview and persistence after reopening the database. Additional tests cover startup failure/cancellation cleanup, retention of both startup and cleanup errors, temporary setting removal, digest/platform overrides, HTTP status rejection and architecture validation. Real container tests explicitly bypass local `.env` files and clean only their randomly named Compose project's test volumes. No real PAT or PikPak operation was used; simulated Docker commands do not prove container behavior. Local formatting/UI targets also passed using a POSIX shell.

Source commit `ee95986` passed the updated Makefile workflows in GitHub Actions:

| Validation | Evidence |
|---|---|
| Linux race/vet/build, localization, Docker startup/authentication/non-root/persistence | [CI](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37188737472) |
| latest/main linux/amd64 and linux/arm64 publishing and manifest checks | [Main publishing](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37188737448) |
| Anonymous latest pulls, Compose persistence and manifests on both architectures | [Latest smoke](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37188948670) |

Release decision: no new version or release-tag update. These changes affect development, tests and CI only; application behavior, data schema and deployment remain unchanged. The latest/main multiarch images were refreshed through the existing publication workflow; `v1.0.0` was not retagged or republished.

## Concise interface messages (2026-10-04)

Removed the response-privacy explanation from PikPak failures, the PAT encryption/browser-return paragraph, and repeated automatic-retry explanations. Storage failures, PAT confirmation and limits use concise wording. Error causes, required actions and overwrite/backup confirmations remain visible.

Existing task and activity messages use the shorter copy in Traditional Chinese and English without rewriting stored history. Localization regressions verify historical errors/statuses are normalized while identical user names and filenames remain unchanged.

Passed locally with Go 1.27.1: `go test ./...`, `go vet ./...`, application build with `-buildvcs=false`, `node scripts/test-i18n.cjs`, JavaScript syntax checks and `git diff --check`. No live PikPak operation or browser screenshot was needed for this copy change.

## Missing remote download tasks (2026-10-04)

Passed locally with Go 1.27.1: `go test ./...`, `go vet ./...`, application build with `-buildvcs=false`, localization checks and `git diff --check`.

The reported task's persisted infohash matched its public RSS torrent metadata. Read-only official MCP checks found a task-get 404, no matching remote task and an empty persisted staging folder. The user confirmed cancelling/deleting the remote task. Folder-detail lookup returned 400 for that staging ID; equivalent read-only lookups of the isolated test folder and file succeeded. Empty/ambiguous staging listings now require review without querying folder details. The verified worker reconciled the cancelled local job to `needs_review`, retained its task ID and performed no cloud mutation. These findings explain the stale local download status after cancellation; they do not establish why the download previously stayed at 0%.

Real Magnet comparison used only the 163,783-byte public-domain Alice fixture under `_pikpak-rss-manager-test/magnet-20261004T073539-ac24ca`. The application's existing percent-encoded Magnet completed at 100%, with its expected cloud filename, complete phase and size confirmed. An equivalent Magnet with an unescaped, first-position infohash was accepted and initially reported 90%. All created cloud content was retained. No production download was resubmitted; no credentials were printed or sent to CI.

Regression tests cover the official MCP's sanitized 404 classification, unique completed staging-file/folder recovery after a missing task, persisted organization after reopening SQLite, empty/ambiguous/partial/trashed staging content, invalid folder identity, absent staging IDs without root scans, authorization/quota pauses and transient/rate backoff. A missing task without confirmed completed content requires review immediately. The persisted task ID is retained, and recovery never submits another download. Docker and Linux race checks were not run locally.

## Empty staging cleanup (2026-10-04)

README permission instructions were checked against the official Connected App permission reference: Manage files includes read/write plus Trash, and Cloud Download permits offline tasks. The README now keeps setup, required scopes, task behavior and update steps, with detailed architecture/verification linked separately.

Passed locally: Go 1.27.1 `go test ./...`, `go vet ./...`, application build reporting `v1.0.0`, localization checks and `git diff --check`. The build emitted a nonfatal shared module-cache metadata permission warning.

Regression tests cover nested empty torrent folders, downloaded-file preservation, replacement backups (including empty `_Replaced` folders), unexpected files, sibling/legacy/review jobs, account changes, folder identity revalidation, encrypted restart recovery, scheduling completed cleanup jobs, three-attempt limits, missing Manage files permission and uncertain Trash responses at nested/job/container boundaries. The mocked official MCP adapter verifies `rm` receives only an `ids` array.

Read-only discovery against the official hosted MCP confirmed that `rm` moves files/folders to Trash and accepts `ids`. After correcting the test to create each path segment separately, opt-in `TestLiveStagingCleanup` passed in 38.89 seconds using `_pikpak-rss-manager-test/cleanup-20261004T080039-889d29`. A small public-domain Alice HTTP download completed, its original filename was preserved in the destination, and both its empty job folder and `_PikPak-RSS-Staging` container were moved to Trash. The run directory and downloaded fixture remain. This proves the single-file HTTP cleanup path with the current local PAT; nested torrent cleanup, replacement backups and failure recovery were checked with mocks. No credentials were printed or sent to CI, and no permanent deletion or VPS deployment was performed.

To repeat locally with an administrator/PAT configured through the UI, set `PIKPAK_LIVE_CLEANUP_TEST=1` and run `go test ./tests/integration -run TestLiveStagingCleanup -count=1 -v`. The test retains a small public-domain HTTP fixture beneath a new `_pikpak-rss-manager-test/cleanup-<run-id>` and only moves its empty staging folders to Trash.

The existing v1.0.0 tag was reissued at verified source commit `320dbf9`. Both v1.0.0 and latest published images passed anonymous OCI revision checks against that exact commit on linux/amd64 and linux/arm64. Publication and public container checks passed:

| Validation | Evidence |
|---|---|
| Linux race/vet/build, localization, non-root startup/persistence | [CI](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37187637247) |
| main/latest amd64/arm64 publishing and manifest | [Main publishing](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37187637189) |
| Reissued v1.0.0 amd64/arm64 publishing and manifest | [Tag publishing](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37187668713) |
| Anonymous latest pulls and persistence on both architectures | [Latest smoke](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37187840374) |
| Anonymous v1.0.0 pulls and persistence on both architectures | [Release smoke](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37187857185) |

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
make verify
make test-race
make docker-build IMAGE=pikpak-rss-manager:test
make test-container IMAGE=pikpak-rss-manager:test
make test-manifest IMAGE=ghcr.io/wade00754/pikpak-rss-manager:latest
```

Container and manifest targets require Docker with Compose/Buildx; race tests run in Linux CI. Historical entries below and above retain the commands used at the time.

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
