# Verification

Documentation maintenance (2026-10-05): preserved user-directed README removals. AGENTS.md requires a README review for every release, with edits only for outdated instructions or missing information users need. README review found no further necessary edits. `git diff --check` and local Markdown links/anchors pass. Required documents remain. Documentation only, no new release or application tests. Commit uses `[skip ci]`.

## Current release: v2.0.2 (2026-10-05)

Unconfirmed rename messages now use a short conflict-check reminder, including existing stored messages in both languages. Task details show unchanged filenames once. Changed filenames retain a labeled original for comparison. Pending actions display the known filename rather than the requested rename. User filenames, stored records and cloud behavior are unchanged.

Release decision: v2.0.2 for user-visible task-detail and message fixes. No existing tags are changed. Source VERSION and README are synchronized. Post-push CI, multiarch publication and anonymous pulls will be recorded in the GitHub Release.

Passed locally: `make verify` (formatting, `go test ./...`, `go vet ./...`, JavaScript syntax/localization, task-details checks and application build) with Go 1.27.1, `GOFLAGS=-buildvcs=false` and task-owned workspace caches. The executable reports `v2.0.2`. `git diff --check` and local Markdown file links pass. Mocked regression coverage includes destination/root submission, torrent directories and attachments, suffix readback, Regex/nonmatches, rejected/uncertain renames, partial-action/restart recovery, missing task/file IDs, auth/quota/rate failures, baseline/repeated-download/idempotent confirmation behavior, task cancellation and CSRF. Pending submission intent survives account pause and restart, blocks unconfirmed record deletion and prevents resubmission. Fixture-based UI checks cover one-line unchanged filenames across completed/review/pending/renaming states, labeled original/current comparisons, both legacy message versions, and preservation of filenames matching application messages in both languages. No real PikPak operations were performed for this UI fix. The earlier direct-download validation does not prove the minimal PAT scope combination. Task-created build caches, temporary build directories and the verification executable were removed. Development data, keys and intentional backups were preserved.

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
make test-manifest IMAGE=ghcr.io/wade00754/pikpak-rss-manager:v2.0.2
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
