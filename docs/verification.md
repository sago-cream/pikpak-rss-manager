# Verification

## Current validation (2026-10-05)

Management pages place theme and language controls in the sidebar footer with connection status, version and sign-out. Navigation uses 20 px SVG icons, a 12 px desktop label gap and 44 px minimum row height. The footer stays available on mobile. Active navigation exposes `aria-current`.

Release decision: publish v2.0.2 after this draft PR is merged, because the sidebar changes are user-visible. VERSION and README retain the current published v2.0.1 until release preparation. No tag or image is published from this draft branch.

Passed locally: `make verify` and `make test-race` with Go 1.27.1 and `GOFLAGS=-buildvcs=false`. Formatting, `go test ./...`, `go vet ./...`, JavaScript syntax/localization, task-details checks and the application build pass. The executable reports `v2.0.1`. `git diff --check` and local Markdown links pass. Mocked regression coverage includes destination/root submission, torrent directories and attachments, rejected/uncertain renames, restart recovery, authentication/quota/rate failures, baselines, repeated downloads, confirmation idempotency, task cancellation and CSRF.

Browser checks use an isolated mocked account with no production data or PAT. Desktop before/after captures use the exact PR base and changed templates/assets at the same 1280 × 800 viewport. Checks cover 20 px navigation icons, a 12 px desktop gap, no top bar, footer fit at 320 px, Traditional Chinese/English language persistence, light/dark theme persistence, active navigation and footer access at 900 × 400. Login controls also render correctly. Screenshots are PR attachments. No real PikPak operations were performed for this sidebar change. Docker checks run in Actions because the development machine has no Docker. Post-push workflow results belong in the PR and delivery response. Release publication and anonymous pulls wait until merge.

## Real PikPak validation and limits

On 2026-10-04, `TestLiveDirectDownloadAndRename` passed in 23.94 seconds against the official hosted MCP, using a locally encrypted UI-configured PAT inside `_pikpak-rss-manager-test/direct-20261004T124346-65facc`. Three public-domain Alice HTTP downloads (163,783 bytes each) completed directly in the destination. Repeat downloads produced `alice.txt` and `alice(1).txt`. Worker renaming produced `direct-renamed.txt`. A same-name rename was rejected and retained its current filename without changing the other files. Three files and no extra folders remained. Fixtures were preserved. No moves, Trash, permanent deletion or production-task changes occurred.

Real nested torrents, root-destination downloads, restart recovery and suffix-on-rename behavior were not tested. Applicable paths use mocks. Magnet availability/instant download has not been proven. Folder browsing/creation and file-preserving task removal were validated in earlier isolated runs recorded in [Git history](https://github.com/wade00754/pikpak-rss-manager/tree/09ca537cd6e6469b1143c929acc2cca09c6b99d8/docs). Mocked checks do not prove real cloud operations. OAuth registration, consent and refresh flows remain unimplemented and untested. No VPS was deployed. Docker validation runs in Actions because the development machine has no Docker.

## Reproduce

From the repository root:

```sh
make verify
make test-race
make docker-build IMAGE=pikpak-rss-manager:test
make test-container IMAGE=pikpak-rss-manager:test
make test-manifest IMAGE=ghcr.io/wade00754/pikpak-rss-manager:v2.0.1
```

Docker commands require Compose and Buildx. Cloud integration tests are opt-in and require a PAT already configured through the Web UI. No PAT is sent to CI. Set `PIKPAK_LIVE_DATA_DIR` for a custom local credential data directory.

| Scope | Opt-in variable | Test |
|---|---|---|
| Folder browsing/creation only | `PIKPAK_LIVE_FOLDERS_TEST=1` | `TestLiveFolderBrowsingAndCreation` |
| Direct HTTP downloads, repeat names and rename rejection | `PIKPAK_LIVE_DIRECT_TEST=1` | `TestLiveDirectDownloadAndRename` |

For example, on a POSIX shell:

```sh
PIKPAK_LIVE_DIRECT_TEST=1 go test ./tests/integration -run '^TestLiveDirectDownloadAndRename$' -count=1 -v -timeout=7m
```

Live tests create isolated cloud fixtures and preserve them. Local test databases use automatically cleaned temporary directories. Remove verification-only binaries, caches and other scratch artifacts after recording useful results. Preserve production data, credentials, intentional backups and requested deliverables.

Record local checks and release decisions here within the implementation commit. Record subsequent CI/publication/anonymous-pull results in the corresponding GitHub Release and delivery response. Tasks without a release use the delivery response and Actions links. Do not add separate evidence-only commits after pushing.
