package realtime

import (
	"encoding/json"
	"fmt"
)

// GameState holds the in-memory state for an active game session.
// Per ADR-004, all state mutations happen in the Room goroutine (no mutex needed).
type GameState struct {
	SessionID         string                            `json:"session_id"`
	Status            string                            `json:"status"`
	CurrentScene      string                            `json:"current_scene,omitempty"`
	Players           map[string]PlayerState            `json:"players,omitempty"`
	PlayerAttributes  map[string]map[string]any         `json:"player_attributes,omitempty"`  // playerID → attrName → value
	DiceHistory       []DiceResult                      `json:"dice_history,omitempty"`
	Variables         map[string]any                    `json:"variables,omitempty"`
	RevealedItems     map[string][]string               `json:"revealed_items,omitempty"`      // playerID → []itemID (deprecated, kept for compat)
	PlayerInventory   map[string][]InventoryEntry      `json:"player_inventory,omitempty"`    // playerID → []InventoryEntry
	RevealedNPCFields map[string]map[string][]string    `json:"revealed_npc_fields,omitempty"` // playerID → npcID → []fieldKey
	LastSequence      int64                             `json:"last_sequence"`
}

// PlayerState tracks per-player state within a game session.
type PlayerState struct {
	UserID        string `json:"user_id"`
	Username      string `json:"username"`
	CharacterID   string `json:"character_id,omitempty"`
	CharacterName string `json:"character_name,omitempty"`
	CurrentScene  string `json:"current_scene"`
	Online        bool   `json:"online"`
}

// InventoryEntry represents an item in a player's inventory.
type InventoryEntry struct {
	ItemID   string `json:"item_id"`
	Quantity int    `json:"quantity"`
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
		// Update all online players' current_scene.
		for uid, ps := range gs.Players {
			if ps.Online {
				ps.CurrentScene = p.SceneID
				gs.Players[uid] = ps
			}
		}
	case EventDiceRolled:
		var dr DiceResult
		if err := json.Unmarshal(payload, &dr); err != nil {
			return fmt.Errorf("realtime: invalid dice_rolled payload: %w", err)
		}
		gs.DiceHistory = append(gs.DiceHistory, dr)
	case EventVariableChanged:
		var p struct {
			Name     string `json:"name"`
			NewValue any    `json:"new_value"`
		}
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("realtime: invalid variable_changed payload: %w", err)
		}
		if gs.Variables == nil {
			gs.Variables = make(map[string]any)
		}
		gs.Variables[p.Name] = p.NewValue
	case EventItemRevealed:
		var p struct {
			ItemID    string   `json:"item_id"`
			PlayerIDs []string `json:"player_ids"`
		}
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("realtime: invalid item_revealed payload: %w", err)
		}
		if gs.RevealedItems == nil {
			gs.RevealedItems = make(map[string][]string)
		}
		for _, pid := range p.PlayerIDs {
			if !gs.IsItemRevealed(pid, p.ItemID) {
				gs.RevealedItems[pid] = append(gs.RevealedItems[pid], p.ItemID)
			}
		}
		// Backward compat: also write to PlayerInventory (qty 1).
		gs.giveInventoryItem(p.PlayerIDs, p.ItemID, 1)
	case EventItemGiven:
		var p struct {
			ItemID    string   `json:"item_id"`
			PlayerIDs []string `json:"player_ids"`
			Quantity  int      `json:"quantity"`
		}
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("realtime: invalid item_given payload: %w", err)
		}
		qty := p.Quantity
		if qty <= 0 {
			qty = 1
		}
		gs.giveInventoryItem(p.PlayerIDs, p.ItemID, qty)
	case EventItemRemoved:
		var p struct {
			ItemID    string   `json:"item_id"`
			PlayerIDs []string `json:"player_ids"`
			Quantity  int      `json:"quantity"`
		}
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("realtime: invalid item_removed payload: %w", err)
		}
		gs.removeInventoryItem(p.PlayerIDs, p.ItemID, p.Quantity)
	case EventNPCFieldRevealed:
		var p struct {
			NPCID     string   `json:"npc_id"`
			FieldKey  string   `json:"field_key"`
			PlayerIDs []string `json:"player_ids"`
		}
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("realtime: invalid npc_field_revealed payload: %w", err)
		}
		if gs.RevealedNPCFields == nil {
			gs.RevealedNPCFields = make(map[string]map[string][]string)
		}
		for _, pid := range p.PlayerIDs {
			if gs.RevealedNPCFields[pid] == nil {
				gs.RevealedNPCFields[pid] = make(map[string][]string)
			}
			// Dedup check.
			found := false
			for _, fk := range gs.RevealedNPCFields[pid][p.NPCID] {
				if fk == p.FieldKey {
					found = true
					break
				}
			}
			if !found {
				gs.RevealedNPCFields[pid][p.NPCID] = append(gs.RevealedNPCFields[pid][p.NPCID], p.FieldKey)
			}
		}
	case EventPlayerJoined:
		var p struct {
			UserID        string         `json:"user_id"`
			Username      string         `json:"username"`
			CharacterID   string         `json:"character_id,omitempty"`
			CharacterName string         `json:"character_name,omitempty"`
			Attributes    map[string]any `json:"attributes,omitempty"`
		}
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("realtime: invalid player_joined payload: %w", err)
		}
		if gs.Players == nil {
			gs.Players = make(map[string]PlayerState)
		}
		gs.Players[p.UserID] = PlayerState{
			UserID:        p.UserID,
			Username:      p.Username,
			CharacterID:   p.CharacterID,
			CharacterName: p.CharacterName,
			CurrentScene:  gs.CurrentScene,
			Online:        true,
		}
		if len(p.Attributes) > 0 {
			if gs.PlayerAttributes == nil {
				gs.PlayerAttributes = make(map[string]map[string]any)
			}
			gs.PlayerAttributes[p.UserID] = p.Attributes
		}
	case EventPlayerLeft:
		var p struct {
			UserID string `json:"user_id"`
		}
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("realtime: invalid player_left payload: %w", err)
		}
		if gs.Players != nil {
			if ps, ok := gs.Players[p.UserID]; ok {
				ps.Online = false
				gs.Players[p.UserID] = ps
			}
		}
	case EventPlayerChoice:
		// Informational event for audit trail. No state mutation needed.
	case EventPlayerVotes:
		// Transient vote tally broadcast. No state mutation needed.
	case EventGMBroadcast:
		// Informational event. No state mutation needed; sequence is advanced below.
	default:
		// Unknown event types are accepted for forward compatibility.
	}

	gs.LastSequence = sequence
	return nil
}

// InitVariables initializes the Variables map from scenario variable definitions.
func (gs *GameState) InitVariables(vars []Variable) {
	if len(vars) == 0 {
		return
	}
	gs.Variables = make(map[string]any, len(vars))
	for _, v := range vars {
		gs.Variables[v.Name] = v.Default
	}
}

// IsItemRevealed returns true if the given item has been revealed to the player.
// Also checks PlayerInventory for backward compatibility.
func (gs *GameState) IsItemRevealed(playerID, itemID string) bool {
	if gs.HasItem(playerID, itemID) {
		return true
	}
	if gs.RevealedItems == nil {
		return false
	}
	for _, id := range gs.RevealedItems[playerID] {
		if id == itemID {
			return true
		}
	}
	return false
}

// HasItem returns true if the player has at least 1 of the given item in inventory.
func (gs *GameState) HasItem(playerID, itemID string) bool {
	return gs.ItemQuantity(playerID, itemID) > 0
}

// ItemQuantity returns the quantity of a specific item for a player (0 if not found).
func (gs *GameState) ItemQuantity(playerID, itemID string) int {
	if gs.PlayerInventory == nil {
		return 0
	}
	for _, e := range gs.PlayerInventory[playerID] {
		if e.ItemID == itemID {
			return e.Quantity
		}
	}
	return 0
}

// giveInventoryItem adds or stacks an item in the player(s) inventory.
func (gs *GameState) giveInventoryItem(playerIDs []string, itemID string, qty int) {
	if gs.PlayerInventory == nil {
		gs.PlayerInventory = make(map[string][]InventoryEntry)
	}
	for _, pid := range playerIDs {
		found := false
		for i, e := range gs.PlayerInventory[pid] {
			if e.ItemID == itemID {
				gs.PlayerInventory[pid][i].Quantity += qty
				found = true
				break
			}
		}
		if !found {
			gs.PlayerInventory[pid] = append(gs.PlayerInventory[pid], InventoryEntry{
				ItemID:   itemID,
				Quantity: qty,
			})
		}
	}
}

// removeInventoryItem removes quantity of an item from the player(s) inventory.
// qty=0 means remove all.
func (gs *GameState) removeInventoryItem(playerIDs []string, itemID string, qty int) {
	if gs.PlayerInventory == nil {
		return
	}
	for _, pid := range playerIDs {
		entries := gs.PlayerInventory[pid]
		for i, e := range entries {
			if e.ItemID == itemID {
				if qty <= 0 || e.Quantity <= qty {
					// Remove entry entirely.
					gs.PlayerInventory[pid] = append(entries[:i], entries[i+1:]...)
				} else {
					gs.PlayerInventory[pid][i].Quantity -= qty
				}
				break
			}
		}
	}
}

// RevealedFieldsForNPC returns the list of revealed field keys for a given NPC and player.
func (gs *GameState) RevealedFieldsForNPC(playerID, npcID string) []string {
	if gs.RevealedNPCFields == nil {
		return nil
	}
	npcMap := gs.RevealedNPCFields[playerID]
	if npcMap == nil {
		return nil
	}
	return npcMap[npcID]
}

// StateJSON serializes the GameState for state_sync broadcasts.
func (gs *GameState) StateJSON() json.RawMessage {
	data, _ := json.Marshal(gs)
	return data
}
