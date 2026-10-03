# 部署與維護

## 首次網頁設定

直接執行 `docker compose up -d`，無須建立 `.env` 或提供管理密碼／PAT。開啟管理網址後：

1. 設定並確認管理密碼，僅要求非空，沒有長度或字元限制。
2. 確認網站網址（預填目前來源）；內網 RSS 預設停用，需要時勾選。
3. 建立管理員後自動登入，到「系統設定」驗證並綁定 PikPak PAT。

管理密碼只保存 Argon2id 加鹽雜湊，PAT 以 AES-256-GCM 加密。初始化完成後入口關閉，重啟不會重新設定密碼。網站網址及內網 RSS 選項可於登入後修改。先在可控的連線中完成初始化，再開放外部存取；首次設定介面會讓第一位完成設定的人建立管理員。

程序層級只保留以下可選環境設定，映像已內建適當預設，不需要 `.env`：

| 設定 | 預設／用途 |
|---|---|
| `APP_LISTEN` | 本機 `127.0.0.1:8080`；映像內 `0.0.0.0:8080` |
| `APP_DATA_DIR` | 本機 `data`；映像內 `/data` |

舊 `APP_ADMIN_PASSWORD`、`APP_ADMIN_PASSWORD_FILE`、`PIKPAK_TOKEN`、`PIKPAK_TOKEN_FILE`、`APP_PUBLIC_URL` 與 `APP_ALLOW_PRIVATE_FEEDS` 均不再讀取。舊 `.env` 保留但不載入。升級後先在網頁重新設定管理密碼；既有訂閱、任務與已加密保存的 PAT 保留。僅存在外部設定中的 PAT 請重新貼到網頁綁定。初始化前背景排程暫停。請使用本版 Compose，移除舊編排中強制要求密碼的插值設定。

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

在首次設定／系統設定填入 `https://你的網域` 作為網站網址，用同一來源開啟面板，確保 Origin 驗證與 Secure Cookie 正常。服務不依賴轉送標頭推斷登入 IP；大量錯誤登入可能共用代理來源的限流。僅需要私人存取時也可使用 SSH tunnel 連到 localhost。

## 更新、固定版本與備份

`latest` 隨 main 更新；希望固定版本，可將 image 改為 `:v0.4.0` 或已驗證 digest。更新：

```sh
docker compose pull
docker compose up -d
docker compose logs --tail=50
```

資料卷內包含加密 SQLite 與金鑰，更新不要加 `-v`。備份應先停止程序，避免只複製 DB 主檔漏掉 WAL；備份保存於你可控的加密位置。

v0.2.0 新增資料夾選擇／建立與可選的 Regex 尋找／替換。更新後既有訂閱仍使用原本命名範本；新增訂閱預設不重命名。重新啟動服務並重新整理瀏覽器即可使用新介面。

v0.3.0 新增 RSS／種子檔名範例、清楚顯示檔名字元調整，並將建立資料夾改為按鈕開啟小視窗。新任務暫存在訂閱目標目錄內；選根目錄時仍放根目錄。升級前任務與既有暫存資料沿用原位置，不自動搬移或刪除。本次未改動 PAT 綁定方式。[OAuth 比較](auth-comparison.md)。

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
