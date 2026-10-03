# 驗證紀錄

## 可重現的本機／CI 驗證

```sh
go test ./...
go vet ./...
CGO_ENABLED=0 go build ./cmd/pikpak-rss-manager
```

Linux CI 加跑 `go test -race ./...`；容器建置後使用 `scripts/container-smoke.py` 啟動本專案的 Compose，測試健康、未登入限制、短密碼登入、Regex 新舊檔名預覽、non-root UID、重建容器後的訂閱、重命名關閉設定與 AES 金鑰持久化。不配置真實 PAT，不連線 PikPak 或實際 RSS。

自動化測試涵蓋 RSS／Atom、相對 torrent enclosure、Magnet hex／base32／v2、原始 bencode infohash、資料大小／私人網路限制、中文命名、補零、實際副檔名、RE2 限制、首次基準、補抓、跨訂閱去重、多檔歧義、附屬檔案、名稱衝突、提交回應遺失、重啟恢復、部分 rename／move、授權／配額／限流、換帳號與 HTTP 登入／CSRF／私密欄位隔離。

修正版亦涵蓋含 403 的配額錯誤、網站來源不得含 query／fragment、明確配置的空白 PAT 檔案不得回退至其他憑證，以及新帳號補抓不沿用舊帳號的已處理 fingerprint／雲端 ID。選擇性實測資源固定至上游 commit，避免未來 master 變更擴大測試內容。

密碼規則更新的測試涵蓋一字元、中文／Emoji、超過 72 bytes、前後空白的環境／檔案密碼，並透過真正的 HTTP handler 驗證登入、session 與超長密碼尾端差異，避免只比較前 72 bytes。登入表單不再設定 minlength／maxlength。

v0.2.0 測試新增：資料夾分頁與檔案過濾、同名資料夾的 ID 身分、循環路徑、建立同名項目阻擋、建立回應遺失後不重送、CSRF／帳號切換隔離；重命名關閉、數字／具名群組、移除匹配、字面 `$`、所有匹配替換、未匹配保留原名、無效群組／空檔名、舊訂閱與任務規則相容、選取 ID 及開關重啟持久化。正式 worker 測試確認未勾選時沒有 rename 呼叫，仍執行 move；Regex 模式逐檔替換，不使用 RSS 標題回退。

## 真實 PikPak 實測（2026-10-03，台灣時間）

用本機使用者提供的 PAT 透過官方 MCP 實測，PAT 不進入測試輸出、Git 或 Actions。測試目錄：`_pikpak-rss-manager-test/run-20261002T201936-d8752c`。所有測試 mutation 僅限本專案建立的測試目錄；測試檔案與任務保留供核對，未刪除既有內容。

- 已通過：官方 SDK Streamable HTTP、PAT 授權、帳號與目錄讀取、建立目錄、提交任務、查詢完成、取得實際副檔名、真實雲端重命名與移動、同一來源在本機任務儲存中去重而不再提交。
- 測試資源：[webtorrent/webtorrent-fixtures](https://github.com/webtorrent/webtorrent-fixtures) 的公開領域 Alice 文字檔，163,783 bytes。先只下載 325-byte `.torrent` 中繼資料並提交 Magnet；90 秒內未完成。
- 改以相同小型檔案的 HTTP URL 提交 PikPak 雲端下載，成功完成、重命名及移動；服務沒有下載文字檔內容。這項結果證明真正的雲端工作流程，**未證明該 Magnet 的來源可用或秒傳**。
- 官方 MCP `task_get` 實際回傳百分比文字與 `Complete (PHASE_TYPE_COMPLETE)` 狀態標籤，適配器已涵蓋此格式。

選擇性重跑（會在新的專用目錄建立小型雲端任務並使用配額）：

```powershell
$env:PIKPAK_LIVE_TEST='1'
go test ./tests/integration -run TestLivePikPak -count=1 -v
Remove-Item Env:PIKPAK_LIVE_TEST
```

本機 `.env` 需含 `PIKPAK_TOKEN`，不得把權杖放在命令列。測試資料庫／非敏感摘要置於忽略的 `.local`，供失敗核對。來源授權由 WebTorrent fixture README 說明；不使用大型影片測試。

### 資料夾功能實測（2026-10-03）

官方 MCP 已通過新增資料夾功能的實測，專用目錄：`_pikpak-rss-manager-test/folders-run-20261003T075736-0410f1ec`。建立「目錄選擇測試」子資料夾、列出子資料夾、依 ID 追溯完整路徑與阻擋同名重建皆通過。此次僅操作專用測試目錄的資料夾中繼資料，沒有新建下載任務、移動或刪除既有帳號內容，測試目錄保留。

```powershell
$env:PIKPAK_LIVE_FOLDERS_TEST='1'
go test ./tests/integration -run TestLiveFolderBrowsingAndCreation -count=1 -v
Remove-Item Env:PIKPAK_LIVE_FOLDERS_TEST
```

需已設定本機管理密碼及 PAT。一般 `go test ./...` 不執行這項真實操作；CI／Actions 不提供 PAT。

## 發布與介面驗證

本機沒有 Docker。本機 `go test ./...` 與 `go vet ./...` 已通過，以下 GitHub Actions 驗證亦已成功：

| 驗證 | 結果／證據 |
|---|---|
| Linux race、vet、Go build、Docker build、Compose 啟動／持久化 | [v0.1.1 程式碼 CI 通過](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37066105759) |
| GHCR 多架構發布、版本／提交標籤、manifest 核對 | [v0.1.1 發布通過](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37066109017)；[main/latest 發布通過](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37066106170) |
| 無 GHCR 登入的實際拉取、兩種架構啟動、登入／健康／資料保留 | [v0.1.1 Public image smoke 通過](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37066855304) |
| 套件公開權限與匿名 manifest | [套件頁](https://github.com/wade00754/pikpak-rss-manager/pkgs/container/pikpak-rss-manager) 顯示 Public；以無 GitHub 憑證的 pull grant 取得兩種架構 |

`amd64` 在 Linux Runner 原生執行；`arm64` 透過 QEMU 執行實際容器，亦通過 Compose 重建與加密資料持久化。這不代表已在使用者的 ARM VPS 上部署。Go／Docker／公開映像的驗證工作流保留在 `.github/workflows`，公開映像 smoke 可指定 Tag 手動重跑。

首次交付版本為 [`v0.1.1`](https://github.com/wade00754/pikpak-rss-manager/releases/tag/v0.1.1)；上表驗證對應 commit `a93bb3b`。取消密碼規則的更新納入 [`v0.1.2`](https://github.com/wade00754/pikpak-rss-manager/releases/tag/v0.1.2)，映像 `ghcr.io/wade00754/pikpak-rss-manager:v0.1.2`。該更新已通過本機 Go 測試／vet 及 PowerShell 一字元密碼輸入測試；容器 smoke 也改用少於 12 字元的臨時測試密碼，發布驗證結果附於 Release。`latest` 跟隨 main。

瀏覽器已驗證登入／手機登出、兩筆訂閱的獨立規則保存、重載後資料、中文命名預覽與實際 `.mp4` 副檔名；在 1280px 與 390px 寬度檢查排版，手機沒有頁面橫向溢出。使用停用的測試訂閱與獨立本機資料目錄，不觸發新的雲端下載。畫面記錄：`docs/screenshots/subscriptions.jpg`、`docs/screenshots/mobile.jpg`。

v0.2.0 介面以獨立的 mocked PikPak 帳號與資料目錄測試：新增訂閱預設不重命名；資料夾導覽、建立、選取與重新載入；勾選後顯示 Regex／替換欄位；前綴移除與具名群組新舊預覽；1280px 桌面與 390px 手機檢查。頁面與對話框沒有橫向溢出。此 UI 測試不讀取 PAT、不呼叫真實 PikPak，雲端資料夾真實驗證由上一節分開記錄。畫面：`docs/screenshots/folder-picker.png`、`docs/screenshots/subscription-editor.png`、`docs/screenshots/mobile-editor.png`。

這次功能更新的版本為 [`v0.2.0`](https://github.com/wade00754/pikpak-rss-manager/releases/tag/v0.2.0)，本機 Go 測試、vet、CGO-free build 與 JavaScript／Python 語法檢查通過；對應 CI、多架構發布與匿名容器 smoke 的實際結果附於 Release。

尚未在使用者的 Linux VPS 或實際 1Panel 版本上部署；1Panel 操作步驟依標準 Compose 編排說明。官方 MCP 的未來 API 變更、不同帳號配額和來源可用性不屬於 mocked tests 能保證的範圍。
