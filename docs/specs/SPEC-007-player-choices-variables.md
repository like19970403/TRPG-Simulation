# SPEC-007：Player Choices, Variables & Item/NPC Reveal

> 玩家選擇、場景變數、道具/NPC 揭露系統。

| 欄位 | 內容 |
|------|------|
| **規格 ID** | SPEC-007 |
| **關聯 ADR** | ADR-003（劇本資料模型與 DSL）、ADR-004（遊戲狀態管理） |
| **估算複雜度** | 高 |
| **建議模型** | Opus |
| **HITL 等級** | standard |
| **狀態** | Completed |

---

## 🎯 目標（Goal）

> 讓玩家能在場景中做出選擇（`player_choice` 觸發器），GM 能手動揭露道具與 NPC 欄位，並支援場景變數系統（`on_enter`/`on_exit` hooks）。實現 per-player 場景過濾，讓不同玩家看到不同的道具與 NPC。

---

## 📥 輸入規格（Inputs）

### WebSocket Actions

| Action | 角色 | Payload |
|--------|------|---------|
| `player_choice` | Player | `{"transition_index": 0}` |
| `reveal_item` | GM | `{"item_id": "key", "player_ids": ["p1", "p2"]}` |
| `reveal_npc_field` | GM | `{"npc_id": "butler", "field_key": "secret", "player_ids": ["p1"]}` |

### Scenario DSL 擴展

| 欄位 | 說明 |
|------|------|
| `variables[]` | 場景變數定義（name, type, default） |
| `scenes[].on_enter[]` | 進入場景時執行的 actions（set_var, reveal_item, reveal_npc_field） |
| `scenes[].on_exit[]` | 離開場景時執行的 actions |
| `transitions[].trigger` | `player_choice` 觸發器類型 |

---

## 📤 輸出規格（Expected Output）

### Broadcast Events

| Event | Payload | 目標 |
|-------|---------|------|
| `player_choice` | `{user_id, transition_index, scene_id, previous_scene}` | 全體 |
| `item_revealed` | `{item_id, player_ids}` | 目標玩家 + GM |
| `npc_field_revealed` | `{npc_id, field_key, field_value, player_ids}` | 目標玩家 + GM |
| `variable_changed` | `{name, old_value, new_value}` | 全體 |

### GameState 擴展

| 欄位 | 說明 |
|------|------|
| `Variables map[string]any` | 場景變數 |
| `RevealedItems map[string][]string` | playerID → []itemID |
| `RevealedNPCFields map[string]map[string][]string` | playerID → npcID → []fieldKey |

---

## ⚠️ 邊界條件（Edge Cases）

- Player 嘗試 reveal_item/reveal_npc_field → 拒絕（GM only）
- transition_index 超出範圍 → error
- Transition trigger 非 player_choice → error
- 重複揭露同一道具 → 冪等（不重複加入）
- on_enter action 執行失敗 → log warning，不阻斷場景切換
- Scenario 未載入時 player_choice → error

---

## ✅ 驗收標準（Done When）

- [x] `player_choice` WebSocket action 實作完成，含場景轉場
- [x] `reveal_item` / `reveal_npc_field` GM-only actions 實作完成
- [x] on_enter / on_exit action 系統（set_var, reveal_item, reveal_npc_field）
- [x] Per-player 場景過濾（items_available, npcs_present）
- [x] Scenario variables 初始化（Variables → GameState）
- [x] `go test ./... -race` 全部通過
- [x] 已更新 `docs/api-changelog.md`

---

## 🚫 禁止事項（Out of Scope）

- expr-lang/expr 條件引擎（→ SPEC-008）
- Snapshot 系統（→ SPEC-008）
- GM Broadcast（→ SPEC-008）
- auto / condition_met 觸發器（→ SPEC-008）
