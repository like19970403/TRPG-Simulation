# [ADR-005]: 認證與權限模型

| 欄位 | 內容 |
|------|------|
| **狀態** | `Accepted` |
| **日期** | 2026-02-27 |
| **決策者** | 專案擁有者 |

---

## 背景（Context）

TRPG-Simulation 有兩種角色：GM（主持人）和 Player（玩家），兩者權限差異極大。GM 擁有劇本完整可見性和遊戲控制權，玩家只能看到經過濾的資訊。需要決定認證方案、權限模型，以及 WebSocket 連線的授權機制。

---

## 評估選項（Options Considered）

### 認證方案

#### 選項 A：JWT（Access Token + Refresh Token）

- **優點**：無狀態、前端 SPA 友善、Go 實作簡單（golang-jwt）、WebSocket 連線時可用 token 驗證
- **缺點**：token 撤銷需額外機制（黑名單或短效期）
- **風險**：低。短效期 access token + refresh token 是業界標準

#### 選項 B：Session Cookie

- **優點**：傳統方案、伺服器端可即時撤銷
- **缺點**：需 session store（Redis/DB）、跨域配置複雜、WebSocket 連線帶 cookie 不直覺
- **風險**：低，但增加狀態管理複雜度

#### 選項 C：OAuth 2.0（第三方登入）

- **優點**：用戶無需記住密碼
- **缺點**：依賴第三方服務、需處理多種 provider、MVP 階段過度
- **風險**：中。可作為未來功能，不應是 MVP 唯一方案

### 權限模型

#### 選項 A：角色基礎存取控制（RBAC）— 簡化版

- **優點**：GM / Player 兩種角色清晰、實作簡單、與 TRPG 語境自然對應
- **缺點**：無法做細粒度權限控制（如特定場景的特殊權限）
- **風險**：低。TRPG 場景的權限需求正好是 GM vs Player 二分

#### 選項 B：屬性基礎存取控制（ABAC）

- **優點**：極細粒度控制
- **缺點**：實作複雜、TRPG 場景不需要這麼細的控制
- **風險**：高。過度設計

---

## 決策（Decision）

選擇 **JWT 認證 + 簡化 RBAC 權限模型**。

### 認證流程

#### 註冊 / 登入

```
POST /api/auth/register  →  { username, email, password }  →  { user_id }
POST /api/auth/login      →  { email, password }            →  { access_token, refresh_token }
POST /api/auth/refresh    →  { refresh_token }               →  { access_token }
```

#### JWT Token 設計

**Access Token**（短效期）：
```json
{
  "sub": "user_uuid",
  "username": "player1",
  "exp": 1709024400,
  "iat": 1709020800
}
```
- 有效期：**15 分鐘**
- 儲存：前端記憶體（不存 localStorage，防 XSS）

**Refresh Token**（長效期）：
- 有效期：**7 天**
- 儲存：HttpOnly Cookie（防 XSS）
- 資料庫記錄 refresh token hash，支援撤銷

```sql
CREATE TABLE refresh_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id),
    token_hash  VARCHAR(64) NOT NULL,  -- SHA-256 hash
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_refresh_tokens_user ON refresh_tokens(user_id);
```

#### 密碼儲存

- 使用 **bcrypt**（cost factor 12）
- Go stdlib `golang.org/x/crypto/bcrypt`

### 權限模型

#### 系統層級角色

TRPG-Simulation 不需要全域管理員角色（個人專案）。所有註冊用戶平等，角色差異體現在 GameSession 層級。

#### GameSession 層級角色

| 角色 | 說明 | 來源 |
|------|------|------|
| **GM** | 遊戲主持人，完整控制權 | 建立 GameSession 的用戶 |
| **Player** | 玩家，受限可見性 | 透過邀請碼加入的用戶 |

#### 權限矩陣

| 操作 | GM | Player |
|------|:--:|:------:|
| 建立 GameSession | ✓ | ✓（任何人都可建立並成為 GM） |
| 開始/暫停/結束遊戲 | ✓ | ✗ |
| 切換場景 | ✓ | ✗ |
| 手動揭露道具 | ✓ | ✗ |
| 查看 GM 筆記 | ✓ | ✗ |
| 查看所有玩家位置 | ✓ | ✗ |
| 查看未揭露道具 | ✓ | ✗ |
| 投放訊息給玩家 | ✓ | ✗ |
| 揭露 NPC 角色卡欄位 | ✓ | ✗ |
| 做出選擇 | ✗ | ✓ |
| 擲骰子 | ✓ | ✓ |
| 查看當前場景內容 | ✓（完整） | ✓（過濾後） |
| 查看已揭露道具 | ✓（全部） | ✓（僅自己的） |
| 查看 NPC 角色卡 | ✓（全部欄位） | ✓（僅已揭露欄位） |
| 撰寫筆記 | ✓（GM 筆記） | ✓（玩家個人筆記） |

### WebSocket 授權

#### 連線建立

1. Client 發起 WebSocket 連線，URL 帶上 access token：
   ```
   ws://host/api/ws?token=<access_token>&session_id=<uuid>
   ```
2. Server 驗證 JWT token 有效性
3. Server 驗證該用戶是 GameSession 的成員（GM 或已加入的 Player）
4. 驗證通過後升級為 WebSocket 連線，建立 Client 結構

#### 連線中權限檢查

- 每條 WebSocket 訊息都在 Room goroutine 中檢查 `sender_role`
- GM 操作（場景切換、道具揭露）驗證 `sender_role == GM`
- Player 操作（選擇、擲骰）驗證 `sender_role == Player`

#### 廣播過濾

伺服器端過濾是安全核心，玩家永遠不會收到：

| 資料類型 | GM 收到 | Player 收到 |
|----------|---------|------------|
| 場景 content | ✓ | ✓ |
| 場景 gm_notes | ✓ | ✗ |
| 未揭露道具 | ✓ | ✗ |
| 已揭露道具（自己的） | ✓ | ✓ |
| 已揭露道具（他人的） | ✓ | ✗（除非 GM 設為公開） |
| 其他玩家位置 | ✓ | ✗ |
| 劇本變數 | ✓ | ✗ |
| 所有骰子結果 | ✓ | ✓（僅公開的） |
| NPC 角色卡完整資料 | ✓ | ✗ |
| NPC 已揭露欄位 | ✓ | ✓（僅自己被揭露的） |
| GM 投放訊息 | ✓ | ✓（僅指定對象） |

### REST API 中間件

```
Request → JWT Middleware → Route Handler
              │
              ├── 解析 Authorization header
              ├── 驗證 token 簽名和過期時間
              ├── 將 user_id 注入 context
              └── 401 Unauthorized（無效 token）
```

GameSession 相關 API 額外檢查：
- `GET /api/sessions/:id` — 需是 session 成員
- `PUT /api/sessions/:id` — 需是 GM
- `DELETE /api/sessions/:id` — 需是 GM

### 邀請碼機制

```sql
-- game_sessions 表已有 invite_code 欄位
-- invite_code: 6 位英數字，大小寫不敏感
```

- GM 建立 GameSession 時自動生成邀請碼
- 玩家用邀請碼加入：`POST /api/sessions/join { "invite_code": "ABC123" }`
- 邀請碼在遊戲開始後可由 GM 選擇失效（防止中途加入）
- 邀請碼不含易混淆字元（0/O、1/I/l）

---

## 後果（Consequences）

**正面影響：**
- JWT 無狀態認證適合 SPA + WebSocket 架構
- 簡化 RBAC（GM / Player）與 TRPG 語境自然對應
- 伺服器端廣播過濾確保玩家永遠不會收到未授權資料
- Refresh token 存 DB 可精確撤銷

**負面影響 / 技術債：**
- Access token 存記憶體，重新整理頁面需用 refresh token 重新取得
- 無 OAuth 第三方登入（可作為未來功能）
- 無全域管理員角色（個人專案暫不需要）

**後續追蹤：**
- [ ] SPEC：JWT middleware 實作細節
- [ ] SPEC：WebSocket 授權流程時序圖
- [ ] 未來考慮：OAuth 2.0 第三方登入
- [ ] SPEC：GM 投放訊息（gm_broadcast）與圖片上傳流程

---

## 關聯（Relations）

- 取代：（無）
- 被取代：（無）
- 參考：ADR-001（技術棧選型）、ADR-002（即時通訊策略）
