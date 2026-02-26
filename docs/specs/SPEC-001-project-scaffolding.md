# SPEC-001：專案基礎建設（Phase 0）

> 建立可運行的 Go 專案骨架，讓後續 SPEC 有基礎可疊加。

| 欄位 | 內容 |
|------|------|
| **規格 ID** | SPEC-001 |
| **關聯 ADR** | ADR-001（技術棧選型） |
| **估算複雜度** | 中 |
| **建議模型** | Sonnet |
| **HITL 等級** | standard |

---

## 🎯 目標（Goal）

> 從零建立 TRPG-Simulation 的 Go 後端骨架：專案結構、設定載入、資料庫連線池、SQL migration（全部 9 張表）、health check endpoint、Docker Compose 部署。一條 `docker compose up` 指令即可啟動。

---

## 📥 輸入規格（Inputs）

| 參數名稱 | 型別 | 來源 | 限制條件 |
|----------|------|------|----------|
| PORT | int | 環境變數 | 預設 8080 |
| DATABASE_URL | string | 環境變數 | 必填，PostgreSQL 連線字串 |
| LOG_LEVEL | string | 環境變數 | 預設 "info"，可選 debug/info/warn/error |

---

## 📤 輸出規格（Expected Output）

**成功情境（GET /api/health）：**
```json
{
  "status": "ok",
  "timestamp": "2026-02-27T12:00:00Z",
  "database": "ok"
}
```
HTTP 200 OK

**失敗情境（DB 不可用）：**
```json
{
  "status": "degraded",
  "timestamp": "2026-02-27T12:00:00Z",
  "database": "error"
}
```
HTTP 503 Service Unavailable

**其他失敗情境：**

| 錯誤類型 | 處理方式 |
|----------|----------|
| DATABASE_URL 未設定 | 程式啟動失敗，log error 後 exit 1 |
| DB 連線失敗 | 程式啟動失敗，log error 後 exit 1 |
| 非 GET /api/health 路由 | 404 Not Found |

---

## ⚠️ 邊界條件（Edge Cases）

- DB 啟動中尚未 ready → health check 回傳 503 degraded
- 收到 SIGINT/SIGTERM → 10 秒 graceful shutdown
- middleware panic recovery → log stack trace，回傳 500

---

## ✅ 驗收標準（Done When）

- [ ] `go build ./...` 成功
- [ ] `go test ./... -v -race` 全數通過
- [ ] `golangci-lint run ./...` 無 error
- [ ] `docker compose build` 成功
- [ ] `docker compose up -d` 啟動 app + postgres
- [ ] `GET /api/health` 回傳 `{"status":"ok","database":"ok",...}`
- [ ] DB 停機時 `GET /api/health` 回傳 `{"status":"degraded","database":"error",...}`
- [ ] Migration 建立全部 9 張表
- [ ] `goose down` 可乾淨回滾

---

## 🚫 禁止事項（Out of Scope）

- 不要實作 JWT 認證（SPEC-002）
- 不要實作 WebSocket
- 不要實作任何業務邏輯（劇本、遊戲、骰子）
- 不要建立前端 React SPA
- 不要引入 gorilla/websocket、golang-jwt、expr-lang、yaml.v3

---

## 📎 參考資料（References）

- ADR-001：初始技術棧選型
- ADR-003：劇本資料模型（DB schema 來源）
- ADR-004：遊戲狀態管理（game_events 表 schema）
- ADR-005：認證與權限模型（users、refresh_tokens 表 schema）
- `docs/architecture.md`：領域模型 ER 圖
