package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/like19970403/TRPG-Simulation/internal/game"
	"github.com/like19970403/TRPG-Simulation/internal/scenario"
)

// mockSessionRepo implements SessionRepository for unit tests.
type mockSessionRepo struct {
	createFn          func(ctx context.Context, scenarioID, gmID string) (*game.GameSession, error)
	getByIDFn         func(ctx context.Context, id string) (*game.GameSession, error)
	listByGMFn        func(ctx context.Context, gmID string, limit, offset int) ([]*game.GameSession, int, error)
	listByPlayerFn    func(ctx context.Context, userID string, limit, offset int) ([]*game.GameSession, int, error)
	updateStatusFn    func(ctx context.Context, id, newStatus string) (*game.GameSession, error)
	getByInviteCodeFn func(ctx context.Context, code string) (*game.GameSession, error)
	addPlayerFn       func(ctx context.Context, sessionID, userID string) (*game.SessionPlayer, error)
	listPlayersFn     func(ctx context.Context, sessionID string) ([]*game.SessionPlayer, error)
	removePlayerFn    func(ctx context.Context, sessionID, userID string) error
	getPlayerFn       func(ctx context.Context, sessionID, userID string) (*game.SessionPlayer, error)
	setCharacterIDFn  func(ctx context.Context, sessionID, userID, characterID string) (*game.SessionPlayer, error)
	deleteFn          func(ctx context.Context, id string) error
}

func (m *mockSessionRepo) Create(ctx context.Context, scenarioID, gmID string) (*game.GameSession, error) {
	return m.createFn(ctx, scenarioID, gmID)
}

func (m *mockSessionRepo) GetByID(ctx context.Context, id string) (*game.GameSession, error) {
	return m.getByIDFn(ctx, id)
}

func (m *mockSessionRepo) ListByGM(ctx context.Context, gmID string, limit, offset int) ([]*game.GameSession, int, error) {
	return m.listByGMFn(ctx, gmID, limit, offset)
}

func (m *mockSessionRepo) ListByPlayer(ctx context.Context, userID string, limit, offset int) ([]*game.GameSession, int, error) {
	if m.listByPlayerFn != nil {
		return m.listByPlayerFn(ctx, userID, limit, offset)
	}
	return nil, 0, nil
}

func (m *mockSessionRepo) UpdateStatus(ctx context.Context, id, newStatus string) (*game.GameSession, error) {
	return m.updateStatusFn(ctx, id, newStatus)
}

func (m *mockSessionRepo) GetByInviteCode(ctx context.Context, code string) (*game.GameSession, error) {
	return m.getByInviteCodeFn(ctx, code)
}

func (m *mockSessionRepo) AddPlayer(ctx context.Context, sessionID, userID string) (*game.SessionPlayer, error) {
	return m.addPlayerFn(ctx, sessionID, userID)
}

func (m *mockSessionRepo) ListPlayers(ctx context.Context, sessionID string) ([]*game.SessionPlayer, error) {
	return m.listPlayersFn(ctx, sessionID)
}

func (m *mockSessionRepo) RemovePlayer(ctx context.Context, sessionID, userID string) error {
	return m.removePlayerFn(ctx, sessionID, userID)
}

func (m *mockSessionRepo) GetPlayer(ctx context.Context, sessionID, userID string) (*game.SessionPlayer, error) {
	return m.getPlayerFn(ctx, sessionID, userID)
}

func (m *mockSessionRepo) SetCharacterID(ctx context.Context, sessionID, userID, characterID string) (*game.SessionPlayer, error) {
	return m.setCharacterIDFn(ctx, sessionID, userID, characterID)
}

func (m *mockSessionRepo) Delete(ctx context.Context, id string) error {
	return m.deleteFn(ctx, id)
}

func newSessionTestServer(sessionRepo SessionRepository, scenarioRepo ScenarioRepository) *Server {
	cfg := testConfig()
	srv := New(cfg, nil, testLogger())
	srv.sessionRepo = sessionRepo
	if scenarioRepo != nil {
		srv.scenarioRepo = scenarioRepo
	}
	return srv
}

const (
	testSessionID  = "550e8400-e29b-41d4-a716-446655440000"
	testScenarioID = "660e8400-e29b-41d4-a716-446655440001"
	testGMID       = "gm-user-1"
	testPlayerID   = "player-user-1"
	testPlayerID2  = "player-user-2"
	testInviteCode = "ABC123"
)

func sampleSession(gmID, status string) *game.GameSession {
	now := time.Now().UTC()
	return &game.GameSession{
		ID:         testSessionID,
		ScenarioID: testScenarioID,
		GMID:       gmID,
		Status:     status,
		InviteCode: testInviteCode,
		CreatedAt:  now,
	}
}

func samplePlayer(sessionID, userID string) *game.SessionPlayer {
	return &game.SessionPlayer{
		ID:        "sp-1",
		SessionID: sessionID,
		UserID:    userID,
		Status:    "active",
		JoinedAt:  time.Now().UTC(),
	}
}

func publishedScenario(authorID string) *scenario.Scenario {
	now := time.Now().UTC()
	return &scenario.Scenario{
		ID:       testScenarioID,
		AuthorID: authorID,
		Title:    "Published Quest",
		Status:   "published",
		Content:  json.RawMessage(`{"scenes":[]}`),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// --- Create Session tests ---

func TestHandleCreateSession_Success(t *testing.T) {
	scnRepo := &mockScenarioRepo{
		getByIDFn: func(_ context.Context, id string) (*scenario.Scenario, error) {
			return publishedScenario(testGMID), nil
		},
	}
	sessRepo := &mockSessionRepo{
		createFn: func(_ context.Context, scenarioID, gmID string) (*game.GameSession, error) {
			return sampleSession(gmID, "lobby"), nil
		},
	}
	srv := newSessionTestServer(sessRepo, scnRepo)

	body := `{"scenarioId":"` + testScenarioID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions", strings.NewReader(body))
	req = withAuth(req, testGMID, "gm1")
	w := httptest.NewRecorder()

	srv.handleCreateSession(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
	}
	var resp SessionResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Status != "lobby" {
		t.Errorf("Status = %q, want %q", resp.Status, "lobby")
	}
	if resp.InviteCode == "" {
		t.Error("InviteCode should not be empty")
	}
}

func TestHandleCreateSession_InvalidJSON(t *testing.T) {
	srv := newSessionTestServer(&mockSessionRepo{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions", strings.NewReader("{bad"))
	req = withAuth(req, testGMID, "gm1")
	w := httptest.NewRecorder()

	srv.handleCreateSession(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleCreateSession_InvalidScenarioID(t *testing.T) {
	srv := newSessionTestServer(&mockSessionRepo{}, nil)

	body := `{"scenarioId":"not-a-uuid"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions", strings.NewReader(body))
	req = withAuth(req, testGMID, "gm1")
	w := httptest.NewRecorder()

	srv.handleCreateSession(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleCreateSession_ScenarioNotFound(t *testing.T) {
	scnRepo := &mockScenarioRepo{
		getByIDFn: func(_ context.Context, id string) (*scenario.Scenario, error) {
			return nil, errors.New("not found")
		},
	}
	srv := newSessionTestServer(&mockSessionRepo{}, scnRepo)

	body := `{"scenarioId":"` + testScenarioID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions", strings.NewReader(body))
	req = withAuth(req, testGMID, "gm1")
	w := httptest.NewRecorder()

	srv.handleCreateSession(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleCreateSession_ScenarioNotPublished(t *testing.T) {
	scnRepo := &mockScenarioRepo{
		getByIDFn: func(_ context.Context, id string) (*scenario.Scenario, error) {
			s := publishedScenario(testGMID)
			s.Status = "draft"
			return s, nil
		},
	}
	srv := newSessionTestServer(&mockSessionRepo{}, scnRepo)

	body := `{"scenarioId":"` + testScenarioID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions", strings.NewReader(body))
	req = withAuth(req, testGMID, "gm1")
	w := httptest.NewRecorder()

	srv.handleCreateSession(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestHandleCreateSession_RepoError(t *testing.T) {
	scnRepo := &mockScenarioRepo{
		getByIDFn: func(_ context.Context, id string) (*scenario.Scenario, error) {
			return publishedScenario(testGMID), nil
		},
	}
	sessRepo := &mockSessionRepo{
		createFn: func(_ context.Context, scenarioID, gmID string) (*game.GameSession, error) {
			return nil, errors.New("db error")
		},
	}
	srv := newSessionTestServer(sessRepo, scnRepo)

	body := `{"scenarioId":"` + testScenarioID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions", strings.NewReader(body))
	req = withAuth(req, testGMID, "gm1")
	w := httptest.NewRecorder()

	srv.handleCreateSession(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// --- List Sessions tests ---

func TestHandleListSessions_Success(t *testing.T) {
	sessRepo := &mockSessionRepo{
		listByGMFn: func(_ context.Context, gmID string, limit, offset int) ([]*game.GameSession, int, error) {
			s := sampleSession(gmID, "lobby")
			return []*game.GameSession{s}, 1, nil
		},
	}
	srv := newSessionTestServer(sessRepo, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions", nil)
	req = withAuth(req, testGMID, "gm1")
	w := httptest.NewRecorder()

	srv.handleListSessions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var resp SessionListResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Total != 1 {
		t.Errorf("Total = %d, want 1", resp.Total)
	}
}

func TestHandleListSessions_Empty(t *testing.T) {
	sessRepo := &mockSessionRepo{
		listByGMFn: func(_ context.Context, gmID string, limit, offset int) ([]*game.GameSession, int, error) {
			return []*game.GameSession{}, 0, nil
		},
	}
	srv := newSessionTestServer(sessRepo, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions", nil)
	req = withAuth(req, testGMID, "gm1")
	w := httptest.NewRecorder()

	srv.handleListSessions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var resp SessionListResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp.Sessions) != 0 {
		t.Errorf("Sessions count = %d, want 0", len(resp.Sessions))
	}
}

func TestHandleListSessions_InvalidPagination(t *testing.T) {
	srv := newSessionTestServer(&mockSessionRepo{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions?limit=abc", nil)
	req = withAuth(req, testGMID, "gm1")
	w := httptest.NewRecorder()

	srv.handleListSessions(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- Get Session tests ---

func TestHandleGetSession_SuccessAsGM(t *testing.T) {
	sessRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, id string) (*game.GameSession, error) {
			return sampleSession(testGMID, "lobby"), nil
		},
	}
	srv := newSessionTestServer(sessRepo, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/"+testSessionID, nil)
	req.SetPathValue("id", testSessionID)
	req = withAuth(req, testGMID, "gm1")
	w := httptest.NewRecorder()

	srv.handleGetSession(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleGetSession_SuccessAsPlayer(t *testing.T) {
	sessRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, id string) (*game.GameSession, error) {
			return sampleSession(testGMID, "lobby"), nil
		},
		getPlayerFn: func(_ context.Context, sessionID, userID string) (*game.SessionPlayer, error) {
			return samplePlayer(sessionID, userID), nil
		},
	}
	srv := newSessionTestServer(sessRepo, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/"+testSessionID, nil)
	req.SetPathValue("id", testSessionID)
	req = withAuth(req, testPlayerID, "player1")
	w := httptest.NewRecorder()

	srv.handleGetSession(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleGetSession_NotFound(t *testing.T) {
	sessRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, id string) (*game.GameSession, error) {
			return nil, errors.New("game: not found")
		},
	}
	srv := newSessionTestServer(sessRepo, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/"+testSessionID, nil)
	req.SetPathValue("id", testSessionID)
	req = withAuth(req, testGMID, "gm1")
	w := httptest.NewRecorder()

	srv.handleGetSession(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleGetSession_Forbidden(t *testing.T) {
	sessRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, id string) (*game.GameSession, error) {
			return sampleSession(testGMID, "lobby"), nil
		},
		getPlayerFn: func(_ context.Context, sessionID, userID string) (*game.SessionPlayer, error) {
			return nil, errors.New("game: player not found")
		},
	}
	srv := newSessionTestServer(sessRepo, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/"+testSessionID, nil)
	req.SetPathValue("id", testSessionID)
	req = withAuth(req, "stranger", "stranger")
	w := httptest.NewRecorder()

	srv.handleGetSession(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestHandleGetSession_InvalidUUID(t *testing.T) {
	srv := newSessionTestServer(&mockSessionRepo{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/bad-id", nil)
	req.SetPathValue("id", "bad-id")
	req = withAuth(req, testGMID, "gm1")
	w := httptest.NewRecorder()

	srv.handleGetSession(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- Start Session tests ---

func TestHandleStartSession_Success(t *testing.T) {
	sessRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, id string) (*game.GameSession, error) {
			return sampleSession(testGMID, "lobby"), nil
		},
		updateStatusFn: func(_ context.Context, id, status string) (*game.GameSession, error) {
			s := sampleSession(testGMID, status)
			now := time.Now().UTC()
			s.StartedAt = &now
			return s, nil
		},
	}
	srv := newSessionTestServer(sessRepo, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/"+testSessionID+"/start", nil)
	req.SetPathValue("id", testSessionID)
	req = withAuth(req, testGMID, "gm1")
	w := httptest.NewRecorder()

	srv.handleStartSession(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var resp SessionResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Status != "active" {
		t.Errorf("Status = %q, want %q", resp.Status, "active")
	}
	if resp.StartedAt == nil {
		t.Error("StartedAt should not be nil")
	}
}

func TestHandleStartSession_NotGM(t *testing.T) {
	sessRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, id string) (*game.GameSession, error) {
			return sampleSession(testGMID, "lobby"), nil
		},
	}
	srv := newSessionTestServer(sessRepo, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/"+testSessionID+"/start", nil)
	req.SetPathValue("id", testSessionID)
	req = withAuth(req, "other-user", "other")
	w := httptest.NewRecorder()

	srv.handleStartSession(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestHandleStartSession_NotLobby(t *testing.T) {
	sessRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, id string) (*game.GameSession, error) {
			return sampleSession(testGMID, "active"), nil
		},
	}
	srv := newSessionTestServer(sessRepo, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/"+testSessionID+"/start", nil)
	req.SetPathValue("id", testSessionID)
	req = withAuth(req, testGMID, "gm1")
	w := httptest.NewRecorder()

	srv.handleStartSession(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestHandleStartSession_NotFound(t *testing.T) {
	sessRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, id string) (*game.GameSession, error) {
			return nil, errors.New("game: not found")
		},
	}
	srv := newSessionTestServer(sessRepo, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/"+testSessionID+"/start", nil)
	req.SetPathValue("id", testSessionID)
	req = withAuth(req, testGMID, "gm1")
	w := httptest.NewRecorder()

	srv.handleStartSession(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- Pause Session tests ---

func TestHandlePauseSession_Success(t *testing.T) {
	sessRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, id string) (*game.GameSession, error) {
			return sampleSession(testGMID, "active"), nil
		},
		updateStatusFn: func(_ context.Context, id, status string) (*game.GameSession, error) {
			return sampleSession(testGMID, status), nil
		},
	}
	srv := newSessionTestServer(sessRepo, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/"+testSessionID+"/pause", nil)
	req.SetPathValue("id", testSessionID)
	req = withAuth(req, testGMID, "gm1")
	w := httptest.NewRecorder()

	srv.handlePauseSession(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var resp SessionResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Status != "paused" {
		t.Errorf("Status = %q, want %q", resp.Status, "paused")
	}
}

func TestHandlePauseSession_NotGM(t *testing.T) {
	sessRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, id string) (*game.GameSession, error) {
			return sampleSession(testGMID, "active"), nil
		},
	}
	srv := newSessionTestServer(sessRepo, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/"+testSessionID+"/pause", nil)
	req.SetPathValue("id", testSessionID)
	req = withAuth(req, "other-user", "other")
	w := httptest.NewRecorder()

	srv.handlePauseSession(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestHandlePauseSession_NotActive(t *testing.T) {
	sessRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, id string) (*game.GameSession, error) {
			return sampleSession(testGMID, "lobby"), nil
		},
	}
	srv := newSessionTestServer(sessRepo, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/"+testSessionID+"/pause", nil)
	req.SetPathValue("id", testSessionID)
	req = withAuth(req, testGMID, "gm1")
	w := httptest.NewRecorder()

	srv.handlePauseSession(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

// --- Resume Session tests ---

func TestHandleResumeSession_Success(t *testing.T) {
	sessRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, id string) (*game.GameSession, error) {
			return sampleSession(testGMID, "paused"), nil
		},
		updateStatusFn: func(_ context.Context, id, status string) (*game.GameSession, error) {
			return sampleSession(testGMID, status), nil
		},
	}
	srv := newSessionTestServer(sessRepo, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/"+testSessionID+"/resume", nil)
	req.SetPathValue("id", testSessionID)
	req = withAuth(req, testGMID, "gm1")
	w := httptest.NewRecorder()

	srv.handleResumeSession(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var resp SessionResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Status != "active" {
		t.Errorf("Status = %q, want %q", resp.Status, "active")
	}
}

func TestHandleResumeSession_NotGM(t *testing.T) {
	sessRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, id string) (*game.GameSession, error) {
			return sampleSession(testGMID, "paused"), nil
		},
	}
	srv := newSessionTestServer(sessRepo, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/"+testSessionID+"/resume", nil)
	req.SetPathValue("id", testSessionID)
	req = withAuth(req, "other-user", "other")
	w := httptest.NewRecorder()

	srv.handleResumeSession(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestHandleResumeSession_NotPaused(t *testing.T) {
	sessRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, id string) (*game.GameSession, error) {
			return sampleSession(testGMID, "active"), nil
		},
	}
	srv := newSessionTestServer(sessRepo, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/"+testSessionID+"/resume", nil)
	req.SetPathValue("id", testSessionID)
	req = withAuth(req, testGMID, "gm1")
	w := httptest.NewRecorder()

	srv.handleResumeSession(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

// --- End Session tests ---

func TestHandleEndSession_SuccessFromActive(t *testing.T) {
	sessRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, id string) (*game.GameSession, error) {
			return sampleSession(testGMID, "active"), nil
		},
		updateStatusFn: func(_ context.Context, id, status string) (*game.GameSession, error) {
			s := sampleSession(testGMID, status)
			now := time.Now().UTC()
			s.EndedAt = &now
			return s, nil
		},
	}
	srv := newSessionTestServer(sessRepo, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/"+testSessionID+"/end", nil)
	req.SetPathValue("id", testSessionID)
	req = withAuth(req, testGMID, "gm1")
	w := httptest.NewRecorder()

	srv.handleEndSession(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var resp SessionResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Status != "completed" {
		t.Errorf("Status = %q, want %q", resp.Status, "completed")
	}
}

func TestHandleEndSession_SuccessFromPaused(t *testing.T) {
	sessRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, id string) (*game.GameSession, error) {
			return sampleSession(testGMID, "paused"), nil
		},
		updateStatusFn: func(_ context.Context, id, status string) (*game.GameSession, error) {
			s := sampleSession(testGMID, status)
			now := time.Now().UTC()
			s.EndedAt = &now
			return s, nil
		},
	}
	srv := newSessionTestServer(sessRepo, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/"+testSessionID+"/end", nil)
	req.SetPathValue("id", testSessionID)
	req = withAuth(req, testGMID, "gm1")
	w := httptest.NewRecorder()

	srv.handleEndSession(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleEndSession_NotGM(t *testing.T) {
	sessRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, id string) (*game.GameSession, error) {
			return sampleSession(testGMID, "active"), nil
		},
	}
	srv := newSessionTestServer(sessRepo, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/"+testSessionID+"/end", nil)
	req.SetPathValue("id", testSessionID)
	req = withAuth(req, "other-user", "other")
	w := httptest.NewRecorder()

	srv.handleEndSession(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestHandleEndSession_FromLobby(t *testing.T) {
	sessRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, id string) (*game.GameSession, error) {
			return sampleSession(testGMID, "lobby"), nil
		},
	}
	srv := newSessionTestServer(sessRepo, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/"+testSessionID+"/end", nil)
	req.SetPathValue("id", testSessionID)
	req = withAuth(req, testGMID, "gm1")
	w := httptest.NewRecorder()

	srv.handleEndSession(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestHandleEndSession_FromCompleted(t *testing.T) {
	sessRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, id string) (*game.GameSession, error) {
			return sampleSession(testGMID, "completed"), nil
		},
	}
	srv := newSessionTestServer(sessRepo, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/"+testSessionID+"/end", nil)
	req.SetPathValue("id", testSessionID)
	req = withAuth(req, testGMID, "gm1")
	w := httptest.NewRecorder()

	srv.handleEndSession(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

// --- Join Session tests ---

func TestHandleJoinSession_Success(t *testing.T) {
	sessRepo := &mockSessionRepo{
		getByInviteCodeFn: func(_ context.Context, code string) (*game.GameSession, error) {
			return sampleSession(testGMID, "lobby"), nil
		},
		addPlayerFn: func(_ context.Context, sessionID, userID string) (*game.SessionPlayer, error) {
			return samplePlayer(sessionID, userID), nil
		},
	}
	srv := newSessionTestServer(sessRepo, nil)

	body := `{"inviteCode":"abc123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/join", strings.NewReader(body))
	req = withAuth(req, testPlayerID, "player1")
	w := httptest.NewRecorder()

	srv.handleJoinSession(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
	}
}

func TestHandleJoinSession_InvalidBody(t *testing.T) {
	srv := newSessionTestServer(&mockSessionRepo{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/join", strings.NewReader("{bad"))
	req = withAuth(req, testPlayerID, "player1")
	w := httptest.NewRecorder()

	srv.handleJoinSession(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleJoinSession_EmptyCode(t *testing.T) {
	srv := newSessionTestServer(&mockSessionRepo{}, nil)

	body := `{"inviteCode":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/join", strings.NewReader(body))
	req = withAuth(req, testPlayerID, "player1")
	w := httptest.NewRecorder()

	srv.handleJoinSession(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleJoinSession_InvalidCode(t *testing.T) {
	sessRepo := &mockSessionRepo{
		getByInviteCodeFn: func(_ context.Context, code string) (*game.GameSession, error) {
			return nil, errors.New("game: not found")
		},
	}
	srv := newSessionTestServer(sessRepo, nil)

	body := `{"inviteCode":"XXXXXX"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/join", strings.NewReader(body))
	req = withAuth(req, testPlayerID, "player1")
	w := httptest.NewRecorder()

	srv.handleJoinSession(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleJoinSession_NotLobby(t *testing.T) {
	sessRepo := &mockSessionRepo{
		getByInviteCodeFn: func(_ context.Context, code string) (*game.GameSession, error) {
			return sampleSession(testGMID, "active"), nil
		},
	}
	srv := newSessionTestServer(sessRepo, nil)

	body := `{"inviteCode":"ABC123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/join", strings.NewReader(body))
	req = withAuth(req, testPlayerID, "player1")
	w := httptest.NewRecorder()

	srv.handleJoinSession(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestHandleJoinSession_GMCannotJoin(t *testing.T) {
	sessRepo := &mockSessionRepo{
		getByInviteCodeFn: func(_ context.Context, code string) (*game.GameSession, error) {
			return sampleSession(testGMID, "lobby"), nil
		},
	}
	srv := newSessionTestServer(sessRepo, nil)

	body := `{"inviteCode":"ABC123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/join", strings.NewReader(body))
	req = withAuth(req, testGMID, "gm1")
	w := httptest.NewRecorder()

	srv.handleJoinSession(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestHandleJoinSession_AlreadyJoined(t *testing.T) {
	sessRepo := &mockSessionRepo{
		getByInviteCodeFn: func(_ context.Context, code string) (*game.GameSession, error) {
			return sampleSession(testGMID, "lobby"), nil
		},
		addPlayerFn: func(_ context.Context, sessionID, userID string) (*game.SessionPlayer, error) {
			return nil, errors.New("game: player already joined")
		},
	}
	srv := newSessionTestServer(sessRepo, nil)

	body := `{"inviteCode":"ABC123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/join", strings.NewReader(body))
	req = withAuth(req, testPlayerID, "player1")
	w := httptest.NewRecorder()

	srv.handleJoinSession(w, req)

	// Idempotent: already-joined returns 200 with session data so client can navigate.
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d (idempotent join)", w.Code, http.StatusOK)
	}
}

// --- List Session Players tests ---

func TestHandleListSessionPlayers_SuccessAsGM(t *testing.T) {
	sessRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, id string) (*game.GameSession, error) {
			return sampleSession(testGMID, "lobby"), nil
		},
		listPlayersFn: func(_ context.Context, sessionID string) ([]*game.SessionPlayer, error) {
			return []*game.SessionPlayer{samplePlayer(sessionID, testPlayerID)}, nil
		},
	}
	srv := newSessionTestServer(sessRepo, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/"+testSessionID+"/players", nil)
	req.SetPathValue("id", testSessionID)
	req = withAuth(req, testGMID, "gm1")
	w := httptest.NewRecorder()

	srv.handleListSessionPlayers(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var resp SessionPlayerListResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp.Players) != 1 {
		t.Errorf("Players count = %d, want 1", len(resp.Players))
	}
}

func TestHandleListSessionPlayers_SuccessAsPlayer(t *testing.T) {
	sessRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, id string) (*game.GameSession, error) {
			return sampleSession(testGMID, "lobby"), nil
		},
		getPlayerFn: func(_ context.Context, sessionID, userID string) (*game.SessionPlayer, error) {
			return samplePlayer(sessionID, userID), nil
		},
		listPlayersFn: func(_ context.Context, sessionID string) ([]*game.SessionPlayer, error) {
			return []*game.SessionPlayer{samplePlayer(sessionID, testPlayerID)}, nil
		},
	}
	srv := newSessionTestServer(sessRepo, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/"+testSessionID+"/players", nil)
	req.SetPathValue("id", testSessionID)
	req = withAuth(req, testPlayerID, "player1")
	w := httptest.NewRecorder()

	srv.handleListSessionPlayers(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleListSessionPlayers_Forbidden(t *testing.T) {
	sessRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, id string) (*game.GameSession, error) {
			return sampleSession(testGMID, "lobby"), nil
		},
		getPlayerFn: func(_ context.Context, sessionID, userID string) (*game.SessionPlayer, error) {
			return nil, errors.New("game: player not found")
		},
	}
	srv := newSessionTestServer(sessRepo, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/"+testSessionID+"/players", nil)
	req.SetPathValue("id", testSessionID)
	req = withAuth(req, "stranger", "stranger")
	w := httptest.NewRecorder()

	srv.handleListSessionPlayers(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestHandleListSessionPlayers_NotFound(t *testing.T) {
	sessRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, id string) (*game.GameSession, error) {
			return nil, errors.New("game: not found")
		},
	}
	srv := newSessionTestServer(sessRepo, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/"+testSessionID+"/players", nil)
	req.SetPathValue("id", testSessionID)
	req = withAuth(req, testGMID, "gm1")
	w := httptest.NewRecorder()

	srv.handleListSessionPlayers(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- Remove Session Player tests ---

func TestHandleRemoveSessionPlayer_GMKicks(t *testing.T) {
	sessRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, id string) (*game.GameSession, error) {
			return sampleSession(testGMID, "lobby"), nil
		},
		removePlayerFn: func(_ context.Context, sessionID, userID string) error {
			return nil
		},
	}
	srv := newSessionTestServer(sessRepo, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/sessions/"+testSessionID+"/players/"+testPlayerID, nil)
	req.SetPathValue("id", testSessionID)
	req.SetPathValue("userId", testPlayerID)
	req = withAuth(req, testGMID, "gm1")
	w := httptest.NewRecorder()

	srv.handleRemoveSessionPlayer(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestHandleRemoveSessionPlayer_SelfLeave(t *testing.T) {
	sessRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, id string) (*game.GameSession, error) {
			return sampleSession(testGMID, "lobby"), nil
		},
		removePlayerFn: func(_ context.Context, sessionID, userID string) error {
			return nil
		},
	}
	srv := newSessionTestServer(sessRepo, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/sessions/"+testSessionID+"/players/"+testPlayerID, nil)
	req.SetPathValue("id", testSessionID)
	req.SetPathValue("userId", testPlayerID)
	req = withAuth(req, testPlayerID, "player1")
	w := httptest.NewRecorder()

	srv.handleRemoveSessionPlayer(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestHandleRemoveSessionPlayer_PlayerKicksOther_Forbidden(t *testing.T) {
	sessRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, id string) (*game.GameSession, error) {
			return sampleSession(testGMID, "lobby"), nil
		},
	}
	srv := newSessionTestServer(sessRepo, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/sessions/"+testSessionID+"/players/"+testPlayerID2, nil)
	req.SetPathValue("id", testSessionID)
	req.SetPathValue("userId", testPlayerID2)
	req = withAuth(req, testPlayerID, "player1")
	w := httptest.NewRecorder()

	srv.handleRemoveSessionPlayer(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestHandleRemoveSessionPlayer_SessionNotFound(t *testing.T) {
	sessRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, id string) (*game.GameSession, error) {
			return nil, errors.New("game: not found")
		},
	}
	srv := newSessionTestServer(sessRepo, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/sessions/"+testSessionID+"/players/"+testPlayerID, nil)
	req.SetPathValue("id", testSessionID)
	req.SetPathValue("userId", testPlayerID)
	req = withAuth(req, testGMID, "gm1")
	w := httptest.NewRecorder()

	srv.handleRemoveSessionPlayer(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleRemoveSessionPlayer_PlayerNotFound(t *testing.T) {
	sessRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, id string) (*game.GameSession, error) {
			return sampleSession(testGMID, "lobby"), nil
		},
		removePlayerFn: func(_ context.Context, sessionID, userID string) error {
			return errors.New("game: player not found")
		},
	}
	srv := newSessionTestServer(sessRepo, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/sessions/"+testSessionID+"/players/"+testPlayerID, nil)
	req.SetPathValue("id", testSessionID)
	req.SetPathValue("userId", testPlayerID)
	req = withAuth(req, testGMID, "gm1")
	w := httptest.NewRecorder()

	srv.handleRemoveSessionPlayer(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleRemoveSessionPlayer_InvalidUUID(t *testing.T) {
	srv := newSessionTestServer(&mockSessionRepo{}, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/sessions/bad-id/players/"+testPlayerID, nil)
	req.SetPathValue("id", "bad-id")
	req.SetPathValue("userId", testPlayerID)
	req = withAuth(req, testGMID, "gm1")
	w := httptest.NewRecorder()

	srv.handleRemoveSessionPlayer(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleRemoveSessionPlayer_CompletedSession(t *testing.T) {
	sessRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, id string) (*game.GameSession, error) {
			return sampleSession(testGMID, "completed"), nil
		},
	}
	srv := newSessionTestServer(sessRepo, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/sessions/"+testSessionID+"/players/"+testPlayerID, nil)
	req.SetPathValue("id", testSessionID)
	req.SetPathValue("userId", testPlayerID)
	req = withAuth(req, testGMID, "gm1")
	w := httptest.NewRecorder()

	srv.handleRemoveSessionPlayer(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}
