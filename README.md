# PikPak RSS Manager

Self-hosted RSS/Atom subscriptions and offline downloads through the official PikPak MCP. Media stays in PikPak.

## Start

Download [docker-compose.yml](docker-compose.yml), then run:

```sh
docker compose up -d
```

Open the site, create an administrator password, confirm the public URL, and enter a PikPak PAT in **Settings**. There is no default password; credentials are configured through the UI.

Release image: `ghcr.io/wade00754/pikpak-rss-manager:v1.2.0` (linux/amd64 and linux/arm64). The Compose file tracks `latest`; set its image tag to `v1.2.0` to pin this release. Compose binds to `127.0.0.1:8080`; use a reverse proxy for remote access. [Deployment, 1Panel, updates and backups](docs/deployment.md) · [Release notes](docs/releases/v1.2.0.md).

## PAT permissions

[Create a PAT](https://mypikpak.com/en-US/help-center/connected_apps/personal_access_tokens/create_personal_access_token) with:

| Permission | Purpose |
|---|---|
| **Manage files** | Includes reading/writing, browsing/creating folders and renaming files; also supports existing staged jobs. |
| **Cloud Download** | Creates offline download tasks. |

New tasks do not use Trash. Existing staged jobs retain their cleanup permission requirements; missing permission leaves their staging with a warning. Permanent deletion, Share, Invite and Account settings permissions are not needed. [Official permission reference](https://mypikpak.com/en-US/help-center/connected_apps/managing_connected_apps/connected_app_permissions).

## Use

- **Subscriptions:** add an RSS URL, destination and interval. First check establishes a baseline. **Backfill** selects torrents and confirms downloads; multi-file torrents download in full. Repeated explicit downloads are allowed.
- **Offline tasks:** submit a Magnet, torrent URL or direct HTTP/HTTPS URL and choose a destination. Original filenames are retained. **Delete task** cancels its PikPak download before removing the record; downloaded files are preserved. **Clear completed tasks**, between **Refresh** and **New task** in the page header, removes completed records only.
- **Renaming:** optional for new subscriptions. Regex replacement supports `$1`, `${name}` and `$$`; nonmatches keep their names. Select a torrent filename to preview. Existing template rules remain supported.
- **Destinations:** browse/create folders or enter a path. Reselect saved folders after switching accounts.

New tasks download directly to the destination. Optional renaming changes filenames in place; torrent directories and attachments remain where PikPak downloads them. Duplicate downloads use PikPak's filename suffixes. No staging, backup or attachment folders are created, and new jobs do not move, overwrite, trash or clean cloud content. Existing jobs keep their original staged workflow and IDs.

Jobs resume after restarts using saved task/file IDs and rename progress. Actual names are read back after renaming. Rejected renames retain the current file for review. Uncertain submissions without an ID are not resubmitted; missing file IDs require review only when renaming is enabled. Authorization/quota errors pause downloads; update the PAT or check the connection, then resume affected tasks.

Failed cancellation retains the task record. For an uncertain submission without a PikPak task ID, cancel it in PikPak first, then confirm record deletion. Task deletion preserves feed baselines; retrying the same confirmed request does not recreate a deleted task. Clearing completed legacy records also stops their remaining empty-folder cleanup retries.

Magnet links use a normalized infohash as their first parameter, with trackers and display names retained. Already submitted tasks keep their PikPak task IDs and are not automatically resubmitted when the link format changes.

**Refresh** immediately reloads server-recorded subscriptions, tasks, events and connection status. The visible dashboard also refreshes every 15 seconds when task/subscription dialogs are closed. Refresh does not trigger RSS checks or resume downloads.

To update, back up the complete data volume, then run `docker compose pull` and `docker compose up -d`. Preserve `secret.key` with the database. v1.2.0 retains schema 4. Downgrading with direct jobs requires the pre-upgrade backup. [Upgrade and restore details](docs/deployment.md).

## Development

Install Go 1.27.1, GNU Make and Node.js. The Makefile works with Windows `cmd.exe` and POSIX shells; Python and PowerShell scripts are not required. Run targets from the repository root:

```sh
make dev
make verify
```

`make verify` checks Go formatting, runs tests and vet, checks JavaScript syntax/translations, and builds `.local/pikpak-rss-manager` (`.exe` on Windows). Individual targets: `build`, `test`, `test-race`, `vet`, `test-ui` and `version`. Use `make help` for all targets. Direct Go commands remain supported.

Remove task-created scratch programs, fixture data, logs, verification binaries and caches after use. Keep reusable tests under `tests/`; Go tests use automatically cleaned temporary directories and report evidence in test output. Preserve production data, credentials, active binaries, backups and requested deliverables. See [cleanup rules](AGENTS.md#temporary-files-and-cleanup).

Container tests require Docker with Compose and Buildx:

```sh
make docker-build IMAGE=pikpak-rss-manager:test
make test-container IMAGE=pikpak-rss-manager:test
make test-manifest IMAGE=ghcr.io/wade00754/pikpak-rss-manager:latest
```

Container tests use isolated temporary volumes and fixture data without a PAT. They bypass local `.env` files and remove their test volumes afterward. Optional test settings: `SMOKE_PLATFORM` and `SMOKE_EXPECTED_VERSION`. The Windows development machine runs container validation in GitHub Actions.

Implementation commits include local verification and the release decision in [Verification](docs/verification.md). Post-release CI, image publication and anonymous-pull results are added to the corresponding [GitHub Release](https://github.com/wade00754/pikpak-rss-manager/releases), without a follow-up documentation commit. Tasks without a release report post-push results with Actions links in the delivery response.

Go 1.27.1; SQLite; embedded UI; MIT license. Optional process settings: `APP_LISTEN` and `APP_DATA_DIR`. The service does not load `.env`. [Architecture/API](docs/architecture.md) · [Verification](docs/verification.md) · [PAT/OAuth](docs/auth-comparison.md).
