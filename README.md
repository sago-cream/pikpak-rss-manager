# PikPak RSS Manager

Self-hosted RSS/Atom subscriptions and offline downloads through the official PikPak MCP.

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
| **Manage files** | Includes reading/writing, browsing/creating folders and renaming files. |
| **Cloud Download** | Creates offline download tasks. |

## Use

- **Subscriptions:** choose an RSS URL, destination and interval. First check establishes a baseline. **Backfill** selects whole torrents. Repeated explicit downloads are allowed.
- **Offline tasks:** submit a Magnet, torrent URL or HTTP/HTTPS download URL. **Delete task** cancels incomplete downloads while preserving files. **Clear completed tasks** removes completed records only.
- **Renaming:** disabled by default. Regex replacement supports `$1`, `${name}` and `$$`. Nonmatches keep their names. Select a torrent filename to preview. Existing template rules remain supported.
- **Destinations:** browse/create folders or enter a path. Reselect saved folders after switching accounts.

New tasks download directly to the destination. Optional renaming preserves torrent directories and attachments. Existing staged jobs retain their workflow. Rejected renames and uncertain submissions require review. Authorization/quota errors pause jobs. Update the PAT or check the connection, then resume affected tasks.

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
