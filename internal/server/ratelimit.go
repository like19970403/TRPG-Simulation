package server

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// rateLimitConfig defines rate limit parameters for an endpoint.
type rateLimitConfig struct {
	rate  rate.Limit
	burst int
}

// ipLimiter holds a rate limiter and its last-seen time for cleanup.
type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// rateLimiterStore manages per-IP rate limiters with automatic cleanup.
type rateLimiterStore struct {
	mu       sync.Mutex
	limiters map[string]*ipLimiter
	cfg      rateLimitConfig
	stopCh   chan struct{}
}

func newRateLimiterStore(cfg rateLimitConfig) *rateLimiterStore {
	s := &rateLimiterStore{
		limiters: make(map[string]*ipLimiter),
		cfg:      cfg,
		stopCh:   make(chan struct{}),
	}
	go s.cleanup()
	return s
}

// Close stops the background cleanup goroutine.
func (s *rateLimiterStore) Close() {
	close(s.stopCh)
}

func (s *rateLimiterStore) getLimiter(ip string) *rate.Limiter {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.limiters[ip]
	if !exists {
		entry = &ipLimiter{
			limiter: rate.NewLimiter(s.cfg.rate, s.cfg.burst),
		}
		s.limiters[ip] = entry
	}
	entry.lastSeen = time.Now()
	return entry.limiter
}

// cleanup removes stale entries every 5 minutes (entries older than 10 minutes).
func (s *rateLimiterStore) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			cutoff := time.Now().Add(-10 * time.Minute)
			for ip, entry := range s.limiters {
				if entry.lastSeen.Before(cutoff) {
					delete(s.limiters, ip)
				}
			}
			s.mu.Unlock()
		case <-s.stopCh:
			return
		}
	}
}

// rateLimit wraps a handler with per-IP rate limiting.
func rateLimit(store *rateLimiterStore, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := extractIP(r)
		limiter := store.getLimiter(ip)
		if !limiter.Allow() {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"RATE_LIMITED","message":"Too many requests. Please try again later."}`))
			return
		}
		next(w, r)
	}
}

// extractIP gets the client IP, respecting X-Forwarded-For from trusted proxy (Caddy).
func extractIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		for i, c := range xff {
			if c == ',' {
				return xff[:i]
			}
		}
		return xff
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
