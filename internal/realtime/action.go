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
// evaluator is optional (nil = no expression evaluation, backward compatible).
func executeActions(actions []Action, triggerPlayerID string, connectedPlayerIDs []string, currentVars map[string]any, evaluator *ExprEvaluator) ([]actionResult, error) {
	var results []actionResult

	for _, action := range actions {
		switch {
		case action.SetVar != nil:
			r, err := executeSetVar(action.SetVar, currentVars, evaluator)
			if err != nil {
				return nil, fmt.Errorf("action set_var: %w", err)
			}
			results = append(results, r)
			// Update currentVars with the evaluated new_value from the result payload.
			var m map[string]any
			json.Unmarshal(r.payload, &m)
			if currentVars == nil {
				currentVars = make(map[string]any)
			}
			currentVars[action.SetVar.Name] = m["new_value"]

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

		case action.GiveItem != nil:
			r, err := executeGiveItem(action.GiveItem, triggerPlayerID, connectedPlayerIDs)
			if err != nil {
				return nil, fmt.Errorf("action give_item: %w", err)
			}
			results = append(results, r)

		case action.RemoveItem != nil:
			r, err := executeRemoveItem(action.RemoveItem, triggerPlayerID, connectedPlayerIDs)
			if err != nil {
				return nil, fmt.Errorf("action remove_item: %w", err)
			}
			results = append(results, r)
		}
	}

	return results, nil
}

func executeSetVar(sv *SetVarAction, currentVars map[string]any, evaluator *ExprEvaluator) (actionResult, error) {
	if sv.Name == "" {
		return actionResult{}, fmt.Errorf("name is required")
	}

	var oldValue any
	if currentVars != nil {
		oldValue = currentVars[sv.Name]
	}

	newValue := sv.Value

	// If Expr is set, evaluate it and use the result as the new value.
	if sv.Expr != "" && evaluator != nil {
		result, err := evaluator.Eval(sv.Expr)
		if err != nil {
			return actionResult{}, fmt.Errorf("set_var expr evaluation failed: %w", err)
		}
		newValue = result
	}

	payload, _ := json.Marshal(map[string]any{
		"name":      sv.Name,
		"old_value": oldValue,
		"new_value": newValue,
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

func executeGiveItem(gi *GiveItemAction, triggerPlayerID string, connectedPlayerIDs []string) (actionResult, error) {
	if gi.ItemID == "" {
		return actionResult{}, fmt.Errorf("item_id is required")
	}

	playerIDs := resolveTargetPlayers(gi.To, triggerPlayerID, connectedPlayerIDs)

	qty := gi.Quantity
	if qty <= 0 {
		qty = 1
	}

	payload, _ := json.Marshal(map[string]any{
		"item_id":    gi.ItemID,
		"player_ids": playerIDs,
		"quantity":   qty,
		"method":     "on_enter",
	})

	return actionResult{
		eventType: EventItemGiven,
		payload:   payload,
	}, nil
}

func executeRemoveItem(ri *RemoveItemAction, triggerPlayerID string, connectedPlayerIDs []string) (actionResult, error) {
	if ri.ItemID == "" {
		return actionResult{}, fmt.Errorf("item_id is required")
	}

	playerIDs := resolveTargetPlayers(ri.From, triggerPlayerID, connectedPlayerIDs)

	payload, _ := json.Marshal(map[string]any{
		"item_id":    ri.ItemID,
		"player_ids": playerIDs,
		"quantity":   ri.Quantity,
		"method":     "on_enter",
	})

	return actionResult{
		eventType: EventItemRemoved,
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
