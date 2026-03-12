package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/like19970403/TRPG-Simulation/internal/apperror"
	"github.com/like19970403/TRPG-Simulation/internal/auth"
	"github.com/like19970403/TRPG-Simulation/internal/scenario"
)

// mockScenarioRepo implements ScenarioRepository for unit tests.
type mockScenarioRepo struct {
	createFn       func(ctx context.Context, authorID, title, description string, content json.RawMessage) (*scenario.Scenario, error)
	listByAuthorFn func(ctx context.Context, authorID string, limit, offset int) ([]*scenario.Scenario, int, error)
	getByIDFn      func(ctx context.Context, id string) (*scenario.Scenario, error)
	updateFn       func(ctx context.Context, id, title, description string, content json.RawMessage) (*scenario.Scenario, error)
	deleteFn       func(ctx context.Context, id string) error
	updateStatusFn func(ctx context.Context, id, newStatus string) (*scenario.Scenario, error)
}

func (m *mockScenarioRepo) Create(ctx context.Context, authorID, title, description string, content json.RawMessage) (*scenario.Scenario, error) {
	return m.createFn(ctx, authorID, title, description, content)
}

func (m *mockScenarioRepo) ListByAuthor(ctx context.Context, authorID string, limit, offset int) ([]*scenario.Scenario, int, error) {
	return m.listByAuthorFn(ctx, authorID, limit, offset)
}

func (m *mockScenarioRepo) GetByID(ctx context.Context, id string) (*scenario.Scenario, error) {
	return m.getByIDFn(ctx, id)
}

func (m *mockScenarioRepo) Update(ctx context.Context, id, title, description string, content json.RawMessage) (*scenario.Scenario, error) {
	return m.updateFn(ctx, id, title, description, content)
}

func (m *mockScenarioRepo) Delete(ctx context.Context, id string) error {
	return m.deleteFn(ctx, id)
}

func (m *mockScenarioRepo) UpdateStatus(ctx context.Context, id, newStatus string) (*scenario.Scenario, error) {
	return m.updateStatusFn(ctx, id, newStatus)
}

func newScenarioTestServer(repo ScenarioRepository) *Server {
	cfg := testConfig()
	srv := New(cfg, nil, testLogger())
	srv.scenarioRepo = repo
	return srv
}

// withAuth injects auth claims into the request context (bypasses requireAuth middleware).
func withAuth(r *http.Request, userID, username string) *http.Request {
	claims := &auth.Claims{UserID: userID, Username: username}
	ctx := context.WithValue(r.Context(), userClaimsKey, claims)
	return r.WithContext(ctx)
}

var sampleContent = json.RawMessage(`{"start_scene":"s1","scenes":[{"id":"s1","name":"Start"}]}`)

func sampleScenario(authorID string) *scenario.Scenario {
	now := time.Now().UTC()
	return &scenario.Scenario{
		ID:          "550e8400-e29b-41d4-a716-446655440000",
		AuthorID:    authorID,
		Title:       "Test Quest",
		Description: "A test scenario",
		Version:     1,
		Status:      "draft",
		Content:     sampleContent,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// --- Create tests ---

func TestHandleCreateScenario_Success(t *testing.T) {
	repo := &mockScenarioRepo{
		createFn: func(_ context.Context, authorID, title, description string, content json.RawMessage) (*scenario.Scenario, error) {
			s := sampleScenario(authorID)
			s.Title = title
			s.Description = description
			s.Content = content
			return s, nil
		},
	}
	srv := newScenarioTestServer(repo)

	body := `{"title":"Test Quest","description":"A test scenario","content":{"start_scene":"s1"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/scenarios", strings.NewReader(body))
	req = withAuth(req, "user-1", "player1")
	w := httptest.NewRecorder()

	srv.handleCreateScenario(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
	}

	var resp ScenarioResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.Title != "Test Quest" {
		t.Errorf("Title = %q, want %q", resp.Title, "Test Quest")
	}
	if resp.Status != "draft" {
		t.Errorf("Status = %q, want %q", resp.Status, "draft")
	}
}

func TestHandleCreateScenario_InvalidJSON(t *testing.T) {
	srv := newScenarioTestServer(&mockScenarioRepo{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/scenarios", strings.NewReader("{bad"))
	req = withAuth(req, "user-1", "player1")
	w := httptest.NewRecorder()

	srv.handleCreateScenario(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleCreateScenario_MissingTitle(t *testing.T) {
	srv := newScenarioTestServer(&mockScenarioRepo{})

	body := `{"title":"","description":"desc","content":{"key":"val"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/scenarios", strings.NewReader(body))
	req = withAuth(req, "user-1", "player1")
	w := httptest.NewRecorder()

	srv.handleCreateScenario(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var errResp ErrorResponse
	json.NewDecoder(w.Body).Decode(&errResp)
	if errResp.Error != "VALIDATION_ERROR" {
		t.Errorf("error = %q, want %q", errResp.Error, "VALIDATION_ERROR")
	}
}

func TestHandleCreateScenario_TitleTooLong(t *testing.T) {
	srv := newScenarioTestServer(&mockScenarioRepo{})

	body := `{"title":"` + strings.Repeat("a", 201) + `","description":"desc","content":{"key":"val"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/scenarios", strings.NewReader(body))
	req = withAuth(req, "user-1", "player1")
	w := httptest.NewRecorder()

	srv.handleCreateScenario(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleCreateScenario_MissingContent(t *testing.T) {
	srv := newScenarioTestServer(&mockScenarioRepo{})

	body := `{"title":"Test","description":"desc"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/scenarios", strings.NewReader(body))
	req = withAuth(req, "user-1", "player1")
	w := httptest.NewRecorder()

	srv.handleCreateScenario(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleCreateScenario_InvalidContent(t *testing.T) {
	srv := newScenarioTestServer(&mockScenarioRepo{})

	body := `{"title":"Test","description":"desc","content":"not-json-object"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/scenarios", strings.NewReader(body))
	req = withAuth(req, "user-1", "player1")
	w := httptest.NewRecorder()

	srv.handleCreateScenario(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleCreateScenario_RepoError(t *testing.T) {
	repo := &mockScenarioRepo{
		createFn: func(_ context.Context, _, _, _ string, _ json.RawMessage) (*scenario.Scenario, error) {
			return nil, errors.New("db error")
		},
	}
	srv := newScenarioTestServer(repo)

	body := `{"title":"Test Quest","description":"desc","content":{"key":"val"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/scenarios", strings.NewReader(body))
	req = withAuth(req, "user-1", "player1")
	w := httptest.NewRecorder()

	srv.handleCreateScenario(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// --- List tests ---

func TestHandleListScenarios_Success(t *testing.T) {
	s1 := sampleScenario("user-1")
	repo := &mockScenarioRepo{
		listByAuthorFn: func(_ context.Context, _ string, limit, offset int) ([]*scenario.Scenario, int, error) {
			if limit != 20 || offset != 0 {
				t.Errorf("limit=%d offset=%d, want 20/0", limit, offset)
			}
			return []*scenario.Scenario{s1}, 1, nil
		},
	}
	srv := newScenarioTestServer(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios", nil)
	req = withAuth(req, "user-1", "player1")
	w := httptest.NewRecorder()

	srv.handleListScenarios(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp ScenarioListResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Total != 1 {
		t.Errorf("total = %d, want 1", resp.Total)
	}
	if len(resp.Scenarios) != 1 {
		t.Errorf("scenarios count = %d, want 1", len(resp.Scenarios))
	}
	if resp.Limit != 20 {
		t.Errorf("limit = %d, want 20", resp.Limit)
	}
	if resp.Offset != 0 {
		t.Errorf("offset = %d, want 0", resp.Offset)
	}
}

func TestHandleListScenarios_Empty(t *testing.T) {
	repo := &mockScenarioRepo{
		listByAuthorFn: func(_ context.Context, _ string, _, _ int) ([]*scenario.Scenario, int, error) {
			return []*scenario.Scenario{}, 0, nil
		},
	}
	srv := newScenarioTestServer(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios", nil)
	req = withAuth(req, "user-1", "player1")
	w := httptest.NewRecorder()

	srv.handleListScenarios(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp ScenarioListResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Total != 0 {
		t.Errorf("total = %d, want 0", resp.Total)
	}
	if len(resp.Scenarios) != 0 {
		t.Errorf("scenarios count = %d, want 0", len(resp.Scenarios))
	}
}

func TestHandleListScenarios_CustomPagination(t *testing.T) {
	repo := &mockScenarioRepo{
		listByAuthorFn: func(_ context.Context, _ string, limit, offset int) ([]*scenario.Scenario, int, error) {
			if limit != 5 || offset != 10 {
				t.Errorf("limit=%d offset=%d, want 5/10", limit, offset)
			}
			return []*scenario.Scenario{}, 0, nil
		},
	}
	srv := newScenarioTestServer(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios?limit=5&offset=10", nil)
	req = withAuth(req, "user-1", "player1")
	w := httptest.NewRecorder()

	srv.handleListScenarios(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp ScenarioListResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Limit != 5 {
		t.Errorf("limit = %d, want 5", resp.Limit)
	}
	if resp.Offset != 10 {
		t.Errorf("offset = %d, want 10", resp.Offset)
	}
}

func TestHandleListScenarios_InvalidPagination(t *testing.T) {
	srv := newScenarioTestServer(&mockScenarioRepo{})

	tests := []struct {
		name  string
		query string
	}{
		{"negative limit", "?limit=-1"},
		{"limit over max", "?limit=200"},
		{"negative offset", "?offset=-1"},
		{"non-numeric limit", "?limit=abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios"+tt.query, nil)
			req = withAuth(req, "user-1", "player1")
			w := httptest.NewRecorder()

			srv.handleListScenarios(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestHandleListScenarios_RepoError(t *testing.T) {
	repo := &mockScenarioRepo{
		listByAuthorFn: func(_ context.Context, _ string, _, _ int) ([]*scenario.Scenario, int, error) {
			return nil, 0, errors.New("db error")
		},
	}
	srv := newScenarioTestServer(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios", nil)
	req = withAuth(req, "user-1", "player1")
	w := httptest.NewRecorder()

	srv.handleListScenarios(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// --- Get tests ---

func TestHandleGetScenario_Success(t *testing.T) {
	s1 := sampleScenario("user-1")
	repo := &mockScenarioRepo{
		getByIDFn: func(_ context.Context, id string) (*scenario.Scenario, error) {
			if id != s1.ID {
				t.Errorf("id = %q, want %q", id, s1.ID)
			}
			return s1, nil
		},
	}
	srv := newScenarioTestServer(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/"+s1.ID, nil)
	req.SetPathValue("id", s1.ID)
	req = withAuth(req, "user-1", "player1")
	w := httptest.NewRecorder()

	srv.handleGetScenario(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp ScenarioResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.ID != s1.ID {
		t.Errorf("ID = %q, want %q", resp.ID, s1.ID)
	}
}

func TestHandleGetScenario_NotFound(t *testing.T) {
	repo := &mockScenarioRepo{
		getByIDFn: func(_ context.Context, _ string) (*scenario.Scenario, error) {
			return nil, fmt.Errorf("scenario: get: %w", apperror.ErrNotFound)
		},
	}
	srv := newScenarioTestServer(repo)

	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/"+id, nil)
	req.SetPathValue("id", id)
	req = withAuth(req, "user-1", "player1")
	w := httptest.NewRecorder()

	srv.handleGetScenario(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleGetScenario_Forbidden(t *testing.T) {
	s1 := sampleScenario("other-user")
	repo := &mockScenarioRepo{
		getByIDFn: func(_ context.Context, _ string) (*scenario.Scenario, error) {
			return s1, nil
		},
	}
	srv := newScenarioTestServer(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/"+s1.ID, nil)
	req.SetPathValue("id", s1.ID)
	req = withAuth(req, "user-1", "player1")
	w := httptest.NewRecorder()

	srv.handleGetScenario(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestHandleGetScenario_InvalidUUID(t *testing.T) {
	srv := newScenarioTestServer(&mockScenarioRepo{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/not-a-uuid", nil)
	req.SetPathValue("id", "not-a-uuid")
	req = withAuth(req, "user-1", "player1")
	w := httptest.NewRecorder()

	srv.handleGetScenario(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- Update tests ---

func TestHandleUpdateScenario_Success(t *testing.T) {
	s1 := sampleScenario("user-1")
	repo := &mockScenarioRepo{
		getByIDFn: func(_ context.Context, _ string) (*scenario.Scenario, error) {
			return s1, nil
		},
		updateFn: func(_ context.Context, id, title, desc string, content json.RawMessage) (*scenario.Scenario, error) {
			updated := *s1
			updated.Title = title
			updated.Description = desc
			updated.Content = content
			return &updated, nil
		},
	}
	srv := newScenarioTestServer(repo)

	body := `{"title":"Updated Title","description":"Updated desc","content":{"key":"val"}}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/scenarios/"+s1.ID, strings.NewReader(body))
	req.SetPathValue("id", s1.ID)
	req = withAuth(req, "user-1", "player1")
	w := httptest.NewRecorder()

	srv.handleUpdateScenario(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp ScenarioResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Title != "Updated Title" {
		t.Errorf("Title = %q, want %q", resp.Title, "Updated Title")
	}
}

func TestHandleUpdateScenario_NotFound(t *testing.T) {
	repo := &mockScenarioRepo{
		getByIDFn: func(_ context.Context, _ string) (*scenario.Scenario, error) {
			return nil, fmt.Errorf("scenario: get: %w", apperror.ErrNotFound)
		},
	}
	srv := newScenarioTestServer(repo)

	id := "550e8400-e29b-41d4-a716-446655440000"
	body := `{"title":"Test","description":"desc","content":{"key":"val"}}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/scenarios/"+id, strings.NewReader(body))
	req.SetPathValue("id", id)
	req = withAuth(req, "user-1", "player1")
	w := httptest.NewRecorder()

	srv.handleUpdateScenario(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleUpdateScenario_Forbidden(t *testing.T) {
	s1 := sampleScenario("other-user")
	repo := &mockScenarioRepo{
		getByIDFn: func(_ context.Context, _ string) (*scenario.Scenario, error) {
			return s1, nil
		},
	}
	srv := newScenarioTestServer(repo)

	body := `{"title":"Test","description":"desc","content":{"key":"val"}}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/scenarios/"+s1.ID, strings.NewReader(body))
	req.SetPathValue("id", s1.ID)
	req = withAuth(req, "user-1", "player1")
	w := httptest.NewRecorder()

	srv.handleUpdateScenario(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestHandleUpdateScenario_NotDraft(t *testing.T) {
	s1 := sampleScenario("user-1")
	s1.Status = "published"
	repo := &mockScenarioRepo{
		getByIDFn: func(_ context.Context, _ string) (*scenario.Scenario, error) {
			return s1, nil
		},
	}
	srv := newScenarioTestServer(repo)

	body := `{"title":"Test","description":"desc","content":{"key":"val"}}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/scenarios/"+s1.ID, strings.NewReader(body))
	req.SetPathValue("id", s1.ID)
	req = withAuth(req, "user-1", "player1")
	w := httptest.NewRecorder()

	srv.handleUpdateScenario(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestHandleUpdateScenario_InvalidBody(t *testing.T) {
	srv := newScenarioTestServer(&mockScenarioRepo{})

	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest(http.MethodPut, "/api/v1/scenarios/"+id, strings.NewReader("{bad"))
	req.SetPathValue("id", id)
	req = withAuth(req, "user-1", "player1")
	w := httptest.NewRecorder()

	srv.handleUpdateScenario(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleUpdateScenario_ValidationError(t *testing.T) {
	s1 := sampleScenario("user-1")
	repo := &mockScenarioRepo{
		getByIDFn: func(_ context.Context, _ string) (*scenario.Scenario, error) {
			return s1, nil
		},
	}
	srv := newScenarioTestServer(repo)

	body := `{"title":"","description":"desc","content":{"key":"val"}}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/scenarios/"+s1.ID, strings.NewReader(body))
	req.SetPathValue("id", s1.ID)
	req = withAuth(req, "user-1", "player1")
	w := httptest.NewRecorder()

	srv.handleUpdateScenario(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- Delete tests ---

func TestHandleDeleteScenario_Success(t *testing.T) {
	s1 := sampleScenario("user-1")
	repo := &mockScenarioRepo{
		getByIDFn: func(_ context.Context, _ string) (*scenario.Scenario, error) {
			return s1, nil
		},
		deleteFn: func(_ context.Context, id string) error {
			if id != s1.ID {
				t.Errorf("id = %q, want %q", id, s1.ID)
			}
			return nil
		},
	}
	srv := newScenarioTestServer(repo)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/scenarios/"+s1.ID, nil)
	req.SetPathValue("id", s1.ID)
	req = withAuth(req, "user-1", "player1")
	w := httptest.NewRecorder()

	srv.handleDeleteScenario(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestHandleDeleteScenario_NotFound(t *testing.T) {
	repo := &mockScenarioRepo{
		getByIDFn: func(_ context.Context, _ string) (*scenario.Scenario, error) {
			return nil, fmt.Errorf("scenario: get: %w", apperror.ErrNotFound)
		},
	}
	srv := newScenarioTestServer(repo)

	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/scenarios/"+id, nil)
	req.SetPathValue("id", id)
	req = withAuth(req, "user-1", "player1")
	w := httptest.NewRecorder()

	srv.handleDeleteScenario(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleDeleteScenario_Forbidden(t *testing.T) {
	s1 := sampleScenario("other-user")
	repo := &mockScenarioRepo{
		getByIDFn: func(_ context.Context, _ string) (*scenario.Scenario, error) {
			return s1, nil
		},
	}
	srv := newScenarioTestServer(repo)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/scenarios/"+s1.ID, nil)
	req.SetPathValue("id", s1.ID)
	req = withAuth(req, "user-1", "player1")
	w := httptest.NewRecorder()

	srv.handleDeleteScenario(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestHandleDeleteScenario_NotDraft(t *testing.T) {
	s1 := sampleScenario("user-1")
	s1.Status = "published"
	repo := &mockScenarioRepo{
		getByIDFn: func(_ context.Context, _ string) (*scenario.Scenario, error) {
			return s1, nil
		},
	}
	srv := newScenarioTestServer(repo)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/scenarios/"+s1.ID, nil)
	req.SetPathValue("id", s1.ID)
	req = withAuth(req, "user-1", "player1")
	w := httptest.NewRecorder()

	srv.handleDeleteScenario(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

// --- Publish tests ---

func TestHandlePublishScenario_Success(t *testing.T) {
	s1 := sampleScenario("user-1")
	repo := &mockScenarioRepo{
		getByIDFn: func(_ context.Context, _ string) (*scenario.Scenario, error) {
			return s1, nil
		},
		updateStatusFn: func(_ context.Context, id, status string) (*scenario.Scenario, error) {
			updated := *s1
			updated.Status = status
			return &updated, nil
		},
	}
	srv := newScenarioTestServer(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/scenarios/"+s1.ID+"/publish", nil)
	req.SetPathValue("id", s1.ID)
	req = withAuth(req, "user-1", "player1")
	w := httptest.NewRecorder()

	srv.handlePublishScenario(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp ScenarioResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Status != "published" {
		t.Errorf("Status = %q, want %q", resp.Status, "published")
	}
}

func TestHandlePublishScenario_NotFound(t *testing.T) {
	repo := &mockScenarioRepo{
		getByIDFn: func(_ context.Context, _ string) (*scenario.Scenario, error) {
			return nil, fmt.Errorf("scenario: get: %w", apperror.ErrNotFound)
		},
	}
	srv := newScenarioTestServer(repo)

	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest(http.MethodPost, "/api/v1/scenarios/"+id+"/publish", nil)
	req.SetPathValue("id", id)
	req = withAuth(req, "user-1", "player1")
	w := httptest.NewRecorder()

	srv.handlePublishScenario(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandlePublishScenario_Forbidden(t *testing.T) {
	s1 := sampleScenario("other-user")
	repo := &mockScenarioRepo{
		getByIDFn: func(_ context.Context, _ string) (*scenario.Scenario, error) {
			return s1, nil
		},
	}
	srv := newScenarioTestServer(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/scenarios/"+s1.ID+"/publish", nil)
	req.SetPathValue("id", s1.ID)
	req = withAuth(req, "user-1", "player1")
	w := httptest.NewRecorder()

	srv.handlePublishScenario(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestHandlePublishScenario_AlreadyPublished(t *testing.T) {
	s1 := sampleScenario("user-1")
	s1.Status = "published"
	repo := &mockScenarioRepo{
		getByIDFn: func(_ context.Context, _ string) (*scenario.Scenario, error) {
			return s1, nil
		},
	}
	srv := newScenarioTestServer(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/scenarios/"+s1.ID+"/publish", nil)
	req.SetPathValue("id", s1.ID)
	req = withAuth(req, "user-1", "player1")
	w := httptest.NewRecorder()

	srv.handlePublishScenario(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestHandlePublishScenario_Archived(t *testing.T) {
	s1 := sampleScenario("user-1")
	s1.Status = "archived"
	repo := &mockScenarioRepo{
		getByIDFn: func(_ context.Context, _ string) (*scenario.Scenario, error) {
			return s1, nil
		},
	}
	srv := newScenarioTestServer(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/scenarios/"+s1.ID+"/publish", nil)
	req.SetPathValue("id", s1.ID)
	req = withAuth(req, "user-1", "player1")
	w := httptest.NewRecorder()

	srv.handlePublishScenario(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

// --- Archive tests ---

func TestHandleArchiveScenario_Success(t *testing.T) {
	s1 := sampleScenario("user-1")
	s1.Status = "published"
	repo := &mockScenarioRepo{
		getByIDFn: func(_ context.Context, _ string) (*scenario.Scenario, error) {
			return s1, nil
		},
		updateStatusFn: func(_ context.Context, id, status string) (*scenario.Scenario, error) {
			updated := *s1
			updated.Status = status
			return &updated, nil
		},
	}
	srv := newScenarioTestServer(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/scenarios/"+s1.ID+"/archive", nil)
	req.SetPathValue("id", s1.ID)
	req = withAuth(req, "user-1", "player1")
	w := httptest.NewRecorder()

	srv.handleArchiveScenario(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp ScenarioResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Status != "archived" {
		t.Errorf("Status = %q, want %q", resp.Status, "archived")
	}
}

func TestHandleArchiveScenario_NotFound(t *testing.T) {
	repo := &mockScenarioRepo{
		getByIDFn: func(_ context.Context, _ string) (*scenario.Scenario, error) {
			return nil, fmt.Errorf("scenario: get: %w", apperror.ErrNotFound)
		},
	}
	srv := newScenarioTestServer(repo)

	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest(http.MethodPost, "/api/v1/scenarios/"+id+"/archive", nil)
	req.SetPathValue("id", id)
	req = withAuth(req, "user-1", "player1")
	w := httptest.NewRecorder()

	srv.handleArchiveScenario(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleArchiveScenario_Forbidden(t *testing.T) {
	s1 := sampleScenario("other-user")
	s1.Status = "published"
	repo := &mockScenarioRepo{
		getByIDFn: func(_ context.Context, _ string) (*scenario.Scenario, error) {
			return s1, nil
		},
	}
	srv := newScenarioTestServer(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/scenarios/"+s1.ID+"/archive", nil)
	req.SetPathValue("id", s1.ID)
	req = withAuth(req, "user-1", "player1")
	w := httptest.NewRecorder()

	srv.handleArchiveScenario(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestHandleArchiveScenario_NotPublished(t *testing.T) {
	s1 := sampleScenario("user-1")
	s1.Status = "draft"
	repo := &mockScenarioRepo{
		getByIDFn: func(_ context.Context, _ string) (*scenario.Scenario, error) {
			return s1, nil
		},
	}
	srv := newScenarioTestServer(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/scenarios/"+s1.ID+"/archive", nil)
	req.SetPathValue("id", s1.ID)
	req = withAuth(req, "user-1", "player1")
	w := httptest.NewRecorder()

	srv.handleArchiveScenario(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

// --- Helper tests ---

func TestValidateCreateScenario(t *testing.T) {
	valid := CreateScenarioRequest{
		Title:       "Test Quest",
		Description: "A test",
		Content:     json.RawMessage(`{"key":"val"}`),
	}
	if errs := validateCreateScenario(valid); len(errs) > 0 {
		t.Errorf("valid request should have no errors, got %v", errs)
	}

	noTitle := CreateScenarioRequest{
		Title:   "",
		Content: json.RawMessage(`{"key":"val"}`),
	}
	if errs := validateCreateScenario(noTitle); len(errs) == 0 {
		t.Error("empty title should be invalid")
	}

	longTitle := CreateScenarioRequest{
		Title:   strings.Repeat("a", 201),
		Content: json.RawMessage(`{"key":"val"}`),
	}
	if errs := validateCreateScenario(longTitle); len(errs) == 0 {
		t.Error("title over 200 chars should be invalid")
	}

	noContent := CreateScenarioRequest{
		Title: "Test",
	}
	if errs := validateCreateScenario(noContent); len(errs) == 0 {
		t.Error("nil content should be invalid")
	}

	invalidContent := CreateScenarioRequest{
		Title:   "Test",
		Content: json.RawMessage(`"just a string"`),
	}
	if errs := validateCreateScenario(invalidContent); len(errs) == 0 {
		t.Error("non-object content should be invalid")
	}
}

func TestValidateUpdateScenario(t *testing.T) {
	valid := UpdateScenarioRequest{
		Title:       "Updated",
		Description: "desc",
		Content:     json.RawMessage(`{"key":"val"}`),
	}
	if errs := validateUpdateScenario(valid); len(errs) > 0 {
		t.Errorf("valid request should have no errors, got %v", errs)
	}

	noTitle := UpdateScenarioRequest{
		Title:   "",
		Content: json.RawMessage(`{"key":"val"}`),
	}
	if errs := validateUpdateScenario(noTitle); len(errs) == 0 {
		t.Error("empty title should be invalid")
	}
}

func TestParsePagination(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		limit   int
		offset  int
		wantErr bool
	}{
		{"defaults", "", 20, 0, false},
		{"custom", "limit=10&offset=5", 10, 5, false},
		{"max limit", "limit=100", 100, 0, false},
		{"over max", "limit=200", 0, 0, true},
		{"negative limit", "limit=-1", 0, 0, true},
		{"negative offset", "offset=-1", 0, 0, true},
		{"non-numeric", "limit=abc", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test?"+tt.query, nil)
			limit, offset, err := parsePagination(req)
			if (err != nil) != tt.wantErr {
				t.Errorf("err = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if limit != tt.limit {
					t.Errorf("limit = %d, want %d", limit, tt.limit)
				}
				if offset != tt.offset {
					t.Errorf("offset = %d, want %d", offset, tt.offset)
				}
			}
		})
	}
}

func TestIsValidUUID(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"550e8400-e29b-41d4-a716-446655440000", true},
		{"not-a-uuid", false},
		{"", false},
		{"550e8400e29b41d4a716446655440000", false}, // no dashes
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := isValidUUID(tt.input); got != tt.valid {
				t.Errorf("isValidUUID(%q) = %v, want %v", tt.input, got, tt.valid)
			}
		})
	}
}
