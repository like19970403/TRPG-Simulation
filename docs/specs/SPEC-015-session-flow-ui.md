# SPEC-015：Session Flow UI

> 補齊 PoC Demo 三個阻斷缺口：Session 建立 / 加入 / 列表大廳 UI。

| 欄位 | 內容 |
|------|------|
| **規格 ID** | SPEC-015 |
| **關聯 ADR** | ADR-001（技術棧）、ADR-004（Session 生命週期） |
| **關聯 SPEC** | SPEC-004（Game Session 後端）、SPEC-011（SPA Phase 1）、SPEC-012（GM Console UI）、SPEC-013（Player Game Screen UI） |
| **估算複雜度** | 中 |
| **建議模型** | Opus |
| **HITL 等級** | standard |
| **狀態** | Completed |

---

## 🎯 目標（Goal）

> 填補前端 PoC Demo 三個阻斷缺口，讓完整的端到端流程可在前端完成：
>
> 1. **G1**: GM 無法從前端建立 Session（無 UI 呼叫 `createSession`）
> 2. **G2**: 玩家無法透過邀請碼加入 Session（無 Join UI）
> 3. **G3**: 使用者看不到 Session 列表、無法從 Lobby 進入遊戲
>
> 附帶修復 **W1**：Dashboard placeholder 改為功能導航入口。

---

## 📐 技術決策

| 項目 | 選擇 |
|------|------|
| Session 列表 | 呼叫 `listSessions()`，渲染 `SessionCard` 元件 |
| 加入 Session | Modal 輸入邀請碼 → `joinSession({ inviteCode })` |
| 建立 Session | Scenario Detail 頁「Host Game」按鈕 → `createSession({ scenarioId })` → 導航到 Lobby |
| Lobby 狀態偵測 | Polling `getSession()` 每 3 秒，偵測 lobby → active 轉換後自動導航 |
| 玩家清單 | Polling `listSessionPlayers()` 每 3 秒，顯示已加入玩家 |
| GM / Player 區分 | `useAuthStore` 的 `user.id` 與 `session.gmId` 比對 |

---

## 📥 輸入規格（Inputs）

### REST API Endpoints（全部已實作於 `api/sessions.ts`）

| Method | Path | 說明 |
|--------|------|------|
| POST | `/api/v1/sessions` | 建立 Session |
| GET | `/api/v1/sessions` | 列出自己的 Session（分頁） |
| GET | `/api/v1/sessions/{id}` | 取得 Session 詳情 |
| POST | `/api/v1/sessions/{id}/start` | 啟動 Session（lobby → active） |
| POST | `/api/v1/sessions/join` | 透過邀請碼加入 |
| GET | `/api/v1/sessions/{id}/players` | 列出 Session 玩家 |

### Request / Response Types（全部已定義於 `api/types.ts`）

```typescript
interface SessionResponse {
  id: string
  scenarioId: string
  gmId: string
  status: SessionStatus         // 'lobby' | 'active' | 'paused' | 'completed'
  inviteCode: string
  createdAt: string
  startedAt: string | null
  endedAt: string | null
}

interface CreateSessionRequest { scenarioId: string }
interface JoinSessionRequest { inviteCode: string }

interface SessionPlayerResponse {
  id: string
  userId: string
  characterId: string | null
  status: string
  joinedAt: string
}
```

### Session 狀態機

```
[lobby] ──start──▶ [active] ──pause──▶ [paused] ──resume──▶ [active]
                      │                                        │
                      └──────────── end ──────────────────────▶ [completed]
```

---

## 📤 輸出規格（Expected Output）

### 頁面路由

```
/sessions                → AuthGuard → AppLayout → SessionListPage
/sessions/{id}/lobby     → AuthGuard → AppLayout → SessionLobbyPage
```

### 新增/修改目錄結構

```
新增（6 檔）:
  web/src/components/session/
    session-status-badge.tsx     # Session 狀態 badge（lobby=藍, active=綠, paused=黃, completed=灰）
    session-card.tsx             # Session 卡片元件（劇本標題、狀態、邀請碼、日期）
    session-player-list.tsx      # Lobby 玩家清單（polling 3 秒更新）
    join-session-modal.tsx       # 邀請碼加入 Modal
  web/src/pages/
    session-list-page.tsx        # Session 列表頁
    session-lobby-page.tsx       # Session 大廳頁

修改（5 檔）:
  web/src/lib/constants.ts                          # +SESSIONS, SESSION_LOBBY routes
  web/src/components/scenario/scenario-toolbar.tsx   # published 狀態 +「Host Game」按鈕
  web/src/pages/scenario-detail-page.tsx            # +handleHostGame handler
  web/src/pages/dashboard-page.tsx                  # placeholder → 功能導航卡片
  web/src/layouts/app-layout.tsx                    # navbar +Scenarios/Sessions 連結
  web/src/router.tsx                                # +Session 路由

測試（3 檔）:
  web/src/pages/session-list-page.test.tsx           # 2 cases
  web/src/components/session/join-session-modal.test.tsx  # 2 cases
  web/src/pages/session-lobby-page.test.tsx          # 3 cases
```

### 頁面設計

#### Session 列表頁

```
┌─────────────────────────────────────────────────────────┐
│ [App Layout Top Bar: TRPG | Scenarios | Sessions]       │
├─────────────────────────────────────────────────────────┤
│ Sessions                              [Join Session]    │
│                                                         │
│ ┌─────────────────────────────────────────────────┐    │
│ │ The Haunted Mansion              [lobby]         │    │
│ │ Code: ABC123              Created Mar 1, 2026    │    │
│ └─────────────────────────────────────────────────┘    │
│ ┌─────────────────────────────────────────────────┐    │
│ │ Dragon's Lair                    [active]        │    │
│ │ Code: XYZ789              Created Feb 28, 2026   │    │
│ └─────────────────────────────────────────────────┘    │
│                                                         │
│ (empty: "No sessions yet")                              │
└─────────────────────────────────────────────────────────┘
```

#### Join Session Modal

```
┌───────────────────────────────────┐
│ Join Session                   ✕  │
│                                   │
│ Enter the invite code from your   │
│ GM to join a game session.        │
│                                   │
│ Invite Code:                      │
│ [________________________]        │
│                                   │
│ (error message if any)            │
│                                   │
│              [Cancel]  [Join]     │
└───────────────────────────────────┘
```

#### Session Lobby 頁

```
┌─────────────────────────────────────────────────────────┐
│ [← Back to Sessions]                                    │
│                                                         │
│ The Haunted Mansion                     [Start Game]    │
│ [lobby] · You are the GM        ← GM 看到 Start Game   │
│                                  ← Player 看到等待訊息  │
│                                                         │
│ Invite Code:                                            │
│ A B C 1 2 3                    [Copy]                   │
│ Share this code with players...                         │
│                                                         │
│ Players:                                                │
│ ┌─────────────────────────────────────┐                │
│ │ Player a1b2c3d4        Joined 14:30 │                │
│ │ Player e5f6g7h8        Joined 14:32 │                │
│ └─────────────────────────────────────┘                │
│                                                         │
│ (Player view: "Waiting for GM to start the game...")    │
└─────────────────────────────────────────────────────────┘
```

#### Scenario Detail 頁（修改部分）

```
工具列按鈕（published 狀態）：
  [Host Game]  [Archive]  [Delete]
               ↑ 新增 Host Game 按鈕
```

#### Dashboard（修改部分）

```
┌─────────────────────────────────────────────────────────┐
│ Welcome, username                                       │
│ Your adventure awaits.                                  │
│                                                         │
│ ┌──────────────┐  ┌──────────────┐                    │
│ │  Scenarios   │  │  Sessions    │  ← 可點擊 Link     │
│ │  Create and  │  │  Host or     │                    │
│ │  manage...   │  │  join game.. │                    │
│ └──────────────┘  └──────────────┘                    │
└─────────────────────────────────────────────────────────┘
```

### 功能清單

| 功能 | 說明 |
|------|------|
| **Session 列表** | 呼叫 `listSessions()`，每個 Session 解析劇本標題，渲染 SessionCard |
| **Session 狀態 Badge** | lobby=藍, active=綠, paused=黃, completed=灰 |
| **Join Session** | Modal 輸入邀請碼 → `joinSession()` → 導航到 Lobby |
| **Host Game** | Scenario Detail 頁 published 狀態按鈕 → `createSession()` → 導航到 Lobby |
| **Session Lobby** | 顯示邀請碼（可複製）、玩家清單（polling）、GM/Player 角色區分 |
| **GM Start Game** | Lobby 中 GM 點擊 Start Game → `startSession()` → 自動導航到 GM Console |
| **自動導航** | Lobby polling 偵測 status=active → GM 導航到 `/sessions/{id}/gm`、Player 導航到 `/sessions/{id}/play` |
| **Dashboard 導航** | 替換 placeholder 為 Scenarios / Sessions 可點擊卡片 |
| **Navbar 連結** | AppLayout 新增 Scenarios / Sessions 導航連結 |

### 實作步驟

| Step | 說明 | 檔案 |
|------|------|------|
| 1 | Constants + Routes | `constants.ts` |
| 2 | Session Status Badge | `session-status-badge.tsx` |
| 3 | Session Card | `session-card.tsx` |
| 4 | Session List Page + Tests (2 cases) | `session-list-page.tsx`, `.test.tsx` |
| 5 | Join Session Modal + Tests (2 cases) | `join-session-modal.tsx`, `.test.tsx` |
| 6 | Scenario Toolbar + Detail Page — Host Game 按鈕 | `scenario-toolbar.tsx`, `scenario-detail-page.tsx` |
| 7 | Session Player List | `session-player-list.tsx` |
| 8 | Session Lobby Page + Tests (3 cases) | `session-lobby-page.tsx`, `.test.tsx` |
| 9 | Dashboard 改版 | `dashboard-page.tsx` |
| 10 | Router Integration + AppLayout navbar | `router.tsx`, `app-layout.tsx` |
| 11 | ESLint + Tests（81 total pass） | — |

---

## ⚠️ 邊界條件（Edge Cases）

- 無效邀請碼 → 後端回傳 404，前端顯示「Invalid invite code」
- 重複加入同一 Session → 後端回傳 409，前端顯示錯誤訊息
- Session 已 active/completed → Lobby 仍可進入，按狀態顯示不同 UI
- Lobby polling 網路斷線 → catch 後靜默重試，不顯示錯誤
- GM 關閉 Lobby 後玩家仍在等待 → polling 偵測到 active 後自動導航
- 非 GM 點擊 Start Game → 按鈕只對 GM 顯示，後端也有權限驗證
- Session 列表為空 → 顯示「No sessions yet」
- `navigator.clipboard.writeText` 不可用 → 靜默失敗（非阻斷）

---

## ✅ 驗收標準（Done When）

- [x] Session 列表頁正常顯示 SessionCard
- [x] Join Session Modal 正常運作（輸入碼 → 加入 → 導航）
- [x] Scenario Detail 頁 published 狀態顯示 Host Game 按鈕
- [x] Host Game → createSession → 導航到 Lobby
- [x] Session Lobby 頁正常顯示邀請碼、玩家清單
- [x] GM 看到 Start Game 按鈕，Player 看到等待訊息
- [x] Start Game → startSession → 自動導航到 GM Console
- [x] Lobby polling 偵測 active → Player 自動導航到 Player Game
- [x] Dashboard 改為功能導航卡片
- [x] AppLayout navbar 有 Scenarios / Sessions 連結
- [x] 路由整合至 AuthGuard + AppLayout
- [x] 單元測試 7 cases（2 + 2 + 3）
- [x] ESLint 無 error/warning
- [x] 全部 81 tests pass

---

## 🚫 禁止事項（Out of Scope）

- 不實作角色選擇流程（Character Selection — 未來 SPEC）
- 不實作 Session 暫停/恢復/結束 UI（已在 GM Console 內處理）
- 不實作 Session 刪除 UI
- 不新增後端 API endpoint
- 不修改 DB schema
- 不實作 WebSocket 即時通知 Lobby 狀態變化（使用 REST polling）

---

## 📎 參考資料（References）

- 後端 Session API：`internal/server/session_handlers.go`
- Session 狀態機：`internal/game/session.go`
- 前端 API client：`web/src/api/sessions.ts`（全部函數已實作）
- 前端型別：`web/src/api/types.ts`（`SessionResponse`, `SessionPlayerResponse` 等）
- GM Console：`web/src/pages/gm-console-page.tsx`
- Player Game Screen：`web/src/pages/player-game-page.tsx`
- Pencil 設計系統：`docs/designs/pencil-new.pen`（色彩/字體變數）

---

## 🔗 PoC Demo 完整流程（修補後）

```
1. 使用者 A 註冊 → 登入 → 建立劇本 → 發佈劇本
2. 使用者 A 在劇本頁點「Host Game」→ 建立 Session → 進入 Lobby
3. 使用者 A 複製邀請碼
4. 使用者 B 註冊 → 登入 → 點「Join Session」→ 輸入邀請碼 → 進入 Lobby
5. 使用者 A（GM）在 Lobby 點「Start Game」→ 自動進入 GM Console
6. 使用者 B 自動進入 Player Game Screen
7. GM 推進場景 → 玩家看到場景更新
8. GM 擲骰 / 玩家擲骰 → 即時同步
9. 玩家做出選擇 → 場景切換
10. GM 揭露道具 → 玩家道具欄更新
11. GM 結束遊戲 → 玩家看到 Game Over overlay
```
