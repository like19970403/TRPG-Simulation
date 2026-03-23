package realtime

import (
	"context"
	"fmt"
	"time"

	"github.com/expr-lang/expr"
)

// maxExprTimeout is the maximum duration for expression evaluation (ADR-003).
const maxExprTimeout = 100 * time.Millisecond

// ExprEvaluator evaluates expr-lang/expr expressions with game-context functions injected.
// Per ADR-003, 7 custom functions are available: has_item, item_count, all_have_item, roll, attr, var, player_count.
type ExprEvaluator struct {
	gameState          *GameState
	scenario           *ScenarioContent
	triggerPlayerID    string
	connectedPlayerIDs []string
}

// NewExprEvaluator creates an evaluator bound to the current game context.
func NewExprEvaluator(
	gameState *GameState,
	scenario *ScenarioContent,
	triggerPlayerID string,
	connectedPlayerIDs []string,
) *ExprEvaluator {
	return &ExprEvaluator{
		gameState:          gameState,
		scenario:           scenario,
		triggerPlayerID:    triggerPlayerID,
		connectedPlayerIDs: connectedPlayerIDs,
	}
}

// Eval compiles and evaluates the given expression string, returning the result.
// Enforces a 100ms timeout.
// Game variables are injected into the environment so bare identifiers work
// (e.g. `持有鑰匙 == true`) in addition to `var('持有鑰匙') == true`.
func (e *ExprEvaluator) Eval(expression string) (any, error) {
	if expression == "" {
		return nil, fmt.Errorf("expr: empty expression")
	}

	// Build environment map from game variables so bare identifiers resolve.
	env := make(map[string]any)
	if e.gameState.Variables != nil {
		for k, v := range e.gameState.Variables {
			env[k] = v
		}
	}

	opts := e.buildOptions()
	opts = append(opts, expr.Env(env))

	program, err := expr.Compile(expression, opts...)
	if err != nil {
		return nil, fmt.Errorf("expr: compile error: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), maxExprTimeout)
	defer cancel()

	type evalResult struct {
		value any
		err   error
	}
	ch := make(chan evalResult, 1)

	go func() {
		result, err := expr.Run(program, env)
		ch <- evalResult{value: result, err: err}
	}()

	select {
	case r := <-ch:
		if r.err != nil {
			return nil, fmt.Errorf("expr: runtime error: %w", r.err)
		}
		return r.value, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("expr: evaluation timed out (>%v)", maxExprTimeout)
	}
}

// EvalBool is a convenience method that evaluates and asserts a bool result.
func (e *ExprEvaluator) EvalBool(expression string) (bool, error) {
	result, err := e.Eval(expression)
	if err != nil {
		return false, err
	}
	b, ok := result.(bool)
	if !ok {
		return false, fmt.Errorf("expr: expected bool result, got %T", result)
	}
	return b, nil
}

// buildOptions returns the expr.Option list with all 6 injected functions.
func (e *ExprEvaluator) buildOptions() []expr.Option {
	return []expr.Option{
		expr.Function("has_item", e.hasItem,
			new(func(string) bool),
		),
		expr.Function("roll", e.rollDice,
			new(func(string) int),
		),
		expr.Function("attr", e.getAttr,
			new(func(string) int),
		),
		expr.Function("var", e.getVar,
			new(func(string) any),
		),
		expr.Function("all_have_item", e.allHaveItem,
			new(func(string) bool),
		),
		expr.Function("player_count", e.playerCount,
			new(func() int),
		),
		expr.Function("item_count", e.itemCount,
			new(func(string) int),
		),
	}
}

// hasItem checks if the trigger player has the specified item revealed.
func (e *ExprEvaluator) hasItem(params ...any) (any, error) {
	itemID, ok := params[0].(string)
	if !ok {
		return false, fmt.Errorf("has_item: item_id must be a string")
	}
	return e.gameState.IsItemRevealed(e.triggerPlayerID, itemID), nil
}

// rollDice rolls dice using the existing RollDice function and returns the total.
func (e *ExprEvaluator) rollDice(params ...any) (any, error) {
	notation, ok := params[0].(string)
	if !ok {
		return 0, fmt.Errorf("roll: notation must be a string")
	}
	result, err := RollDice(notation)
	if err != nil {
		return 0, fmt.Errorf("roll: %w", err)
	}
	return result.Total, nil
}

// getAttr reads a character attribute. It first checks the trigger player's
// character attributes (from player_joined), then falls back to scenario defaults.
// Returns 0 if the attribute is not found.
func (e *ExprEvaluator) getAttr(params ...any) (any, error) {
	name, ok := params[0].(string)
	if !ok {
		return 0, fmt.Errorf("attr: name must be a string")
	}
	// Check player's character attributes first.
	if e.triggerPlayerID != "" && e.gameState.PlayerAttributes != nil {
		if attrs, ok := e.gameState.PlayerAttributes[e.triggerPlayerID]; ok {
			if v, ok := attrs[name]; ok {
				return v, nil
			}
		}
	}
	// Fall back to scenario defaults.
	if e.scenario != nil && e.scenario.Rules != nil {
		for _, a := range e.scenario.Rules.Attributes {
			if a.Name == name || a.Display == name {
				return a.Default, nil
			}
		}
	}
	return 0, nil
}

// getVar reads a scenario variable value from the current game state.
// Returns nil if the variable is not found.
func (e *ExprEvaluator) getVar(params ...any) (any, error) {
	name, ok := params[0].(string)
	if !ok {
		return nil, fmt.Errorf("var: name must be a string")
	}
	if e.gameState.Variables != nil {
		return e.gameState.Variables[name], nil
	}
	return nil, nil
}

// allHaveItem checks if all connected players have the specified item revealed.
// Returns false if there are no connected players.
func (e *ExprEvaluator) allHaveItem(params ...any) (any, error) {
	itemID, ok := params[0].(string)
	if !ok {
		return false, fmt.Errorf("all_have_item: item_id must be a string")
	}
	if len(e.connectedPlayerIDs) == 0 {
		return false, nil
	}
	for _, pid := range e.connectedPlayerIDs {
		if !e.gameState.IsItemRevealed(pid, itemID) {
			return false, nil
		}
	}
	return true, nil
}

// playerCount returns the number of connected players.
func (e *ExprEvaluator) playerCount(params ...any) (any, error) {
	return len(e.connectedPlayerIDs), nil
}

// itemCount returns the quantity of the specified item in the trigger player's inventory.
func (e *ExprEvaluator) itemCount(params ...any) (any, error) {
	itemID, ok := params[0].(string)
	if !ok {
		return 0, fmt.Errorf("item_count: item_id must be a string")
	}
	return e.gameState.ItemQuantity(e.triggerPlayerID, itemID), nil
}

