# SPEC-013：Player Game Screen UI

> 玩家遊戲畫面 — 道具欄側邊 + 羊皮紙場景中心、WebSocket 即時同步。

| 欄位 | 內容 |
|------|------|
| **規格 ID** | SPEC-013 |
| **關聯 ADR** | ADR-001（技術棧）、ADR-003（劇本格式）、ADR-004（遊戲狀態管理） |
| **關聯 SPEC** | SPEC-005（WebSocket）、SPEC-006（場景/骰子）、SPEC-011（SPA Phase 1）、SPEC-012（GM Console） |
| **估算複雜度** | 高 |
| **建議模型** | Opus |
| **HITL 等級** | standard |
| **狀態** | Completed |

---

## 🎯 目標（Goal）

> 實作 Player Game Screen 前端頁面，讓玩家透過 WebSocket 即時參與遊戲。包含：左側道具欄（揭露的道具 + 角色屬性）、中央羊皮紙風格場景區（場景內容 + 選擇按鈕 + 骰子）、即時接收 GM 廣播與遊戲事件。UI 設計已於 Pencil 確認（`docs/designs/pencil-new.pen` Player Game Screen frame）。不含聊天功能（由 Discord 處理）。

---

## 📐 技術決策

| 項目 | 選擇 |
|------|------|
| 即時同步 | WebSocket（與 SPEC-012 共用 `use-game-socket` hook） |
| 狀態管理 | Zustand game-store（與 SPEC-012 共用） |
| 場景渲染 | Markdown → HTML，羊皮紙風格（parchment bg + gold border） |
| 道具圖片 | 道具 `image` 欄位直接顯示（GM 上傳的 URL） |

---

## 📥 輸入規格（Inputs）

### WebSocket 連線

```
GET /api/v1/sessions/{id}/ws?token={jwt}&last_event_seq={seq}
```

- 連線後收到 `state_sync` 事件（玩家版本：場景內容已過濾 `gm_notes`，道具/NPC 僅顯示已揭露的）
- Session 必須處於 `active` 或 `paused` 狀態
- 使用者必須是該 session 的玩家（非 GM）

### 玩家可發送的 Actions（Client → Server）

| Action | Payload | 說明 |
|--------|---------|------|
| `player_choice` | `{ transition_index: number }` | 選擇場景轉場（0-based index） |
| `dice_roll` | `{ formula: string, purpose?: string }` | 擲骰 |

### 玩家接收的 Events（Server → Client）

| Event | 說明 |
|-------|------|
| `state_sync` | 首次連線 — 過濾後的 GameState |
| `scene_changed` | 場景切換（已過濾 gm_notes，僅含已揭露道具/NPC） |
| `dice_rolled` | 骰子結果（任何人的骰子對所有人可見） |
| `item_revealed` | 道具被 GM 揭露給該玩家 |
| `npc_field_revealed` | NPC 欄位被揭露 |
| `variable_changed` | 變數變更 |
| `player_choice` | 其他玩家做出選擇 |
| `gm_broadcast` | GM 推送的訊息/圖片 |
| `game_paused` / `game_resumed` / `game_ended` | 生命週期事件 |
| `error` | 錯誤訊息 |

---

## 📤 輸出規格（Expected Output）

### 頁面路由

```
/sessions/{id}/play → AuthGuard → PlayerGuard → PlayerGamePage
```

### 佈局結構（參照 Pencil 設計）

```
┌─────────────────────────────────────────────────────────┐
│ [Top Bar] 📜 TRPG · {scenario.title}  {charName} ⚙   │
├──────────┬──────────────────────────────────────────────┤
│ Inventory│           ┌─────────────────┐                │
│ (240px)  │           │  ⚔ Main Hall    │                │
│          │           │                 │                │
│ 🗝 Rusty │           │ [scene content] │                │
│   Key    │           │  (parchment bg) │                │
│ 📄 Town  │           │                 │                │
│   Letter │           │ [GM Broadcast]  │                │
│          │           │                 │                │
│ ─────── │           │ [Choice 1]      │                │
│ Character│           │ [Choice 2]      │                │
│ STR: 16  │           │                 │                │
│ DEX: 14  │           │ 🎲 [Roll Dice]  │                │
│ CHA: 12  │           └─────────────────┘                │
└──────────┴──────────────────────────────────────────────┘
```

### 新增目錄結構

```
web/src/
  components/
    player/
      inventory-sidebar.tsx    # 左欄 — 已揭露道具列表 + 角色屬性
      scene-view.tsx           # 中央 — 羊皮紙場景區（Markdown + 選擇 + 骰子）
      gm-broadcast-toast.tsx   # GM 廣播浮動通知
      player-top-bar.tsx       # 頂部導航列
      player-guard.tsx         # 檢查用戶是否為玩家（非 GM）
      dice-roller.tsx          # 骰子 UI（輸入公式 + 結果動畫）
      item-detail-modal.tsx    # 道具詳情彈窗（名稱、描述、圖片）
      npc-detail-modal.tsx     # NPC 詳情彈窗（已揭露欄位）
      game-status-overlay.tsx  # 遊戲暫停/結束覆蓋層
  pages/
    player-game-page.tsx       # Player Game Screen 主頁面
```

### 功能清單

| 功能 | 說明 |
|------|------|
| **場景顯示** | 羊皮紙風格卡片（`parchment` bg + `gold` border），Markdown 渲染場景內容 |
| **場景選擇** | 顯示 `trigger=player_choice` 的轉場按鈕，點擊發送 `player_choice` action |
| **道具欄** | 顯示已揭露給該玩家的道具，點擊展開詳情（名稱、描述、圖片） |
| **NPC 互動** | 場景內的 NPC 可點擊展開詳情，僅顯示 `public` + 已揭露的 `hidden` 欄位 |
| **角色屬性** | 道具欄底部顯示角色名稱 + attributes（key-value 列表） |
| **骰子** | 場景區底部骰子按鈕，點擊展開輸入公式，結果即時顯示 |
| **GM 廣播** | 收到 `gm_broadcast` 事件時，以 toast/modal 形式顯示（含圖片） |
| **遊戲狀態** | 暫停時全屏半透明覆蓋「Game Paused」；結束時顯示「Game Over」+ 返回按鈕 |
| **斷線重連** | 共用 SPEC-012 的 WebSocket 重連機制 |

---

## ⚠️ 邊界條件（Edge Cases）

- 場景無 `player_choice` 轉場 → 不顯示選擇按鈕（等待 GM 推進或 auto-transition）
- 玩家尚無已揭露道具 → 道具欄顯示「No items revealed yet」
- 玩家尚未被指派角色 → 角色區顯示「No character assigned」
- GM 廣播僅針對特定玩家 → 只有目標玩家看到
- 遊戲暫停中 → 禁用所有互動（選擇、骰子）、顯示覆蓋層
- 遊戲結束 → 顯示結束覆蓋層 + 返回 dashboard 按鈕
- 道具圖片 URL 無效 → 顯示預設圖片佔位
- 場景內容為空 → 顯示場景名稱 + 「The scene unfolds...」
- 大量骰子紀錄 → 場景區只顯示最近 5 次，完整記錄在 modal 中
- WebSocket 斷線時收到 GM 廣播 → 重連後 `state_sync` 恢復完整狀態

---

## ✅ 驗收標準（Done When）

- [ ] Player Game Screen 可透過 `/sessions/{id}/play` 訪問
- [ ] WebSocket 連線成功並接收過濾後的 `state_sync`
- [ ] 左側道具欄顯示已揭露道具 + 角色屬性
- [ ] 中央場景區以羊皮紙風格渲染 Markdown
- [ ] 玩家選擇（player_choice）功能正常
- [ ] 骰子擲骰功能正常
- [ ] GM 廣播以 toast/modal 形式顯示
- [ ] 遊戲暫停/結束覆蓋層正確顯示
- [ ] 道具/NPC 詳情彈窗正常
- [ ] 斷線自動重連
- [ ] 非玩家用戶訪問被 PlayerGuard 攔截
- [ ] 與 SPEC-012 共用 game-store / use-game-socket
- [ ] 單元測試 ≥ 12 cases
- [ ] ESLint 無 error

---

## 🚫 禁止事項（Out of Scope）

- 不修改後端 WebSocket 邏輯
- 不實作聊天功能（聊天由 Discord 處理）
- 不實作 GM 專屬功能（揭露、推進、廣播 — 屬於 SPEC-012）
- 不新增後端 API endpoint
- 不修改 DB schema

---

## 📎 參考資料（References）

- UI 設計：`docs/designs/pencil-new.pen`（Player Game Screen frame `q08ud`）
- WebSocket 過濾：`internal/realtime/room.go` — `filteredSceneForPlayer()`
- 場景格式：`internal/realtime/scenario.go`（Scene / Item / NPC / Transition）
- 遊戲狀態：`internal/realtime/gamestate.go`（GameState.RevealedItems, RevealedNPCFields）
- 骰子格式：`internal/realtime/gamestate.go`（DiceResult）
- GM 廣播：`internal/realtime/message.go`（EventGMBroadcast）
