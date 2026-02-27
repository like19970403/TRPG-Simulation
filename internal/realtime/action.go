package realtime

import (
	"encoding/json"
	"fmt"
)

// actionResult represents a single event produced by executing a scene action.
type actionResult struct {
	eventType string
	payload   json.RawMessage
}

// executeActions processes a list of scene actions and returns the event results.
// triggerPlayerID is the player who triggered the scene transition (for "current_player" targeting).
// connectedPlayerIDs is the list of all connected player IDs (for "all" targeting).
// currentVars holds the current scenario variables (for old_value in set_var events).
func executeActions(actions []Action, triggerPlayerID string, connectedPlayerIDs []string, currentVars map[string]any) ([]actionResult, error) {
	var results []actionResult

	for _, action := range actions {
		switch {
		case action.SetVar != nil:
			r, err := executeSetVar(action.SetVar, currentVars)
			if err != nil {
				return nil, fmt.Errorf("action set_var: %w", err)
			}
			results = append(results, r)
			// Update currentVars so subsequent actions see the new value.
			currentVars[action.SetVar.Name] = action.SetVar.Value

		case action.RevealItem != nil:
			r, err := executeRevealItem(action.RevealItem, triggerPlayerID, connectedPlayerIDs)
			if err != nil {
				return nil, fmt.Errorf("action reveal_item: %w", err)
			}
			results = append(results, r)

		case action.RevealNPCField != nil:
			r, err := executeRevealNPCField(action.RevealNPCField, triggerPlayerID, connectedPlayerIDs)
			if err != nil {
				return nil, fmt.Errorf("action reveal_npc_field: %w", err)
			}
			results = append(results, r)
		}
	}

	return results, nil
}

func executeSetVar(sv *SetVarAction, currentVars map[string]any) (actionResult, error) {
	if sv.Name == "" {
		return actionResult{}, fmt.Errorf("name is required")
	}

	var oldValue any
	if currentVars != nil {
		oldValue = currentVars[sv.Name]
	}

	payload, _ := json.Marshal(map[string]any{
		"name":      sv.Name,
		"old_value": oldValue,
		"new_value": sv.Value,
	})

	return actionResult{
		eventType: EventVariableChanged,
		payload:   payload,
	}, nil
}

func executeRevealItem(ri *RevealItemAction, triggerPlayerID string, connectedPlayerIDs []string) (actionResult, error) {
	if ri.ItemID == "" {
		return actionResult{}, fmt.Errorf("item_id is required")
	}

	playerIDs := resolveTargetPlayers(ri.To, triggerPlayerID, connectedPlayerIDs)

	payload, _ := json.Marshal(map[string]any{
		"item_id":    ri.ItemID,
		"player_ids": playerIDs,
		"method":     "on_enter",
	})

	return actionResult{
		eventType: EventItemRevealed,
		payload:   payload,
	}, nil
}

func executeRevealNPCField(rnf *RevealNPCFieldAction, triggerPlayerID string, connectedPlayerIDs []string) (actionResult, error) {
	if rnf.NPCID == "" {
		return actionResult{}, fmt.Errorf("npc_id is required")
	}
	if rnf.FieldKey == "" {
		return actionResult{}, fmt.Errorf("field_key is required")
	}

	playerIDs := resolveTargetPlayers(rnf.To, triggerPlayerID, connectedPlayerIDs)

	payload, _ := json.Marshal(map[string]any{
		"npc_id":     rnf.NPCID,
		"field_key":  rnf.FieldKey,
		"player_ids": playerIDs,
	})

	return actionResult{
		eventType: EventNPCFieldRevealed,
		payload:   payload,
	}, nil
}

// resolveTargetPlayers resolves the "to" field into a list of player IDs.
func resolveTargetPlayers(to, triggerPlayerID string, connectedPlayerIDs []string) []string {
	switch to {
	case "current_player":
		if triggerPlayerID != "" {
			return []string{triggerPlayerID}
		}
		return []string{}
	case "all":
		return connectedPlayerIDs
	default:
		// Treat as a specific player ID.
		if to != "" {
			return []string{to}
		}
		return []string{}
	}
}
