package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/like19970403/TRPG-Simulation/internal/apperror"
)

func (s *Server) handleCreateCharacter(w http.ResponseWriter, r *http.Request) {
	claims := UserClaimsFromContext(r.Context())

	var req CreateCharacterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	if errs := validateCreateCharacter(req); len(errs) > 0 {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Request validation failed", errs)
		return
	}

	// Supply defaults for optional JSONB fields.
	attributes := req.Attributes
	if len(attributes) == 0 {
		attributes = json.RawMessage(`{}`)
	}
	inventory := req.Inventory
	if len(inventory) == 0 {
		inventory = json.RawMessage(`[]`)
	}

	c, err := s.characterRepo.Create(r.Context(), claims.UserID, req.Name, attributes, inventory, req.Notes, req.ImageURL)
	if err != nil {
		s.logger.Error("character: create", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	s.writeJSON(w, http.StatusCreated, toCharacterResponse(c))
}

func (s *Server) handleListCharacters(w http.ResponseWriter, r *http.Request) {
	claims := UserClaimsFromContext(r.Context())

	limit, offset, err := parsePagination(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}

	chars, total, err := s.characterRepo.ListByUser(r.Context(), claims.UserID, limit, offset)
	if err != nil {
		s.logger.Error("character: list", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	items := make([]CharacterResponse, 0, len(chars))
	for _, c := range chars {
		items = append(items, toCharacterResponse(c))
	}

	s.writeJSON(w, http.StatusOK, CharacterListResponse{
		Characters: items,
		Total:      total,
		Limit:      limit,
		Offset:     offset,
	})
}

func (s *Server) handleGetCharacter(w http.ResponseWriter, r *http.Request) {
	claims := UserClaimsFromContext(r.Context())
	id := r.PathValue("id")
	if !isValidUUID(id) {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid character ID", nil)
		return
	}

	c, err := s.characterRepo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Character not found", nil)
			return
		}
		s.logger.Error("character: get", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	if c.UserID != claims.UserID {
		s.writeError(w, http.StatusForbidden, "FORBIDDEN", "You do not have access to this character", nil)
		return
	}

	s.writeJSON(w, http.StatusOK, toCharacterResponse(c))
}

func (s *Server) handleUpdateCharacter(w http.ResponseWriter, r *http.Request) {
	claims := UserClaimsFromContext(r.Context())
	id := r.PathValue("id")
	if !isValidUUID(id) {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid character ID", nil)
		return
	}

	// Check existence and ownership.
	existing, err := s.characterRepo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Character not found", nil)
			return
		}
		s.logger.Error("character: get for update", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	if existing.UserID != claims.UserID {
		s.writeError(w, http.StatusForbidden, "FORBIDDEN", "You do not have access to this character", nil)
		return
	}

	var req UpdateCharacterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	if errs := validateUpdateCharacter(req); len(errs) > 0 {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Request validation failed", errs)
		return
	}

	attributes := req.Attributes
	if len(attributes) == 0 {
		attributes = json.RawMessage(`{}`)
	}
	inventory := req.Inventory
	if len(inventory) == 0 {
		inventory = json.RawMessage(`[]`)
	}

	updated, err := s.characterRepo.Update(r.Context(), id, req.Name, attributes, inventory, req.Notes, req.ImageURL)
	if err != nil {
		s.logger.Error("character: update", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	s.writeJSON(w, http.StatusOK, toCharacterResponse(updated))
}

func (s *Server) handleDeleteCharacter(w http.ResponseWriter, r *http.Request) {
	claims := UserClaimsFromContext(r.Context())
	id := r.PathValue("id")
	if !isValidUUID(id) {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid character ID", nil)
		return
	}

	// Check existence and ownership.
	c, err := s.characterRepo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Character not found", nil)
			return
		}
		s.logger.Error("character: get for delete", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	if c.UserID != claims.UserID {
		s.writeError(w, http.StatusForbidden, "FORBIDDEN", "You do not have access to this character", nil)
		return
	}

	// Check if character is linked to any session.
	linked, err := s.characterRepo.IsLinkedToSession(r.Context(), id)
	if err != nil {
		s.logger.Error("character: check session link", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}
	if linked {
		s.writeError(w, http.StatusConflict, "CONFLICT", "Cannot delete a character that is assigned to a session", nil)
		return
	}

	if err := s.characterRepo.Delete(r.Context(), id); err != nil {
		s.logger.Error("character: delete", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAssignCharacter(w http.ResponseWriter, r *http.Request) {
	claims := UserClaimsFromContext(r.Context())
	sessionID := r.PathValue("id")
	if !isValidUUID(sessionID) {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid session ID", nil)
		return
	}

	var req AssignCharacterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	if errs := validateAssignCharacter(req); len(errs) > 0 {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Request validation failed", errs)
		return
	}

	// Check session exists.
	gs, err := s.sessionRepo.GetByID(r.Context(), sessionID)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Session not found", nil)
			return
		}
		s.logger.Error("character: get session for assign", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	// Session must be in lobby state.
	if gs.Status != "lobby" {
		s.writeError(w, http.StatusConflict, "CONFLICT", "Characters can only be assigned in lobby state", nil)
		return
	}

	// User must be a player in the session (not GM).
	_, err = s.sessionRepo.GetPlayer(r.Context(), sessionID, claims.UserID)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			s.writeError(w, http.StatusForbidden, "FORBIDDEN", "You are not a player in this session", nil)
			return
		}
		s.logger.Error("character: get player for assign", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	// Check character exists and belongs to user.
	c, err := s.characterRepo.GetByID(r.Context(), req.CharacterID)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Character not found", nil)
			return
		}
		s.logger.Error("character: get for assign", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	if c.UserID != claims.UserID {
		s.writeError(w, http.StatusForbidden, "FORBIDDEN", "You do not own this character", nil)
		return
	}

	// System matching: check character's system matches scenario's system.
	// If scenarioRepo is unavailable (e.g. tests), skip gracefully.
	sc, scErr := s.scenarioRepo.GetByID(r.Context(), gs.ScenarioID)
	if scErr == nil && sc != nil && len(sc.Content) > 0 {
		var content struct {
			System string `json:"system"`
		}
		if json.Unmarshal(sc.Content, &content) == nil && content.System != "" {
			var profile struct {
				System string `json:"_system"`
			}
			_ = json.Unmarshal([]byte(c.Notes), &profile)
			if profile.System != content.System {
				s.writeError(w, http.StatusBadRequest, "SYSTEM_MISMATCH",
					"此劇本需要 "+content.System+" 系統的角色", nil)
				return
			}
		}
	}

	// Assign character to session player.
	sp, err := s.sessionRepo.SetCharacterID(r.Context(), sessionID, claims.UserID, req.CharacterID)
	if err != nil {
		s.logger.Error("character: assign", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	s.writeJSON(w, http.StatusOK, toSessionPlayerResponse(sp))
}
