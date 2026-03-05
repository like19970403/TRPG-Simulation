package server

import (
	"encoding/json"
	"net"
	"net/http"
	"time"

	"github.com/like19970403/TRPG-Simulation/internal/database"
)

// HealthResponse is the full JSON response for the detailed health check endpoint.
type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Database  string `json:"database"`
}

// PublicHealthResponse is the JSON response for the public health endpoint.
type PublicHealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	dbOK := s.checkDB(r)

	status := "ok"
	statusCode := http.StatusOK
	if !dbOK {
		status = "degraded"
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(PublicHealthResponse{
		Status:    status,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleHealthDetail(w http.ResponseWriter, r *http.Request) {
	ip := extractIPFromRemoteAddr(r)
	if !isPrivateIP(ip) {
		http.NotFound(w, r)
		return
	}

	dbStatus := "ok"
	if !s.checkDB(r) {
		dbStatus = "error"
	}

	status := "ok"
	statusCode := http.StatusOK
	if dbStatus != "ok" {
		status = "degraded"
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(HealthResponse{
		Status:    status,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Database:  dbStatus,
	})
}

func (s *Server) checkDB(r *http.Request) bool {
	if s.pool == nil {
		return false
	}
	if err := database.HealthCheck(r.Context(), s.pool); err != nil {
		s.logger.Error("health check: database unhealthy", "error", err)
		return false
	}
	return true
}

// extractIPFromRemoteAddr extracts the IP from r.RemoteAddr (not X-Forwarded-For).
func extractIPFromRemoteAddr(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
