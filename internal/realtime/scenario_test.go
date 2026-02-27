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
