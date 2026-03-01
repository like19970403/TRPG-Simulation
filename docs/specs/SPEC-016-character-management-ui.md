# SPEC-016：Character Management UI

> 角色建立/管理 + Session Lobby 角色指派。

| 欄位 | 內容 |
|------|------|
| **規格 ID** | SPEC-016 |
| **關聯 ADR** | ADR-001（技術棧）、ADR-005（認證與權限） |
| **關聯 SPEC** | SPEC-010（Character CRUD 後端）、SPEC-015（Session Flow UI） |
| **估算複雜度** | 中 |
| **建議模型** | Opus |
| **HITL 等級** | standard |
| **狀態** | Completed |

---

## 🎯 目標（Goal）

> 實作 Character 前端頁面，讓玩家可建立、編輯、刪除角色，並在 Session Lobby 中選擇角色加入遊戲。完成後，玩家在遊戲中的 Inventory Sidebar 會顯示角色名稱，GM 在 Lobby 和 Console 中可看到每位玩家的角色。

---

## 📐 技術決策

| 項目 | 選擇 |
|------|------|
| 角色 Attributes | JSON 文字編輯器（同 Scenario content editor 模式） |
| 角色 Inventory | JSON 陣列編輯器 |
| 角色選擇 | Session Lobby 下拉選單 → `POST /sessions/{id}/characters` |
| 角色列表 | 獨立頁面 `/characters`，navbar 連結 |

---

## 📥 輸入規格（Inputs）

### REST API Endpoints（全部已實作於後端）

| Method | Path | 說明 |
|--------|------|------|
| POST | `/api/v1/characters` | 建立角色 |
| GET | `/api/v1/characters` | 列出自己的角色（分頁） |
| GET | `/api/v1/characters/{id}` | 取得角色詳情 |
| PUT | `/api/v1/characters/{id}` | 更新角色 |
| DELETE | `/api/v1/characters/{id}` | 刪除角色 |
| POST | `/api/v1/sessions/{id}/characters` | 指派角色到 Session |

### Request Types

**CreateCharacterRequest / UpdateCharacterRequest:**
```json
{
  "name": "string (1-100 chars)",
  "attributes": { /* 任意 JSON object */ },
  "inventory": [ /* 任意 JSON array */ ],
  "notes": "string"
}
```

**AssignCharacterRequest:**
```json
{
  "characterId": "uuid"
}
```

### Response Types

**CharacterResponse:**
```json
{
  "id": "uuid",
  "userId": "uuid",
  "name": "string",
  "attributes": { /* JSON */ },
  "inventory": [ /* JSON */ ],
  "notes": "string",
  "createdAt": "RFC3339",
  "updatedAt": "RFC3339"
}
```

**CharacterListResponse:**
```json
{
  "characters": [...],
  "total": 5,
  "limit": 20,
  "offset": 0
}
```

---

## 📤 輸出規格（Expected Output）

### 頁面路由

```
/characters     → AuthGuard → AppLayout → CharacterListPage
```

### 新增/修改目錄結構

```
新增（7 檔）:
  web/src/api/characters.ts                        # Character API client
  web/src/pages/character-list-page.tsx             # 角色列表頁
  web/src/pages/character-list-page.test.tsx        # 2 cases
  web/src/components/character/
    character-card.tsx                              # 角色卡片元件
    character-form-modal.tsx                        # 建立/編輯角色 Modal
    character-form-modal.test.tsx                   # 2 cases

修改（5 檔）:
  web/src/api/types.ts                             # +CharacterResponse 等型別
  web/src/lib/constants.ts                         # +CHARACTERS route, API.CHARACTERS
  web/src/router.tsx                               # +Character 路由
  web/src/layouts/app-layout.tsx                   # navbar +Characters
  web/src/pages/dashboard-page.tsx                 # +Characters 卡片
  web/src/pages/session-lobby-page.tsx             # +角色選擇下拉（Player only）
```

### 頁面設計

#### 角色列表頁

```
┌─────────────────────────────────────────────────────────┐
│ [App Layout: TRPG | Scenarios | Sessions | Characters]  │
├─────────────────────────────────────────────────────────┤
│ Characters                          [+ New Character]   │
│                                                         │
│ ┌─────────────────────────────────────────────────┐    │
│ │ Aragorn the Ranger                               │    │
│ │ STR: 16 · DEX: 14 · Notes: "Tracking expert"    │    │
│ │                              [Edit] [Delete]     │    │
│ └─────────────────────────────────────────────────┘    │
│ ┌─────────────────────────────────────────────────┐    │
│ │ Gandalf the Grey                                 │    │
│ │ INT: 20 · WIS: 18 · Notes: "Wizard"             │    │
│ │                              [Edit] [Delete]     │    │
│ └─────────────────────────────────────────────────┘    │
│                                                         │
│ (empty: "No characters yet. Create your first one!")    │
└─────────────────────────────────────────────────────────┘
```

#### 角色建立/編輯 Modal

```
┌───────────────────────────────────┐
│ Create Character               ✕  │
│                                   │
│ Name:                             │
│ [________________________]        │
│                                   │
│ Notes:                            │
│ [________________________]        │
│                                   │
│ Attributes (JSON):                │
│ ┌─────────────────────────────┐  │
│ │ { "STR": 16, "DEX": 14 }   │  │
│ └─────────────────────────────┘  │
│ ✓ Valid JSON                      │
│                                   │
│ Inventory (JSON):                 │
│ ┌─────────────────────────────┐  │
│ │ ["sword", "shield"]         │  │
│ └─────────────────────────────┘  │
│ ✓ Valid JSON                      │
│                                   │
│              [Cancel]  [Create]   │
└───────────────────────────────────┘
```

#### Session Lobby — 角色選擇（Player only）

```
┌─────────────────────────────────────────────────────────┐
│ ...existing lobby content...                            │
│                                                         │
│ Your Character:                                         │
│ [Select a character ▾]                                  │
│   ├─ Aragorn the Ranger                                 │
│   ├─ Gandalf the Grey                                   │
│   └─ + Create New Character                             │
│                                                         │
│ [Assign Character]                                      │
│                                                         │
│ ✓ Character assigned: Aragorn the Ranger                │
└─────────────────────────────────────────────────────────┘
```

### 功能清單

| 功能 | 說明 |
|------|------|
| **角色列表** | 呼叫 `listCharacters()`，渲染 CharacterCard |
| **建立角色** | Modal 表單 → `createCharacter()` → 重新整理列表 |
| **編輯角色** | Modal 預填 → `updateCharacter()` |
| **刪除角色** | 確認 dialog → `deleteCharacter()` |
| **Session 指派** | Lobby 下拉選角色 → `assignCharacter()` |
| **Dashboard 導航** | +Characters 卡片 |
| **Navbar 連結** | +Characters |

### 實作步驟

| Step | 說明 |
|------|------|
| 1 | `api/types.ts` 加 CharacterResponse 等型別 |
| 2 | `api/characters.ts` — API client |
| 3 | `constants.ts` — +CHARACTERS route, API.CHARACTERS |
| 4 | `character-card.tsx` — 角色卡片元件 |
| 5 | `character-form-modal.tsx` + test (2 cases) — 建立/編輯 Modal |
| 6 | `character-list-page.tsx` + test (2 cases) — 角色列表頁 |
| 7 | `session-lobby-page.tsx` — +角色選擇下拉 |
| 8 | `router.tsx` + `app-layout.tsx` + `dashboard-page.tsx` — 路由整合 |
| 9 | ESLint + 全部測試通過 |

---

## ⚠️ 邊界條件（Edge Cases）

- 玩家在 Lobby 選擇角色後離開頁面再回來 → 應顯示已選角色
- 刪除已指派到 Session 的角色 → 後端 FK constraint 阻止（顯示錯誤）
- Attributes JSON 為空 `{}` → 允許（後端預設）
- Inventory JSON 為空 `[]` → 允許（後端預設）
- 角色名重複 → 允許（不同角色可同名）
- GM 也可建立角色（但通常不需要指派）
- 玩家尚未建立任何角色 → 下拉顯示 "No characters — create one first"

---

## ✅ 驗收標準（Done When）

- [ ] 角色列表頁正常顯示 + 空狀態
- [ ] 建立/編輯/刪除角色功能正常
- [ ] JSON attributes/inventory 即時驗證
- [ ] Session Lobby 角色選擇 + 指派正常
- [ ] Dashboard + Navbar 導航正常
- [ ] 路由整合至 AuthGuard + AppLayout
- [ ] 單元測試 4 cases
- [ ] ESLint 無 error

---

## 🚫 禁止事項（Out of Scope）

- 不實作角色屬性表（character sheet）的可視化呈現
- 不實作角色等級/經驗值系統
- 不實作角色模板（未來 SPEC）
- 不修改後端 API
- 不修改 DB schema

---

## 📎 參考資料（References）

- 後端 Character API：`internal/server/character_handlers.go`
- 後端型別定義：`internal/server/types.go`（CharacterResponse, AssignCharacterRequest）
- Session 指派：`internal/server/session_handlers.go`（handleAssignCharacter）
- 前端 Session API：`web/src/api/sessions.ts`
- 前端 Lobby 頁：`web/src/pages/session-lobby-page.tsx`
