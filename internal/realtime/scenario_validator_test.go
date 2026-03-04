package realtime

import (
	"testing"
)

func makeValidScenario() *ScenarioContent {
	return &ScenarioContent{
		Title:      "Test",
		StartScene: "scene1",
		Scenes: []Scene{
			{ID: "scene1", Name: "Scene 1", Transitions: []Transition{
				{Target: "scene2", Trigger: "gm_decision"},
			}},
			{ID: "scene2", Name: "Scene 2"},
		},
		Items: []Item{
			{ID: "key", Name: "Key", Type: "item", Description: "A key"},
		},
		NPCs: []NPC{
			{ID: "npc1", Name: "NPC 1", Fields: []NPCField{
				{Key: "secret", Label: "Secret", Value: "hidden", Visibility: "hidden"},
			}},
		},
		Variables: []Variable{
			{Name: "found_clue", Type: "bool", Default: false},
		},
	}
}

func TestValidateScenarioContent_Valid(t *testing.T) {
	sc := makeValidScenario()
	errs := ValidateScenarioContent(sc)
	if HasErrors(errs) {
		t.Errorf("expected no errors, got: %+v", errs)
	}
}

func TestValidateScenarioContent_InvalidStartScene(t *testing.T) {
	sc := makeValidScenario()
	sc.StartScene = "nonexistent"
	errs := ValidateScenarioContent(sc)
	found := false
	for _, e := range errs {
		if e.Code == "invalid_start_scene" && e.Severity == SeverityError {
			found = true
		}
	}
	if !found {
		t.Error("expected invalid_start_scene error")
	}
}

func TestValidateScenarioContent_InvalidTransitionTarget(t *testing.T) {
	sc := makeValidScenario()
	sc.Scenes[0].Transitions = []Transition{{Target: "nowhere", Trigger: "gm_decision"}}
	errs := ValidateScenarioContent(sc)
	found := false
	for _, e := range errs {
		if e.Code == "invalid_target" {
			found = true
		}
	}
	if !found {
		t.Error("expected invalid_target error")
	}
}

func TestValidateScenarioContent_SelfTransitionWarning(t *testing.T) {
	sc := makeValidScenario()
	sc.Scenes[0].Transitions = append(sc.Scenes[0].Transitions,
		Transition{Target: "scene1", Trigger: "gm_decision"})
	errs := ValidateScenarioContent(sc)
	found := false
	for _, e := range errs {
		if e.Code == "self_transition" && e.Severity == SeverityWarning {
			found = true
		}
	}
	if !found {
		t.Error("expected self_transition warning")
	}
}

func TestValidateScenarioContent_OrphanScene(t *testing.T) {
	sc := makeValidScenario()
	sc.Scenes = append(sc.Scenes, Scene{ID: "orphan", Name: "Orphan"})
	errs := ValidateScenarioContent(sc)
	found := false
	for _, e := range errs {
		if e.Code == "orphan_scene" && e.Severity == SeverityWarning {
			found = true
		}
	}
	if !found {
		t.Error("expected orphan_scene warning")
	}
}

func TestValidateScenarioContent_InvalidItemRef(t *testing.T) {
	sc := makeValidScenario()
	sc.Scenes[0].ItemsAvailable = []string{"nonexistent_item"}
	errs := ValidateScenarioContent(sc)
	found := false
	for _, e := range errs {
		if e.Code == "invalid_item_ref" {
			found = true
		}
	}
	if !found {
		t.Error("expected invalid_item_ref error")
	}
}

func TestValidateScenarioContent_InvalidNPCRef(t *testing.T) {
	sc := makeValidScenario()
	sc.Scenes[0].NPCsPresent = []string{"nonexistent_npc"}
	errs := ValidateScenarioContent(sc)
	found := false
	for _, e := range errs {
		if e.Code == "invalid_npc_ref" {
			found = true
		}
	}
	if !found {
		t.Error("expected invalid_npc_ref error")
	}
}

func TestValidateScenarioContent_ActionItemRef(t *testing.T) {
	sc := makeValidScenario()
	sc.Scenes[0].OnEnter = []Action{
		{GiveItem: &GiveItemAction{ItemID: "nonexistent", To: "all"}},
	}
	errs := ValidateScenarioContent(sc)
	found := false
	for _, e := range errs {
		if e.Code == "invalid_item_ref" {
			found = true
		}
	}
	if !found {
		t.Error("expected invalid_item_ref error for action")
	}
}

func TestValidateScenarioContent_ActionNPCFieldRef(t *testing.T) {
	sc := makeValidScenario()
	sc.Scenes[0].OnEnter = []Action{
		{RevealNPCField: &RevealNPCFieldAction{NPCID: "npc1", FieldKey: "nonexistent", To: "all"}},
	}
	errs := ValidateScenarioContent(sc)
	found := false
	for _, e := range errs {
		if e.Code == "invalid_field_ref" {
			found = true
		}
	}
	if !found {
		t.Error("expected invalid_field_ref error")
	}
}

func TestValidateScenarioContent_UndefinedVariable(t *testing.T) {
	sc := makeValidScenario()
	sc.Scenes[0].OnEnter = []Action{
		{SetVar: &SetVarAction{Name: "unknown_var", Value: true}},
	}
	errs := ValidateScenarioContent(sc)
	found := false
	for _, e := range errs {
		if e.Code == "undefined_variable" && e.Severity == SeverityWarning {
			found = true
		}
	}
	if !found {
		t.Error("expected undefined_variable warning")
	}
}

func TestValidateScenarioContent_InvalidExpression(t *testing.T) {
	sc := makeValidScenario()
	sc.Scenes[0].Transitions = []Transition{
		{Target: "scene2", Trigger: "condition_met", Condition: "invalid +++"},
	}
	errs := ValidateScenarioContent(sc)
	found := false
	for _, e := range errs {
		if e.Code == "invalid_expression" {
			found = true
		}
	}
	if !found {
		t.Error("expected invalid_expression error")
	}
}

func TestValidateScenarioContent_ValidExpression(t *testing.T) {
	sc := makeValidScenario()
	sc.Scenes[0].Transitions = []Transition{
		{Target: "scene2", Trigger: "condition_met", Condition: "has_item(\"key\") && var(\"found_clue\") == true"},
	}
	errs := ValidateScenarioContent(sc)
	for _, e := range errs {
		if e.Code == "invalid_expression" {
			t.Errorf("unexpected invalid_expression error: %s", e.Message)
		}
	}
}

func TestValidateScenarioContent_SetVarInvalidExpr(t *testing.T) {
	sc := makeValidScenario()
	sc.Scenes[0].OnEnter = []Action{
		{SetVar: &SetVarAction{Name: "found_clue", Expr: "(("}},
	}
	errs := ValidateScenarioContent(sc)
	found := false
	for _, e := range errs {
		if e.Code == "invalid_expression" {
			found = true
		}
	}
	if !found {
		t.Error("expected invalid_expression error for set_var expr")
	}
}

func TestValidateScenarioContent_MissingSceneFields(t *testing.T) {
	sc := &ScenarioContent{
		StartScene: "s1",
		Scenes:     []Scene{{ID: "", Name: ""}, {ID: "s1", Name: "S1"}},
	}
	errs := ValidateScenarioContent(sc)
	idErr, nameErr := false, false
	for _, e := range errs {
		if e.Code == "missing_id" {
			idErr = true
		}
		if e.Code == "missing_name" {
			nameErr = true
		}
	}
	if !idErr {
		t.Error("expected missing_id error")
	}
	if !nameErr {
		t.Error("expected missing_name error")
	}
}

func TestHasErrors(t *testing.T) {
	warnings := []ValidationError{{Severity: SeverityWarning}}
	if HasErrors(warnings) {
		t.Error("expected no errors for warnings only")
	}
	errors := []ValidationError{{Severity: SeverityError}}
	if !HasErrors(errors) {
		t.Error("expected HasErrors to return true")
	}
}
