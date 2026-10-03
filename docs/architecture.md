# Architecture and API

One Go process, one administrator, one active PikPak account and one SQLite data volume. HTML/CSS/JavaScript are embedded; no frontend build step or local media transfer.

| Module | Responsibility |
|---|---|
| internal/web | UI, sessions, CSRF, language catalog, API |
| internal/feed | RSS/Atom, bounded torrent metadata, normalized infohashes |
| internal/rename | Shared regex/template renderer and filename normalization |
| internal/pikpak | Typed official MCP adapter and folder operations |
| internal/worker | Durable queue, reconciliation, file actions and bounded retry |
| internal/store | SQLite migrations, baseline, per-account deduplication, encryption |

## Workflow

Subscriptions establish a baseline on first check; backfill is explicit. Manual tasks have subscription_id 0 and never modify feed baselines. Torrent tasks share normalized infohash deduplication with RSS; direct download URLs use an exact-URL SHA-256 resource key.

The worker saves queued → submitting before the cloud request, then stores the task ID and polls downloading → organizing → complete. Unconfirmed submissions become submission_unknown and reconcile the dedicated staging folder without submitting again. Authorization/quota failures pause jobs; transient failures back off from 30 seconds to two hours, with review after six attempts.

New staging folders live under the destination; persisted staging IDs always retain their location. File actions persist before rename/move and resume by file ID. Name collisions retain originals for review. Regex replacement works on actual filenames; legacy episode fallback is permitted only for one primary file. Auxiliary files retain their names/subdirectories under `_附件/<jobID>`.

## Limits and security

- Feed/torrent metadata: 2 MiB per resource. Full sample reads: one feed snapshot, each distinct torrent once, at most five concurrent requests, 32 MiB total metadata, 10,000 filenames/8 MiB output, 90-second overall and 15-second torrent timeouts. Partial successes are retained. The cursor API remains available.
- Preview reads metadata only; it does not submit tasks or change baselines. The shared renderer returns raw_name, normalized name and warnings.
- Selected folder IDs carry opaque account references. The server revalidates IDs/account ownership. Folder creation is sent once; uncertain results require review.
- First-run password: nonempty, salted Argon2id verifier. PATs/private URLs: AES-GCM with a persistent secret.key. Credentials are never returned or logged. No external password/PAT sources or .env loading.
- Sessions: 12 hours, memory only. Mutations require matching CSRF cookie/header and same-origin checks. HTTP-only/SameSite cookies, login throttling and CSP are enabled.
- UI language is a browser preference. Only application literals and message fields are translated; names, paths, URLs, regex and filenames remain intact.

## JSON API

Except health, session, setup and login, endpoints require an authenticated session. POST/PUT/DELETE require pp_csrf and X-CSRF-Token; setup/login also require CSRF.

| Endpoint | Purpose |
|---|---|
| GET /healthz | Database health/version |
| GET /api/session | Setup/login status and CSRF |
| POST /api/setup | One-time password, public_url, allow_private_feeds |
| POST /api/login, /api/logout | Session lifecycle |
| GET/POST /api/subscriptions | List/create |
| PUT/DELETE /api/subscriptions/{id} | Update/delete |
| POST /api/subscriptions/{id}/check | Check; optional backfill |
| GET /api/pikpak/folders?parent_id=…&token=… | Browse with account_ref |
| POST /api/pikpak/folders | Create with parent_id, name, account_ref |
| POST /api/feeds/samples | url, subscription_id, all:true; metadata preview |
| POST /api/rules/preview | rule, title, filename; shared renderer |
| GET /api/jobs, /api/jobs/{id} | Latest 200 jobs / file actions |
| POST /api/jobs | url, destination, optional name/source_type/destination_id/destination_account_ref; 201 queued, 409 duplicate |
| POST /api/jobs/{id}/retry | Resume/reconcile safely |
| GET /api/events | Last 150 entries, 30-day retention |
| GET/POST /api/settings/app | Site URL/private-feed policy |
| POST /api/settings/password | current_password, new_password, confirm_password; revoke all sessions on success |
| GET/POST /api/settings/pikpak | Status/write-only PAT |
| POST /api/settings/pikpak/check | Reconnect |

Manual tasks detect Magnet, `.torrent` and HTTP/HTTPS links automatically. Existing API clients can still specify source_type (`torrent` or `url`) and name. Omitted names use Magnet dn, torrent metadata or the URL path/host, with a resource-key fallback. Direct URLs are submitted to PikPak without fetching their content locally. Torrent URLs are resolved using the same metadata size/timeout/private-network policy as feeds. Manual tasks preserve filenames. Settings and cloud processing share a lock to prevent account switches between destination validation and queue insertion.
