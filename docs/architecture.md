# Architecture and API

One Go process, one administrator, one active PikPak account and one SQLite data volume. HTML/CSS/JavaScript are embedded. No frontend build step or local media transfer.

| Module | Responsibility |
|---|---|
| internal/web | UI, sessions, CSRF, language catalog, API |
| internal/feed | RSS/Atom, bounded torrent metadata, normalized infohashes |
| internal/rename | Shared regex/template renderer and filename normalization |
| internal/pikpak | Typed official MCP adapter and folder operations |
| internal/worker | Durable queue, reconciliation, file actions and bounded retry |
| internal/store | SQLite migrations, baseline, repeat-download migration, encryption |

## Workflow

Subscriptions establish a baseline on first check. Backfill is explicit. Manual tasks have subscription_id 0 and never modify feed baselines. Torrent tasks retain normalized infohashes as resource identifiers. Direct download URLs use an exact-URL SHA-256 key. Resource identity does not suppress repeated explicit downloads. Automatic checks skip previously seen feed fingerprints.

The worker saves queued → submitting before the cloud request, then stores the task ID and polls downloading → organizing → complete. New jobs persist download_mode: direct and submit to the destination ID. Unconfirmed submissions become submission_unknown. Without a task ID they require manual verification and never scan the destination or resubmit. Official file IDs are retained from submission and polling. Completed download-only jobs need no file ID. Renaming requires official file IDs. Authorization/quota failures pause jobs. Transient failures back off from 30 seconds to two hours, with review after six attempts.

Deletion shares the worker/account lock with submission and file actions. Incomplete jobs with a task ID are checked against their owning account and canceled once through optional official `task_rm`, explicitly setting `delete-files:false`. Failure preserves the local job. An absent remote task permits deletion. Unknown submissions without an ID require manual cancellation and explicit `local_only:true`. This flag cannot bypass cancellation of a known task. Completed deletion makes no cloud call. Local deletion atomically removes jobs/file actions and stores ID tombstones, retaining feed baselines and event history. Tombstones prevent repeated confirmation requests from recreating deleted jobs. A new explicit download still receives a new ID. Clearing completed jobs includes records outside the latest-200 list and stops their pending cleanup retries.

Direct jobs rename primary files in place inside the official returned file/folder ID, preserving torrent structure and auxiliary filenames/locations. File actions persist original/requested names, parent IDs and renaming intent before each call, then read back actual_name (including server suffixes). A lost response is reconciled by ID without repeating the rename. An unchanged name requires review. Direct jobs do not create staging, backup or attachment folders, scan destination contents, move, replace, trash or clean files. Existing jobs without download_mode keep the staged workflow and saved IDs. Backfill preview reads one RSS snapshot and each distinct torrent URL once, with bounded concurrency and metadata/output budgets. Session/account-bound plans expire after 15 minutes. Confirmations validate the subscription and destination again, then atomically queue selected torrents using stable job IDs. New backfill confirmations queue direct jobs without overwrite. Old clients may send overwrite, but it is ignored. Existing overwrite jobs retain their persisted backup plans and move originals into `<staging>/_Replaced/<new-file-ID>`. Regex replacement works on actual filenames. Legacy episode fallback is permitted only for one primary file. Only legacy staged jobs move auxiliary files under `_附件/<jobID>`.

## Limits and security

Only existing staged jobs with cleanup ownership run empty-folder cleanup. After such a task completes, the worker moves only verified empty folders in its staging subtree to Trash, then removes the empty `_PikPak-RSS-Staging` container. `_Replaced`, nonempty folders, other jobs and legacy records without cleanup metadata remain intact. Cleanup ownership and up to three attempts persist in the job payload. Completed cleanup jobs are scheduled separately from downloads. Cleanup errors generate a warning without resubmitting, repeating file actions or pausing download credentials. The official `rm` tool requires Manage files permission and is optional at connection time. Permanent deletion is never called.

- Feed/torrent metadata: 2 MiB per resource. Full sample reads: one feed snapshot, each distinct torrent once, at most five concurrent requests, 32 MiB total metadata, 10,000 filenames/8 MiB output, 90-second overall and 15-second torrent timeouts. Partial successes are retained. The cursor API remains available.
- Preview reads metadata only. It does not submit tasks or change baselines. The shared renderer returns raw_name, normalized name and warnings.
- Selected folder IDs carry opaque account references. The server revalidates IDs/account ownership. Folder creation is sent once. Uncertain results require review.
- First-run password: nonempty, salted Argon2id verifier. PATs/private URLs: AES-GCM with a persistent secret.key. Credentials are never returned or logged. No external password/PAT sources or .env loading.
- Sessions: 12 hours, memory only. Mutations require matching CSRF cookie/header and same-origin checks. HTTP-only/SameSite cookies, login throttling and CSP are enabled.
- UI language is a browser preference. Only application literals and message fields are translated. Names, paths, URLs, regex and filenames remain intact.

## JSON API

Except health, session, setup and login, endpoints require an authenticated session. POST/PUT/DELETE require pp_csrf and X-CSRF-Token. Setup/login also require CSRF.

| Endpoint | Purpose |
|---|---|
| GET /healthz | Database health/version |
| GET /api/session | Setup/login status and CSRF |
| POST /api/setup | One-time password, public_url, allow_private_feeds |
| POST /api/login, /api/logout | Session lifecycle |
| GET/POST /api/subscriptions | List/create |
| PUT/DELETE /api/subscriptions/{id} | Update/delete |
| POST /api/subscriptions/{id}/check | Check new items. Backfill requires selection |
| GET /api/pikpak/folders?parent_id=…&token=… | Browse with account_ref |
| POST /api/pikpak/folders | Create with parent_id, name, account_ref |
| POST /api/subscriptions/{id}/backfill/preview | Read downloadable torrents and filenames. No queue/baseline/cloud mutations |
| POST /api/subscriptions/{id}/backfill | Confirm token and selected IDs. Atomically queue direct downloads (overwrite is ignored) |
| POST /api/feeds/samples | url, subscription_id, all:true. Metadata preview |
| POST /api/rules/preview | rule, title, filename. Shared renderer |
| GET /api/jobs, /api/jobs/{id} | Latest 200 jobs / file actions |
| POST /api/jobs | url, destination, optional name/source_type/destination_id/destination_account_ref. 201 queued. Repeated sources allowed |
| POST /api/jobs/{id}/retry | Resume/reconcile safely |
| DELETE /api/jobs/{id} | Cancel incomplete PikPak task, preserve files, remove local record. Optional local_only for unknown submissions without a task ID |
| DELETE /api/jobs/completed | Remove all completed records/file actions, preserve cloud files. Returns deleted count |
| GET /api/events | Last 150 entries, 30-day retention |
| GET/POST /api/settings/app | Site URL/private-feed policy |
| POST /api/settings/password | current_password, new_password, confirm_password. Revoke all sessions on success |
| GET/POST /api/settings/pikpak | Status/write-only PAT |
| POST /api/settings/pikpak/check | Reconnect |

Manual tasks detect Magnet, `.torrent` and HTTP/HTTPS links automatically. Existing API clients can still specify source_type (`torrent` or `url`) and name. Omitted names use Magnet dn, torrent metadata or the URL path/host, with a resource-key fallback. Direct URLs are submitted to PikPak without fetching their content locally. Torrent URLs are resolved using the same metadata size/timeout/private-network policy as feeds. Manual tasks preserve filenames. Settings and cloud processing share a lock to prevent account switches between destination validation and queue insertion.
