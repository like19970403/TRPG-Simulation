# [ADR-001]: 初始技術棧選型

| 欄位 | 內容 |
|------|------|
| **狀態** | `Accepted` |
| **日期** | 2026-02-27 |
| **決策者** | 專案擁有者 |

---

## 背景（Context）

TRPG-Simulation 是一個線上即時 TRPG（桌上角色扮演遊戲）平台，需求如下：

- **自創規則系統**：混合型（敘事導向 + 關鍵時刻骰子檢定），不綁定任何現有 TRPG 系統
- **GM 工具 + 玩家唯讀視圖**：GM 控制台 + 玩家網頁檢視畫面（含選擇與擲骰），語音與文字聊天由外部工具（如 Discord）處理
- **即時同步**：遊戲機制（場景、道具、骰子、選擇、GM 投放）透過 WebSocket 即時同步
- **劇本驅動**：GM 上傳劇本定義場景圖、分支路線、道具、觸發條件
- **道具/線索系統**：GM 可即時揭露道具和線索給特定玩家

需要決定整體技術棧，涵蓋：後端語言、資料庫、前端框架、即時通訊方案、劇本格式、部署架構。

---

## 評估選項（Options Considered）

### 後端語言

#### 選項 A：Go

- **優點**：原生高並發（goroutine 處理 WebSocket）、單一二進位部署簡單、記憶體效率高、強型別
- **缺點**：泛型支援較新、ORM 生態不如 Java/Python 成熟
- **風險**：低。Go 的 WebSocket 生態（gorilla/websocket）成熟穩定

#### 選項 B：Node.js (TypeScript)

- **優點**：前後端共用 TypeScript、即時應用生態豐富（Socket.io）
- **缺點**：單執行緒模型需 cluster 處理 CPU 密集任務、runtime 依賴
- **風險**：中。大量 WebSocket 連線下記憶體和 CPU 效率不如 Go

#### 選項 C：Python (FastAPI)

- **優點**：開發速度快、AI/ML 整合方便
- **缺點**：GIL 限制並發效能、WebSocket 處理非原生強項
- **風險**：高。即時通訊密集場景下效能瓶頸明顯

### 資料庫

#### 選項 A：PostgreSQL

- **優點**：JSONB 支援彈性 schema（劇本內容、角色屬性）、關聯完整性、成熟的 Go 驅動（pgx）、全文搜索
- **缺點**：比 SQLite 重，需要額外服務
- **風險**：低

#### 選項 B：MongoDB

- **優點**：文件儲存天然適合劇本資料
- **缺點**：交易支援較弱、Go driver 不如 pgx 人體工學、增加運維複雜度
- **風險**：中

#### 選項 C：SQLite

- **優點**：零運維、嵌入式
- **缺點**：多 WebSocket 連線同時寫入時效能不足
- **風險**：高。不適合即時多玩家並發場景

### 前端框架

#### 選項 A：React + TypeScript (Vite)

- **優點**：生態系最大、複雜 UI 元件資源豐富（場景編輯器、拖拉、即時看板）
- **缺點**：學習曲線較 Vue 陡
- **風險**：低

#### 選項 B：Vue + TypeScript

- **優點**：學習曲線低、同樣能勝任
- **缺點**：複雜 UI 元件（樹狀編輯器、拖拉）的第三方庫較少
- **風險**：低

### 即時通訊

#### 選項 A：REST + WebSocket 混合

- **優點**：REST 處理 CRUD（認證、劇本管理）、WebSocket 處理遊戲內即時狀態，各司其職
- **缺點**：前端需維護兩種連線
- **風險**：低。業界成熟模式

#### 選項 B：純 WebSocket

- **優點**：統一通訊層
- **缺點**：需重新實作 request-response 模式、失去 HTTP 快取/中間件/OpenAPI 文件工具
- **風險**：中。過度設計

### 劇本格式

#### 選項 A：YAML DSL + 表達式引擎（expr-lang/expr）

- **優點**：宣告式、安全（無任意程式碼執行）、GM 友善、Go 端解析簡單
- **缺點**：極複雜腳本邏輯受限
- **風險**：低。未來可在上層疊加 Lua

#### 選項 B：Lua 嵌入式腳本

- **優點**：圖靈完備、遊戲業界常用
- **缺點**：GM 需學 Lua、需安全沙箱、開發成本高
- **風險**：中。個人專案過度工程化

### 部署架構

#### 選項 A：Docker Compose

- **優點**：簡單、單台 VPS 即可運行、適合個人專案
- **缺點**：無自動擴展
- **風險**：低。Go 單實例處理數千 WebSocket 綽綽有餘

#### 選項 B：Kubernetes

- **優點**：自動擴展、自動修復
- **缺點**：運維成本極高（cluster 管理、監控、YAML 膨脹）
- **風險**：高。個人專案不需要這個複雜度

---

## 決策（Decision）

| 層級 | 選型 | 理由 |
|------|------|------|
| 後端語言 | **Go** | 高並發 WebSocket 處理、單一二進位部署、記憶體效率 |
| 架構模式 | **模組化單體** | 個人專案不需微服務開銷，Go package 天然支援模組邊界，未來可拆分 |
| 資料庫 | **PostgreSQL** | JSONB 彈性 schema + 關聯完整性，pgx 驅動成熟 |
| 快取 | **Redis（MVP 可選）** | WebSocket session 註冊、pub/sub 橫向擴展。MVP 用 in-memory |
| 前端 | **React + TypeScript (Vite)** | 複雜 UI 需求（劇本編輯器、即時看板），生態系最大 |
| 即時通訊 | **REST + WebSocket 混合** | REST 處理 CRUD，WebSocket 處理遊戲內即時狀態同步 |
| 劇本格式 | **YAML DSL + expr-lang/expr** | 宣告式、安全、GM 友善 |
| 遊戲狀態 | **Event Sourcing** | 每個遊戲動作記錄為事件，支援回放、斷線恢復、GM 審計 |
| 部署 | **Docker Compose** | 單台 VPS 即可，不需 K8S |
| 擴展策略 | **Redis pub/sub + Nginx** | 未來需多實例時加入 |

### 關鍵 Go 依賴

| 用途 | 函式庫 |
|------|--------|
| HTTP Router | `net/http` (Go 1.22+) 或 `go-chi/chi` |
| WebSocket | `github.com/gorilla/websocket` |
| DB Driver | `github.com/jackc/pgx/v5` |
| Migration | `github.com/pressly/goose/v3` |
| 表達式引擎 | `github.com/expr-lang/expr` |
| YAML 解析 | `gopkg.in/yaml.v3` |
| JWT | `github.com/golang-jwt/jwt/v5` |
| UUID | `github.com/google/uuid` |
| 日誌 | `log/slog`（stdlib） |
| 環境設定 | `github.com/caarlos0/env/v11` |

---

## 後果（Consequences）

**正面影響：**
- Go 模組化單體降低開發和部署複雜度，適合個人專案快速迭代
- PostgreSQL JSONB 讓劇本和角色屬性可以靈活儲存，無需頻繁 migration
- REST + WebSocket 混合架構各司其職，不過度設計
- YAML DSL 讓 GM 無需程式背景即可編寫劇本
- Event Sourcing 為遊戲回放和斷線恢復提供天然支援
- Docker Compose 部署簡單，單台 VPS 即可上線

**負面影響 / 技術債：**
- Go 沒有成熟的全功能 ORM，需手寫較多 SQL（可接受，pgx 的 QueryRow 足夠直觀）
- YAML DSL 在極複雜腳本邏輯下可能不夠，未來可能需要疊加 Lua 層
- React SPA 需要額外的 SEO 考量（對遊戲平台影響較小）
- MVP 不含 Redis，橫向擴展需額外工作（但單實例夠用很久）

**後續追蹤：**
- [ ] ADR-002：即時通訊策略（WebSocket Hub-Room 詳細設計）
- [ ] ADR-003：劇本資料模型與 DSL 設計
- [ ] ADR-004：遊戲狀態管理（Event Sourcing 細節）
- [ ] ADR-005：認證與權限模型

---

## 關聯（Relations）

- 取代：（無）
- 被取代：（無）
- 參考：無
