# 驗證紀錄

## 可重現的本機／CI 驗證

```sh
go test ./...
go vet ./...
CGO_ENABLED=0 go build ./cmd/pikpak-rss-manager
```

Linux CI 加跑 `go test -race ./...`；容器建置後使用 `scripts/container-smoke.py` 啟動本專案的 Compose，無須密碼環境變數或 `.env`，測試網頁初始化、拒絕重新初始化、健康、未登入限制、重建後的短密碼登入、Regex 新舊檔名預覽、non-root UID、重建容器後的訂閱、重命名關閉設定與 AES 金鑰持久化。不配置真實 PAT，不連線 PikPak 或實際 RSS。

自動化測試涵蓋 RSS／Atom、相對 torrent enclosure、Magnet hex／base32／v2、原始 bencode infohash、資料大小／私人網路限制、中文命名、補零、實際副檔名、RE2 限制、首次基準、補抓、跨訂閱去重、多檔歧義、附屬檔案、名稱衝突、提交回應遺失、重啟恢復、部分 rename／move、授權／配額／限流、換帳號與 HTTP 登入／CSRF／私密欄位隔離。

修正版亦涵蓋含 403 的配額錯誤、網站來源不得含 query／fragment，以及新帳號補抓不沿用舊帳號的已處理 fingerprint／雲端 ID。外部 PAT 檔案來源已於 v0.4.0 移除，改由網頁綁定。選擇性實測資源固定至上游 commit，避免未來 master 變更擴大測試內容。

密碼測試涵蓋一字元、中文／Emoji、超過 72 bytes、前後空白，並透過真正的 HTTP handler 驗證網頁初始化、登入、session 與超長密碼尾端差異。初始化／登入表單不設定 minlength／maxlength。

v0.2.0 測試新增：資料夾分頁與檔案過濾、同名資料夾的 ID 身分、循環路徑、建立同名項目阻擋、建立回應遺失後不重送、CSRF／帳號切換隔離；重命名關閉、數字／具名群組、移除匹配、字面 `$`、所有匹配替換、未匹配保留原名、無效群組／空檔名、舊訂閱與任務規則相容、選取 ID 及開關重啟持久化。正式 worker 測試確認未勾選時沒有 rename 呼叫，仍執行 move；Regex 模式逐檔替換，不使用 RSS 標題回退。

v0.3.0 測試新增：RSS 標題／Magnet 顯示名稱／真實種子檔名的來源區分；v1 多檔、UTF-8 路徑、v2 file tree；五筆來源、三次種子讀取與中繼資料／私人網路限制；來源 API 登入／CSRF 保護、敏感 URL 不回傳、同帳號原始檔名記錄、換帳號不沿用舊名稱；讀取範例不新建任務或修改基準。以使用者提供的 `\[(\d+)\]` → `S01E$1` 和含 `/` 的中英標題，驗證原始替換結果、實際檔名與警告。Worker 驗證新暫存目錄位於目標內，以及舊 staging ID 重試時保持原位置。容器 smoke 亦驗證 raw_name 與正規化提示。

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

先在網頁初始化並綁定 PAT；測試從本機 `data` 的加密設定讀取權杖，不讀取 `.env`。自訂資料目錄時設定 `PIKPAK_LIVE_DATA_DIR`，不得把權杖放在命令列。測試資料庫／非敏感摘要置於忽略的 `.local`，供失敗核對。來源授權由 WebTorrent fixture README 說明；不使用大型影片測試。

### 資料夾功能實測（2026-10-03）

官方 MCP 已通過新增資料夾功能的實測，專用目錄：`_pikpak-rss-manager-test/folders-run-20261003T075736-0410f1ec`。建立「目錄選擇測試」子資料夾、列出子資料夾、依 ID 追溯完整路徑與阻擋同名重建皆通過。此次僅操作專用測試目錄的資料夾中繼資料，沒有新建下載任務、移動或刪除既有帳號內容，測試目錄保留。

```powershell
$env:PIKPAK_LIVE_FOLDERS_TEST='1'
go test ./tests/integration -run TestLiveFolderBrowsingAndCreation -count=1 -v
Remove-Item Env:PIKPAK_LIVE_FOLDERS_TEST
```

需已在本機網頁初始化並綁定 PAT；自訂資料目錄使用 `PIKPAK_LIVE_DATA_DIR`。一般 `go test ./...` 不執行這項真實操作；CI／Actions 不提供 PAT。

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

v0.3.0 已通過本機完整 Go 測試、vet、CGO-free build 與 JavaScript／Python 語法檢查。介面使用獨立 localhost RSS／小型種子中繼資料與 mocked PikPak，驗證來源選擇與自動預覽、含 `/` 的原始替換及實際檔名分列、按鈕開啟小型新增資料夾視窗、Enter 建立與 Escape 取消、選取保存與重新載入。1280px 桌面及 390px 手機沒有頁面或對話框橫向溢出，瀏覽器無錯誤；沒有讀取 PAT 或新增真實雲端任務。新增畫面：`docs/screenshots/source-preview.png`、`docs/screenshots/folder-create.png`；既有資料夾／訂閱／手機截圖亦已更新。容器與發布的最終結果附於 [v0.3.0 Release](https://github.com/wade00754/pikpak-rss-manager/releases/tag/v0.3.0)。

OAuth 調研僅讀取官方公開文件及 discovery（無憑證），確認宣告的端點與流程；沒有註冊 client、完成同意流程或測試 refresh token，不代表 OAuth 已可在本服務使用。詳見 [授權比較](auth-comparison.md)。

### 即時檔名預覽與簡化範例（2026-10-03）

本機 `go test ./...`、`go vet ./...`、`CGO_ENABLED=0 go build -trimpath` 與 `node --check internal/web/static/app.js` 通過。預覽沿用後端 Go renderer，輸入使用 200 ms 防抖，並在每次編輯時立即使舊請求失效。

使用獨立的 localhost RSS／小型種子中繼資料、mocked PikPak 帳號與 `.local/realtime-ui-data` 驗證：混合 RSS 標題、Magnet 顯示名稱與種子多檔的來源只顯示兩筆種子檔名，選項沒有來源前綴；切換 `.mkv`／`.ass`、編輯原始檔名、Regex 或替換格式，以及快速連續編輯時會自動更新結果。另確認具名群組、無效 Regex 修正、未匹配保留原名與清空輸入提示。結果只顯示一列最終檔名；`S01/E$1` 的 `/` 轉為 `_` 時，只附一行「檔名含不適用字元，已自動替換或移除。」

1280px 桌面與 390px 手機下，頁面及對話框沒有橫向溢出，瀏覽器無錯誤或警告。此次未讀取 PAT、未呼叫真實 PikPak，亦未新建下載任務。畫面：`docs/screenshots/source-preview.jpg`、`docs/screenshots/realtime-mobile.jpg`。本機無 Docker，容器驗證由 GitHub Actions 執行；發布後的容器驗證結果請見 [v0.3.1 Release](https://github.com/wade00754/pikpak-rss-manager/releases/tag/v0.3.1)。

### 分批讀取更多種子檔名（2026-10-03）

本機 `go test ./...`、`go vet ./...`、`CGO_ENABLED=0 go build -trimpath`、JavaScript 語法與 diff 空白檢查通過。新增測試驗證七個種子在每批三次讀取的限制內全部處理、混合 Magnet 的五筆來源限制、前批讀取失敗後仍會接續檢查未讀取的種子，以及 65 個多檔檔名以 30／30／5 接續讀取，沒有遺漏或重複。

續讀位置只含偏移與內容摘要；測試涵蓋無效位置在網路請求前拒絕、RSS 或種子檔名變更時拒絕舊位置、不回傳私密 URL／帳號，以及來源同時提供 Magnet 和 `.torrent` 時預覽讀取實際種子檔名。既有下載來源優先順序保持通過原有測試。HTTP 分頁測試涵蓋 CSRF、同帳號快取檔名只附於首次回應、帳號切換隔離，以及不修改基準或建立雲端任務。

以獨立的 `.local/paging-ui-data`、mocked PikPak 與 localhost v1／v2 小型種子，驗證三批取得 3／6／8 個檔名，追加時保留目前選項與手動檔名／即時結果，結束後收起續讀按鈕；更換 RSS 清除舊選項與分頁狀態。1280px 桌面與 390px 手機沒有頁面或對話框橫向溢出，瀏覽器無錯誤或警告。畫面：`docs/screenshots/source-paging.jpg`、`docs/screenshots/source-paging-mobile.jpg`。此次未讀取 PAT、未呼叫真實 PikPak。容器與發布結果請見 [v0.3.2 Release](https://github.com/wade00754/pikpak-rss-manager/releases/tag/v0.3.2)。

### v0.3.3 本機開發版本顯示（2026-10-03）

v0.3.3 修正 `main.go` 固定使用 `0.3.0-dev`、`dev.ps1` 未傳入版本的問題。該版腳本以 `git describe` 取得版本並透過 Go linker 傳入，含標籤後的提交與未提交變更；無可用 Git 資訊時顯示 `dev`。該版發布映像由建置流程傳入正式標籤或提交。

本機完整 `go test ./...`、`go vet ./...`、CGO-free build、PowerShell 語法與 diff 空白檢查通過。實際執行 `./scripts/dev.ps1 -Version`，結果 `v0.3.2-dirty-dev` 與當時 Git 版本一致；未注入版本的建置執行 `version` 顯示 `dev`。在獨立 PowerShell 程序模擬沒有 Git 及無法讀取 Git 資訊，兩者皆回退至 `dev` 且正常結束。版本查詢會在設定載入與密碼提示前返回；此次未讀取 PAT、未啟動正式服務或呼叫 PikPak。容器與發布結果請見 [v0.3.3 Release](https://github.com/wade00754/pikpak-rss-manager/releases/tag/v0.3.3)。

### 內建目前版本號（2026-10-03）

v0.3.4 將 `cmd/pikpak-rss-manager/VERSION` 內建至程式，開發腳本直接沿用此版本號。正式版本檔隨發布更新；CI 核對建置執行檔的版本，發布工作流亦核對 Tag 與版本檔一致。`latest` 與 Tag 映像皆使用此版本號；預設容器建置不傳入版本時也使用內建值。

本機完整 `go test ./...`、`go vet ./...`、CGO-free build、PowerShell／Python 語法與 diff 空白檢查通過。實際執行 `dev.ps1 -Version`、直接 `go run ... version`、不嵌入 VCS metadata 的建置，以及使用空白 linker 版本參數的建置，全部顯示 `v0.3.4`。從獨立測試程序的 PATH 移除 Git 並確認不可用後，`dev.ps1 -Version` 仍顯示 `v0.3.4`。這些查詢不讀取設定或 PAT，也不啟動正式服務。

容器 smoke 新增核對執行檔、健康／session API、登入頁與登入後頁面的版本，並比較預期的正式版本號；公開 smoke 會檢查正式 Tag 或目前 `latest`／`main` 的版本。容器與發布的實際結果請見 [v0.3.4 Release](https://github.com/wade00754/pikpak-rss-manager/releases/tag/v0.3.4)。本機沒有 Docker；此次未呼叫真實 PikPak。

尚未在使用者的 Linux VPS 或實際 1Panel 版本上部署；1Panel 操作步驟依標準 Compose 編排說明。官方 MCP 的未來 API 變更、不同帳號配額和來源可用性不屬於 mocked tests 能保證的範圍。

## v0.4.0 網頁初始化（2026-10-04）

本機 Go 1.27.1 已通過 `go test ./...`、`go vet ./...` 與 `go build ./cmd/pikpak-rss-manager`；程式回報 `v0.4.0`。本機建置僅在該程序設定工作區的 Git safe.directory，處理沙箱與登入使用者的目錄擁有權差異，沒有變更全域 Git 設定。

新增測試驗證：空資料目錄可啟動、首次設定頁、非空／密碼確認、CSRF／跨來源限制、完成後自動登入、拒絕匿名或已登入重新初始化、兩個資料庫連線競爭只有一個初始化成功、v1→v2 遷移保留既有資料、重啟以原雜湊登入、網站設定持久化、內網 RSS 開關等待既有檢查完成且無競態、HTTPS 代理 Secure Cookie、登入限流及損毀驗證值拒絕啟動。測試直接掃描持久化資料目錄的 DB／WAL／SHM／金鑰檔，確認沒有原始密碼；設定測試確認舊 `.env` 與外部密碼／PAT 不再載入。

瀏覽器使用獨立 `.local` 資料目錄與「介面測試帳號」mock，通過：兩次密碼不一致的提示、首次初始化後自動進入系統設定、mock PAT 綁定後清空輸入框、網站設定儲存及重載、實際重啟測試服務後用原密碼登入。1280×900 與 390×844 檢查首次設定；手機系統設定也未橫向溢出，未觀察到 console error／warning。畫面僅含 fixture：`docs/screenshots/setup-desktop.jpg`、`setup-mobile.jpg`、`web-settings-desktop.jpg`。

此 UI 測試沒有讀取使用者 PAT、連線真實 PikPak 或建立下載任務；mock 綁定不證明真實授權操作。既有 `.env` 保留未改寫。容器 smoke 已改為從 Web API 初始化並驗證重建後登入；下列 GitHub Actions 已全部通過，對應功能提交 `4dbe268`：

| 驗證 | 實際結果／證據 |
|---|---|
| Linux race、vet、build、容器啟動／網頁初始化／重建登入／資料保留 | [CI 通過](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37138343219) |
| main/latest 的 amd64、arm64 映像發布及 manifest | [main 發布通過](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37138343227) |
| v0.4.0 的 amd64、arm64 映像發布及 manifest | [Tag 發布通過](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37138512990) |
| 未登入 GHCR，兩種架構匿名拉取、網頁初始化與重建後雜湊登入 | [Public image smoke 通過](https://github.com/wade00754/pikpak-rss-manager/actions/runs/37138687394) |

固定映像：`ghcr.io/wade00754/pikpak-rss-manager:v0.4.0`；`latest` 同步發布。這些容器與 UI 測試使用隔離 fixture，未重跑真實 PikPak 下載／重命名／移動，也未部署至 VPS。

## v0.4.1 一次讀取全部種子檔名（2026-10-04）

本機 Go 1.27.1 的完整 `go test ./...`、`go vet ./...`、CGO-free build、JavaScript 語法與 diff 空白檢查通過；建置執行檔回報 `v0.4.1`。新增測試驗證 RSS 只讀一次、重複種子連結只讀一次、最多三個並行請求、65 個多檔檔名完整保留及 RSS 順序固定。另涵蓋部分失敗／取消後保留成功檔名、內網保護、32 MiB 中繼資料與 10,000 筆清單上限明確提示。HTTP 測試確認一鍵回應無續讀位置、CSRF 保護、拒絕同時指定續讀位置、切換帳號隔離，且不修改基準或呼叫雲端下載。

瀏覽器以獨立 `.local/all-ui-data2`、mocked PikPak 帳號及 localhost v1／v2 種子測試：單次按「讀取種子檔名」取得 72 個實際檔名，來源伺服器記錄一次 RSS、八次不同種子請求，畫面沒有「載入更多」按鈕。紫色按鈕在讀取期間停用並顯示「讀取中…」；重新讀取保留手動檔名，變更 RSS 取消舊請求且不接受舊結果。選取第 65 個多檔檔名仍正確更新替換結果。1280×900 桌面與 390×844 手機沒有頁面或對話框橫向溢出，瀏覽器無錯誤或警告。畫面：`docs/screenshots/source-all.png`、`docs/screenshots/source-all-mobile.png`。

此次未讀取使用者 PAT、連線真實 PikPak 或建立下載任務。本機沒有 Docker；容器、Linux race、多架構發布與匿名拉取的實際結果於發布後記錄在 [v0.4.1 Release](https://github.com/wade00754/pikpak-rss-manager/releases/tag/v0.4.1)。
