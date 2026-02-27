# API Changelog

所有 API 相關變更紀錄。格式依循 [Keep a Changelog](https://keepachangelog.com/)。

## [v0.10.0] - 2026-02-28

### Added
- `POST /api/v1/characters` — 建立角色（name, attributes, inventory, notes）
- `GET /api/v1/characters` — 列出用戶的角色（分頁）
- `GET /api/v1/characters/{id}` — 取得角色詳情（僅擁有者）
- `PUT /api/v1/characters/{id}` — 更新角色（僅擁有者）
- `DELETE /api/v1/characters/{id}` — 刪除角色（僅擁有者，已指派場次時 409）
- `POST /api/v1/sessions/{id}/characters` — 指派角色至場次（僅 lobby 狀態）
- `SessionPlayerResponse` 新增 `characterId` 欄位（nullable）

## [v0.9.0] - 2026-02-28

### Fixed
- Snapshot SQL 修正：`SaveSnapshot`/`LoadSnapshot` 改用 `game_sessions` 表（原誤引 `game_snapshots`）

### Added
- Hub 自動 `RecoverFromSnapshot`：Room 建立時從 DB 快照 + 事件重放恢復狀態
- OpenAPI spec 更新至 v0.9.0：文件化所有 WebSocket action types 和 broadcast event types
- WebSocketEnvelope / IncomingAction schemas

## [v0.8.0] - 2026-02-28

### Added
- `expr-lang/expr` 表達式引擎整合（6 個注入函式：`has_item`/`roll`/`attr`/`var`/`all_have_item`/`player_count`）
- `condition_met` 觸發器（條件轉場，EvalBool）
- `auto` 觸發器（自動鏈式跳轉，maxTransitionChainDepth=10）
- `set_var` 表達式支援（`SetVarAction.Expr` 欄位）
- Snapshot 系統（每 50 事件自動快照，`Room.RecoverFromSnapshot`）
- WebSocket `gm_broadcast` action — GM 推送文字/圖片給特定或所有玩家
- `gm_broadcast` 事件（per-player 目標過濾）

## [v0.7.0] - 2026-02-28

### Added
- WebSocket `player_choice` action — 玩家場景轉場選擇
- WebSocket `reveal_item` action — GM 手動揭露道具
- WebSocket `reveal_npc_field` action — GM 揭露 NPC 欄位
- `player_choice` 事件（玩家選擇審計記錄 + 場景轉場）
- `item_revealed` 事件（per-player 道具揭露追蹤）
- `npc_field_revealed` 事件（per-player NPC 欄位揭露）
- `variable_changed` 事件（場景變數變更）
- on_enter / on_exit 場景 action 系統（set_var, reveal_item, reveal_npc_field）
- Per-player 場景過濾（items_available、npcs_present 權限過濾）
- Scenario variables 初始化（Variables → GameState）

## [v0.6.0] - 2026-02-27

### Added
- WebSocket `advance_scene` action — GM 場景切換（含 gm_notes 權限過濾廣播）
- WebSocket `dice_roll` action — GM/Player 骰子擲骰
- `scene_changed` 事件（場景切換 + per-role payload 過濾）
- `dice_rolled` 事件（骰子結果廣播）
- Scenario content JSON → Go types 解析（ScenarioContent, Scene, Transition 等）
- Dice engine（NdS, NdS+M, NdS-M, dS 格式，crypto/rand）
- GameState 擴展：current_scene, players, dice_history
- ScenarioLoader interface（consumer-side，bridges repos → realtime）

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
