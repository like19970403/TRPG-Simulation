# SPEC-019：劇本視覺化表單編輯器

> 將 JSON textarea 替換為分段式表單 UI，讓 GM 不需手寫 JSON 即可建立/編輯劇本內容。

| 欄位 | 內容 |
|------|------|
| **規格 ID** | SPEC-019 |
| **關聯 ADR** | ADR-001（技術棧）、ADR-003（劇本格式） |
| **關聯 SPEC** | SPEC-003（Scenario CRUD）、SPEC-014（Scenario Manager UI） |
| **狀態** | Completed |
| **估算複雜度** | 高 |
| **建議模型** | Opus |
| **HITL 等級** | standard |

---

## 🎯 目標（Goal）

> 目前 GM 建立/編輯劇本時，需要在 JSON textarea 裡手寫完整的 `ScenarioContent` 結構（場景、道具、NPC、變數、規則），對非技術用戶極不友善。本 SPEC 將「內容」欄位改為分段式表單 UI，讓 GM 透過表單欄位直接填寫所有劇本內容，同時保留 JSON 模式供進階用戶使用。純前端變更，不動後端。

---

## 📥 輸入規格（Inputs）

### 資料模型（ScenarioContent — 完整結構）

後端 `internal/realtime/scenario.go` 為 source of truth，前端 `web/src/api/types.ts` 需補齊缺失的 `Action` 相關型別。

```typescript
// --- 以下已存在於 types.ts ---
interface ScenarioContent {
  id: string
  title: string            // 此為內容 ID，非劇本標題
  start_scene: string      // 首場景 ID
  scenes: Scene[]
  items: Item[]
  npcs: NPC[]
  variables: ScenarioVariable[]
  rules?: Rules
}

interface Scene {
  id: string
  name: string
  content: string           // 場景描述文字
  gm_notes?: string
  items_available?: string[] // Item ID 列表
  npcs_present?: string[]    // NPC ID 列表
  transitions?: Transition[]
  // ⚠️ 以下兩個欄位 Go 後端已支援，但 TS 缺失，需補齊：
  on_enter?: Action[]
  on_exit?: Action[]
}

interface Transition {
  target: string            // 目標場景 ID
  trigger: string           // "auto" | "player_choice" | "gm"
  condition?: string        // expr 條件表達式
  label?: string            // 顯示給玩家的選項文字
}

interface Item {
  id: string; name: string; type: string; description: string; image?: string
}

interface NPC {
  id: string; name: string; image?: string; fields?: NPCField[]
}

interface NPCField {
  key: string; label: string; value: string; visibility: string // "visible" | "hidden" | "gm_only"
}

interface ScenarioVariable {
  name: string; type: string; default: unknown // type: "bool" | "int" | "string"
}

interface Rules {
  attributes?: Attribute[]; dice_formula?: string; check_method?: string // "gte" | "gt"
}

interface Attribute {
  name: string; display: string; default: number
}

// --- 以下為新增型別（Go 後端已有，TS 缺失）---
interface Action {
  set_var?: SetVarAction
  reveal_item?: RevealItemAction
  reveal_npc_field?: RevealNPCFieldAction
}

interface SetVarAction {
  name: string
  value: unknown
  expr?: string            // expr 表達式（與 value 二擇一）
}

interface RevealItemAction {
  item_id: string
  to: string               // "current_player" | "all"
}

interface RevealNPCFieldAction {
  npc_id: string
  field_key: string
  to: string               // "current_player" | "all"
}
```

### 既有元件（可重用）

| 元件 | 路徑 | 用途 |
|------|------|------|
| `Input` | `web/src/components/ui/input.tsx` | 文字輸入框 |
| `Button` | `web/src/components/ui/button.tsx` | 按鈕 |
| `FormField` | `web/src/components/ui/form-field.tsx` | label + error 包裝 |
| `ContentEditor` | `web/src/components/scenario/content-editor.tsx` | 既有 JSON 編輯器（JSON 模式複用） |

---

## 📤 輸出規格（Expected Output）

### 新增檔案（15 個）

**UI 基礎元件（2 個）**

| 檔案 | 說明 |
|------|------|
| `web/src/components/ui/textarea.tsx` | 可重用 `<textarea>`，與 `Input` 同風格（error prop、邊框、focus 樣式） |
| `web/src/components/ui/select.tsx` | 可重用 `<select>`，與 `Input` 同風格（error prop、邊框、focus 樣式） |

**劇本表單元件（13 個）**

| 檔案 | 說明 |
|------|------|
| `web/src/components/scenario/content-form-editor.tsx` | 頂層表單編輯器：tab 導航 + 渲染對應 section |
| `web/src/components/scenario/sections/basic-info-section.tsx` | 內容 ID + 標題 + 起始場景下拉選單 |
| `web/src/components/scenario/sections/scenes-section.tsx` | 場景列表管理（新增/刪除），渲染 SceneCard |
| `web/src/components/scenario/sections/scene-card.tsx` | 可收合的單一場景卡片（含所有子欄位） |
| `web/src/components/scenario/sections/transition-editor.tsx` | 場景轉換行編輯器（目標場景、觸發、條件、標籤） |
| `web/src/components/scenario/sections/action-editor.tsx` | on_enter / on_exit 動作行編輯器 |
| `web/src/components/scenario/sections/items-section.tsx` | 道具列表管理 |
| `web/src/components/scenario/sections/item-card.tsx` | 可收合的單一道具卡片 |
| `web/src/components/scenario/sections/npcs-section.tsx` | NPC 列表管理 |
| `web/src/components/scenario/sections/npc-card.tsx` | 可收合的單一 NPC 卡片 |
| `web/src/components/scenario/sections/npc-field-row.tsx` | NPC 欄位行編輯器 |
| `web/src/components/scenario/sections/variables-section.tsx` | 變數列表管理 |
| `web/src/components/scenario/sections/rules-section.tsx` | 規則設定 |

### 修改檔案（2 個）

| 檔案 | 變更 |
|------|------|
| `web/src/api/types.ts` | 新增 `Action`、`SetVarAction`、`RevealItemAction`、`RevealNPCFieldAction` 介面；Scene 補 `on_enter`、`on_exit` |
| `web/src/pages/scenario-edit-page.tsx` | 新增 `editorMode` 切換（表單/JSON）、`formData` state、雙向同步邏輯、渲染 `ContentFormEditor` |

### 頁面設計

#### 模式切換（頁面表單區域上方）

```
                   編輯劇本                      [載入範例]  [儲存草稿]
← 取消
─────────────────────────────────────────────────────────────────────
  標題:  [The Haunted Mansion_________________________]
  描述:  [A spooky adventure..._______________________]

  ┌──────────────────────────────────────────────────────────────┐
  │  [ 表單模式 ]  [ JSON 模式 ]     ← 模式切換 toggle          │
  ├──────────────────────────────────────────────────────────────┤
  │  [ 基本資訊 | 場景 | 道具 | NPC | 變數 | 規則 ]  ← tab     │
  │                                                              │
  │  （以下為各 tab 內容）                                       │
  └──────────────────────────────────────────────────────────────┘
```

#### Tab 1：基本資訊

```
  內容 ID     [haunted-mansion____________]
  內容標題    [鬧鬼大宅____________________]
  起始場景    [ entrance           ▼ ]        ← 下拉選單（選項來自已建場景 ID）
```

#### Tab 2：場景

```
  場景列表                              [+ 新增場景]

  ┌─────────────────────────────────────────────────────┐
  │ ▼  entrance — 大宅入口                    [🗑 刪除] │  ← 收合時
  ├─────────────────────────────────────────────────────┤
  │  場景 ID    [entrance_______________]               │  ← 展開時
  │  名稱       [大宅入口_________________]              │
  │                                                     │
  │  場景描述                                           │
  │  ┌───────────────────────────────────────────────┐ │
  │  │ 你站在一座古老大宅的門前...                    │ │  ← Textarea 6行
  │  └───────────────────────────────────────────────┘ │
  │                                                     │
  │  GM 備註                                            │
  │  ┌───────────────────────────────────────────────┐ │
  │  │ 可讓玩家自由探索...                           │ │  ← Textarea 3行
  │  └───────────────────────────────────────────────┘ │
  │                                                     │
  │  可用道具                                           │
  │  ☑ old_key — 舊鑰匙                                │  ← checkbox
  │  ☐ dusty_book — 佈滿灰塵的書                      │
  │                                                     │
  │  在場 NPC                                           │
  │  ☑ butler — 管家                                    │  ← checkbox
  │                                                     │
  │  場景轉換                          [+ 新增轉換]     │
  │  ┌─────────────────────────────────────────────┐   │
  │  │ 目標 [hallway ▼]  觸發 [player_choice ▼]   │   │
  │  │ 條件 [__________]  標籤 [進入走廊_________] │   │
  │  │                                    [🗑]     │   │
  │  └─────────────────────────────────────────────┘   │
  │                                                     │
  │  進入動作 (on_enter)               [+ 新增動作]     │
  │  ┌─────────────────────────────────────────────┐   │
  │  │ 類型 [reveal_item ▼]                        │   │
  │  │ 道具 [old_key ▼]  對象 [current_player ▼]   │   │
  │  │                                    [🗑]     │   │
  │  └─────────────────────────────────────────────┘   │
  │                                                     │
  │  離開動作 (on_exit)                [+ 新增動作]     │
  │  （無動作）                                         │
  └─────────────────────────────────────────────────────┘

  ▶  library — 圖書館                          [🗑 刪除]  ← 收合狀態
  ▶  basement — 地下室                         [🗑 刪除]
```

#### Tab 3：道具

```
  道具列表                              [+ 新增道具]

  ┌─────────────────────────────────────────────────────┐
  │ ▼  old_key — 舊鑰匙                      [🗑 刪除] │
  ├─────────────────────────────────────────────────────┤
  │  道具 ID    [old_key________________]               │
  │  名稱       [舊鑰匙__________________]              │
  │  類型       [ item           ▼ ]                    │  ← item / clue / consumable
  │                                                     │
  │  描述                                               │
  │  ┌───────────────────────────────────────────────┐ │
  │  │ 一把生鏽的鐵鑰匙...                          │ │
  │  └───────────────────────────────────────────────┘ │
  │                                                     │
  │  圖片 URL   [https://...____________] (選填)        │
  └─────────────────────────────────────────────────────┘
```

#### Tab 4：NPC

```
  NPC 列表                              [+ 新增 NPC]

  ┌─────────────────────────────────────────────────────┐
  │ ▼  butler — 管家                          [🗑 刪除] │
  ├─────────────────────────────────────────────────────┤
  │  NPC ID     [butler_________________]               │
  │  名稱       [管家____________________]              │
  │  圖片 URL   [________________________] (選填)       │
  │                                                     │
  │  欄位資料                            [+ 新增欄位]   │
  │  ┌─────────────────────────────────────────────┐   │
  │  │ Key [personality]  Label [性格]              │   │
  │  │ Value [陰沉寡言]   可見性 [hidden ▼]        │   │
  │  │                                    [🗑]     │   │
  │  └─────────────────────────────────────────────┘   │
  │  ┌─────────────────────────────────────────────┐   │
  │  │ Key [secret]       Label [秘密]              │   │
  │  │ Value [他才是兇手] 可見性 [gm_only ▼]       │   │
  │  │                                    [🗑]     │   │
  │  └─────────────────────────────────────────────┘   │
  └─────────────────────────────────────────────────────┘
```

#### Tab 5：變數

```
  變數列表                              [+ 新增變數]

  ┌──────────────────────────────────────────────────────┐
  │  名稱 [has_key_______]  類型 [bool ▼]  預設 [☐]  [🗑]│  ← bool → checkbox
  ├──────────────────────────────────────────────────────┤
  │  名稱 [courage_______]  類型 [int  ▼]  預設 [0_] [🗑]│  ← int → number input
  ├──────────────────────────────────────────────────────┤
  │  名稱 [ending________]  類型 [string▼] 預設 [__] [🗑]│  ← string → text input
  └──────────────────────────────────────────────────────┘
```

#### Tab 6：規則

```
  規則設定（選填）

  骰子公式     [2d6___________________]     例如：2d6、d20+5
  檢定方式     [ gte            ▼ ]         ← gte（大於等於）/ gt（大於）

  屬性列表                              [+ 新增屬性]
  ┌──────────────────────────────────────────────────────┐
  │  Name [str_____]  Display [力量___]  Default [10] [🗑]│
  ├──────────────────────────────────────────────────────┤
  │  Name [dex_____]  Display [敏捷___]  Default [10] [🗑]│
  └──────────────────────────────────────────────────────┘
```

### 模式切換邏輯

```
表單模式 → JSON 模式：formData 序列化為 JSON 字串，寫入 content state
JSON 模式 → 表單模式：解析 content JSON 字串為 formData
                      解析失敗 → 顯示錯誤提示，阻止切換
```

### State 管理

```typescript
// scenario-edit-page.tsx 新增 state
const [editorMode, setEditorMode] = useState<'form' | 'json'>('form')
const [formData, setFormData] = useState<ScenarioContent>(defaultContent)
// 既有的 content (JSON string) 保留給 JSON 模式

// 每個 section 元件接收資料切片 + onChange callback
<ScenesSection
  scenes={formData.scenes}
  onChange={(scenes) => setFormData(prev => ({ ...prev, scenes }))}
  allItemIds={formData.items.map(i => i.id)}
  allNpcIds={formData.npcs.map(n => n.id)}
  allSceneIds={formData.scenes.map(s => s.id)}
/>

// 儲存時
if (editorMode === 'form') {
  parsedContent = formData
} else {
  parsedContent = JSON.parse(content)  // 既有邏輯
}
```

### 設計風格

| 元素 | 樣式 |
|------|------|
| 模式切換按鈕 | 與 GM console tab 同風格（active: `border-b-2 border-gold text-gold`） |
| Section tabs | 同上 |
| 卡片 | `bg-bg-card rounded-lg border border-border`，收合/展開按 ▶/▼ 圖示 |
| 新增按鈕 | `Button variant="secondary"` 小尺寸 |
| 刪除按鈕 | `text-text-tertiary hover:text-error` 小尺寸 |
| 行內編輯器（轉換/動作/NPC 欄位） | `bg-[#1A1A1A] rounded border border-border p-3` |
| Input / Textarea / Select | 與既有 `Input` 同風格（`bg-[#1A1A1A] border-border rounded-md`） |

---

## ⚠️ 邊界條件（Edge Cases）

- 場景 ID 為空 → 允許暫存但顯示紅框提示
- 起始場景下拉選單 → 選項為已建場景 ID，若尚未建場景顯示「請先新增場景」
- 轉換目標場景 → 下拉選單排除自身場景
- 可用道具 / 在場 NPC → checkbox 列表隨道具/NPC tab 的增刪即時更新
- 變數類型切換 → 預設值自動重置（bool→false, int→0, string→""）
- JSON → 表單切換解析失敗 → 顯示錯誤「JSON 格式不正確，無法切換至表單模式」，保持 JSON 模式
- 表單模式新增項目 → 自動展開新卡片，並捲動至可見區域
- 刪除正在被引用的道具/NPC → 僅刪除定義，不自動移除場景中的引用（顯示懸掛 ID 作為文字）
- 載入範例按鈕 → 在表單模式下同樣生效（直接設定 formData）
- 空劇本（無場景）→ 允許儲存（後端允許 `{}` 作為 content）
- 編輯已有劇本 → API 返回 content 後解析為 formData；若解析失敗自動切換至 JSON 模式

---

## ✅ 驗收標準（Done When）

- [ ] `npx tsc --noEmit` 通過
- [ ] `types.ts` 新增 `Action`、`SetVarAction`、`RevealItemAction`、`RevealNPCFieldAction` 介面
- [ ] `Scene` 介面補齊 `on_enter`、`on_exit` 欄位
- [ ] 新建劇本 → 表單模式填寫完整場景/道具/NPC/變數/規則 → 儲存成功
- [ ] 編輯既有劇本 → 表單模式正確載入所有欄位 → 修改後儲存
- [ ] 表單 ↔ JSON 模式切換不遺失資料
- [ ] JSON 模式 → 表單模式解析失敗時顯示錯誤並阻止切換
- [ ] 場景卡片收合/展開功能正常
- [ ] 道具/NPC 卡片收合/展開功能正常
- [ ] 轉換/動作行編輯器新增/刪除功能正常
- [ ] 可用道具/在場 NPC checkbox 即時反映 Items/NPCs tab 的變更
- [ ] 變數類型切換時預設值自動重置
- [ ] 載入範例按鈕在表單模式下正常運作
- [ ] 儲存後的劇本在 GM 控制台能正常使用（場景切換、道具揭露、NPC、轉換）
- [ ] ESLint 無 error

---

## 🚫 禁止事項（Out of Scope）

- 不修改後端 API / DB schema
- 不實作拖拽排序（可考慮未來加上/下移動按鈕）
- 不實作場景圖視覺化（節點拖拉）
- 不實作 YAML 編輯模式
- 不新增第三方 npm 套件
- 不實作即時多人協作編輯
- 不實作 Undo/Redo（瀏覽器原生 Ctrl+Z 在 input 內有效）

---

## 📎 參考資料（References）

- 後端 ScenarioContent 結構：`internal/realtime/scenario.go`
- 前端型別定義：`web/src/api/types.ts`
- 現有 JSON 編輯器：`web/src/components/scenario/content-editor.tsx`
- 現有編輯頁面：`web/src/pages/scenario-edit-page.tsx`
- 範例劇本 JSON：`docs/sample-scenario.json`
- UI 元件：`web/src/components/ui/` — `input.tsx`、`button.tsx`、`form-field.tsx`
- 設計系統色彩：`docs/designs/pencil-new.pen` 中的變數（bg-card=#161616, border=#1F1F1F, gold=#C9A962 等）
- 相關 SPEC：SPEC-003（Scenario CRUD）、SPEC-014（Scenario Manager UI）

---

## 🔧 實作順序

1. **types.ts** — 補齊 Action 相關介面，Scene 補 on_enter/on_exit
2. **UI 基礎元件** — `Textarea`、`Select`
3. **葉層元件** — `TransitionEditor`、`ActionEditor`、`NpcFieldRow`
4. **卡片元件** — `SceneCard`、`ItemCard`、`NpcCard`
5. **Section 元件** — `BasicInfoSection`、`ScenesSection`、`ItemsSection`、`NpcsSection`、`VariablesSection`、`RulesSection`
6. **頂層整合** — `ContentFormEditor`
7. **頁面整合** — 修改 `ScenarioEditPage`（模式切換、formData state、雙向同步）
8. **TypeScript 檢查** — `npx tsc --noEmit`
9. **Pencil UI 設計** — 在 .pen 檔案中畫出最終 UI mockup
