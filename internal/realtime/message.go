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
	EventSceneChanged     = "scene_changed"
	EventDiceRolled       = "dice_rolled"
	EventItemRevealed     = "item_revealed"
	EventItemGiven        = "item_given"
	EventItemRemoved      = "item_removed"
	EventNPCFieldRevealed = "npc_field_revealed"
	EventVariableChanged  = "variable_changed"
	EventPlayerChoice     = "player_choice"
	EventPlayerVotes      = "player_votes"
	EventGMBroadcast      = "gm_broadcast"
	EventTransitionsUpdated = "transitions_updated"
	EventPlayerJoined       = "player_joined"
	EventPlayerLeft         = "player_left"
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

// RevealItemPayload is the payload for a reveal_item action (GM-only).
type RevealItemPayload struct {
	ItemID    string   `json:"item_id"`
	PlayerIDs []string `json:"player_ids,omitempty"` // empty = all connected players
}

// GiveItemPayload is the payload for a give_item action (GM-only).
type GiveItemPayload struct {
	ItemID    string   `json:"item_id"`
	PlayerID  string   `json:"player_id,omitempty"`  // single target
	PlayerIDs []string `json:"player_ids,omitempty"` // multiple targets; empty = all connected
	Quantity  int      `json:"quantity,omitempty"`    // default 1
}

// RemoveItemPayload is the payload for a remove_item action (GM-only).
type RemoveItemPayload struct {
	ItemID    string   `json:"item_id"`
	PlayerID  string   `json:"player_id,omitempty"`  // single target
	PlayerIDs []string `json:"player_ids,omitempty"` // multiple targets; empty = all connected
	Quantity  int      `json:"quantity,omitempty"`    // default 1; 0 = remove all
}

// RevealNPCFieldPayload is the payload for a reveal_npc_field action (GM-only).
type RevealNPCFieldPayload struct {
	NPCID     string   `json:"npc_id"`
	FieldKey  string   `json:"field_key"`
	PlayerIDs []string `json:"player_ids,omitempty"` // empty = all connected players
}

// PlayerChoicePayload is the payload for a player_choice action.
type PlayerChoicePayload struct {
	TransitionIndex int `json:"transition_index"`
}

// GMBroadcastPayload is the payload for a gm_broadcast action (GM-only).
type GMBroadcastPayload struct {
	Content   string   `json:"content,omitempty"`
	ImageURL  string   `json:"image_url,omitempty"`
	PlayerIDs []string `json:"player_ids,omitempty"` // empty = all connected players
}

// SetVariablePayload is the payload for a set_variable action (GM-only).
type SetVariablePayload struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
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
