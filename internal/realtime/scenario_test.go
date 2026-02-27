package realtime

import (
	"encoding/json"
	"testing"
)

func TestParseScenarioContent_Valid(t *testing.T) {
	raw := json.RawMessage(`{
		"id": "test-scenario",
		"title": "Test",
		"start_scene": "entrance",
		"scenes": [
			{"id": "entrance", "name": "Entrance", "content": "You enter."},
			{"id": "library", "name": "Library", "content": "Books everywhere."}
		],
		"items": [{"id": "key", "name": "Key", "type": "item", "description": "A rusty key"}],
		"npcs": [{"id": "butler", "name": "Butler", "fields": [
			{"key": "appearance", "label": "Appearance", "value": "Tall", "visibility": "public"}
		]}],
		"variables": [{"name": "found_key", "type": "bool", "default": false}],
		"rules": {"dice_formula": "2d6", "attributes": [{"name": "str", "display": "Strength", "default": 10}]}
	}`)

	sc, err := ParseScenarioContent(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sc.ID != "test-scenario" {
		t.Errorf("ID = %q, want %q", sc.ID, "test-scenario")
	}
	if sc.Title != "Test" {
		t.Errorf("Title = %q, want %q", sc.Title, "Test")
	}
	if sc.StartScene != "entrance" {
		t.Errorf("StartScene = %q, want %q", sc.StartScene, "entrance")
	}
	if len(sc.Scenes) != 2 {
		t.Errorf("len(Scenes) = %d, want 2", len(sc.Scenes))
	}
	if len(sc.Items) != 1 {
		t.Errorf("len(Items) = %d, want 1", len(sc.Items))
	}
	if len(sc.NPCs) != 1 {
		t.Errorf("len(NPCs) = %d, want 1", len(sc.NPCs))
	}
	if len(sc.Variables) != 1 {
		t.Errorf("len(Variables) = %d, want 1", len(sc.Variables))
	}
	if sc.Rules == nil {
		t.Fatal("Rules should not be nil")
	}
	if sc.Rules.DiceFormula != "2d6" {
		t.Errorf("Rules.DiceFormula = %q, want %q", sc.Rules.DiceFormula, "2d6")
	}
	if len(sc.Rules.Attributes) != 1 {
		t.Errorf("len(Rules.Attributes) = %d, want 1", len(sc.Rules.Attributes))
	}
}

func TestParseScenarioContent_MinimalValid(t *testing.T) {
	raw := json.RawMessage(`{
		"start_scene": "s1",
		"scenes": [{"id": "s1", "name": "Start", "content": "Begin"}]
	}`)

	sc, err := ParseScenarioContent(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sc.StartScene != "s1" {
		t.Errorf("StartScene = %q, want %q", sc.StartScene, "s1")
	}
	if len(sc.Scenes) != 1 {
		t.Errorf("len(Scenes) = %d, want 1", len(sc.Scenes))
	}
	if sc.Items != nil {
		t.Errorf("Items should be nil, got %v", sc.Items)
	}
	if sc.Rules != nil {
		t.Errorf("Rules should be nil, got %v", sc.Rules)
	}
}

func TestParseScenarioContent_InvalidJSON(t *testing.T) {
	raw := json.RawMessage(`{not valid json}`)

	_, err := ParseScenarioContent(raw)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseScenarioContent_EmptyContent(t *testing.T) {
	_, err := ParseScenarioContent(nil)
	if err == nil {
		t.Fatal("expected error for empty content")
	}
}

func TestParseScenarioContent_MissingStartScene(t *testing.T) {
	raw := json.RawMessage(`{
		"scenes": [{"id": "s1", "name": "Start", "content": "Begin"}]
	}`)

	_, err := ParseScenarioContent(raw)
	if err == nil {
		t.Fatal("expected error for missing start_scene")
	}
}

func TestParseScenarioContent_NoScenes(t *testing.T) {
	raw := json.RawMessage(`{
		"start_scene": "s1",
		"scenes": []
	}`)

	_, err := ParseScenarioContent(raw)
	if err == nil {
		t.Fatal("expected error for no scenes")
	}
}

func TestFindScene_Exists(t *testing.T) {
	sc := &ScenarioContent{
		Scenes: []Scene{
			{ID: "entrance", Name: "Entrance"},
			{ID: "library", Name: "Library"},
		},
	}

	scene := sc.FindScene("library")
	if scene == nil {
		t.Fatal("expected to find scene 'library'")
	}
	if scene.Name != "Library" {
		t.Errorf("Name = %q, want %q", scene.Name, "Library")
	}
}

func TestFindScene_NotFound(t *testing.T) {
	sc := &ScenarioContent{
		Scenes: []Scene{
			{ID: "entrance", Name: "Entrance"},
		},
	}

	scene := sc.FindScene("nonexistent")
	if scene != nil {
		t.Errorf("expected nil for nonexistent scene, got %v", scene)
	}
}

func TestParseScenarioContent_WithGMNotes(t *testing.T) {
	raw := json.RawMessage(`{
		"start_scene": "s1",
		"scenes": [{
			"id": "s1",
			"name": "Start",
			"content": "Begin",
			"gm_notes": "Secret GM info"
		}]
	}`)

	sc, err := ParseScenarioContent(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sc.Scenes[0].GMNotes != "Secret GM info" {
		t.Errorf("GMNotes = %q, want %q", sc.Scenes[0].GMNotes, "Secret GM info")
	}
}

func TestParseScenarioContent_WithTransitions(t *testing.T) {
	raw := json.RawMessage(`{
		"start_scene": "s1",
		"scenes": [{
			"id": "s1",
			"name": "Start",
			"content": "Begin",
			"transitions": [
				{"target": "s2", "trigger": "player_choice", "label": "Go to s2"},
				{"target": "s3", "trigger": "condition_met", "condition": "has_item('key')", "label": "Secret path"}
			]
		}, {"id": "s2", "name": "S2", "content": "S2"},
		   {"id": "s3", "name": "S3", "content": "S3"}]
	}`)

	sc, err := ParseScenarioContent(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sc.Scenes[0].Transitions) != 2 {
		t.Fatalf("len(Transitions) = %d, want 2", len(sc.Scenes[0].Transitions))
	}

	tr := sc.Scenes[0].Transitions[1]
	if tr.Target != "s3" {
		t.Errorf("Target = %q, want %q", tr.Target, "s3")
	}
	if tr.Condition != "has_item('key')" {
		t.Errorf("Condition = %q, want %q", tr.Condition, "has_item('key')")
	}
}

func TestParseScenarioContent_WithItems(t *testing.T) {
	raw := json.RawMessage(`{
		"start_scene": "s1",
		"scenes": [{"id": "s1", "name": "S1", "content": "C1", "items_available": ["key"]}],
		"items": [
			{"id": "key", "name": "Rusty Key", "type": "item", "description": "A rusty key", "image": "key.jpg"}
		]
	}`)

	sc, err := ParseScenarioContent(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sc.Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1", len(sc.Items))
	}
	if sc.Items[0].Image != "key.jpg" {
		t.Errorf("Image = %q, want %q", sc.Items[0].Image, "key.jpg")
	}
	if len(sc.Scenes[0].ItemsAvailable) != 1 {
		t.Errorf("len(ItemsAvailable) = %d, want 1", len(sc.Scenes[0].ItemsAvailable))
	}
}

func TestParseScenarioContent_WithOnEnter(t *testing.T) {
	raw := json.RawMessage(`{
		"start_scene": "s1",
		"scenes": [{
			"id": "s1", "name": "Start", "content": "Begin",
			"on_enter": [
				{"set_var": {"name": "visited", "value": true}}
			]
		}]
	}`)

	sc, err := ParseScenarioContent(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sc.Scenes[0].OnEnter) != 1 {
		t.Fatalf("len(OnEnter) = %d, want 1", len(sc.Scenes[0].OnEnter))
	}
	if sc.Scenes[0].OnEnter[0].SetVar == nil {
		t.Fatal("expected SetVar action, got nil")
	}
	if sc.Scenes[0].OnEnter[0].SetVar.Name != "visited" {
		t.Errorf("SetVar.Name = %q, want %q", sc.Scenes[0].OnEnter[0].SetVar.Name, "visited")
	}
}

func TestParseScenarioContent_WithOnExit(t *testing.T) {
	raw := json.RawMessage(`{
		"start_scene": "s1",
		"scenes": [{
			"id": "s1", "name": "Start", "content": "Begin",
			"on_exit": [
				{"set_var": {"name": "left_scene", "value": true}}
			]
		}]
	}`)

	sc, err := ParseScenarioContent(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sc.Scenes[0].OnExit) != 1 {
		t.Fatalf("len(OnExit) = %d, want 1", len(sc.Scenes[0].OnExit))
	}
	if sc.Scenes[0].OnExit[0].SetVar == nil {
		t.Fatal("expected SetVar action, got nil")
	}
}

func TestParseScenarioContent_WithOnEnterSetVar(t *testing.T) {
	raw := json.RawMessage(`{
		"start_scene": "s1",
		"scenes": [{
			"id": "s1", "name": "Start", "content": "Begin",
			"on_enter": [{"set_var": {"name": "anger", "value": 5}}]
		}]
	}`)

	sc, err := ParseScenarioContent(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sv := sc.Scenes[0].OnEnter[0].SetVar
	if sv.Name != "anger" {
		t.Errorf("Name = %q, want %q", sv.Name, "anger")
	}
	// JSON numbers decode as float64.
	if sv.Value != float64(5) {
		t.Errorf("Value = %v (%T), want 5", sv.Value, sv.Value)
	}
}

func TestParseScenarioContent_WithOnEnterRevealItem(t *testing.T) {
	raw := json.RawMessage(`{
		"start_scene": "s1",
		"scenes": [{
			"id": "s1", "name": "Start", "content": "Begin",
			"on_enter": [{"reveal_item": {"item_id": "key", "to": "current_player"}}]
		}]
	}`)

	sc, err := ParseScenarioContent(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ri := sc.Scenes[0].OnEnter[0].RevealItem
	if ri == nil {
		t.Fatal("expected RevealItem action, got nil")
	}
	if ri.ItemID != "key" {
		t.Errorf("ItemID = %q, want %q", ri.ItemID, "key")
	}
	if ri.To != "current_player" {
		t.Errorf("To = %q, want %q", ri.To, "current_player")
	}
}

func TestParseScenarioContent_WithOnEnterRevealNPCField(t *testing.T) {
	raw := json.RawMessage(`{
		"start_scene": "s1",
		"scenes": [{
			"id": "s1", "name": "Start", "content": "Begin",
			"on_enter": [{"reveal_npc_field": {"npc_id": "butler", "field_key": "secret", "to": "all"}}]
		}]
	}`)

	sc, err := ParseScenarioContent(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rnf := sc.Scenes[0].OnEnter[0].RevealNPCField
	if rnf == nil {
		t.Fatal("expected RevealNPCField action, got nil")
	}
	if rnf.NPCID != "butler" {
		t.Errorf("NPCID = %q, want %q", rnf.NPCID, "butler")
	}
	if rnf.FieldKey != "secret" {
		t.Errorf("FieldKey = %q, want %q", rnf.FieldKey, "secret")
	}
	if rnf.To != "all" {
		t.Errorf("To = %q, want %q", rnf.To, "all")
	}
}

func TestParseScenarioContent_MultipleOnEnterActions(t *testing.T) {
	raw := json.RawMessage(`{
		"start_scene": "s1",
		"scenes": [{
			"id": "s1", "name": "Start", "content": "Begin",
			"on_enter": [
				{"set_var": {"name": "visited", "value": true}},
				{"reveal_item": {"item_id": "diary", "to": "current_player"}},
				{"reveal_npc_field": {"npc_id": "butler", "field_key": "secret", "to": "all"}}
			]
		}]
	}`)

	sc, err := ParseScenarioContent(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sc.Scenes[0].OnEnter) != 3 {
		t.Fatalf("len(OnEnter) = %d, want 3", len(sc.Scenes[0].OnEnter))
	}
	if sc.Scenes[0].OnEnter[0].SetVar == nil {
		t.Error("expected SetVar at index 0")
	}
	if sc.Scenes[0].OnEnter[1].RevealItem == nil {
		t.Error("expected RevealItem at index 1")
	}
	if sc.Scenes[0].OnEnter[2].RevealNPCField == nil {
		t.Error("expected RevealNPCField at index 2")
	}
}

func TestFindItem_Exists(t *testing.T) {
	sc := &ScenarioContent{
		Items: []Item{
			{ID: "key", Name: "Rusty Key"},
			{ID: "diary", Name: "Torn Diary"},
		},
	}

	item := sc.FindItem("diary")
	if item == nil {
		t.Fatal("expected to find item 'diary'")
	}
	if item.Name != "Torn Diary" {
		t.Errorf("Name = %q, want %q", item.Name, "Torn Diary")
	}
}

func TestFindItem_NotFound(t *testing.T) {
	sc := &ScenarioContent{
		Items: []Item{{ID: "key", Name: "Key"}},
	}

	item := sc.FindItem("nonexistent")
	if item != nil {
		t.Errorf("expected nil for nonexistent item, got %v", item)
	}
}

func TestFindNPC_Exists(t *testing.T) {
	sc := &ScenarioContent{
		NPCs: []NPC{
			{ID: "butler", Name: "Old Butler"},
			{ID: "ghost", Name: "Ghost Child"},
		},
	}

	npc := sc.FindNPC("ghost")
	if npc == nil {
		t.Fatal("expected to find NPC 'ghost'")
	}
	if npc.Name != "Ghost Child" {
		t.Errorf("Name = %q, want %q", npc.Name, "Ghost Child")
	}
}

func TestFindNPC_NotFound(t *testing.T) {
	sc := &ScenarioContent{
		NPCs: []NPC{{ID: "butler", Name: "Butler"}},
	}

	npc := sc.FindNPC("nonexistent")
	if npc != nil {
		t.Errorf("expected nil for nonexistent NPC, got %v", npc)
	}
}

func TestNPCFindField_Exists(t *testing.T) {
	npc := &NPC{
		ID:   "butler",
		Name: "Butler",
		Fields: []NPCField{
			{Key: "appearance", Label: "Appearance", Value: "Tall", Visibility: "public"},
			{Key: "secret", Label: "Secret", Value: "Ghost", Visibility: "hidden"},
		},
	}

	field := npc.FindField("secret")
	if field == nil {
		t.Fatal("expected to find field 'secret'")
	}
	if field.Value != "Ghost" {
		t.Errorf("Value = %q, want %q", field.Value, "Ghost")
	}
	if field.Visibility != "hidden" {
		t.Errorf("Visibility = %q, want %q", field.Visibility, "hidden")
	}
}

func TestNPCFindField_NotFound(t *testing.T) {
	npc := &NPC{
		ID:     "butler",
		Name:   "Butler",
		Fields: []NPCField{{Key: "appearance", Label: "Appearance", Value: "Tall", Visibility: "public"}},
	}

	field := npc.FindField("nonexistent")
	if field != nil {
		t.Errorf("expected nil for nonexistent field, got %v", field)
	}
}

func TestParseScenarioContent_WithNPCs(t *testing.T) {
	raw := json.RawMessage(`{
		"start_scene": "s1",
		"scenes": [{"id": "s1", "name": "S1", "content": "C1", "npcs_present": ["butler"]}],
		"npcs": [{
			"id": "butler",
			"name": "Old Butler",
			"image": "butler.jpg",
			"fields": [
				{"key": "appearance", "label": "Appearance", "value": "Tall and thin", "visibility": "public"},
				{"key": "secret", "label": "Secret", "value": "He is a ghost", "visibility": "hidden"}
			]
		}]
	}`)

	sc, err := ParseScenarioContent(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sc.NPCs) != 1 {
		t.Fatalf("len(NPCs) = %d, want 1", len(sc.NPCs))
	}
	npc := sc.NPCs[0]
	if npc.Image != "butler.jpg" {
		t.Errorf("Image = %q, want %q", npc.Image, "butler.jpg")
	}
	if len(npc.Fields) != 2 {
		t.Fatalf("len(Fields) = %d, want 2", len(npc.Fields))
	}
	if npc.Fields[0].Visibility != "public" {
		t.Errorf("Visibility = %q, want %q", npc.Fields[0].Visibility, "public")
	}
	if npc.Fields[1].Visibility != "hidden" {
		t.Errorf("Visibility = %q, want %q", npc.Fields[1].Visibility, "hidden")
	}
}
