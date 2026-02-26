package auth

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("failed to connect to test DB: %v", err)
	}
	t.Cleanup(func() {
		// Clean up test data
		pool.Exec(ctx, "DELETE FROM refresh_tokens")
		pool.Exec(ctx, "DELETE FROM session_players")
		pool.Exec(ctx, "DELETE FROM characters")
		pool.Exec(ctx, "DELETE FROM game_events")
		pool.Exec(ctx, "DELETE FROM revealed_items")
		pool.Exec(ctx, "DELETE FROM revealed_npc_fields")
		pool.Exec(ctx, "DELETE FROM game_sessions")
		pool.Exec(ctx, "DELETE FROM scenarios")
		pool.Exec(ctx, "DELETE FROM users")
		pool.Close()
	})
	// Also clean before test to ensure clean state
	pool.Exec(ctx, "DELETE FROM refresh_tokens")
	pool.Exec(ctx, "DELETE FROM session_players")
	pool.Exec(ctx, "DELETE FROM characters")
	pool.Exec(ctx, "DELETE FROM game_events")
	pool.Exec(ctx, "DELETE FROM revealed_items")
	pool.Exec(ctx, "DELETE FROM revealed_npc_fields")
	pool.Exec(ctx, "DELETE FROM game_sessions")
	pool.Exec(ctx, "DELETE FROM scenarios")
	pool.Exec(ctx, "DELETE FROM users")
	return pool
}

func TestCreateUser(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	user, err := repo.CreateUser(ctx, "testuser", "test@example.com", "$2a$12$fakehash")
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if user.ID == "" {
		t.Error("CreateUser() returned empty ID")
	}
	if user.Username != "testuser" {
		t.Errorf("Username = %q, want %q", user.Username, "testuser")
	}
	if user.Email != "test@example.com" {
		t.Errorf("Email = %q, want %q", user.Email, "test@example.com")
	}
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	_, err := repo.CreateUser(ctx, "user1", "dup@example.com", "$2a$12$fakehash")
	if err != nil {
		t.Fatalf("first CreateUser() error = %v", err)
	}

	_, err = repo.CreateUser(ctx, "user2", "dup@example.com", "$2a$12$fakehash")
	if err == nil {
		t.Fatal("CreateUser() expected error for duplicate email, got nil")
	}
}

func TestCreateUser_DuplicateUsername(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	_, err := repo.CreateUser(ctx, "dupuser", "a@example.com", "$2a$12$fakehash")
	if err != nil {
		t.Fatalf("first CreateUser() error = %v", err)
	}

	_, err = repo.CreateUser(ctx, "dupuser", "b@example.com", "$2a$12$fakehash")
	if err == nil {
		t.Fatal("CreateUser() expected error for duplicate username, got nil")
	}
}

func TestGetUserByEmail(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	created, err := repo.CreateUser(ctx, "findme", "find@example.com", "$2a$12$fakehash")
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	found, err := repo.GetUserByEmail(ctx, "find@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail() error = %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("ID = %q, want %q", found.ID, created.ID)
	}
}

func TestGetUserByEmail_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	_, err := repo.GetUserByEmail(ctx, "nonexistent@example.com")
	if err == nil {
		t.Fatal("GetUserByEmail() expected error for nonexistent email, got nil")
	}
}

func TestRefreshTokenLifecycle(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	user, err := repo.CreateUser(ctx, "tokenuser", "token@example.com", "$2a$12$fakehash")
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	// Store a refresh token
	tokenHash := "abc123def456abc123def456abc123def456abc123def456abc123def456abcd"
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	err = repo.StoreRefreshToken(ctx, user.ID, tokenHash, expiresAt)
	if err != nil {
		t.Fatalf("StoreRefreshToken() error = %v", err)
	}

	// Retrieve it
	rt, err := repo.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		t.Fatalf("GetRefreshTokenByHash() error = %v", err)
	}
	if rt.UserID != user.ID {
		t.Errorf("UserID = %q, want %q", rt.UserID, user.ID)
	}
	if rt.Revoked {
		t.Error("token should not be revoked yet")
	}

	// Revoke it
	err = repo.RevokeRefreshToken(ctx, rt.ID)
	if err != nil {
		t.Fatalf("RevokeRefreshToken() error = %v", err)
	}

	// Verify revoked
	rt2, err := repo.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		t.Fatalf("GetRefreshTokenByHash() after revoke error = %v", err)
	}
	if !rt2.Revoked {
		t.Error("token should be revoked")
	}
}

func TestRevokeAllUserRefreshTokens(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	user, _ := repo.CreateUser(ctx, "revokeall", "revokeall@example.com", "$2a$12$fakehash")
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	hash1 := "1111111111111111111111111111111111111111111111111111111111111111"
	hash2 := "2222222222222222222222222222222222222222222222222222222222222222"
	_ = repo.StoreRefreshToken(ctx, user.ID, hash1, expiresAt)
	_ = repo.StoreRefreshToken(ctx, user.ID, hash2, expiresAt)

	err := repo.RevokeAllUserRefreshTokens(ctx, user.ID)
	if err != nil {
		t.Fatalf("RevokeAllUserRefreshTokens() error = %v", err)
	}

	rt1, _ := repo.GetRefreshTokenByHash(ctx, hash1)
	rt2, _ := repo.GetRefreshTokenByHash(ctx, hash2)
	if !rt1.Revoked || !rt2.Revoked {
		t.Error("all tokens should be revoked")
	}
}
