package realtime

import (
	"encoding/json"
	"fmt"
)

// ScenarioContent represents the parsed content of a scenario YAML/JSON (ADR-003).
type ScenarioContent struct {
	ID         string     `json:"id"`
	Title      string     `json:"title"`
	StartScene string     `json:"start_scene"`
	Scenes     []Scene    `json:"scenes"`
	Items      []Item     `json:"items,omitempty"`
	NPCs       []NPC      `json:"npcs,omitempty"`
	Variables  []Variable `json:"variables,omitempty"`
	Rules      *Rules     `json:"rules,omitempty"`
}

// Scene represents a single scene in the scenario graph.
type Scene struct {
	ID             string       `json:"id"`
	Name           string       `json:"name"`
	Content        string       `json:"content"`
	GMNotes        string       `json:"gm_notes,omitempty"`
	ItemsAvailable []string     `json:"items_available,omitempty"`
	NPCsPresent    []string     `json:"npcs_present,omitempty"`
	Transitions    []Transition `json:"transitions,omitempty"`
	OnEnter        []Action     `json:"on_enter,omitempty"`
	OnExit         []Action     `json:"on_exit,omitempty"`
}

// Transition represents a directed edge between scenes.
type Transition struct {
	Target    string `json:"target"`
	Trigger   string `json:"trigger"`
	Condition string `json:"condition,omitempty"`
	Label     string `json:"label,omitempty"`
}

// Item represents a game item or clue.
type Item struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Image       string `json:"image,omitempty"`
}

// NPC represents a non-player character with fields.
type NPC struct {
	ID     string     `json:"id"`
	Name   string     `json:"name"`
	Image  string     `json:"image,omitempty"`
	Fields []NPCField `json:"fields,omitempty"`
}

// NPCField represents a single field on an NPC card.
type NPCField struct {
	Key        string `json:"key"`
	Label      string `json:"label"`
	Value      string `json:"value"`
	Visibility string `json:"visibility"`
}

// Variable represents a scenario-level variable.
type Variable struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Default any    `json:"default"`
}

// Rules represents the scenario custom rules.
type Rules struct {
	Attributes  []Attribute `json:"attributes,omitempty"`
	DiceFormula string      `json:"dice_formula,omitempty"`
	CheckMethod string      `json:"check_method,omitempty"`
}

// Attribute represents a character attribute definition.
type Attribute struct {
	Name    string `json:"name"`
	Display string `json:"display"`
	Default int    `json:"default"`
}

// Action represents an on_enter or on_exit action in a scene (ADR-003).
// Exactly one of the fields should be non-nil.
type Action struct {
	SetVar         *SetVarAction         `json:"set_var,omitempty"`
	RevealItem     *RevealItemAction     `json:"reveal_item,omitempty"`
	RevealNPCField *RevealNPCFieldAction `json:"reveal_npc_field,omitempty"`
}

// SetVarAction sets a scenario variable to a literal value.
type SetVarAction struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
}

// RevealItemAction reveals an item to a player or all players.
// To: "current_player" | "all" | specific player ID.
type RevealItemAction struct {
	ItemID string `json:"item_id"`
	To     string `json:"to"`
}

// RevealNPCFieldAction reveals an NPC field to a player or all players.
// To: "current_player" | "all" | specific player ID.
type RevealNPCFieldAction struct {
	NPCID    string `json:"npc_id"`
	FieldKey string `json:"field_key"`
	To       string `json:"to"`
}

// ParseScenarioContent parses raw JSON into a ScenarioContent struct.
// Returns an error if the JSON is invalid or required fields are missing.
func ParseScenarioContent(raw json.RawMessage) (*ScenarioContent, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("scenario: empty content")
	}

	var sc ScenarioContent
	if err := json.Unmarshal(raw, &sc); err != nil {
		return nil, fmt.Errorf("scenario: invalid JSON: %w", err)
	}

	if sc.StartScene == "" {
		return nil, fmt.Errorf("scenario: start_scene is required")
	}
	if len(sc.Scenes) == 0 {
		return nil, fmt.Errorf("scenario: at least one scene is required")
	}

	return &sc, nil
}

// FindScene returns the scene with the given ID, or nil if not found.
func (sc *ScenarioContent) FindScene(sceneID string) *Scene {
	for i := range sc.Scenes {
		if sc.Scenes[i].ID == sceneID {
			return &sc.Scenes[i]
		}
	}
	return nil
}

// FindItem returns the item with the given ID, or nil if not found.
func (sc *ScenarioContent) FindItem(itemID string) *Item {
	for i := range sc.Items {
		if sc.Items[i].ID == itemID {
			return &sc.Items[i]
		}
	}
	return nil
}

// FindNPC returns the NPC with the given ID, or nil if not found.
func (sc *ScenarioContent) FindNPC(npcID string) *NPC {
	for i := range sc.NPCs {
		if sc.NPCs[i].ID == npcID {
			return &sc.NPCs[i]
		}
	}
	return nil
}

// FindField returns the field with the given key, or nil if not found.
func (npc *NPC) FindField(fieldKey string) *NPCField {
	for i := range npc.Fields {
		if npc.Fields[i].Key == fieldKey {
			return &npc.Fields[i]
		}
	}
	return nil
}
