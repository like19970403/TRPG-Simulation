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
