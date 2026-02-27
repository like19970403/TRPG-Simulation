package realtime

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNewGameState(t *testing.T) {
	gs := NewGameState("sess-1")
	if gs.SessionID != "sess-1" {
		t.Errorf("SessionID = %q, want %q", gs.SessionID, "sess-1")
	}
	if gs.Status != "active" {
		t.Errorf("Status = %q, want %q", gs.Status, "active")
	}
	if gs.LastSequence != 0 {
		t.Errorf("LastSequence = %d, want 0", gs.LastSequence)
	}
}

func TestApply_GameStarted(t *testing.T) {
	gs := &GameState{SessionID: "s1", Status: "lobby", LastSequence: 0}
	err := gs.Apply(EventGameStarted, 1, json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gs.Status != "active" {
		t.Errorf("Status = %q, want %q", gs.Status, "active")
	}
	if gs.LastSequence != 1 {
		t.Errorf("LastSequence = %d, want 1", gs.LastSequence)
	}
}

func TestApply_GamePaused(t *testing.T) {
	gs := NewGameState("s1")
	gs.LastSequence = 1
	err := gs.Apply(EventGamePaused, 2, json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gs.Status != "paused" {
		t.Errorf("Status = %q, want %q", gs.Status, "paused")
	}
}

func TestApply_GamePaused_WhenNotActive(t *testing.T) {
	gs := &GameState{SessionID: "s1", Status: "paused", LastSequence: 1}
	err := gs.Apply(EventGamePaused, 2, json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "cannot pause") {
		t.Errorf("error = %q, want to contain 'cannot pause'", err.Error())
	}
}

func TestApply_GameResumed(t *testing.T) {
	gs := &GameState{SessionID: "s1", Status: "paused", LastSequence: 2}
	err := gs.Apply(EventGameResumed, 3, json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gs.Status != "active" {
		t.Errorf("Status = %q, want %q", gs.Status, "active")
	}
}

func TestApply_GameResumed_WhenNotPaused(t *testing.T) {
	gs := NewGameState("s1")
	gs.LastSequence = 1
	err := gs.Apply(EventGameResumed, 2, json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "cannot resume") {
		t.Errorf("error = %q, want to contain 'cannot resume'", err.Error())
	}
}

func TestApply_GameEnded_FromActive(t *testing.T) {
	gs := NewGameState("s1")
	gs.LastSequence = 1
	err := gs.Apply(EventGameEnded, 2, json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gs.Status != "completed" {
		t.Errorf("Status = %q, want %q", gs.Status, "completed")
	}
}

func TestApply_GameEnded_FromPaused(t *testing.T) {
	gs := &GameState{SessionID: "s1", Status: "paused", LastSequence: 2}
	err := gs.Apply(EventGameEnded, 3, json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gs.Status != "completed" {
		t.Errorf("Status = %q, want %q", gs.Status, "completed")
	}
}

func TestApply_GameEnded_FromCompleted(t *testing.T) {
	gs := &GameState{SessionID: "s1", Status: "completed", LastSequence: 3}
	err := gs.Apply(EventGameEnded, 4, json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "cannot end") {
		t.Errorf("error = %q, want to contain 'cannot end'", err.Error())
	}
}

func TestApply_StaleSequence(t *testing.T) {
	gs := NewGameState("s1")
	gs.LastSequence = 5
	err := gs.Apply(EventGameStarted, 3, json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error for stale sequence, got nil")
	}
	if !strings.Contains(err.Error(), "stale") {
		t.Errorf("error = %q, want to contain 'stale'", err.Error())
	}
}

func TestApply_UnknownEvent(t *testing.T) {
	gs := NewGameState("s1")
	err := gs.Apply("scene_changed", 1, json.RawMessage(`{"scene_id":"library"}`))
	if err != nil {
		t.Fatalf("unexpected error for unknown event: %v", err)
	}
	if gs.LastSequence != 1 {
		t.Errorf("LastSequence = %d, want 1", gs.LastSequence)
	}
}

func TestStateJSON(t *testing.T) {
	gs := NewGameState("sess-123")
	gs.LastSequence = 5

	data := gs.StateJSON()
	var decoded GameState
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.SessionID != "sess-123" {
		t.Errorf("SessionID = %q, want %q", decoded.SessionID, "sess-123")
	}
	if decoded.Status != "active" {
		t.Errorf("Status = %q, want %q", decoded.Status, "active")
	}
	if decoded.LastSequence != 5 {
		t.Errorf("LastSequence = %d, want 5", decoded.LastSequence)
	}
}
