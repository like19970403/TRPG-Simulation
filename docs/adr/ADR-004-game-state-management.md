# [ADR-004]: 遊戲狀態管理

| 欄位 | 內容 |
|------|------|
| **狀態** | `Accepted` |
| **日期** | 2026-02-27 |
| **決策者** | 專案擁有者 |

---

## 背景（Context）

TRPG-Simulation 的遊戲進行過程中，需要管理多種狀態：當前場景、玩家位置、已揭露道具、劇本變數、骰子紀錄等。需要決定狀態管理的架構模式，確保：

1. **斷線恢復**：玩家斷線重連後能看到正確狀態
2. **GM 審計**：GM 能回顧遊戲過程中的所有事件
3. **狀態一致性**：多個玩家看到的狀態一致（經權限過濾後）
4. **遊戲回放**：未來可支援遊戲紀錄回放

---

## 評估選項（Options Considered）

### 選項 A：Event Sourcing

- **優點**：完整事件歷史、天然支援回放和斷線恢復、可從事件重建任意時間點的狀態、GM 審計簡單
- **缺點**：需實作事件重放邏輯、狀態查詢需從事件聚合（可用快照優化）
- **風險**：低。TRPG 遊戲事件頻率不高（每秒 < 10 事件），不需要 CQRS 等複雜模式

### 選項 B：State Snapshot（僅儲存最新狀態）

- **優點**：實作簡單、查詢直接
- **缺點**：無事件歷史、無法回放、斷線恢復只能靠最新快照（可能丟失中間事件）、GM 無法審計
- **風險**：中。失去 TRPG 最有價值的遊戲紀錄

### 選項 C：CQRS + Event Sourcing

- **優點**：讀寫分離、讀取效能最佳
- **缺點**：需維護讀模型的投影（projection）、最終一致性、開發成本高
- **風險**：高。個人專案不需要這個複雜度

---

## 決策（Decision）

選擇 **選項 A：Event Sourcing（簡化版）**，搭配定期快照優化。

### GameSession 狀態機

```
lobby → active → paused → active → completed
  │                                    ↑
  └──────────── abandoned ─────────────┘
```

| 狀態 | 說明 | 允許操作 |
|------|------|----------|
| `lobby` | GM 建立遊戲，等待玩家加入 | 玩家加入/離開、GM 修改設定 |
| `active` | 遊戲進行中 | 場景切換、道具揭露、骰子檢定、玩家選擇、GM 投放 |
| `paused` | GM 暫停遊戲 | 僅查看目前狀態（場景、道具、紀錄） |
| `completed` | 遊戲正常結束 | 僅查看紀錄 |
| `abandoned` | GM 放棄遊戲 | 僅查看紀錄 |

### 事件模型

所有遊戲動作記錄為 `GameEvent`：

```sql
CREATE TABLE game_events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id  UUID NOT NULL REFERENCES game_sessions(id),
    sequence    BIGINT NOT NULL,  -- 單調遞增序號，用於排序和斷線重連
    type        VARCHAR(50) NOT NULL,
    actor_id    UUID,  -- 觸發者（GM 或玩家），系統事件為 NULL
    payload     JSONB NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(session_id, sequence)
);

CREATE INDEX idx_game_events_session_seq ON game_events(session_id, sequence);
```

### 事件類型

#### 持久化事件（寫入 game_events 表）

| 事件類型 | 觸發者 | payload 範例 | Apply 行為 |
|----------|--------|-------------|------------|
| `game_started` | GM | `{}` | status → active |
| `game_paused` | GM | `{"reason": "休息時間"}` | status → paused |
| `game_resumed` | GM | `{}` | status → active |
| `game_ended` | GM | `{"reason": "劇本完成"}` | status → completed |
| `scene_changed` | GM/System | `{"scene_id": "library", "previous_scene": "entrance"}` | 更新 CurrentScene + 所有線上玩家 CurrentScene |
| `item_revealed` | GM/System | `{"item_id": "rusty_key", "player_ids": ["uuid"]}` | 寫入 RevealedItems + PlayerInventory（向後相容） |
| `item_given` | GM/System | `{"item_id": "key", "player_ids": ["uuid"], "quantity": 1}` | 加入/堆疊到 PlayerInventory |
| `item_removed` | GM/System | `{"item_id": "key", "player_ids": ["uuid"], "quantity": 1}` | 扣減/移除 PlayerInventory（qty=0 全部移除） |
| `npc_field_revealed` | GM/System | `{"npc_id": "old_butler", "field_key": "personality", "player_ids": ["uuid"]}` | 寫入 RevealedNPCFields |
| `dice_rolled` | GM/Player | `{"formula": "2d6", "results": [3, 5], "total": 8, "purpose": "perception_check"}` | 追加 DiceHistory |
| `variable_changed` | GM/System | `{"name": "ghost_anger", "old_value": 0, "new_value": 1}` | 更新 Variables |
| `player_choice` | Player | `{"scene_id": "entrance", "transition_label": "前往圖書館", "target_scene": "library"}` | 審計記錄，無狀態變更 |
| `gm_broadcast` | GM | `{"content": "你聽到遠處傳來低語...", "image_url": null, "player_ids": ["uuid"]}` | 審計記錄，無狀態變更 |
| `player_joined` | System | `{"user_id": "uuid", "username": "Alice", "character_id": "uuid", "character_name": "艾倫", "attributes": {...}}` | 寫入 Players + PlayerAttributes |
| `player_left` | System | `{"user_id": "uuid", "reason": "disconnected"}` | 標記 Online=false |

#### 瞬態事件（僅 WebSocket 廣播，不持久化）

| 事件類型 | 說明 |
|----------|------|
| `state_sync` | 連線時全量狀態同步（完整 GameState JSON） |
| `player_votes` | 轉場投票統計（`{scene_id, votes: {idx: count}}`） |
| `transitions_updated` | 狀態變更後 per-player 轉場條件重算結果 |
| `error` | 錯誤回應 |

> **設計決策**：原規劃的 `action_executed` 事件未實作。實際實作中，場景 action（set_var、give_item 等）直接產生具體事件（`variable_changed`、`item_given`），比泛用的 wrapper 事件更有用。

### 遊戲狀態結構（記憶體中）

```
GameState
├── SessionID
├── Status            (lobby / active / paused / completed / abandoned)
├── CurrentScene      (scene_id)
├── Players map[PlayerID]
│   ├── UserID
│   ├── Username
│   ├── CharacterID   (Session 中綁定的角色 ID)
│   ├── CharacterName (角色名稱，顯示用)
│   ├── CurrentScene
│   └── Online        (bool，斷線時標記為 false)
├── PlayerAttributes map[PlayerID]map[AttrName]any   (角色屬性快取)
├── RevealedItems map[PlayerID][]ItemID               (legacy，向後相容)
├── PlayerInventory map[PlayerID][]InventoryEntry     (背包系統，主要道具追蹤)
│   └── InventoryEntry { ItemID, Quantity }
├── RevealedNPCFields map[PlayerID]map[NPCID][]FieldKey
├── Variables map[string]any
├── DiceHistory []DiceResult
└── LastSequence      (最新事件序號)
```

> **斷線重連**：`lastEventSeq` 由 WebSocket Client 自行追蹤（非 GameState 欄位），連線時帶上，Server 重放缺失事件。

### 狀態重建流程

1. **遊戲啟動**：從 `game_events` 表讀取該 session 所有事件，依序重放建立 `GameState`
2. **遊戲進行**：每個新事件先持久化到 DB，再更新記憶體中的 `GameState`，最後透過 WebSocket 廣播
3. **斷線重連**：Client 帶上 `lastEventSeq`，Server 從 DB 查詢 `sequence > lastEventSeq` 的事件重放給 Client

### 快照策略

- 每 50 個事件自動建立一個快照（序列化 `GameState` 為 JSONB）
- 快照存入 `game_sessions.state` 欄位
- 狀態重建時：先載入最新快照，再重放快照之後的事件
- 遊戲結束時建立最終快照

```sql
-- game_sessions 表中的 state 欄位存快照
ALTER TABLE game_sessions ADD COLUMN snapshot_seq BIGINT DEFAULT 0;
-- state JSONB 已存在於 game_sessions 表
```

### 事件處理流程

```
Player/GM 動作
     │
     ▼
  驗證動作合法性
  （狀態機 + 權限檢查）
     │
     ▼
  分配 sequence 序號
     │
     ▼
  持久化 GameEvent 到 DB
     │
     ▼
  更新記憶體 GameState
     │
     ▼
  透過 Room 廣播
  （GM 收完整事件，Player 收過濾後事件）
```

### 並發控制

- 每個 Room（GameSession）由單一 goroutine 處理所有事件（channel-based）
- 無需 mutex：所有狀態修改都在 Room goroutine 中序列化執行
- sequence 序號由 Room goroutine 單調分配，確保順序一致性

---

## 後果（Consequences）

**正面影響：**
- 完整事件歷史，支援 GM 審計和未來遊戲回放
- 斷線恢復基於事件重放，可靠且精確
- 單 goroutine 處理避免並發問題，邏輯簡單
- 快照優化避免長遊戲重建時間過長

**負面影響 / 技術債：**
- 事件表資料量隨遊戲進行增長（每場遊戲預估 100-1000 事件，可接受）
- 需實作事件重放邏輯（每種事件類型的 apply 函式）
- 快照格式變更時需處理向後相容

**後續追蹤：**
- [x] SPEC-005~008：各事件類型的 apply 函式和驗證規則（已完成）
- [x] SPEC-009：快照系統修正與 Hub 整合（已完成）
- [ ] 未來考慮：遊戲回放 UI

---

## 關聯（Relations）

- 取代：（無）
- 被取代：（無）
- 參考：ADR-001（技術棧選型）、ADR-002（即時通訊策略）、ADR-003（劇本資料模型）
