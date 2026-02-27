# SPEC-009：Integration Hardening & State Recovery

> Snapshot SQL 修正、Hub 狀態恢復接線、OpenAPI 文件化。

| 欄位 | 內容 |
|------|------|
| **規格 ID** | SPEC-009 |
| **關聯 ADR** | ADR-004（遊戲狀態管理） |
| **估算複雜度** | 中 |
| **建議模型** | Sonnet |
| **HITL 等級** | standard |
| **狀態** | Completed |

---

## 🎯 目標（Goal）

> 修復 SPEC-008 中 Snapshot SQL 引用不存在 `game_snapshots` 表的 Critical Bug，在 Hub 層接線 `RecoverFromSnapshot` 使 Server 重啟後能自動恢復 Room 狀態，並更新 OpenAPI spec 至 v0.9.0 文件化所有 WebSocket 事件。

---

## 📥 輸入規格（Inputs）

### 問題 1: Snapshot SQL Bug（Critical）

| 項目 | 原始（錯誤） | 修正 |
|------|-------------|------|
| `SaveSnapshot` | `INSERT INTO game_snapshots` | `UPDATE game_sessions SET state=$2, snapshot_seq=$3` |
| `LoadSnapshot` | `SELECT FROM game_snapshots` | `SELECT FROM game_sessions WHERE id=$1` |

ADR-004 明確規定 snapshot 存入 `game_sessions.state`，migration 中無 `game_snapshots` 表。

### 問題 2: RecoverFromSnapshot 未接線

`Room.RecoverFromSnapshot()` 方法存在但 `Hub.GetOrCreateRoom()` 從未呼叫。Server 重啟 = 狀態全部遺失。

### 問題 3: OpenAPI 過期

`openapi.yaml` 停在 v0.5.0，SPEC-006~008 的 WebSocket action/event types 未文件化。

---

## 📤 輸出規格（Expected Output）

### 修正後行為

| 項目 | 預期 |
|------|------|
| `SaveSnapshot` | 更新 `game_sessions` 表的 `state` 和 `snapshot_seq` |
| `LoadSnapshot` | 從 `game_sessions` 讀取，`snapshot_seq=0` 視為無 snapshot |
| Hub state recovery | Room 建立時自動 `RecoverFromSnapshot`，失敗則降級為全新狀態 |
| OpenAPI | v0.9.0，包含所有 WS action types 和 event types |

---

## ⚠️ 邊界條件（Edge Cases）

- `snapshot_seq=0` → 視為無 snapshot（區分「從未存」和「已存在」）
- `RecoverFromSnapshot` 在 `room.Run()` 前執行 → 無並發問題
- Recovery 失敗 → Warn log，降級為全新狀態（與 recovery 前行為相同）
- 不新增 migration（`game_sessions` 表已有 `state`/`snapshot_seq` 欄位）

---

## ✅ 驗收標準（Done When）

- [x] `SaveSnapshot` / `LoadSnapshot` 正確使用 `game_sessions` 表
- [x] `Hub.GetOrCreateRoom` 自動呼叫 `RecoverFromSnapshot`
- [x] Recovery 失敗時 Room 仍正常運作（graceful degradation）
- [x] +3 Hub state recovery 測試通過
- [x] OpenAPI spec 更新至 v0.9.0
- [x] `go test ./... -race` 全部通過（384 tests, 0 races）
- [x] `docs/api-changelog.md` 更新（v0.7.0~v0.9.0）

---

## 🚫 禁止事項（Out of Scope）

- 新增 migration
- 修改 Room/GameState 核心邏輯
- 前端實作
