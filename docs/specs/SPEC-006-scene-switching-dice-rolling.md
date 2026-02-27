# SPEC-006: Scene Switching Engine & Dice Rolling System

| 項目 | 說明 |
|------|------|
| 前置 | SPEC-005（WebSocket Hub-Room & Real-Time Game Engine Foundation） |
| ADR 參考 | ADR-002（即時通訊）、ADR-003（劇本資料模型與 DSL）、ADR-004（遊戲狀態管理） |
| 狀態 | Completed |

## 目標

實現 GM 主導的場景切換引擎與骰子系統，建立 WebSocket incoming message 處理管線，加入權限過濾廣播（GM 看 `gm_notes`，Player 看不到）。

## 範圍

### In Scope
- Scenario content JSON → Go types 解析（`ScenarioContent`, `Scene`, `Transition`, `Item`, `NPC` 等）
- 骰子引擎（`NdS`, `NdS+M`, `NdS-M`, `dS` 格式，`crypto/rand`）
- GameState 擴展（`current_scene`, `players`, `dice_history`）
- WebSocket incoming message dispatch（`handleIncoming`）
- `advance_scene` action（GM only，場景切換 + 驗證 + gm_notes 過濾廣播）
- `dice_roll` action（GM + Player，骰子擲骰 + 結果廣播）
- `ScenarioLoader` interface（consumer-side，bridges repos → realtime）
- 權限過濾廣播（`broadcastFiltered` + `filterScenePayload`）

### Out of Scope
- Player choice transitions（`trigger: player_choice`）
- Variable system（`set_var`, `var()` expr）
- `on_enter` / `on_exit` actions
- Item reveal / NPC field reveal
- expr-lang/expr 條件引擎
- 快照優化（50 事件快照）

## WebSocket Actions

### Incoming Message Format

```json
{
  "type": "advance_scene | dice_roll",
  "payload": { ... }
}
```

### advance_scene（GM Only）

**Request payload:**
```json
{
  "type": "advance_scene",
  "payload": {
    "scene_id": "library"
  }
}
```

**Broadcast event type:** `scene_changed`

**Broadcast payload（GM）：**
```json
{
  "scene_id": "library",
  "previous_scene": "entrance",
  "scene": {
    "id": "library",
    "name": "圖書館",
    "content": "成排的書架直達天花板...",
    "gm_notes": "書桌上有日記碎片...",
    "items_available": ["torn_diary"],
    "npcs_present": [],
    "transitions": [...]
  }
}
```

**Broadcast payload（Player）：** 同上但移除 `gm_notes` 欄位

**錯誤情境：**
- 非 GM 發送 → `error` envelope, `"Only the GM can advance the scene"`
- Session 非 active → `error` envelope, `"Game is not active"`
- `scene_id` 為空 → `error` envelope, `"scene_id is required"`
- Scene 不存在 → `error` envelope, `"Scene not found: {scene_id}"`
- Scenario 未載入 → `error` envelope, `"Scenario not loaded"`

### dice_roll（GM + Player）

**Request payload:**
```json
{
  "type": "dice_roll",
  "payload": {
    "formula": "2d6+3",
    "purpose": "感知檢定"
  }
}
```

**Broadcast event type:** `dice_rolled`

**Broadcast payload：**
```json
{
  "roller_id": "user-uuid",
  "formula": "2d6+3",
  "results": [4, 2],
  "modifier": 3,
  "total": 9,
  "purpose": "感知檢定"
}
```

**錯誤情境：**
- Session 非 active → `error` envelope, `"Game is not active"`
- `formula` 為空 → `error` envelope, `"formula is required"`
- Formula 格式錯誤 → `error` envelope, `"Invalid dice formula: {detail}"`

## Dice Formula Syntax

| 格式 | 範例 | 說明 |
|------|------|------|
| `NdS` | `2d6` | 擲 N 個 S 面骰 |
| `NdS+M` | `1d20+3` | 擲 N 個 S 面骰 + 修正值 |
| `NdS-M` | `3d8-2` | 擲 N 個 S 面骰 - 修正值 |
| `dS` | `d20` | = `1dS`（省略 1） |

**限制：**
- Dice count (N): 1-100
- Sides (S): 2-1000
- 隨機源：`crypto/rand`

## GameState 擴展

```go
type GameState struct {
    SessionID    string                 `json:"session_id"`
    Status       string                 `json:"status"`
    CurrentScene string                 `json:"current_scene,omitempty"`
    Players      map[string]PlayerState `json:"players,omitempty"`
    DiceHistory  []DiceResult           `json:"dice_history,omitempty"`
    LastSequence int64                  `json:"last_sequence"`
}

type PlayerState struct {
    UserID       string `json:"user_id"`
    CurrentScene string `json:"current_scene"`
}
```

### Apply() 新 event 處理

| Event Type | Payload 欄位 | State 更新 |
|------------|-------------|-----------|
| `scene_changed` | `scene_id`, `previous_scene` | `CurrentScene = scene_id` |
| `dice_rolled` | `DiceResult` fields | append to `DiceHistory` |

## Architecture

### Event Processing Pipeline（場景切換）

```
Client → incoming channel → Room goroutine
    ↓
handleIncoming() → parse JSON → switch type
    ↓
handleAdvanceScene()
    ├── 權限檢查（GM only）
    ├── 狀態檢查（must be active）
    ├── 解析 payload（scene_id）
    ├── 驗證 scene 存在於 scenario
    ├── 建立 event payload
    ├── assign seq → persist → apply state
    └── broadcastFiltered(filterScenePayload)
         ├── GM: 完整 payload（含 gm_notes）
         └── Player: 移除 gm_notes
```

### ScenarioLoader Interface

```go
// 定義在 realtime 包（consumer-side interface）
type ScenarioLoader interface {
    LoadScenarioForSession(ctx context.Context, sessionID string) (*ScenarioContent, error)
}
```

實作在 server 包（`scenarioLoaderAdapter`），組合 `SessionRepository` + `ScenarioRepository`。

## Edge Cases

| 情境 | 處理 |
|------|------|
| ScenarioLoader 失敗 | Room 仍建立，lifecycle events 正常；advance_scene 回 error |
| Persist event 失敗 | State 不更新，回 error 給發送者，sequence 回滾 |
| 場景不存在於 scenario | 回 error 給 GM |
| Game 非 active 時操作 | 回 error |
| Player 嘗試 advance_scene | 回 error（GM only） |
| Unknown action type | 回 error 給發送者 |
| Invalid JSON incoming | 回 error 給發送者 |

## 驗收條件

1. Scenario content JSON 可正確解析為 Go types（含 scenes, items, NPCs, transitions）
2. 骰子引擎支援 `NdS`, `NdS+M`, `NdS-M`, `dS` 格式，使用 `crypto/rand`
3. GM 透過 WebSocket 發送 `advance_scene` 後，所有 Client 收到 `scene_changed` event
4. Player 收到的 `scene_changed` payload 不含 `gm_notes`
5. GM 和 Player 皆可發送 `dice_roll`，所有 Client 收到 `dice_rolled` event
6. GameState 正確更新 `current_scene` 和 `dice_history`
7. Event 正確 persist 到 `game_events` 表
8. `go test ./... -race` 全部通過，新增 ~68 個測試
9. 現有 194 個 unit tests 不受影響
