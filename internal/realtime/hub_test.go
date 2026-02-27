package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/like19970403/TRPG-Simulation/internal/game"
)

func TestHub_GetOrCreateRoom_CreatesNewRoom(t *testing.T) {
	hub := NewHub(successEventRepo(), nil, testLogger())
	defer hub.Stop()

	room := hub.GetOrCreateRoom("sess-1", "gm-1")
	if room == nil {
		t.Fatal("expected room, got nil")
	}
	if room.SessionID() != "sess-1" {
		t.Errorf("SessionID = %q, want %q", room.SessionID(), "sess-1")
	}
	if hub.RoomCount() != 1 {
		t.Errorf("RoomCount = %d, want 1", hub.RoomCount())
	}
}

func TestHub_GetOrCreateRoom_ReturnsExisting(t *testing.T) {
	hub := NewHub(successEventRepo(), nil, testLogger())
	defer hub.Stop()

	room1 := hub.GetOrCreateRoom("sess-1", "gm-1")
	room2 := hub.GetOrCreateRoom("sess-1", "gm-1")

	if room1 != room2 {
		t.Error("expected same room instance")
	}
	if hub.RoomCount() != 1 {
		t.Errorf("RoomCount = %d, want 1", hub.RoomCount())
	}
}

func TestHub_GetRoom_ReturnsNilIfNotExists(t *testing.T) {
	hub := NewHub(successEventRepo(), nil, testLogger())
	defer hub.Stop()

	room := hub.GetRoom("nonexistent")
	if room != nil {
		t.Error("expected nil, got room")
	}
}

func TestHub_RemoveRoom(t *testing.T) {
	hub := NewHub(successEventRepo(), nil, testLogger())
	defer hub.Stop()

	hub.GetOrCreateRoom("sess-1", "gm-1")
	if hub.RoomCount() != 1 {
		t.Fatalf("RoomCount = %d, want 1", hub.RoomCount())
	}

	hub.RemoveRoom("sess-1")
	time.Sleep(50 * time.Millisecond)

	if hub.RoomCount() != 0 {
		t.Errorf("RoomCount = %d, want 0", hub.RoomCount())
	}
	if hub.GetRoom("sess-1") != nil {
		t.Error("room should be nil after removal")
	}
}

func TestHub_Stop_CleansUpAllRooms(t *testing.T) {
	hub := NewHub(successEventRepo(), nil, testLogger())

	hub.GetOrCreateRoom("sess-1", "gm-1")
	hub.GetOrCreateRoom("sess-2", "gm-2")

	if hub.RoomCount() != 2 {
		t.Fatalf("RoomCount = %d, want 2", hub.RoomCount())
	}

	hub.Stop()

	if hub.RoomCount() != 0 {
		t.Errorf("RoomCount = %d, want 0 after Stop", hub.RoomCount())
	}
}

// --- State recovery tests ---

func TestHub_GetOrCreateRoom_RecoverState(t *testing.T) {
	// Prepare a snapshot with known state.
	snapshotState := &GameState{
		SessionID:    "sess-recover",
		Status:       "active",
		CurrentScene: "library",
		LastSequence: 50,
		Variables:    map[string]any{"anger": float64(3)},
	}
	snapshotData, _ := json.Marshal(snapshotState)

	// Events after snapshot.
	event51Payload, _ := json.Marshal(map[string]any{"scene_id": "dungeon"})

	repo := successEventRepo()
	repo.loadSnapshotFn = func(_ context.Context, _ string) (int64, json.RawMessage, error) {
		return 50, snapshotData, nil
	}
	repo.listEventsSinceFn = func(_ context.Context, _ string, afterSeq int64) ([]*game.GameEvent, error) {
		return []*game.GameEvent{
			{ID: "e51", SessionID: "sess-recover", Sequence: 51, Type: EventSceneChanged, Payload: event51Payload},
		}, nil
	}

	hub := NewHub(repo, nil, testLogger())
	defer hub.Stop()

	room := hub.GetOrCreateRoom("sess-recover", "gm-1")
	time.Sleep(50 * time.Millisecond)

	state := room.StateSnapshot()
	if state.LastSequence != 51 {
		t.Errorf("LastSequence = %d, want 51", state.LastSequence)
	}
	if state.CurrentScene != "dungeon" {
		t.Errorf("CurrentScene = %q, want 'dungeon'", state.CurrentScene)
	}
	if state.Variables["anger"] != float64(3) {
		t.Errorf("anger = %v, want 3", state.Variables["anger"])
	}
}

func TestHub_GetOrCreateRoom_RecoverFails_GracefulDegradation(t *testing.T) {
	repo := successEventRepo()
	repo.loadSnapshotFn = func(_ context.Context, _ string) (int64, json.RawMessage, error) {
		return 0, nil, errors.New("db connection failed")
	}

	hub := NewHub(repo, nil, testLogger())
	defer hub.Stop()

	// Room should still be created and functional despite recovery failure.
	room := hub.GetOrCreateRoom("sess-fail", "gm-1")
	if room == nil {
		t.Fatal("expected room, got nil")
	}
	time.Sleep(50 * time.Millisecond)

	state := room.StateSnapshot()
	// Should have fresh state (LastSequence=0, Status="active").
	if state.LastSequence != 0 {
		t.Errorf("LastSequence = %d, want 0 (fresh state)", state.LastSequence)
	}
	if state.Status != "active" {
		t.Errorf("Status = %q, want 'active'", state.Status)
	}
}

// mockScenarioLoader implements ScenarioLoader for testing.
type mockScenarioLoader struct {
	loadFn func(ctx context.Context, sessionID string) (*ScenarioContent, error)
}

func (m *mockScenarioLoader) LoadScenarioForSession(ctx context.Context, sessionID string) (*ScenarioContent, error) {
	if m.loadFn != nil {
		return m.loadFn(ctx, sessionID)
	}
	return nil, nil
}

func TestHub_GetOrCreateRoom_WithScenarioAndRecovery(t *testing.T) {
	// Snapshot at seq=10 with a scene set.
	snapshotState := &GameState{
		SessionID:    "sess-combo",
		Status:       "active",
		CurrentScene: "entrance",
		LastSequence: 10,
	}
	snapshotData, _ := json.Marshal(snapshotState)

	repo := successEventRepo()
	repo.loadSnapshotFn = func(_ context.Context, _ string) (int64, json.RawMessage, error) {
		return 10, snapshotData, nil
	}
	repo.listEventsSinceFn = func(_ context.Context, _ string, _ int64) ([]*game.GameEvent, error) {
		return []*game.GameEvent{}, nil
	}

	loader := &mockScenarioLoader{
		loadFn: func(_ context.Context, _ string) (*ScenarioContent, error) {
			return &ScenarioContent{
				Scenes: []Scene{{ID: "entrance"}, {ID: "library"}},
			}, nil
		},
	}

	hub := NewHub(repo, loader, testLogger())
	defer hub.Stop()

	room := hub.GetOrCreateRoom("sess-combo", "gm-1")
	time.Sleep(50 * time.Millisecond)

	// Verify state was recovered.
	state := room.StateSnapshot()
	if state.LastSequence != 10 {
		t.Errorf("LastSequence = %d, want 10", state.LastSequence)
	}
	if state.CurrentScene != "entrance" {
		t.Errorf("CurrentScene = %q, want 'entrance'", state.CurrentScene)
	}

	// Verify scenario was loaded (room can find scenes).
	if room.scenario == nil {
		t.Fatal("scenario should be loaded")
	}
	if room.scenario.FindScene("library") == nil {
		t.Error("scenario should contain 'library' scene")
	}
}
