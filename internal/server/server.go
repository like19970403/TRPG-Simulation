package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"encoding/json"

	"github.com/like19970403/TRPG-Simulation/internal/auth"
	"github.com/like19970403/TRPG-Simulation/internal/config"
	"github.com/like19970403/TRPG-Simulation/internal/game"
	"github.com/like19970403/TRPG-Simulation/internal/realtime"
	"github.com/like19970403/TRPG-Simulation/internal/scenario"
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

// SessionRepository defines the interface for game session database operations.
type SessionRepository interface {
	Create(ctx context.Context, scenarioID, gmID string) (*game.GameSession, error)
	GetByID(ctx context.Context, id string) (*game.GameSession, error)
	ListByGM(ctx context.Context, gmID string, limit, offset int) ([]*game.GameSession, int, error)
	UpdateStatus(ctx context.Context, id, newStatus string) (*game.GameSession, error)
	GetByInviteCode(ctx context.Context, code string) (*game.GameSession, error)
	AddPlayer(ctx context.Context, sessionID, userID string) (*game.SessionPlayer, error)
	ListPlayers(ctx context.Context, sessionID string) ([]*game.SessionPlayer, error)
	RemovePlayer(ctx context.Context, sessionID, userID string) error
	GetPlayer(ctx context.Context, sessionID, userID string) (*game.SessionPlayer, error)
}

// ScenarioRepository defines the interface for scenario database operations.
type ScenarioRepository interface {
	Create(ctx context.Context, authorID, title, description string, content json.RawMessage) (*scenario.Scenario, error)
	ListByAuthor(ctx context.Context, authorID string, limit, offset int) ([]*scenario.Scenario, int, error)
	GetByID(ctx context.Context, id string) (*scenario.Scenario, error)
	Update(ctx context.Context, id, title, description string, content json.RawMessage) (*scenario.Scenario, error)
	Delete(ctx context.Context, id string) error
	UpdateStatus(ctx context.Context, id, newStatus string) (*scenario.Scenario, error)
}

// Server wraps the HTTP server, database pool, and logger.
type Server struct {
	httpServer   *http.Server
	handler      http.Handler
	pool         *pgxpool.Pool
	logger       *slog.Logger
	authRepo     AuthRepository
	scenarioRepo ScenarioRepository
	sessionRepo  SessionRepository
	hub          *realtime.Hub
	jwtSecret    string
	accessTTL    time.Duration
	refreshTTL   time.Duration
	bcryptCost   int
}

// New creates a new Server with routes and middleware configured.
func New(cfg *config.Config, pool *pgxpool.Pool, logger *slog.Logger) *Server {
	repo := game.NewRepository(pool)
	s := &Server{
		pool:         pool,
		logger:       logger,
		authRepo:     auth.NewRepository(pool),
		scenarioRepo: scenario.NewRepository(pool),
		sessionRepo:  repo,
		jwtSecret:    cfg.JWTSecret,
		accessTTL:    time.Duration(cfg.JWTAccessTokenTTL) * time.Second,
		refreshTTL:   time.Duration(cfg.JWTRefreshTokenTTL) * time.Second,
		bcryptCost:   cfg.BcryptCost,
	}
	if pool != nil {
		s.hub = realtime.NewHub(repo, logger)
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

	// Scenarios — protected
	mux.HandleFunc("POST /api/v1/scenarios", s.requireAuth(s.handleCreateScenario))
	mux.HandleFunc("GET /api/v1/scenarios", s.requireAuth(s.handleListScenarios))
	mux.HandleFunc("GET /api/v1/scenarios/{id}", s.requireAuth(s.handleGetScenario))
	mux.HandleFunc("PUT /api/v1/scenarios/{id}", s.requireAuth(s.handleUpdateScenario))
	mux.HandleFunc("DELETE /api/v1/scenarios/{id}", s.requireAuth(s.handleDeleteScenario))
	mux.HandleFunc("POST /api/v1/scenarios/{id}/publish", s.requireAuth(s.handlePublishScenario))
	mux.HandleFunc("POST /api/v1/scenarios/{id}/archive", s.requireAuth(s.handleArchiveScenario))

	// Sessions — protected
	mux.HandleFunc("POST /api/v1/sessions", s.requireAuth(s.handleCreateSession))
	mux.HandleFunc("GET /api/v1/sessions", s.requireAuth(s.handleListSessions))
	mux.HandleFunc("GET /api/v1/sessions/{id}", s.requireAuth(s.handleGetSession))
	mux.HandleFunc("POST /api/v1/sessions/{id}/start", s.requireAuth(s.handleStartSession))
	mux.HandleFunc("POST /api/v1/sessions/{id}/pause", s.requireAuth(s.handlePauseSession))
	mux.HandleFunc("POST /api/v1/sessions/{id}/resume", s.requireAuth(s.handleResumeSession))
	mux.HandleFunc("POST /api/v1/sessions/{id}/end", s.requireAuth(s.handleEndSession))
	mux.HandleFunc("POST /api/v1/sessions/join", s.requireAuth(s.handleJoinSession))
	mux.HandleFunc("GET /api/v1/sessions/{id}/players", s.requireAuth(s.handleListSessionPlayers))
	mux.HandleFunc("DELETE /api/v1/sessions/{id}/players/{userId}", s.requireAuth(s.handleRemoveSessionPlayer))

	// WebSocket — auth via query param token
	mux.HandleFunc("GET /api/v1/sessions/{id}/ws", s.handleWS)
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

// Shutdown gracefully stops the HTTP server and the WebSocket hub.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.hub != nil {
		s.hub.Stop()
	}
	return s.httpServer.Shutdown(ctx)
}
