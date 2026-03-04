package realtime

import (
	"fmt"

	"github.com/expr-lang/expr"
)

// ValidationSeverity indicates how critical a validation finding is.
type ValidationSeverity string

const (
	SeverityError   ValidationSeverity = "error"
	SeverityWarning ValidationSeverity = "warning"
)

// ValidationError represents a single validation finding.
type ValidationError struct {
	Field    string             `json:"field"`
	Code     string             `json:"code"`
	Message  string             `json:"message"`
	Severity ValidationSeverity `json:"severity"`
}

// Size limits.
const (
	maxScenes           = 200
	maxItems            = 500
	maxNPCs             = 100
	maxTransitionsPerSc = 20
)

// ValidateScenarioContent performs comprehensive validation on a parsed ScenarioContent.
// Returns a slice of validation findings (errors and warnings).
func ValidateScenarioContent(sc *ScenarioContent) []ValidationError {
	var errs []ValidationError

	// Build lookup maps.
	sceneIDs := make(map[string]bool, len(sc.Scenes))
	for _, s := range sc.Scenes {
		sceneIDs[s.ID] = true
	}
	itemIDs := make(map[string]bool, len(sc.Items))
	for _, it := range sc.Items {
		itemIDs[it.ID] = true
	}
	npcIDs := make(map[string]bool, len(sc.NPCs))
	npcFieldKeys := make(map[string]map[string]bool) // npcID → fieldKey set
	for _, n := range sc.NPCs {
		npcIDs[n.ID] = true
		fk := make(map[string]bool, len(n.Fields))
		for _, f := range n.Fields {
			fk[f.Key] = true
		}
		npcFieldKeys[n.ID] = fk
	}
	varNames := make(map[string]bool, len(sc.Variables))
	for _, v := range sc.Variables {
		varNames[v.Name] = true
	}

	// --- Structure validation ---
	errs = append(errs, validateStructure(sc, sceneIDs)...)

	// --- Size limits ---
	errs = append(errs, validateSizeLimits(sc)...)

	// --- Scene graph integrity ---
	errs = append(errs, validateSceneGraph(sc, sceneIDs)...)

	// --- Reference integrity ---
	errs = append(errs, validateReferences(sc, itemIDs, npcIDs, npcFieldKeys, varNames)...)

	// --- Expression pre-validation ---
	errs = append(errs, validateExpressions(sc)...)

	return errs
}

// HasErrors returns true if any validation error has severity "error".
func HasErrors(errs []ValidationError) bool {
	for _, e := range errs {
		if e.Severity == SeverityError {
			return true
		}
	}
	return false
}

func validateStructure(sc *ScenarioContent, sceneIDs map[string]bool) []ValidationError {
	var errs []ValidationError

	for i, s := range sc.Scenes {
		if s.ID == "" {
			errs = append(errs, ValidationError{
				Field:    fmt.Sprintf("scenes[%d]", i),
				Code:     "missing_id",
				Message:  "Scene must have an id",
				Severity: SeverityError,
			})
		}
		if s.Name == "" {
			errs = append(errs, ValidationError{
				Field:    fmt.Sprintf("scenes[%d]", i),
				Code:     "missing_name",
				Message:  fmt.Sprintf("Scene '%s' must have a name", s.ID),
				Severity: SeverityError,
			})
		}
	}
	for i, it := range sc.Items {
		if it.ID == "" {
			errs = append(errs, ValidationError{
				Field:    fmt.Sprintf("items[%d]", i),
				Code:     "missing_id",
				Message:  "Item must have an id",
				Severity: SeverityError,
			})
		}
		if it.Name == "" {
			errs = append(errs, ValidationError{
				Field:    fmt.Sprintf("items[%d]", i),
				Code:     "missing_name",
				Message:  fmt.Sprintf("Item '%s' must have a name", it.ID),
				Severity: SeverityError,
			})
		}
	}
	for i, n := range sc.NPCs {
		if n.ID == "" {
			errs = append(errs, ValidationError{
				Field:    fmt.Sprintf("npcs[%d]", i),
				Code:     "missing_id",
				Message:  "NPC must have an id",
				Severity: SeverityError,
			})
		}
		if n.Name == "" {
			errs = append(errs, ValidationError{
				Field:    fmt.Sprintf("npcs[%d]", i),
				Code:     "missing_name",
				Message:  fmt.Sprintf("NPC '%s' must have a name", n.ID),
				Severity: SeverityError,
			})
		}
	}

	// start_scene must exist
	if sc.StartScene != "" && !sceneIDs[sc.StartScene] {
		errs = append(errs, ValidationError{
			Field:    "start_scene",
			Code:     "invalid_start_scene",
			Message:  fmt.Sprintf("start_scene '%s' does not exist in scenes", sc.StartScene),
			Severity: SeverityError,
		})
	}

	return errs
}

func validateSizeLimits(sc *ScenarioContent) []ValidationError {
	var errs []ValidationError

	if len(sc.Scenes) > maxScenes {
		errs = append(errs, ValidationError{
			Field:    "scenes",
			Code:     "too_many_scenes",
			Message:  fmt.Sprintf("Too many scenes (%d, max %d)", len(sc.Scenes), maxScenes),
			Severity: SeverityError,
		})
	}
	if len(sc.Items) > maxItems {
		errs = append(errs, ValidationError{
			Field:    "items",
			Code:     "too_many_items",
			Message:  fmt.Sprintf("Too many items (%d, max %d)", len(sc.Items), maxItems),
			Severity: SeverityError,
		})
	}
	if len(sc.NPCs) > maxNPCs {
		errs = append(errs, ValidationError{
			Field:    "npcs",
			Code:     "too_many_npcs",
			Message:  fmt.Sprintf("Too many NPCs (%d, max %d)", len(sc.NPCs), maxNPCs),
			Severity: SeverityError,
		})
	}
	for _, s := range sc.Scenes {
		if len(s.Transitions) > maxTransitionsPerSc {
			errs = append(errs, ValidationError{
				Field:    fmt.Sprintf("scenes[%s].transitions", s.ID),
				Code:     "too_many_transitions",
				Message:  fmt.Sprintf("Scene '%s' has too many transitions (%d, max %d)", s.ID, len(s.Transitions), maxTransitionsPerSc),
				Severity: SeverityError,
			})
		}
	}

	return errs
}

func validateSceneGraph(sc *ScenarioContent, sceneIDs map[string]bool) []ValidationError {
	var errs []ValidationError

	for _, s := range sc.Scenes {
		for j, t := range s.Transitions {
			// Target must exist.
			if !sceneIDs[t.Target] {
				errs = append(errs, ValidationError{
					Field:    fmt.Sprintf("scenes[%s].transitions[%d].target", s.ID, j),
					Code:     "invalid_target",
					Message:  fmt.Sprintf("Transition target '%s' does not exist in scenes", t.Target),
					Severity: SeverityError,
				})
			}
			// Self-referencing transition warning.
			if t.Target == s.ID {
				errs = append(errs, ValidationError{
					Field:    fmt.Sprintf("scenes[%s].transitions[%d]", s.ID, j),
					Code:     "self_transition",
					Message:  fmt.Sprintf("Scene '%s' has a self-referencing transition", s.ID),
					Severity: SeverityWarning,
				})
			}
		}
	}

	// Reachability: BFS from start_scene.
	if sc.StartScene != "" && sceneIDs[sc.StartScene] {
		reachable := make(map[string]bool)
		queue := []string{sc.StartScene}
		reachable[sc.StartScene] = true
		for len(queue) > 0 {
			curr := queue[0]
			queue = queue[1:]
			scene := sc.FindScene(curr)
			if scene == nil {
				continue
			}
			for _, t := range scene.Transitions {
				if !reachable[t.Target] && sceneIDs[t.Target] {
					reachable[t.Target] = true
					queue = append(queue, t.Target)
				}
			}
		}
		for _, s := range sc.Scenes {
			if !reachable[s.ID] {
				errs = append(errs, ValidationError{
					Field:    fmt.Sprintf("scenes[%s]", s.ID),
					Code:     "orphan_scene",
					Message:  fmt.Sprintf("Scene '%s' is unreachable from start_scene", s.ID),
					Severity: SeverityWarning,
				})
			}
		}
	}

	return errs
}

func validateReferences(sc *ScenarioContent, itemIDs, npcIDs map[string]bool, npcFieldKeys map[string]map[string]bool, varNames map[string]bool) []ValidationError {
	var errs []ValidationError

	for _, s := range sc.Scenes {
		// items_available references.
		for j, itemID := range s.ItemsAvailable {
			if !itemIDs[itemID] {
				errs = append(errs, ValidationError{
					Field:    fmt.Sprintf("scenes[%s].items_available[%d]", s.ID, j),
					Code:     "invalid_item_ref",
					Message:  fmt.Sprintf("Item '%s' not found in items", itemID),
					Severity: SeverityError,
				})
			}
		}
		// npcs_present references.
		for j, npcID := range s.NPCsPresent {
			if !npcIDs[npcID] {
				errs = append(errs, ValidationError{
					Field:    fmt.Sprintf("scenes[%s].npcs_present[%d]", s.ID, j),
					Code:     "invalid_npc_ref",
					Message:  fmt.Sprintf("NPC '%s' not found in npcs", npcID),
					Severity: SeverityError,
				})
			}
		}
		// on_enter / on_exit action references.
		validateActions(s.ID, "on_enter", s.OnEnter, itemIDs, npcIDs, npcFieldKeys, varNames, &errs)
		validateActions(s.ID, "on_exit", s.OnExit, itemIDs, npcIDs, npcFieldKeys, varNames, &errs)
	}

	return errs
}

func validateActions(sceneID, phase string, actions []Action, itemIDs, npcIDs map[string]bool, npcFieldKeys map[string]map[string]bool, varNames map[string]bool, errs *[]ValidationError) {
	for i, a := range actions {
		prefix := fmt.Sprintf("scenes[%s].%s[%d]", sceneID, phase, i)

		if a.RevealItem != nil && !itemIDs[a.RevealItem.ItemID] {
			*errs = append(*errs, ValidationError{
				Field:    prefix + ".reveal_item.item_id",
				Code:     "invalid_item_ref",
				Message:  fmt.Sprintf("Item '%s' not found in items", a.RevealItem.ItemID),
				Severity: SeverityError,
			})
		}
		if a.GiveItem != nil && !itemIDs[a.GiveItem.ItemID] {
			*errs = append(*errs, ValidationError{
				Field:    prefix + ".give_item.item_id",
				Code:     "invalid_item_ref",
				Message:  fmt.Sprintf("Item '%s' not found in items", a.GiveItem.ItemID),
				Severity: SeverityError,
			})
		}
		if a.RemoveItem != nil && !itemIDs[a.RemoveItem.ItemID] {
			*errs = append(*errs, ValidationError{
				Field:    prefix + ".remove_item.item_id",
				Code:     "invalid_item_ref",
				Message:  fmt.Sprintf("Item '%s' not found in items", a.RemoveItem.ItemID),
				Severity: SeverityError,
			})
		}
		if a.RevealNPCField != nil {
			npcID := a.RevealNPCField.NPCID
			if !npcIDs[npcID] {
				*errs = append(*errs, ValidationError{
					Field:    prefix + ".reveal_npc_field.npc_id",
					Code:     "invalid_npc_ref",
					Message:  fmt.Sprintf("NPC '%s' not found in npcs", npcID),
					Severity: SeverityError,
				})
			} else if fk, ok := npcFieldKeys[npcID]; ok && !fk[a.RevealNPCField.FieldKey] {
				*errs = append(*errs, ValidationError{
					Field:    prefix + ".reveal_npc_field.field_key",
					Code:     "invalid_field_ref",
					Message:  fmt.Sprintf("NPC '%s' has no field '%s'", npcID, a.RevealNPCField.FieldKey),
					Severity: SeverityError,
				})
			}
		}
		if a.SetVar != nil && !varNames[a.SetVar.Name] {
			*errs = append(*errs, ValidationError{
				Field:    prefix + ".set_var.name",
				Code:     "undefined_variable",
				Message:  fmt.Sprintf("Variable '%s' not defined in variables (GM can still set it at runtime)", a.SetVar.Name),
				Severity: SeverityWarning,
			})
		}
	}
}

// exprValidationOptions returns expr options with stub functions for compile-only validation.
func exprValidationOptions() []expr.Option {
	stub1 := func(...any) (any, error) { return nil, nil }
	return []expr.Option{
		expr.Function("has_item", stub1, new(func(string) bool)),
		expr.Function("roll", stub1, new(func(string) int)),
		expr.Function("attr", stub1, new(func(string) int)),
		expr.Function("var", stub1, new(func(string) any)),
		expr.Function("all_have_item", stub1, new(func(string) bool)),
		expr.Function("player_count", stub1, new(func() int)),
		expr.Function("item_count", stub1, new(func(string) int)),
	}
}

func validateExpressions(sc *ScenarioContent) []ValidationError {
	var errs []ValidationError
	opts := exprValidationOptions()

	for _, s := range sc.Scenes {
		// Transition conditions.
		for j, t := range s.Transitions {
			if t.Condition != "" {
				if _, err := expr.Compile(t.Condition, opts...); err != nil {
					errs = append(errs, ValidationError{
						Field:    fmt.Sprintf("scenes[%s].transitions[%d].condition", s.ID, j),
						Code:     "invalid_expression",
						Message:  fmt.Sprintf("Cannot compile condition: %v", err),
						Severity: SeverityError,
					})
				}
			}
		}
		// on_enter / on_exit set_var expressions.
		validateActionExprs(s.ID, "on_enter", s.OnEnter, opts, &errs)
		validateActionExprs(s.ID, "on_exit", s.OnExit, opts, &errs)
	}

	return errs
}

func validateActionExprs(sceneID, phase string, actions []Action, opts []expr.Option, errs *[]ValidationError) {
	for i, a := range actions {
		if a.SetVar != nil && a.SetVar.Expr != "" {
			if _, err := expr.Compile(a.SetVar.Expr, opts...); err != nil {
				*errs = append(*errs, ValidationError{
					Field:    fmt.Sprintf("scenes[%s].%s[%d].set_var.expr", sceneID, phase, i),
					Code:     "invalid_expression",
					Message:  fmt.Sprintf("Cannot compile expression: %v", err),
					Severity: SeverityError,
				})
			}
		}
	}
}
