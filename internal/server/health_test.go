package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleHealth_PublicResponse(t *testing.T) {
	srv := New(testConfig(), nil, testLogger())

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()

	// With nil pool, HealthCheck will fail → degraded
	srv.handleHealth(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}

	var body PublicHealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("json decode error: %v", err)
	}

	if body.Status != "degraded" {
		t.Errorf("status = %q, want %q", body.Status, "degraded")
	}
	if body.Timestamp == "" {
		t.Error("timestamp should not be empty")
	}
}

func TestHandleHealth_NoDatabaseField(t *testing.T) {
	srv := New(testConfig(), nil, testLogger())

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()

	srv.handleHealth(w, req)

	// Decode as raw map to check no "database" field exists
	var raw map[string]interface{}
	json.NewDecoder(w.Body).Decode(&raw)
	if _, ok := raw["database"]; ok {
		t.Error("public health endpoint should not expose database field")
	}
}

func TestHandleHealth_ContentType(t *testing.T) {
	srv := New(testConfig(), nil, testLogger())

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()

	srv.handleHealth(w, req)

	ct := w.Result().Header.Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}
}

func TestHandleHealthDetail_PrivateIP(t *testing.T) {
	srv := New(testConfig(), nil, testLogger())

	req := httptest.NewRequest(http.MethodGet, "/api/health/detail", nil)
	req.RemoteAddr = "127.0.0.1:54321"
	w := httptest.NewRecorder()

	srv.handleHealthDetail(w, req)

	if w.Code == http.StatusNotFound {
		t.Error("localhost should be allowed to access health detail")
	}

	var body HealthResponse
	json.NewDecoder(w.Body).Decode(&body)
	if body.Database == "" {
		t.Error("detail response should include database field")
	}
}

func TestHandleHealthDetail_PublicIP(t *testing.T) {
	srv := New(testConfig(), nil, testLogger())

	req := httptest.NewRequest(http.MethodGet, "/api/health/detail", nil)
	req.RemoteAddr = "203.0.113.50:54321"
	w := httptest.NewRecorder()

	srv.handleHealthDetail(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("public IP should get 404, got %d", w.Code)
	}
}
