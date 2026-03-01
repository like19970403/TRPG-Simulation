# SPEC-018：Session Polish — 刪除、移除玩家、列表篩選

> 完善 Session 管理 UX，讓 GM 可以管理 Session 生命週期。

| 欄位 | 內容 |
|------|------|
| **規格 ID** | SPEC-018 |
| **關聯 ADR** | ADR-004（Session 生命週期） |
| **關聯 SPEC** | SPEC-004（Game Session 後端）、SPEC-015（Session Flow UI） |
| **估算複雜度** | 低 |
| **建議模型** | Opus |
| **HITL 等級** | standard |
| **狀態** | Completed |

---

## 🎯 目標（Goal）

> 為 Session 管理加入三項缺失功能：
> 1. **列表篩選** — tab 切換（All / Lobby / Active / Completed）
> 2. **刪除 Session** — GM 可在 Lobby 刪除 Session
> 3. **移除玩家** — GM 可在 Lobby 移除個別玩家

---

## 📐 技術決策

| 項目 | 選擇 |
|------|------|
| 篩選方式 | 前端 tab 篩選（同 scenario-list-page 模式） |
| 刪除 Session | 確認 dialog → `deleteSession()` → 導航回列表 |
| 移除玩家 | API 呼叫 `removeSessionPlayer()` → 重新整理玩家列表 |

---

## 📥 輸入規格（Inputs）

### REST API Endpoints（全部已實作於後端）

| Method | Path | 說明 | 前端 API |
|--------|------|------|----------|
| DELETE | `/api/v1/sessions/{id}` | 刪除 Session | `deleteSession()` ✅ |
| DELETE | `/api/v1/sessions/{id}/players/{userId}` | 移除玩家 | ❌ 需新增 |

---

## 📤 輸出規格（Expected Output）

### 檔案清單

```
修改（4 檔）:
  web/src/api/sessions.ts                          # +removeSessionPlayer()
  web/src/pages/session-list-page.tsx              # +status tab 篩選
  web/src/pages/session-lobby-page.tsx             # +刪除 Session 按鈕
  web/src/components/session/session-player-list.tsx # +移除玩家按鈕
```

### 頁面設計

#### Session 列表頁 — Tab 篩選

```
┌─────────────────────────────────────────────────────────┐
│ Sessions                              [Join Session]    │
│                                                         │
│ [All] [Lobby] [Active] [Completed]    ← tab 篩選      │
│                                                         │
│ ┌─────────────────────────────────────────────────┐    │
│ │ The Haunted Mansion              [lobby]    [×]  │    │
│ │ Code: ABC123              Created Mar 1, 2026    │    │
│ └─────────────────────────────────────────────────┘    │
│                                                         │
│ (filtered empty: "No sessions matching this filter")    │
└─────────────────────────────────────────────────────────┘
```

#### Session Lobby — 刪除按鈕

```
┌─────────────────────────────────────────────────────────┐
│ The Haunted Mansion      [Delete Session] [Start Game]  │
│                          ↑ GM only, lobby status only   │
└─────────────────────────────────────────────────────────┘
```

#### 玩家清單 — 移除按鈕

```
┌─────────────────────────────────────┐
│ Player a1b2c3d4     Joined 14:30 [×]│  ← GM only
│ Player e5f6g7h8     Joined 14:32 [×]│
└─────────────────────────────────────┘
```

### 功能清單

| 功能 | 說明 |
|------|------|
| **Tab 篩選** | All / Lobby / Active / Completed — 前端篩選（不改 API） |
| **刪除 Session** | GM 在 Lobby 點刪除 → 確認 dialog → `deleteSession()` → 導航回列表 |
| **移除玩家** | GM 在 Lobby 點玩家旁 [×] → `removeSessionPlayer()` → 重新整理列表 |

### 實作步驟

| Step | 說明 |
|------|------|
| 1 | `api/sessions.ts` — +`removeSessionPlayer(sessionId, userId)` |
| 2 | `session-list-page.tsx` — +status tab 篩選 |
| 3 | `session-lobby-page.tsx` — +刪除 Session 按鈕（GM + lobby） |
| 4 | `session-player-list.tsx` — +移除玩家按鈕（GM only） |
| 5 | ESLint + 全部測試通過 |

---

## ⚠️ 邊界條件（Edge Cases）

- 刪除已 active 的 Session → 後端應回傳 409（只能刪除 lobby 狀態）
- 移除最後一位玩家 → 允許（Session 仍可存在，只是沒玩家）
- GM 自己不能被移除（後端驗證）
- 篩選結果為空 → 顯示「No sessions matching this filter」
- Completed session 不顯示刪除按鈕（已結束不可刪）

---

## ✅ 驗收標準（Done When）

- [ ] Session 列表 tab 篩選正常運作
- [ ] GM 可在 Lobby 刪除 Session + 確認 dialog
- [ ] GM 可在 Lobby 移除個別玩家
- [ ] `removeSessionPlayer()` API 函數正確實作
- [ ] ESLint 無 error
- [ ] 全部測試通過

---

## 🚫 禁止事項（Out of Scope）

- 不實作 Session 封存功能
- 不修改後端 API（只新增前端 API client 函數）
- 不修改 DB schema
- 不實作批次刪除
- 不實作 Session 搜尋

---

## 📎 參考資料（References）

- 後端 Session 刪除：`internal/server/session_handlers.go`（handleDeleteSession）
- 後端移除玩家：`internal/server/session_handlers.go`（handleRemoveSessionPlayer）
- 前端 Session API：`web/src/api/sessions.ts`
- 前端 Session 列表：`web/src/pages/session-list-page.tsx`
- 前端 Lobby 頁：`web/src/pages/session-lobby-page.tsx`
- 前端玩家清單：`web/src/components/session/session-player-list.tsx`
- Scenario 列表 tab 篩選模式：`web/src/pages/scenario-list-page.tsx`
