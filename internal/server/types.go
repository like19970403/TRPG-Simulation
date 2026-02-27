package server

import (
	"encoding/json"
	"time"

	"github.com/like19970403/TRPG-Simulation/internal/game"
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

// ScenarioResponse is the JSON response for a single scenario.
type ScenarioResponse struct {
	ID          string          `json:"id"`
	AuthorID    string          `json:"authorId"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Version     int             `json:"version"`
	Status      string          `json:"status"`
	Content     json.RawMessage `json:"content"`
	CreatedAt   string          `json:"createdAt"`
	UpdatedAt   string          `json:"updatedAt"`
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
	ID       string `json:"id"`
	UserID   string `json:"userId"`
	Status   string `json:"status"`
	JoinedAt string `json:"joinedAt"`
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
		ID:       sp.ID,
		UserID:   sp.UserID,
		Status:   sp.Status,
		JoinedAt: sp.JoinedAt.UTC().Format(time.RFC3339),
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
