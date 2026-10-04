# Verification

## Documentation maintenance (2026-10-04)

Removed superseded UI screenshots and the local v1.1.0/v1.1.1 release-note copies. Those releases remain available on [GitHub Releases](https://github.com/wade00754/pikpak-rss-manager/releases). Consolidated this document around current validation and reproducible checks; deployment, API, authentication research and v1.2.0 source notes remain available. Earlier verification records and screenshots are preserved in [Git history](https://github.com/wade00754/pikpak-rss-manager/tree/09ca537cd6e6469b1143c929acc2cca09c6b99d8/docs).

The [AGENTS documentation rules](../AGENTS.md#documentation-maintenance) define required documents, current verification scope, screenshot/release-note retention, historical evidence, reference checks and delivery. Future edits consolidate obsolete records rather than adding development diaries or archive directories.

Local validation: checked repository Markdown links and anchors, removed-file references, required documentation and `git diff --check`. No application, test program or workflow changed; application/container/live tests were not rerun. Release decision: no new version for this documentation-only task; v1.2.0 and existing tags remain unchanged. This commit uses `[skip ci]`; post-push workflow status is reported in the delivery response.

## Current release: v1.2.0

Source commit `7f2ef54` passed local `make verify` with Go 1.27.1, `GOFLAGS=-buildvcs=false` and workspace-local caches. Tests cover direct destination/root submission, torrent directories and attachments, filename suffix readback, regex/nonmatches, rejected/uncertain renames, restart and partial-action recovery, missing task/file IDs, uncertain submission without another download, auth/quota/rate errors, automatic baselines, repeated downloads and idempotent backfill confirmations. Existing staged recovery, backup and cleanup tests remain. UI tests cover localization and mocked English/Traditional Chinese task details; no browser screenshot test was performed for v1.2.0.

| Validation | Evidence |
|---|---|
| Linux race/vet/build/UI and Docker startup/authentication/non-root/persistence | [CI](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37203412869) |
| main/latest amd64/arm64 publishing and manifest | [Main publishing](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37203412860) |
| v1.2.0 amd64/arm64 publishing and manifest | [Tag publishing](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37203437221) |
| Anonymous v1.2.0 pulls, Compose persistence and manifests on both architectures | [Release smoke](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37203652687) |
| Anonymous latest pulls, Compose persistence and manifests on both architectures | [Latest smoke](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37203623470) |

Subsequent test/local-artifact cleanup commit `157ebf0` passed [CI](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37207943285) and [main/latest publishing](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37207943362). `TestLivePikPak` now uses `t.TempDir()` and test output instead of persistent local databases/reports; this cleanup change was compiled but not rerun against PikPak.

## Real PikPak validation and limits

On 2026-10-04, `TestLiveDirectDownloadAndRename` passed in 23.94 seconds against the official hosted MCP, using a locally encrypted UI-configured PAT inside `_pikpak-rss-manager-test/direct-20261004T124346-65facc`. Three public-domain Alice HTTP downloads (163,783 bytes each) completed directly in the destination. Repeat downloads produced `alice.txt` and `alice(1).txt`; worker renaming produced `direct-renamed.txt`. A same-name rename was rejected and retained its current filename without changing the other files. Three files and no extra folders remained. Fixtures were preserved; no moves, Trash, permanent deletion or production-task changes occurred.

Real nested torrents, root-destination downloads, restart recovery and suffix-on-rename behavior were not tested; applicable paths use mocks. Magnet availability/instant download has not been proven. Folder browsing/creation, staged HTTP rename/move/empty-folder cleanup and file-preserving task removal were validated in earlier isolated runs recorded in Git history. Mocked checks do not prove real cloud operations. OAuth registration, consent and refresh flows remain unimplemented and untested. No VPS was deployed; Docker validation runs in Actions because the development machine has no Docker.

## Reproduce

From the repository root:

```sh
make verify
make test-race
make docker-build IMAGE=pikpak-rss-manager:test
make test-container IMAGE=pikpak-rss-manager:test
make test-manifest IMAGE=ghcr.io/wade00754/pikpak-rss-manager:v1.2.0
```

Docker commands require Compose and Buildx. Cloud integration tests are opt-in and require a PAT already configured through the Web UI; no PAT is sent to CI. Set `PIKPAK_LIVE_DATA_DIR` for a custom local credential data directory.

| Scope | Opt-in variable | Test |
|---|---|---|
| Folder browsing/creation only | `PIKPAK_LIVE_FOLDERS_TEST=1` | `TestLiveFolderBrowsingAndCreation` |
| Direct HTTP downloads, repeat names and rename rejection | `PIKPAK_LIVE_DIRECT_TEST=1` | `TestLiveDirectDownloadAndRename` |
| Legacy staged Magnet/HTTP rename/move | `PIKPAK_LIVE_TEST=1` | `TestLivePikPak` |
| Legacy empty-staging cleanup | `PIKPAK_LIVE_CLEANUP_TEST=1` | `TestLiveStagingCleanup` |

For example, on a POSIX shell:

```sh
PIKPAK_LIVE_DIRECT_TEST=1 go test ./tests/integration -run '^TestLiveDirectDownloadAndRename$' -count=1 -v -timeout=7m
```

Live tests create isolated cloud fixtures and preserve them. Local test databases use automatically cleaned temporary directories. Remove verification-only binaries, caches and other scratch artifacts after recording useful results; preserve production data, credentials, intentional backups and requested deliverables.

Record local checks and release decisions here within the implementation commit. Record subsequent CI/publication/anonymous-pull results in the corresponding GitHub Release and delivery response; tasks without a release use the delivery response and Actions links. Do not add separate evidence-only commits after pushing.
