package server

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/like19970403/TRPG-Simulation/internal/config"
)

func testLogger() *slog.Logger {
	return slog.Default()
}

func testConfig() *config.Config {
	return &config.Config{
		Port:               8080,
		DatabaseURL:        "postgres://test:test@localhost/test",
		LogLevel:           "info",
		JWTSecret:          "test-secret-key-at-least-32-chars!!",
		JWTAccessTokenTTL:  900,
		JWTRefreshTokenTTL: 604800,
		BcryptCost:         4, // low cost for fast tests
		CookieSecure:       true,
	}
}

func TestNew(t *testing.T) {
	logger := slog.Default()
	srv := New(testConfig(), nil, logger)

	if srv == nil {
		t.Fatal("New() returned nil")
	}
}

func TestRouting_HealthEndpoint(t *testing.T) {
	logger := slog.Default()
	srv := New(testConfig(), nil, logger)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Result().StatusCode == http.StatusNotFound {
		t.Error("GET /api/health should not return 404")
	}
}

func TestRouting_NotFound(t *testing.T) {
	logger := slog.Default()
	srv := New(testConfig(), nil, logger)

	req := httptest.NewRequest(http.MethodGet, "/api/nonexistent", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Result().StatusCode, http.StatusNotFound)
	}
}
