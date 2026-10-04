# PikPak RSS Manager

A self-hosted Go service for PikPak RSS/Atom subscriptions and offline downloads. Uses the official PikPak MCP with PAT authentication and per-subscription naming rules.

## Start

Download [docker-compose.yml](docker-compose.yml), then run:

```sh
docker compose up -d
```

Open the site, create an administrator password, confirm the public URL, and enter a PikPak PAT in **Settings**.

See the [deployment guide](docs/deployment.md) for 1Panel setup, HTTPS reverse proxy configuration and process settings.

### PAT permissions

[Create a PAT](https://mypikpak.com/en-US/help-center/connected_apps/personal_access_tokens/create_personal_access_token) with:

| Permission | Purpose |
|---|---|
| **Read & write files** | Browses/creates folders and renames files. |
| **Cloud Download** | Creates offline download tasks. |

Manage files and permanent deletion are not required. [Official permission reference](https://mypikpak.com/en-US/help-center/connected_apps/managing_connected_apps/connected_app_permissions).

## Use

- **Subscriptions:** choose an RSS URL, destination and interval. First check establishes a baseline. **Backfill** selects whole torrents. Repeated explicit downloads are allowed.
- **Offline tasks:** submit a Magnet, torrent URL or HTTP/HTTPS download URL. **Delete task** cancels incomplete downloads while preserving files. **Clear completed tasks** removes completed records only.
- **Renaming:** disabled by default. Regex replacement supports `$1`, `${name}` and `$$`. Nonmatches keep their names. Select a torrent filename to preview.
- **Destinations:** browse/create folders or enter a path. Reselect saved folders after switching accounts.

Tasks download directly to the destination. Optional renaming preserves torrent directories and attachments. Rejected renames and uncertain submissions require review. Authorization/quota errors pause jobs. Update the PAT or check the connection, then resume affected tasks.

Current release: [v2.0.0](https://github.com/wade00754/pikpak-rss-manager/releases/tag/v2.0.0). Image: `ghcr.io/wade00754/pikpak-rss-manager:v2.0.0` (amd64/arm64). Compose tracks `latest`.

## Update

Back up the full data volume, including `secret.key`, then run:

```sh
docker compose pull
docker compose up -d
```

v2.0.0 removes staged jobs and template naming compatibility. Do not reuse databases containing those jobs or rules. Downgrades require restoring a pre-upgrade backup. [Backup and restore](docs/deployment.md#backup--restore).

## Development

Requires Go 1.27.1, GNU Make and Node.js. Run from the repository root:

```sh
make dev
make verify
```

Use `make help` for individual targets. [Test commands and verification](docs/verification.md) · [Architecture/API](docs/architecture.md) · [PAT/OAuth](docs/auth-comparison.md).
