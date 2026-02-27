# API Changelog

所有 API 相關變更紀錄。格式依循 [Keep a Changelog](https://keepachangelog.com/)。

## [v0.5.0] - 2026-02-27

### Added
- `GET /api/v1/sessions/{id}/ws` — WebSocket 連線（JWT query param 驗證 + 斷線重連）
- 生命週期事件廣播：`game_started`、`game_paused`、`game_resumed`、`game_ended`
- `game_events` 事件持久化（AppendEvent + ListEventsSince）
- Hub-Room-Client 即時通訊架構

## [v0.4.0] - 2026-02-27

### Added
- `POST /api/v1/sessions` — 建立遊戲場次（基於已發布劇本）
- `GET /api/v1/sessions` — 列出 GM 的場次（分頁）
- `GET /api/v1/sessions/{id}` — 取得場次詳情
- `POST /api/v1/sessions/{id}/start` — 開始遊戲（lobby → active）
- `POST /api/v1/sessions/{id}/pause` — 暫停遊戲（active → paused）
- `POST /api/v1/sessions/{id}/resume` — 恢復遊戲（paused → active）
- `POST /api/v1/sessions/{id}/end` — 結束遊戲（active/paused → completed）
- `POST /api/v1/sessions/join` — 透過邀請碼加入場次
- `GET /api/v1/sessions/{id}/players` — 列出場次玩家
- `DELETE /api/v1/sessions/{id}/players/{userId}` — 踢玩家/離開場次

## [v0.3.0] - 2026-02-27

### Added
- `POST /api/v1/scenarios` — 建立劇本
- `GET /api/v1/scenarios` — 列出用戶的劇本（分頁）
- `GET /api/v1/scenarios/{id}` — 取得劇本詳情
- `PUT /api/v1/scenarios/{id}` — 更新劇本（僅 draft）
- `DELETE /api/v1/scenarios/{id}` — 刪除劇本（僅 draft）
- `POST /api/v1/scenarios/{id}/publish` — 發布劇本（draft → published）
- `POST /api/v1/scenarios/{id}/archive` — 封存劇本（published → archived）

## [v0.2.0] - 2026-02-27

### Added
- `POST /api/v1/users` — 用戶註冊（username, email, password）
- `POST /api/v1/auth/login` — 用戶登入（回傳 JWT access token + HttpOnly refresh token cookie）
- `POST /api/v1/auth/refresh` — 刷新 access token（refresh token 旋轉策略）
- `POST /api/v1/auth/logout` — 用戶登出（撤銷 refresh token）

## [v0.1.0] - 2026-02-27

### Added
- `GET /api/health` — 服務健康檢查（含 PostgreSQL 連線狀態）
