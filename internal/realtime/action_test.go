package realtime

import (
	"encoding/json"
	"testing"
)

func TestExecuteActions_Empty(t *testing.T) {
	results, err := executeActions(nil, "p1", []string{"p1", "p2"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("len(results) = %d, want 0", len(results))
	}
}

func TestExecuteActions_SetVar_LiteralBool(t *testing.T) {
	actions := []Action{
		{SetVar: &SetVarAction{Name: "found_key", Value: true}},
	}
	results, err := executeActions(actions, "p1", nil, map[string]any{"found_key": false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].eventType != EventVariableChanged {
		t.Errorf("eventType = %q, want %q", results[0].eventType, EventVariableChanged)
	}
}

func TestExecuteActions_SetVar_LiteralInt(t *testing.T) {
	actions := []Action{
		{SetVar: &SetVarAction{Name: "anger", Value: float64(5)}},
	}
	results, err := executeActions(actions, "p1", nil, map[string]any{"anger": float64(0)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}

	var m map[string]any
	json.Unmarshal(results[0].payload, &m)
	if m["new_value"] != float64(5) {
		t.Errorf("new_value = %v, want 5", m["new_value"])
	}
}

func TestExecuteActions_SetVar_LiteralString(t *testing.T) {
	actions := []Action{
		{SetVar: &SetVarAction{Name: "ally", Value: "Alice"}},
	}
	results, err := executeActions(actions, "p1", nil, map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]any
	json.Unmarshal(results[0].payload, &m)
	if m["new_value"] != "Alice" {
		t.Errorf("new_value = %v, want 'Alice'", m["new_value"])
	}
}

func TestExecuteActions_SetVar_ProducesVariableChangedEvent(t *testing.T) {
	actions := []Action{
		{SetVar: &SetVarAction{Name: "visited", Value: true}},
	}
	results, err := executeActions(actions, "p1", nil, map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if results[0].eventType != EventVariableChanged {
		t.Errorf("eventType = %q, want %q", results[0].eventType, EventVariableChanged)
	}

	var m map[string]any
	json.Unmarshal(results[0].payload, &m)
	if m["name"] != "visited" {
		t.Errorf("name = %v, want 'visited'", m["name"])
	}
}

func TestExecuteActions_SetVar_OldValueFromCurrentVars(t *testing.T) {
	actions := []Action{
		{SetVar: &SetVarAction{Name: "anger", Value: float64(3)}},
	}
	results, err := executeActions(actions, "p1", nil, map[string]any{"anger": float64(1)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]any
	json.Unmarshal(results[0].payload, &m)
	if m["old_value"] != float64(1) {
		t.Errorf("old_value = %v, want 1", m["old_value"])
	}
	if m["new_value"] != float64(3) {
		t.Errorf("new_value = %v, want 3", m["new_value"])
	}
}

func TestExecuteActions_RevealItem_CurrentPlayer(t *testing.T) {
	actions := []Action{
		{RevealItem: &RevealItemAction{ItemID: "key", To: "current_player"}},
	}
	results, err := executeActions(actions, "p1", []string{"p1", "p2"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]any
	json.Unmarshal(results[0].payload, &m)
	pids := m["player_ids"].([]any)
	if len(pids) != 1 || pids[0] != "p1" {
		t.Errorf("player_ids = %v, want [p1]", pids)
	}
}

func TestExecuteActions_RevealItem_All(t *testing.T) {
	actions := []Action{
		{RevealItem: &RevealItemAction{ItemID: "key", To: "all"}},
	}
	results, err := executeActions(actions, "p1", []string{"p1", "p2", "p3"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]any
	json.Unmarshal(results[0].payload, &m)
	pids := m["player_ids"].([]any)
	if len(pids) != 3 {
		t.Errorf("len(player_ids) = %d, want 3", len(pids))
	}
}

func TestExecuteActions_RevealItem_SpecificPlayerID(t *testing.T) {
	actions := []Action{
		{RevealItem: &RevealItemAction{ItemID: "key", To: "player-42"}},
	}
	results, err := executeActions(actions, "p1", []string{"p1", "p2"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]any
	json.Unmarshal(results[0].payload, &m)
	pids := m["player_ids"].([]any)
	if len(pids) != 1 || pids[0] != "player-42" {
		t.Errorf("player_ids = %v, want [player-42]", pids)
	}
}

func TestExecuteActions_RevealItem_ProducesItemRevealedEvent(t *testing.T) {
	actions := []Action{
		{RevealItem: &RevealItemAction{ItemID: "diary", To: "current_player"}},
	}
	results, err := executeActions(actions, "p1", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if results[0].eventType != EventItemRevealed {
		t.Errorf("eventType = %q, want %q", results[0].eventType, EventItemRevealed)
	}

	var m map[string]any
	json.Unmarshal(results[0].payload, &m)
	if m["item_id"] != "diary" {
		t.Errorf("item_id = %v, want 'diary'", m["item_id"])
	}
	if m["method"] != "on_enter" {
		t.Errorf("method = %v, want 'on_enter'", m["method"])
	}
}

func TestExecuteActions_RevealNPCField_CurrentPlayer(t *testing.T) {
	actions := []Action{
		{RevealNPCField: &RevealNPCFieldAction{NPCID: "butler", FieldKey: "secret", To: "current_player"}},
	}
	results, err := executeActions(actions, "p1", []string{"p1", "p2"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]any
	json.Unmarshal(results[0].payload, &m)
	pids := m["player_ids"].([]any)
	if len(pids) != 1 || pids[0] != "p1" {
		t.Errorf("player_ids = %v, want [p1]", pids)
	}
}

func TestExecuteActions_RevealNPCField_All(t *testing.T) {
	actions := []Action{
		{RevealNPCField: &RevealNPCFieldAction{NPCID: "butler", FieldKey: "secret", To: "all"}},
	}
	results, err := executeActions(actions, "p1", []string{"p1", "p2"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]any
	json.Unmarshal(results[0].payload, &m)
	pids := m["player_ids"].([]any)
	if len(pids) != 2 {
		t.Errorf("len(player_ids) = %d, want 2", len(pids))
	}
}

func TestExecuteActions_RevealNPCField_ProducesNPCFieldRevealedEvent(t *testing.T) {
	actions := []Action{
		{RevealNPCField: &RevealNPCFieldAction{NPCID: "butler", FieldKey: "personality", To: "p1"}},
	}
	results, err := executeActions(actions, "p1", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if results[0].eventType != EventNPCFieldRevealed {
		t.Errorf("eventType = %q, want %q", results[0].eventType, EventNPCFieldRevealed)
	}

	var m map[string]any
	json.Unmarshal(results[0].payload, &m)
	if m["npc_id"] != "butler" {
		t.Errorf("npc_id = %v, want 'butler'", m["npc_id"])
	}
	if m["field_key"] != "personality" {
		t.Errorf("field_key = %v, want 'personality'", m["field_key"])
	}
}

func TestExecuteActions_MultipleActions(t *testing.T) {
	actions := []Action{
		{SetVar: &SetVarAction{Name: "visited", Value: true}},
		{RevealItem: &RevealItemAction{ItemID: "key", To: "all"}},
	}
	results, err := executeActions(actions, "p1", []string{"p1"}, map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	if results[0].eventType != EventVariableChanged {
		t.Errorf("results[0].eventType = %q, want %q", results[0].eventType, EventVariableChanged)
	}
	if results[1].eventType != EventItemRevealed {
		t.Errorf("results[1].eventType = %q, want %q", results[1].eventType, EventItemRevealed)
	}
}

func TestExecuteActions_SetVar_MissingName(t *testing.T) {
	actions := []Action{
		{SetVar: &SetVarAction{Name: "", Value: true}},
	}
	_, err := executeActions(actions, "p1", nil, nil)
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}
