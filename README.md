# PikPak RSS Manager

輕量的個人自託管 RSS／Atom 追蹤與 PikPak 雲端整理服務。每筆訂閱各自設定目標目錄、Regex 與命名範本，透過官方 PAT＋MCP 建立離線任務，再依檔案 ID 重命名與移動。

Go 1.27.1、SQLite、原生 JavaScript/CSS；單一執行檔，無需 rclone、PikPak CLI、LLM、MySQL、Redis 或前端建置工具。MIT 授權。

## 功能

- 繁體中文管理面板、手機介面、單一管理員登入。
- RSS／Atom 訂閱新增、編輯、刪除、停用、手動檢查及補抓。
- 每筆獨立的季數、具名 Regex、命名範本與解析預覽。
- Magnet v1/v2 與小型 `.torrent` 中繼資料解析；同一帳號按正規化 infohash 去重。
- 離線任務與逐檔操作持久化，重啟後接續，提交結果不明時先核對。
- 多檔種子逐檔解析；衝突或無法判定時保留原名，附屬檔案保持完整。
- PAT 可由面板加密保存、環境變數或私密檔案提供；不回傳或記錄權杖。
- GHCR 公開映像：`ghcr.io/wade00754/pikpak-rss-manager:latest`，支援 `linux/amd64`、`linux/arm64`。

![訂閱管理介面](docs/screenshots/subscriptions.jpg)

## Docker Compose 快速開始

將 [docker-compose.yml](docker-compose.yml) 放進一個目錄。在同一目錄建立 `.env`，填入你自己的管理密碼：

```dotenv
APP_ADMIN_PASSWORD=請換成你自己的12至72位元組密碼
# 使用 HTTPS 反向代理時設定精確的網站來源，不包含路徑。
APP_PUBLIC_URL=https://rss.example.com
```

```sh
docker compose up -d
docker compose ps
```

映像預設只將面板映射到主機 `127.0.0.1:8080`。同一主機的反向代理可連線至 `http://127.0.0.1:8080`；代理本身若在容器內，請依 [部署文件](docs/deployment.md) 配置共同 Docker 網路。初次登入後，前往「授權設定」綁定 PikPak PAT。

### 1Panel「容器 → 編排」

1. 建立編排，名稱可用 `pikpak-rss-manager`，貼上本專案的 Compose。
2. 在編排的環境設定／`.env` 設定 `APP_ADMIN_PASSWORD`；如果該版本的 1Panel 沒有此欄位，將 Compose 的 `${APP_ADMIN_PASSWORD:?...}` 換成**你自己的密碼**。包含 `:`、`#`、`$` 等字元時注意 YAML 與 Compose 插值規則（字面 `$` 使用 `$$`）。
3. 使用網域反向代理時，同時設定 `APP_PUBLIC_URL`，例如 `https://rss.example.com`。
4. 啟動編排，確認容器健康。反向代理的上游為主機的 `127.0.0.1:8080`，或共同網路中的 `pikpak-rss-manager:8080`。
5. 登入 → 授權設定 → PAT 綁定 → 新增訂閱 → 測試命名 → 儲存。

PikPak PAT 不需要放進 GitHub 或 Actions。Compose 使用面板綁定，會把加密權杖與金鑰保存在持久化資料卷。

## 授權設定

在 PikPak 官方「存取與整合／Access & Integrations」建立 PAT，授予帳號資訊、檔案讀取、寫入／管理與雲端下載的必要權限；實際權限名稱以官方介面為準。本服務不使用分享、永久刪除或邀請工具。

PAT 來源的優先順序為 `PIKPAK_TOKEN_FILE` → `PIKPAK_TOKEN` → 面板加密保存的權杖。使用外部來源時，面板不能覆寫 PAT；更新外部設定後需重新啟動。失效或配額不足會暫停相關任務；更新授權、重新檢查連線後，可逐一接續暫停的任務。更換帳號會暫停舊帳號任務。

PAT 可設定 30 天至一年有效期；已連接應用共用各流量維度月配額的 25%。RSS、種子中繼資料與管理 API 仍會使用少量 VPS 網路流量；影片內容不經 VPS。離線能否秒傳取決於 PikPak 快取、來源與配額。[官方 PAT 說明](https://mypikpak.com/en-US/help-center/connected_apps/personal_access_tokens/create_personal_access_token)、[Connected Apps FAQ](https://mypikpak.com/en-US/connect-apps-faq)。

## 訂閱與命名

新增訂閱的第一次檢查只建立基準，後續才處理新項目。若要處理 RSS 目前仍保留的項目，按該訂閱的「補抓」。資訊源變更時重新建立基準；已提交的相同 infohash 仍會按帳號去重。刪除訂閱保留已建立的任務與雲端檔案；停用訂閱只停止排程檢查。

例如發布標題：`[字幕組] 葬送的芙莉蓮 - 03 [1080p]`。

```text
作品名稱：葬送的芙莉蓮
目錄：Anime/葬送的芙莉蓮
Regex：-\s*(?P<ep>\d+).*?\[(?P<resolution>\d+p)\]
範本：{title} - S{season:02}E{ep:02} [{resolution}].{ext}
結果：葬送的芙莉蓮 - S01E03 [1080p].mkv
```

支援 `{title}`、`{season}`、`{ep}`、`{resolution}`、`{ext}`。季數與集數可用 `{season:02}`、`{ep:03}` 補零，寬度 1–6。`title` 為該訂閱的作品名稱，`ext` 取自實際雲端檔案；預覽未指定檔名時以 `.mkv` 示意。

Regex 使用 Go RE2，具名群組寫成 `(?P<ep>...)`、`(?P<season>...)`、`(?P<resolution>...)`。不支援 lookbehind 與反向參照。預設規則處理 `S01E03`、`第3話`、` - 03 ` 等常見格式；請用實際發布標題及檔名測試每筆訂閱。

多檔種子優先逐一解析影片檔名。只有一個主要檔案才可用 RSS 標題補足變數；多個影片不可把同一集數套給全部。附屬檔案移至目標目錄的 `_附件/<任務ID>/`，保留原名與子目錄。無法解析或名稱衝突的影片留在 `_PikPak-RSS-Staging/<任務ID>/`，可修正訂閱規則後重試。空暫存目錄保留，不自動刪除。

官方 MCP `add_link` 不接受新檔名，因此下載完成後才執行雲端重命名／移動。沒有本地媒體搬運，也不覆寫同名檔案。提交結果不明的任務只會重新核對專用暫存目錄；未能唯一確認時需在 PikPak 檢查，不會重新提交。[官方 MCP 說明](https://mypikpak.com/en-US/help-center/connected_apps/mcp/connect_ai_tool)。

## 本機開發

需要 Git 與支援自動下載 toolchain 的 Go。`go.mod` 固定 Go 1.27.1；`GOTOOLCHAIN=auto` 可自動取得，不會更改系統 Go 安裝。

```sh
cp .env.example .env  # 僅限尚未存在 .env；不要覆蓋既有權杖
# 編輯 .env，設定管理密碼
go run ./cmd/pikpak-rss-manager
go test ./...
go vet ./...
```

Windows 可用 `./scripts/dev.ps1` 與 `./scripts/verify.ps1`。測試預設不使用真實 PAT 或網路雲端操作；[驗證文件](docs/verification.md) 說明選擇性實測與限制。JSON API 需要登入 Cookie；修改請求須同時帶 `pp_csrf` Cookie 與 `X-CSRF-Token`。詳細資料結構及流程見 [架構文件](docs/architecture.md)。

## 發布、更新與備份

GitHub Actions 在推送 `main` 或 Tag 時，先執行測試、vet 及 Compose 啟動／資料持久化測試，再發布兩種架構。`main` 產生 `latest`、`main`、`sha-...`；Tag 產生對應 Tag、版本與提交標籤。使用 `GITHUB_TOKEN`，不需自行保存 GHCR 發布權杖。[GitHub 官方 GHCR 文件](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)。

```sh
docker compose pull
docker compose up -d
```

備份前停止服務，備份**整個資料卷**，包含 `manager.db`、可能存在的 WAL／SHM 與 `secret.key`。遺失金鑰就無法解密已保存的權杖與私密訂閱連結；金鑰和備份都應限制存取。不要使用 `docker compose down -v` 更新服務。[完整部署／備份步驟](docs/deployment.md)。
