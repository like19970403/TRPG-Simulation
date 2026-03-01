# SPEC-017：Sample Scenario + Scenario Format Guide

> 範例劇本 JSON + 格式說明文件，讓新用戶 5 分鐘內能開始第一場遊戲。

| 欄位 | 內容 |
|------|------|
| **規格 ID** | SPEC-017 |
| **關聯 ADR** | ADR-003（劇本格式 YAML DSL） |
| **關聯 SPEC** | SPEC-003（Scenario CRUD）、SPEC-014（Scenario Manager UI） |
| **估算複雜度** | 低 |
| **建議模型** | Opus |
| **HITL 等級** | standard |
| **狀態** | Draft |

---

## 🎯 目標（Goal）

> 1. 提供一個完整可玩的範例劇本 JSON（「鬼屋探險 — The Haunted Mansion」），涵蓋平台所有遊戲機制。
> 2. 撰寫 Scenario Content JSON 格式說明文件，讓 GM 理解每個欄位的用途。
> 3. 在劇本編輯頁加入「Load Sample」按鈕，一鍵載入範例劇本。

---

## 📐 技術決策

| 項目 | 選擇 |
|------|------|
| 範例劇本格式 | JSON（與 API 一致），存放於 `docs/sample-scenario.json` |
| 格式文件 | Markdown，存放於 `docs/scenario-format-guide.md` |
| 前端載入方式 | 靜態 import JSON 檔案（Vite 支援 `import data from './data.json'`） |

---

## 📤 輸出規格（Expected Output）

### 檔案清單

```
新增（2 檔）:
  docs/sample-scenario.json              # 完整可玩範例劇本
  docs/scenario-format-guide.md          # JSON 格式說明文件

修改（1 檔）:
  web/src/pages/scenario-edit-page.tsx   # +「Load Sample」按鈕（僅 create mode）
```

### 範例劇本規格

**劇本名稱：** The Haunted Mansion（鬼屋探險）

**劇情概要：** 玩家進入一座荒廢的洋館調查失蹤事件。需要在不同房間搜索線索、與 NPC 對話、使用道具、做出關鍵選擇，最終揭開真相。

**包含的遊戲機制：**

| 機制 | 範例內容 |
|------|----------|
| **場景（4+）** | entrance, library, kitchen, secret_room, ending_escape, ending_trapped |
| **場景轉場** | `player_choice`（玩家選擇）+ `gm_decision`（GM 引導） |
| **條件轉場** | `condition: "has_key == true"`（需持有鑰匙才能進入密室） |
| **道具（3+）** | rusty_key, old_diary, mysterious_potion |
| **道具揭露** | `on_enter` 自動揭露 + GM 手動揭露 |
| **NPC（2+）** | butler（管家）、ghost（幽靈） |
| **NPC 欄位** | 公開欄位（name, role）+ 隱藏欄位（secret, weakness） |
| **變數** | visited_library (bool), has_key (bool), courage (int) |
| **on_enter actions** | setVar（標記已造訪）、revealItem（自動揭露道具） |
| **骰子** | dice_formula: "2d6+0", check_method: "gte"（≥ 目標值即成功） |
| **GM Notes** | 每個場景的 GM 提示（玩家不可見） |
| **多結局** | ending_escape（逃出）vs ending_trapped（被困） |

**場景圖：**

```
[entrance] ──player_choice──▶ [library]
    │                            │
    └──player_choice──▶ [kitchen]│
                            │    │
                            ▼    ▼
                      [secret_room] ←── condition: has_key
                            │
                    ┌───────┴───────┐
                    ▼               ▼
            [ending_escape]  [ending_trapped]
```

### 格式文件規格

**文件結構：**

1. **概述** — ScenarioContent 是什麼
2. **頂層結構** — id, title, start_scene, scenes, items, npcs, variables, rules
3. **場景（Scene）** — id, name, content (Markdown), gm_notes, transitions, on_enter, on_exit
4. **轉場（Transition）** — target, trigger (player_choice / gm_decision), condition, label
5. **道具（Item）** — id, name, type, description, image
6. **NPC** — id, name, image, fields (key, label, value, visibility)
7. **變數（Variable）** — name, type (bool/int/string), default
8. **Actions** — set_var, reveal_item, reveal_npc_field
9. **規則（Rules）** — attributes, dice_formula, check_method
10. **完整範例** — 指向 `sample-scenario.json`

### 前端修改

**`scenario-edit-page.tsx`** — 僅在 create mode（新建劇本）顯示：

```
[← Cancel]     New Scenario     [Load Sample] [Save Draft]
```

按下「Load Sample」後：
- title 填入 "The Haunted Mansion"
- description 填入劇本概要
- content 填入完整 JSON

---

## ⚠️ 邊界條件（Edge Cases）

- 使用者已有內容時按 Load Sample → 覆蓋前彈出確認 dialog
- 範例劇本 JSON 必須通過後端 `ParseScenarioContent()` 驗證
- 範例劇本發布後可直接建立 Session 進行遊戲
- 格式文件中的程式碼範例必須與後端 `ScenarioContent` struct 欄位名一致（snake_case）

---

## ✅ 驗收標準（Done When）

- [ ] `docs/sample-scenario.json` 完整且可通過 `ParseScenarioContent()` 驗證
- [ ] 範例劇本包含所有遊戲機制（場景/道具/NPC/變數/on_enter/條件轉場/多結局）
- [ ] `docs/scenario-format-guide.md` 涵蓋所有欄位說明 + 範例
- [ ] 劇本編輯頁 create mode 顯示「Load Sample」按鈕
- [ ] 載入範例後可直接儲存 → 發布 → 建立 Session → 遊玩
- [ ] ESLint 無 error

---

## 🚫 禁止事項（Out of Scope）

- 不實作多個範例劇本（一個就夠）
- 不實作 YAML 格式支援（API 使用 JSON）
- 不實作視覺化場景編輯器
- 不修改後端 API
- 不修改 DB schema
- 不實作劇本匯入/匯出功能

---

## 📎 參考資料（References）

- 後端 ScenarioContent：`internal/realtime/scenario.go`（完整 struct 定義）
- 後端解析：`internal/realtime/scenario.go:ParseScenarioContent()`
- 後端測試範例：`internal/realtime/room_test.go:testScenarioFull()`
- 前端編輯頁：`web/src/pages/scenario-edit-page.tsx`
- 前端型別：`web/src/api/types.ts`（ScenarioContent 相關）
