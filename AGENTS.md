# PikPak RSS Manager

## Purpose and architecture
Personal self-hosted Go service. Use the official hosted PikPak MCP with PAT authorization. Do not introduce rclone, a PikPak CLI runtime dependency, an LLM, MySQL, Redis, or a frontend build pipeline.
Keep HTTP/UI in `internal/web`, RSS and torrent metadata in `internal/feed`, naming rules in `internal/rename`, the typed MCP adapter in `internal/pikpak`, durable workflow in `internal/worker`, and SQLite migrations in `internal/store/migrations`.
Use Go 1.27.1, native JavaScript/CSS and embedded templates/static assets. The UI and primary documentation use Traditional Chinese. MIT license.

## Product decisions
- One administrator and one active PikPak account; no default administrator password. The user explicitly removed password length and character restrictions; require only a nonempty configured password.
- Every subscription owns its destination, interval, season, regex and naming template.
- Establish an initial feed baseline by default; backfill is explicit.
- Normalize infohashes and deduplicate per PikPak account.
- Download on PikPak, then rename/move by file ID. Never transfer media through this service.
- Parse multiple files independently. RSS episode fallback is allowed only for one primary file. Keep ambiguous files and name collisions for review; never overwrite existing files.
- Persist task IDs and file action progress. Reconcile ambiguous submissions instead of automatically submitting another task.
- Pause authentication/quota errors and use bounded backoff for transient failures.

## Commands and verification
`go test ./...`, `go vet ./...`, `go build ./cmd/pikpak-rss-manager`.
Linux CI additionally runs `go test -race ./...` and container startup/persistence smoke tests.
The development machine has no Docker. Use GitHub Actions for Docker validation and multiarch publishing rather than installing Docker.
Meaningful tests must cover RSS/Atom and torrent parsing, independent naming rules, baseline/deduplication, restart recovery, partial actions, uncertain submissions, auth/quota/rate failures, HTTP authentication and CSRF.
Document actual results in `docs/verification.md`; never claim a mocked test proves a real PikPak operation.

## Secrets and external actions
The user-provided `.env` contains `PIKPAK_TOKEN`. Never print, commit, embed in a build, or upload it to GitHub/Actions. Do not read it into tool output. Preserve the file and unrelated user settings.
Ignore databases, private settings, keys, local credentials and binaries before initializing Git. Check staged paths and secret leakage before every publication.
UI credentials are write-only and encrypted with AES-GCM; keep the key in the persistent data directory with restrictive permissions. Log no credentials, authorization headers or sensitive URL queries.
User authorization already covers creating the public `wade00754/pikpak-rss-manager` repository, pushing this implementation, publishing GHCR images, and making the package public. Do not ask again for these same actions. It does not cover deploying to a VPS or altering unrelated repositories.
Real PikPak tests may create offline tasks and rename/move only inside `_pikpak-rss-manager-test/<run-id>` created by this project. Use small Public Domain/Creative Commons fixtures; preserve existing account content. Never permanently delete content. The PAT stays on the local development machine.

## Delivery
Provide non-root multistage Docker images for linux/amd64 and linux/arm64, main/tag publishing to GHCR with GITHUB_TOKEN, a compose file using the published image, and 1Panel/reverse proxy/update/backup instructions. Confirm anonymous access after first GHCR publication; public repository visibility does not make a package public automatically.
Routine implementation choices within these principles are authorized. Clearly record external blockers and incomplete validation.
