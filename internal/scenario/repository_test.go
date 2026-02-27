package scenario

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Fatalf("connect to database: %v", err)
	}

	t.Cleanup(func() {
		// Clean up in dependency order
		pool.Exec(context.Background(), "DELETE FROM scenarios")
		pool.Exec(context.Background(), "DELETE FROM refresh_tokens")
		pool.Exec(context.Background(), "DELETE FROM users")
		pool.Close()
	})

	return pool
}

func createTestUser(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO users (username, email, password_hash)
		 VALUES ('testgm', 'testgm@test.com', '$2a$04$fakehash000000000000000000000000000000000000000000')
		 ON CONFLICT (email) DO UPDATE SET email = EXCLUDED.email
		 RETURNING id`,
	).Scan(&id)
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}
	return id
}

func createTestUser2(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO users (username, email, password_hash)
		 VALUES ('othergm', 'othergm@test.com', '$2a$04$fakehash000000000000000000000000000000000000000000')
		 ON CONFLICT (email) DO UPDATE SET email = EXCLUDED.email
		 RETURNING id`,
	).Scan(&id)
	if err != nil {
		t.Fatalf("create test user 2: %v", err)
	}
	return id
}

func TestCreate(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	userID := createTestUser(t, pool)

	content := json.RawMessage(`{"start_scene":"s1","scenes":[{"id":"s1","name":"Start"}]}`)
	sc, err := repo.Create(context.Background(), userID, "Test Quest", "A test scenario", content)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if sc.ID == "" {
		t.Error("ID should not be empty")
	}
	if sc.AuthorID != userID {
		t.Errorf("AuthorID = %q, want %q", sc.AuthorID, userID)
	}
	if sc.Title != "Test Quest" {
		t.Errorf("Title = %q, want %q", sc.Title, "Test Quest")
	}
	if sc.Description != "A test scenario" {
		t.Errorf("Description = %q, want %q", sc.Description, "A test scenario")
	}
	if sc.Version != 1 {
		t.Errorf("Version = %d, want 1", sc.Version)
	}
	if sc.Status != "draft" {
		t.Errorf("Status = %q, want %q", sc.Status, "draft")
	}
	if sc.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

func TestGetByID(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	userID := createTestUser(t, pool)

	content := json.RawMessage(`{"test":true}`)
	created, err := repo.Create(context.Background(), userID, "Get Test", "", content)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	sc, err := repo.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if sc.ID != created.ID {
		t.Errorf("ID = %q, want %q", sc.ID, created.ID)
	}
	if sc.Title != "Get Test" {
		t.Errorf("Title = %q, want %q", sc.Title, "Get Test")
	}
}

func TestGetByID_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)

	_, err := repo.GetByID(context.Background(), "00000000-0000-0000-0000-000000000000")
	if err == nil {
		t.Fatal("GetByID() expected error for nonexistent ID")
	}
}

func TestListByAuthor_Empty(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	userID := createTestUser(t, pool)

	scenarios, total, err := repo.ListByAuthor(context.Background(), userID, 20, 0)
	if err != nil {
		t.Fatalf("ListByAuthor() error = %v", err)
	}
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
	if len(scenarios) != 0 {
		t.Errorf("len(scenarios) = %d, want 0", len(scenarios))
	}
}

func TestListByAuthor_Pagination(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	userID := createTestUser(t, pool)

	content := json.RawMessage(`{}`)
	for i := 0; i < 5; i++ {
		_, err := repo.Create(context.Background(), userID, "Scenario", "", content)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	scenarios, total, err := repo.ListByAuthor(context.Background(), userID, 2, 0)
	if err != nil {
		t.Fatalf("ListByAuthor() error = %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(scenarios) != 2 {
		t.Errorf("len(scenarios) = %d, want 2", len(scenarios))
	}

	// Offset
	scenarios2, total2, err := repo.ListByAuthor(context.Background(), userID, 2, 3)
	if err != nil {
		t.Fatalf("ListByAuthor() error = %v", err)
	}
	if total2 != 5 {
		t.Errorf("total = %d, want 5", total2)
	}
	if len(scenarios2) != 2 {
		t.Errorf("len(scenarios) = %d, want 2", len(scenarios2))
	}
}

func TestListByAuthor_Isolation(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	user1 := createTestUser(t, pool)
	user2 := createTestUser2(t, pool)

	content := json.RawMessage(`{}`)
	repo.Create(context.Background(), user1, "User1 Scenario", "", content)
	repo.Create(context.Background(), user2, "User2 Scenario", "", content)

	scenarios, total, err := repo.ListByAuthor(context.Background(), user1, 20, 0)
	if err != nil {
		t.Fatalf("ListByAuthor() error = %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(scenarios) != 1 {
		t.Errorf("len(scenarios) = %d, want 1", len(scenarios))
	}
	if scenarios[0].Title != "User1 Scenario" {
		t.Errorf("Title = %q, want %q", scenarios[0].Title, "User1 Scenario")
	}
}

func TestUpdate(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	userID := createTestUser(t, pool)

	content := json.RawMessage(`{"old":true}`)
	created, err := repo.Create(context.Background(), userID, "Original", "desc", content)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	newContent := json.RawMessage(`{"new":true}`)
	updated, err := repo.Update(context.Background(), created.ID, "Updated Title", "new desc", newContent)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if updated.Title != "Updated Title" {
		t.Errorf("Title = %q, want %q", updated.Title, "Updated Title")
	}
	if updated.Description != "new desc" {
		t.Errorf("Description = %q, want %q", updated.Description, "new desc")
	}
	if updated.UpdatedAt.Equal(created.UpdatedAt) || updated.UpdatedAt.Before(created.UpdatedAt) {
		t.Error("UpdatedAt should be after creation time")
	}
}

func TestUpdate_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)

	content := json.RawMessage(`{}`)
	_, err := repo.Update(context.Background(), "00000000-0000-0000-0000-000000000000", "Title", "", content)
	if err == nil {
		t.Fatal("Update() expected error for nonexistent ID")
	}
}

func TestDelete(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	userID := createTestUser(t, pool)

	content := json.RawMessage(`{}`)
	created, _ := repo.Create(context.Background(), userID, "To Delete", "", content)

	err := repo.Delete(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err = repo.GetByID(context.Background(), created.ID)
	if err == nil {
		t.Error("GetByID() should fail after delete")
	}
}

func TestDelete_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)

	err := repo.Delete(context.Background(), "00000000-0000-0000-0000-000000000000")
	if err == nil {
		t.Fatal("Delete() expected error for nonexistent ID")
	}
}

func TestUpdateStatus_DraftToPublished(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	userID := createTestUser(t, pool)

	content := json.RawMessage(`{}`)
	created, _ := repo.Create(context.Background(), userID, "To Publish", "", content)

	updated, err := repo.UpdateStatus(context.Background(), created.ID, "published")
	if err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}
	if updated.Status != "published" {
		t.Errorf("Status = %q, want %q", updated.Status, "published")
	}
}

func TestUpdateStatus_PublishedToArchived(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	userID := createTestUser(t, pool)

	content := json.RawMessage(`{}`)
	created, _ := repo.Create(context.Background(), userID, "To Archive", "", content)
	repo.UpdateStatus(context.Background(), created.ID, "published")

	updated, err := repo.UpdateStatus(context.Background(), created.ID, "archived")
	if err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}
	if updated.Status != "archived" {
		t.Errorf("Status = %q, want %q", updated.Status, "archived")
	}
}

func TestUpdateStatus_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)

	_, err := repo.UpdateStatus(context.Background(), "00000000-0000-0000-0000-000000000000", "published")
	if err == nil {
		t.Fatal("UpdateStatus() expected error for nonexistent ID")
	}
}
