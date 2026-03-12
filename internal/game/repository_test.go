package game

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/like19970403/TRPG-Simulation/internal/apperror"
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
		ctx := context.Background()
		pool.Exec(ctx, "DELETE FROM session_players")
		pool.Exec(ctx, "DELETE FROM game_events")
		pool.Exec(ctx, "DELETE FROM game_sessions")
		pool.Exec(ctx, "DELETE FROM scenarios")
		pool.Exec(ctx, "DELETE FROM refresh_tokens")
		pool.Exec(ctx, "DELETE FROM users")
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
		 VALUES ('player1', 'player1@test.com', '$2a$04$fakehash000000000000000000000000000000000000000000')
		 ON CONFLICT (email) DO UPDATE SET email = EXCLUDED.email
		 RETURNING id`,
	).Scan(&id)
	if err != nil {
		t.Fatalf("create test user 2: %v", err)
	}
	return id
}

func createPublishedScenario(t *testing.T, pool *pgxpool.Pool, authorID string) string {
	t.Helper()
	var id string
	content := json.RawMessage(`{"start_scene":"s1","scenes":[{"id":"s1","name":"Start"}]}`)
	err := pool.QueryRow(context.Background(),
		`INSERT INTO scenarios (author_id, title, description, content, status)
		 VALUES ($1, 'Test Scenario', 'A test', $2, 'published')
		 RETURNING id`,
		authorID, content,
	).Scan(&id)
	if err != nil {
		t.Fatalf("create published scenario: %v", err)
	}
	return id
}

// --- GenerateInviteCode ---

func TestGenerateInviteCode(t *testing.T) {
	code, err := GenerateInviteCode()
	if err != nil {
		t.Fatalf("GenerateInviteCode error: %v", err)
	}
	if len(code) != 6 {
		t.Errorf("code length = %d, want 6", len(code))
	}
	// Check no confusable characters
	for _, c := range code {
		if c == '0' || c == 'O' || c == '1' || c == 'I' {
			t.Errorf("code contains confusable character: %c", c)
		}
	}
}

// --- Create ---

func TestCreate(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	userID := createTestUser(t, pool)
	scenarioID := createPublishedScenario(t, pool, userID)

	gs, err := repo.Create(context.Background(), scenarioID, userID)
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if gs.ID == "" {
		t.Error("ID should not be empty")
	}
	if gs.ScenarioID != scenarioID {
		t.Errorf("ScenarioID = %q, want %q", gs.ScenarioID, scenarioID)
	}
	if gs.GMID != userID {
		t.Errorf("GMID = %q, want %q", gs.GMID, userID)
	}
	if gs.Status != "lobby" {
		t.Errorf("Status = %q, want %q", gs.Status, "lobby")
	}
	if len(gs.InviteCode) != 6 {
		t.Errorf("InviteCode length = %d, want 6", len(gs.InviteCode))
	}
	if gs.StartedAt != nil {
		t.Error("StartedAt should be nil")
	}
	if gs.EndedAt != nil {
		t.Error("EndedAt should be nil")
	}
}

// --- GetByID ---

func TestGetByID(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	userID := createTestUser(t, pool)
	scenarioID := createPublishedScenario(t, pool, userID)

	created, _ := repo.Create(context.Background(), scenarioID, userID)

	gs, err := repo.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetByID error: %v", err)
	}
	if gs.ID != created.ID {
		t.Errorf("ID = %q, want %q", gs.ID, created.ID)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)

	_, err := repo.GetByID(context.Background(), "00000000-0000-0000-0000-000000000000")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("error = %v, want apperror.ErrNotFound", err)
	}
}

// --- ListByGM ---

func TestListByGM_Empty(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	userID := createTestUser(t, pool)

	sessions, total, err := repo.ListByGM(context.Background(), userID, 20, 0)
	if err != nil {
		t.Fatalf("ListByGM error: %v", err)
	}
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
	if len(sessions) != 0 {
		t.Errorf("sessions count = %d, want 0", len(sessions))
	}
}

func TestListByGM_Pagination(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	userID := createTestUser(t, pool)
	scenarioID := createPublishedScenario(t, pool, userID)

	for i := 0; i < 5; i++ {
		_, err := repo.Create(context.Background(), scenarioID, userID)
		if err != nil {
			t.Fatalf("Create %d error: %v", i, err)
		}
	}

	sessions, total, err := repo.ListByGM(context.Background(), userID, 2, 0)
	if err != nil {
		t.Fatalf("ListByGM error: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(sessions) != 2 {
		t.Errorf("sessions count = %d, want 2", len(sessions))
	}
}

func TestListByGM_Isolation(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	user1 := createTestUser(t, pool)
	user2 := createTestUser2(t, pool)
	scenarioID := createPublishedScenario(t, pool, user1)

	repo.Create(context.Background(), scenarioID, user1)
	repo.Create(context.Background(), scenarioID, user2)

	sessions, total, err := repo.ListByGM(context.Background(), user1, 20, 0)
	if err != nil {
		t.Fatalf("ListByGM error: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(sessions) != 1 {
		t.Errorf("sessions count = %d, want 1", len(sessions))
	}
}

// --- UpdateStatus ---

func TestUpdateStatus_LobbyToActive(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	userID := createTestUser(t, pool)
	scenarioID := createPublishedScenario(t, pool, userID)

	created, _ := repo.Create(context.Background(), scenarioID, userID)

	gs, err := repo.UpdateStatus(context.Background(), created.ID, "active")
	if err != nil {
		t.Fatalf("UpdateStatus error: %v", err)
	}
	if gs.Status != "active" {
		t.Errorf("Status = %q, want %q", gs.Status, "active")
	}
	if gs.StartedAt == nil {
		t.Error("StartedAt should be set")
	}
	if gs.EndedAt != nil {
		t.Error("EndedAt should be nil")
	}
}

func TestUpdateStatus_ActiveToPaused(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	userID := createTestUser(t, pool)
	scenarioID := createPublishedScenario(t, pool, userID)

	created, _ := repo.Create(context.Background(), scenarioID, userID)
	repo.UpdateStatus(context.Background(), created.ID, "active")

	gs, err := repo.UpdateStatus(context.Background(), created.ID, "paused")
	if err != nil {
		t.Fatalf("UpdateStatus error: %v", err)
	}
	if gs.Status != "paused" {
		t.Errorf("Status = %q, want %q", gs.Status, "paused")
	}
}

func TestUpdateStatus_PausedToActive(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	userID := createTestUser(t, pool)
	scenarioID := createPublishedScenario(t, pool, userID)

	created, _ := repo.Create(context.Background(), scenarioID, userID)
	repo.UpdateStatus(context.Background(), created.ID, "active")
	original, _ := repo.GetByID(context.Background(), created.ID)
	repo.UpdateStatus(context.Background(), created.ID, "paused")

	gs, err := repo.UpdateStatus(context.Background(), created.ID, "active")
	if err != nil {
		t.Fatalf("UpdateStatus error: %v", err)
	}
	if gs.Status != "active" {
		t.Errorf("Status = %q, want %q", gs.Status, "active")
	}
	// started_at should not be overwritten
	if gs.StartedAt == nil {
		t.Fatal("StartedAt should not be nil")
	}
	if !gs.StartedAt.Equal(*original.StartedAt) {
		t.Errorf("StartedAt changed: %v vs %v", gs.StartedAt, original.StartedAt)
	}
}

func TestUpdateStatus_ActiveToCompleted(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	userID := createTestUser(t, pool)
	scenarioID := createPublishedScenario(t, pool, userID)

	created, _ := repo.Create(context.Background(), scenarioID, userID)
	repo.UpdateStatus(context.Background(), created.ID, "active")

	gs, err := repo.UpdateStatus(context.Background(), created.ID, "completed")
	if err != nil {
		t.Fatalf("UpdateStatus error: %v", err)
	}
	if gs.Status != "completed" {
		t.Errorf("Status = %q, want %q", gs.Status, "completed")
	}
	if gs.EndedAt == nil {
		t.Error("EndedAt should be set")
	}
}

func TestUpdateStatus_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)

	_, err := repo.UpdateStatus(context.Background(), "00000000-0000-0000-0000-000000000000", "active")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("error = %v, want apperror.ErrNotFound", err)
	}
}

// --- GetByInviteCode ---

func TestGetByInviteCode(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	userID := createTestUser(t, pool)
	scenarioID := createPublishedScenario(t, pool, userID)

	created, _ := repo.Create(context.Background(), scenarioID, userID)

	gs, err := repo.GetByInviteCode(context.Background(), created.InviteCode)
	if err != nil {
		t.Fatalf("GetByInviteCode error: %v", err)
	}
	if gs.ID != created.ID {
		t.Errorf("ID = %q, want %q", gs.ID, created.ID)
	}
}

func TestGetByInviteCode_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)

	_, err := repo.GetByInviteCode(context.Background(), "XXXXXX")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("error = %v, want apperror.ErrNotFound", err)
	}
}

// --- AddPlayer ---

func TestAddPlayer(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	gmID := createTestUser(t, pool)
	playerID := createTestUser2(t, pool)
	scenarioID := createPublishedScenario(t, pool, gmID)

	session, _ := repo.Create(context.Background(), scenarioID, gmID)

	sp, err := repo.AddPlayer(context.Background(), session.ID, playerID)
	if err != nil {
		t.Fatalf("AddPlayer error: %v", err)
	}
	if sp.ID == "" {
		t.Error("ID should not be empty")
	}
	if sp.SessionID != session.ID {
		t.Errorf("SessionID = %q, want %q", sp.SessionID, session.ID)
	}
	if sp.UserID != playerID {
		t.Errorf("UserID = %q, want %q", sp.UserID, playerID)
	}
	if sp.Status != "joined" {
		t.Errorf("Status = %q, want %q", sp.Status, "joined")
	}
}

func TestAddPlayer_Duplicate(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	gmID := createTestUser(t, pool)
	playerID := createTestUser2(t, pool)
	scenarioID := createPublishedScenario(t, pool, gmID)

	session, _ := repo.Create(context.Background(), scenarioID, gmID)
	repo.AddPlayer(context.Background(), session.ID, playerID)

	_, err := repo.AddPlayer(context.Background(), session.ID, playerID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, apperror.ErrDuplicate) {
		t.Errorf("error = %v, want apperror.ErrDuplicate", err)
	}
}

// --- ListPlayers ---

func TestListPlayers(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	gmID := createTestUser(t, pool)
	playerID := createTestUser2(t, pool)
	scenarioID := createPublishedScenario(t, pool, gmID)

	session, _ := repo.Create(context.Background(), scenarioID, gmID)
	repo.AddPlayer(context.Background(), session.ID, playerID)

	players, err := repo.ListPlayers(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("ListPlayers error: %v", err)
	}
	if len(players) != 1 {
		t.Errorf("players count = %d, want 1", len(players))
	}
}

func TestListPlayers_Empty(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	gmID := createTestUser(t, pool)
	scenarioID := createPublishedScenario(t, pool, gmID)

	session, _ := repo.Create(context.Background(), scenarioID, gmID)

	players, err := repo.ListPlayers(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("ListPlayers error: %v", err)
	}
	if len(players) != 0 {
		t.Errorf("players count = %d, want 0", len(players))
	}
}

// --- RemovePlayer ---

func TestRemovePlayer(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	gmID := createTestUser(t, pool)
	playerID := createTestUser2(t, pool)
	scenarioID := createPublishedScenario(t, pool, gmID)

	session, _ := repo.Create(context.Background(), scenarioID, gmID)
	repo.AddPlayer(context.Background(), session.ID, playerID)

	err := repo.RemovePlayer(context.Background(), session.ID, playerID)
	if err != nil {
		t.Fatalf("RemovePlayer error: %v", err)
	}

	// Verify removal
	players, _ := repo.ListPlayers(context.Background(), session.ID)
	if len(players) != 0 {
		t.Errorf("players count after removal = %d, want 0", len(players))
	}
}

func TestRemovePlayer_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	gmID := createTestUser(t, pool)
	scenarioID := createPublishedScenario(t, pool, gmID)

	session, _ := repo.Create(context.Background(), scenarioID, gmID)

	err := repo.RemovePlayer(context.Background(), session.ID, "00000000-0000-0000-0000-000000000000")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("error = %v, want apperror.ErrNotFound", err)
	}
}

// --- GetPlayer ---

func TestGetPlayer(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	gmID := createTestUser(t, pool)
	playerID := createTestUser2(t, pool)
	scenarioID := createPublishedScenario(t, pool, gmID)

	session, _ := repo.Create(context.Background(), scenarioID, gmID)
	repo.AddPlayer(context.Background(), session.ID, playerID)

	sp, err := repo.GetPlayer(context.Background(), session.ID, playerID)
	if err != nil {
		t.Fatalf("GetPlayer error: %v", err)
	}
	if sp.UserID != playerID {
		t.Errorf("UserID = %q, want %q", sp.UserID, playerID)
	}
}

func TestGetPlayer_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	gmID := createTestUser(t, pool)
	scenarioID := createPublishedScenario(t, pool, gmID)

	session, _ := repo.Create(context.Background(), scenarioID, gmID)

	_, err := repo.GetPlayer(context.Background(), session.ID, "00000000-0000-0000-0000-000000000000")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("error = %v, want apperror.ErrNotFound", err)
	}
}
