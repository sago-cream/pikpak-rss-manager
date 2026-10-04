# PikPak RSS Manager

Self-hosted RSS/Atom subscriptions and offline downloads through the official PikPak MCP. Media stays in PikPak.

## Start

Download [docker-compose.yml](docker-compose.yml), then run:

```sh
docker compose up -d
```

Open the site, create an administrator password, confirm the public URL, and enter a PikPak PAT in **Settings**. There is no default password; credentials are configured through the UI.

Release image: `ghcr.io/wade00754/pikpak-rss-manager:v1.1.1` (linux/amd64 and linux/arm64). The Compose file tracks `latest`; set its image tag to `v1.1.1` to pin this release. Compose binds to `127.0.0.1:8080`; use a reverse proxy for remote access. [Deployment, 1Panel, updates and backups](docs/deployment.md) · [Release notes](docs/releases/v1.1.1.md).

## PAT permissions

[Create a PAT](https://mypikpak.com/en-US/help-center/connected_apps/personal_access_tokens/create_personal_access_token) with:

| Permission | Purpose |
|---|---|
| **Manage files** | Includes reading/writing, browsing/creating folders, renaming/moving files and moving empty staging folders to Trash. |
| **Cloud Download** | Creates offline download tasks. |

Read & write files alone cannot clean staging folders. For an existing PAT, grant Manage files or create a replacement and save it in Settings. Missing cleanup permission retains staging with a warning; completed downloads remain successful. Permanent deletion, Share, Invite and Account settings permissions are not needed. [Official permission reference](https://mypikpak.com/en-US/help-center/connected_apps/managing_connected_apps/connected_app_permissions).

## Use

- **Subscriptions:** add an RSS URL, destination and interval. First check establishes a baseline. **Backfill** selects torrents and confirms download/replacement; multi-file torrents download in full. Repeated explicit downloads are allowed.
- **Offline tasks:** submit a Magnet, torrent URL or direct HTTP/HTTPS URL and choose a destination. Original filenames are retained. **Delete task** cancels its PikPak download before removing the record; downloaded files are preserved. **Clear completed tasks**, between **Refresh** and **New task** in the page header, removes completed records only.
- **Renaming:** optional for new subscriptions. Regex replacement supports `$1`, `${name}` and `$$`; nonmatches keep their names. Select a torrent filename to preview. Existing template rules remain supported.
- **Destinations:** browse/create folders or enter a path. Reselect saved folders after switching accounts.

New tasks download into `<destination>/_PikPak-RSS-Staging/<jobID>`. After completion, empty task/torrent folders and the empty staging container move to Trash. Backups, unfinished tasks, remaining files and legacy staging are preserved. Confirmed replacements keep originals under the task's `_Replaced` folder; ordinary name/folder collisions require review.

Jobs resume after restarts. Uncertain submissions require reconciliation rather than another submission. Authorization/quota errors pause downloads; update the PAT or check the connection, then resume affected tasks.

Failed cancellation retains the task record. For an uncertain submission without a PikPak task ID, cancel it in PikPak first, then confirm record deletion. Task deletion preserves feed baselines; retrying the same confirmed request does not recreate a deleted task. Clearing completed records also stops their remaining empty-folder cleanup retries.

Magnet links use a normalized infohash as their first parameter, with trackers and display names retained. Already submitted tasks keep their PikPak task IDs and are not automatically resubmitted when the link format changes.

**Refresh** immediately reloads server-recorded subscriptions, tasks, events and connection status. The visible dashboard also refreshes every 15 seconds when task/subscription dialogs are closed. Refresh does not trigger RSS checks or resume downloads.

To update, back up the complete data volume, then run `docker compose pull` and `docker compose up -d`. Preserve `secret.key` with the database. v1.1.0 migrates to schema 4; downgrade requires the pre-upgrade backup. [Upgrade and restore details](docs/deployment.md).

## Development

Install Go 1.27.1, GNU Make and Node.js. The Makefile works with Windows `cmd.exe` and POSIX shells; Python and PowerShell scripts are not required. Run targets from the repository root:

```sh
make dev
make verify
```

`make verify` checks Go formatting, runs tests and vet, checks JavaScript syntax/translations, and builds `.local/pikpak-rss-manager` (`.exe` on Windows). Individual targets: `build`, `test`, `test-race`, `vet`, `test-ui` and `version`. Use `make help` for all targets. Direct Go commands remain supported.

Container tests require Docker with Compose and Buildx:

```sh
make docker-build IMAGE=pikpak-rss-manager:test
make test-container IMAGE=pikpak-rss-manager:test
make test-manifest IMAGE=ghcr.io/wade00754/pikpak-rss-manager:latest
```

Container tests use isolated temporary volumes and fixture data without a PAT. They bypass local `.env` files and remove their test volumes afterward. Optional test settings: `SMOKE_PLATFORM` and `SMOKE_EXPECTED_VERSION`. The Windows development machine runs container validation in GitHub Actions.

Go 1.27.1; SQLite; embedded UI; MIT license. Optional process settings: `APP_LISTEN` and `APP_DATA_DIR`. The service does not load `.env`. [Architecture/API](docs/architecture.md) · [Verification](docs/verification.md) · [PAT/OAuth](docs/auth-comparison.md).
