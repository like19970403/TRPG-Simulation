package server

import (
	"encoding/json"
	"time"

	"github.com/like19970403/TRPG-Simulation/internal/character"
	"github.com/like19970403/TRPG-Simulation/internal/game"
	"github.com/like19970403/TRPG-Simulation/internal/realtime"
	"github.com/like19970403/TRPG-Simulation/internal/scenario"
)

// RegisterRequest is the JSON body for POST /api/v1/users.
type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RegisterResponse is the JSON response for POST /api/v1/users.
type RegisterResponse struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	CreatedAt string `json:"createdAt"`
}

// LoginRequest is the JSON body for POST /api/v1/auth/login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// TokenResponse is the JSON response for login and refresh endpoints.
type TokenResponse struct {
	AccessToken string `json:"accessToken"`
	ExpiresIn   int    `json:"expiresIn"`
	TokenType   string `json:"tokenType"`
}

// ErrorResponse is the unified error format per OpenAPI spec.
type ErrorResponse struct {
	Error   string        `json:"error"`
	Message string        `json:"message"`
	Details []ErrorDetail `json:"details,omitempty"`
}

// ErrorDetail provides field-level error information.
type ErrorDetail struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

// CreateScenarioRequest is the JSON body for POST /api/v1/scenarios.
type CreateScenarioRequest struct {
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Content     json.RawMessage `json:"content"`
}

// UpdateScenarioRequest is the JSON body for PUT /api/v1/scenarios/{id}.
type UpdateScenarioRequest struct {
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Content     json.RawMessage `json:"content"`
}

// ScenarioValidationWarning is a validation finding returned in scenario responses.
type ScenarioValidationWarning struct {
	Field    string `json:"field"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

// ScenarioResponse is the JSON response for a single scenario.
type ScenarioResponse struct {
	ID                 string                       `json:"id"`
	AuthorID           string                       `json:"authorId"`
	Title              string                       `json:"title"`
	Description        string                       `json:"description"`
	Version            int                          `json:"version"`
	Status             string                       `json:"status"`
	Content            json.RawMessage              `json:"content"`
	CreatedAt          string                       `json:"createdAt"`
	UpdatedAt          string                       `json:"updatedAt"`
	ValidationWarnings []ScenarioValidationWarning  `json:"validationWarnings,omitempty"`
}

// ScenarioListResponse is the JSON response for GET /api/v1/scenarios.
type ScenarioListResponse struct {
	Scenarios []ScenarioResponse `json:"scenarios"`
	Total     int                `json:"total"`
	Limit     int                `json:"limit"`
	Offset    int                `json:"offset"`
}

func toScenarioResponse(s *scenario.Scenario) ScenarioResponse {
	return ScenarioResponse{
		ID:          s.ID,
		AuthorID:    s.AuthorID,
		Title:       s.Title,
		Description: s.Description,
		Version:     s.Version,
		Status:      s.Status,
		Content:     s.Content,
		CreatedAt:   s.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:   s.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func toScenarioResponseWithWarnings(s *scenario.Scenario, warnings []realtime.ValidationError) ScenarioResponse {
	resp := toScenarioResponse(s)
	if len(warnings) > 0 {
		resp.ValidationWarnings = make([]ScenarioValidationWarning, len(warnings))
		for i, w := range warnings {
			resp.ValidationWarnings[i] = ScenarioValidationWarning{
				Field:    w.Field,
				Code:     w.Code,
				Message:  w.Message,
				Severity: string(w.Severity),
			}
		}
	}
	return resp
}

// CreateSessionRequest is the JSON body for POST /api/v1/sessions.
type CreateSessionRequest struct {
	ScenarioID string `json:"scenarioId"`
}

// JoinSessionRequest is the JSON body for POST /api/v1/sessions/join.
type JoinSessionRequest struct {
	InviteCode string `json:"inviteCode"`
}

// SessionResponse is the JSON response for a single game session.
type SessionResponse struct {
	ID         string  `json:"id"`
	ScenarioID string  `json:"scenarioId"`
	GMID       string  `json:"gmId"`
	Status     string  `json:"status"`
	InviteCode string  `json:"inviteCode"`
	CreatedAt  string  `json:"createdAt"`
	StartedAt  *string `json:"startedAt"`
	EndedAt    *string `json:"endedAt"`
}

// SessionListResponse is the JSON response for GET /api/v1/sessions.
type SessionListResponse struct {
	Sessions []SessionResponse `json:"sessions"`
	Total    int               `json:"total"`
	Limit    int               `json:"limit"`
	Offset   int               `json:"offset"`
}

// SessionPlayerResponse is the JSON response for a single session player.
type SessionPlayerResponse struct {
	ID          string  `json:"id"`
	UserID      string  `json:"userId"`
	CharacterID *string `json:"characterId"`
	Status      string  `json:"status"`
	JoinedAt    string  `json:"joinedAt"`
}

// SessionPlayerListResponse is the JSON response for GET /api/v1/sessions/{id}/players.
type SessionPlayerListResponse struct {
	Players []SessionPlayerResponse `json:"players"`
}

func toSessionResponse(gs *game.GameSession) SessionResponse {
	resp := SessionResponse{
		ID:         gs.ID,
		ScenarioID: gs.ScenarioID,
		GMID:       gs.GMID,
		Status:     gs.Status,
		InviteCode: gs.InviteCode,
		CreatedAt:  gs.CreatedAt.UTC().Format(time.RFC3339),
	}
	if gs.StartedAt != nil {
		t := gs.StartedAt.UTC().Format(time.RFC3339)
		resp.StartedAt = &t
	}
	if gs.EndedAt != nil {
		t := gs.EndedAt.UTC().Format(time.RFC3339)
		resp.EndedAt = &t
	}
	return resp
}

func toSessionPlayerResponse(sp *game.SessionPlayer) SessionPlayerResponse {
	return SessionPlayerResponse{
		ID:          sp.ID,
		UserID:      sp.UserID,
		CharacterID: sp.CharacterID,
		Status:      sp.Status,
		JoinedAt:    sp.JoinedAt.UTC().Format(time.RFC3339),
	}
}

func validateCreateSession(req CreateSessionRequest) []ErrorDetail {
	var errs []ErrorDetail
	if !isValidUUID(req.ScenarioID) {
		errs = append(errs, ErrorDetail{Field: "scenarioId", Reason: "must be a valid UUID"})
	}
	return errs
}

func validateJoinSession(req JoinSessionRequest) []ErrorDetail {
	var errs []ErrorDetail
	if len(req.InviteCode) == 0 {
		errs = append(errs, ErrorDetail{Field: "inviteCode", Reason: "must not be empty"})
	}
	return errs
}

// --- Character types ---

// CreateCharacterRequest is the JSON body for POST /api/v1/characters.
type CreateCharacterRequest struct {
	Name       string          `json:"name"`
	Attributes json.RawMessage `json:"attributes,omitempty"`
	Inventory  json.RawMessage `json:"inventory,omitempty"`
	Notes      string          `json:"notes,omitempty"`
}

// UpdateCharacterRequest is the JSON body for PUT /api/v1/characters/{id}.
type UpdateCharacterRequest struct {
	Name       string          `json:"name"`
	Attributes json.RawMessage `json:"attributes,omitempty"`
	Inventory  json.RawMessage `json:"inventory,omitempty"`
	Notes      string          `json:"notes,omitempty"`
}

// CharacterResponse is the JSON response for a single character.
type CharacterResponse struct {
	ID         string          `json:"id"`
	UserID     string          `json:"userId"`
	Name       string          `json:"name"`
	Attributes json.RawMessage `json:"attributes"`
	Inventory  json.RawMessage `json:"inventory"`
	Notes      string          `json:"notes"`
	CreatedAt  string          `json:"createdAt"`
	UpdatedAt  string          `json:"updatedAt"`
}

// CharacterListResponse is the JSON response for GET /api/v1/characters.
type CharacterListResponse struct {
	Characters []CharacterResponse `json:"characters"`
	Total      int                 `json:"total"`
	Limit      int                 `json:"limit"`
	Offset     int                 `json:"offset"`
}

// AssignCharacterRequest is the JSON body for POST /api/v1/sessions/{id}/characters.
type AssignCharacterRequest struct {
	CharacterID string `json:"characterId"`
}

func toCharacterResponse(c *character.Character) CharacterResponse {
	return CharacterResponse{
		ID:         c.ID,
		UserID:     c.UserID,
		Name:       c.Name,
		Attributes: c.Attributes,
		Inventory:  c.Inventory,
		Notes:      c.Notes,
		CreatedAt:  c.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:  c.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func validateCreateCharacter(req CreateCharacterRequest) []ErrorDetail {
	var errs []ErrorDetail
	if len(req.Name) == 0 || len(req.Name) > 100 {
		errs = append(errs, ErrorDetail{Field: "name", Reason: "must be between 1 and 100 characters"})
	}
	if len(req.Attributes) > 0 {
		var obj map[string]any
		if err := json.Unmarshal(req.Attributes, &obj); err != nil {
			errs = append(errs, ErrorDetail{Field: "attributes", Reason: "must be a valid JSON object"})
		}
	}
	if len(req.Inventory) > 0 {
		var arr []any
		if err := json.Unmarshal(req.Inventory, &arr); err != nil {
			errs = append(errs, ErrorDetail{Field: "inventory", Reason: "must be a valid JSON array"})
		}
	}
	return errs
}

func validateUpdateCharacter(req UpdateCharacterRequest) []ErrorDetail {
	var errs []ErrorDetail
	if len(req.Name) == 0 || len(req.Name) > 100 {
		errs = append(errs, ErrorDetail{Field: "name", Reason: "must be between 1 and 100 characters"})
	}
	if len(req.Attributes) > 0 {
		var obj map[string]any
		if err := json.Unmarshal(req.Attributes, &obj); err != nil {
			errs = append(errs, ErrorDetail{Field: "attributes", Reason: "must be a valid JSON object"})
		}
	}
	if len(req.Inventory) > 0 {
		var arr []any
		if err := json.Unmarshal(req.Inventory, &arr); err != nil {
			errs = append(errs, ErrorDetail{Field: "inventory", Reason: "must be a valid JSON array"})
		}
	}
	return errs
}

func validateAssignCharacter(req AssignCharacterRequest) []ErrorDetail {
	var errs []ErrorDetail
	if !isValidUUID(req.CharacterID) {
		errs = append(errs, ErrorDetail{Field: "characterId", Reason: "must be a valid UUID"})
	}
	return errs
}
