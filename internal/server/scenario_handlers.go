package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func isValidUUID(s string) bool {
	return uuidRegex.MatchString(strings.ToLower(s))
}

func parsePagination(r *http.Request) (limit, offset int, err error) {
	limit = 20
	offset = 0

	if v := r.URL.Query().Get("limit"); v != "" {
		limit, err = strconv.Atoi(v)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid limit")
		}
		if limit < 0 || limit > 100 {
			return 0, 0, fmt.Errorf("limit must be between 0 and 100")
		}
	}

	if v := r.URL.Query().Get("offset"); v != "" {
		offset, err = strconv.Atoi(v)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid offset")
		}
		if offset < 0 {
			return 0, 0, fmt.Errorf("offset must be non-negative")
		}
	}

	return limit, offset, nil
}

func validateCreateScenario(req CreateScenarioRequest) []ErrorDetail {
	var errs []ErrorDetail
	if len(req.Title) == 0 || len(req.Title) > 200 {
		errs = append(errs, ErrorDetail{Field: "title", Reason: "must be between 1 and 200 characters"})
	}
	if len(req.Content) == 0 {
		errs = append(errs, ErrorDetail{Field: "content", Reason: "must not be empty"})
	} else {
		var obj map[string]any
		if err := json.Unmarshal(req.Content, &obj); err != nil {
			errs = append(errs, ErrorDetail{Field: "content", Reason: "must be a valid JSON object"})
		}
	}
	return errs
}

func validateUpdateScenario(req UpdateScenarioRequest) []ErrorDetail {
	var errs []ErrorDetail
	if len(req.Title) == 0 || len(req.Title) > 200 {
		errs = append(errs, ErrorDetail{Field: "title", Reason: "must be between 1 and 200 characters"})
	}
	if len(req.Content) > 0 {
		var obj map[string]any
		if err := json.Unmarshal(req.Content, &obj); err != nil {
			errs = append(errs, ErrorDetail{Field: "content", Reason: "must be a valid JSON object"})
		}
	}
	return errs
}

func (s *Server) handleCreateScenario(w http.ResponseWriter, r *http.Request) {
	claims := UserClaimsFromContext(r.Context())

	var req CreateScenarioRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	if errs := validateCreateScenario(req); len(errs) > 0 {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Request validation failed", errs)
		return
	}

	sc, err := s.scenarioRepo.Create(r.Context(), claims.UserID, req.Title, req.Description, req.Content)
	if err != nil {
		s.logger.Error("scenario: create", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	s.writeJSON(w, http.StatusCreated, toScenarioResponse(sc))
}

func (s *Server) handleListScenarios(w http.ResponseWriter, r *http.Request) {
	claims := UserClaimsFromContext(r.Context())

	limit, offset, err := parsePagination(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}

	scenarios, total, err := s.scenarioRepo.ListByAuthor(r.Context(), claims.UserID, limit, offset)
	if err != nil {
		s.logger.Error("scenario: list", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	items := make([]ScenarioResponse, 0, len(scenarios))
	for _, sc := range scenarios {
		items = append(items, toScenarioResponse(sc))
	}

	s.writeJSON(w, http.StatusOK, ScenarioListResponse{
		Scenarios: items,
		Total:     total,
		Limit:     limit,
		Offset:    offset,
	})
}

func (s *Server) handleGetScenario(w http.ResponseWriter, r *http.Request) {
	claims := UserClaimsFromContext(r.Context())
	id := r.PathValue("id")

	if !isValidUUID(id) {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid scenario ID", nil)
		return
	}

	sc, err := s.scenarioRepo.GetByID(r.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Scenario not found", nil)
			return
		}
		s.logger.Error("scenario: get", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	if sc.AuthorID != claims.UserID {
		s.writeError(w, http.StatusForbidden, "FORBIDDEN", "You do not have access to this scenario", nil)
		return
	}

	s.writeJSON(w, http.StatusOK, toScenarioResponse(sc))
}

func (s *Server) handleUpdateScenario(w http.ResponseWriter, r *http.Request) {
	claims := UserClaimsFromContext(r.Context())
	id := r.PathValue("id")

	if !isValidUUID(id) {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid scenario ID", nil)
		return
	}

	var req UpdateScenarioRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	// Fetch existing scenario for auth + status checks
	existing, err := s.scenarioRepo.GetByID(r.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Scenario not found", nil)
			return
		}
		s.logger.Error("scenario: get for update", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	if existing.AuthorID != claims.UserID {
		s.writeError(w, http.StatusForbidden, "FORBIDDEN", "You do not have access to this scenario", nil)
		return
	}

	if existing.Status != "draft" {
		s.writeError(w, http.StatusConflict, "CONFLICT", "Only draft scenarios can be updated", nil)
		return
	}

	if errs := validateUpdateScenario(req); len(errs) > 0 {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Request validation failed", errs)
		return
	}

	sc, err := s.scenarioRepo.Update(r.Context(), id, req.Title, req.Description, req.Content)
	if err != nil {
		s.logger.Error("scenario: update", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	s.writeJSON(w, http.StatusOK, toScenarioResponse(sc))
}

func (s *Server) handleDeleteScenario(w http.ResponseWriter, r *http.Request) {
	claims := UserClaimsFromContext(r.Context())
	id := r.PathValue("id")

	if !isValidUUID(id) {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid scenario ID", nil)
		return
	}

	existing, err := s.scenarioRepo.GetByID(r.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Scenario not found", nil)
			return
		}
		s.logger.Error("scenario: get for delete", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	if existing.AuthorID != claims.UserID {
		s.writeError(w, http.StatusForbidden, "FORBIDDEN", "You do not have access to this scenario", nil)
		return
	}

	if existing.Status != "draft" {
		s.writeError(w, http.StatusConflict, "CONFLICT", "Only draft scenarios can be deleted", nil)
		return
	}

	if err := s.scenarioRepo.Delete(r.Context(), id); err != nil {
		s.logger.Error("scenario: delete", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handlePublishScenario(w http.ResponseWriter, r *http.Request) {
	claims := UserClaimsFromContext(r.Context())
	id := r.PathValue("id")

	if !isValidUUID(id) {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid scenario ID", nil)
		return
	}

	existing, err := s.scenarioRepo.GetByID(r.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Scenario not found", nil)
			return
		}
		s.logger.Error("scenario: get for publish", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	if existing.AuthorID != claims.UserID {
		s.writeError(w, http.StatusForbidden, "FORBIDDEN", "You do not have access to this scenario", nil)
		return
	}

	if existing.Status != "draft" {
		s.writeError(w, http.StatusConflict, "CONFLICT", "Only draft scenarios can be published", nil)
		return
	}

	sc, err := s.scenarioRepo.UpdateStatus(r.Context(), id, "published")
	if err != nil {
		s.logger.Error("scenario: publish", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	s.writeJSON(w, http.StatusOK, toScenarioResponse(sc))
}

func (s *Server) handleArchiveScenario(w http.ResponseWriter, r *http.Request) {
	claims := UserClaimsFromContext(r.Context())
	id := r.PathValue("id")

	if !isValidUUID(id) {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid scenario ID", nil)
		return
	}

	existing, err := s.scenarioRepo.GetByID(r.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Scenario not found", nil)
			return
		}
		s.logger.Error("scenario: get for archive", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	if existing.AuthorID != claims.UserID {
		s.writeError(w, http.StatusForbidden, "FORBIDDEN", "You do not have access to this scenario", nil)
		return
	}

	if existing.Status != "published" {
		s.writeError(w, http.StatusConflict, "CONFLICT", "Only published scenarios can be archived", nil)
		return
	}

	sc, err := s.scenarioRepo.UpdateStatus(r.Context(), id, "archived")
	if err != nil {
		s.logger.Error("scenario: archive", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	s.writeJSON(w, http.StatusOK, toScenarioResponse(sc))
}
