# SPEC-003：劇本 CRUD

> 實作劇本的建立、讀取、更新、刪除，以及生命週期管理（draft → published → archived）。

| 欄位 | 內容 |
|------|------|
| **規格 ID** | SPEC-003 |
| **關聯 ADR** | ADR-003（劇本資料模型與 DSL 設計） |
| **估算複雜度** | 高 |
| **建議模型** | Opus |
| **HITL 等級** | standard |

---

## 🎯 目標（Goal）

> GM 可透過 REST API 管理自己的劇本：建立新劇本、列出所有劇本、查看詳情、編輯草稿、刪除草稿、發布劇本供遊戲使用、封存不再使用的劇本。

---

## 📥 輸入規格（Inputs）

### POST /api/v1/scenarios（建立劇本）

| 參數名稱 | 型別 | 來源 | 限制條件 |
|----------|------|------|----------|
| title | string | JSON body | 1-200 字元，必填 |
| description | string | JSON body | 選填 |
| content | object | JSON body | 必填，有效 JSON 物件 |
| Authorization | string | Header | `Bearer <access_token>` |

### GET /api/v1/scenarios（列出劇本）

| 參數名稱 | 型別 | 來源 | 限制條件 |
|----------|------|------|----------|
| limit | int | Query | 1-100，預設 20 |
| offset | int | Query | ≥0，預設 0 |
| Authorization | string | Header | `Bearer <access_token>` |

### GET /api/v1/scenarios/{id}（取得劇本）

| 參數名稱 | 型別 | 來源 | 限制條件 |
|----------|------|------|----------|
| id | UUID | URL path | 有效 UUID 格式 |
| Authorization | string | Header | `Bearer <access_token>` |

### PUT /api/v1/scenarios/{id}（更新劇本）

| 參數名稱 | 型別 | 來源 | 限制條件 |
|----------|------|------|----------|
| id | UUID | URL path | 有效 UUID 格式 |
| title | string | JSON body | 1-200 字元，必填 |
| description | string | JSON body | 選填（空字串代表清除） |
| content | object | JSON body | 必填，有效 JSON 物件 |
| Authorization | string | Header | `Bearer <access_token>` |

### DELETE /api/v1/scenarios/{id}（刪除劇本）

| 參數名稱 | 型別 | 來源 | 限制條件 |
|----------|------|------|----------|
| id | UUID | URL path | 有效 UUID 格式 |
| Authorization | string | Header | `Bearer <access_token>` |

### POST /api/v1/scenarios/{id}/publish（發布劇本）

| 參數名稱 | 型別 | 來源 | 限制條件 |
|----------|------|------|----------|
| id | UUID | URL path | 有效 UUID 格式 |
| Authorization | string | Header | `Bearer <access_token>` |

### POST /api/v1/scenarios/{id}/archive（封存劇本）

| 參數名稱 | 型別 | 來源 | 限制條件 |
|----------|------|------|----------|
| id | UUID | URL path | 有效 UUID 格式 |
| Authorization | string | Header | `Bearer <access_token>` |

---

## 📤 輸出規格（Expected Output）

**建立成功（201 Created）：**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "authorId": "660e8400-e29b-41d4-a716-446655440000",
  "title": "迷霧森林",
  "description": "一個神秘的森林冒險",
  "version": 1,
  "status": "draft",
  "content": {"start_scene": "s1", "scenes": [{"id": "s1", "name": "Start"}]},
  "createdAt": "2026-02-27T12:00:00Z",
  "updatedAt": "2026-02-27T12:00:00Z"
}
```

**列出成功（200 OK）：**
```json
{
  "scenarios": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "authorId": "660e8400-e29b-41d4-a716-446655440000",
      "title": "迷霧森林",
      "description": "一個神秘的森林冒險",
      "version": 1,
      "status": "draft",
      "content": {},
      "createdAt": "2026-02-27T12:00:00Z",
      "updatedAt": "2026-02-27T12:00:00Z"
    }
  ],
  "total": 1,
  "limit": 20,
  "offset": 0
}
```

**取得/更新/發布/封存成功（200 OK）：** 同建立成功格式

**刪除成功（204 No Content）：** 無回應 body

**失敗情境：**

| 錯誤類型 | HTTP Code | error code | 說明 |
|----------|-----------|------------|------|
| 參數驗證失敗 | 400 | VALIDATION_ERROR | 含 details 欄位級別錯誤 |
| 未認證 | 401 | UNAUTHORIZED | 缺少或無效的 access token |
| 無權限 | 403 | FORBIDDEN | 非劇本作者 |
| 找不到 | 404 | NOT_FOUND | 劇本不存在 |
| 狀態衝突 | 409 | CONFLICT | 劇本狀態不允許此操作 |

---

## ⚠️ 邊界條件（Edge Cases）

- 更新/刪除非 draft 狀態的劇本 → 409 CONFLICT
- 發布非 draft 劇本 → 409 CONFLICT
- 封存非 published 劇本 → 409 CONFLICT
- 操作他人的劇本 → 403 FORBIDDEN（不透露劇本是否存在）
- UUID 格式不正確 → 400 VALIDATION_ERROR
- 空 JSON body / Content-Type 非 application/json → 400 VALIDATION_ERROR
- content 為無效 JSON → 400 VALIDATION_ERROR
- title 超過 200 字元 → 400 VALIDATION_ERROR
- 列出劇本但無任何劇本 → 200 with 空陣列（不是 404）
- limit 超出範圍 → 使用預設值（不報錯）

---

## ✅ 驗收標準（Done When）

- [ ] `go test ./... -v -race` 全數通過
- [ ] `go vet ./...` 無 error
- [ ] 7 個 scenario endpoints 全部可運作
- [ ] 只有劇本作者可操作自己的劇本
- [ ] draft 劇本可更新/刪除/發布
- [ ] published 劇本可封存，不可更新/刪除
- [ ] API 回應符合 OpenAPI spec 定義
- [ ] `docs/api-changelog.md` 已更新

---

## 🚫 禁止事項（Out of Scope）

- 不要實作 YAML 解析或場景圖驗證 — 後續 SPEC
- 不要實作 expr 表達式驗證 — 後續 SPEC
- 不要實作公開劇本瀏覽 — 僅列出自己的劇本
- 不要實作 unpublish（published → draft）
- 不要修改 DB schema
- 不要引入新的外部依賴

---

## 📎 參考資料（References）

- ADR-003：劇本資料模型與 DSL 設計
- `migrations/20260227120000_create_initial_schema.sql`：DB schema
- `docs/openapi.yaml`：API spec (v0.3.0)
