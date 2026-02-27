package realtime

import (
	"encoding/json"
	"time"
)

// Event type constants for lifecycle events (SPEC-005 scope).
const (
	EventGameStarted = "game_started"
	EventGamePaused  = "game_paused"
	EventGameResumed = "game_resumed"
	EventGameEnded   = "game_ended"
	EventStateSync   = "state_sync"
	EventError       = "error"
)

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
