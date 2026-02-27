package server

import (
	"encoding/json"
	"time"

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
