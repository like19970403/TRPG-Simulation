package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/like19970403/TRPG-Simulation/internal/config"
)

func TestRequestIDMiddleware(t *testing.T) {
	srv := New(testConfig(), nil, testLogger())

	handler := srv.requestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request ID is in context
		id := RequestIDFromContext(r.Context())
		if id == "" {
			t.Error("request ID should be set in context")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Verify X-Request-ID header is set in response
	reqID := w.Result().Header.Get("X-Request-ID")
	if reqID == "" {
		t.Error("X-Request-ID header should be set")
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	srv := New(testConfig(), nil, testLogger())

	handler := srv.recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	// Should not panic
	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Result().StatusCode, http.StatusInternalServerError)
	}
}

func TestLoggingMiddleware(t *testing.T) {
	srv := New(testConfig(), nil, testLogger())

	called := false
	handler := srv.logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("inner handler should have been called")
	}
	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Result().StatusCode, http.StatusOK)
	}
}

func TestSecurityHeadersMiddleware(t *testing.T) {
	srv := New(testConfig(), nil, testLogger())
	handler := srv.securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	headers := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":       "DENY",
		"X-XSS-Protection":      "0",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	}
	for key, want := range headers {
		if got := w.Header().Get(key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
	if csp := w.Header().Get("Content-Security-Policy"); csp == "" {
		t.Error("CSP header should be set")
	}
	if pp := w.Header().Get("Permissions-Policy"); pp == "" {
		t.Error("Permissions-Policy header should be set")
	}
	// testConfig() has CookieSecure=true, so HSTS should be present
	if hsts := w.Header().Get("Strict-Transport-Security"); hsts == "" {
		t.Error("HSTS header should be set when CookieSecure is true")
	}
}

func TestSecurityHeadersMiddleware_NoHSTS_WhenInsecure(t *testing.T) {
	cfg := testConfig()
	cfg.CookieSecure = false
	srv := New(cfg, nil, testLogger())
	handler := srv.securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if hsts := w.Header().Get("Strict-Transport-Security"); hsts != "" {
		t.Error("HSTS header should NOT be set when CookieSecure is false")
	}
}

func TestBodyLimitMiddleware_EnforcesLimit(t *testing.T) {
	cfg := testConfig()
	cfg.MaxJSONBodySize = 100
	srv := New(cfg, nil, testLogger())

	handler := srv.bodyLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := make([]byte, 200)
		_, err := r.Body.Read(body)
		if err == nil {
			t.Error("expected error reading oversized body")
		}
		w.WriteHeader(http.StatusOK)
	}))

	largeBody := strings.NewReader(strings.Repeat("x", 200))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/test", largeBody)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
}

func TestBodyLimitMiddleware_SkipsUpload(t *testing.T) {
	cfg := testConfig()
	cfg.MaxJSONBodySize = 100
	srv := New(cfg, nil, testLogger())

	bodyCaptured := false
	handler := srv.bodyLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := make([]byte, 200)
		n, _ := r.Body.Read(body)
		if n == 200 {
			bodyCaptured = true
		}
		w.WriteHeader(http.StatusOK)
	}))

	largeBody := strings.NewReader(strings.Repeat("x", 200))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/upload", largeBody)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !bodyCaptured {
		t.Error("upload endpoint should not have body limit applied")
	}
}

func TestCORSMiddleware_AllowedOrigin(t *testing.T) {
	cfg := testConfig()
	cfg.AllowedOrigins = "https://trpg.example.com"
	srv := New(cfg, nil, testLogger())

	handler := srv.cors(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://trpg.example.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://trpg.example.com" {
		t.Errorf("ACAO = %q, want %q", got, "https://trpg.example.com")
	}
	if got := w.Header().Get("Vary"); got != "Origin" {
		t.Errorf("Vary = %q, want %q", got, "Origin")
	}
}

func TestCORSMiddleware_DisallowedOrigin(t *testing.T) {
	cfg := testConfig()
	cfg.AllowedOrigins = "https://trpg.example.com"
	srv := New(cfg, nil, testLogger())

	handler := srv.cors(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://evil.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("ACAO should be empty for disallowed origin, got %q", got)
	}
}

func TestCORSMiddleware_Preflight(t *testing.T) {
	cfg := testConfig()
	cfg.AllowedOrigins = "https://trpg.example.com"
	srv := New(cfg, nil, testLogger())

	innerCalled := false
	handler := srv.cors(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		innerCalled = true
	}))

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://trpg.example.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
	if innerCalled {
		t.Error("preflight should not call inner handler")
	}
	if got := w.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Error("ACAM header should be set on preflight")
	}
}

func TestCORSMiddleware_NoOriginConfig(t *testing.T) {
	cfg := testConfig()
	cfg.AllowedOrigins = ""
	srv := New(cfg, nil, testLogger())

	handler := srv.cors(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://trpg.example.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Error("no CORS headers should be set when AllowedOrigins is empty")
	}
}

func helperConfigWithOrigins(origins string) *config.Config {
	cfg := testConfig()
	cfg.AllowedOrigins = origins
	return cfg
}
