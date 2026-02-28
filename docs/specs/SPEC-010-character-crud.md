# SPEC-010：Character CRUD & Session Assignment

> 角色建立/管理 CRUD 與場次角色指派。

| 欄位 | 內容 |
|------|------|
| **規格 ID** | SPEC-010 |
| **關聯 ADR** | ADR-004（遊戲狀態管理）、ADR-005（認證與權限模型） |
| **估算複雜度** | 中 |
| **建議模型** | Opus |
| **HITL 等級** | standard |
| **狀態** | Completed |

---

## 🎯 目標（Goal）

> 實作玩家角色 CRUD API 和場次角色指派端點，讓玩家能建立角色、管理屬性與道具，並在加入場次後指派角色參與遊戲。DB `characters` 表已在 migration 中建立，`session_players.character_id` FK 已存在但目前永遠為 NULL。

---

## 📥 輸入規格（Inputs）

### REST API Endpoints

| Method | Path | 角色 | 說明 |
|--------|------|------|------|
| POST | `/api/v1/characters` | User | 建立角色 |
| GET | `/api/v1/characters` | User | 列出自己的角色（分頁） |
| GET | `/api/v1/characters/{id}` | Owner | 取得角色詳情 |
| PUT | `/api/v1/characters/{id}` | Owner | 更新角色 |
| DELETE | `/api/v1/characters/{id}` | Owner | 刪除角色 |
| POST | `/api/v1/sessions/{id}/characters` | Player | 指派角色至場次 |

### Request Bodies

**POST /api/v1/characters:**
```json
{
  "name": "Aragorn",
  "attributes": {"str": 16, "dex": 14},
  "inventory": ["sword", "shield"],
  "notes": "Ranger of the North"
}
```
- `name`：必填，1-100 字元
- `attributes`：選填，JSON object，預設 `{}`
- `inventory`：選填，JSON array，預設 `[]`
- `notes`：選填，預設空字串

**PUT /api/v1/characters/{id}:** 同 POST 格式

**POST /api/v1/sessions/{id}/characters:**
```json
{
  "characterId": "uuid"
}
```

---

## 📤 輸出規格（Expected Output）

**CharacterResponse:**
```json
{
  "id": "uuid",
  "userId": "uuid",
  "name": "Aragorn",
  "attributes": {"str": 16, "dex": 14},
  "inventory": ["sword", "shield"],
  "notes": "Ranger of the North",
  "createdAt": "2026-02-28T12:00:00Z",
  "updatedAt": "2026-02-28T12:00:00Z"
}
```

**CharacterListResponse:**
```json
{
  "characters": [...],
  "total": 5,
  "limit": 20,
  "offset": 0
}
```

**失敗情境：**

| 錯誤類型 | HTTP Code | 說明 |
|----------|-----------|------|
| Invalid JSON / 欄位驗證 | 400 | 含 field + reason details |
| 未認證 | 401 | Missing/invalid JWT |
| 非擁有者 | 403 | 角色不屬於當前用戶 |
| 角色/場次不存在 | 404 | — |
| 角色已指派至場次 | 409 | 刪除時若有 session_players 引用 |
| 場次非 lobby 狀態 | 409 | 角色指派只允許在 lobby |

---

## ⚠️ 邊界條件（Edge Cases）

- 刪除角色時若 `session_players` 有引用 → 409 CONFLICT（FK RESTRICT）
- 角色指派限 `lobby` 狀態 → 遊戲開始後不可換角色
- attributes 傳入非 object JSON → 400
- inventory 傳入非 array JSON → 400
- 兩個請求同時指派不同角色到同一 player → 最後寫入者勝出（PostgreSQL 原子性）
- SessionPlayerResponse 新增 `characterId` 欄位 → 向後相容（nullable）

---

## ✅ 驗收標準（Done When）

- [x] 6 個 REST endpoint 全部實作
- [x] Character CRUD repository 含 6 methods
- [x] `SessionPlayerResponse` 包含 `characterId` 欄位
- [x] ~42 個新測試通過（~12 integration + ~30 unit）
- [x] `go test ./... -race` 全部通過
- [x] `go vet ./...` 無警告
- [x] `docs/openapi.yaml` 更新至 v0.10.0
- [x] `docs/api-changelog.md` 更新

---

## 🚫 禁止事項（Out of Scope）

- 不新增 migration（`characters` 表已存在）
- 不修改 `internal/player/`（保留給未來 player profile）
- 不修改 realtime WebSocket 邏輯
- 不引入新依賴

---

## 📎 參考資料（References）

- 現有 CRUD 模式：`internal/scenario/repository.go`、`internal/server/scenario_handlers.go`
- DB schema：`migrations/20260227120000_create_initial_schema.sql`（characters + session_players）
