# SPEC-014：Scenario Manager UI

> 劇本管理頁面 — CRUD + 狀態流轉 + JSON 內容編輯器。

| 欄位 | 內容 |
|------|------|
| **規格 ID** | SPEC-014 |
| **關聯 ADR** | ADR-001（技術棧）、ADR-003（劇本格式） |
| **關聯 SPEC** | SPEC-003（Scenario CRUD）、SPEC-011（SPA Phase 1） |
| **估算複雜度** | 中 |
| **建議模型** | Opus |
| **HITL 等級** | standard |
| **狀態** | Draft |

---

## 🎯 目標（Goal）

> 實作 Scenario Manager 前端頁面，讓 GM 能管理劇本的完整生命週期：建立草稿、編輯內容（JSON 格式）、發布、封存、刪除。包含列表頁（分頁 + 狀態篩選）和詳情/編輯頁。此 SPEC 不含視覺化場景編輯器（未來 SPEC），內容編輯以 JSON 文字編輯器為主。

---

## 📐 技術決策

| 項目 | 選擇 |
|------|------|
| 內容編輯 | JSON 文字編輯器（`<textarea>` + JSON 驗證），未來可升級為 Monaco Editor |
| 列表分頁 | URL query params（`?limit=20&offset=0`） |
| 狀態篩選 | 前端 tab 切換（All / Draft / Published / Archived） |
| 確認對話框 | 自建 Modal 元件（Publish / Archive / Delete 前確認） |

---

## 📥 輸入規格（Inputs）

### REST API Endpoints

| Method | Path | 說明 |
|--------|------|------|
| POST | `/api/v1/scenarios` | 建立劇本（草稿） |
| GET | `/api/v1/scenarios` | 列出自己的劇本（分頁） |
| GET | `/api/v1/scenarios/{id}` | 取得劇本詳情 |
| PUT | `/api/v1/scenarios/{id}` | 更新劇本（僅草稿） |
| DELETE | `/api/v1/scenarios/{id}` | 刪除劇本（僅草稿） |
| POST | `/api/v1/scenarios/{id}/publish` | 發布（draft → published） |
| POST | `/api/v1/scenarios/{id}/archive` | 封存（published → archived） |

### Request Types

**CreateScenarioRequest / UpdateScenarioRequest:**
```json
{
  "title": "string (1-200 chars)",
  "description": "string",
  "content": { /* ScenarioContent JSON object */ }
}
```

### Response Types

**ScenarioResponse:**
```json
{
  "id": "uuid",
  "authorId": "uuid",
  "title": "string",
  "description": "string",
  "version": 1,
  "status": "draft | published | archived",
  "content": { /* JSON */ },
  "createdAt": "RFC3339",
  "updatedAt": "RFC3339"
}
```

**ScenarioListResponse:**
```json
{
  "scenarios": [...],
  "total": 25,
  "limit": 20,
  "offset": 0
}
```

### 劇本狀態機

```
[draft] ──publish──▶ [published] ──archive──▶ [archived]
  │                      │
  │ (可編輯/刪除)        │ (不可編輯/刪除，可建立 session)
  └──────────────────────┘
```

### 劇本內容格式（ScenarioContent）

```typescript
interface ScenarioContent {
  id: string
  title: string
  startScene: string          // 首場景 ID
  scenes: Scene[]             // 至少 1 個
  items?: Item[]
  npcs?: NPC[]
  variables?: Variable[]
  rules?: Rules
}

interface Scene {
  id: string
  name: string
  content: string             // Markdown
  gmNotes?: string
  itemsAvailable?: string[]   // Item IDs
  npcsPresent?: string[]      // NPC IDs
  transitions: Transition[]
  onEnter?: Action[]
  onExit?: Action[]
}
```

---

## 📤 輸出規格（Expected Output）

### 頁面路由

```
/scenarios           → AuthGuard → AppLayout → ScenarioListPage
/scenarios/new       → AuthGuard → AppLayout → ScenarioEditPage (create mode)
/scenarios/{id}      → AuthGuard → AppLayout → ScenarioDetailPage
/scenarios/{id}/edit → AuthGuard → AppLayout → ScenarioEditPage (edit mode)
```

### 新增目錄結構

```
web/src/
  api/
    scenarios.ts               # Scenario REST API calls
  components/
    scenario/
      scenario-card.tsx        # 列表中的劇本卡片（標題、狀態 badge、日期）
      scenario-status-badge.tsx # 狀態標籤（Draft=灰, Published=綠, Archived=橘）
      content-editor.tsx       # JSON 內容編輯器（textarea + 驗證）
      confirm-modal.tsx        # 確認對話框（Publish / Archive / Delete）
      scenario-toolbar.tsx     # 操作按鈕列（Edit / Publish / Archive / Delete）
  pages/
    scenario-list-page.tsx     # 劇本列表頁
    scenario-detail-page.tsx   # 劇本詳情頁（唯讀 + 操作按鈕）
    scenario-edit-page.tsx     # 劇本編輯頁（新建 / 編輯）
```

### 頁面設計

#### 劇本列表頁

```
┌─────────────────────────────────────────────────────────┐
│ [App Layout Top Bar]                                    │
├─────────────────────────────────────────────────────────┤
│ Scenarios                         [+ New Scenario]      │
│                                                         │
│ [All] [Draft] [Published] [Archived]     ← tab 篩選    │
│                                                         │
│ ┌─────────────────────────────────────────────────┐    │
│ │ 📖 The Haunted Mansion         [Draft]          │    │
│ │ A spooky adventure...          Updated 2h ago   │    │
│ └─────────────────────────────────────────────────┘    │
│ ┌─────────────────────────────────────────────────┐    │
│ │ 📖 Dragon's Lair               [Published]      │    │
│ │ Fight the ancient dragon...    Updated 1d ago   │    │
│ └─────────────────────────────────────────────────┘    │
│                                                         │
│ Showing 1-20 of 25          [< Prev] [Next >]          │
└─────────────────────────────────────────────────────────┘
```

#### 劇本詳情頁

```
┌─────────────────────────────────────────────────────────┐
│ [← Back to Scenarios]                                   │
│                                                         │
│ The Haunted Mansion                  [Edit] [Publish]   │
│ Status: Draft  ·  Version: 3  ·  Updated: 2026-02-28   │
│                                                         │
│ Description:                                            │
│ A spooky adventure in a haunted mansion...              │
│                                                         │
│ Content Preview:                                        │
│ ┌───────────────────────────────────────────┐          │
│ │ { "startScene": "entrance",               │          │
│ │   "scenes": [...],                        │          │
│ │   "items": [...] }                        │          │
│ └───────────────────────────────────────────┘          │
│                                                         │
│ Scenes: 5  ·  Items: 8  ·  NPCs: 3                    │
└─────────────────────────────────────────────────────────┘
```

#### 劇本編輯頁

```
┌─────────────────────────────────────────────────────────┐
│ [← Cancel]          Edit Scenario        [Save Draft]   │
│                                                         │
│ Title:     [The Haunted Mansion________________]        │
│ Description: [A spooky adventure...____________]        │
│                                                         │
│ Content (JSON):                                         │
│ ┌───────────────────────────────────────────┐          │
│ │ {                                         │          │
│ │   "startScene": "entrance",               │ ← 等寬  │
│ │   "scenes": [                             │   字體   │
│ │     { "id": "entrance", ... }             │          │
│ │   ],                                      │          │
│ │   "items": [...]                          │          │
│ │ }                                         │          │
│ └───────────────────────────────────────────┘          │
│ ✓ Valid JSON                    ← 即時驗證              │
│                          ✗ Invalid JSON: Unexpected...  │
└─────────────────────────────────────────────────────────┘
```

### 功能清單

| 功能 | 說明 |
|------|------|
| **列表頁** | 分頁顯示（20 筆/頁），tab 篩選狀態，點擊進入詳情 |
| **新建劇本** | 填寫 title + description + content JSON，儲存為 draft |
| **編輯劇本** | 僅 draft 可編輯，即時 JSON 驗證 |
| **詳情頁** | 唯讀顯示，content 以格式化 JSON 呈現，顯示場景/道具/NPC 統計 |
| **發布** | draft → published，確認 dialog |
| **封存** | published → archived，確認 dialog |
| **刪除** | 僅 draft 可刪除，確認 dialog |
| **JSON 驗證** | 編輯時即時驗證 JSON 格式，顯示錯誤位置 |
| **統計摘要** | 詳情頁顯示場景數、道具數、NPC 數（從 content 解析） |

---

## ⚠️ 邊界條件（Edge Cases）

- 編輯非 draft 劇本 → 後端回傳 409，前端顯示「Only draft scenarios can be edited」
- 刪除有 session 引用的劇本 → 後端回傳 409，前端顯示錯誤
- Content JSON 為空物件 `{}` → 允許儲存（後端驗證為 valid JSON object）
- Content JSON 語法錯誤 → 前端阻止提交，顯示錯誤訊息
- 極大的 content JSON → textarea 不設上限，但提醒使用者檔案大小
- 列表為空 → 顯示「No scenarios yet. Create your first one!」
- 分頁越界（offset > total）→ 顯示空列表
- Title 為 201 字元 → 前端驗證阻止（1-200 chars）
- 併發編輯（兩個 tab）→ 後者覆蓋前者（last-write-wins，版本號自增）

---

## ✅ 驗收標準（Done When）

- [ ] 列表頁正常顯示分頁劇本 + tab 篩選
- [ ] 新建劇本功能正常（title + description + content）
- [ ] 編輯劇本功能正常（僅 draft）
- [ ] 詳情頁正常顯示劇本資訊 + content 預覽
- [ ] 發布功能正常（draft → published + 確認 dialog）
- [ ] 封存功能正常（published → archived + 確認 dialog）
- [ ] 刪除功能正常（僅 draft + 確認 dialog）
- [ ] JSON 即時驗證正常
- [ ] 狀態 badge 顯示正確（Draft=灰, Published=綠, Archived=橘）
- [ ] 非作者訪問被攔截（403 處理）
- [ ] 路由整合至 AppLayout + AuthGuard
- [ ] 單元測試 ≥ 12 cases
- [ ] ESLint 無 error

---

## 🚫 禁止事項（Out of Scope）

- 不實作視覺化場景編輯器（拖拉節點、場景圖 — 未來 SPEC）
- 不實作 YAML 編輯模式（API 使用 JSON）
- 不實作劇本版本歷史 / diff 比較
- 不新增後端 API endpoint
- 不修改 DB schema
- 不實作劇本匯入/匯出

---

## 📎 參考資料（References）

- 後端 Scenario API：`internal/server/scenario_handlers.go`
- 後端驗證規則：`internal/server/types.go`（CreateScenarioRequest, validateCreateScenario）
- 劇本內容格式：`internal/realtime/scenario.go`（ScenarioContent 完整結構）
- 狀態流轉：`internal/scenario/repository.go`（UpdateStatus — draft→published→archived）
- Pencil 設計系統：`docs/designs/pencil-new.pen`（色彩/字體變數）
