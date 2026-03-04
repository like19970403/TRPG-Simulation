// Package integration_test provides Go integration tests that require a real database.
// These tests are skipped when the E2E_DATABASE_URL environment variable is not set.
// Run with: E2E_DATABASE_URL="postgres://..." go test ./internal/integration_test/... -v
package integration_test

import (
	"context"
	"fmt"
	"log/slog"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/like19970403/TRPG-Simulation/internal/config"
	"github.com/like19970403/TRPG-Simulation/internal/server"
)

// testDBURL returns the E2E database URL or skips the test.
func testDBURL(t *testing.T) string {
	t.Helper()
	url := os.Getenv("E2E_DATABASE_URL")
	if url == "" {
		t.Skip("E2E_DATABASE_URL not set — skipping integration test")
	}
	return url
}

// setupPool creates a pgxpool connected to the test database.
func setupPool(t *testing.T, dbURL string) *pgxpool.Pool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		t.Fatalf("parse pool config: %v", err)
	}
	cfg.MaxConns = 5

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("connect pool: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping db: %v", err)
	}
	return pool
}

// setupServer creates a test HTTP server backed by a real database.
func setupServer(t *testing.T, pool *pgxpool.Pool) *httptest.Server {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := &config.Config{
		Port:               0,
		JWTSecret:          "integration-test-secret-min-32-characters!",
		JWTAccessTokenTTL:  900,
		JWTRefreshTokenTTL: 86400,
		BcryptCost:         4, // fast for tests
		UploadDir:          t.TempDir(),
	}
	s := server.New(cfg, pool, logger)
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	return ts
}

// uniqueEmail generates a unique test email.
func uniqueEmail(prefix string) string {
	return fmt.Sprintf("%s_%d@test.local", prefix, time.Now().UnixNano())
}
