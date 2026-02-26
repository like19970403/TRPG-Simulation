# [ADR-003]: 劇本資料模型與 DSL 設計

| 欄位 | 內容 |
|------|------|
| **狀態** | `Accepted` |
| **日期** | 2026-02-27 |
| **決策者** | 專案擁有者 |

---

## 背景（Context）

TRPG-Simulation 的核心是劇本驅動遊戲流程。GM 需要一種方式定義場景圖（有向圖）、分支邏輯、道具線索、NPC 角色卡、條件觸發和骰子檢定規則。此外，GM 和玩家在遊戲中都需要筆記功能。需要決定劇本的儲存格式、資料模型、條件表達式引擎，以及驗證機制。

---

## 評估選項（Options Considered）

### 劇本格式

#### 選項 A：YAML DSL

- **優點**：宣告式、人類可讀、GM 無需程式背景、Go 解析簡單（`gopkg.in/yaml.v3`）、可用 JSON Schema 驗證
- **缺點**：極複雜邏輯受限（深層巢狀條件可讀性下降）
- **風險**：低。未來可疊加 Lua 層擴展

#### 選項 B：JSON

- **優點**：Go 原生支援、與 PostgreSQL JSONB 直接對應
- **缺點**：人類可讀性差（大量引號和括號）、GM 編寫體驗不佳、不支援註解
- **風險**：低，但使用體驗差

#### 選項 C：Lua 嵌入式腳本

- **優點**：圖靈完備、遊戲業界常用
- **缺點**：GM 需學程式語言、需安全沙箱、開發成本高、錯誤除錯困難
- **風險**：高。個人專案過度工程化

### 表達式引擎

#### 選項 A：expr-lang/expr

- **優點**：Go 原生、沙箱執行（無 I/O、無系統呼叫）、支援自訂函式注入、效能好、語法直覺
- **缺點**：非圖靈完備（設計上的安全限制）
- **風險**：低

#### 選項 B：Lua (gopher-lua)

- **優點**：圖靈完備、可處理任意複雜邏輯
- **缺點**：需建立安全沙箱、記憶體限制、超時控制；GM 學習成本高
- **風險**：中

#### 選項 C：自建 DSL 直譯器

- **優點**：完全客製化語法
- **缺點**：開發和維護成本極高、測試覆蓋困難
- **風險**：高。不值得投入

### 儲存方式

#### 選項 A：PostgreSQL JSONB（整份劇本為一個 JSONB 欄位）

- **優點**：一次讀取完整劇本、無 JOIN、JSONB 支援索引和查詢、schema 彈性高
- **缺點**：單一場景更新需讀寫整份 JSONB、大型劇本（100+ 場景）效能可能下降
- **風險**：低。TRPG 劇本通常 10-50 場景，JSONB 綽綽有餘

#### 選項 B：正規化關聯表（scenes、transitions、items 各自獨立表）

- **優點**：標準 SQL 查詢、單場景更新效率高
- **缺點**：讀取完整劇本需多次 JOIN、schema migration 頻繁、場景圖重建邏輯複雜
- **風險**：中。過度正規化增加開發複雜度

---

## 決策（Decision）

選擇 **YAML DSL + expr-lang/expr + PostgreSQL JSONB**。

### 劇本 YAML 結構

```yaml
scenario:
  id: "haunted-mansion"
  title: "鬼屋探險"
  description: "一座廢棄的維多利亞式豪宅..."
  version: 1
  author: "GM_name"

  # 劇本自訂規則
  rules:
    attributes:
      - { name: "strength", display: "力量", default: 10 }
      - { name: "perception", display: "感知", default: 10 }
      - { name: "charisma", display: "魅力", default: 10 }
      - { name: "arcana", display: "奧術", default: 5 }
    dice_formula: "2d6"  # 預設骰子公式
    check_method: "roll('2d6') + attr('{attribute}') >= {difficulty}"

  # 劇本變數（遊戲進行中可改變）
  variables:
    - { name: "found_secret_passage", type: "bool", default: false }
    - { name: "ghost_anger", type: "int", default: 0 }
    - { name: "ally_name", type: "string", default: "" }

  # 道具/線索定義
  items:
    - id: "rusty_key"
      name: "生鏽的鑰匙"
      type: "item"
      description: "一把沾滿鐵鏽的老舊鑰匙，上面刻著奇怪的符號。"
      image: "rusty_key.jpg"  # 可選，GM 上傳的圖片檔名

    - id: "torn_diary"
      name: "撕裂的日記"
      type: "clue"
      description: "日記中提到地下室有一條密道..."
      image: "torn_diary.png"

    - id: "crystal_ball"
      name: "水晶球"
      type: "prop"
      description: "一顆散發微弱紫光的水晶球。"
      # image 欄位可省略

  # NPC 角色卡定義
  npcs:
    - id: "old_butler"
      name: "乾枯的管家"
      image: "old_butler.jpg"  # 可選
      fields:
        - key: "appearance"
          label: "外觀"
          value: "身材瘦長，穿著褪色的燕尾服，眼窩深陷。"
          visibility: "public"  # public: 玩家遇到即可見 / hidden: GM 手動揭露
        - key: "personality"
          label: "性格"
          value: "表面恭敬有禮，實際上隱藏著深深的怨恨。"
          visibility: "hidden"
        - key: "secret"
          label: "秘密"
          value: "他其實是 50 年前被困在宅邸的靈魂。"
          visibility: "hidden"
        - key: "dialogue_hint"
          label: "對話提示"
          value: "會反覆提及『主人還在等您』，迴避關於自己身份的問題。"
          visibility: "hidden"

    - id: "ghost_child"
      name: "幽靈小女孩"
      fields:
        - key: "appearance"
          label: "外觀"
          value: "半透明的身影，穿著維多利亞時代的白色洋裝。"
          visibility: "public"
        - key: "background"
          label: "背景"
          value: "宅邸前主人的女兒，死於一場離奇火災。"
          visibility: "hidden"

  # 場景圖（有向圖）
  scenes:
    - id: "entrance"
      name: "大廳入口"
      content: |
        你推開沉重的橡木大門，踏入了一座陰暗的大廳。
        空氣中瀰漫著灰塵和腐朽的氣味。
        牆上掛著褪色的肖像畫，畫中人的眼睛似乎在跟隨你。
      gm_notes: "讓玩家感受到不安，但不要太早揭露鬼魂。"
      items_available: ["rusty_key"]
      npcs_present: ["old_butler"]  # 該場景中出現的 NPC

      on_enter:
        - set_var: { name: "ghost_anger", value: "var('ghost_anger') + 1" }

      transitions:
        - target: "library"
          trigger: "player_choice"
          label: "前往圖書館"

        - target: "kitchen"
          trigger: "player_choice"
          label: "前往廚房"

        - target: "secret_basement"
          trigger: "condition_met"
          condition: "has_item('rusty_key') && var('found_secret_passage')"
          label: "進入密道"

    - id: "library"
      name: "圖書館"
      content: |
        成排的書架直達天花板，大部分書籍已經腐爛。
        角落有一張覆蓋灰塵的書桌。
      gm_notes: "書桌上有日記碎片，需要感知檢定才能找到。"
      items_available: ["torn_diary"]

      transitions:
        - target: "entrance"
          trigger: "player_choice"
          label: "回到大廳"

        - target: "library_discovery"
          trigger: "condition_met"
          condition: "roll('2d6') + attr('perception') >= 10"
          label: "仔細搜索書桌"

    - id: "library_discovery"
      name: "書桌發現"
      content: "你在書桌的暗格中發現了一本撕裂的日記！"

      on_enter:
        - reveal_item: { item_id: "torn_diary", to: "current_player" }
        - set_var: { name: "found_secret_passage", value: true }

      transitions:
        - target: "library"
          trigger: "auto"

  # 起始場景
  start_scene: "entrance"
```

### 資料模型

#### YAML → Go 結構

```
Scenario
├── Meta (id, title, description, version, author)
├── Rules
│   ├── Attributes[]    (name, display, default)
│   ├── DiceFormula     (預設骰子公式)
│   └── CheckMethod     (檢定方式模板)
├── Variables[]          (name, type, default)
├── Items[]              (id, name, type, description, image?)
├── NPCs[]               (id, name, image?, fields[])
│   └── Field            (key, label, value, visibility: public/hidden)
├── Scenes[]
│   ├── id, name, content, gm_notes
│   ├── items_available[] (引用 Items.id)
│   ├── npcs_present[]    (引用 NPCs.id)
│   ├── on_enter[]        (Action: set_var / reveal_item / reveal_npc_field)
│   ├── on_exit[]         (Action)
│   └── Transitions[]
│       ├── target        (引用 Scenes.id)
│       ├── trigger       (auto / gm_decision / player_choice / condition_met)
│       ├── condition      (expr 表達式字串)
│       └── label
└── StartScene            (引用 Scenes.id)
```

#### PostgreSQL 儲存

```sql
CREATE TABLE scenarios (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    author_id   UUID NOT NULL REFERENCES users(id),
    title       VARCHAR(200) NOT NULL,
    description TEXT,
    version     INT NOT NULL DEFAULT 1,
    status      VARCHAR(20) NOT NULL DEFAULT 'draft',  -- draft / published / archived
    content     JSONB NOT NULL,  -- 完整 YAML 解析後的 JSON
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 支援全文搜索劇本標題和描述
CREATE INDEX idx_scenarios_author ON scenarios(author_id);
CREATE INDEX idx_scenarios_status ON scenarios(status);
```

GM 上傳 YAML → 後端解析為 Go struct → 驗證 → 序列化為 JSON 存入 `content` JSONB 欄位。遊戲載入時一次讀取整份劇本到記憶體。

### 表達式引擎

使用 `expr-lang/expr` 做條件求值，注入以下自訂函式：

| 函式 | 簽名 | 說明 |
|------|------|------|
| `has_item` | `has_item(item_id: string) → bool` | 當前玩家是否持有指定道具 |
| `roll` | `roll(notation: string) → int` | 擲骰並回傳結果（如 `roll('2d6')`） |
| `attr` | `attr(name: string) → int` | 讀取當前玩家的角色屬性值 |
| `var` | `var(name: string) → any` | 讀取劇本變數值 |
| `all_have_item` | `all_have_item(item_id: string) → bool` | 所有玩家都持有指定道具 |
| `player_count` | `player_count() → int` | 當前遊戲玩家人數 |

expr 引擎的安全限制：
- **無 I/O**：不能存取檔案系統或網路
- **無迴圈**：不能寫 `for` / `while`
- **執行超時**：每次求值限制 100ms
- **記憶體限制**：透過 Go runtime 控制

### NPC 角色卡機制

NPC 角色卡在劇本中預先定義，每個欄位有獨立的可見性控制：

- **`public`**：玩家進入 NPC 所在場景時自動可見（如外觀）
- **`hidden`**：GM 在遊戲中手動揭露給指定玩家

GM 揭露 NPC 欄位的方式：
1. **手動揭露**：GM 在遊戲中即時選擇 NPC → 選擇欄位 → 選擇目標玩家
2. **劇本觸發**：`on_enter` action 中使用 `reveal_npc_field`

```yaml
on_enter:
  - reveal_npc_field: { npc_id: "old_butler", field_key: "personality", to: "current_player" }
```

玩家端呈現為 NPC 角色卡，隨遊戲進行逐步揭露更多資訊。

### 筆記系統

筆記是遊戲期間的純文字備忘錄，不屬於劇本 YAML，而是遊戲 Session 的即時資料：

| 類型 | 誰寫 | 誰能看 | 儲存位置 |
|------|------|--------|----------|
| **GM 筆記**（場景層級） | 劇本預定義 | 僅 GM | 劇本 YAML `gm_notes` 欄位 |
| **GM 即時筆記** | GM 在遊戲中撰寫 | 僅 GM | `game_sessions.gm_notes` JSONB |
| **玩家筆記** | 玩家在遊戲中撰寫 | 僅該玩家自己 | `session_players.notes` TEXT |

```sql
-- GM 即時筆記存在 game_sessions 表
ALTER TABLE game_sessions ADD COLUMN gm_notes JSONB DEFAULT '{}';

-- 玩家筆記存在 session_players 表
ALTER TABLE session_players ADD COLUMN notes TEXT DEFAULT '';
```

筆記透過 REST API 儲存（非即時事件，不經 WebSocket）：
- `PUT /api/sessions/:id/notes` — GM 更新自己的即時筆記
- `PUT /api/sessions/:id/players/:player_id/notes` — 玩家更新自己的筆記

### 場景圖驗證規則

YAML 上傳時執行以下驗證：

1. **結構驗證**：必填欄位存在、型別正確
2. **場景圖完整性**：
   - `start_scene` 指向存在的場景
   - 所有 `transition.target` 指向存在的場景
   - 無孤立場景（從 `start_scene` 出發可達所有場景，或標記為 `optional`）
3. **引用完整性**：
   - `items_available` 中的 item_id 都在 `items` 中定義
   - `npcs_present` 中的 npc_id 都在 `npcs` 中定義
   - `reveal_item` action 中的 item_id 都在 `items` 中定義
   - `reveal_npc_field` action 中的 npc_id 和 field_key 都有效
   - `on_enter` / `on_exit` 中引用的變數都在 `variables` 中定義
4. **表達式驗證**：
   - 所有 `condition` 欄位的 expr 表達式可解析（語法正確）
   - 引用的函式存在（`has_item`、`roll`、`attr`、`var`）
5. **限制檢查**：
   - 場景數量 ≤ 200
   - 單場景 content 長度 ≤ 10,000 字
   - 道具數量 ≤ 500
   - NPC 數量 ≤ 100
   - 單個 NPC 欄位數量 ≤ 20
   - 變數數量 ≤ 100

### 劇本生命週期

```
draft → published → archived
  ↑        │
  └────────┘ (unpublish)
```

- **draft**：GM 可編輯，不可用於建立 GameSession
- **published**：不可編輯，可用於建立 GameSession；修改需建立新版本（version + 1）
- **archived**：不可用於新 GameSession，已有的 GameSession 不受影響

---

## 後果（Consequences）

**正面影響：**
- YAML DSL 讓 GM 無需程式背景即可編寫劇本
- expr-lang/expr 沙箱執行確保安全，不會有任意程式碼執行風險
- JSONB 儲存讓劇本結構可以靈活演進，無需頻繁 migration
- 嚴格驗證規則在上傳時攔截錯誤，避免遊戲中遇到壞資料

**負面影響 / 技術債：**
- YAML DSL 在極複雜劇本邏輯下可能不夠（超過 5 層條件巢狀可讀性差）
- 整份劇本存為單一 JSONB，協同編輯困難（MVP 不需要）
- 尚無視覺化劇本編輯器，GM 需直接編寫 YAML

**後續追蹤：**
- [ ] SPEC：YAML DSL 完整 schema 定義（JSON Schema）
- [ ] SPEC：expr 自訂函式實作細節
- [ ] 未來考慮：視覺化劇本編輯器（Phase 2+）
- [ ] 未來考慮：極複雜劇本疊加 Lua 層

---

## 關聯（Relations）

- 取代：（無）
- 被取代：（無）
- 參考：ADR-001（技術棧選型）、ADR-004（遊戲狀態管理）
