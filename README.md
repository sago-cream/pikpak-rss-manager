# PikPak RSS Manager

Self-hosted RSS/Atom subscriptions and offline downloads through the official PikPak MCP. Media stays in PikPak.

## Start

Download [docker-compose.yml](docker-compose.yml), then run:

```sh
docker compose up -d
```

Open the site, create an administrator password, confirm the public URL, and enter a PikPak PAT in **Settings**. There is no default password.

Image: `ghcr.io/wade00754/pikpak-rss-manager:v1.2.0` (`linux/amd64`, `linux/arm64`). Compose tracks `latest`; change its image tag to `v1.2.0` to pin this release. It binds to `127.0.0.1:8080`; use an HTTPS reverse proxy for remote access. [Deployment and 1Panel](docs/deployment.md) · [Release notes](docs/releases/v1.2.0.md).

[Create a PAT](https://mypikpak.com/en-US/help-center/connected_apps/personal_access_tokens/create_personal_access_token) with **Manage files** and **Cloud Download** permissions.

## Use

- **Subscriptions:** choose an RSS URL, destination and interval. First check establishes a baseline. **Backfill** selects whole torrents; repeated explicit downloads are allowed.
- **Offline tasks:** submit a Magnet, torrent URL or HTTP/HTTPS download URL. **Delete task** cancels incomplete downloads while preserving files. **Clear completed tasks** removes completed records only.
- **Renaming:** disabled by default. Regex replacement supports `$1`, `${name}` and `$$`; nonmatches keep their names. Select a torrent filename to preview. Existing template rules remain supported.
- **Destinations:** browse/create folders or enter a path. Reselect saved folders after switching accounts.

New tasks download directly to the destination; optional renaming preserves torrent directories and attachments. Existing staged jobs retain their workflow. Rejected renames and uncertain submissions require review. Authorization/quota errors pause jobs; update the PAT or check the connection, then resume affected tasks.

## Update

Back up the full data volume, including `secret.key`, then run:

```sh
docker compose pull
docker compose up -d
```

Downgrading with direct jobs requires restoring a pre-upgrade backup. [Backup and restore](docs/deployment.md#backup--restore).

## Development

Requires Go 1.27.1, GNU Make and Node.js. Run from the repository root:

```sh
make dev
make verify
```

Use `make help` for individual targets. [Test commands and verification](docs/verification.md) · [Architecture/API](docs/architecture.md) · [PAT/OAuth](docs/auth-comparison.md).

Optional process settings: `APP_LISTEN`, `APP_DATA_DIR`. Configure credentials through the UI; the service does not load `.env`. SQLite, embedded UI, MIT license.
