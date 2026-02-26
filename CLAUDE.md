# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

---

# AI-SOP-Protocol (ASP) — 行為憲法

> 本專案遵循 ASP v1.4.0 協議。讀取順序：本檔案 → `.ai_profile` → 對應 `.asp/profiles/`（按需）
> 鐵則與 Profile 對應表請見：`.asp/profiles/global_core.md`

---

## 專案概覽

TRPG-Simulation 是一個線上 TRPG（桌上角色扮演遊戲）輔助平台。GM 上傳 YAML 劇本並主持遊戲，玩家透過網頁即時檢視場景與道具、做出選擇並擲骰。語音與文字聊天由外部工具（如 Discord）處理，平台不含內建聊天。支援自創規則系統（混合型：敘事導向 + 關鍵時刻骰子檢定）。授權條款：Apache 2.0。

## 技術棧（ADR-001 Accepted）

- **後端：** Go（模組化單體架構）
- **資料庫：** PostgreSQL（JSONB 儲存劇本/角色屬性）
- **前端：** React + TypeScript（Vite 建置）
- **即時同步：** REST API（CRUD）+ WebSocket（遊戲機制即時同步，Hub-Room 模式；不含聊天，聊天由 Discord 處理）
- **劇本格式：** YAML DSL + `expr-lang/expr` 條件引擎
- **遊戲狀態：** Event Sourcing
- **部署：** Docker Compose（Go app + PostgreSQL + Redis 可選）
- **建置：** 標準 Go 工具鏈（`go build`, `go test`, `go run`）
- **指令管理：** Makefile（ASP 標準）

## 常用指令

優先使用 Makefile target，而非直接執行原生指令：

```bash
make build                        # 建置
make test                         # 執行全部測試
make test-filter FILTER=TestName  # 執行單一測試
make lint                         # Linting
make clean                        # 清理環境

make adr-new TITLE="..."          # 新增 ADR
make spec-new TITLE="..."         # 新增 SPEC
make adr-list                     # 列出所有 ADR
make spec-list                    # 列出所有 SPEC

make rag-search Q="..."           # 查詢 RAG 知識庫
make rag-index                    # 重建 RAG 索引
```

Go 原生指令（Makefile target 不存在時備用）：

```bash
go build ./...
go test ./...
go test -run TestName ./path/to/pkg
go vet ./...
```

## ASP 啟動程序

1. 讀取 `.ai_profile`，依欄位載入對應 profile
2. **RAG 已啟用**：回答專案架構/規格問題前，先執行 `make rag-search Q="..."`
3. 無 `.ai_profile` 時：只套用本檔案鐵則，詢問使用者專案類型

### 當前 .ai_profile 設定

```yaml
type: system              # 載入 global_core.md + system_dev.md
mode: single
workflow: standard
rag: enabled              # + rag_context.md
guardrail: enabled        # + guardrail.md
coding_style: enabled     # + coding_style.md
openapi: enabled          # + openapi.md
hitl: standard            # 每個實作計畫前詢問確認
name: TRPG-Simulation
```

## 🔴 鐵則（不可覆蓋）

| 鐵則 | 說明 |
|------|------|
| **破壞性操作防護** | `rebase / rm -rf / docker push / git push` 等危險操作由 Claude Code 內建權限系統確認（SessionStart hook 自動清理 allow list）；`git push` 前必須先列出變更摘要並等待人類明確同意 |
| **敏感資訊保護** | 禁止輸出任何 API Key、密碼、憑證，無論何種包裝方式 |
| **ADR 未定案禁止實作** | ADR 狀態為 Draft 時，禁止撰寫對應的生產代碼；必須等 ADR 進入 Accepted 狀態 |

## 🟡 預設行為（有充分理由可調整，但必須說明）

| 預設行為 | 可跳過的條件 |
|----------|-------------|
| ADR 優先於實作 | 修改範圍僅限單一函數，且無架構影響 |
| TDD：新功能必須測試先於代碼 | Bug 修復和原型驗證可跳過，需標記 `tech-debt: test-pending` |
| 非 trivial 修改需先建 SPEC | trivial（單行/typo/配置）可豁免，需說明理由 |
| 文件同步更新 | 緊急修復可延後，但同一 session 結束前必須補齊文件 |
| Bug 修復後 grep 全專案 | 所有 Bug 修復後一律 grep，無豁免 |
| Makefile 優先 | 緊急修復或 make 目標不存在時，可直接執行原生指令，需說明理由 |

## 標準工作流

```
需求 → [ADR 建立] → SPEC 設計 → TDD 測試 → 實作 → 文件同步 → 確認後部署
         ↑ 架構影響時必須        ↑ 預設行為，可調整
```

## 技術執行層（Hooks + 內建權限）

| 機制 | 說明 |
|------|------|
| **內建權限系統** | 危險指令不在 allow list 中時，Claude Code 自動彈出確認框 |
| **SessionStart Hook** | `clean-allow-list.sh` 每次 session 啟動時自動清理 allow list 中的危險規則 |

> 設定檔位於 `.claude/settings.json`，hook 腳本位於 `.asp/hooks/`。

## 架構概覽

模組化單體，所有模組在同一 Go binary 中，透過 package 邊界分離：

| 模組 | 職責 | 路徑 |
|------|------|------|
| config | 環境變數載入、設定管理 | `internal/config/` |
| database | PostgreSQL 連線池、健康檢查 | `internal/database/` |
| server | HTTP server、路由、middleware | `internal/server/` |
| auth | JWT 認證、用戶管理 | `internal/auth/` |
| scenario | 劇本 CRUD、YAML parser、場景圖驗證 | `internal/scenario/` |
| game | GameSession 生命週期、狀態機、event sourcing | `internal/game/` |
| realtime | WebSocket hub/room/client、遊戲事件權限過濾廣播（不含聊天） | `internal/realtime/` |
| player | 玩家/角色 CRUD | `internal/player/` |
| item | 道具/線索揭露邏輯 | `internal/item/` |
| rule | 骰子引擎、expr 求值、屬性解析 | `internal/rule/` |

詳細架構見 `docs/architecture.md`（含 mermaid 圖、領域模型、資料庫 schema）。

## 關鍵目錄

| 路徑 | 用途 |
|------|------|
| `cmd/server/` | 應用程式入口 |
| `internal/` | 業務邏輯模組（不對外暴露） |
| `pkg/` | 可共用的類型定義（ws 訊息、領域模型） |
| `migrations/` | SQL migration 檔案 |
| `web/` | React SPA 前端 |
| `docs/adr/` | 架構決策記錄（ADR） |
| `docs/specs/` | 功能規格書（SPEC） |
| `docs/openapi.yaml` | OpenAPI spec（API single source of truth） |
| `docs/api-changelog.md` | API 變更紀錄 |
| `docs/architecture.md` | 系統架構文件 |
| `.ai_profile` | ASP profile 設定 |
| `.asp/profiles/` | ASP 行為 profile 定義 |
| `.asp/templates/` | ADR / SPEC / 架構模板 |
| `.asp/hooks/` | SessionStart hook 腳本 |
