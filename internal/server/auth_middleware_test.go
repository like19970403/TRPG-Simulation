package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/like19970403/TRPG-Simulation/internal/auth"
)

func TestRequireAuth_ValidToken(t *testing.T) {
	cfg := testConfig()
	srv := New(cfg, nil, testLogger())

	token, err := auth.GenerateAccessToken("user-123", "player1", cfg.JWTSecret, 15*time.Minute)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	called := false
	handler := srv.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		called = true
		claims := UserClaimsFromContext(r.Context())
		if claims == nil {
			t.Fatal("claims should not be nil")
		}
		if claims.UserID != "user-123" {
			t.Errorf("UserID = %q, want %q", claims.UserID, "user-123")
		}
		if claims.Username != "player1" {
			t.Errorf("Username = %q, want %q", claims.Username, "player1")
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	handler(w, req)

	if !called {
		t.Error("handler should have been called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRequireAuth_MissingHeader(t *testing.T) {
	srv := New(testConfig(), nil, testLogger())

	handler := srv.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestRequireAuth_InvalidToken(t *testing.T) {
	srv := New(testConfig(), nil, testLogger())

	handler := srv.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestRequireAuth_ExpiredToken(t *testing.T) {
	cfg := testConfig()
	srv := New(cfg, nil, testLogger())

	token, _ := auth.GenerateAccessToken("user-123", "player1", cfg.JWTSecret, -1*time.Minute)

	handler := srv.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestRequireAuth_MalformedHeader(t *testing.T) {
	srv := New(testConfig(), nil, testLogger())

	handler := srv.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Authorization", "NotBearer some-token")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestUserClaimsFromContext_NoValue(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	claims := UserClaimsFromContext(req.Context())
	if claims != nil {
		t.Error("claims should be nil when not set")
	}
}
