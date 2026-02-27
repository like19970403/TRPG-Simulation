package realtime

import (
	"encoding/json"
	"fmt"
)

// GameState holds the in-memory state for an active game session.
// Per ADR-004, all state mutations happen in the Room goroutine (no mutex needed).
type GameState struct {
	SessionID    string                 `json:"session_id"`
	Status       string                 `json:"status"`
	CurrentScene string                 `json:"current_scene,omitempty"`
	Players      map[string]PlayerState `json:"players,omitempty"`
	DiceHistory  []DiceResult           `json:"dice_history,omitempty"`
	LastSequence int64                  `json:"last_sequence"`
}

// PlayerState tracks per-player state within a game session.
type PlayerState struct {
	UserID       string `json:"user_id"`
	CurrentScene string `json:"current_scene"`
}

// NewGameState creates a GameState for a newly started session.
func NewGameState(sessionID string) *GameState {
	return &GameState{
		SessionID:    sessionID,
		Status:       "active",
		LastSequence: 0,
	}
}

// Apply processes an event and updates the game state.
// Returns an error if the transition is invalid or the sequence is stale.
func (gs *GameState) Apply(eventType string, sequence int64, payload json.RawMessage) error {
	if sequence <= gs.LastSequence {
		return fmt.Errorf("realtime: stale event sequence %d, last was %d", sequence, gs.LastSequence)
	}

	switch eventType {
	case EventGameStarted:
		gs.Status = "active"
	case EventGamePaused:
		if gs.Status != "active" {
			return fmt.Errorf("realtime: cannot pause, status is %q (expected active)", gs.Status)
		}
		gs.Status = "paused"
	case EventGameResumed:
		if gs.Status != "paused" {
			return fmt.Errorf("realtime: cannot resume, status is %q (expected paused)", gs.Status)
		}
		gs.Status = "active"
	case EventGameEnded:
		if gs.Status != "active" && gs.Status != "paused" {
			return fmt.Errorf("realtime: cannot end, status is %q (expected active or paused)", gs.Status)
		}
		gs.Status = "completed"
	case EventSceneChanged:
		var p struct {
			SceneID string `json:"scene_id"`
		}
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("realtime: invalid scene_changed payload: %w", err)
		}
		gs.CurrentScene = p.SceneID
	case EventDiceRolled:
		var dr DiceResult
		if err := json.Unmarshal(payload, &dr); err != nil {
			return fmt.Errorf("realtime: invalid dice_rolled payload: %w", err)
		}
		gs.DiceHistory = append(gs.DiceHistory, dr)
	default:
		// Unknown event types are accepted for forward compatibility.
	}

	gs.LastSequence = sequence
	return nil
}

// StateJSON serializes the GameState for state_sync broadcasts.
func (gs *GameState) StateJSON() json.RawMessage {
	data, _ := json.Marshal(gs)
	return data
}
