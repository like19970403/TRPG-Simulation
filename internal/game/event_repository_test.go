package game

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/like19970403/TRPG-Simulation/internal/apperror"
)

func createTestSession(t *testing.T, repo *Repository, scenarioID, gmID string) string {
	t.Helper()
	gs, err := repo.Create(context.Background(), scenarioID, gmID)
	if err != nil {
		t.Fatalf("create test session: %v", err)
	}
	return gs.ID
}

func TestAppendEvent_Success(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	gmID := createTestUser(t, pool)
	scnID := createPublishedScenario(t, pool, gmID)
	sessID := createTestSession(t, repo, scnID, gmID)

	payload := json.RawMessage(`{}`)
	e, err := repo.AppendEvent(context.Background(), sessID, 1, "game_started", &gmID, payload)
	if err != nil {
		t.Fatalf("AppendEvent: %v", err)
	}

	if e.ID == "" {
		t.Error("ID should not be empty")
	}
	if e.SessionID != sessID {
		t.Errorf("SessionID = %q, want %q", e.SessionID, sessID)
	}
	if e.Sequence != 1 {
		t.Errorf("Sequence = %d, want 1", e.Sequence)
	}
	if e.Type != "game_started" {
		t.Errorf("Type = %q, want %q", e.Type, "game_started")
	}
	if e.ActorID == nil || *e.ActorID != gmID {
		t.Errorf("ActorID = %v, want %q", e.ActorID, gmID)
	}
	if e.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

func TestAppendEvent_DuplicateSequence(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	gmID := createTestUser(t, pool)
	scnID := createPublishedScenario(t, pool, gmID)
	sessID := createTestSession(t, repo, scnID, gmID)

	payload := json.RawMessage(`{}`)
	_, err := repo.AppendEvent(context.Background(), sessID, 1, "game_started", &gmID, payload)
	if err != nil {
		t.Fatalf("first AppendEvent: %v", err)
	}

	_, err = repo.AppendEvent(context.Background(), sessID, 1, "game_paused", &gmID, payload)
	if err == nil {
		t.Fatal("expected error for duplicate sequence, got nil")
	}
	if !errors.Is(err, apperror.ErrDuplicate) {
		t.Errorf("error = %v, want apperror.ErrDuplicate", err)
	}
}

func TestListEventsSince_Partial(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	gmID := createTestUser(t, pool)
	scnID := createPublishedScenario(t, pool, gmID)
	sessID := createTestSession(t, repo, scnID, gmID)

	payload := json.RawMessage(`{}`)
	for i := int64(1); i <= 5; i++ {
		_, err := repo.AppendEvent(context.Background(), sessID, i, "game_started", &gmID, payload)
		if err != nil {
			t.Fatalf("AppendEvent seq %d: %v", i, err)
		}
	}

	events, err := repo.ListEventsSince(context.Background(), sessID, 2)
	if err != nil {
		t.Fatalf("ListEventsSince: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("got %d events, want 3", len(events))
	}
	if events[0].Sequence != 3 {
		t.Errorf("first event sequence = %d, want 3", events[0].Sequence)
	}
	if events[2].Sequence != 5 {
		t.Errorf("last event sequence = %d, want 5", events[2].Sequence)
	}
}

func TestListEventsSince_Empty(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	gmID := createTestUser(t, pool)
	scnID := createPublishedScenario(t, pool, gmID)
	sessID := createTestSession(t, repo, scnID, gmID)

	events, err := repo.ListEventsSince(context.Background(), sessID, 999)
	if err != nil {
		t.Fatalf("ListEventsSince: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("got %d events, want 0", len(events))
	}
}

func TestListEventsSince_All(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)
	gmID := createTestUser(t, pool)
	scnID := createPublishedScenario(t, pool, gmID)
	sessID := createTestSession(t, repo, scnID, gmID)

	payload := json.RawMessage(`{}`)
	for i := int64(1); i <= 3; i++ {
		_, err := repo.AppendEvent(context.Background(), sessID, i, "game_started", &gmID, payload)
		if err != nil {
			t.Fatalf("AppendEvent seq %d: %v", i, err)
		}
	}

	events, err := repo.ListEventsSince(context.Background(), sessID, 0)
	if err != nil {
		t.Fatalf("ListEventsSince: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("got %d events, want 3", len(events))
	}
}
