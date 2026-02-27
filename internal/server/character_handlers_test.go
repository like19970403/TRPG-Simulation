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

	"github.com/like19970403/TRPG-Simulation/internal/character"
	"github.com/like19970403/TRPG-Simulation/internal/game"
)

// mockCharacterRepo implements CharacterRepository for unit tests.
type mockCharacterRepo struct {
	createFn            func(ctx context.Context, userID, name string, attributes, inventory json.RawMessage, notes string) (*character.Character, error)
	getByIDFn           func(ctx context.Context, id string) (*character.Character, error)
	listByUserFn        func(ctx context.Context, userID string, limit, offset int) ([]*character.Character, int, error)
	updateFn            func(ctx context.Context, id, name string, attributes, inventory json.RawMessage, notes string) (*character.Character, error)
	deleteFn            func(ctx context.Context, id string) error
	isLinkedToSessionFn func(ctx context.Context, id string) (bool, error)
}

func (m *mockCharacterRepo) Create(ctx context.Context, userID, name string, attributes, inventory json.RawMessage, notes string) (*character.Character, error) {
	return m.createFn(ctx, userID, name, attributes, inventory, notes)
}

func (m *mockCharacterRepo) GetByID(ctx context.Context, id string) (*character.Character, error) {
	return m.getByIDFn(ctx, id)
}

func (m *mockCharacterRepo) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*character.Character, int, error) {
	return m.listByUserFn(ctx, userID, limit, offset)
}

func (m *mockCharacterRepo) Update(ctx context.Context, id, name string, attributes, inventory json.RawMessage, notes string) (*character.Character, error) {
	return m.updateFn(ctx, id, name, attributes, inventory, notes)
}

func (m *mockCharacterRepo) Delete(ctx context.Context, id string) error {
	return m.deleteFn(ctx, id)
}

func (m *mockCharacterRepo) IsLinkedToSession(ctx context.Context, id string) (bool, error) {
	return m.isLinkedToSessionFn(ctx, id)
}

const testCharacterID = "550e8400-e29b-41d4-a716-446655440000"
const testUserID = "user-1"

func sampleCharacter(userID string) *character.Character {
	now := time.Now().UTC()
	return &character.Character{
		ID:         testCharacterID,
		UserID:     userID,
		Name:       "Aragorn",
		Attributes: json.RawMessage(`{"str":16,"dex":14}`),
		Inventory:  json.RawMessage(`["sword","shield"]`),
		Notes:      "Ranger of the North",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

func newCharacterTestServer(charRepo CharacterRepository, sessRepo SessionRepository) *Server {
	cfg := testConfig()
	srv := New(cfg, nil, testLogger())
	srv.characterRepo = charRepo
	if sessRepo != nil {
		srv.sessionRepo = sessRepo
	}
	return srv
}

// --- Create ---

func TestHandleCreateCharacter_Success(t *testing.T) {
	repo := &mockCharacterRepo{
		createFn: func(_ context.Context, userID, name string, attrs, inv json.RawMessage, notes string) (*character.Character, error) {
			c := sampleCharacter(userID)
			c.Name = name
			return c, nil
		},
	}
	srv := newCharacterTestServer(repo, nil)

	body := `{"name":"Aragorn","attributes":{"str":16},"inventory":["sword"],"notes":"Ranger"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/characters", strings.NewReader(body))
	req = withAuth(req, testUserID, "player1")
	w := httptest.NewRecorder()

	srv.handleCreateCharacter(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
	}
	var resp CharacterResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Name != "Aragorn" {
		t.Errorf("Name = %q, want %q", resp.Name, "Aragorn")
	}
}

func TestHandleCreateCharacter_InvalidJSON(t *testing.T) {
	srv := newCharacterTestServer(&mockCharacterRepo{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/characters", strings.NewReader("{invalid"))
	req = withAuth(req, testUserID, "player1")
	w := httptest.NewRecorder()

	srv.handleCreateCharacter(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleCreateCharacter_MissingName(t *testing.T) {
	srv := newCharacterTestServer(&mockCharacterRepo{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/characters", strings.NewReader(`{"name":""}`))
	req = withAuth(req, testUserID, "player1")
	w := httptest.NewRecorder()

	srv.handleCreateCharacter(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleCreateCharacter_NameTooLong(t *testing.T) {
	srv := newCharacterTestServer(&mockCharacterRepo{}, nil)

	longName := strings.Repeat("a", 101)
	body := `{"name":"` + longName + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/characters", strings.NewReader(body))
	req = withAuth(req, testUserID, "player1")
	w := httptest.NewRecorder()

	srv.handleCreateCharacter(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleCreateCharacter_InvalidAttributes(t *testing.T) {
	srv := newCharacterTestServer(&mockCharacterRepo{}, nil)

	body := `{"name":"Test","attributes":"not-an-object"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/characters", strings.NewReader(body))
	req = withAuth(req, testUserID, "player1")
	w := httptest.NewRecorder()

	srv.handleCreateCharacter(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleCreateCharacter_InvalidInventory(t *testing.T) {
	srv := newCharacterTestServer(&mockCharacterRepo{}, nil)

	body := `{"name":"Test","inventory":"not-an-array"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/characters", strings.NewReader(body))
	req = withAuth(req, testUserID, "player1")
	w := httptest.NewRecorder()

	srv.handleCreateCharacter(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleCreateCharacter_DefaultValues(t *testing.T) {
	repo := &mockCharacterRepo{
		createFn: func(_ context.Context, userID, name string, attrs, inv json.RawMessage, notes string) (*character.Character, error) {
			// Verify defaults were supplied by the handler.
			if string(attrs) != "{}" {
				t.Errorf("attrs = %s, want {}", string(attrs))
			}
			if string(inv) != "[]" {
				t.Errorf("inv = %s, want []", string(inv))
			}
			c := sampleCharacter(userID)
			c.Name = name
			c.Attributes = attrs
			c.Inventory = inv
			return c, nil
		},
	}
	srv := newCharacterTestServer(repo, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/characters", strings.NewReader(`{"name":"Minimal"}`))
	req = withAuth(req, testUserID, "player1")
	w := httptest.NewRecorder()

	srv.handleCreateCharacter(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
	}
}

func TestHandleCreateCharacter_RepoError(t *testing.T) {
	repo := &mockCharacterRepo{
		createFn: func(_ context.Context, _, _ string, _, _ json.RawMessage, _ string) (*character.Character, error) {
			return nil, errors.New("db error")
		},
	}
	srv := newCharacterTestServer(repo, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/characters", strings.NewReader(`{"name":"Test"}`))
	req = withAuth(req, testUserID, "player1")
	w := httptest.NewRecorder()

	srv.handleCreateCharacter(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// --- List ---

func TestHandleListCharacters_Success(t *testing.T) {
	repo := &mockCharacterRepo{
		listByUserFn: func(_ context.Context, userID string, limit, offset int) ([]*character.Character, int, error) {
			return []*character.Character{sampleCharacter(userID)}, 1, nil
		},
	}
	srv := newCharacterTestServer(repo, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/characters", nil)
	req = withAuth(req, testUserID, "player1")
	w := httptest.NewRecorder()

	srv.handleListCharacters(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var resp CharacterListResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Total != 1 {
		t.Errorf("Total = %d, want 1", resp.Total)
	}
	if len(resp.Characters) != 1 {
		t.Errorf("len = %d, want 1", len(resp.Characters))
	}
}

func TestHandleListCharacters_Empty(t *testing.T) {
	repo := &mockCharacterRepo{
		listByUserFn: func(_ context.Context, _ string, _, _ int) ([]*character.Character, int, error) {
			return []*character.Character{}, 0, nil
		},
	}
	srv := newCharacterTestServer(repo, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/characters", nil)
	req = withAuth(req, testUserID, "player1")
	w := httptest.NewRecorder()

	srv.handleListCharacters(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleListCharacters_InvalidPagination(t *testing.T) {
	srv := newCharacterTestServer(&mockCharacterRepo{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/characters?limit=-1", nil)
	req = withAuth(req, testUserID, "player1")
	w := httptest.NewRecorder()

	srv.handleListCharacters(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleListCharacters_RepoError(t *testing.T) {
	repo := &mockCharacterRepo{
		listByUserFn: func(_ context.Context, _ string, _, _ int) ([]*character.Character, int, error) {
			return nil, 0, errors.New("db error")
		},
	}
	srv := newCharacterTestServer(repo, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/characters", nil)
	req = withAuth(req, testUserID, "player1")
	w := httptest.NewRecorder()

	srv.handleListCharacters(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// --- Get ---

func TestHandleGetCharacter_Success(t *testing.T) {
	repo := &mockCharacterRepo{
		getByIDFn: func(_ context.Context, id string) (*character.Character, error) {
			return sampleCharacter(testUserID), nil
		},
	}
	srv := newCharacterTestServer(repo, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/characters/"+testCharacterID, nil)
	req.SetPathValue("id", testCharacterID)
	req = withAuth(req, testUserID, "player1")
	w := httptest.NewRecorder()

	srv.handleGetCharacter(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleGetCharacter_NotFound(t *testing.T) {
	repo := &mockCharacterRepo{
		getByIDFn: func(_ context.Context, _ string) (*character.Character, error) {
			return nil, errors.New("character: not found")
		},
	}
	srv := newCharacterTestServer(repo, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/characters/"+testCharacterID, nil)
	req.SetPathValue("id", testCharacterID)
	req = withAuth(req, testUserID, "player1")
	w := httptest.NewRecorder()

	srv.handleGetCharacter(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleGetCharacter_Forbidden(t *testing.T) {
	repo := &mockCharacterRepo{
		getByIDFn: func(_ context.Context, _ string) (*character.Character, error) {
			return sampleCharacter("other-user"), nil
		},
	}
	srv := newCharacterTestServer(repo, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/characters/"+testCharacterID, nil)
	req.SetPathValue("id", testCharacterID)
	req = withAuth(req, testUserID, "player1")
	w := httptest.NewRecorder()

	srv.handleGetCharacter(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestHandleGetCharacter_InvalidUUID(t *testing.T) {
	srv := newCharacterTestServer(&mockCharacterRepo{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/characters/bad-id", nil)
	req.SetPathValue("id", "bad-id")
	req = withAuth(req, testUserID, "player1")
	w := httptest.NewRecorder()

	srv.handleGetCharacter(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- Update ---

func TestHandleUpdateCharacter_Success(t *testing.T) {
	repo := &mockCharacterRepo{
		getByIDFn: func(_ context.Context, _ string) (*character.Character, error) {
			return sampleCharacter(testUserID), nil
		},
		updateFn: func(_ context.Context, id, name string, attrs, inv json.RawMessage, notes string) (*character.Character, error) {
			c := sampleCharacter(testUserID)
			c.Name = name
			return c, nil
		},
	}
	srv := newCharacterTestServer(repo, nil)

	body := `{"name":"NewName","attributes":{"str":18}}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/characters/"+testCharacterID, strings.NewReader(body))
	req.SetPathValue("id", testCharacterID)
	req = withAuth(req, testUserID, "player1")
	w := httptest.NewRecorder()

	srv.handleUpdateCharacter(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var resp CharacterResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Name != "NewName" {
		t.Errorf("Name = %q, want %q", resp.Name, "NewName")
	}
}

func TestHandleUpdateCharacter_NotFound(t *testing.T) {
	repo := &mockCharacterRepo{
		getByIDFn: func(_ context.Context, _ string) (*character.Character, error) {
			return nil, errors.New("character: not found")
		},
	}
	srv := newCharacterTestServer(repo, nil)

	body := `{"name":"NewName"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/characters/"+testCharacterID, strings.NewReader(body))
	req.SetPathValue("id", testCharacterID)
	req = withAuth(req, testUserID, "player1")
	w := httptest.NewRecorder()

	srv.handleUpdateCharacter(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleUpdateCharacter_Forbidden(t *testing.T) {
	repo := &mockCharacterRepo{
		getByIDFn: func(_ context.Context, _ string) (*character.Character, error) {
			return sampleCharacter("other-user"), nil
		},
	}
	srv := newCharacterTestServer(repo, nil)

	body := `{"name":"NewName"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/characters/"+testCharacterID, strings.NewReader(body))
	req.SetPathValue("id", testCharacterID)
	req = withAuth(req, testUserID, "player1")
	w := httptest.NewRecorder()

	srv.handleUpdateCharacter(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestHandleUpdateCharacter_InvalidBody(t *testing.T) {
	repo := &mockCharacterRepo{
		getByIDFn: func(_ context.Context, _ string) (*character.Character, error) {
			return sampleCharacter(testUserID), nil
		},
	}
	srv := newCharacterTestServer(repo, nil)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/characters/"+testCharacterID, strings.NewReader("{invalid"))
	req.SetPathValue("id", testCharacterID)
	req = withAuth(req, testUserID, "player1")
	w := httptest.NewRecorder()

	srv.handleUpdateCharacter(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleUpdateCharacter_ValidationError(t *testing.T) {
	repo := &mockCharacterRepo{
		getByIDFn: func(_ context.Context, _ string) (*character.Character, error) {
			return sampleCharacter(testUserID), nil
		},
	}
	srv := newCharacterTestServer(repo, nil)

	body := `{"name":""}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/characters/"+testCharacterID, strings.NewReader(body))
	req.SetPathValue("id", testCharacterID)
	req = withAuth(req, testUserID, "player1")
	w := httptest.NewRecorder()

	srv.handleUpdateCharacter(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- Delete ---

func TestHandleDeleteCharacter_Success(t *testing.T) {
	repo := &mockCharacterRepo{
		getByIDFn: func(_ context.Context, _ string) (*character.Character, error) {
			return sampleCharacter(testUserID), nil
		},
		isLinkedToSessionFn: func(_ context.Context, _ string) (bool, error) {
			return false, nil
		},
		deleteFn: func(_ context.Context, _ string) error {
			return nil
		},
	}
	srv := newCharacterTestServer(repo, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/characters/"+testCharacterID, nil)
	req.SetPathValue("id", testCharacterID)
	req = withAuth(req, testUserID, "player1")
	w := httptest.NewRecorder()

	srv.handleDeleteCharacter(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestHandleDeleteCharacter_NotFound(t *testing.T) {
	repo := &mockCharacterRepo{
		getByIDFn: func(_ context.Context, _ string) (*character.Character, error) {
			return nil, errors.New("character: not found")
		},
	}
	srv := newCharacterTestServer(repo, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/characters/"+testCharacterID, nil)
	req.SetPathValue("id", testCharacterID)
	req = withAuth(req, testUserID, "player1")
	w := httptest.NewRecorder()

	srv.handleDeleteCharacter(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleDeleteCharacter_Forbidden(t *testing.T) {
	repo := &mockCharacterRepo{
		getByIDFn: func(_ context.Context, _ string) (*character.Character, error) {
			return sampleCharacter("other-user"), nil
		},
	}
	srv := newCharacterTestServer(repo, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/characters/"+testCharacterID, nil)
	req.SetPathValue("id", testCharacterID)
	req = withAuth(req, testUserID, "player1")
	w := httptest.NewRecorder()

	srv.handleDeleteCharacter(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestHandleDeleteCharacter_LinkedToSession(t *testing.T) {
	repo := &mockCharacterRepo{
		getByIDFn: func(_ context.Context, _ string) (*character.Character, error) {
			return sampleCharacter(testUserID), nil
		},
		isLinkedToSessionFn: func(_ context.Context, _ string) (bool, error) {
			return true, nil
		},
	}
	srv := newCharacterTestServer(repo, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/characters/"+testCharacterID, nil)
	req.SetPathValue("id", testCharacterID)
	req = withAuth(req, testUserID, "player1")
	w := httptest.NewRecorder()

	srv.handleDeleteCharacter(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestHandleDeleteCharacter_InvalidUUID(t *testing.T) {
	srv := newCharacterTestServer(&mockCharacterRepo{}, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/characters/bad-id", nil)
	req.SetPathValue("id", "bad-id")
	req = withAuth(req, testUserID, "player1")
	w := httptest.NewRecorder()

	srv.handleDeleteCharacter(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- Assign ---

func sampleLobbySession() *game.GameSession {
	return &game.GameSession{
		ID:         testSessionID,
		ScenarioID: testScenarioID,
		GMID:       "gm-user-id",
		Status:     "lobby",
		InviteCode: "ABCD1234",
		CreatedAt:  time.Now().UTC(),
	}
}

func sampleSessionPlayer(userID string) *game.SessionPlayer {
	charID := testCharacterID
	return &game.SessionPlayer{
		ID:          "sp-1",
		SessionID:   testSessionID,
		UserID:      userID,
		CharacterID: &charID,
		Status:      "joined",
		JoinedAt:    time.Now().UTC(),
	}
}

func TestHandleAssignCharacter_Success(t *testing.T) {
	charRepo := &mockCharacterRepo{
		getByIDFn: func(_ context.Context, _ string) (*character.Character, error) {
			return sampleCharacter(testUserID), nil
		},
	}
	sessRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, _ string) (*game.GameSession, error) {
			return sampleLobbySession(), nil
		},
		getPlayerFn: func(_ context.Context, _, _ string) (*game.SessionPlayer, error) {
			return &game.SessionPlayer{UserID: testUserID, Status: "joined"}, nil
		},
		setCharacterIDFn: func(_ context.Context, _, _, _ string) (*game.SessionPlayer, error) {
			return sampleSessionPlayer(testUserID), nil
		},
	}
	srv := newCharacterTestServer(charRepo, sessRepo)

	body := `{"characterId":"` + testCharacterID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/"+testSessionID+"/characters", strings.NewReader(body))
	req.SetPathValue("id", testSessionID)
	req = withAuth(req, testUserID, "player1")
	w := httptest.NewRecorder()

	srv.handleAssignCharacter(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var resp SessionPlayerResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.CharacterID == nil || *resp.CharacterID != testCharacterID {
		t.Errorf("CharacterID = %v, want %q", resp.CharacterID, testCharacterID)
	}
}

func TestHandleAssignCharacter_SessionNotFound(t *testing.T) {
	sessRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, _ string) (*game.GameSession, error) {
			return nil, errors.New("game: not found")
		},
	}
	srv := newCharacterTestServer(&mockCharacterRepo{}, sessRepo)

	body := `{"characterId":"` + testCharacterID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/"+testSessionID+"/characters", strings.NewReader(body))
	req.SetPathValue("id", testSessionID)
	req = withAuth(req, testUserID, "player1")
	w := httptest.NewRecorder()

	srv.handleAssignCharacter(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleAssignCharacter_NotLobby(t *testing.T) {
	sessRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, _ string) (*game.GameSession, error) {
			gs := sampleLobbySession()
			gs.Status = "active"
			return gs, nil
		},
	}
	srv := newCharacterTestServer(&mockCharacterRepo{}, sessRepo)

	body := `{"characterId":"` + testCharacterID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/"+testSessionID+"/characters", strings.NewReader(body))
	req.SetPathValue("id", testSessionID)
	req = withAuth(req, testUserID, "player1")
	w := httptest.NewRecorder()

	srv.handleAssignCharacter(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestHandleAssignCharacter_NotPlayer(t *testing.T) {
	sessRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, _ string) (*game.GameSession, error) {
			return sampleLobbySession(), nil
		},
		getPlayerFn: func(_ context.Context, _, _ string) (*game.SessionPlayer, error) {
			return nil, errors.New("game: player not found")
		},
	}
	srv := newCharacterTestServer(&mockCharacterRepo{}, sessRepo)

	body := `{"characterId":"` + testCharacterID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/"+testSessionID+"/characters", strings.NewReader(body))
	req.SetPathValue("id", testSessionID)
	req = withAuth(req, testUserID, "player1")
	w := httptest.NewRecorder()

	srv.handleAssignCharacter(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestHandleAssignCharacter_CharacterNotFound(t *testing.T) {
	charRepo := &mockCharacterRepo{
		getByIDFn: func(_ context.Context, _ string) (*character.Character, error) {
			return nil, errors.New("character: not found")
		},
	}
	sessRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, _ string) (*game.GameSession, error) {
			return sampleLobbySession(), nil
		},
		getPlayerFn: func(_ context.Context, _, _ string) (*game.SessionPlayer, error) {
			return &game.SessionPlayer{UserID: testUserID}, nil
		},
	}
	srv := newCharacterTestServer(charRepo, sessRepo)

	body := `{"characterId":"` + testCharacterID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/"+testSessionID+"/characters", strings.NewReader(body))
	req.SetPathValue("id", testSessionID)
	req = withAuth(req, testUserID, "player1")
	w := httptest.NewRecorder()

	srv.handleAssignCharacter(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleAssignCharacter_CharacterNotOwned(t *testing.T) {
	charRepo := &mockCharacterRepo{
		getByIDFn: func(_ context.Context, _ string) (*character.Character, error) {
			return sampleCharacter("other-user"), nil
		},
	}
	sessRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, _ string) (*game.GameSession, error) {
			return sampleLobbySession(), nil
		},
		getPlayerFn: func(_ context.Context, _, _ string) (*game.SessionPlayer, error) {
			return &game.SessionPlayer{UserID: testUserID}, nil
		},
	}
	srv := newCharacterTestServer(charRepo, sessRepo)

	body := `{"characterId":"` + testCharacterID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/"+testSessionID+"/characters", strings.NewReader(body))
	req.SetPathValue("id", testSessionID)
	req = withAuth(req, testUserID, "player1")
	w := httptest.NewRecorder()

	srv.handleAssignCharacter(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestHandleAssignCharacter_InvalidBody(t *testing.T) {
	sessRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, _ string) (*game.GameSession, error) {
			return sampleLobbySession(), nil
		},
	}
	srv := newCharacterTestServer(&mockCharacterRepo{}, sessRepo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/"+testSessionID+"/characters", strings.NewReader("{invalid"))
	req.SetPathValue("id", testSessionID)
	req = withAuth(req, testUserID, "player1")
	w := httptest.NewRecorder()

	srv.handleAssignCharacter(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
