# SPEC-008：Expression Engine, Conditional Transitions, Snapshots & GM Broadcast

> expr-lang/expr 條件引擎、自動/條件轉場、快照系統、GM 廣播。

| 欄位 | 內容 |
|------|------|
| **規格 ID** | SPEC-008 |
| **關聯 ADR** | ADR-003（劇本資料模型與 DSL）、ADR-004（遊戲狀態管理） |
| **估算複雜度** | 高 |
| **建議模型** | Opus |
| **HITL 等級** | standard |
| **狀態** | Completed |

---

## 🎯 目標（Goal）

> 整合 `expr-lang/expr` 表達式引擎，支援條件轉場（`condition_met`）與自動轉場（`auto`），讓劇本能根據遊戲狀態動態控制流程。同時實作快照系統降低 event replay 成本，並新增 GM Broadcast 推送功能。

---

## 📥 輸入規格（Inputs）

### Expression Engine 注入函式

| 函式 | 簽名 | 說明 |
|------|------|------|
| `has_item` | `has_item(item_id) → bool` | 當前玩家是否持有道具 |
| `roll` | `roll(formula) → int` | 執行骰子投擲 |
| `attr` | `attr(name) → any` | 讀取角色屬性 |
| `var` | `var(name) → any` | 讀取場景變數 |
| `all_have_item` | `all_have_item(item_id) → bool` | 所有玩家是否持有道具 |
| `player_count` | `player_count() → int` | 當前連線玩家數 |

### Transition Triggers

| Trigger | 說明 |
|---------|------|
| `condition_met` | 當 `condition` expr 求值為 true 時觸發 |
| `auto` | 場景進入後自動執行（chain depth limit: 10） |

### WebSocket Actions

| Action | 角色 | Payload |
|--------|------|---------|
| `gm_broadcast` | GM | `{"content": "...", "image_url": "...", "player_ids": ["p1"]}` |

### set_var 表達式支援

```yaml
set_var:
  name: "anger"
  expr: "var('anger') + 1"  # 新增 Expr 欄位
```

---

## 📤 輸出規格（Expected Output）

### Broadcast Events

| Event | Payload | 目標 |
|-------|---------|------|
| `gm_broadcast` | `{content, image_url, player_ids}` | 目標玩家 + GM |

### Snapshot System

| 行為 | 說明 |
|------|------|
| 觸發條件 | 每 50 個 event 自動觸發 |
| 儲存位置 | `game_sessions.state` + `game_sessions.snapshot_seq` |
| 恢復流程 | `LoadSnapshot` → unmarshal → `ListEventsSince` → Apply each |

---

## ⚠️ 邊界條件（Edge Cases）

- expr 執行超時（100ms）→ EvalBool 回傳 false
- auto transition chain depth > 10 → 中止，防止無窮迴圈
- gm_broadcast 缺少 content 和 image_url → error
- Player 嘗試 gm_broadcast → 拒絕（GM only）
- Snapshot save 失敗 → log warning，不中斷遊戲
- condition_met expr 語法錯誤 → log warning，跳過轉場

---

## ✅ 驗收標準（Done When）

- [x] expr-lang/expr 整合完成，6 個注入函式可用
- [x] `condition_met` 觸發器正確求值並觸發轉場
- [x] `auto` 觸發器支援鏈式跳轉（max depth 10）
- [x] `set_var` 支援 Expr 欄位
- [x] Snapshot 每 50 事件自動保存
- [x] `RecoverFromSnapshot` 可從 snapshot + events 恢復狀態
- [x] `gm_broadcast` WebSocket action 正常運作
- [x] `go test ./... -race` 全部通過（381 tests, 0 races）
- [x] 已更新 `docs/api-changelog.md`

---

## 🚫 禁止事項（Out of Scope）

- Snapshot SQL table 修正（→ SPEC-009）
- Hub RecoverFromSnapshot 接線（→ SPEC-009）
- OpenAPI spec 更新（→ SPEC-009）
