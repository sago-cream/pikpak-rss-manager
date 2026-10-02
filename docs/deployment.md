# 部署與維護

## 環境設定

| 設定 | 預設／用途 |
|---|---|
| `APP_ADMIN_PASSWORD` | 必填，12–72 位元組，沒有預設密碼 |
| `APP_ADMIN_PASSWORD_FILE` | 私密檔案，優先於密碼環境變數 |
| `APP_LISTEN` | 本機 `127.0.0.1:8080`；映像內 `0.0.0.0:8080` |
| `APP_DATA_DIR` | 本機 `data`；映像內 `/data` |
| `APP_PUBLIC_URL` | 代理後精確的網站來源，例如 `https://rss.example.com`，不含路徑 |
| `APP_ALLOW_PRIVATE_FEEDS` | `false`；允許內網 RSS 才設 `true` |
| `PIKPAK_TOKEN_FILE` | 可選的私密 PAT 檔案，優先於環境／面板 |
| `PIKPAK_TOKEN` | 可選的環境 PAT，優先於面板 |

環境變數優先於開發用 `.env`；檔案來源優先於對應值。Docker 映像不包含 `.env`。不要把整份本機開發 `.env` 放進 Actions secret 或映像。

## 反向代理

Compose 將連接埠綁定 `127.0.0.1:8080`。若 1Panel 的 OpenResty 在主機上，使用此上游；若代理在容器內，`127.0.0.1` 指向代理自己的容器，請將兩者接上共同的 Docker 網路。

```yaml
# 在既有 Compose 的 service 內加入此網路；proxy-net 須先存在，且代理也接上。
services:
  pikpak-rss-manager:
    networks: [proxy-net]
networks:
  proxy-net:
    external: true
```

上游改為 `http://pikpak-rss-manager:8080`。以實際代理環境選擇其中一種連線方式。Nginx／OpenResty 範例（網域與憑證由 1Panel 管理）：

```nginx
location / {
    proxy_pass http://127.0.0.1:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_read_timeout 130s;
}
```

同時設定 `APP_PUBLIC_URL=https://你的網域`，用同一來源開啟面板，確保 Origin 驗證與 Secure Cookie 正常。服務不依賴轉送標頭推斷登入 IP；大量錯誤登入可能共用代理來源的限流。僅需要私人存取時也可使用 SSH tunnel 連到 localhost。

## 使用私密檔案提供 PAT

預設建議面板綁定。若使用檔案：

```yaml
services:
  pikpak-rss-manager:
    environment:
      PIKPAK_TOKEN_FILE: /run/secrets/pikpak_pat
    secrets: [pikpak_pat]
secrets:
  pikpak_pat:
    file: ./secrets/pikpak-pat.txt
```

將此片段合併至完整 Compose，保留密碼等既有設定。保護本機 secrets 目錄，PAT 不放入 Git；檔案必須可供容器 UID 65532 讀取，並符合主機的安全權限安排。外部來源模式不允許面板更改權杖。

## 更新、固定版本與備份

`latest` 隨 main 更新；希望固定版本，可將 image 改為 `:v0.1.0` 或已驗證 digest。更新：

```sh
docker compose pull
docker compose up -d
docker compose logs --tail=50
```

資料卷內包含加密 SQLite 與金鑰，更新不要加 `-v`。備份應先停止程序，避免只複製 DB 主檔漏掉 WAL；備份保存於你可控的加密位置。

```sh
docker compose stop
# 用 docker volume ls 找到這個編排的實際資料卷名稱；通常含編排名稱前綴。
docker volume ls
docker run --rm -v <實際資料卷名稱>:/data:ro -v "$PWD":/backup \
  alpine:3.23 tar -czf /backup/pikpak-rss-backup.tar.gz -C /data .
docker compose start
```

還原：停止服務，把完整備份解壓至原資料卷，保留 `secret.key`，檔案／目錄的擁有者為 `65532:65532`；重新啟動並檢查授權。勿只還原 DB 而遺失金鑰。容器使用 non-root、唯讀 root filesystem、移除 capabilities；資料卷和 `/tmp` 為可寫位置。

只有需要 bind mount 時才自訂 host path；先建立目錄並給 UID 65532 適當權限。預設 named volume 會繼承映像的 `/data` 擁有權，通常更省事。

## 發布到你自己的 GHCR

Fork 後 Actions 自動使用 `ghcr.io/${{ github.repository }}` 作為發布目的地。同步更新 Compose、Dockerfile 的 source label、README 中的映像名稱。第一次推送後，在 GitHub 套件設定把套件設為 Public；公開 repository 不代表套件已公開。完成後應在未登入 GHCR 的機器實際拉取。

Actions 使用倉庫的 `GITHUB_TOKEN`（`packages: write`），不需要你的 PikPak PAT。必要的 Go／Docker 驗證在發布前執行，多架構 manifest 在發布後核對。[官方 GHCR 文件](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)。
