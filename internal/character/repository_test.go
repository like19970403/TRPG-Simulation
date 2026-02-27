package character

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
		// Clean up in dependency order.
		pool.Exec(context.Background(), "DELETE FROM session_players")
		pool.Exec(context.Background(), "DELETE FROM game_events")
		pool.Exec(context.Background(), "DELETE FROM game_sessions")
		pool.Exec(context.Background(), "DELETE FROM characters")
		pool.Exec(context.Background(), "DELETE FROM scenarios")
		pool.Exec(context.Background(), "DELETE FROM refresh_tokens")
		pool.Exec(context.Background(), "DELETE FROM users")
		pool.Close()
	})

	return pool
}

func createTestUser(t *testing.T, pool *pgxpool.Pool, email string) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO users (username, email, password_hash)
		 VALUES ($1, $2, '$2a$04$fakehash000000000000000000000000000000000000000000')
		 ON CONFLICT (email) DO UPDATE SET email = EXCLUDED.email
		 RETURNING id`,
		email, email,
	).Scan(&id)
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}
	return id
}

func TestCreate(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	userID := createTestUser(t, pool, "char-test@test.com")

	c, err := repo.Create(context.Background(), userID, "Aragorn",
		json.RawMessage(`{"str":16,"dex":14}`),
		json.RawMessage(`["sword","shield"]`),
		"Ranger of the North",
	)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if c.ID == "" {
		t.Error("ID should not be empty")
	}
	if c.UserID != userID {
		t.Errorf("UserID = %q, want %q", c.UserID, userID)
	}
	if c.Name != "Aragorn" {
		t.Errorf("Name = %q, want %q", c.Name, "Aragorn")
	}
	if c.Notes != "Ranger of the North" {
		t.Errorf("Notes = %q, want %q", c.Notes, "Ranger of the North")
	}
}

func TestCreate_DefaultValues(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	userID := createTestUser(t, pool, "char-default@test.com")

	c, err := repo.Create(context.Background(), userID, "Legolas",
		json.RawMessage(`{}`),
		json.RawMessage(`[]`),
		"",
	)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if string(c.Attributes) != "{}" {
		t.Errorf("Attributes = %s, want {}", string(c.Attributes))
	}
	if string(c.Inventory) != "[]" {
		t.Errorf("Inventory = %s, want []", string(c.Inventory))
	}
	if c.Notes != "" {
		t.Errorf("Notes = %q, want empty", c.Notes)
	}
}

func TestGetByID(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	userID := createTestUser(t, pool, "char-get@test.com")

	created, err := repo.Create(context.Background(), userID, "Gandalf",
		json.RawMessage(`{"wis":20}`), json.RawMessage(`["staff"]`), "A wizard")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := repo.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.Name != "Gandalf" {
		t.Errorf("Name = %q, want %q", got.Name, "Gandalf")
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

func TestListByUser_Empty(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	userID := createTestUser(t, pool, "char-empty@test.com")

	chars, total, err := repo.ListByUser(context.Background(), userID, 20, 0)
	if err != nil {
		t.Fatalf("ListByUser() error = %v", err)
	}
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
	if len(chars) != 0 {
		t.Errorf("len = %d, want 0", len(chars))
	}
}

func TestListByUser_Pagination(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	userID := createTestUser(t, pool, "char-page@test.com")

	for i := 0; i < 5; i++ {
		_, err := repo.Create(context.Background(), userID, "char-"+string(rune('A'+i)),
			json.RawMessage(`{}`), json.RawMessage(`[]`), "")
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	chars, total, err := repo.ListByUser(context.Background(), userID, 2, 0)
	if err != nil {
		t.Fatalf("ListByUser() error = %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(chars) != 2 {
		t.Errorf("len = %d, want 2", len(chars))
	}
}

func TestListByUser_Isolation(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	user1 := createTestUser(t, pool, "char-iso1@test.com")
	user2 := createTestUser(t, pool, "char-iso2@test.com")

	repo.Create(context.Background(), user1, "User1-Char", json.RawMessage(`{}`), json.RawMessage(`[]`), "")
	repo.Create(context.Background(), user2, "User2-Char", json.RawMessage(`{}`), json.RawMessage(`[]`), "")

	chars, total, err := repo.ListByUser(context.Background(), user1, 20, 0)
	if err != nil {
		t.Fatalf("ListByUser() error = %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(chars) != 1 {
		t.Errorf("len = %d, want 1", len(chars))
	}
	if chars[0].Name != "User1-Char" {
		t.Errorf("Name = %q, want %q", chars[0].Name, "User1-Char")
	}
}

func TestUpdate(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	userID := createTestUser(t, pool, "char-update@test.com")

	created, err := repo.Create(context.Background(), userID, "OldName",
		json.RawMessage(`{"str":10}`), json.RawMessage(`[]`), "Old notes")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	updated, err := repo.Update(context.Background(), created.ID, "NewName",
		json.RawMessage(`{"str":16,"dex":14}`), json.RawMessage(`["sword"]`), "New notes")
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Name != "NewName" {
		t.Errorf("Name = %q, want %q", updated.Name, "NewName")
	}
	if updated.Notes != "New notes" {
		t.Errorf("Notes = %q, want %q", updated.Notes, "New notes")
	}
	if !updated.UpdatedAt.After(created.UpdatedAt) {
		t.Error("UpdatedAt should be after creation time")
	}
}

func TestUpdate_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)

	_, err := repo.Update(context.Background(), "00000000-0000-0000-0000-000000000000",
		"Name", json.RawMessage(`{}`), json.RawMessage(`[]`), "")
	if err == nil {
		t.Fatal("Update() expected error for nonexistent ID")
	}
}

func TestDelete(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	userID := createTestUser(t, pool, "char-delete@test.com")

	created, err := repo.Create(context.Background(), userID, "ToDelete",
		json.RawMessage(`{}`), json.RawMessage(`[]`), "")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	err = repo.Delete(context.Background(), created.ID)
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

func TestIsLinkedToSession_False(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	userID := createTestUser(t, pool, "char-link-f@test.com")

	created, err := repo.Create(context.Background(), userID, "Unlinked",
		json.RawMessage(`{}`), json.RawMessage(`[]`), "")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	linked, err := repo.IsLinkedToSession(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("IsLinkedToSession() error = %v", err)
	}
	if linked {
		t.Error("expected false for unlinked character")
	}
}

func TestIsLinkedToSession_True(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	userID := createTestUser(t, pool, "char-link-t@test.com")

	// Create character.
	created, err := repo.Create(context.Background(), userID, "Linked",
		json.RawMessage(`{}`), json.RawMessage(`[]`), "")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Create scenario + session to link the character.
	var scenarioID string
	err = pool.QueryRow(context.Background(),
		`INSERT INTO scenarios (author_id, title, description, content)
		 VALUES ($1, 'Test', 'Test', '{}')
		 RETURNING id`, userID,
	).Scan(&scenarioID)
	if err != nil {
		t.Fatalf("create scenario: %v", err)
	}

	var sessionID string
	err = pool.QueryRow(context.Background(),
		`INSERT INTO game_sessions (scenario_id, gm_id, invite_code)
		 VALUES ($1, $2, 'TESTCODE')
		 RETURNING id`, scenarioID, userID,
	).Scan(&sessionID)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	_, err = pool.Exec(context.Background(),
		`INSERT INTO session_players (session_id, user_id, character_id)
		 VALUES ($1, $2, $3)`, sessionID, userID, created.ID,
	)
	if err != nil {
		t.Fatalf("create session player: %v", err)
	}

	linked, err := repo.IsLinkedToSession(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("IsLinkedToSession() error = %v", err)
	}
	if !linked {
		t.Error("expected true for linked character")
	}
}
