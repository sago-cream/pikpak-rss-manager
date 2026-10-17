# Verification

## Current change: backfill select-all (2026-10-17)

The backfill dialog can select or clear all available torrents. The checkbox shows partial selection, excludes rows with metadata errors, and resets when the dialog opens. Confirmation and retry keep the selected whole torrents fixed. Traditional Chinese and English labels are supported.

Release decision: this user-visible feature warrants a minor release after review and merge. This delivery opens a draft PR only. The published version remains v2.0.1. No release, tag, image publication or deployment is part of this delivery.

Local checks use Go 1.27.1 and Node.js 24.10.0 with task-owned temporary tools and caches. Passed: `CGO_ENABLED=0 GOFLAGS=-buildvcs=false make verify` (formatting, `go test ./...`, `go vet ./...`, UI syntax/localization/task-details/backfill checks and application build), `git diff --check`, and local Markdown file links. The executable reports `v2.0.1`. `make test-race` needs a C compiler, which is unavailable on this worker. Container checks require Actions because Docker is unavailable locally.

Chromium checks passed against isolated fixture APIs and the actual template/static assets. They cover select-all/clear, partial selection, unavailable rows, keyboard activation, reopening/reset, English labels and a 390-pixel mobile viewport. Matched desktop screenshots compare the exact PR base with the changed assets. JavaScript regression tests also cover loading/empty/all-unavailable plans, canceled confirmation, duplicate-submit protection, identical retry payloads and ignored stale/closed previews. These mocked checks do not prove real PikPak operations. No real PikPak operations were performed for this change.

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
