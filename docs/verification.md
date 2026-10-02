# 驗證紀錄

## 可重現的本機／CI 驗證

```sh
go test ./...
go vet ./...
CGO_ENABLED=0 go build ./cmd/pikpak-rss-manager
```

Linux CI 加跑 `go test -race ./...`；容器建置後使用 `scripts/container-smoke.py` 啟動本專案的 Compose，測試健康、未登入限制、登入、non-root UID、重建容器後的訂閱與 AES 金鑰持久化。不配置真實 PAT，不連線 PikPak 或實際 RSS。

自動化測試涵蓋 RSS／Atom、相對 torrent enclosure、Magnet hex／base32／v2、原始 bencode infohash、資料大小／私人網路限制、中文命名、補零、實際副檔名、RE2 限制、首次基準、補抓、跨訂閱去重、多檔歧義、附屬檔案、名稱衝突、提交回應遺失、重啟恢復、部分 rename／move、授權／配額／限流、換帳號與 HTTP 登入／CSRF／私密欄位隔離。

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

## 發布與介面驗證

本機沒有 Docker。本機 `go test ./...` 與 `go vet ./...` 已通過。Docker／Compose、race 與兩種架構驗證由 GitHub Actions 執行；最終結果與匿名 GHCR 驗證會在發布完成後更新於此。

瀏覽器已驗證登入、兩筆訂閱的獨立規則保存、重載後資料、中文命名預覽與實際 `.mp4` 副檔名；在 1280px 與 390px 寬度檢查排版，手機沒有頁面橫向溢出。使用停用的測試訂閱與獨立本機資料目錄，不觸發新的雲端下載。畫面記錄：`docs/screenshots/subscriptions.jpg`、`docs/screenshots/mobile.jpg`。

尚未在使用者的 Linux VPS 或實際 1Panel 版本上部署；1Panel 操作步驟依標準 Compose 編排說明。官方 MCP 的未來 API 變更、不同帳號配額和來源可用性不屬於 mocked tests 能保證的範圍。
