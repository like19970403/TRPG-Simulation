# SPEC-002：JWT 認證

> 實作用戶註冊、登入、token 刷新、登出，以及 JWT auth middleware。

| 欄位 | 內容 |
|------|------|
| **規格 ID** | SPEC-002 |
| **關聯 ADR** | ADR-005（認證與權限模型）、ADR-001（技術棧選型） |
| **估算複雜度** | 高 |
| **建議模型** | Opus |
| **HITL 等級** | standard |

---

## 🎯 目標（Goal）

> 為所有後續業務 API 提供認證基礎：用戶可註冊帳號、以 email/password 登入取得 JWT access token，透過 refresh token 延長 session，並安全登出。

---

## 📥 輸入規格（Inputs）

### POST /api/v1/users（註冊）

| 參數名稱 | 型別 | 來源 | 限制條件 |
|----------|------|------|----------|
| username | string | JSON body | 3-50 字元，`^[a-zA-Z0-9_]+$` |
| email | string | JSON body | 有效 email 格式，≤255 字元 |
| password | string | JSON body | 8-72 字元（72 為 bcrypt 上限） |

### POST /api/v1/auth/login（登入）

| 參數名稱 | 型別 | 來源 | 限制條件 |
|----------|------|------|----------|
| email | string | JSON body | 有效 email 格式 |
| password | string | JSON body | 非空 |

### POST /api/v1/auth/refresh（刷新）

| 參數名稱 | 型別 | 來源 | 限制條件 |
|----------|------|------|----------|
| refresh_token | string | HttpOnly Cookie | 由登入/刷新時自動設定 |

### POST /api/v1/auth/logout（登出）

| 參數名稱 | 型別 | 來源 | 限制條件 |
|----------|------|------|----------|
| Authorization | string | Header | `Bearer <access_token>` |
| refresh_token | string | HttpOnly Cookie | 可選（有則撤銷） |

---

## 📤 輸出規格（Expected Output）

**註冊成功（201 Created）：**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "username": "player1",
  "email": "player1@example.com",
  "createdAt": "2026-02-27T12:00:00Z"
}
```

**登入成功（200 OK）：**
```json
{
  "accessToken": "eyJhbGciOiJIUzI1NiIs...",
  "expiresIn": 900,
  "tokenType": "Bearer"
}
```
+ `Set-Cookie: refresh_token=<hex>; Path=/api/v1/auth; HttpOnly; Secure; SameSite=Strict; Max-Age=604800`

**刷新成功（200 OK）：** 同登入回應格式 + 新 Cookie

**登出成功（204 No Content）：** 無回應 body + 清除 Cookie

**失敗情境：**

| 錯誤類型 | HTTP Code | error code | 說明 |
|----------|-----------|------------|------|
| 參數驗證失敗 | 400 | VALIDATION_ERROR | 含 details 欄位級別錯誤 |
| 帳密錯誤 | 401 | INVALID_CREDENTIALS | 不透露是 email 還是密碼錯 |
| Token 無效/過期 | 401 | UNAUTHORIZED / TOKEN_EXPIRED | access token 問題 |
| Token 已撤銷 | 401 | TOKEN_REVOKED | refresh token 被撤銷 |
| 重複 username/email | 409 | CONFLICT | 不透露是哪個欄位衝突 |

---

## ⚠️ 邊界條件（Edge Cases）

- 同時以相同 email 重複註冊 → DB UNIQUE constraint 保護，回 409
- Refresh token 在過期後使用 → 回 401 TOKEN_EXPIRED
- 已撤銷的 refresh token 被重複使用 → 撤銷該用戶所有 token（防 token theft）
- Password 剛好 72 bytes（bcrypt 上限） → 正常處理
- 空 JSON body / Content-Type 非 application/json → 400 VALIDATION_ERROR
- JWT_SECRET 未設定 → 應用程式啟動失敗（env required,notEmpty）
- bcrypt cost 超出合理範圍 → config 驗證

---

## ✅ 驗收標準（Done When）

- [ ] `go test ./... -v -race` 全數通過
- [ ] `golangci-lint run ./...` 無 error
- [ ] `POST /api/v1/users` 可成功註冊，重複 email/username 回 409
- [ ] `POST /api/v1/auth/login` 可成功登入，回傳 JWT + Set-Cookie
- [ ] `POST /api/v1/auth/refresh` 以 Cookie 刷新 token，舊 token 被撤銷
- [ ] `POST /api/v1/auth/logout` 撤銷 refresh token + 清除 Cookie
- [ ] 受保護端點無 Bearer token 回 401
- [ ] API 回應符合 OpenAPI spec 定義
- [ ] `docs/api-changelog.md` 已更新

---

## 🚫 禁止事項（Out of Scope）

- 不要實作 session-level 權限（GM/Player）— 後續 SPEC
- 不要實作 WebSocket 認證
- 不要實作 email 驗證 / 忘記密碼
- 不要引入 gorilla/websocket、expr-lang、yaml.v3
- 不要修改 DB schema（users + refresh_tokens 表已存在）

---

## 📎 參考資料（References）

- ADR-005：認證與權限模型
- ADR-001：初始技術棧選型
- `migrations/20260227120000_create_initial_schema.sql`：DB schema
- `docs/openapi.yaml`：API spec (v0.2.0)
