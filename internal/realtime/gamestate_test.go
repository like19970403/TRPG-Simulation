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
	err := gs.Apply("future_event_type", 1, json.RawMessage(`{}`))
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

func TestApply_SceneChanged(t *testing.T) {
	gs := NewGameState("s1")
	err := gs.Apply(EventSceneChanged, 1, json.RawMessage(`{"scene_id":"library"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gs.CurrentScene != "library" {
		t.Errorf("CurrentScene = %q, want %q", gs.CurrentScene, "library")
	}
	if gs.LastSequence != 1 {
		t.Errorf("LastSequence = %d, want 1", gs.LastSequence)
	}
}

func TestApply_SceneChanged_Double(t *testing.T) {
	gs := NewGameState("s1")
	_ = gs.Apply(EventSceneChanged, 1, json.RawMessage(`{"scene_id":"entrance"}`))
	err := gs.Apply(EventSceneChanged, 2, json.RawMessage(`{"scene_id":"library"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gs.CurrentScene != "library" {
		t.Errorf("CurrentScene = %q, want %q", gs.CurrentScene, "library")
	}
}

func TestApply_SceneChanged_InvalidPayload(t *testing.T) {
	gs := NewGameState("s1")
	err := gs.Apply(EventSceneChanged, 1, json.RawMessage(`{invalid}`))
	if err == nil {
		t.Fatal("expected error for invalid payload")
	}
	if !strings.Contains(err.Error(), "invalid scene_changed payload") {
		t.Errorf("error = %q, want to contain 'invalid scene_changed payload'", err.Error())
	}
}

func TestApply_DiceRolled(t *testing.T) {
	gs := NewGameState("s1")
	payload := json.RawMessage(`{"formula":"2d6","results":[3,4],"modifier":0,"total":7}`)
	err := gs.Apply(EventDiceRolled, 1, payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gs.DiceHistory) != 1 {
		t.Fatalf("len(DiceHistory) = %d, want 1", len(gs.DiceHistory))
	}
	if gs.DiceHistory[0].Formula != "2d6" {
		t.Errorf("Formula = %q, want %q", gs.DiceHistory[0].Formula, "2d6")
	}
	if gs.DiceHistory[0].Total != 7 {
		t.Errorf("Total = %d, want 7", gs.DiceHistory[0].Total)
	}
}

func TestApply_DiceRolled_Multiple(t *testing.T) {
	gs := NewGameState("s1")
	_ = gs.Apply(EventDiceRolled, 1, json.RawMessage(`{"formula":"1d6","results":[3],"modifier":0,"total":3}`))
	err := gs.Apply(EventDiceRolled, 2, json.RawMessage(`{"formula":"1d20","results":[15],"modifier":0,"total":15}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gs.DiceHistory) != 2 {
		t.Fatalf("len(DiceHistory) = %d, want 2", len(gs.DiceHistory))
	}
}

func TestApply_DiceRolled_InvalidPayload(t *testing.T) {
	gs := NewGameState("s1")
	err := gs.Apply(EventDiceRolled, 1, json.RawMessage(`{invalid}`))
	if err == nil {
		t.Fatal("expected error for invalid payload")
	}
	if !strings.Contains(err.Error(), "invalid dice_rolled payload") {
		t.Errorf("error = %q, want to contain 'invalid dice_rolled payload'", err.Error())
	}
}

func TestApply_PlayersMapInitialized(t *testing.T) {
	gs := NewGameState("s1")
	// Players map starts nil by default.
	if gs.Players != nil {
		t.Errorf("Players should be nil initially, got %v", gs.Players)
	}
}

func TestStateJSON_WithNewFields(t *testing.T) {
	gs := NewGameState("sess-1")
	gs.CurrentScene = "library"
	gs.DiceHistory = []DiceResult{
		{Formula: "1d6", Results: []int{4}, Modifier: 0, Total: 4},
	}
	gs.LastSequence = 3

	data := gs.StateJSON()
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded["current_scene"] != "library" {
		t.Errorf("current_scene = %v, want %q", decoded["current_scene"], "library")
	}
	dh, ok := decoded["dice_history"].([]any)
	if !ok {
		t.Fatalf("dice_history is not an array: %v", decoded["dice_history"])
	}
	if len(dh) != 1 {
		t.Errorf("len(dice_history) = %d, want 1", len(dh))
	}
}
