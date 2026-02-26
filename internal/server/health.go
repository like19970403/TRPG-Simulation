package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/like19970403/TRPG-Simulation/internal/database"
)

// HealthResponse is the JSON response for the health check endpoint.
type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Database  string `json:"database"`
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	dbStatus := "ok"
	if s.pool == nil {
		dbStatus = "error"
	} else if err := database.HealthCheck(r.Context(), s.pool); err != nil {
		dbStatus = "error"
		s.logger.Error("health check: database unhealthy", "error", err)
	}

	overallStatus := "ok"
	statusCode := http.StatusOK
	if dbStatus != "ok" {
		overallStatus = "degraded"
		statusCode = http.StatusServiceUnavailable
	}

	resp := HealthResponse{
		Status:    overallStatus,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Database:  dbStatus,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		s.logger.Error("health check: failed to encode response", "error", err)
	}
}
