# SPEC-005: WebSocket Hub-Room & Real-Time Game Engine Foundation

| 項目 | 說明 |
|------|------|
| 前置 | SPEC-004（Game Session REST API） |
| ADR 參考 | ADR-002（即時通訊）、ADR-004（事件溯源）、ADR-005（權限模型） |
| 狀態 | Completed |

## 目標

建立 Hub-Room-Client WebSocket 架構，實現遊戲場次的即時通訊基礎設施。驗證完整 pipeline：REST handler → 事件持久化 → 狀態更新 → WebSocket 廣播。

## 範圍

### In Scope
- `gorilla/websocket` 整合
- Hub-Room-Client 架構（`internal/realtime/`）
- WebSocket 升級端點（JWT query param 驗證）
- `game_events` 表 CRUD（AppendEvent + ListEventsSince）
- 4 個生命週期事件：`game_started`、`game_paused`、`game_resumed`、`game_ended`
- 心跳機制（30s ping / 10s pong timeout）
- 斷線重連（replay events since lastEventSeq）
- REST lifecycle handlers 整合 Hub 廣播

### Out of Scope
- 場景切換、道具揭露、NPC 欄位揭露、骰子擲骰（SPEC-006+）
- YAML DSL 解析器
- 角色/玩家 CRUD
- 快照優化（50 事件快照）
- 權限過濾廣播（GM 全資料 vs Player 過濾資料）

## API

### WebSocket Endpoint

```
GET /api/v1/sessions/{id}/ws?token=<jwt>&last_event_seq=0
```

| 參數 | 類型 | 必要 | 說明 |
|------|------|------|------|
| `id` | path | 是 | Session UUID |
| `token` | query | 是 | JWT access token |
| `last_event_seq` | query | 否 | 最後收到的 event sequence（預設 0） |

**成功：** 101 Switching Protocols → WebSocket 連線建立

**錯誤（升級前）：**
- 400: Invalid session ID / Missing token
- 401: Invalid or expired token
- 403: Not a session member / Session not active
- 404: Session not found

### Message Envelope（ADR-002）

```json
{
  "type": "game_started | game_paused | game_resumed | game_ended | state_sync | error",
  "session_id": "uuid",
  "sender_id": "uuid",
  "target_ids": ["uuid"],
  "payload": {},
  "timestamp": 1709020800
}
```

### Lifecycle Events

| Event Type | Actor | Trigger | Payload |
|------------|-------|---------|---------|
| `game_started` | GM | `POST /sessions/{id}/start` | `{}` |
| `game_paused` | GM | `POST /sessions/{id}/pause` | `{}` |
| `game_resumed` | GM | `POST /sessions/{id}/resume` | `{}` |
| `game_ended` | GM | `POST /sessions/{id}/end` | `{}` |
| `state_sync` | System | Client 連線/重連 | `{"session_id":"...","status":"...","last_sequence":N}` |

## 架構

```
Hub (manages all Rooms, sync.RWMutex)
 ├── Room A (session-1, single goroutine)
 │    ├── GM Client  (readPump + writePump goroutines)
 │    └── Player Client
 └── Room B (session-2)
      └── GM Client
```

### Event Processing Pipeline

```
REST Handler (handleStartSession)
     ↓
Hub.GetOrCreateRoom(sessionID, gmID)
     ↓
Room.BroadcastEvent(eventType, actorID, payload)
     ↓  (via processEvent channel → Room goroutine)
Assign sequence (LastSequence++)
     ↓
Persist to game_events (AppendEvent)
     ↓
Apply to in-memory GameState
     ↓
Broadcast Envelope to all connected Clients
```

## Edge Cases

| 情境 | 處理 |
|------|------|
| WS 連線時 session 非 active/paused | 拒絕升級，403 |
| GM 自己也連 WS | 允許，role=gm |
| 重連 last_event_seq > 當前 | replay 回空結果，發送 state_sync |
| Room 不存在時收到 lifecycle event | Hub 建立新 Room |
| Client send buffer 滿 | 斷開該 Client |
| Server shutdown | Hub.Stop() → 所有 Room.Stop() → Close 所有 Client |

## 驗收條件

1. WebSocket 連線可成功建立（JWT + session member 驗證通過）
2. GM 呼叫 `POST /sessions/{id}/start` 後，連線的 Client 收到 `game_started` envelope
3. 斷線重連後，收到遺漏的 event replay + state_sync
4. 心跳正常運作（30s ping / 10s pong timeout）
5. `go test ./... -race` 全部通過，包含 realtime 包單元測試
6. 現有 142 個 unit tests 不受影響
