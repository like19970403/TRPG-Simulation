package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/like19970403/TRPG-Simulation/internal/game"
)

type mockEventRepo struct {
	appendEventFn     func(ctx context.Context, sessionID string, sequence int64, eventType string, actorID *string, payload json.RawMessage) (*game.GameEvent, error)
	listEventsSinceFn func(ctx context.Context, sessionID string, afterSeq int64) ([]*game.GameEvent, error)
}

func (m *mockEventRepo) AppendEvent(ctx context.Context, sessionID string, sequence int64, eventType string, actorID *string, payload json.RawMessage) (*game.GameEvent, error) {
	return m.appendEventFn(ctx, sessionID, sequence, eventType, actorID, payload)
}

func (m *mockEventRepo) ListEventsSince(ctx context.Context, sessionID string, afterSeq int64) ([]*game.GameEvent, error) {
	return m.listEventsSinceFn(ctx, sessionID, afterSeq)
}

func successEventRepo() *mockEventRepo {
	return &mockEventRepo{
		appendEventFn: func(_ context.Context, sessionID string, sequence int64, eventType string, actorID *string, payload json.RawMessage) (*game.GameEvent, error) {
			return &game.GameEvent{
				ID:        "evt-1",
				SessionID: sessionID,
				Sequence:  sequence,
				Type:      eventType,
				ActorID:   actorID,
				Payload:   payload,
				CreatedAt: time.Now(),
			}, nil
		},
		listEventsSinceFn: func(_ context.Context, sessionID string, afterSeq int64) ([]*game.GameEvent, error) {
			return []*game.GameEvent{}, nil
		},
	}
}

// fakeClient creates a Client with only a send channel (no real WS connection).
func fakeClient(userID string, role SenderRole, room *Room) *Client {
	return &Client{
		send:   make(chan []byte, sendBufferSize),
		userID: userID,
		role:   role,
		room:   room,
		logger: testLogger(),
		done:   make(chan struct{}),
	}
}

func TestRoom_RegisterUnregister(t *testing.T) {
	repo := successEventRepo()
	room := NewRoom("sess-1", "gm-1", nil, repo, testLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go room.Run(ctx)
	defer room.Stop()

	c1 := fakeClient("user-1", RolePlayer, room)
	c2 := fakeClient("user-2", RolePlayer, room)

	room.register <- c1
	room.register <- c2
	time.Sleep(50 * time.Millisecond)

	if room.ClientCount() != 2 {
		t.Errorf("ClientCount = %d, want 2", room.ClientCount())
	}

	room.unregister <- c1
	time.Sleep(50 * time.Millisecond)

	if room.ClientCount() != 1 {
		t.Errorf("ClientCount = %d, want 1", room.ClientCount())
	}
}

func TestRoom_BroadcastEvent_PersistsAndBroadcasts(t *testing.T) {
	var appendCalled atomic.Bool
	repo := &mockEventRepo{
		appendEventFn: func(_ context.Context, sessionID string, sequence int64, eventType string, actorID *string, payload json.RawMessage) (*game.GameEvent, error) {
			appendCalled.Store(true)
			return &game.GameEvent{ID: "evt-1", SessionID: sessionID, Sequence: sequence, Type: eventType}, nil
		},
		listEventsSinceFn: func(_ context.Context, _ string, _ int64) ([]*game.GameEvent, error) {
			return []*game.GameEvent{}, nil
		},
	}
	room := NewRoom("sess-1", "gm-1", nil, repo, testLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go room.Run(ctx)
	defer room.Stop()

	c := fakeClient("user-1", RolePlayer, room)
	room.register <- c
	time.Sleep(50 * time.Millisecond)

	actorID := "gm-1"
	room.BroadcastEvent(EventGameStarted, &actorID, json.RawMessage(`{}`))
	time.Sleep(50 * time.Millisecond)

	if !appendCalled.Load() {
		t.Error("AppendEvent was not called")
	}

	select {
	case msg := <-c.send:
		var env Envelope
		if err := json.Unmarshal(msg, &env); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if env.Type != EventGameStarted {
			t.Errorf("Type = %q, want %q", env.Type, EventGameStarted)
		}
	default:
		t.Error("no message received by client")
	}
}

func TestRoom_BroadcastEvent_UpdatesGameState(t *testing.T) {
	repo := successEventRepo()
	room := NewRoom("sess-1", "gm-1", nil, repo, testLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go room.Run(ctx)
	defer room.Stop()

	actorID := "gm-1"
	room.BroadcastEvent(EventGameStarted, &actorID, json.RawMessage(`{}`))
	time.Sleep(50 * time.Millisecond)

	if room.StateSnapshot().Status != "active" {
		t.Errorf("Status = %q, want %q", room.StateSnapshot().Status, "active")
	}
	if room.StateSnapshot().LastSequence != 1 {
		t.Errorf("LastSequence = %d, want 1", room.StateSnapshot().LastSequence)
	}
}

func TestRoom_Stop_DisconnectsAllClients(t *testing.T) {
	repo := successEventRepo()
	room := NewRoom("sess-1", "gm-1", nil, repo, testLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go room.Run(ctx)

	c1 := fakeClient("user-1", RolePlayer, room)
	c2 := fakeClient("user-2", RolePlayer, room)
	room.register <- c1
	room.register <- c2
	time.Sleep(50 * time.Millisecond)

	room.Stop()

	// send channels should be closed.
	_, ok1 := <-c1.send
	_, ok2 := <-c2.send
	if ok1 {
		t.Error("c1.send should be closed")
	}
	if ok2 {
		t.Error("c2.send should be closed")
	}
}

func TestRoom_ReplayEvents(t *testing.T) {
	repo := &mockEventRepo{
		appendEventFn: func(_ context.Context, _ string, _ int64, _ string, _ *string, _ json.RawMessage) (*game.GameEvent, error) {
			return &game.GameEvent{}, nil
		},
		listEventsSinceFn: func(_ context.Context, _ string, afterSeq int64) ([]*game.GameEvent, error) {
			events := []*game.GameEvent{
				{ID: "e1", SessionID: "sess-1", Sequence: 2, Type: EventGameStarted, Payload: json.RawMessage(`{}`), CreatedAt: time.Now()},
				{ID: "e2", SessionID: "sess-1", Sequence: 3, Type: EventGamePaused, Payload: json.RawMessage(`{}`), CreatedAt: time.Now()},
				{ID: "e3", SessionID: "sess-1", Sequence: 4, Type: EventGameResumed, Payload: json.RawMessage(`{}`), CreatedAt: time.Now()},
			}
			return events, nil
		},
	}
	room := NewRoom("sess-1", "gm-1", nil, repo, testLogger())

	c := fakeClient("user-1", RolePlayer, room)
	err := room.ReplayEvents(context.Background(), c, 1)
	if err != nil {
		t.Fatalf("ReplayEvents: %v", err)
	}

	// Should receive 3 messages.
	for i := 0; i < 3; i++ {
		select {
		case msg := <-c.send:
			var env Envelope
			json.Unmarshal(msg, &env)
			if env.SessionID != "sess-1" {
				t.Errorf("event %d: SessionID = %q, want %q", i, env.SessionID, "sess-1")
			}
		default:
			t.Fatalf("expected message %d, got none", i)
		}
	}
}

func TestRoom_HandleIncoming_InvalidJSON(t *testing.T) {
	repo := successEventRepo()
	room := NewRoom("sess-1", "gm-1", nil, repo, testLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go room.Run(ctx)
	defer room.Stop()

	c := fakeClient("user-1", RolePlayer, room)
	room.register <- c
	time.Sleep(50 * time.Millisecond)

	room.incoming <- incomingMessage{client: c, data: []byte(`{not valid json}`)}
	time.Sleep(50 * time.Millisecond)

	select {
	case msg := <-c.send:
		var env Envelope
		json.Unmarshal(msg, &env)
		if env.Type != EventError {
			t.Errorf("Type = %q, want %q", env.Type, EventError)
		}
	default:
		t.Error("expected error message, got none")
	}
}

func TestRoom_HandleIncoming_UnknownAction(t *testing.T) {
	repo := successEventRepo()
	room := NewRoom("sess-1", "gm-1", nil, repo, testLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go room.Run(ctx)
	defer room.Stop()

	c := fakeClient("user-1", RolePlayer, room)
	room.register <- c
	time.Sleep(50 * time.Millisecond)

	room.incoming <- incomingMessage{client: c, data: []byte(`{"type":"unknown_action","payload":{}}`)}
	time.Sleep(50 * time.Millisecond)

	select {
	case msg := <-c.send:
		var env Envelope
		json.Unmarshal(msg, &env)
		if env.Type != EventError {
			t.Errorf("Type = %q, want %q", env.Type, EventError)
		}
		var payload map[string]string
		json.Unmarshal(env.Payload, &payload)
		if payload["message"] == "" {
			t.Error("expected error message in payload")
		}
	default:
		t.Error("expected error message, got none")
	}
}

func TestRoom_HandleIncoming_AdvanceSceneDispatches(t *testing.T) {
	repo := successEventRepo()
	room := NewRoom("sess-1", "gm-1", nil, repo, testLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go room.Run(ctx)
	defer room.Stop()

	c := fakeClient("gm-1", RoleGM, room)
	room.register <- c
	time.Sleep(50 * time.Millisecond)

	room.incoming <- incomingMessage{client: c, data: []byte(`{"type":"advance_scene","payload":{"scene_id":"library"}}`)}
	time.Sleep(50 * time.Millisecond)

	select {
	case <-c.send:
		// Dispatch worked — we got a response.
	default:
		t.Error("expected response from advance_scene dispatch, got none")
	}
}

func TestRoom_HandleIncoming_DiceRollDispatches(t *testing.T) {
	repo := successEventRepo()
	room := NewRoom("sess-1", "gm-1", nil, repo, testLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go room.Run(ctx)
	defer room.Stop()

	c := fakeClient("user-1", RolePlayer, room)
	room.register <- c
	time.Sleep(50 * time.Millisecond)

	room.incoming <- incomingMessage{client: c, data: []byte(`{"type":"dice_roll","payload":{"formula":"2d6"}}`)}
	time.Sleep(50 * time.Millisecond)

	select {
	case <-c.send:
		// Dispatch worked.
	default:
		t.Error("expected response from dice_roll dispatch, got none")
	}
}

func TestRoom_SendError_Format(t *testing.T) {
	repo := successEventRepo()
	room := NewRoom("sess-1", "gm-1", nil, repo, testLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go room.Run(ctx)
	defer room.Stop()

	c := fakeClient("user-1", RolePlayer, room)
	room.register <- c
	time.Sleep(50 * time.Millisecond)

	room.incoming <- incomingMessage{client: c, data: []byte(`!!!`)}
	time.Sleep(50 * time.Millisecond)

	select {
	case msg := <-c.send:
		var env Envelope
		if err := json.Unmarshal(msg, &env); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if env.Type != EventError {
			t.Errorf("Type = %q, want %q", env.Type, EventError)
		}
		if env.SessionID != "sess-1" {
			t.Errorf("SessionID = %q, want %q", env.SessionID, "sess-1")
		}
		var payload map[string]string
		json.Unmarshal(env.Payload, &payload)
		if payload["message"] == "" {
			t.Error("expected error message in payload")
		}
	default:
		t.Error("expected error message, got none")
	}
}

func TestFilterScenePayload_GM(t *testing.T) {
	payload := json.RawMessage(`{"scene_id":"lib","scene":{"id":"lib","gm_notes":"secret"}}`)
	result := filterScenePayload(payload, RoleGM)
	var m map[string]json.RawMessage
	json.Unmarshal(result, &m)
	if _, ok := m["scene"]; !ok {
		t.Fatal("scene should be present")
	}
	var scene map[string]json.RawMessage
	json.Unmarshal(m["scene"], &scene)
	if _, ok := scene["gm_notes"]; !ok {
		t.Error("GM should see gm_notes")
	}
}

func TestFilterScenePayload_Player(t *testing.T) {
	payload := json.RawMessage(`{"scene_id":"lib","scene":{"id":"lib","name":"Library","gm_notes":"secret"}}`)
	result := filterScenePayload(payload, RolePlayer)
	var m map[string]json.RawMessage
	json.Unmarshal(result, &m)
	var scene map[string]json.RawMessage
	json.Unmarshal(m["scene"], &scene)
	if _, ok := scene["gm_notes"]; ok {
		t.Error("Player should NOT see gm_notes")
	}
	if _, ok := scene["name"]; !ok {
		t.Error("Player should see name")
	}
}

func TestFilterScenePayload_NoGMNotes(t *testing.T) {
	payload := json.RawMessage(`{"scene_id":"lib","scene":{"id":"lib","name":"Library"}}`)
	result := filterScenePayload(payload, RolePlayer)
	var m map[string]json.RawMessage
	json.Unmarshal(result, &m)
	var scene map[string]json.RawMessage
	json.Unmarshal(m["scene"], &scene)
	if _, ok := scene["gm_notes"]; ok {
		t.Error("should not have gm_notes when none present")
	}
	if _, ok := scene["name"]; !ok {
		t.Error("should preserve name")
	}
}

func TestRoom_BroadcastFiltered_GMAndPlayer(t *testing.T) {
	repo := successEventRepo()
	room := NewRoom("sess-1", "gm-1", nil, repo, testLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go room.Run(ctx)
	defer room.Stop()

	gm := fakeClient("gm-1", RoleGM, room)
	player := fakeClient("user-1", RolePlayer, room)
	room.register <- gm
	room.register <- player
	time.Sleep(50 * time.Millisecond)

	actorID := "gm-1"
	room.BroadcastEvent(EventGameStarted, &actorID, json.RawMessage(`{}`))
	time.Sleep(50 * time.Millisecond)

	for _, c := range []*Client{gm, player} {
		select {
		case <-c.send:
			// OK
		default:
			t.Errorf("client %s did not receive broadcast", c.userID)
		}
	}
}

func TestRoom_BroadcastEvent_PersistError_RollsBack(t *testing.T) {
	repo := &mockEventRepo{
		appendEventFn: func(_ context.Context, _ string, _ int64, _ string, _ *string, _ json.RawMessage) (*game.GameEvent, error) {
			return nil, errors.New("db error")
		},
		listEventsSinceFn: func(_ context.Context, _ string, _ int64) ([]*game.GameEvent, error) {
			return []*game.GameEvent{}, nil
		},
	}
	room := NewRoom("sess-1", "gm-1", nil, repo, testLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go room.Run(ctx)
	defer room.Stop()

	actorID := "gm-1"
	room.BroadcastEvent(EventGameStarted, &actorID, json.RawMessage(`{}`))
	time.Sleep(50 * time.Millisecond)

	// Sequence should be rolled back to 0.
	if room.StateSnapshot().LastSequence != 0 {
		t.Errorf("LastSequence = %d, want 0 (should rollback on persist error)", room.StateSnapshot().LastSequence)
	}
}

// --- Test helpers for Step 6+ ---

func testScenario() *ScenarioContent {
	return &ScenarioContent{
		ID:         "test-scenario",
		Title:      "Test",
		StartScene: "entrance",
		Scenes: []Scene{
			{ID: "entrance", Name: "Entrance", Content: "You enter.", GMNotes: "GM secret notes"},
			{ID: "library", Name: "Library", Content: "Books everywhere.", GMNotes: "Check bookshelf"},
		},
	}
}

// startedRoom creates a Room with scenario, starts it, and advances state to "active".
func startedRoom(repo *mockEventRepo, sc *ScenarioContent) (*Room, context.CancelFunc) {
	room := NewRoom("sess-1", "gm-1", sc, repo, testLogger())
	ctx, cancel := context.WithCancel(context.Background())
	go room.Run(ctx)

	// Fire game_started to move status to "active".
	actorID := "gm-1"
	room.BroadcastEvent(EventGameStarted, &actorID, json.RawMessage(`{}`))
	time.Sleep(50 * time.Millisecond)

	return room, cancel
}

// --- Step 6: advance_scene handler tests ---

func TestAdvanceScene_GMSuccess(t *testing.T) {
	repo := successEventRepo()
	room, cancel := startedRoom(repo, testScenario())
	defer cancel()
	defer room.Stop()

	gm := fakeClient("gm-1", RoleGM, room)
	player := fakeClient("user-1", RolePlayer, room)
	room.register <- gm
	room.register <- player
	time.Sleep(50 * time.Millisecond)

	room.incoming <- incomingMessage{client: gm, data: []byte(`{"type":"advance_scene","payload":{"scene_id":"library"}}`)}
	time.Sleep(50 * time.Millisecond)

	// Both clients should receive scene_changed.
	for _, c := range []*Client{gm, player} {
		select {
		case msg := <-c.send:
			var env Envelope
			json.Unmarshal(msg, &env)
			if env.Type != EventSceneChanged {
				t.Errorf("client %s: Type = %q, want %q", c.userID, env.Type, EventSceneChanged)
			}
		default:
			t.Errorf("client %s: no message received", c.userID)
		}
	}

	// State should reflect new scene.
	if room.StateSnapshot().CurrentScene != "library" {
		t.Errorf("CurrentScene = %q, want %q", room.StateSnapshot().CurrentScene, "library")
	}
}

func TestAdvanceScene_PlayerForbidden(t *testing.T) {
	repo := successEventRepo()
	room, cancel := startedRoom(repo, testScenario())
	defer cancel()
	defer room.Stop()

	player := fakeClient("user-1", RolePlayer, room)
	room.register <- player
	time.Sleep(50 * time.Millisecond)

	room.incoming <- incomingMessage{client: player, data: []byte(`{"type":"advance_scene","payload":{"scene_id":"library"}}`)}
	time.Sleep(50 * time.Millisecond)

	select {
	case msg := <-player.send:
		var env Envelope
		json.Unmarshal(msg, &env)
		if env.Type != EventError {
			t.Errorf("Type = %q, want %q", env.Type, EventError)
		}
		var payload map[string]string
		json.Unmarshal(env.Payload, &payload)
		if payload["message"] != "Only the GM can advance the scene" {
			t.Errorf("message = %q", payload["message"])
		}
	default:
		t.Error("expected error, got none")
	}
}

func TestAdvanceScene_SceneNotFound(t *testing.T) {
	repo := successEventRepo()
	room, cancel := startedRoom(repo, testScenario())
	defer cancel()
	defer room.Stop()

	gm := fakeClient("gm-1", RoleGM, room)
	room.register <- gm
	time.Sleep(50 * time.Millisecond)

	room.incoming <- incomingMessage{client: gm, data: []byte(`{"type":"advance_scene","payload":{"scene_id":"nonexistent"}}`)}
	time.Sleep(50 * time.Millisecond)

	select {
	case msg := <-gm.send:
		var env Envelope
		json.Unmarshal(msg, &env)
		if env.Type != EventError {
			t.Errorf("Type = %q, want %q", env.Type, EventError)
		}
	default:
		t.Error("expected error, got none")
	}
}

func TestAdvanceScene_NoScenario(t *testing.T) {
	repo := successEventRepo()
	room, cancel := startedRoom(repo, nil) // no scenario
	defer cancel()
	defer room.Stop()

	gm := fakeClient("gm-1", RoleGM, room)
	room.register <- gm
	time.Sleep(50 * time.Millisecond)

	room.incoming <- incomingMessage{client: gm, data: []byte(`{"type":"advance_scene","payload":{"scene_id":"library"}}`)}
	time.Sleep(50 * time.Millisecond)

	select {
	case msg := <-gm.send:
		var env Envelope
		json.Unmarshal(msg, &env)
		if env.Type != EventError {
			t.Errorf("Type = %q, want %q", env.Type, EventError)
		}
		var payload map[string]string
		json.Unmarshal(env.Payload, &payload)
		if payload["message"] != "Scenario not loaded" {
			t.Errorf("message = %q", payload["message"])
		}
	default:
		t.Error("expected error, got none")
	}
}

func TestAdvanceScene_GameNotActive(t *testing.T) {
	repo := successEventRepo()
	room, cancel := startedRoom(repo, testScenario())
	defer cancel()
	defer room.Stop()

	// Pause the game.
	actorID := "gm-1"
	room.BroadcastEvent(EventGamePaused, &actorID, json.RawMessage(`{}`))
	time.Sleep(100 * time.Millisecond)

	// Verify state is paused before proceeding.
	if room.StateSnapshot().Status != "paused" {
		t.Fatalf("Status = %q, want %q (precondition)", room.StateSnapshot().Status, "paused")
	}

	gm := fakeClient("gm-1", RoleGM, room)
	room.register <- gm
	time.Sleep(50 * time.Millisecond)

	room.incoming <- incomingMessage{client: gm, data: []byte(`{"type":"advance_scene","payload":{"scene_id":"library"}}`)}
	time.Sleep(50 * time.Millisecond)

	select {
	case msg := <-gm.send:
		var env Envelope
		json.Unmarshal(msg, &env)
		if env.Type != EventError {
			t.Errorf("Type = %q, want %q", env.Type, EventError)
		}
	default:
		t.Error("expected error, got none")
	}
}

func TestAdvanceScene_EmptySceneID(t *testing.T) {
	repo := successEventRepo()
	room, cancel := startedRoom(repo, testScenario())
	defer cancel()
	defer room.Stop()

	gm := fakeClient("gm-1", RoleGM, room)
	room.register <- gm
	time.Sleep(50 * time.Millisecond)

	room.incoming <- incomingMessage{client: gm, data: []byte(`{"type":"advance_scene","payload":{"scene_id":""}}`)}
	time.Sleep(50 * time.Millisecond)

	select {
	case msg := <-gm.send:
		var env Envelope
		json.Unmarshal(msg, &env)
		if env.Type != EventError {
			t.Errorf("Type = %q, want %q", env.Type, EventError)
		}
	default:
		t.Error("expected error, got none")
	}
}

func TestAdvanceScene_PersistError_RollsBack(t *testing.T) {
	callCount := 0
	repo := &mockEventRepo{
		appendEventFn: func(_ context.Context, _ string, _ int64, eventType string, _ *string, _ json.RawMessage) (*game.GameEvent, error) {
			callCount++
			if eventType == EventSceneChanged {
				return nil, errors.New("db error")
			}
			return &game.GameEvent{ID: "evt-1"}, nil
		},
		listEventsSinceFn: func(_ context.Context, _ string, _ int64) ([]*game.GameEvent, error) {
			return []*game.GameEvent{}, nil
		},
	}
	room, cancel := startedRoom(repo, testScenario())
	defer cancel()
	defer room.Stop()

	seqBefore := room.StateSnapshot().LastSequence

	gm := fakeClient("gm-1", RoleGM, room)
	room.register <- gm
	time.Sleep(50 * time.Millisecond)

	room.incoming <- incomingMessage{client: gm, data: []byte(`{"type":"advance_scene","payload":{"scene_id":"library"}}`)}
	time.Sleep(50 * time.Millisecond)

	// Sequence should roll back.
	if room.StateSnapshot().LastSequence != seqBefore {
		t.Errorf("LastSequence = %d, want %d (rollback)", room.StateSnapshot().LastSequence, seqBefore)
	}

	// Should get error response.
	select {
	case msg := <-gm.send:
		var env Envelope
		json.Unmarshal(msg, &env)
		if env.Type != EventError {
			t.Errorf("Type = %q, want %q", env.Type, EventError)
		}
	default:
		t.Error("expected error, got none")
	}
}

func TestAdvanceScene_GMGetsGMNotes(t *testing.T) {
	repo := successEventRepo()
	room, cancel := startedRoom(repo, testScenario())
	defer cancel()
	defer room.Stop()

	gm := fakeClient("gm-1", RoleGM, room)
	room.register <- gm
	time.Sleep(50 * time.Millisecond)

	room.incoming <- incomingMessage{client: gm, data: []byte(`{"type":"advance_scene","payload":{"scene_id":"entrance"}}`)}
	time.Sleep(50 * time.Millisecond)

	select {
	case msg := <-gm.send:
		var env Envelope
		json.Unmarshal(msg, &env)
		var p map[string]json.RawMessage
		json.Unmarshal(env.Payload, &p)
		var scene map[string]json.RawMessage
		json.Unmarshal(p["scene"], &scene)
		if _, ok := scene["gm_notes"]; !ok {
			t.Error("GM should see gm_notes")
		}
	default:
		t.Error("expected message, got none")
	}
}

func TestAdvanceScene_PlayerNoGMNotes(t *testing.T) {
	repo := successEventRepo()
	room, cancel := startedRoom(repo, testScenario())
	defer cancel()
	defer room.Stop()

	gm := fakeClient("gm-1", RoleGM, room)
	player := fakeClient("user-1", RolePlayer, room)
	room.register <- gm
	room.register <- player
	time.Sleep(50 * time.Millisecond)

	room.incoming <- incomingMessage{client: gm, data: []byte(`{"type":"advance_scene","payload":{"scene_id":"entrance"}}`)}
	time.Sleep(50 * time.Millisecond)

	// Drain GM message.
	<-gm.send

	select {
	case msg := <-player.send:
		var env Envelope
		json.Unmarshal(msg, &env)
		var p map[string]json.RawMessage
		json.Unmarshal(env.Payload, &p)
		var scene map[string]json.RawMessage
		json.Unmarshal(p["scene"], &scene)
		if _, ok := scene["gm_notes"]; ok {
			t.Error("Player should NOT see gm_notes")
		}
	default:
		t.Error("expected message, got none")
	}
}

func TestAdvanceScene_UpdatesCurrentScene(t *testing.T) {
	repo := successEventRepo()
	room, cancel := startedRoom(repo, testScenario())
	defer cancel()
	defer room.Stop()

	gm := fakeClient("gm-1", RoleGM, room)
	room.register <- gm
	time.Sleep(50 * time.Millisecond)

	room.incoming <- incomingMessage{client: gm, data: []byte(`{"type":"advance_scene","payload":{"scene_id":"library"}}`)}
	time.Sleep(50 * time.Millisecond)
	<-gm.send // drain

	if room.StateSnapshot().CurrentScene != "library" {
		t.Errorf("CurrentScene = %q, want %q", room.StateSnapshot().CurrentScene, "library")
	}
}

func TestAdvanceScene_PreviousSceneInPayload(t *testing.T) {
	repo := successEventRepo()
	room, cancel := startedRoom(repo, testScenario())
	defer cancel()
	defer room.Stop()

	gm := fakeClient("gm-1", RoleGM, room)
	room.register <- gm
	time.Sleep(50 * time.Millisecond)

	// First advance to entrance.
	room.incoming <- incomingMessage{client: gm, data: []byte(`{"type":"advance_scene","payload":{"scene_id":"entrance"}}`)}
	time.Sleep(50 * time.Millisecond)
	<-gm.send // drain first message

	// Now advance to library.
	room.incoming <- incomingMessage{client: gm, data: []byte(`{"type":"advance_scene","payload":{"scene_id":"library"}}`)}
	time.Sleep(50 * time.Millisecond)

	select {
	case msg := <-gm.send:
		var env Envelope
		json.Unmarshal(msg, &env)
		var p map[string]json.RawMessage
		json.Unmarshal(env.Payload, &p)
		var prevScene string
		json.Unmarshal(p["previous_scene"], &prevScene)
		if prevScene != "entrance" {
			t.Errorf("previous_scene = %q, want %q", prevScene, "entrance")
		}
	default:
		t.Error("expected message, got none")
	}
}

// --- Step 7: dice_roll handler tests ---

func TestDiceRoll_PlayerSuccess(t *testing.T) {
	repo := successEventRepo()
	room, cancel := startedRoom(repo, testScenario())
	defer cancel()
	defer room.Stop()

	gm := fakeClient("gm-1", RoleGM, room)
	player := fakeClient("user-1", RolePlayer, room)
	room.register <- gm
	room.register <- player
	time.Sleep(50 * time.Millisecond)

	room.incoming <- incomingMessage{client: player, data: []byte(`{"type":"dice_roll","payload":{"formula":"2d6","purpose":"力量檢定"}}`)}
	time.Sleep(50 * time.Millisecond)

	// Both GM and player should receive dice_rolled.
	for _, c := range []*Client{gm, player} {
		select {
		case msg := <-c.send:
			var env Envelope
			json.Unmarshal(msg, &env)
			if env.Type != EventDiceRolled {
				t.Errorf("client %s: Type = %q, want %q", c.userID, env.Type, EventDiceRolled)
			}
		default:
			t.Errorf("client %s: no message received", c.userID)
		}
	}
}

func TestDiceRoll_GMSuccess(t *testing.T) {
	repo := successEventRepo()
	room, cancel := startedRoom(repo, testScenario())
	defer cancel()
	defer room.Stop()

	gm := fakeClient("gm-1", RoleGM, room)
	room.register <- gm
	time.Sleep(50 * time.Millisecond)

	room.incoming <- incomingMessage{client: gm, data: []byte(`{"type":"dice_roll","payload":{"formula":"1d20+3"}}`)}
	time.Sleep(50 * time.Millisecond)

	select {
	case msg := <-gm.send:
		var env Envelope
		json.Unmarshal(msg, &env)
		if env.Type != EventDiceRolled {
			t.Errorf("Type = %q, want %q", env.Type, EventDiceRolled)
		}
	default:
		t.Error("expected dice_rolled message, got none")
	}
}

func TestDiceRoll_InvalidFormula(t *testing.T) {
	repo := successEventRepo()
	room, cancel := startedRoom(repo, testScenario())
	defer cancel()
	defer room.Stop()

	player := fakeClient("user-1", RolePlayer, room)
	room.register <- player
	time.Sleep(50 * time.Millisecond)

	room.incoming <- incomingMessage{client: player, data: []byte(`{"type":"dice_roll","payload":{"formula":"invalid"}}`)}
	time.Sleep(50 * time.Millisecond)

	select {
	case msg := <-player.send:
		var env Envelope
		json.Unmarshal(msg, &env)
		if env.Type != EventError {
			t.Errorf("Type = %q, want %q", env.Type, EventError)
		}
	default:
		t.Error("expected error, got none")
	}
}

func TestDiceRoll_EmptyFormula(t *testing.T) {
	repo := successEventRepo()
	room, cancel := startedRoom(repo, testScenario())
	defer cancel()
	defer room.Stop()

	player := fakeClient("user-1", RolePlayer, room)
	room.register <- player
	time.Sleep(50 * time.Millisecond)

	room.incoming <- incomingMessage{client: player, data: []byte(`{"type":"dice_roll","payload":{"formula":""}}`)}
	time.Sleep(50 * time.Millisecond)

	select {
	case msg := <-player.send:
		var env Envelope
		json.Unmarshal(msg, &env)
		if env.Type != EventError {
			t.Errorf("Type = %q, want %q", env.Type, EventError)
		}
	default:
		t.Error("expected error, got none")
	}
}

func TestDiceRoll_GameNotActive(t *testing.T) {
	repo := successEventRepo()
	room, cancel := startedRoom(repo, testScenario())
	defer cancel()
	defer room.Stop()

	actorID := "gm-1"
	room.BroadcastEvent(EventGamePaused, &actorID, json.RawMessage(`{}`))
	time.Sleep(100 * time.Millisecond)

	player := fakeClient("user-1", RolePlayer, room)
	room.register <- player
	time.Sleep(50 * time.Millisecond)

	room.incoming <- incomingMessage{client: player, data: []byte(`{"type":"dice_roll","payload":{"formula":"1d6"}}`)}
	time.Sleep(50 * time.Millisecond)

	select {
	case msg := <-player.send:
		var env Envelope
		json.Unmarshal(msg, &env)
		if env.Type != EventError {
			t.Errorf("Type = %q, want %q", env.Type, EventError)
		}
	default:
		t.Error("expected error, got none")
	}
}

func TestDiceRoll_PersistError(t *testing.T) {
	callCount := 0
	repo := &mockEventRepo{
		appendEventFn: func(_ context.Context, _ string, _ int64, eventType string, _ *string, _ json.RawMessage) (*game.GameEvent, error) {
			callCount++
			if eventType == EventDiceRolled {
				return nil, errors.New("db error")
			}
			return &game.GameEvent{ID: "evt-1"}, nil
		},
		listEventsSinceFn: func(_ context.Context, _ string, _ int64) ([]*game.GameEvent, error) {
			return []*game.GameEvent{}, nil
		},
	}
	room, cancel := startedRoom(repo, testScenario())
	defer cancel()
	defer room.Stop()

	seqBefore := room.StateSnapshot().LastSequence

	player := fakeClient("user-1", RolePlayer, room)
	room.register <- player
	time.Sleep(50 * time.Millisecond)

	room.incoming <- incomingMessage{client: player, data: []byte(`{"type":"dice_roll","payload":{"formula":"1d6"}}`)}
	time.Sleep(50 * time.Millisecond)

	if room.StateSnapshot().LastSequence != seqBefore {
		t.Errorf("LastSequence = %d, want %d", room.StateSnapshot().LastSequence, seqBefore)
	}

	select {
	case msg := <-player.send:
		var env Envelope
		json.Unmarshal(msg, &env)
		if env.Type != EventError {
			t.Errorf("Type = %q, want %q", env.Type, EventError)
		}
	default:
		t.Error("expected error, got none")
	}
}

func TestDiceRoll_WithPurpose(t *testing.T) {
	repo := successEventRepo()
	room, cancel := startedRoom(repo, testScenario())
	defer cancel()
	defer room.Stop()

	player := fakeClient("user-1", RolePlayer, room)
	room.register <- player
	time.Sleep(50 * time.Millisecond)

	room.incoming <- incomingMessage{client: player, data: []byte(`{"type":"dice_roll","payload":{"formula":"1d20","purpose":"感知檢定"}}`)}
	time.Sleep(50 * time.Millisecond)

	select {
	case msg := <-player.send:
		var env Envelope
		json.Unmarshal(msg, &env)
		var payload map[string]any
		json.Unmarshal(env.Payload, &payload)
		if payload["purpose"] != "感知檢定" {
			t.Errorf("purpose = %v, want %q", payload["purpose"], "感知檢定")
		}
	default:
		t.Error("expected dice_rolled message, got none")
	}
}

func TestDiceRoll_NoPurpose(t *testing.T) {
	repo := successEventRepo()
	room, cancel := startedRoom(repo, testScenario())
	defer cancel()
	defer room.Stop()

	player := fakeClient("user-1", RolePlayer, room)
	room.register <- player
	time.Sleep(50 * time.Millisecond)

	room.incoming <- incomingMessage{client: player, data: []byte(`{"type":"dice_roll","payload":{"formula":"1d6"}}`)}
	time.Sleep(50 * time.Millisecond)

	select {
	case msg := <-player.send:
		var env Envelope
		json.Unmarshal(msg, &env)
		if env.Type != EventDiceRolled {
			t.Errorf("Type = %q, want %q", env.Type, EventDiceRolled)
		}
	default:
		t.Error("expected dice_rolled message, got none")
	}
}

func TestDiceRoll_UpdatesDiceHistory(t *testing.T) {
	repo := successEventRepo()
	room, cancel := startedRoom(repo, testScenario())
	defer cancel()
	defer room.Stop()

	player := fakeClient("user-1", RolePlayer, room)
	room.register <- player
	time.Sleep(50 * time.Millisecond)

	room.incoming <- incomingMessage{client: player, data: []byte(`{"type":"dice_roll","payload":{"formula":"2d6"}}`)}
	time.Sleep(50 * time.Millisecond)
	<-player.send // drain

	state := room.StateSnapshot()
	if len(state.DiceHistory) != 1 {
		t.Fatalf("len(DiceHistory) = %d, want 1", len(state.DiceHistory))
	}
	if state.DiceHistory[0].Formula != "2d6" {
		t.Errorf("Formula = %q, want %q", state.DiceHistory[0].Formula, "2d6")
	}
}

func TestDiceRoll_BroadcastContainsRollerID(t *testing.T) {
	repo := successEventRepo()
	room, cancel := startedRoom(repo, testScenario())
	defer cancel()
	defer room.Stop()

	player := fakeClient("user-1", RolePlayer, room)
	room.register <- player
	time.Sleep(50 * time.Millisecond)

	room.incoming <- incomingMessage{client: player, data: []byte(`{"type":"dice_roll","payload":{"formula":"1d6"}}`)}
	time.Sleep(50 * time.Millisecond)

	select {
	case msg := <-player.send:
		var env Envelope
		json.Unmarshal(msg, &env)
		var payload map[string]any
		json.Unmarshal(env.Payload, &payload)
		if payload["roller_id"] != "user-1" {
			t.Errorf("roller_id = %v, want %q", payload["roller_id"], "user-1")
		}
	default:
		t.Error("expected dice_rolled message, got none")
	}
}
