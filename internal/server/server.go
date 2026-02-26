package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/like19970403/TRPG-Simulation/internal/auth"
	"github.com/like19970403/TRPG-Simulation/internal/config"
)

// AuthRepository defines the interface for auth database operations.
type AuthRepository interface {
	CreateUser(ctx context.Context, username, email, passwordHash string) (*auth.User, error)
	GetUserByEmail(ctx context.Context, email string) (*auth.User, error)
	GetUserByID(ctx context.Context, id string) (*auth.User, error)
	StoreRefreshToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error
	GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*auth.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, tokenID string) error
	RevokeAllUserRefreshTokens(ctx context.Context, userID string) error
}

// Server wraps the HTTP server, database pool, and logger.
type Server struct {
	httpServer *http.Server
	handler    http.Handler
	pool       *pgxpool.Pool
	logger     *slog.Logger
	authRepo   AuthRepository
	jwtSecret  string
	accessTTL  time.Duration
	refreshTTL time.Duration
	bcryptCost int
}

// New creates a new Server with routes and middleware configured.
func New(cfg *config.Config, pool *pgxpool.Pool, logger *slog.Logger) *Server {
	s := &Server{
		pool:       pool,
		logger:     logger,
		authRepo:   auth.NewRepository(pool),
		jwtSecret:  cfg.JWTSecret,
		accessTTL:  time.Duration(cfg.JWTAccessTokenTTL) * time.Second,
		refreshTTL: time.Duration(cfg.JWTRefreshTokenTTL) * time.Second,
		bcryptCost: cfg.BcryptCost,
	}

	mux := http.NewServeMux()
	s.registerRoutes(mux)

	handler := s.requestID(s.logging(s.recovery(mux)))
	s.handler = handler

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

func (s *Server) registerRoutes(mux *http.ServeMux) {
	// Infrastructure (unversioned, no auth)
	mux.HandleFunc("GET /api/health", s.handleHealth)

	// Auth — public
	mux.HandleFunc("POST /api/v1/users", s.handleRegister)
	mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/v1/auth/refresh", s.handleRefresh)

	// Auth — protected
	mux.HandleFunc("POST /api/v1/auth/logout", s.requireAuth(s.handleLogout))
}

// Handler returns the top-level HTTP handler (for testing).
func (s *Server) Handler() http.Handler {
	return s.handler
}

// Start begins listening for HTTP requests.
func (s *Server) Start() error {
	s.logger.Info("server starting", "addr", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
