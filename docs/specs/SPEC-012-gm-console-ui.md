# SPEC-012：GM Console UI

> GM 遊戲主持介面 — 三欄佈局 + 底部日誌列、WebSocket 即時同步。

| 欄位 | 內容 |
|------|------|
| **規格 ID** | SPEC-012 |
| **關聯 ADR** | ADR-001（技術棧）、ADR-003（劇本格式）、ADR-004（遊戲狀態管理） |
| **關聯 SPEC** | SPEC-004（Game Session）、SPEC-005（WebSocket）、SPEC-006（場景/骰子）、SPEC-011（React SPA Phase 1） |
| **估算複雜度** | 高 |
| **建議模型** | Opus |
| **HITL 等級** | standard |
| **狀態** | Completed |

---

## 🎯 目標（Goal）

> 實作 GM Console 前端頁面，讓 GM 能透過 WebSocket 即時主持遊戲。包含：三欄佈局（玩家列表 | 場景中心 | 道具與線索）、底部事件日誌、場景推進控制、骰子擲骰、道具/NPC 揭露、GM 廣播、遊戲生命週期控制（暫停/結束）。UI 設計已於 Pencil 確認（`docs/designs/pencil-new.pen` GM Console frame）。

---

## 📐 技術決策

| 項目 | 選擇 |
|------|------|
| 即時同步 | WebSocket（`/api/v1/sessions/{id}/ws?token={jwt}`） |
| 狀態管理 | Zustand store（game-store） |
| 重連機制 | `last_event_seq` 參數，斷線自動重連 + 事件回放 |
| 場景內容 | Markdown 渲染（GM 可見 `gm_notes`） |

---

## 📥 輸入規格（Inputs）

### WebSocket 連線

```
GET /api/v1/sessions/{id}/ws?token={jwt}&last_event_seq={seq}
```

- 連線後收到 `state_sync` 事件，包含完整 `GameState`
- Session 必須處於 `active` 或 `paused` 狀態
- 使用者必須是該 session 的 GM

### GM 可發送的 Actions（Client → Server）

| Action | Payload | 說明 |
|--------|---------|------|
| `advance_scene` | `{ scene_id: string }` | 切換場景 |
| `dice_roll` | `{ formula: string, purpose?: string }` | 擲骰（如 `"2d6"`, `"d20+5"`） |
| `reveal_item` | `{ item_id: string, player_ids?: string[] }` | 揭露道具給指定玩家（空 = 全部） |
| `reveal_npc_field` | `{ npc_id: string, field_key: string, player_ids?: string[] }` | 揭露 NPC 屬性 |
| `gm_broadcast` | `{ content?: string, image_url?: string, player_ids?: string[] }` | 推送訊息/圖片 |

### GM 接收的 Events（Server → Client）

| Event | 說明 |
|-------|------|
| `state_sync` | 首次連線 — 完整 GameState |
| `scene_changed` | 場景切換（GM 看到完整場景 + gm_notes） |
| `dice_rolled` | 骰子結果 |
| `item_revealed` | 道具揭露確認 |
| `npc_field_revealed` | NPC 屬性揭露確認 |
| `variable_changed` | 變數變更（on_enter/on_exit 觸發） |
| `player_choice` | 玩家做出選擇 |
| `game_paused` / `game_resumed` / `game_ended` | 生命週期事件 |
| `error` | 錯誤訊息 |

### REST API（Session 生命週期控制）

| Method | Path | 說明 |
|--------|------|------|
| GET | `/api/v1/sessions/{id}` | 取得 session 資訊 |
| GET | `/api/v1/sessions/{id}/players` | 取得玩家列表 |
| POST | `/api/v1/sessions/{id}/pause` | 暫停遊戲 |
| POST | `/api/v1/sessions/{id}/resume` | 恢復遊戲 |
| POST | `/api/v1/sessions/{id}/end` | 結束遊戲 |

---

## 📤 輸出規格（Expected Output）

### 頁面路由

```
/sessions/{id}/gm → AuthGuard → GMGuard → AppLayout → GMConsolePage
```

### 佈局結構（參照 Pencil 設計）

```
┌─────────────────────────────────────────────────────────────┐
│ [Top Bar]  📜 TRPG · {scenario.title}    [Pause] [End Game]│
├─────────────────────────────────────────────────────────────┤
│ Players (260px) │ Scene Area (flex)     │ Items & Clues     │
│                 │                       │ (300px)           │
│ ● Luna          │ ⚔ Main Hall           │ ▸ Rusty Key       │
│ ● Kai           │                       │ ▸ Town Letter      │
│ ● Frey          │ [scene content]       │ ▸ Ancient Map      │
│                 │ [gm_notes section]    │                   │
│                 │                       │ NPCs              │
│                 │ [Scene Transitions]   │ ▸ Old Merchant     │
│                 │ [Enter Library] [...]│                   │
├─────────────────────────────────────────────────────────────┤
│ [Events] [Dice Log] [Broadcast]                             │
│ Events log / Dice history / Broadcast input                 │
│                                                    [Send ▶]│
└─────────────────────────────────────────────────────────────┘
```

### 新增目錄結構

```
web/src/
  api/
    sessions.ts              # Session REST API calls
    websocket.ts             # WebSocket connection manager
  stores/
    game-store.ts            # GameState + events Zustand store
  hooks/
    use-game-socket.ts       # WebSocket hook（connect/disconnect/send）
  components/
    gm/
      player-panel.tsx       # 左欄 — 玩家列表
      scene-panel.tsx        # 中欄 — 場景內容 + 轉場按鈕
      items-panel.tsx        # 右欄 — 道具/NPC 列表 + 揭露按鈕
      event-log.tsx          # 底部 — 事件日誌
      dice-log.tsx           # 底部 — 骰子紀錄 + 擲骰 UI
      broadcast-panel.tsx    # 底部 — GM 廣播
      gm-top-bar.tsx         # 頂部導航列 + 控制按鈕
      gm-guard.tsx           # 檢查用戶是否為 GM
  pages/
    gm-console-page.tsx      # GM Console 主頁面
```

### 功能清單

| 功能 | 說明 |
|------|------|
| **場景顯示** | 顯示當前場景名稱、內容（Markdown）、GM Notes（金色背景區塊） |
| **場景推進** | 顯示可用轉場按鈕（trigger=player_choice 顯示標籤，auto/condition_met 由 GM 手動觸發） |
| **玩家列表** | 顯示連線狀態（綠點/灰點）、角色名稱、當前場景 |
| **道具列表** | 顯示場景內道具（已揭露標記 ✓），點擊展開揭露面板（選擇玩家） |
| **NPC 列表** | 顯示場景內 NPC，展開顯示欄位，hidden 欄位可揭露 |
| **骰子** | 輸入公式（如 `2d6+3`）、顯示歷史紀錄、結果動畫 |
| **GM 廣播** | 文字輸入 + 可選圖片 URL，選擇目標玩家或全體 |
| **事件日誌** | 時間戳 + 事件類型 + 內容摘要，自動捲動 |
| **遊戲控制** | Pause/Resume 切換、End Game（確認 dialog） |
| **斷線重連** | 自動重連 + `last_event_seq` 回放，顯示連線狀態指示器 |

---

## ⚠️ 邊界條件（Edge Cases）

- WebSocket 斷線 → 顯示黃色 banner + 自動重連（指數退避，最多 30 秒）
- 重連後 `state_sync` 恢復 → 事件序列號對齊
- 場景無可用轉場 → 禁用推進按鈕 + 顯示「End of scenario」
- 併發場景切換（auto-transition chain）→ UI 等待最終 `scene_changed` 事件
- Session 在其他裝置被暫停/結束 → WebSocket 收到事件，UI 即時更新
- GM 廣播空內容 → 前端驗證阻止
- 大量事件日誌 → 虛擬捲動或限制顯示最近 200 條
- 玩家在 GM 揭露道具時斷線 → 重連後 `state_sync` 包含已揭露狀態

---

## ✅ 驗收標準（Done When）

- [ ] GM Console 頁面可透過 `/sessions/{id}/gm` 訪問
- [ ] WebSocket 連線成功建立並接收 `state_sync`
- [ ] 三欄佈局正確渲染：玩家列表、場景區、道具/NPC 列表
- [ ] 場景推進功能正常（advance_scene action）
- [ ] 骰子擲骰功能正常（dice_roll action + 結果顯示）
- [ ] 道具/NPC 揭露功能正常（reveal_item / reveal_npc_field）
- [ ] GM 廣播功能正常
- [ ] 暫停/恢復/結束遊戲功能正常
- [ ] 斷線自動重連 + 事件回放
- [ ] 事件日誌正確顯示所有遊戲事件
- [ ] 非 GM 用戶訪問被 GMGuard 攔截
- [ ] 單元測試 ≥ 15 cases
- [ ] ESLint 無 error

---

## 🚫 禁止事項（Out of Scope）

- 不修改後端 WebSocket 邏輯
- 不實作聊天功能（聊天由 Discord 處理）
- 不實作場景編輯器（屬於 Scenario Manager）
- 不修改現有 Auth / Session REST API
- 不引入新的 CSS 框架（使用現有 Tailwind + Pencil 設計系統）

---

## 📎 參考資料（References）

- UI 設計：`docs/designs/pencil-new.pen`（GM Console frame `xGVw0`）
- WebSocket 實作：`internal/realtime/` — `message.go`（Envelope）、`room.go`（action dispatch）、`client.go`（read/write pump）
- 遊戲狀態：`internal/realtime/gamestate.go`（GameState 結構）
- 場景格式：`internal/realtime/scenario.go`（ScenarioContent / Scene / Item / NPC）
- Session API：`internal/server/session_handlers.go`
- 事件類型：`internal/realtime/message.go`（EventType constants）
