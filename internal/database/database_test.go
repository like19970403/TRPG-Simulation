package database

import (
	"context"
	"os"
	"testing"
)

func TestNewPool_InvalidURL(t *testing.T) {
	ctx := context.Background()
	_, err := NewPool(ctx, "not-a-valid-url")
	if err == nil {
		t.Fatal("NewPool() expected error for invalid URL, got nil")
	}
}

func TestNewPool_UnreachableHost(t *testing.T) {
	ctx := context.Background()
	_, err := NewPool(ctx, "postgres://user:pass@127.0.0.1:59999/nodb?sslmode=disable&connect_timeout=1")
	if err == nil {
		t.Fatal("NewPool() expected error for unreachable host, got nil")
	}
}

func TestNewPool_Integration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	ctx := context.Background()
	pool, err := NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("NewPool() error = %v", err)
	}
	defer pool.Close()

	if err := HealthCheck(ctx, pool); err != nil {
		t.Fatalf("HealthCheck() error = %v", err)
	}
}
