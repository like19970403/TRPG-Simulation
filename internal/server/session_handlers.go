package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/like19970403/TRPG-Simulation/internal/apperror"
	"github.com/like19970403/TRPG-Simulation/internal/realtime"
)

// isSessionParticipant checks if the user is the GM or a player of the session.
func (s *Server) isSessionParticipant(r *http.Request, sessionID, userID, gmID string) bool {
	if userID == gmID {
		return true
	}
	_, err := s.sessionRepo.GetPlayer(r.Context(), sessionID, userID)
	return err == nil
}

func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	claims := UserClaimsFromContext(r.Context())

	var req CreateSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	if errs := validateCreateSession(req); len(errs) > 0 {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Request validation failed", errs)
		return
	}

	// Verify scenario exists and is published.
	sc, err := s.scenarioRepo.GetByID(r.Context(), req.ScenarioID)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Scenario not found", nil)
			return
		}
		s.logger.Error("session: get scenario", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}
	if sc.Status != "published" {
		s.writeError(w, http.StatusConflict, "CONFLICT", "Only published scenarios can be used to create sessions", nil)
		return
	}

	gs, err := s.sessionRepo.Create(r.Context(), req.ScenarioID, claims.UserID)
	if err != nil {
		s.logger.Error("session: create", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	s.writeJSON(w, http.StatusCreated, toSessionResponse(gs))
}

func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	claims := UserClaimsFromContext(r.Context())

	limit, offset, err := parsePagination(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}

	// Fetch sessions where user is GM.
	gmSessions, gmTotal, err := s.sessionRepo.ListByGM(r.Context(), claims.UserID, limit, offset)
	if err != nil {
		s.logger.Error("session: list gm", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	// Fetch sessions where user is a player.
	playerSessions, playerTotal, err := s.sessionRepo.ListByPlayer(r.Context(), claims.UserID, limit, offset)
	if err != nil {
		s.logger.Error("session: list player", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	// Merge and deduplicate (GM sessions first, then player sessions).
	seen := make(map[string]bool, len(gmSessions))
	items := make([]SessionResponse, 0, len(gmSessions)+len(playerSessions))
	for _, gs := range gmSessions {
		seen[gs.ID] = true
		items = append(items, toSessionResponse(gs))
	}
	for _, gs := range playerSessions {
		if !seen[gs.ID] {
			items = append(items, toSessionResponse(gs))
		}
	}

	total := gmTotal + playerTotal

	s.writeJSON(w, http.StatusOK, SessionListResponse{
		Sessions: items,
		Total:    total,
		Limit:    limit,
		Offset:   offset,
	})
}

func (s *Server) handleGetSession(w http.ResponseWriter, r *http.Request) {
	claims := UserClaimsFromContext(r.Context())
	id := r.PathValue("id")

	if !isValidUUID(id) {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid session ID", nil)
		return
	}

	gs, err := s.sessionRepo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Session not found", nil)
			return
		}
		s.logger.Error("session: get", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	if !s.isSessionParticipant(r, gs.ID, claims.UserID, gs.GMID) {
		s.writeError(w, http.StatusForbidden, "FORBIDDEN", "You do not have access to this session", nil)
		return
	}

	s.writeJSON(w, http.StatusOK, toSessionResponse(gs))
}

func (s *Server) handleStartSession(w http.ResponseWriter, r *http.Request) {
	s.handleSessionTransition(w, r, "lobby", "active", "Only lobby sessions can be started", realtime.EventGameStarted)
}

func (s *Server) handlePauseSession(w http.ResponseWriter, r *http.Request) {
	s.handleSessionTransition(w, r, "active", "paused", "Only active sessions can be paused", realtime.EventGamePaused)
}

func (s *Server) handleResumeSession(w http.ResponseWriter, r *http.Request) {
	s.handleSessionTransition(w, r, "paused", "active", "Only paused sessions can be resumed", realtime.EventGameResumed)
}

func (s *Server) handleEndSession(w http.ResponseWriter, r *http.Request) {
	claims := UserClaimsFromContext(r.Context())
	id := r.PathValue("id")

	if !isValidUUID(id) {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid session ID", nil)
		return
	}

	gs, err := s.sessionRepo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Session not found", nil)
			return
		}
		s.logger.Error("session: get for end", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	if gs.GMID != claims.UserID {
		s.writeError(w, http.StatusForbidden, "FORBIDDEN", "Only the GM can end the session", nil)
		return
	}

	if gs.Status != "active" && gs.Status != "paused" {
		s.writeError(w, http.StatusConflict, "CONFLICT", "Only active or paused sessions can be ended", nil)
		return
	}

	updated, err := s.sessionRepo.UpdateStatus(r.Context(), id, "completed")
	if err != nil {
		s.logger.Error("session: end", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	// Persist player inventories back to characters before closing the room.
	if s.hub != nil {
		if room := s.hub.GetRoom(gs.ID); room != nil {
			state := room.StateSnapshot()
			players, pErr := s.sessionRepo.ListPlayers(r.Context(), gs.ID)
			if pErr == nil {
				for _, sp := range players {
					if sp.CharacterID == nil {
						continue
					}
					inv := state.PlayerInventory[sp.UserID]
					invJSON, jErr := json.Marshal(inv)
					if jErr != nil {
						continue
					}
					func() {
						defer func() { recover() }() //nolint:errcheck
						ch, cErr := s.characterRepo.GetByID(r.Context(), *sp.CharacterID)
						if cErr != nil {
							s.logger.Warn("session: end: get character for inventory sync", "error", cErr, "characterID", *sp.CharacterID)
							return
						}
						if _, uErr := s.characterRepo.Update(r.Context(), ch.ID, ch.Name, ch.Attributes, invJSON, ch.Notes, ch.ImageURL); uErr != nil {
							s.logger.Error("session: end: sync inventory to character", "error", uErr, "characterID", ch.ID)
						}
					}()
				}
			}

			room.BroadcastEvent(realtime.EventGameEnded, &claims.UserID, json.RawMessage(`{}`))
		}
		s.hub.RemoveRoom(gs.ID)
	}

	s.writeJSON(w, http.StatusOK, toSessionResponse(updated))
}

// handleSessionTransition is a generic handler for single-status GM-only transitions.
func (s *Server) handleSessionTransition(w http.ResponseWriter, r *http.Request, requiredStatus, newStatus, conflictMsg, eventType string) {
	claims := UserClaimsFromContext(r.Context())
	id := r.PathValue("id")

	if !isValidUUID(id) {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid session ID", nil)
		return
	}

	gs, err := s.sessionRepo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Session not found", nil)
			return
		}
		s.logger.Error("session: get for transition", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	if gs.GMID != claims.UserID {
		s.writeError(w, http.StatusForbidden, "FORBIDDEN", "Only the GM can perform this action", nil)
		return
	}

	if gs.Status != requiredStatus {
		s.writeError(w, http.StatusConflict, "CONFLICT", conflictMsg, nil)
		return
	}

	updated, err := s.sessionRepo.UpdateStatus(r.Context(), id, newStatus)
	if err != nil {
		s.logger.Error("session: transition", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	// Broadcast lifecycle event to connected WebSocket clients.
	if s.hub != nil {
		room := s.hub.GetOrCreateRoom(gs.ID, gs.GMID)
		room.BroadcastEvent(eventType, &claims.UserID, json.RawMessage(`{}`))
	}

	s.writeJSON(w, http.StatusOK, toSessionResponse(updated))
}

func (s *Server) handleJoinSession(w http.ResponseWriter, r *http.Request) {
	claims := UserClaimsFromContext(r.Context())

	var req JoinSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	if errs := validateJoinSession(req); len(errs) > 0 {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Request validation failed", errs)
		return
	}

	// Case-insensitive invite code lookup.
	gs, err := s.sessionRepo.GetByInviteCode(r.Context(), strings.ToUpper(req.InviteCode))
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Invalid invite code", nil)
			return
		}
		s.logger.Error("session: get by invite code", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	if gs.Status != "lobby" {
		s.writeError(w, http.StatusConflict, "CONFLICT", "Can only join sessions in lobby state", nil)
		return
	}

	if gs.GMID == claims.UserID {
		s.writeError(w, http.StatusConflict, "CONFLICT", "GM cannot join their own session as a player", nil)
		return
	}

	sp, err := s.sessionRepo.AddPlayer(r.Context(), gs.ID, claims.UserID)
	if err != nil {
		if errors.Is(err, apperror.ErrDuplicate) {
			// Idempotent: return the session so the client can navigate.
			s.writeJSON(w, http.StatusOK, toSessionResponse(gs))
			return
		}
		s.logger.Error("session: add player", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	_ = sp // player record created successfully

	s.writeJSON(w, http.StatusCreated, toSessionResponse(gs))
}

func (s *Server) handleListSessionPlayers(w http.ResponseWriter, r *http.Request) {
	claims := UserClaimsFromContext(r.Context())
	id := r.PathValue("id")

	if !isValidUUID(id) {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid session ID", nil)
		return
	}

	gs, err := s.sessionRepo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Session not found", nil)
			return
		}
		s.logger.Error("session: get for list players", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	if !s.isSessionParticipant(r, gs.ID, claims.UserID, gs.GMID) {
		s.writeError(w, http.StatusForbidden, "FORBIDDEN", "You do not have access to this session", nil)
		return
	}

	players, err := s.sessionRepo.ListPlayers(r.Context(), id)
	if err != nil {
		s.logger.Error("session: list players", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	items := make([]SessionPlayerResponse, 0, len(players))
	for _, sp := range players {
		resp := toSessionPlayerResponse(sp)
		// Enrich with username (best-effort, skip on error)
		func() {
			defer func() { recover() }() //nolint:errcheck
			if u, uErr := s.authRepo.GetUserByID(r.Context(), sp.UserID); uErr == nil {
				resp.Username = u.Username
			}
		}()
		// Enrich with character name (best-effort, skip on error)
		if sp.CharacterID != nil {
			func() {
				defer func() { recover() }() //nolint:errcheck
				if c, cErr := s.characterRepo.GetByID(r.Context(), *sp.CharacterID); cErr == nil {
					resp.CharacterName = c.Name
					resp.CharacterNotes = c.Notes
				}
			}()
		}
		items = append(items, resp)
	}

	s.writeJSON(w, http.StatusOK, SessionPlayerListResponse{Players: items})
}

func (s *Server) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	claims := UserClaimsFromContext(r.Context())
	id := r.PathValue("id")

	if !isValidUUID(id) {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid session ID", nil)
		return
	}

	gs, err := s.sessionRepo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Session not found", nil)
			return
		}
		s.logger.Error("session: get for delete", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	if gs.GMID != claims.UserID {
		s.writeError(w, http.StatusForbidden, "FORBIDDEN", "Only the GM can delete the session", nil)
		return
	}

	// Only lobby sessions can be deleted; active/paused must be ended first.
	if gs.Status != "lobby" && gs.Status != "completed" && gs.Status != "abandoned" {
		s.writeError(w, http.StatusConflict, "CONFLICT", "Only lobby, completed, or abandoned sessions can be deleted", nil)
		return
	}

	// Remove WebSocket room if it exists.
	if s.hub != nil {
		s.hub.RemoveRoom(gs.ID)
	}

	if err := s.sessionRepo.Delete(r.Context(), id); err != nil {
		s.logger.Error("session: delete", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleRemoveSessionPlayer(w http.ResponseWriter, r *http.Request) {
	claims := UserClaimsFromContext(r.Context())
	id := r.PathValue("id")
	targetUserID := r.PathValue("userId")

	if !isValidUUID(id) {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid session ID", nil)
		return
	}

	gs, err := s.sessionRepo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Session not found", nil)
			return
		}
		s.logger.Error("session: get for remove player", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	// Only GM or the player themselves can remove.
	if gs.GMID != claims.UserID && claims.UserID != targetUserID {
		s.writeError(w, http.StatusForbidden, "FORBIDDEN", "Only the GM or the player themselves can remove a player", nil)
		return
	}

	// Cannot remove players from terminal states.
	if gs.Status == "completed" || gs.Status == "abandoned" {
		s.writeError(w, http.StatusConflict, "CONFLICT", "Cannot remove players from a completed session", nil)
		return
	}

	if err := s.sessionRepo.RemovePlayer(r.Context(), id, targetUserID); err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Player not found in session", nil)
			return
		}
		s.logger.Error("session: remove player", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleListSessionEvents returns all game events for a completed session.
// Only the GM or session participants can access replay data.
func (s *Server) handleListSessionEvents(w http.ResponseWriter, r *http.Request) {
	claims := UserClaimsFromContext(r.Context())
	id := r.PathValue("id")
	if id == "" {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Missing session ID", nil)
		return
	}

	gs, err := s.sessionRepo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Session not found", nil)
			return
		}
		s.logger.Error("session: get for events", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	// Only participants can view events
	if !s.isSessionParticipant(r, id, claims.UserID, gs.GMID) {
		s.writeError(w, http.StatusForbidden, "FORBIDDEN", "Not a participant of this session", nil)
		return
	}

	// Only completed sessions can be replayed
	if gs.Status != "completed" {
		s.writeError(w, http.StatusConflict, "CONFLICT", "Only completed sessions can be replayed", nil)
		return
	}

	events, err := s.eventRepo.ListEventsSince(r.Context(), id, 0)
	if err != nil {
		s.logger.Error("session: list events", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	type eventResponse struct {
		ID        string          `json:"id"`
		Sequence  int64           `json:"sequence"`
		Type      string          `json:"type"`
		ActorID   *string         `json:"actorId,omitempty"`
		Payload   json.RawMessage `json:"payload"`
		CreatedAt string          `json:"createdAt"`
	}

	resp := make([]eventResponse, len(events))
	for i, e := range events {
		resp[i] = eventResponse{
			ID:        e.ID,
			Sequence:  e.Sequence,
			Type:      e.Type,
			ActorID:   e.ActorID,
			Payload:   e.Payload,
			CreatedAt: e.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	s.writeJSON(w, http.StatusOK, resp)
}
