# 官方 OAuth 與 PAT 調研

核對日期：2026-10-03。此版本仍以 PAT 呼叫官方託管 MCP；以下 OAuth 資料為官方文件及公開 discovery 的核對結果，尚未實作或完成真實 OAuth 授權測試。

## 找到的官方資料

[MCP 連接說明](https://mypikpak.com/en-US/help-center/connected_apps/mcp/connect_ai_tool) 描述瀏覽器開啟 PikPak 授權頁，使用者確認工具及權限；不支援這種連接方式的工具可使用 PAT。這是使用者操作說明，尚未找到完整的第三方應用開發教學。

不帶任何 PAT 讀取下列官方端點，皆回傳 HTTP 200 JSON：

- [MCP protected resource metadata](https://pikpak.ai/.well-known/oauth-protected-resource/mcp)：resource 為 `https://pikpak.ai/mcp`，authorization server 為 `https://user.mypikpak.com`，Bearer 放在 HTTP header。
- [OpenID configuration](https://user.mypikpak.com/.well-known/openid-configuration)：宣告 authorization code、refresh token、device code 等 grant；PKCE 有 `S256`；提供 authorization、token、revocation 及 registration 端點。

| 欄位 | 官方宣告的端點 |
|---|---|
| 授權頁 | `https://mypikpak.com/drive/auth-center` |
| 權杖 | `https://user.mypikpak.com/v1/auth/token` |
| 撤銷 | `https://user.mypikpak.com/v1/auth/revoke` |
| 用戶端註冊 | `https://user.mypikpak.com/v1/clients-registrations/openid-connect` |
| Device code | `https://user.mypikpak.com/v1/auth/device/code` |

Discovery 宣告不等於每種自託管應用都能直接註冊成功。尚未測試 client registration、允許的 callback、MCP scopes 授權結果、access／refresh token 壽命、更新與輪替策略。未向官方註冊應用、未建立 OAuth 連接，也沒有嘗試擴大既有 PAT 的權限。

## 哪個適合這個服務？

| 比較 | PAT | OAuth |
|---|---|---|
| 個人部署 | 建立一次後貼入面板或掛載私密檔案，適合背景排程 | 可做成「連接 PikPak」按鈕及官方同意畫面 |
| 有效期 | 官方提供 30 天至一年，預設 90 天；到期需更換 | 宣告 refresh grant，可設計自動更新；具體壽命／失效條件尚待實測 |
| 程式與部署 | 只需保護一個 Bearer 憑證，現已驗證 | 需 client 註冊、callback、state／PKCE、加密 refresh token 與更新失敗恢復 |
| 權限與撤銷 | 使用者勾選必要權限，可在 Connected Apps 撤銷 | 使用者在同意頁確認 scopes，亦可撤銷連接；實際可授權範圍依應用／連接類型 |
| 本專案狀態 | 已實作並完成本機真實 MCP 測試 | 官方端點已核對，尚未實作或授權實測 |

建議目前單一帳號、自託管與無人值守部署繼續使用 PAT，設定所需權限與合適有效期。若下一版要改善多人安裝時的一鍵連接體驗，可增加 OAuth 並保留 PAT 備援；應優先採 authorization code + PKCE S256，驗證 state、精確 callback，並加密保存更新權杖。這是實作建議，不代表 PikPak 所有相關限制已驗證。

PAT 是持有即可操作的憑證；OAuth 發出的 access／refresh token 同樣需要保護。不能只因使用 OAuth 就推論較不會遭風控；官方未提供可支持這個比較的保證。[官方 PAT 有效期及建立方式](https://mypikpak.com/en-US/help-center/connected_apps/personal_access_tokens/create_personal_access_token)。

## 截圖中「永久刪除／邀請請用 OAuth」的限制

[官方權限說明](https://mypikpak.com/en-US/help-center/connected_apps/managing_connected_apps/connected_app_permissions) 同時指出 **PAT 與 PikPak MCP 都不能授予永久刪除和邀請權限**。因此改用 OAuth 並不保證透過這個託管 MCP 就能使用它們；還需核對應用類型與實際授權。本 RSS 服務僅需要帳號資訊、讀寫／整理檔案和雲端下載，不需要這兩項操作，也不自動刪除暫存目錄。

所有 Connected Apps 共用各流量維度月配額的 25%；更換 PAT／OAuth 不會繞過這個共用限制。[官方配額 FAQ](https://mypikpak.com/en-US/connect-apps-faq)。
