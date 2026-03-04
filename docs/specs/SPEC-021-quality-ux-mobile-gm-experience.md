# SPEC-021：Quality, UX, Mobile & GM Experience — 四階段綜合更新

| 欄位 | 內容 |
|------|------|
| **規格 ID** | SPEC-021 |
| **關聯 ADR** | ADR-001（技術棧）、ADR-002（即時通訊）、ADR-003（劇本資料模型）、ADR-004（遊戲狀態管理） |
| **估算複雜度** | 極高（四階段合併） |
| **建議模型** | Opus |
| **HITL 等級** | standard |
| **狀態** | Completed |

---

## 🎯 目標（Goal）

> 合併 Phase 0（Quick Wins）、Phase 1（品質與可靠性）、Phase 2（GM 體驗與遊戲回放）、Phase 3（行動裝置與 PWA）四個階段，全面提升平台品質、安全性、使用者體驗、行動裝置支援和離線可用性。

---

## 階段總覽

| Phase | 主題 | 預估時程 |
|-------|------|---------|
| Phase 0 | Quick Wins（安全修復 + 基礎 UX） | 1-2 週 |
| Phase 1 | 品質與可靠性（測試 + 驗證 + 型別安全） | 3-4 週 |
| Phase 2 | GM 體驗與遊戲回放 | 4-6 週 |
| Phase 3 | 行動裝置與 PWA | 3-5 週 |

---

# Phase 0：Quick Wins

## 📥 P0-1：WebSocket CheckOrigin 安全修復

**現狀：** `internal/server/ws_handler.go:19` 的 `CheckOrigin` 直接 `return true`，允許任意來源 WebSocket 連線，存在 CSRF 風險。

**變更：**

### 後端

**`internal/config/config.go`** — 新增環境變數：

| 參數 | 型別 | 預設值 | 說明 |
|------|------|--------|------|
| `ALLOWED_ORIGINS` | `string` | `""` | 逗號分隔的允許 origin 清單。空值 = 僅允許同 host |

```go
type Config struct {
    // ... 既有欄位 ...
    AllowedOrigins string `env:"ALLOWED_ORIGINS" envDefault:""`
}
```

**`internal/server/ws_handler.go`** — CheckOrigin 邏輯：

```go
// CheckOrigin 驗證邏輯：
// 1. 若 ALLOWED_ORIGINS 非空 → 逗號分隔解析，驗證 r.Header.Get("Origin") 是否在白名單中
// 2. 若 ALLOWED_ORIGINS 為空 → 驗證 Origin host 與 request Host 一致（same-host 策略）
// 3. 開發模式（localhost / 127.0.0.1）始終允許
```

**邊界條件：**
- Origin header 為空（某些 WebSocket client 不送）→ 拒絕（安全優先）
- 多 origin 支援：`ALLOWED_ORIGINS=https://trpg.example.com,https://staging.trpg.example.com`

---

## 📥 P0-2：全域 React ErrorBoundary

**現狀：** `web/src/components/error-boundary.tsx` 已存在（class component + functional wrapper），但未包裹 `App.tsx` 的 `<RouterProvider>`。

**變更：**

**`web/src/App.tsx`：**
```tsx
import { RouterProvider } from 'react-router'
import { ErrorBoundary } from './components/error-boundary'
import { router } from './router'

export default function App() {
  return (
    <ErrorBoundary>
      <RouterProvider router={router} />
    </ErrorBoundary>
  )
}
```

---

## 📥 P0-3：WebSocket 錯誤 Toast 通知

**現狀：** `error` 事件在 `game-store.ts:334-335` 被註釋為「no state mutation, only log」。WebSocket action 失敗（如 give_item 給未知玩家）時使用者看不到任何回饋。

**變更：**

### Toast 基礎設施

**新增 `web/src/stores/toast-store.ts`：**

```typescript
interface Toast {
  id: string
  type: 'error' | 'warning' | 'info' | 'success'
  message: string
  duration?: number  // 毫秒，預設 5000
}

interface ToastStore {
  toasts: Toast[]
  addToast: (toast: Omit<Toast, 'id'>) => void
  removeToast: (id: string) => void
}
```

- `addToast` 自動生成 `id`（`crypto.randomUUID()`）
- 每個 toast 在 `duration` 後自動移除
- 最多顯示 5 個（超出時移除最舊）

**新增 `web/src/components/toast-container.tsx`：**

- 固定定位右上角（`fixed top-4 right-4 z-50`）
- 動畫進出（Tailwind `animate-` 或 CSS transition）
- 顏色映射：error → red、warning → yellow、info → blue、success → green

### 整合

**`web/src/stores/game-store.ts` — handleEvent 新增 `error` case：**

```typescript
case 'error': {
  const { message, code } = payload as { message: string; code?: string }
  useToastStore.getState().addToast({
    type: 'error',
    message: message || '操作失敗',
  })
  break
}
```

**`web/src/App.tsx`** — 掛載 ToastContainer：

```tsx
<ErrorBoundary>
  <RouterProvider router={router} />
  <ToastContainer />
</ErrorBoundary>
```

---

## 📥 P0-4：連線狀態指示器

**現狀：** GM 和 Player 頁面（`gm-console-page.tsx:72-77`、`player-game-page.tsx:49-54`）只在 `reconnecting` 狀態顯示黃色 banner。其他三種狀態（`connecting`、`connected`、`disconnected`）無視覺指示。

`ConnectionStatus` 型別已定義（`types.ts:396`）：`'disconnected' | 'connecting' | 'connected' | 'reconnecting'`

**變更：**

**新增 `web/src/components/connection-indicator.tsx`：**

| 狀態 | 圓點顏色 | 文字 | 額外行為 |
|------|---------|------|---------|
| `connected` | 🟢 green-400 | 已連線 | 3 秒後淡出文字，僅留圓點 |
| `connecting` | 🟡 yellow-400 | 連線中… | 圓點脈動動畫 |
| `reconnecting` | 🟡 yellow-400 | 重新連線中… | 圓點脈動動畫 + 保留既有 banner |
| `disconnected` | 🔴 red-400 | 已斷線 | 持續顯示，加 toast 通知 |

- 位置：GM TopBar / Player TopBar 右側，小圓點 + 文字（`text-xs`）
- Props：`{ status: ConnectionStatus }`

**整合到既有頁面：**
- `gm-console-page.tsx` — GmTopBar 組件內加入 `<ConnectionIndicator status={connectionStatus} />`
- `player-game-page.tsx` — 頂部列加入同上

---

## 📥 P0-5：Session 列表自動刷新

**現狀：** `session-list-page.tsx:61-63` 使用 `useEffect(() => { fetchSessions() }, [fetchSessions])`，僅載入時 fetch 一次，無 polling。

**變更：**

1. 新增手動刷新按鈕（頁面頂部，列表旁）
2. 新增 30 秒自動 polling：

```typescript
useEffect(() => {
  fetchSessions()
  const interval = setInterval(fetchSessions, 30_000)
  return () => clearInterval(interval)
}, [fetchSessions])
```

3. 顯示「上次更新：X 秒前」的時間戳（`text-xs text-neutral-500`）
4. 刷新時不清空現有列表（避免閃爍），僅替換資料

---

# Phase 1：品質與可靠性

## 📥 P1-1：E2E 整合測試框架

**現狀：** 零 E2E / 整合測試。Go 有 unit test（20+ 檔案），React 有 74 個 unit test。無 Playwright / Cypress。

**變更：**

### 測試基礎設施

**技術選型：** Playwright（跨瀏覽器、原生 TypeScript 支援、比 Cypress 更適合 WebSocket 測試）

**目錄結構：**
```
e2e/
├── playwright.config.ts
├── docker-compose.e2e.yml    # 獨立 Postgres + Go server
├── setup/
│   └── global-setup.ts       # 啟動/停止 test server
├── fixtures/
│   └── auth.ts               # 登入 helper、API client
├── tests/
│   ├── auth.spec.ts           # 註冊、登入、登出
│   ├── scenario.spec.ts       # 建立/編輯/發布劇本
│   ├── session-lifecycle.spec.ts  # 建場次→加入→開始→結束
│   └── game-websocket.spec.ts     # WS 遊戲流程
└── tsconfig.json
```

**docker-compose.e2e.yml：**
- PostgreSQL 15（test database，每次 run 前 drop/recreate）
- Go server（`go run ./cmd/server`，使用 test 環境變數）
- 不使用 Redis（MVP 不需要）

### 核心測試案例

| 測試檔案 | 覆蓋範圍 |
|----------|---------|
| `auth.spec.ts` | 註冊 → 登入 → token 刷新 → 登出 |
| `scenario.spec.ts` | 建立劇本 → 編輯 JSON → 發布 → 封存 |
| `session-lifecycle.spec.ts` | 建場次 → 邀請碼加入 → 開始遊戲 → 暫停 → 恢復 → 結束 |
| `game-websocket.spec.ts` | WS 連線 → state_sync → advance_scene → dice_roll → give_item → player_choice → 斷線重連 |

### Makefile 整合

```makefile
e2e-test:        ## 執行 E2E 測試（需 Docker）
e2e-test-headed: ## E2E 測試（有瀏覽器視窗）
e2e-report:      ## 開啟 Playwright HTML 報告
```

---

## 📥 P1-2：劇本驗證強化

**現狀：** `internal/realtime/scenario.go:140-158` 的 `ParseScenarioContent` 僅檢查 `start_scene != ""` 和 `len(scenes) > 0`。ADR-003 定義的 5 類驗證大部分未實作。

**變更：**

**新增 `internal/realtime/scenario_validator.go`：**

```go
type ValidationError struct {
    Field   string // 例如 "scenes[library].transitions[0].target"
    Code    string // 例如 "orphan_scene", "invalid_ref"
    Message string
}

func ValidateScenarioContent(sc *ScenarioContent) []ValidationError
```

### 驗證規則

| 類別 | 規則 | 嚴重度 |
|------|------|--------|
| **場景圖完整性** | start_scene 必須存在於 scenes 中 | Error |
| | 所有 transition.target 必須指向存在的 scene | Error |
| | 從 start_scene 出發，所有 scene 必須可達（無孤立場景） | Warning |
| | 無自引用轉場（scene → 自身） | Warning |
| **引用完整性** | items_available 中的 item_id 必須存在於 items 定義 | Error |
| | npcs_present 中的 npc_id 必須存在於 npcs 定義 | Error |
| | on_enter/on_exit action 中的 item_id / npc_id 必須存在 | Error |
| | set_var 引用的 variable 建議在 variables 中定義（Warning，因 GM 可動態新增） | Warning |
| **表達式預驗證** | condition_met 的 condition 必須可被 `expr.Compile()` 解析 | Error |
| | set_var 的 expr 欄位必須可被 `expr.Compile()` 解析 | Error |
| **大小限制** | 最多 200 個場景 | Error |
| | 最多 500 個道具 | Error |
| | 最多 100 個 NPC | Error |
| | 單場景最多 20 個轉場 | Error |
| **結構驗證** | Scene 必須有 id 和 name | Error |
| | Item 必須有 id 和 name | Error |
| | NPC 必須有 id 和 name | Error |

### 整合點

- `POST /api/v1/scenarios`（建立劇本）→ 驗證 content
- `PUT /api/v1/scenarios/{id}`（更新劇本）→ 驗證 content
- `POST /api/v1/scenarios/{id}/publish`（發布）→ 嚴格驗證（Error 級不可過）
- 建立/更新時：Warning 級允許儲存但回傳 warnings 陣列
- 發布時：Error 級阻止發布

### REST 回應擴展

```json
{
  "id": "uuid",
  "title": "...",
  "validation_warnings": [
    {"field": "scenes[attic]", "code": "orphan_scene", "message": "場景 'attic' 從 start_scene 不可達"}
  ]
}
```

### 前端整合

劇本編輯器（Scenario Form Editor）發布前顯示驗證結果：
- Error → 紅色區塊，阻止發布
- Warning → 黃色區塊，可選擇忽略

---

## 📥 P1-3：Type-safe WebSocket Actions

**現狀：** `use-game-socket.ts:113-114` 的 `sendAction` 簽名為 `(type: string, payload: unknown)`，無編譯期型別檢查。

**變更：**

**`web/src/api/types.ts`** — 新增 Action 型別映射：

```typescript
// Client → Server action payload 映射
interface ActionPayloadMap {
  start_game: Record<string, never>
  pause_game: { reason?: string }
  resume_game: Record<string, never>
  end_game: { reason?: string }
  advance_scene: { scene_id: string }
  dice_roll: { formula: string; purpose?: string }
  reveal_item: { item_id: string; player_id?: string; player_ids?: string[] }
  give_item: { item_id: string; player_id?: string; player_ids?: string[]; quantity?: number }
  remove_item: { item_id: string; player_id?: string; player_ids?: string[]; quantity?: number }
  reveal_npc_field: { npc_id: string; field_key: string; player_id?: string; player_ids?: string[] }
  player_choice: { transition_index: number }
  gm_broadcast: { content: string; image_url?: string; player_ids?: string[] }
  set_variable: { name: string; value: unknown }
}

type ActionType = keyof ActionPayloadMap
```

**`web/src/hooks/use-game-socket.ts`** — 強型別 sendAction：

```typescript
const sendAction = useCallback(
  <T extends ActionType>(type: T, payload: ActionPayloadMap[T]) => {
    wsRef.current?.send({ type, payload })
  },
  [],
)
```

**影響範圍：** 所有呼叫 `sendAction` 的組件需確認 payload 符合型別。不需改邏輯，僅獲得編譯期檢查。

---

## 📥 P1-4：Go 整合測試

**現狀：** Go 測試全為 unit test（mock repository），無 DB / HTTP 層整合測試。

**變更：**

**技術選型：** `testcontainers-go`（自動管理 PostgreSQL container）

**新增 `internal/integration_test/`：**

| 測試檔案 | 覆蓋範圍 |
|----------|---------|
| `auth_test.go` | 註冊 → 登入 → token 驗證 → 刷新 |
| `scenario_test.go` | 建立 → 更新 → 發布 → 封存 → 刪除 |
| `session_test.go` | 建場次 → 加入 → 踢人 → 開始 → 結束 → 刪除 |
| `websocket_test.go` | WS 連線 → state_sync → advance_scene → event 持久化 → 斷線重連重放 |

**測試 helper：**
```go
// testutil/testdb.go
func SetupTestDB(t *testing.T) *pgxpool.Pool  // 啟動 container + 跑 migration
func TeardownTestDB(t *testing.T)

// testutil/testserver.go
func SetupTestServer(t *testing.T, pool *pgxpool.Pool) (*httptest.Server, *config.Config)
```

**Makefile 整合：**
```makefile
test-integration: ## 執行 Go 整合測試（需 Docker）
	go test ./internal/integration_test/... -v -count=1 -timeout=120s
```

---

# Phase 2：GM 體驗與遊戲回放

## 📥 P2-1：GM 鍵盤快捷鍵

**現狀：** GM 控制台所有操作只能透過滑鼠點擊。在 room407 等節奏緊湊的劇本中，GM 需要快速切換面板、推進場景、擲骰。

**變更：**

**新增 `web/src/hooks/use-keyboard-shortcuts.ts`：**

| 快捷鍵 | 動作 | 情境 |
|--------|------|------|
| `Ctrl+1` ~ `Ctrl+5` | 切換底部面板標籤 | GM Console |
| `Ctrl+Enter` | 推進到下一場景（需有單一 gm_decision 轉場） | GM Console — Scene Panel |
| `Ctrl+D` | 開啟骰子對話框 | GM Console |
| `Ctrl+B` | 開啟投放對話框 | GM Console |
| `Ctrl+P` | 暫停/恢復遊戲 | GM Console |
| `Escape` | 關閉當前對話框/彈窗 | 通用 |

**實作方式：**
- 使用 `useEffect` + `document.addEventListener('keydown', ...)`
- 僅在 GM Console 頁面啟用（非全域）
- 若有 input/textarea 聚焦中，跳過快捷鍵（避免衝突）
- 快捷鍵提示：各面板標題旁顯示灰色小字（如 `Ctrl+1`）

---

## 📥 P2-2：GM 投放圖片預覽

**現狀：** `gm_broadcast` payload 支援 `image_url`（`message.go:84`），但前端 BroadcastPanel 無圖片上傳/預覽功能。

**變更：**

**BroadcastPanel 擴展：**
- 新增圖片 URL 輸入框（或拖放上傳到 `POST /api/v1/uploads` → 回傳 URL）
- 輸入後即時顯示縮圖預覽（`max-h-40 object-contain`）
- 發送後玩家端 toast 內嵌圖片（`img` tag，lazy load）

---

## 📥 P2-3：批次道具操作

**現狀：** `GiveItemPayload.PlayerIDs` 已支援 `[]string`（`message.go:55-58`），後端完備。前端 Items 面板只能逐一選擇玩家。

**變更：**

**Items 面板擴展：**
- 「給予道具」對話框新增「全部玩家」checkbox
- 新增多選玩家清單（checkbox list）
- 一次送出 `give_item` action，`player_ids` 包含所選全部玩家
- `remove_item` 同理

---

## 📥 P2-4：遊戲回放 / 回顧 UI

**現狀：** Event Sourcing 後端完備（`game_events` 表 + `GameState.Apply()` + snapshot 系統）。ADR-004 明確列為後續工作。目前已完成場次只能看到最終狀態。

**變更：**

### 後端

**新增 REST endpoint：**

| 方法 | 路徑 | 說明 |
|------|------|------|
| `GET /api/v1/sessions/{id}/events` | 取得場次所有事件 | 僅 completed 狀態 + 場次成員可存取 |

```json
{
  "events": [
    {
      "id": "uuid",
      "sequence": 1,
      "type": "game_started",
      "actor_id": "uuid",
      "payload": {},
      "created_at": "2026-03-04T12:00:00Z"
    }
  ],
  "total": 150
}
```

- 分頁支援：`?limit=100&after_seq=50`
- GM 可看全部事件；Player 看到的 payload 經過與遊戲中相同的權限過濾

### 前端

**新增頁面：** `web/src/pages/game-replay-page.tsx`

**路由：** `/sessions/:id/replay`

**UI 結構：**
```
┌─────────────────────────────────────────────┐
│ 遊戲回放：[場次名稱]              [返回場次] │
├─────────────────────────────────────────────┤
│                                             │
│  ┌─ 場景面板 ─────────────────────────────┐ │
│  │ [當前場景內容，根據時間軸位置重建]     │ │
│  └────────────────────────────────────────┘ │
│                                             │
│  ┌─ 側邊資訊 ────┐                         │
│  │ 玩家列表       │                         │
│  │ 當前變數       │                         │
│  │ 道具狀態       │                         │
│  └───────────────┘                         │
│                                             │
├─────────────────────────────────────────────┤
│ ◀◀  ◀  ▶  ▶▶  │ Event 23/150 │ ━━━●━━━━━ │
│ [時間軸拖曳列]                               │
│ 12:00:00 ─── scene_changed ─── 12:05:23     │
└─────────────────────────────────────────────┘
```

**核心機制：**
1. 載入所有事件到前端
2. 建立空 GameState
3. 依序 Apply 事件到指定位置
4. 拖曳時間軸 → 重新 Apply 到目標 sequence（可從最近 snapshot 開始）
5. 播放模式：自動每 2 秒前進一個事件（可調速）

**事件清單面板：**
- 顯示所有事件的 type + actor + timestamp
- 點擊事件跳轉到該時間點
- 高亮當前事件

---

## 📥 P2-5：角色模板

**現狀：** 每次建角色都從零開始填寫。劇本 `Rules.Attributes`（`scenario.go:76-80`）定義了屬性結構但未用於角色建立流程。

**變更：**

### 後端

**新增 REST endpoint：**

| 方法 | 路徑 | 說明 |
|------|------|------|
| `GET /api/v1/scenarios/{id}/character-template` | 取得劇本定義的角色屬性模板 | 公開（已發布劇本） |

回應：
```json
{
  "attributes": [
    {"name": "strength", "label": "力量", "type": "int", "default": 10},
    {"name": "perception", "label": "感知", "type": "int", "default": 10}
  ]
}
```

### 前端

**角色建立頁面擴展：**
- 進入場次 lobby 後，建角色時自動載入該劇本的屬性模板
- 屬性欄位自動產生（input name/type/default 由模板定義）
- 玩家可修改預設值
- 「套用模板」按鈕重置為預設值

---

# Phase 3：行動裝置與 PWA

## 📥 P3-1：玩家頁面行動裝置適配

**現狀：** `player-game-page.tsx` 使用固定寬度 sidebar（`w-[300px]`）+ 場景區域。在手機上不可用。

**變更：**

### 響應式斷點策略

| 斷點 | 範圍 | 佈局 |
|------|------|------|
| `sm` | < 640px（手機） | 單欄：場景全寬，背包 → 底部抽屜 |
| `md` | 640-1024px（平板） | 雙欄：場景 + 可收合側邊 |
| `lg` | > 1024px（桌面） | 現有佈局不變 |

### 手機佈局

```
┌──────────────┐
│ TopBar + ☰   │
├──────────────┤
│              │
│  場景內容     │
│  （全寬）     │
│              │
│  [選擇按鈕]  │
│  （全寬堆疊） │
│              │
├──────────────┤
│ 🎒 背包(3)   │  ← 底部固定列，點擊展開 bottom sheet
└──────────────┘
```

### 實作要點

- 使用 Tailwind 響應式 class（`lg:w-[300px] lg:block hidden` 等）
- Bottom sheet：slide-up 動畫，半透明背景遮罩，手勢下滑關閉
- 選擇按鈕：手機上 `w-full`，桌面維持 inline
- 骰子面板：手機上浮動按鈕（FAB）觸發 bottom sheet
- 圖片：`max-w-full` 自動縮放

---

## 📥 P3-2：GM 控制台平板適配

**現狀：** `gm-console-page.tsx` 使用三欄佈局（玩家列表 | 場景 | 道具/NPC），在平板和手機上不可用。

**變更：**

### 響應式策略

| 斷點 | 佈局 |
|------|------|
| `lg` (> 1024px) | 現有三欄不變 |
| `md` (640-1024px) | 雙欄：場景 + 右側面板（頁籤切換玩家/道具/NPC） |
| `sm` (< 640px) | 單欄：場景全寬 + 底部頁籤（延伸既有 `gm-console-page.tsx:62-68` 的 tab 模式） |

### 手機佈局

```
┌──────────────────┐
│ TopBar [▶暫停] 🟢│
├──────────────────┤
│                  │
│  場景面板         │
│  （全寬）         │
│                  │
├──────────────────┤
│ 場景│玩家│道具│投放│  ← 底部固定 tab bar
└──────────────────┘
```

- 底部 tab bar 取代側邊欄
- 各 tab 全屏切換（非 overlay）
- 滑動手勢切換 tab（可選）

---

## 📥 P3-3：幫助提示與新手引導

**現狀：** GM 控制台無任何幫助文字。新 GM 需自行摸索各面板功能。

**變更：**

### Tooltip 系統

**新增 `web/src/components/help-tooltip.tsx`：**
- `?` 圖示按鈕（`w-5 h-5 rounded-full border`）
- Hover/click 顯示 popover 說明
- Props: `{ content: string }`

### 各面板提示內容

| 面板 | 提示 |
|------|------|
| 場景面板 | 「顯示當前場景內容。使用轉場按鈕或 Ctrl+Enter 推進劇情。」 |
| 玩家面板 | 「顯示線上玩家和角色。綠點=在線、灰點=離線。」 |
| 道具面板 | 「管理道具給予與移除。支援批次操作。GM Notes 僅 GM 可見。」 |
| NPC 面板 | 「管理 NPC 欄位揭露。Hidden 欄位需手動揭露給指定玩家。」 |
| 投放面板 | 「向指定或全部玩家推送文字/圖片訊息。」 |
| 變數面板 | 「查看並修改劇本變數。變更會即時影響轉場條件判定。」 |
| 骰子區 | 「輸入骰子公式（如 2d6+3）擲骰。結果所有人可見。」 |

### 首次使用引導（可選，建議但非必要）

- 首次進入 GM Console 時顯示半透明 overlay + 步驟導引
- 使用 localStorage 記住「已看過」
- 可從設定或 `?` 選單重新觸發

---

## 📥 P3-4：PWA（Progressive Web App）

**現狀：** 純 SPA，無 PWA 支援。手機使用者必須透過瀏覽器存取，無法「加到主畫面」、無離線 fallback、無推送通知基礎。

**變更：**

### Web App Manifest

**新增 `web/public/manifest.json`：**

```json
{
  "name": "TRPG Simulation",
  "short_name": "TRPG",
  "description": "線上 TRPG 遊戲輔助平台",
  "start_url": "/",
  "display": "standalone",
  "orientation": "portrait",
  "background_color": "#0a0a0a",
  "theme_color": "#C9A962",
  "icons": [
    { "src": "/icons/icon-192.png", "sizes": "192x192", "type": "image/png" },
    { "src": "/icons/icon-512.png", "sizes": "512x512", "type": "image/png" },
    { "src": "/icons/icon-maskable-512.png", "sizes": "512x512", "type": "image/png", "purpose": "maskable" }
  ]
}
```

- `display: standalone` → 隱藏瀏覽器 UI，呈現原生 app 感
- `theme_color: #C9A962` → 配合現有 champagne gold 主題
- `background_color: #0a0a0a` → 配合暗色主題啟動畫面

**`web/index.html`** — 加入 manifest link + meta tags：

```html
<link rel="manifest" href="/manifest.json" />
<meta name="theme-color" content="#C9A962" />
<meta name="apple-mobile-web-app-capable" content="yes" />
<meta name="apple-mobile-web-app-status-bar-style" content="black-translucent" />
<link rel="apple-touch-icon" href="/icons/icon-192.png" />
```

### Service Worker

**技術選型：** `vite-plugin-pwa`（基於 Workbox，與 Vite 無縫整合）

**`web/vite.config.ts`** — 加入 PWA plugin：

```typescript
import { VitePWA } from 'vite-plugin-pwa'

export default defineConfig({
  plugins: [
    // ... 既有 plugins ...
    VitePWA({
      registerType: 'autoUpdate',
      workbox: {
        // 快取策略
        runtimeCaching: [
          {
            // 靜態資源（JS/CSS/圖片）→ CacheFirst
            urlPattern: /\.(js|css|png|jpg|svg|woff2?)$/,
            handler: 'CacheFirst',
            options: { cacheName: 'static-assets', expiration: { maxEntries: 100, maxAgeSeconds: 30 * 24 * 60 * 60 } },
          },
          {
            // API 呼叫 → NetworkFirst（離線時顯示快取資料）
            urlPattern: /\/api\//,
            handler: 'NetworkFirst',
            options: { cacheName: 'api-cache', expiration: { maxEntries: 50, maxAgeSeconds: 5 * 60 } },
          },
        ],
        // 預快取 app shell
        globPatterns: ['**/*.{js,css,html,svg,png,woff2}'],
      },
    }),
  ],
})
```

### 離線 Fallback 頁面

**新增 `web/public/offline.html`：**

- 簡易靜態 HTML（不依賴 React bundle）
- 顯示：「目前離線，請檢查網路連線後重試」
- 樣式：配合暗色主題（`#0a0a0a` 背景、`#C9A962` 強調色）
- 自動偵測上線後刷新（`navigator.onLine` + `online` event）

**Service Worker 導航 fallback：**
- 非快取的導航請求（HTML）→ 離線時返回 `offline.html`

### App 圖示

**新增 `web/public/icons/`：**

| 檔案 | 尺寸 | 用途 |
|------|------|------|
| `icon-192.png` | 192×192 | Android manifest icon |
| `icon-512.png` | 512×512 | Android splash / install prompt |
| `icon-maskable-512.png` | 512×512 | Android adaptive icon（安全區域內繪製） |
| `apple-touch-icon.png` | 180×180 | iOS 主畫面圖示 |
| `favicon.svg` | scalable | 瀏覽器分頁圖示 |

- 設計風格：暗底 + 金色骰子或 TRPG 相關圖案
- maskable icon 需在 80% 安全區域內繪製主要內容

### iOS 特殊處理

- `apple-mobile-web-app-capable: yes` → iOS 全螢幕模式
- `apple-mobile-web-app-status-bar-style: black-translucent` → 狀態列半透明
- iOS 不支援 Service Worker push notification → 不在本 SPEC 範圍
- iOS Safari `100vh` 問題 → 使用 `dvh`（dynamic viewport height）或 CSS `env(safe-area-inset-*)`

### 安裝提示

**新增 `web/src/components/install-prompt.tsx`：**

- 監聽 `beforeinstallprompt` 事件（Android Chrome）
- 首次顯示底部 banner：「將 TRPG Simulation 加到主畫面，獲得更好的遊戲體驗」
- 「安裝」按鈕觸發原生安裝對話框
- 「稍後再說」按鈕關閉（localStorage 記住 7 天不再顯示）
- iOS 無原生安裝 API → 顯示手動步驟提示（「點擊分享 → 加到主畫面」）

### 邊界條件

- WebSocket 在 Service Worker 中不快取（即時通訊不走快取）
- `POST/PUT/DELETE` API 不快取（僅 GET 走 NetworkFirst）
- 離線時進入遊戲頁面 → 顯示離線 fallback（遊戲需要即時連線）
- 離線時瀏覽劇本/場次列表 → 顯示上次快取資料（NetworkFirst fallback）
- Service Worker 更新 → `autoUpdate` 模式自動替換，下次載入生效

---

# 📤 輸出規格（Expected Output）

## 新增檔案

| 檔案 | 說明 |
|------|------|
| `web/src/stores/toast-store.ts` | Toast 通知狀態管理 |
| `web/src/components/toast-container.tsx` | Toast 渲染組件 |
| `web/src/components/connection-indicator.tsx` | 連線狀態指示器 |
| `web/src/hooks/use-keyboard-shortcuts.ts` | GM 鍵盤快捷鍵 |
| `web/src/components/help-tooltip.tsx` | 幫助提示組件 |
| `web/src/pages/game-replay-page.tsx` | 遊戲回放頁面 |
| `web/src/components/install-prompt.tsx` | PWA 安裝提示組件 |
| `web/public/manifest.json` | PWA Web App Manifest |
| `web/public/offline.html` | 離線 fallback 頁面 |
| `web/public/icons/` | PWA 圖示（192/512/maskable/apple-touch） |
| `internal/realtime/scenario_validator.go` | 劇本驗證器 |
| `internal/integration_test/*.go` | Go 整合測試 |
| `e2e/` | Playwright E2E 測試目錄 |

## 修改檔案

| 檔案 | 變更 |
|------|------|
| `internal/config/config.go` | 新增 `AllowedOrigins` 欄位 |
| `internal/server/ws_handler.go` | CheckOrigin 使用 `AllowedOrigins` 驗證 |
| `internal/server/server.go` | 新增 `GET /sessions/{id}/events` + `GET /scenarios/{id}/character-template` 路由 |
| `web/src/App.tsx` | 包裹 ErrorBoundary + ToastContainer |
| `web/src/api/types.ts` | 新增 `ActionPayloadMap` 型別 |
| `web/src/hooks/use-game-socket.ts` | `sendAction` 強型別化 |
| `web/src/stores/game-store.ts` | `error` 事件 → toast |
| `web/src/pages/session-list-page.tsx` | 自動刷新 + 刷新按鈕 |
| `web/src/pages/gm-console-page.tsx` | ConnectionIndicator + 快捷鍵 + 響應式 + HelpTooltip |
| `web/src/pages/player-game-page.tsx` | ConnectionIndicator + 響應式 |
| `web/src/router.tsx` | 新增 `/sessions/:id/replay` 路由 |
| `web/src/App.tsx` | 掛載 InstallPrompt 組件 |
| `web/index.html` | manifest link + PWA meta tags |
| `web/vite.config.ts` | 加入 `vite-plugin-pwa` |
| `Makefile` | 新增 `e2e-test`、`test-integration` targets |
| `docs/openapi.yaml` | 新增 `GET /sessions/{id}/events` + `GET /scenarios/{id}/character-template` |

---

## 🔗 副作用與連動（Side Effects）

| 變更的狀態 / 資源 | 受影響的模組或功能 | 處理方式 |
|--------------------|---------------------|----------|
| `Config.AllowedOrigins` | WebSocket 連線 | CheckOrigin 驗證 origin |
| `sendAction` 型別變更 | 所有呼叫 sendAction 的組件 | 需確認 payload 符合新型別（編譯期檢查） |
| Toast store 新增 | App 根層 | 加入 ToastContainer |
| 劇本驗證新增 | POST/PUT /scenarios、POST publish | 建立/更新回傳 warnings、發布阻止 errors |
| 新增 REST endpoints | OpenAPI spec | 需同步更新 |
| 響應式佈局 | GM Console、Player Game | Tailwind class 變更，不影響邏輯 |
| Service Worker 註冊 | Vite build pipeline | `vite-plugin-pwa` 自動處理，dev 模式可選啟用 |
| manifest.json | 瀏覽器安裝提示 | `beforeinstallprompt` 事件觸發 |

---

## ⚠️ 邊界條件（Edge Cases）

### Phase 0
- CheckOrigin：Origin header 缺失 → 拒絕連線
- CheckOrigin：`ALLOWED_ORIGINS` 設定錯誤（含空格/尾斜線）→ 清理後比對
- Toast：快速連續 error → 最多 5 個，超出 FIFO 淘汰
- Session 列表 polling：頁面不在前台（`document.hidden`）→ 暫停 polling

### Phase 1
- E2E：WebSocket 測試需等待連線建立 → Playwright `waitForEvent` 或 polling store
- 劇本驗證：超大劇本（200 場景邊界）→ 效能測試確認 < 100ms
- Type-safe action：`unknown` → 具體型別可能破壞既有元件 → 需逐一確認

### Phase 2
- 遊戲回放：超長遊戲（1000+ 事件）→ 前端分批載入 + virtual scroll
- 遊戲回放：快照 → 事件重放 → 與原始 GameState 一致性驗證
- 鍵盤快捷鍵：IME 輸入法啟用中 → 跳過快捷鍵
- 角色模板：劇本無 Rules.Attributes 定義 → 不顯示模板，正常建角色

### Phase 3
- 響應式：GM Console 手機版 → 多欄內容在 tab 間切換，確保狀態不丟失
- Bottom sheet：iOS Safari 100vh 問題 → 使用 `dvh` 或 JS fallback
- PWA：WebSocket 不走 Service Worker 快取（即時通訊）
- PWA：離線時進入遊戲頁面 → 顯示離線 fallback（遊戲需即時連線）
- PWA：iOS 無原生安裝 API → 顯示手動步驟提示
- PWA：Service Worker 更新衝突 → `autoUpdate` 自動替換，重新載入生效

---

## ✅ 驗收標準（Done When）

### Phase 0
- [ ] `ALLOWED_ORIGINS` 環境變數生效，非白名單 origin 被拒絕
- [ ] 開發模式（localhost）始終允許連線
- [ ] `App.tsx` 包裹 `<ErrorBoundary>`，子組件 render 錯誤顯示 fallback UI
- [ ] WebSocket `error` 事件顯示紅色 toast 通知
- [ ] GM/Player 頁面顯示連線狀態指示器（4 種狀態各有對應視覺）
- [ ] Session 列表 30 秒自動刷新 + 手動刷新按鈕
- [ ] 頁面不在前台時暫停 polling

### Phase 1
- [ ] Playwright E2E 測試覆蓋登入、劇本 CRUD、場次生命週期、WS 遊戲流程
- [ ] `make e2e-test` 可在 CI 環境（Docker）執行
- [ ] `ValidateScenarioContent` 實作全部 5 類驗證規則
- [ ] 發布劇本時 Error 級驗證失敗阻止發布
- [ ] 劇本編輯器顯示 Warning / Error 驗證結果
- [ ] `sendAction` 獲得編譯期型別檢查，錯誤 payload 在 IDE 中報錯
- [ ] Go 整合測試覆蓋 auth、scenario、session、WebSocket 流程
- [ ] `make test-integration` 可在有 Docker 的環境執行

### Phase 2
- [ ] GM Console 支援鍵盤快捷鍵（至少 6 組），input 聚焦時不觸發
- [ ] GM 投放支援圖片 URL 輸入 + 即時預覽
- [ ] 道具給予/移除支援多選玩家 + 「全部玩家」選項
- [ ] 已完成場次可進入回放頁面，顯示事件時間軸
- [ ] 回放時間軸可拖曳，場景/變數/道具狀態正確重建
- [ ] 回放支援自動播放（每 2 秒一事件）和暫停
- [ ] 角色建立時可套用劇本屬性模板
- [ ] `GET /api/v1/sessions/{id}/events` 和 `GET /api/v1/scenarios/{id}/character-template` 回傳正確

### Phase 3
- [ ] 玩家頁面在 375px 寬度（iPhone SE）可正常操作：閱讀場景、做選擇、查看背包
- [ ] GM 控制台在 768px 寬度（iPad）可正常操作：切換面板、推進場景、管理道具
- [ ] GM 控制台在 375px 寬度可基本操作（底部 tab 模式）
- [ ] GM Console 各面板有 `?` 幫助提示
- [ ] PWA：`manifest.json` 正確配置，Android Chrome 顯示安裝提示
- [ ] PWA：Service Worker 註冊成功，靜態資源離線可用
- [ ] PWA：離線時導航到遊戲頁面顯示 `offline.html` fallback
- [ ] PWA：離線時瀏覽劇本/場次列表顯示上次快取資料
- [ ] PWA：iOS Safari「加到主畫面」後以 standalone 模式開啟
- [ ] PWA：安裝提示 banner 可關閉且 7 天內不再顯示
- [ ] PWA 圖示在 Android/iOS 主畫面正確顯示
- [ ] `go test ./... -race` 全部通過
- [ ] `npm run lint` 無新增錯誤
- [ ] `npm run test` 全部通過

---

## 🚫 禁止事項（Out of Scope）

- 不實作 OAuth 登入（Phase 4，需新 ADR-006）
- 不實作 Redis 多實例擴展（Phase 4，需新 ADR-007）
- 不實作拖拽式視覺場景編輯器（Phase 4，需新 ADR-008）
- 不遷移到 chi 路由器（Phase 4，待路由數接近門檻）
- 不實作 i18n 多語系（待用戶群擴展再考慮）
- 不實作 Lua 腳本層（expr-lang/expr 足夠）
- 不實作變數 undo/redo
- 不實作推送通知（PWA push notification，需後端支援，延後）
- 不修改 DB schema（遊戲回放使用既有 game_events 表）

---

## 📎 參考資料（References）

- ADR-001（技術棧選型 — sqlc/chi 門檻、Redis 擴展策略）
- ADR-002（即時通訊策略 — WS 事件類型完整列表）
- ADR-003（劇本資料模型 — 驗證規則定義、場景圖完整性）
- ADR-004（遊戲狀態管理 — Event Sourcing、快照、回放基礎）
- SPEC-020（背包系統、投票系統 — 本 SPEC 的前置功能）
- `docs/scenario-room407.json`（真實劇本使用情境，驗證 GM 體驗需求）
