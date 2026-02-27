# SPEC-004：遊戲場次管理

> 實作遊戲場次的建立、生命週期管理（lobby → active ↔ paused → completed）、玩家透過邀請碼加入/離開。

| 欄位 | 內容 |
|------|------|
| **規格 ID** | SPEC-004 |
| **關聯 ADR** | ADR-004（遊戲狀態管理）、ADR-005（認證與權限模型） |
| **估算複雜度** | 高 |
| **建議模型** | Opus |
| **HITL 等級** | standard |

---

## 🎯 目標（Goal）

> GM 可基於已發布的劇本建立遊戲場次，透過邀請碼邀請玩家加入，並管理場次生命週期（開始、暫停、恢復、結束）。玩家可透過邀請碼加入場次或自行離開。

---

## 📥 輸入規格（Inputs）

### POST /api/v1/sessions（建立場次）

| 參數名稱 | 型別 | 來源 | 限制條件 |
|----------|------|------|----------|
| scenarioId | UUID | JSON body | 必填，有效 UUID，對應已發布劇本 |
| Authorization | string | Header | `Bearer <access_token>` |

### GET /api/v1/sessions（列出場次）

| 參數名稱 | 型別 | 來源 | 限制條件 |
|----------|------|------|----------|
| limit | int | Query | 1-100，預設 20 |
| offset | int | Query | ≥0，預設 0 |
| Authorization | string | Header | `Bearer <access_token>` |

### GET /api/v1/sessions/{id}（取得場次）

| 參數名稱 | 型別 | 來源 | 限制條件 |
|----------|------|------|----------|
| id | UUID | URL path | 有效 UUID 格式 |
| Authorization | string | Header | `Bearer <access_token>` |

### POST /api/v1/sessions/{id}/start（開始遊戲）

| 參數名稱 | 型別 | 來源 | 限制條件 |
|----------|------|------|----------|
| id | UUID | URL path | 有效 UUID 格式 |
| Authorization | string | Header | `Bearer <access_token>`（GM only） |

### POST /api/v1/sessions/{id}/pause（暫停遊戲）

| 參數名稱 | 型別 | 來源 | 限制條件 |
|----------|------|------|----------|
| id | UUID | URL path | 有效 UUID 格式 |
| Authorization | string | Header | `Bearer <access_token>`（GM only） |

### POST /api/v1/sessions/{id}/resume（恢復遊戲）

| 參數名稱 | 型別 | 來源 | 限制條件 |
|----------|------|------|----------|
| id | UUID | URL path | 有效 UUID 格式 |
| Authorization | string | Header | `Bearer <access_token>`（GM only） |

### POST /api/v1/sessions/{id}/end（結束遊戲）

| 參數名稱 | 型別 | 來源 | 限制條件 |
|----------|------|------|----------|
| id | UUID | URL path | 有效 UUID 格式 |
| Authorization | string | Header | `Bearer <access_token>`（GM only） |

### POST /api/v1/sessions/join（加入場次）

| 參數名稱 | 型別 | 來源 | 限制條件 |
|----------|------|------|----------|
| inviteCode | string | JSON body | 必填，1-10 字元 |
| Authorization | string | Header | `Bearer <access_token>` |

### GET /api/v1/sessions/{id}/players（列出玩家）

| 參數名稱 | 型別 | 來源 | 限制條件 |
|----------|------|------|----------|
| id | UUID | URL path | 有效 UUID 格式 |
| Authorization | string | Header | `Bearer <access_token>`（GM 或成員） |

### DELETE /api/v1/sessions/{id}/players/{userId}（移除玩家）

| 參數名稱 | 型別 | 來源 | 限制條件 |
|----------|------|------|----------|
| id | UUID | URL path | 有效 UUID 格式 |
| userId | UUID | URL path | 有效 UUID 格式 |
| Authorization | string | Header | `Bearer <access_token>`（GM 或本人） |

---

## 📤 輸出規格（Expected Output）

**建立場次成功（201 Created）：**
```json
{
  "id": "770e8400-e29b-41d4-a716-446655440000",
  "scenarioId": "550e8400-e29b-41d4-a716-446655440000",
  "gmId": "660e8400-e29b-41d4-a716-446655440000",
  "status": "lobby",
  "inviteCode": "ABC123",
  "createdAt": "2026-02-27T12:00:00Z"
}
```

**列出場次成功（200 OK）：**
```json
{
  "sessions": [
    {
      "id": "770e8400-e29b-41d4-a716-446655440000",
      "scenarioId": "550e8400-e29b-41d4-a716-446655440000",
      "gmId": "660e8400-e29b-41d4-a716-446655440000",
      "status": "lobby",
      "inviteCode": "ABC123",
      "createdAt": "2026-02-27T12:00:00Z"
    }
  ],
  "total": 1,
  "limit": 20,
  "offset": 0
}
```

**列出玩家成功（200 OK）：**
```json
{
  "players": [
    {
      "id": "880e8400-e29b-41d4-a716-446655440000",
      "sessionId": "770e8400-e29b-41d4-a716-446655440000",
      "userId": "990e8400-e29b-41d4-a716-446655440000",
      "status": "joined",
      "joinedAt": "2026-02-27T12:30:00Z"
    }
  ]
}
```

**狀態變更成功（200 OK）：** 同場次 response 格式，status 更新為新狀態

**移除玩家成功（204 No Content）：** 無回應 body

**失敗情境：**

| 錯誤類型 | HTTP Code | error code | 說明 |
|----------|-----------|------------|------|
| 參數驗證失敗 | 400 | VALIDATION_ERROR | 含 details 欄位級別錯誤 |
| 未認證 | 401 | UNAUTHORIZED | 缺少或無效的 access token |
| 無權限 | 403 | FORBIDDEN | 非 GM / 非成員 / 非本人 |
| 找不到 | 404 | NOT_FOUND | 場次/劇本/玩家不存在 |
| 狀態衝突 | 409 | CONFLICT | 狀態不允許此操作 / 重複加入 / GM 不可加入 |

---

## ⚠️ 邊界條件（Edge Cases）

- 建立場次時 scenarioId 對應的劇本不存在 → 404
- 建立場次時劇本狀態非 published → 409 CONFLICT
- 開始非 lobby 狀態的場次 → 409 CONFLICT
- 暫停非 active 狀態的場次 → 409 CONFLICT
- 恢復非 paused 狀態的場次 → 409 CONFLICT
- 結束已結束的場次 → 409 CONFLICT
- 非 GM 嘗試操作場次生命週期 → 403 FORBIDDEN
- 非 GM 也非成員的用戶取得場次 → 403 FORBIDDEN
- GM 嘗試加入自己的場次 → 409 CONFLICT
- 重複加入場次 → 409 CONFLICT
- 場次非 lobby 時加入 → 409 CONFLICT
- 邀請碼不存在 → 404
- 邀請碼大小寫不敏感（handler 中 ToUpper）
- 玩家嘗試踢其他玩家 → 403 FORBIDDEN
- 移除已結束/已廢棄場次的玩家 → 409 CONFLICT

---

## ✅ 驗收標準（Done When）

- [ ] `go test ./... -v -race` 全數通過
- [ ] `go vet ./...` 無 error
- [ ] 10 個 session endpoints 全部可運作
- [ ] 只有 GM 可操作場次生命週期
- [ ] GM 和成員可查看場次/玩家列表
- [ ] 邀請碼加入正常運作
- [ ] 狀態機轉換正確（lobby→active↔paused→completed）
- [ ] API 回應符合 OpenAPI spec 定義
- [ ] `docs/api-changelog.md` 已更新

---

## 🚫 禁止事項（Out of Scope）

- 不要實作 WebSocket 即時通訊 — 後續 SPEC
- 不要實作事件溯源（game_events） — 後續 SPEC
- 不要實作骰子引擎或場景切換 — 後續 SPEC
- 不要暴露 state / gm_notes / snapshot_seq 於 REST API
- 不要修改 DB schema
- 不要引入新的外部依賴

---

## 📎 參考資料（References）

- ADR-004：遊戲狀態管理
- ADR-005：認證與權限模型
- `migrations/20260227120000_create_initial_schema.sql`：DB schema
- `docs/openapi.yaml`：API spec (v0.4.0)
