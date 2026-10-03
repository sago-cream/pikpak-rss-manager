# PikPak RSS Manager

Self-hosted RSS/Atom subscriptions and offline downloads through the official PikPak MCP. Go 1.27.1, SQLite, embedded UI, MIT license. Media stays in PikPak.

## Start

Download [docker-compose.yml](docker-compose.yml), then run:

```sh
docker compose up -d
```

Open the site, create an administrator password, confirm the public URL, and connect a PikPak PAT in **Settings**. There is no default password. Passwords and PATs are configured through the UI; the service does not load `.env`.

Release image: `ghcr.io/wade00754/pikpak-rss-manager:v1.0.0` (linux/amd64 and linux/arm64). The Compose file tracks `latest`; set its image tag to `v1.0.0` to pin this release. Compose binds to `127.0.0.1:8080`; use a reverse proxy for remote access. [Deployment, 1Panel, updates and backups](docs/deployment.md).

## Use

- **Subscriptions:** add an RSS URL, destination and interval. First check establishes a baseline; **Backfill** lists torrent filenames for selection, then asks to confirm downloading and replacing matching destination files. Multi-file torrents download in full.
- **Offline tasks → Add task:** submit a Magnet, torrent URL or direct HTTP/HTTPS download URL, choose a destination, and retain original filenames.
- **Folders:** browse/create folders or enter a path. Saved folder IDs belong to the connected account; reselect them after switching accounts.
- **Renaming:** disabled by default for new subscriptions. Enable regex replacement and select a torrent filename for automatic preview. Go RE2 supports `$1`, `${name}` and `$$`; nonmatches keep their names. Existing template rules remain compatible.
- **Language:** choose Traditional Chinese or English in the top bar, setup or login page. The choice is saved in the browser; switching reloads the page.
- **Settings → Change administrator password:** enter the current password and confirm a nonempty new password. All devices must sign in again after saving.

The theme switch on setup, login and management pages remembers your light/dark preference. Before a choice is saved, it follows the system theme.

Explicit downloads may repeat a previously downloaded source. Jobs resume after restarts, and retries of the same backfill confirmation do not create another job. Unconfirmed submissions require reconciliation. Confirmed backfills replace matching files only after completion, moving originals to the task’s `_Replaced` backup folder; ordinary jobs and folder/name ambiguities remain subject to review. New jobs stage under `<destination>/_PikPak-RSS-Staging/<jobID>`; existing staging IDs are preserved.

PATs are encrypted with AES-GCM; passwords use salted Argon2id verifiers. Authorization or quota failures pause jobs. Renew the PAT or check the connection, then resume affected tasks. [PAT instructions](https://mypikpak.com/en-US/help-center/connected_apps/personal_access_tokens/create_personal_access_token). OAuth is not implemented; [comparison](docs/auth-comparison.md).

## Development

```sh
go run ./cmd/pikpak-rss-manager
go test ./...
go vet ./...
go build ./cmd/pikpak-rss-manager
node scripts/test-i18n.cjs
```

Optional process settings: `APP_LISTEN` and `APP_DATA_DIR`. Windows helpers: `scripts/dev.ps1`, `scripts/verify.ps1`. [Architecture/API](docs/architecture.md) · [Verification](docs/verification.md).
