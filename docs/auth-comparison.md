# PAT and OAuth

Research checked on 2026-10-03. PAT support is implemented and tested against the official hosted MCP. OAuth is researched only. Client registration, consent and refresh flows have not been implemented or tested.

| | PAT | OAuth |
|---|---|---|
| Setup | Create a token and enter it in Settings | Requires client registration, callback and consent |
| Renewal | Replace before expiry | Discovery advertises refresh tokens. Behavior untested |
| Storage | AES-GCM encrypted bearer token | Would require encrypted access/refresh tokens |
| Project status | Implemented | Not implemented |

Official discovery advertises authorization code, PKCE S256, refresh tokens and registration. These declarations do not prove registration or consent will succeed for this service.

- [MCP connection guide](https://mypikpak.com/en-US/help-center/connected_apps/mcp/connect_ai_tool)
- [Protected resource metadata](https://pikpak.ai/.well-known/oauth-protected-resource/mcp)
- [OpenID configuration](https://user.mypikpak.com/.well-known/openid-configuration)
- [PAT creation](https://mypikpak.com/en-US/help-center/connected_apps/personal_access_tokens/create_personal_access_token)

The official permission documentation restricts permanent deletion and invitations through both PAT and hosted MCP. OAuth does not automatically unlock them. This service does not need those permissions. Connected apps share monthly traffic quotas. [Permissions](https://mypikpak.com/en-US/help-center/connected_apps/managing_connected_apps/connected_app_permissions) · [Quota FAQ](https://mypikpak.com/en-US/connect-apps-faq).
