# API Changelog

所有 API 相關變更紀錄。格式依循 [Keep a Changelog](https://keepachangelog.com/)。

## [v0.2.0] - 2026-02-27

### Added
- `POST /api/v1/users` — 用戶註冊（username, email, password）
- `POST /api/v1/auth/login` — 用戶登入（回傳 JWT access token + HttpOnly refresh token cookie）
- `POST /api/v1/auth/refresh` — 刷新 access token（refresh token 旋轉策略）
- `POST /api/v1/auth/logout` — 用戶登出（撤銷 refresh token）

## [v0.1.0] - 2026-02-27

### Added
- `GET /api/health` — 服務健康檢查（含 PostgreSQL 連線狀態）
