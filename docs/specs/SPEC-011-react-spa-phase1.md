# SPEC-011：React SPA Frontend — Phase 1（框架搭建 + Auth）

> 從零搭建 React SPA 前端，Phase 1 範圍：Vite 專案搭建、Tailwind CSS 暗色主題、Auth 頁面（登入/註冊/登出）、API client、protected route guard。

| 欄位 | 內容 |
|------|------|
| **規格 ID** | SPEC-011 |
| **關聯 ADR** | ADR-001（技術棧）、ADR-005（認證與權限模型） |
| **估算複雜度** | 高 |
| **建議模型** | Opus |
| **HITL 等級** | standard |
| **狀態** | Completed |

---

## 🎯 目標（Goal）

> 後端 API 已完成 22 個 REST endpoint + WebSocket（SPEC-001~010），`web/` 目前為空。需從零搭建 React SPA，Phase 1 實作框架搭建與 Auth 流程，包含：登入、註冊、登出、JWT token 自動刷新、protected route guard。UI 設計已於 Pencil (`docs/designs/pencil-new.pen`) 確認。

---

## 📐 技術決策

| 項目 | 選擇 |
|------|------|
| 框架 | React 18 + TypeScript |
| 建置 | Vite 6 |
| 樣式 | Tailwind CSS v4（`@tailwindcss/vite` + `@theme`） |
| 狀態管理 | Zustand |
| 路由 | React Router v7 |
| 主題 | 暗色模式、champagne gold `#C9A962`、Playfair Display + Manrope |
| 測試 | Vitest + @testing-library/react |

---

## 📥 輸入規格（Inputs）

### 後端 Auth API（已實作，鏡像使用）

| Method | Path | 說明 |
|--------|------|------|
| POST | `/api/v1/users` | 註冊 |
| POST | `/api/v1/auth/login` | 登入（回傳 access token + set refresh cookie） |
| POST | `/api/v1/auth/refresh` | 刷新 token（讀取 refresh cookie） |
| POST | `/api/v1/auth/logout` | 登出（revoke refresh token） |

### Request/Response Types（鏡像 `internal/server/types.go`）

**RegisterRequest:**
```json
{ "username": "string", "email": "string", "password": "string" }
```

**LoginRequest:**
```json
{ "email": "string", "password": "string" }
```

**TokenResponse:**
```json
{ "accessToken": "string", "expiresIn": 900, "tokenType": "Bearer" }
```

**ErrorResponse:**
```json
{ "error": "VALIDATION_ERROR", "message": "string", "details": [{"field": "email", "reason": "string"}] }
```

### 前端驗證規則（鏡像 `internal/server/auth_handlers.go:233-259`）

| 欄位 | 規則 |
|------|------|
| username | 3-50 字元，`^[a-zA-Z0-9_]+$` |
| email | 合法 email 格式，≤255 字元 |
| password | 8-72 字元 |

### Pencil 設計系統色彩（來自 `docs/designs/pencil-new.pen`）

```
bg-page: #0F0F0F     bg-sidebar: #0A0A0A    bg-card: #161616
border: #1F1F1F
gold: #C9A962         gold-tint: #C9A96210   gold-tint-30: #C9A96230
text-primary: #FAF8F5  text-secondary: #888888  text-tertiary: #666666
text-on-gold: #0A0A0A
parchment: #2A2318    parchment-light: #3D3425
error: #F87171        success: #4ADE80        info: #60A5FA
```

---

## 📤 輸出規格（Expected Output）

### 目錄結構

```
web/
  index.html
  package.json
  vite.config.ts
  tsconfig.json / tsconfig.app.json / tsconfig.node.json
  eslint.config.js
  public/favicon.svg
  src/
    main.tsx                    # entry
    App.tsx                     # RouterProvider
    index.css                   # Tailwind @theme
    vite-env.d.ts
    api/
      client.ts                 # fetch wrapper + auto-refresh interceptor
      auth.ts                   # register / login / refresh / logout
      types.ts                  # 鏡像 backend types.go
    stores/
      auth-store.ts             # Zustand: token, user, isAuthenticated
    hooks/
      use-auth.ts               # login/register/logout 封裝
    components/
      ui/button.tsx
      ui/input.tsx
      ui/form-field.tsx
      ui/loading-spinner.tsx
      auth-guard.tsx            # 未登入 → /login
      guest-guard.tsx           # 已登入 → /dashboard
    layouts/
      auth-layout.tsx           # 置中卡片（登入/註冊用）
      app-layout.tsx            # 頂部導航列 + Outlet
    pages/
      login-page.tsx
      register-page.tsx
      dashboard-page.tsx        # 佔位：Welcome, {username}
      not-found-page.tsx
    lib/
      cn.ts                     # clsx wrapper
      constants.ts              # route paths
    router.tsx                  # React Router 路由定義
```

### 頁面設計（已於 Pencil 確認）

1. **Login Page** — 暗色全屏背景、置中卡片、TRPG logo、Email/Password 輸入框、金色 Sign In 按鈕、Register 連結
2. **Register Page** — 同樣卡片風格、Username/Email/Password 三個輸入框、金色 Create Account 按鈕、Login 連結
3. **Dashboard Page** — 頂部導航列（logo + username + logout）、"Welcome, {username}" 大標題、三張佔位卡片（Scenarios/Sessions/Characters）

### Auth 流程

1. Access token 只存記憶體（Zustand store），不存 localStorage
2. Refresh token 透過 HttpOnly cookie 傳送（`credentials: 'include'`）
3. 頁面刷新時，AuthGuard 嘗試 `POST /api/v1/auth/refresh` 恢復登入
4. API client 攔截 401 → 自動 refresh → 重試原請求
5. Refresh 失敗 → 清除 auth state → redirect `/login`
6. 併發 refresh 防護（mutex flag）

---

## ⚠️ 邊界條件（Edge Cases）

- 頁面刷新時無 access token → AuthGuard 嘗試 refresh，成功則維持登入
- Refresh cookie 過期 → 自動導向 /login
- 多個併發 401 請求 → 只觸發一次 refresh，其餘等待結果
- 已登入用戶訪問 /login 或 /register → GuestGuard 導向 /dashboard
- 未登入用戶訪問 /dashboard → AuthGuard 導向 /login
- 註冊成功 → 不自動登入，導向 /login（讓用戶明確登入）
- Server 回傳 field-level validation errors → 對應顯示在表單欄位下方
- Network error → 顯示通用錯誤訊息

---

## 🔧 實作步驟

### Step 0: 後端小改 — Cookie Secure 可配置 ✅ 已完成

- `internal/config/config.go` — 新增 `CookieSecure bool` env var
- `internal/server/server.go` — Server struct 加 `cookieSecure bool`
- `internal/server/auth_handlers.go` — `setRefreshTokenCookie` / `clearRefreshTokenCookie` 改用 `s.cookieSecure`
- `.env.example` — 新增 `COOKIE_SECURE=true`

### Step 1: Vite 專案搭建

- 刪除 `web/.gitkeep`，初始化 Vite + React + TypeScript
- 安裝 dependencies：react-router, zustand, tailwindcss, @tailwindcss/vite, clsx
- 配置 `vite.config.ts`（proxy `/api` → `localhost:8080`，含 `ws: true`）
- 配置 TypeScript strict mode + ESLint flat config

### Step 2: Tailwind CSS v4 + 主題

- `index.css`：`@import "tailwindcss"` + `@theme { ... }` 使用 Pencil 設計系統色彩
- `index.html`：引入 Google Fonts（Playfair Display + Manrope）
- `lib/cn.ts`：clsx wrapper

### Step 3: API Client + Types

- `api/types.ts` — 鏡像 backend types.go 中 Auth 相關 types
- `api/client.ts` — fetch wrapper（auto-auth, 401 refresh interceptor, mutex）
- `api/auth.ts` — register / login / refresh / logout API calls
- `lib/constants.ts` — API paths

### Step 4: Zustand Auth Store

- `stores/auth-store.ts` — `{ accessToken, user, isAuthenticated, setAuth, clearAuth }`
- `hooks/use-auth.ts` — 包裝 store + API calls

### Step 5: 路由 + 佈局 + Guards

- `router.tsx` — createBrowserRouter 路由定義
- `auth-layout.tsx` — 置中卡片佈局
- `app-layout.tsx` — 頂部導航列 + Outlet
- `auth-guard.tsx` / `guest-guard.tsx` — 路由保護

### Step 6: UI 元件

- `button.tsx` — primary/secondary/ghost variants, loading state
- `input.tsx` — 暗色輸入框、金色 focus ring
- `form-field.tsx` — label + input + error message
- `loading-spinner.tsx`

### Step 7: Auth 頁面

- `login-page.tsx` — email + password 表單
- `register-page.tsx` — username + email + password 表單
- `dashboard-page.tsx` — Welcome, {username}
- `not-found-page.tsx` — 404 頁面

### Step 8: Makefile + .gitignore

- Makefile 新增 `web-install`, `web-dev`, `web-build`, `web-lint`, `web-test`
- .gitignore 新增 `web/node_modules/`, `web/dist/`

### Step 9: 單元測試

- `api/client.test.ts` — fetch wrapper 測試
- `stores/auth-store.test.ts` — store 狀態轉換
- `pages/login-page.test.tsx` — 表單驗證、submit、error 顯示
- `pages/register-page.test.tsx` — 表單驗證、submit、error 顯示
- `components/auth-guard.test.tsx` — 路由保護邏輯

使用 vitest + @testing-library/react，~25 test cases

---

## ✅ 驗收標準（Done When）

- [x] Vite 專案可正常 `npm run dev` 與 `npm run build`
- [x] Tailwind CSS v4 主題色彩與 Pencil 設計一致
- [x] 4 個 API 呼叫（register/login/refresh/logout）正常運作
- [x] Access token 僅存記憶體，refresh 靠 HttpOnly cookie
- [x] AuthGuard / GuestGuard 路由保護正常
- [x] Login / Register / Dashboard / 404 四個頁面正常渲染
- [x] 頁面刷新後可透過 refresh token 恢復登入狀態
- [x] 27 個前端測試通過
- [x] ESLint 無 error
- [x] Go tests 仍全部通過（`make test`）
- [x] Makefile 新增 5 個 web targets

---

## 🚫 禁止事項（Out of Scope）

- 不實作 Scenario / Session / Character 前端頁面（Phase 2）
- 不實作 WebSocket 連接（Phase 2+）
- 不使用 localStorage 存 token
- 不新增後端 API endpoint
- 不修改 DB schema / migration

---

## 📎 參考資料（References）

- UI 設計：`docs/designs/pencil-new.pen`（Login Page / Register Page / Dashboard Page）
- 後端 Auth API：`internal/server/auth_handlers.go`
- 後端 Types：`internal/server/types.go`
- JWT Claims：`internal/auth/token.go:14-17`（sub + username）
- Cookie 設定：`internal/server/auth_handlers.go:209-231`（Path: `/api/v1/auth`, SameSite: Strict）
- 驗證規則：`internal/server/auth_handlers.go:233-259`
