# SPEC-020：Retroactive — Inventory, Voting & Real-time Features

> 回溯補建：記錄已實作但未建 SPEC 的功能。

| 欄位 | 內容 |
|------|------|
| **規格 ID** | SPEC-020 |
| **關聯 ADR** | ADR-003（劇本資料模型）、ADR-004（遊戲狀態管理） |
| **估算複雜度** | 高 |
| **建議模型** | Opus |
| **HITL 等級** | standard |
| **狀態** | Completed（回溯補建） |

---

## 🎯 目標（Goal）

> 補齊在 SPEC-007/008 之後實作但未建立規格書的六項功能：背包系統、投票系統、即時轉場更新、GM 道具筆記、set_variable WebSocket action、角色追蹤至 GameState。

---

## 📥 輸入規格（Inputs）

### 功能 A：背包系統（Player Inventory）

| 參數名稱 | 型別 | 來源 | 說明 |
|----------|------|------|------|
| give_item action | WS Action (GM-only) | WebSocket | `{item_id, player_id?, player_ids?, quantity?}` |
| remove_item action | WS Action (GM-only) | WebSocket | `{item_id, player_id?, player_ids?, quantity?}` |
| give_item scene action | ScenarioContent | on_enter/on_exit | `{item_id, to, quantity?}` |
| remove_item scene action | ScenarioContent | on_enter/on_exit | `{item_id, from, quantity?}` |

### 功能 B：投票系統（Player Votes）

| 參數名稱 | 型別 | 來源 | 說明 |
|----------|------|------|------|
| player_choice action | WS Action (Player) | WebSocket | `{transition_index}` — 同 SPEC-007，但現在記錄投票而非直接轉場 |

### 功能 C：即時轉場更新（Transitions Updated）

> 無額外輸入。當 `variable_changed`、`item_given`、`item_removed` 事件發生後，Server 自動重算所有 client 的可見轉場。

### 功能 D：GM 道具筆記

| 參數名稱 | 型別 | 來源 | 說明 |
|----------|------|------|------|
| gm_notes | string | ScenarioContent (Item) | GM-only 描述，劇本 JSON 中定義 |

### 功能 E：set_variable WebSocket Action

| 參數名稱 | 型別 | 來源 | 說明 |
|----------|------|------|------|
| set_variable action | WS Action (GM-only) | WebSocket | `{name, value}` |

### 功能 F：角色追蹤至 GameState

| 參數名稱 | 型別 | 來源 | 說明 |
|----------|------|------|------|
| character_id | string | Client 連線時 | 從 SessionPlayer.CharacterID 載入 |
| character_name | string | Client 連線時 | 從 Character.Name 載入 |

---

## 📤 輸出規格（Expected Output）

### 功能 A：背包系統

**新增事件：**

| Event | Payload | 目標 |
|-------|---------|------|
| `item_given` | `{item_id, player_ids, quantity}` | 目標玩家 + GM |
| `item_removed` | `{item_id, player_ids, quantity}` | 目標玩家 + GM |

**GameState 擴展：**

| 欄位 | 說明 |
|------|------|
| `PlayerInventory map[string][]InventoryEntry` | playerID → entries（取代 RevealedItems 作為主要道具追蹤） |

**InventoryEntry：**
```json
{"item_id": "key", "quantity": 3}
```

**Stackable 邏輯：**
- `stackable: false`（預設）→ 數量永遠為 1，重複 give 會被拒絕
- `stackable: true` → 可累加數量

**向後相容：** `item_revealed` 事件的 Apply 同時寫入 PlayerInventory（qty=1）

### 功能 B：投票系統

| Event | Payload | 目標 |
|-------|---------|------|
| `player_votes` | `{scene_id, votes: {transition_index: count}}` | 全體 |

投票會在場景切換時重置。

### 功能 C：即時轉場更新

| Event | Payload | 目標 |
|-------|---------|------|
| `transitions_updated` | `{scene_id, transitions: FilteredTransition[]}` | Per-player（每人收到自己可見的轉場列表） |

觸發時機：`handleSetVariable`、`handleGiveItem`、`handleRemoveItem`、`executeAndPersistActions` 完成後。

### 功能 D：GM 道具筆記

**Item struct 擴展：**
```go
type Item struct {
    // ... 既有欄位 ...
    GMNotes   string `json:"gm_notes,omitempty"`  // GM-only 描述
    Stackable bool   `json:"stackable,omitempty"` // 可堆疊
}
```

`gm_notes` 僅在 GM 端顯示，玩家端過濾掉。

### 功能 E：set_variable

GM 透過 WebSocket 直接修改劇本變數，觸發 `variable_changed` 事件廣播。

### 功能 F：角色追蹤

**PlayerState 擴展：**
```go
type PlayerState struct {
    // ... 既有欄位 ...
    CharacterID   string `json:"character_id,omitempty"`
    CharacterName string `json:"character_name,omitempty"`
}
```

**Client struct 擴展：** `characterID`、`characterName` 欄位 + `SetCharacter()` 方法。

**player_joined payload 擴展：** 包含 `character_id`、`character_name`。

### 功能 G：表達式函式擴展

| 函式 | 簽名 | 說明 |
|------|------|------|
| `item_count` | `item_count(item_id: string) → int` | 回傳觸發玩家持有該道具的數量 |

`has_item` 和 `all_have_item` 改為也檢查 `PlayerInventory`。

---

## 🔗 副作用與連動（Side Effects）

| 變更的狀態 / 資源 | 受影響的模組或功能 | 處理方式 |
|--------------------|---------------------|----------|
| GameState.PlayerInventory | 表達式引擎 (has_item, all_have_item, item_count) | 已更新為使用 HasItem/ItemQuantity |
| GameState.PlayerInventory | Per-player 場景過濾 (filterScenePayloadPerClient) | 已更新 |
| variable_changed / item_given / item_removed | 轉場條件 | 已實作 refreshClientTransitions() |
| PlayerState.CharacterID/CharacterName | player_joined 事件 payload | 已包含 |
| Item.GMNotes | GM/Player 端顯示過濾 | GM 可見、Player 端不顯示 |

---

## ⚠️ 邊界條件（Edge Cases）

- give_item：非 stackable 且玩家已持有 → 送 error
- remove_item：quantity=0 → 移除全部
- remove_item：玩家未持有 → 送 error
- 投票：轉場 condition 為 false → 該轉場不顯示，不可投票
- transitions_updated：玩家在不同場景 → 只發送當前場景的轉場
- set_variable：變數不在劇本 variables 定義中 → 仍允許設定（GM 靈活性）
- item_revealed 事件重播 → 同時寫入 PlayerInventory（向後相容）

---

## ✅ 驗收標準（Done When）

- [x] `give_item` / `remove_item` WebSocket actions 實作完成（GM-only）
- [x] `give_item` / `remove_item` scene actions (on_enter/on_exit) 實作完成
- [x] `InventoryEntry` struct + `PlayerInventory` GameState 欄位
- [x] stackable 邏輯正確（非 stackable 拒絕重複 give）
- [x] `item_revealed` 向後相容寫入 PlayerInventory
- [x] `player_votes` 事件廣播 + 投票重置邏輯
- [x] `transitions_updated` 即時推送（handleSetVariable / handleGiveItem / handleRemoveItem / executeAndPersistActions 後觸發）
- [x] `Item.GMNotes` / `Item.Stackable` 欄位 + GM/Player 端過濾
- [x] `set_variable` WebSocket action (GM-only)
- [x] `PlayerState.CharacterID` / `CharacterName` + `player_joined` payload
- [x] `item_count` 表達式函式
- [x] `has_item` / `all_have_item` 改用 HasItem（檢查 PlayerInventory）
- [x] `go test ./... -race` 全部通過
- [x] 前端 game-store 處理 `item_given` / `item_removed` / `player_votes` / `transitions_updated` 事件
- [x] GM 道具面板：給予/移除按鈕、角色背包總覽
- [x] 玩家背包：顯示 PlayerInventory + 數量 badge
- [x] 劇本編輯器：Item 新增 gm_notes / stackable 欄位
- [x] 副作用涉及的模組皆已測試或人工確認

---

## 🚫 禁止事項（Out of Scope）

- 不修改 NPC 系統
- 不修改 REST API（道具全在 WebSocket 層處理）
- 不修改 DB schema（PlayerInventory 存在 GameState JSONB 內）

---

## 📎 參考資料（References）

- SPEC-007（player_choice、reveal_item 基礎）
- SPEC-008（表達式引擎、condition_met、snapshot）
- ADR-003（劇本資料模型）
- ADR-004（遊戲狀態管理 — 事件類型將據此更新）
