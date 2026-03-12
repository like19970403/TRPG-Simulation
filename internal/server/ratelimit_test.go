package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestRateLimit_AllowsBelowLimit(t *testing.T) {
	store := newRateLimiterStore(rateLimitConfig{rate: rate.Every(time.Second), burst: 5})
	handler := rateLimit(store, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		handler(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("request %d: status = %d, want %d", i, w.Code, http.StatusOK)
		}
	}
}

func TestRateLimit_BlocksExcess(t *testing.T) {
	store := newRateLimiterStore(rateLimitConfig{rate: rate.Every(time.Minute), burst: 2})
	handler := rateLimit(store, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// First 2 should succeed (burst)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		req.RemoteAddr = "10.0.0.1:9999"
		w := httptest.NewRecorder()
		handler(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i, w.Code)
		}
	}

	// 3rd should be rate-limited
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.RemoteAddr = "10.0.0.1:9999"
	w := httptest.NewRecorder()
	handler(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want %d", w.Code, http.StatusTooManyRequests)
	}
	if ra := w.Header().Get("Retry-After"); ra == "" {
		t.Error("Retry-After header should be set")
	}
}

func TestRateLimit_DifferentIPsIndependent(t *testing.T) {
	store := newRateLimiterStore(rateLimitConfig{rate: rate.Every(time.Minute), burst: 1})
	handler := rateLimit(store, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// IP A uses its burst
	req1 := httptest.NewRequest(http.MethodPost, "/test", nil)
	req1.RemoteAddr = "1.1.1.1:1234"
	w1 := httptest.NewRecorder()
	handler(w1, req1)
	if w1.Code != http.StatusOK {
		t.Errorf("IP A first request: got %d", w1.Code)
	}

	// IP B should still be allowed
	req2 := httptest.NewRequest(http.MethodPost, "/test", nil)
	req2.RemoteAddr = "2.2.2.2:5678"
	w2 := httptest.NewRecorder()
	handler(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("IP B first request: got %d", w2.Code)
	}
}

func TestExtractIP_XForwardedFor(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// Last IP is the one appended by the trusted reverse proxy (Caddy).
	req.Header.Set("X-Forwarded-For", "203.0.113.50, 70.41.3.18")
	req.RemoteAddr = "127.0.0.1:8080"

	ip := extractIP(req)
	if ip != "70.41.3.18" {
		t.Errorf("IP = %q, want %q", ip, "70.41.3.18")
	}
}

func TestExtractIP_XForwardedFor_Single(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.50")
	req.RemoteAddr = "127.0.0.1:8080"

	ip := extractIP(req)
	if ip != "203.0.113.50" {
		t.Errorf("IP = %q, want %q", ip, "203.0.113.50")
	}
}

func TestExtractIP_Fallback(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.100:54321"

	ip := extractIP(req)
	if ip != "192.168.1.100" {
		t.Errorf("IP = %q, want %q", ip, "192.168.1.100")
	}
}
