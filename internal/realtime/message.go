package realtime

import (
	"encoding/json"
	"time"
)

// Event type constants for lifecycle events (SPEC-005) and game events (SPEC-006).
const (
	EventGameStarted  = "game_started"
	EventGamePaused   = "game_paused"
	EventGameResumed  = "game_resumed"
	EventGameEnded    = "game_ended"
	EventStateSync    = "state_sync"
	EventError        = "error"
	EventSceneChanged = "scene_changed"
	EventDiceRolled   = "dice_rolled"
)

// IncomingAction represents a client-to-server WebSocket message.
type IncomingAction struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// AdvanceScenePayload is the payload for an advance_scene action.
type AdvanceScenePayload struct {
	SceneID string `json:"scene_id"`
}

// DiceRollPayload is the payload for a dice_roll action.
type DiceRollPayload struct {
	Formula string `json:"formula"`
	Purpose string `json:"purpose,omitempty"`
}

// Envelope is the wire format for all WebSocket messages (ADR-002).
type Envelope struct {
	Type      string          `json:"type"`
	SessionID string          `json:"session_id"`
	SenderID  string          `json:"sender_id"`
	TargetIDs []string        `json:"target_ids,omitempty"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp int64           `json:"timestamp"`
}

// NewEnvelope creates an Envelope with the current timestamp.
func NewEnvelope(eventType, sessionID, senderID string, payload json.RawMessage) Envelope {
	return Envelope{
		Type:      eventType,
		SessionID: sessionID,
		SenderID:  senderID,
		Payload:   payload,
		Timestamp: time.Now().Unix(),
	}
}

// SenderRole identifies the role of a connected user within a session.
type SenderRole string

const (
	RoleGM     SenderRole = "gm"
	RolePlayer SenderRole = "player"
)
