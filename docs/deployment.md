# Deployment

## Docker / 1Panel

Use the published image and [Compose file](../docker-compose.yml). In 1Panel, create a container orchestration and paste the Compose configuration. No password/PAT environment variables are needed.

1. Start with `docker compose up -d`.
2. Configure an HTTPS reverse proxy.
3. Open the site through a controlled connection and complete first-run setup before public access. Set the public origin, such as `https://rss.example.com`.
4. Connect the PAT in Settings; add subscriptions or manual tasks.

Only `APP_LISTEN` and `APP_DATA_DIR` remain optional process settings. Container defaults: `0.0.0.0:8080`, `/data`. Local defaults: `127.0.0.1:8080`, `data`.

## Reverse proxy

For a host proxy, use `http://127.0.0.1:8080`. For a container proxy, attach both services to the same Docker network and use `http://pikpak-rss-manager:8080`.

```nginx
location / {
    proxy_pass http://127.0.0.1:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_read_timeout 130s;
}
```

Save the public origin in Settings and access the UI from that origin. Preserve Host for request validation. Cookies use Secure for HTTPS origins.

## Update

```sh
docker compose pull
docker compose up -d
docker compose logs --tail=50
```

The reissued v1.0.0 removes resource uniqueness through schema version 3. Pull again even if you already use the v1.0.0 tag. Back up the complete data volume before upgrading; returning to the earlier v1.0.0 build requires restoring its pre-upgrade backup.

The latest v1.0.0 republication also cleans empty staging folders after new tasks complete. Grant the PAT **Manage files** and **Cloud Download**; if an existing PAT cannot be edited, create a replacement and save it in Settings. Cleanup uses Trash, preserves backups/unfinished tasks and leaves legacy staging untouched. Missing cleanup permission retains staging with a warning without failing completed downloads.

Pin an available version tag or digest to control updates. Never use `docker compose down -v` for updates. Existing subscriptions, encrypted PATs, jobs and staging IDs are retained. Upgrading versions predating web setup requires creating a new administrator password; externally configured PATs must be entered in Settings.

## Backup / restore

Stop the service and back up the entire volume, including `manager.db`, WAL/SHM files and `secret.key`. The database contains encrypted PATs and private URLs; losing the key makes them unrecoverable.

```sh
docker compose stop
docker volume ls
docker run --rm -v <actual-volume-name>:/data:ro -v "$PWD":/backup \
  alpine:3.23 tar -czf /backup/pikpak-rss-backup.tar.gz -C /data .
docker compose start
```

To restore, stop the service, extract the full backup into its data volume, set ownership to `65532:65532`, and restart. Protect the backup and key. Named volumes inherit the image's /data ownership; bind mounts require matching host permissions.

## Image publishing

Actions validates tests, race detection, vet, Docker startup and persistence, then publishes amd64/arm64 images on main/tag pushes using GITHUB_TOKEN. Release tags must match `cmd/pikpak-rss-manager/VERSION`.

For forks, update the Compose image and source labels. Set the GHCR package to Public after its first publication and verify anonymous pulls; public repository visibility does not make a package public.
