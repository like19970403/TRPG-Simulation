package realtime

import (
	"testing"
)

func testEvaluator(gs *GameState, sc *ScenarioContent, triggerPlayerID string, connectedPlayerIDs []string) *ExprEvaluator {
	return NewExprEvaluator(gs, sc, triggerPlayerID, connectedPlayerIDs)
}

func TestExprEvaluator_EmptyExpression(t *testing.T) {
	eval := testEvaluator(NewGameState("s1"), nil, "p1", nil)
	_, err := eval.Eval("")
	if err == nil {
		t.Fatal("expected error for empty expression")
	}
}

func TestExprEvaluator_InvalidSyntax(t *testing.T) {
	eval := testEvaluator(NewGameState("s1"), nil, "p1", nil)
	_, err := eval.Eval("((( invalid")
	if err == nil {
		t.Fatal("expected error for invalid syntax")
	}
}

func TestExprEvaluator_SimpleBoolLiteral(t *testing.T) {
	eval := testEvaluator(NewGameState("s1"), nil, "p1", nil)
	result, err := eval.Eval("true")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != true {
		t.Errorf("result = %v, want true", result)
	}
}

func TestExprEvaluator_SimpleArithmetic(t *testing.T) {
	eval := testEvaluator(NewGameState("s1"), nil, "p1", nil)
	result, err := eval.Eval("2 + 3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 5 {
		t.Errorf("result = %v, want 5", result)
	}
}

func TestExprEvaluator_HasItem_True(t *testing.T) {
	gs := NewGameState("s1")
	gs.RevealedItems = map[string][]string{
		"p1": {"rusty_key"},
	}
	eval := testEvaluator(gs, nil, "p1", []string{"p1"})
	result, err := eval.EvalBool("has_item('rusty_key')")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result {
		t.Error("expected true, got false")
	}
}

func TestExprEvaluator_HasItem_False(t *testing.T) {
	gs := NewGameState("s1")
	gs.RevealedItems = map[string][]string{
		"p1": {"other_item"},
	}
	eval := testEvaluator(gs, nil, "p1", []string{"p1"})
	result, err := eval.EvalBool("has_item('rusty_key')")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result {
		t.Error("expected false, got true")
	}
}

func TestExprEvaluator_HasItem_NoRevealedItems(t *testing.T) {
	gs := NewGameState("s1")
	eval := testEvaluator(gs, nil, "p1", []string{"p1"})
	result, err := eval.EvalBool("has_item('rusty_key')")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result {
		t.Error("expected false, got true")
	}
}

func TestExprEvaluator_Roll_ValidFormula(t *testing.T) {
	gs := NewGameState("s1")
	eval := testEvaluator(gs, nil, "p1", nil)
	result, err := eval.Eval("roll('1d6')")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	total, ok := result.(int)
	if !ok {
		t.Fatalf("result type = %T, want int", result)
	}
	if total < 1 || total > 6 {
		t.Errorf("total = %d, want 1-6", total)
	}
}

func TestExprEvaluator_Roll_InvalidFormula(t *testing.T) {
	gs := NewGameState("s1")
	eval := testEvaluator(gs, nil, "p1", nil)
	_, err := eval.Eval("roll('invalid')")
	if err == nil {
		t.Fatal("expected error for invalid formula")
	}
}

func TestExprEvaluator_Attr_Exists(t *testing.T) {
	sc := &ScenarioContent{
		Rules: &Rules{
			Attributes: []Attribute{
				{Name: "strength", Display: "力量", Default: 10},
				{Name: "perception", Display: "感知", Default: 12},
			},
		},
	}
	eval := testEvaluator(NewGameState("s1"), sc, "p1", nil)
	result, err := eval.Eval("attr('perception')")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 12 {
		t.Errorf("result = %v, want 12", result)
	}
}

func TestExprEvaluator_Attr_NotFound_ReturnsZero(t *testing.T) {
	sc := &ScenarioContent{
		Rules: &Rules{
			Attributes: []Attribute{
				{Name: "strength", Display: "力量", Default: 10},
			},
		},
	}
	eval := testEvaluator(NewGameState("s1"), sc, "p1", nil)
	result, err := eval.Eval("attr('nonexistent')")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 0 {
		t.Errorf("result = %v, want 0", result)
	}
}

func TestExprEvaluator_Attr_NoRules_ReturnsZero(t *testing.T) {
	eval := testEvaluator(NewGameState("s1"), nil, "p1", nil)
	result, err := eval.Eval("attr('strength')")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 0 {
		t.Errorf("result = %v, want 0", result)
	}
}

func TestExprEvaluator_Var_Exists(t *testing.T) {
	gs := NewGameState("s1")
	gs.Variables = map[string]any{"ghost_anger": float64(3)}
	eval := testEvaluator(gs, nil, "p1", nil)
	result, err := eval.Eval("var('ghost_anger')")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != float64(3) {
		t.Errorf("result = %v, want 3", result)
	}
}

func TestExprEvaluator_Var_NotFound_ReturnsNil(t *testing.T) {
	gs := NewGameState("s1")
	gs.Variables = map[string]any{}
	eval := testEvaluator(gs, nil, "p1", nil)
	result, err := eval.Eval("var('nonexistent')")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("result = %v, want nil", result)
	}
}

func TestExprEvaluator_Var_NilVariables(t *testing.T) {
	gs := NewGameState("s1")
	eval := testEvaluator(gs, nil, "p1", nil)
	result, err := eval.Eval("var('anything')")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("result = %v, want nil", result)
	}
}

func TestExprEvaluator_AllHaveItem_True(t *testing.T) {
	gs := NewGameState("s1")
	gs.RevealedItems = map[string][]string{
		"p1": {"key"},
		"p2": {"key"},
	}
	eval := testEvaluator(gs, nil, "p1", []string{"p1", "p2"})
	result, err := eval.EvalBool("all_have_item('key')")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result {
		t.Error("expected true, got false")
	}
}

func TestExprEvaluator_AllHaveItem_False_OneMissing(t *testing.T) {
	gs := NewGameState("s1")
	gs.RevealedItems = map[string][]string{
		"p1": {"key"},
	}
	eval := testEvaluator(gs, nil, "p1", []string{"p1", "p2"})
	result, err := eval.EvalBool("all_have_item('key')")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result {
		t.Error("expected false, got true")
	}
}

func TestExprEvaluator_AllHaveItem_NoPlayers(t *testing.T) {
	gs := NewGameState("s1")
	eval := testEvaluator(gs, nil, "p1", []string{})
	result, err := eval.EvalBool("all_have_item('key')")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result {
		t.Error("expected false for empty player list")
	}
}

func TestExprEvaluator_PlayerCount(t *testing.T) {
	eval := testEvaluator(NewGameState("s1"), nil, "p1", []string{"p1", "p2", "p3"})
	result, err := eval.Eval("player_count()")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 3 {
		t.Errorf("result = %v, want 3", result)
	}
}

func TestExprEvaluator_PlayerCount_Empty(t *testing.T) {
	eval := testEvaluator(NewGameState("s1"), nil, "p1", []string{})
	result, err := eval.Eval("player_count()")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 0 {
		t.Errorf("result = %v, want 0", result)
	}
}

func TestExprEvaluator_ComplexCondition_HasItemAndVar(t *testing.T) {
	gs := NewGameState("s1")
	gs.RevealedItems = map[string][]string{
		"p1": {"rusty_key"},
	}
	gs.Variables = map[string]any{"found_passage": true}
	eval := testEvaluator(gs, nil, "p1", []string{"p1"})
	result, err := eval.EvalBool("has_item('rusty_key') && var('found_passage') == true")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result {
		t.Error("expected true, got false")
	}
}

func TestExprEvaluator_ComplexCondition_RollPlusAttr(t *testing.T) {
	sc := &ScenarioContent{
		Rules: &Rules{
			Attributes: []Attribute{
				{Name: "perception", Display: "感知", Default: 100},
			},
		},
	}
	eval := testEvaluator(NewGameState("s1"), sc, "p1", nil)
	// roll('1d6') is 1-6, attr('perception') is 100, so sum >= 10 always true.
	result, err := eval.EvalBool("roll('1d6') + attr('perception') >= 10")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result {
		t.Error("expected true (roll + 100 >= 10)")
	}
}

func TestExprEvaluator_EvalBool_True(t *testing.T) {
	eval := testEvaluator(NewGameState("s1"), nil, "p1", nil)
	result, err := eval.EvalBool("1 + 1 == 2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result {
		t.Error("expected true")
	}
}

func TestExprEvaluator_EvalBool_False(t *testing.T) {
	eval := testEvaluator(NewGameState("s1"), nil, "p1", nil)
	result, err := eval.EvalBool("1 + 1 == 3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result {
		t.Error("expected false")
	}
}

func TestExprEvaluator_EvalBool_NonBoolResult(t *testing.T) {
	eval := testEvaluator(NewGameState("s1"), nil, "p1", nil)
	_, err := eval.EvalBool("1 + 1")
	if err == nil {
		t.Fatal("expected error for non-bool result")
	}
}
