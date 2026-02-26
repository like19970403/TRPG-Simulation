package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/like19970403/TRPG-Simulation/internal/auth"
)

// mockAuthRepo implements AuthRepository for unit tests.
type mockAuthRepo struct {
	createUserFn              func(ctx context.Context, username, email, passwordHash string) (*auth.User, error)
	getUserByEmailFn          func(ctx context.Context, email string) (*auth.User, error)
	getUserByIDFn             func(ctx context.Context, id string) (*auth.User, error)
	storeRefreshTokenFn       func(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error
	getRefreshTokenByHashFn   func(ctx context.Context, tokenHash string) (*auth.RefreshToken, error)
	revokeRefreshTokenFn      func(ctx context.Context, tokenID string) error
	revokeAllUserRefreshFn    func(ctx context.Context, userID string) error
}

func (m *mockAuthRepo) CreateUser(ctx context.Context, username, email, passwordHash string) (*auth.User, error) {
	return m.createUserFn(ctx, username, email, passwordHash)
}

func (m *mockAuthRepo) GetUserByEmail(ctx context.Context, email string) (*auth.User, error) {
	return m.getUserByEmailFn(ctx, email)
}

func (m *mockAuthRepo) GetUserByID(ctx context.Context, id string) (*auth.User, error) {
	return m.getUserByIDFn(ctx, id)
}

func (m *mockAuthRepo) StoreRefreshToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	return m.storeRefreshTokenFn(ctx, userID, tokenHash, expiresAt)
}

func (m *mockAuthRepo) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*auth.RefreshToken, error) {
	return m.getRefreshTokenByHashFn(ctx, tokenHash)
}

func (m *mockAuthRepo) RevokeRefreshToken(ctx context.Context, tokenID string) error {
	return m.revokeRefreshTokenFn(ctx, tokenID)
}

func (m *mockAuthRepo) RevokeAllUserRefreshTokens(ctx context.Context, userID string) error {
	return m.revokeAllUserRefreshFn(ctx, userID)
}

func newTestServer(repo AuthRepository) *Server {
	cfg := testConfig()
	srv := New(cfg, nil, testLogger())
	srv.authRepo = repo
	return srv
}

// --- Register tests ---

func TestHandleRegister_Success(t *testing.T) {
	now := time.Now().UTC()
	repo := &mockAuthRepo{
		createUserFn: func(_ context.Context, username, email, _ string) (*auth.User, error) {
			return &auth.User{
				ID: "uuid-1", Username: username, Email: email,
				CreatedAt: now, UpdatedAt: now,
			}, nil
		},
	}
	srv := newTestServer(repo)

	body := `{"username":"player1","email":"p1@test.com","password":"SecurePass1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleRegister(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
	}

	var resp RegisterResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.ID != "uuid-1" {
		t.Errorf("ID = %q, want %q", resp.ID, "uuid-1")
	}
	if resp.Username != "player1" {
		t.Errorf("Username = %q, want %q", resp.Username, "player1")
	}
	if resp.Email != "p1@test.com" {
		t.Errorf("Email = %q, want %q", resp.Email, "p1@test.com")
	}
}

func TestHandleRegister_ValidationError(t *testing.T) {
	srv := newTestServer(&mockAuthRepo{})

	tests := []struct {
		name string
		body string
	}{
		{"short username", `{"username":"ab","email":"a@b.com","password":"12345678"}`},
		{"invalid username chars", `{"username":"a b!","email":"a@b.com","password":"12345678"}`},
		{"invalid email", `{"username":"player1","email":"not-email","password":"12345678"}`},
		{"short password", `{"username":"player1","email":"a@b.com","password":"short"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(tt.body))
			w := httptest.NewRecorder()

			srv.handleRegister(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}

			var errResp ErrorResponse
			json.NewDecoder(w.Body).Decode(&errResp)
			if errResp.Error != "VALIDATION_ERROR" {
				t.Errorf("error code = %q, want %q", errResp.Error, "VALIDATION_ERROR")
			}
		})
	}
}

func TestHandleRegister_DuplicateConflict(t *testing.T) {
	repo := &mockAuthRepo{
		createUserFn: func(_ context.Context, _, _, _ string) (*auth.User, error) {
			return nil, errors.New("duplicate key value violates unique constraint")
		},
	}
	srv := newTestServer(repo)

	body := `{"username":"player1","email":"p1@test.com","password":"SecurePass1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(body))
	w := httptest.NewRecorder()

	srv.handleRegister(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestHandleRegister_InvalidJSON(t *testing.T) {
	srv := newTestServer(&mockAuthRepo{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader("{bad"))
	w := httptest.NewRecorder()

	srv.handleRegister(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- Login tests ---

func TestHandleLogin_Success(t *testing.T) {
	hash, _ := auth.HashPassword("SecurePass1", 4)
	repo := &mockAuthRepo{
		getUserByEmailFn: func(_ context.Context, _ string) (*auth.User, error) {
			return &auth.User{
				ID: "uuid-1", Username: "player1", Email: "p1@test.com",
				PasswordHash: hash,
			}, nil
		},
		storeRefreshTokenFn: func(_ context.Context, _, _ string, _ time.Time) error {
			return nil
		},
	}
	srv := newTestServer(repo)

	body := `{"email":"p1@test.com","password":"SecurePass1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	w := httptest.NewRecorder()

	srv.handleLogin(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp TokenResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("access token should not be empty")
	}
	if resp.TokenType != "Bearer" {
		t.Errorf("token type = %q, want %q", resp.TokenType, "Bearer")
	}
	if resp.ExpiresIn != int(srv.accessTTL.Seconds()) {
		t.Errorf("expires_in = %d, want %d", resp.ExpiresIn, int(srv.accessTTL.Seconds()))
	}

	// Check refresh token cookie
	cookies := w.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "refresh_token" {
			found = true
			if !c.HttpOnly {
				t.Error("refresh_token cookie should be HttpOnly")
			}
			if !c.Secure {
				t.Error("refresh_token cookie should be Secure")
			}
		}
	}
	if !found {
		t.Error("refresh_token cookie should be set")
	}
}

func TestHandleLogin_InvalidCredentials(t *testing.T) {
	hash, _ := auth.HashPassword("CorrectPassword", 4)
	repo := &mockAuthRepo{
		getUserByEmailFn: func(_ context.Context, _ string) (*auth.User, error) {
			return &auth.User{PasswordHash: hash}, nil
		},
	}
	srv := newTestServer(repo)

	body := `{"email":"p1@test.com","password":"WrongPassword"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	w := httptest.NewRecorder()

	srv.handleLogin(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleLogin_UserNotFound(t *testing.T) {
	repo := &mockAuthRepo{
		getUserByEmailFn: func(_ context.Context, _ string) (*auth.User, error) {
			return nil, errors.New("auth: user not found")
		},
	}
	srv := newTestServer(repo)

	body := `{"email":"nobody@test.com","password":"SomePass123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	w := httptest.NewRecorder()

	srv.handleLogin(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleLogin_ValidationError(t *testing.T) {
	srv := newTestServer(&mockAuthRepo{})

	body := `{"email":"","password":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	w := httptest.NewRecorder()

	srv.handleLogin(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- Refresh tests ---

func TestHandleRefresh_Success(t *testing.T) {
	raw, hash, _ := auth.GenerateRefreshToken()
	repo := &mockAuthRepo{
		getRefreshTokenByHashFn: func(_ context.Context, _ string) (*auth.RefreshToken, error) {
			return &auth.RefreshToken{
				ID: "rt-1", UserID: "uuid-1", TokenHash: hash,
				ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
			}, nil
		},
		revokeRefreshTokenFn: func(_ context.Context, _ string) error {
			return nil
		},
		getUserByIDFn: func(_ context.Context, _ string) (*auth.User, error) {
			return &auth.User{ID: "uuid-1", Username: "player1"}, nil
		},
		storeRefreshTokenFn: func(_ context.Context, _, _ string, _ time.Time) error {
			return nil
		},
	}
	srv := newTestServer(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: raw})
	w := httptest.NewRecorder()

	srv.handleRefresh(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp TokenResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.AccessToken == "" {
		t.Error("access token should not be empty")
	}
}

func TestHandleRefresh_MissingCookie(t *testing.T) {
	srv := newTestServer(&mockAuthRepo{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	w := httptest.NewRecorder()

	srv.handleRefresh(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleRefresh_InvalidToken(t *testing.T) {
	repo := &mockAuthRepo{
		getRefreshTokenByHashFn: func(_ context.Context, _ string) (*auth.RefreshToken, error) {
			return nil, errors.New("auth: refresh token not found")
		},
	}
	srv := newTestServer(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "invalid-token"})
	w := httptest.NewRecorder()

	srv.handleRefresh(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleRefresh_ExpiredToken(t *testing.T) {
	repo := &mockAuthRepo{
		getRefreshTokenByHashFn: func(_ context.Context, _ string) (*auth.RefreshToken, error) {
			return &auth.RefreshToken{
				ID: "rt-1", UserID: "uuid-1",
				ExpiresAt: time.Now().Add(-1 * time.Hour),
			}, nil
		},
	}
	srv := newTestServer(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "some-token"})
	w := httptest.NewRecorder()

	srv.handleRefresh(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleRefresh_RevokedToken_RevokesAll(t *testing.T) {
	revokeAllCalled := false
	repo := &mockAuthRepo{
		getRefreshTokenByHashFn: func(_ context.Context, _ string) (*auth.RefreshToken, error) {
			return &auth.RefreshToken{
				ID: "rt-1", UserID: "uuid-1", Revoked: true,
				ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
			}, nil
		},
		revokeAllUserRefreshFn: func(_ context.Context, userID string) error {
			revokeAllCalled = true
			if userID != "uuid-1" {
				t.Errorf("userID = %q, want %q", userID, "uuid-1")
			}
			return nil
		},
	}
	srv := newTestServer(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "stolen-token"})
	w := httptest.NewRecorder()

	srv.handleRefresh(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
	if !revokeAllCalled {
		t.Error("RevokeAllUserRefreshTokens should have been called")
	}

	// Cookie should be cleared
	cookies := w.Result().Cookies()
	for _, c := range cookies {
		if c.Name == "refresh_token" && c.MaxAge != -1 {
			t.Error("refresh_token cookie should be cleared (MaxAge = -1)")
		}
	}
}

// --- Logout tests ---

func TestHandleLogout_WithCookie(t *testing.T) {
	raw, _, _ := auth.GenerateRefreshToken()
	revokeCalled := false
	repo := &mockAuthRepo{
		getRefreshTokenByHashFn: func(_ context.Context, _ string) (*auth.RefreshToken, error) {
			return &auth.RefreshToken{ID: "rt-1"}, nil
		},
		revokeRefreshTokenFn: func(_ context.Context, tokenID string) error {
			revokeCalled = true
			if tokenID != "rt-1" {
				t.Errorf("tokenID = %q, want %q", tokenID, "rt-1")
			}
			return nil
		},
	}
	srv := newTestServer(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: raw})
	w := httptest.NewRecorder()

	srv.handleLogout(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
	if !revokeCalled {
		t.Error("RevokeRefreshToken should have been called")
	}
}

func TestHandleLogout_NoCookie(t *testing.T) {
	srv := newTestServer(&mockAuthRepo{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	w := httptest.NewRecorder()

	srv.handleLogout(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

// --- Validation helper tests ---

func TestValidateRegister(t *testing.T) {
	valid := RegisterRequest{Username: "player1", Email: "p1@test.com", Password: "12345678"}
	if errs := validateRegister(valid); len(errs) > 0 {
		t.Errorf("valid request should have no errors, got %v", errs)
	}

	// Password at bcrypt max (72 bytes)
	long := RegisterRequest{Username: "player1", Email: "p1@test.com", Password: strings.Repeat("a", 72)}
	if errs := validateRegister(long); len(errs) > 0 {
		t.Errorf("72-char password should be valid, got %v", errs)
	}

	// Password over bcrypt max
	over := RegisterRequest{Username: "player1", Email: "p1@test.com", Password: strings.Repeat("a", 73)}
	if errs := validateRegister(over); len(errs) == 0 {
		t.Error("73-char password should be invalid")
	}
}

func TestValidateLogin(t *testing.T) {
	valid := LoginRequest{Email: "p1@test.com", Password: "anything"}
	if errs := validateLogin(valid); len(errs) > 0 {
		t.Errorf("valid request should have no errors, got %v", errs)
	}

	empty := LoginRequest{Email: "", Password: ""}
	errs := validateLogin(empty)
	if len(errs) != 2 {
		t.Errorf("empty request should have 2 errors, got %d", len(errs))
	}
}

// --- Helper tests ---

func TestWriteJSON(t *testing.T) {
	srv := newTestServer(&mockAuthRepo{})

	w := httptest.NewRecorder()
	srv.writeJSON(w, http.StatusOK, map[string]string{"key": "value"})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}
}

func TestSetRefreshTokenCookie(t *testing.T) {
	srv := newTestServer(&mockAuthRepo{})

	w := httptest.NewRecorder()
	srv.setRefreshTokenCookie(w, "test-token", 7*24*time.Hour)

	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	c := cookies[0]
	if c.Name != "refresh_token" {
		t.Errorf("cookie name = %q, want %q", c.Name, "refresh_token")
	}
	if c.Path != "/api/v1/auth" {
		t.Errorf("cookie path = %q, want %q", c.Path, "/api/v1/auth")
	}
	if !c.HttpOnly {
		t.Error("cookie should be HttpOnly")
	}
	if !c.Secure {
		t.Error("cookie should be Secure")
	}
	if c.SameSite != http.SameSiteStrictMode {
		t.Errorf("SameSite = %d, want %d", c.SameSite, http.SameSiteStrictMode)
	}
}

func TestClearRefreshTokenCookie(t *testing.T) {
	srv := newTestServer(&mockAuthRepo{})

	w := httptest.NewRecorder()
	srv.clearRefreshTokenCookie(w)

	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	if cookies[0].MaxAge != -1 {
		t.Errorf("MaxAge = %d, want -1", cookies[0].MaxAge)
	}
}
