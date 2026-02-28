# Frontend Design Profile — Pencil.dev 設計優先工作流

適用：具有使用者介面的系統開發專案。
載入條件：`frontend_design: enabled`

> **設計原則**：UI 設計是需求的視覺化規格。
> 先設計畫面再定義 API，介面決定資料契約，而非反過來。

---

## 設計原則

### 檔案組織

| 規則 | 說明 |
|------|------|
| 設計檔存放位置 | `docs/designs/` |
| 命名慣例 | kebab-case：`user-dashboard.pen`、`login-flow.pen` |
| 一個 .pen 檔 = 一個功能模組 | 不要把整個應用塞在一個檔案 |
| Design System 獨立管理 | `docs/designs/design-system.pen` |

### Design Token 管理

- 使用 Pencil `set_variables` 定義全域 Design Token（色彩、字型、間距）
- Token 命名使用 semantic naming：`color-primary`、`spacing-md`、`font-heading`
- 所有畫面共用同一套 Token，確保一致性
- Token 變更時須同步更新所有引用的 .pen 檔

### 元件設計慣例

| 對象 | 慣例 | 範例 |
|------|------|------|
| 元件名稱 | PascalCase | `NavBar`、`UserCard`、`LoginForm` |
| 畫面名稱 | 功能描述 | `Dashboard - Overview`、`Settings - Profile` |
| 狀態變體 | 名稱後綴 | `Button - Default`、`Button - Hover`、`Button - Disabled` |
| 響應式斷點 | 獨立 Frame | `Dashboard - Desktop`、`Dashboard - Mobile` |

---

## Design-First 工作流

> 此流程已整合至 `system_dev.md` Pre-Implementation Gate 步驟 4。
> 以下 pseudocode 描述 Gate 觸發後的具體執行邏輯。

```
FUNCTION frontend_design_gate(requirement, designs_dir = "docs/designs/"):

  // ─── 由 Pre-Implementation Gate 步驟 4 觸發 ───
  // 前置條件：SPEC 已確認、ADR 已確認（若需要）

  // ─── 第 0 步：讀取所有相關 ADR ───
  adrs = list_files("docs/adr/")
  relevant_adrs = filter(adrs, relates_to(requirement))
  IF relevant_adrs:
    FOR adr IN relevant_adrs:
      read(adr)  // 理解架構約束（前端框架、元件庫、技術限制等）
    // ADR 中的技術決策影響設計選擇

  // ─── 第 1 步：初始化 Design System（若不存在）───
  ds_path = designs_dir + "design-system.pen"
  IF NOT file_exists(ds_path):
    open_document(ds_path)
    get_guidelines("design-system")
    get_style_guide_tags() → get_style_guide(relevant_tags)
    setup_design_tokens()   // 定義 color, typography, spacing variables
    create_base_components() // 建立基礎元件：Button, Input, Card, Nav 等
    RETURN ask_human(
      title   = "Design System 已建立，請確認基礎元件與 Token",
      content = get_screenshot(ds_path),
      action  = "確認後開始畫面設計"
    )

  // ─── 第 2 步：確認是否已有對應畫面設計 ───
  pen_file = find_design(designs_dir, requirement)
  IF pen_file EXISTS AND is_current(pen_file, requirement):
    // 設計已存在且與需求一致 → 繼續
    PASS

  // ─── 第 3 步：建立或更新畫面設計 ───
  ELSE:
    IF pen_file NOT_EXISTS:
      pen_file = create_new_pen_file(designs_dir, requirement)

    open_document(pen_file)
    guideline_topic = detect_type(requirement)
    // "web-app" | "mobile-app" | "landing-page"
    get_guidelines(guideline_topic)

    // Pencil MCP 設計步驟：
    // 1. 載入 Design System 元件（batch_get design-system.pen, reusable: true）
    // 2. 建立畫面結構（batch_design: layout frames）
    // 3. 填入元件與內容（batch_design: component refs, text）
    // 4. 檢查佈局問題（snapshot_layout, problemsOnly: true）
    // 5. 視覺驗證（get_screenshot）

    RETURN ask_human(
      title   = "畫面設計已完成，請確認",
      content = get_screenshot(pen_file),
      action  = "確認後進入 SDD/API 設計"
    )

  // ─── 第 4 步：設計確認後銜接 API 設計 ───
  // （僅 openapi: enabled 時）
  IF openapi_enabled:
    api_requirements = extract_data_contracts(pen_file)
    // 從畫面元素推導 API endpoint 與資料結構
    // 例：用戶列表 → GET /users → User[]
    //     表單提交 → POST /users → CreateUserRequest
    CALL openapi_gate(api_requirements)

  // ─── 第 5 步：更新設計變更紀錄 ───
  update_design_changelog(designs_dir + "design-changelog.md", requirement)

  // ─── 不可違反的約束 ───
  INVARIANT: 畫面設計是 UI 的 single source of truth
  INVARIANT: ADR 中的技術約束必須反映在設計選擇中
  INVARIANT: 設計確認前不開始 SDD/API 設計
  INVARIANT: 設計變更必須通過人類確認
```

---

## Pencil MCP 工具使用指引

### 設計階段與工具對應

| 階段 | 使用的 MCP 工具 | 說明 |
|------|-----------------|------|
| 初始化 | `open_document`、`get_guidelines`、`get_style_guide_tags` + `get_style_guide` | 開啟/建立 .pen 檔，取得設計指引與風格靈感 |
| Token 定義 | `set_variables` | 設定色彩、字型、間距等 Design Token |
| 元件建立 | `batch_design`（Insert，設定 reusable: true） | 建立可複用的 Design System 元件 |
| 畫面設計 | `batch_design`（Insert ref）、`batch_get` | 組合元件建構畫面 |
| 佈局檢查 | `snapshot_layout`（problemsOnly: true） | 檢查裁切、重疊等佈局問題 |
| 視覺驗證 | `get_screenshot` | 截圖確認設計成果，每個畫面完成後必須驗證 |
| 批次調整 | `search_all_unique_properties`、`replace_all_matching_properties` | 全域風格調整（如統一圓角、色彩替換） |
| 空間規劃 | `find_empty_space_on_canvas` | 在畫布上找到適當位置放置新畫面 |

### get_guidelines topic 選擇

| 專案類型 | topic |
|----------|-------|
| 後台管理系統、SaaS | `web-app` + `design-system` |
| 行銷頁面、Landing Page | `landing-page` |
| 手機應用 | `mobile-app` |
| 資料密集型儀表板 | `web-app` + `table` + `design-system` |
| 簡報投影片 | `slides` |

### 降級處理

若 Pencil MCP 工具不可用（未安裝或未啟用），可手動建立設計稿放入 `docs/designs/`，AI 僅驗證設計檔是否存在並跳過 MCP 操作。

---

## 與 ADR / SPEC 連動

| 情境 | 需要 ADR | 需要設計 |
|------|---------|---------|
| 新增 UI 功能模組 | 視架構影響 | 是 |
| 修改現有畫面佈局 | 否 | 是 |
| 純後端功能（無 UI） | 視架構影響 | 否（豁免） |
| 修改 Design Token | 否 | 是（影響全域） |
| UI Bug 修復 | 否 | trivial 可豁免 |
| 新增響應式斷點 | 視架構影響 | 是 |

### SPEC 整合

當 `frontend_design: enabled` 時，SPEC 的 Done When 應包含設計相關驗收條件：

```
- [ ] 畫面設計已完成且經人類確認（docs/designs/xxx.pen）
- [ ] 實作與設計稿一致（visual regression 或人工比對）
- [ ] Design Token 與實作 CSS 變數一致
```

---

## 設計變更紀錄規則

### 檔案位置

| 檔案 | 職責 |
|------|------|
| `docs/designs/design-changelog.md` | 設計變更的完整歷史紀錄 |

### 變更分類

| 分類 | 說明 |
|------|------|
| **Added** | 新增畫面或元件 |
| **Changed** | 修改既有畫面佈局或互動流程 |
| **Deprecated** | 即將移除的畫面或元件 |
| **Removed** | 已移除的畫面或元件 |
| **Fixed** | UI Bug 修正 |

### 記錄格式範例

```markdown
# Design Changelog

所有設計相關變更紀錄。

## [2025-03-15] Dashboard 改版

### Changed
- `user-dashboard.pen` — 側邊欄改為可收合式設計
- `user-dashboard.pen` — 統計卡片從 3 欄改為 4 欄

### Added
- `user-settings.pen` — 新增使用者偏好設定頁面

## [2025-03-01] 初始設計

### Added
- `design-system.pen` — 建立 Design System（Button、Input、Card、Nav）
- `user-dashboard.pen` — 首頁儀表板
- `login-flow.pen` — 登入/註冊流程
```

---

## Design-to-Code Handoff

設計確認後，進入實作階段時：

1. **提取 Design Token** → 對應到 CSS 變數或 Tailwind config
   - `get_variables(pen_file)` → 轉換為 `--color-primary: #xxx` 等
2. **提取元件結構** → 對應到前端元件
   - `batch_get(pen_file, patterns=[{reusable: true}])` → 建立元件清單
3. **提取佈局資訊** → 對應到 CSS layout
   - `snapshot_layout(pen_file)` → 轉換為 flex/grid 結構
4. **使用 code guideline** → 取得 Pencil 的程式碼生成建議
   - `get_guidelines("code")` → 遵循 Pencil 的 code generation 規範

### 與 openapi 連動（frontend_design + openapi 同時啟用時）

設計稿中的資料需求自動推導 API 契約：

```
畫面元素       → 資料需求          → OpenAPI Schema
用戶列表       → GET /users        → User[]
表單提交       → POST /users       → CreateUserRequest
詳情頁面       → GET /users/{id}   → User
刪除按鈕       → DELETE /users/{id} → 204 No Content
```

流程：設計確認 → 提取資料需求 → openapi_gate() → SDD → TDD → 實作
