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

// --- SPEC-007 helpers ---

// testScenarioFull creates a full scenario with items, NPCs, variables, on_enter/on_exit actions, and player_choice transitions.
func testScenarioFull() *ScenarioContent {
	return &ScenarioContent{
		ID:         "test-scenario",
		Title:      "Test",
		StartScene: "entrance",
		Scenes: []Scene{
			{
				ID: "entrance", Name: "Entrance", Content: "You enter.",
				GMNotes:        "GM secret notes",
				ItemsAvailable: []string{"rusty_key"},
				NPCsPresent:    []string{"butler"},
				Transitions: []Transition{
					{Target: "library", Trigger: "player_choice", Label: "Go to library"},
					{Target: "kitchen", Trigger: "gm_decision", Label: "GM moves to kitchen"},
				},
				OnEnter: []Action{
					{SetVar: &SetVarAction{Name: "visited_entrance", Value: true}},
				},
			},
			{
				ID: "library", Name: "Library", Content: "Books everywhere.",
				GMNotes:        "Check bookshelf",
				ItemsAvailable: []string{"torn_diary"},
				NPCsPresent:    []string{"ghost_child"},
				OnEnter: []Action{
					{RevealItem: &RevealItemAction{ItemID: "torn_diary", To: "current_player"}},
					{RevealNPCField: &RevealNPCFieldAction{NPCID: "ghost_child", FieldKey: "background", To: "current_player"}},
				},
				OnExit: []Action{
					{SetVar: &SetVarAction{Name: "left_library", Value: true}},
				},
			},
			{ID: "kitchen", Name: "Kitchen", Content: "Pots and pans."},
		},
		Items: []Item{
			{ID: "rusty_key", Name: "Rusty Key", Type: "item", Description: "A rusty key"},
			{ID: "torn_diary", Name: "Torn Diary", Type: "clue", Description: "A torn diary"},
		},
		NPCs: []NPC{
			{
				ID: "butler", Name: "Old Butler",
				Fields: []NPCField{
					{Key: "appearance", Label: "Appearance", Value: "Tall", Visibility: "public"},
					{Key: "secret", Label: "Secret", Value: "He is a ghost", Visibility: "hidden"},
				},
			},
			{
				ID: "ghost_child", Name: "Ghost Child",
				Fields: []NPCField{
					{Key: "appearance", Label: "Appearance", Value: "Translucent", Visibility: "public"},
					{Key: "background", Label: "Background", Value: "Died in a fire", Visibility: "hidden"},
				},
			},
		},
		Variables: []Variable{
			{Name: "visited_entrance", Type: "bool", Default: false},
			{Name: "left_library", Type: "bool", Default: false},
			{Name: "ghost_anger", Type: "int", Default: float64(0)},
		},
	}
}

// startedRoomFull creates a Room with full scenario, starts it, registers GM + player, and fires game_started.
func startedRoomFull(repo *mockEventRepo) (*Room, *Client, *Client, context.CancelFunc) {
	sc := testScenarioFull()
	room := NewRoom("sess-1", "gm-1", sc, repo, testLogger())
	ctx, cancel := context.WithCancel(context.Background())
	go room.Run(ctx)

	gm := fakeClient("gm-1", RoleGM, room)
	player := fakeClient("player-1", RolePlayer, room)
	room.register <- gm
	room.register <- player
	time.Sleep(50 * time.Millisecond)

	// Fire game_started.
	actorID := "gm-1"
	room.BroadcastEvent(EventGameStarted, &actorID, json.RawMessage(`{}`))
	time.Sleep(50 * time.Millisecond)

	// Drain game_started from client buffers.
	drainChannel(gm.send)
	drainChannel(player.send)

	return room, gm, player, cancel
}

// drainChannel drains all messages from a channel.
func drainChannel(ch chan []byte) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}

// --- Step 5+6: Variable initialization tests ---

func TestNewRoom_WithScenarioVariables(t *testing.T) {
	sc := testScenarioFull()
	repo := successEventRepo()
	room := NewRoom("sess-1", "gm-1", sc, repo, testLogger())

	state := room.state
	if state.Variables == nil {
		t.Fatal("Variables should not be nil")
	}
	if state.Variables["visited_entrance"] != false {
		t.Errorf("visited_entrance = %v, want false", state.Variables["visited_entrance"])
	}
	if state.Variables["ghost_anger"] != float64(0) {
		t.Errorf("ghost_anger = %v, want 0", state.Variables["ghost_anger"])
	}
}

func TestNewRoom_WithoutScenario(t *testing.T) {
	repo := successEventRepo()
	room := NewRoom("sess-1", "gm-1", nil, repo, testLogger())
	if room.state.Variables != nil {
		t.Errorf("Variables should be nil without scenario, got %v", room.state.Variables)
	}
}

func TestNewRoom_EmptyVariables(t *testing.T) {
	sc := &ScenarioContent{
		StartScene: "s1",
		Scenes:     []Scene{{ID: "s1", Name: "S1", Content: "C1"}},
	}
	repo := successEventRepo()
	room := NewRoom("sess-1", "gm-1", sc, repo, testLogger())
	if room.state.Variables != nil {
		t.Errorf("Variables should be nil for empty variables, got %v", room.state.Variables)
	}
}

func TestConnectedPlayerIDs(t *testing.T) {
	repo := successEventRepo()
	room, gm, player, cancel := startedRoomFull(repo)
	defer cancel()
	defer room.Stop()

	// connectedPlayerIDs is called from inside Room goroutine.
	// We test it indirectly through handlers that use it.
	// For direct test, we need to query via the channel.
	_ = gm
	_ = player
	// Just verify the room is set up correctly.
	if room.ClientCount() != 2 {
		t.Errorf("ClientCount = %d, want 2", room.ClientCount())
	}
}

// --- handleRevealItem tests ---

func TestRevealItem_GMSuccess(t *testing.T) {
	repo := successEventRepo()
	room, gm, player, cancel := startedRoomFull(repo)
	defer cancel()
	defer room.Stop()

	msg := json.RawMessage(`{"type":"reveal_item","payload":{"item_id":"rusty_key","player_ids":["player-1"]}}`)
	room.incoming <- incomingMessage{client: gm, data: msg}
	time.Sleep(50 * time.Millisecond)

	// Check player received the event.
	select {
	case data := <-player.send:
		var env Envelope
		json.Unmarshal(data, &env)
		if env.Type != EventItemRevealed {
			t.Errorf("Type = %q, want %q", env.Type, EventItemRevealed)
		}
	default:
		t.Error("expected item_revealed message, got none")
	}

	// Check state updated.
	state := room.StateSnapshot()
	if !state.IsItemRevealed("player-1", "rusty_key") {
		t.Error("expected rusty_key to be revealed for player-1")
	}
}

func TestRevealItem_PlayerForbidden(t *testing.T) {
	repo := successEventRepo()
	room, _, player, cancel := startedRoomFull(repo)
	defer cancel()
	defer room.Stop()

	msg := json.RawMessage(`{"type":"reveal_item","payload":{"item_id":"rusty_key","player_ids":["player-1"]}}`)
	room.incoming <- incomingMessage{client: player, data: msg}
	time.Sleep(50 * time.Millisecond)

	select {
	case data := <-player.send:
		var env Envelope
		json.Unmarshal(data, &env)
		if env.Type != EventError {
			t.Errorf("Type = %q, want %q", env.Type, EventError)
		}
	default:
		t.Error("expected error message, got none")
	}
}

func TestRevealItem_GameNotActive(t *testing.T) {
	repo := successEventRepo()
	sc := testScenarioFull()
	room := NewRoom("sess-1", "gm-1", sc, repo, testLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go room.Run(ctx)
	defer room.Stop()

	// Don't fire game_started — status remains "active" by default but let's pause it.
	actorID := "gm-1"
	room.BroadcastEvent(EventGameStarted, &actorID, json.RawMessage(`{}`))
	time.Sleep(50 * time.Millisecond)
	room.BroadcastEvent(EventGamePaused, &actorID, json.RawMessage(`{}`))
	time.Sleep(50 * time.Millisecond)

	gm := fakeClient("gm-1", RoleGM, room)
	room.register <- gm
	time.Sleep(50 * time.Millisecond)
	drainChannel(gm.send)

	msg := json.RawMessage(`{"type":"reveal_item","payload":{"item_id":"rusty_key"}}`)
	room.incoming <- incomingMessage{client: gm, data: msg}
	time.Sleep(50 * time.Millisecond)

	select {
	case data := <-gm.send:
		var env Envelope
		json.Unmarshal(data, &env)
		if env.Type != EventError {
			t.Errorf("Type = %q, want %q", env.Type, EventError)
		}
	default:
		t.Error("expected error message, got none")
	}
}

func TestRevealItem_InvalidPayload(t *testing.T) {
	repo := successEventRepo()
	room, gm, _, cancel := startedRoomFull(repo)
	defer cancel()
	defer room.Stop()

	msg := json.RawMessage(`{"type":"reveal_item","payload":"invalid"}`)
	room.incoming <- incomingMessage{client: gm, data: msg}
	time.Sleep(50 * time.Millisecond)

	select {
	case data := <-gm.send:
		var env Envelope
		json.Unmarshal(data, &env)
		if env.Type != EventError {
			t.Errorf("Type = %q, want %q", env.Type, EventError)
		}
	default:
		t.Error("expected error message, got none")
	}
}

func TestRevealItem_ItemNotFound(t *testing.T) {
	repo := successEventRepo()
	room, gm, _, cancel := startedRoomFull(repo)
	defer cancel()
	defer room.Stop()

	msg := json.RawMessage(`{"type":"reveal_item","payload":{"item_id":"nonexistent"}}`)
	room.incoming <- incomingMessage{client: gm, data: msg}
	time.Sleep(50 * time.Millisecond)

	select {
	case data := <-gm.send:
		var env Envelope
		json.Unmarshal(data, &env)
		if env.Type != EventError {
			t.Errorf("Type = %q, want %q", env.Type, EventError)
		}
	default:
		t.Error("expected error message, got none")
	}
}

func TestRevealItem_NoScenario(t *testing.T) {
	repo := successEventRepo()
	room, cancel := startedRoom(repo, nil)
	defer cancel()
	defer room.Stop()

	gm := fakeClient("gm-1", RoleGM, room)
	room.register <- gm
	time.Sleep(50 * time.Millisecond)
	drainChannel(gm.send)

	msg := json.RawMessage(`{"type":"reveal_item","payload":{"item_id":"key"}}`)
	room.incoming <- incomingMessage{client: gm, data: msg}
	time.Sleep(50 * time.Millisecond)

	select {
	case data := <-gm.send:
		var env Envelope
		json.Unmarshal(data, &env)
		if env.Type != EventError {
			t.Errorf("Type = %q, want %q", env.Type, EventError)
		}
	default:
		t.Error("expected error message, got none")
	}
}

func TestRevealItem_EmptyItemID(t *testing.T) {
	repo := successEventRepo()
	room, gm, _, cancel := startedRoomFull(repo)
	defer cancel()
	defer room.Stop()

	msg := json.RawMessage(`{"type":"reveal_item","payload":{"item_id":""}}`)
	room.incoming <- incomingMessage{client: gm, data: msg}
	time.Sleep(50 * time.Millisecond)

	select {
	case data := <-gm.send:
		var env Envelope
		json.Unmarshal(data, &env)
		if env.Type != EventError {
			t.Errorf("Type = %q, want %q", env.Type, EventError)
		}
	default:
		t.Error("expected error message, got none")
	}
}

func TestRevealItem_UpdatesGameState(t *testing.T) {
	repo := successEventRepo()
	room, gm, _, cancel := startedRoomFull(repo)
	defer cancel()
	defer room.Stop()

	msg := json.RawMessage(`{"type":"reveal_item","payload":{"item_id":"rusty_key","player_ids":["player-1"]}}`)
	room.incoming <- incomingMessage{client: gm, data: msg}
	time.Sleep(50 * time.Millisecond)

	state := room.StateSnapshot()
	if !state.IsItemRevealed("player-1", "rusty_key") {
		t.Error("expected rusty_key to be revealed for player-1")
	}
}

func TestRevealItem_PersistError(t *testing.T) {
	callCount := int64(0)
	repo := &mockEventRepo{
		appendEventFn: func(_ context.Context, _ string, _ int64, eventType string, _ *string, _ json.RawMessage) (*game.GameEvent, error) {
			n := atomic.AddInt64(&callCount, 1)
			// First call is game_started, second is reveal_item.
			if n > 1 && eventType == EventItemRevealed {
				return nil, errors.New("db error")
			}
			return &game.GameEvent{ID: "evt-1", Type: eventType, CreatedAt: time.Now()}, nil
		},
		listEventsSinceFn: func(_ context.Context, _ string, _ int64) ([]*game.GameEvent, error) {
			return nil, nil
		},
	}
	room, gm, _, cancel := startedRoomFull(repo)
	defer cancel()
	defer room.Stop()

	msg := json.RawMessage(`{"type":"reveal_item","payload":{"item_id":"rusty_key","player_ids":["player-1"]}}`)
	room.incoming <- incomingMessage{client: gm, data: msg}
	time.Sleep(50 * time.Millisecond)

	// State should NOT be updated on persist error.
	state := room.StateSnapshot()
	if state.IsItemRevealed("player-1", "rusty_key") {
		t.Error("expected rusty_key to NOT be revealed after persist error")
	}
}

func TestRevealItem_AllPlayers(t *testing.T) {
	repo := successEventRepo()
	room, gm, player, cancel := startedRoomFull(repo)
	defer cancel()
	defer room.Stop()

	// Empty player_ids means all connected players.
	msg := json.RawMessage(`{"type":"reveal_item","payload":{"item_id":"rusty_key"}}`)
	room.incoming <- incomingMessage{client: gm, data: msg}
	time.Sleep(50 * time.Millisecond)

	// Drain GM and player messages.
	drainChannel(gm.send)
	drainChannel(player.send)

	state := room.StateSnapshot()
	if !state.IsItemRevealed("player-1", "rusty_key") {
		t.Error("expected rusty_key to be revealed for player-1")
	}
}

// --- handleRevealNPCField tests ---

func TestRevealNPCField_GMSuccess(t *testing.T) {
	repo := successEventRepo()
	room, gm, player, cancel := startedRoomFull(repo)
	defer cancel()
	defer room.Stop()

	msg := json.RawMessage(`{"type":"reveal_npc_field","payload":{"npc_id":"butler","field_key":"secret","player_ids":["player-1"]}}`)
	room.incoming <- incomingMessage{client: gm, data: msg}
	time.Sleep(50 * time.Millisecond)

	select {
	case data := <-player.send:
		var env Envelope
		json.Unmarshal(data, &env)
		if env.Type != EventNPCFieldRevealed {
			t.Errorf("Type = %q, want %q", env.Type, EventNPCFieldRevealed)
		}
	default:
		t.Error("expected npc_field_revealed message, got none")
	}

	state := room.StateSnapshot()
	fields := state.RevealedFieldsForNPC("player-1", "butler")
	if len(fields) != 1 || fields[0] != "secret" {
		t.Errorf("RevealedFields = %v, want [secret]", fields)
	}
}

func TestRevealNPCField_PlayerForbidden(t *testing.T) {
	repo := successEventRepo()
	room, _, player, cancel := startedRoomFull(repo)
	defer cancel()
	defer room.Stop()

	msg := json.RawMessage(`{"type":"reveal_npc_field","payload":{"npc_id":"butler","field_key":"secret"}}`)
	room.incoming <- incomingMessage{client: player, data: msg}
	time.Sleep(50 * time.Millisecond)

	select {
	case data := <-player.send:
		var env Envelope
		json.Unmarshal(data, &env)
		if env.Type != EventError {
			t.Errorf("Type = %q, want %q", env.Type, EventError)
		}
	default:
		t.Error("expected error message, got none")
	}
}

func TestRevealNPCField_GameNotActive(t *testing.T) {
	repo := successEventRepo()
	sc := testScenarioFull()
	room := NewRoom("sess-1", "gm-1", sc, repo, testLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go room.Run(ctx)
	defer room.Stop()

	actorID := "gm-1"
	room.BroadcastEvent(EventGameStarted, &actorID, json.RawMessage(`{}`))
	time.Sleep(50 * time.Millisecond)
	room.BroadcastEvent(EventGamePaused, &actorID, json.RawMessage(`{}`))
	time.Sleep(50 * time.Millisecond)

	gm := fakeClient("gm-1", RoleGM, room)
	room.register <- gm
	time.Sleep(50 * time.Millisecond)
	drainChannel(gm.send)

	msg := json.RawMessage(`{"type":"reveal_npc_field","payload":{"npc_id":"butler","field_key":"secret"}}`)
	room.incoming <- incomingMessage{client: gm, data: msg}
	time.Sleep(50 * time.Millisecond)

	select {
	case data := <-gm.send:
		var env Envelope
		json.Unmarshal(data, &env)
		if env.Type != EventError {
			t.Errorf("Type = %q, want %q", env.Type, EventError)
		}
	default:
		t.Error("expected error message, got none")
	}
}

func TestRevealNPCField_NPCNotFound(t *testing.T) {
	repo := successEventRepo()
	room, gm, _, cancel := startedRoomFull(repo)
	defer cancel()
	defer room.Stop()

	msg := json.RawMessage(`{"type":"reveal_npc_field","payload":{"npc_id":"nonexistent","field_key":"secret"}}`)
	room.incoming <- incomingMessage{client: gm, data: msg}
	time.Sleep(50 * time.Millisecond)

	select {
	case data := <-gm.send:
		var env Envelope
		json.Unmarshal(data, &env)
		if env.Type != EventError {
			t.Errorf("Type = %q, want %q", env.Type, EventError)
		}
	default:
		t.Error("expected error message, got none")
	}
}

func TestRevealNPCField_FieldNotFound(t *testing.T) {
	repo := successEventRepo()
	room, gm, _, cancel := startedRoomFull(repo)
	defer cancel()
	defer room.Stop()

	msg := json.RawMessage(`{"type":"reveal_npc_field","payload":{"npc_id":"butler","field_key":"nonexistent"}}`)
	room.incoming <- incomingMessage{client: gm, data: msg}
	time.Sleep(50 * time.Millisecond)

	select {
	case data := <-gm.send:
		var env Envelope
		json.Unmarshal(data, &env)
		if env.Type != EventError {
			t.Errorf("Type = %q, want %q", env.Type, EventError)
		}
	default:
		t.Error("expected error message, got none")
	}
}

func TestRevealNPCField_UpdatesGameState(t *testing.T) {
	repo := successEventRepo()
	room, gm, _, cancel := startedRoomFull(repo)
	defer cancel()
	defer room.Stop()

	msg := json.RawMessage(`{"type":"reveal_npc_field","payload":{"npc_id":"butler","field_key":"secret","player_ids":["player-1"]}}`)
	room.incoming <- incomingMessage{client: gm, data: msg}
	time.Sleep(50 * time.Millisecond)

	state := room.StateSnapshot()
	fields := state.RevealedFieldsForNPC("player-1", "butler")
	if len(fields) != 1 || fields[0] != "secret" {
		t.Errorf("RevealedFields = %v, want [secret]", fields)
	}
}

func TestRevealNPCField_PersistError(t *testing.T) {
	callCount := int64(0)
	repo := &mockEventRepo{
		appendEventFn: func(_ context.Context, _ string, _ int64, eventType string, _ *string, _ json.RawMessage) (*game.GameEvent, error) {
			n := atomic.AddInt64(&callCount, 1)
			if n > 1 && eventType == EventNPCFieldRevealed {
				return nil, errors.New("db error")
			}
			return &game.GameEvent{ID: "evt-1", Type: eventType, CreatedAt: time.Now()}, nil
		},
		listEventsSinceFn: func(_ context.Context, _ string, _ int64) ([]*game.GameEvent, error) {
			return nil, nil
		},
	}
	room, gm, _, cancel := startedRoomFull(repo)
	defer cancel()
	defer room.Stop()

	msg := json.RawMessage(`{"type":"reveal_npc_field","payload":{"npc_id":"butler","field_key":"secret","player_ids":["player-1"]}}`)
	room.incoming <- incomingMessage{client: gm, data: msg}
	time.Sleep(50 * time.Millisecond)

	state := room.StateSnapshot()
	fields := state.RevealedFieldsForNPC("player-1", "butler")
	if len(fields) != 0 {
		t.Errorf("expected no revealed fields after persist error, got %v", fields)
	}
}

func TestRevealNPCField_AllPlayers(t *testing.T) {
	repo := successEventRepo()
	room, gm, _, cancel := startedRoomFull(repo)
	defer cancel()
	defer room.Stop()

	// Empty player_ids means all connected players.
	msg := json.RawMessage(`{"type":"reveal_npc_field","payload":{"npc_id":"butler","field_key":"secret"}}`)
	room.incoming <- incomingMessage{client: gm, data: msg}
	time.Sleep(50 * time.Millisecond)
	drainChannel(gm.send)

	state := room.StateSnapshot()
	fields := state.RevealedFieldsForNPC("player-1", "butler")
	if len(fields) != 1 || fields[0] != "secret" {
		t.Errorf("RevealedFields for player-1 = %v, want [secret]", fields)
	}
}

// --- handlePlayerChoice tests ---

func TestPlayerChoice_Success(t *testing.T) {
	repo := successEventRepo()
	room, gm, player, cancel := startedRoomFull(repo)
	defer cancel()
	defer room.Stop()

	// First, GM must advance to entrance so there are transitions available.
	advMsg := json.RawMessage(`{"type":"advance_scene","payload":{"scene_id":"entrance"}}`)
	room.incoming <- incomingMessage{client: gm, data: advMsg}
	time.Sleep(50 * time.Millisecond)
	drainChannel(gm.send)
	drainChannel(player.send)

	// Player makes a choice: transition_index 0 is "Go to library" (player_choice).
	choiceMsg := json.RawMessage(`{"type":"player_choice","payload":{"transition_index":0}}`)
	room.incoming <- incomingMessage{client: player, data: choiceMsg}
	time.Sleep(100 * time.Millisecond)

	// Should receive player_choice + scene_changed events (and possibly action events).
	var receivedTypes []string
	for {
		select {
		case data := <-player.send:
			var env Envelope
			json.Unmarshal(data, &env)
			receivedTypes = append(receivedTypes, env.Type)
		default:
			goto done
		}
	}
done:
	hasPlayerChoice := false
	hasSceneChanged := false
	for _, t2 := range receivedTypes {
		if t2 == EventPlayerChoice {
			hasPlayerChoice = true
		}
		if t2 == EventSceneChanged {
			hasSceneChanged = true
		}
	}
	if !hasPlayerChoice {
		t.Error("expected player_choice event")
	}
	if !hasSceneChanged {
		t.Error("expected scene_changed event")
	}

	state := room.StateSnapshot()
	if state.CurrentScene != "library" {
		t.Errorf("CurrentScene = %q, want %q", state.CurrentScene, "library")
	}
}

func TestPlayerChoice_GameNotActive(t *testing.T) {
	repo := successEventRepo()
	sc := testScenarioFull()
	room := NewRoom("sess-1", "gm-1", sc, repo, testLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go room.Run(ctx)
	defer room.Stop()

	actorID := "gm-1"
	room.BroadcastEvent(EventGameStarted, &actorID, json.RawMessage(`{}`))
	time.Sleep(50 * time.Millisecond)
	room.BroadcastEvent(EventGamePaused, &actorID, json.RawMessage(`{}`))
	time.Sleep(50 * time.Millisecond)

	player := fakeClient("player-1", RolePlayer, room)
	room.register <- player
	time.Sleep(50 * time.Millisecond)
	drainChannel(player.send)

	msg := json.RawMessage(`{"type":"player_choice","payload":{"transition_index":0}}`)
	room.incoming <- incomingMessage{client: player, data: msg}
	time.Sleep(50 * time.Millisecond)

	select {
	case data := <-player.send:
		var env Envelope
		json.Unmarshal(data, &env)
		if env.Type != EventError {
			t.Errorf("Type = %q, want %q", env.Type, EventError)
		}
	default:
		t.Error("expected error message, got none")
	}
}

func TestPlayerChoice_InvalidPayload(t *testing.T) {
	repo := successEventRepo()
	room, _, player, cancel := startedRoomFull(repo)
	defer cancel()
	defer room.Stop()

	msg := json.RawMessage(`{"type":"player_choice","payload":"invalid"}`)
	room.incoming <- incomingMessage{client: player, data: msg}
	time.Sleep(50 * time.Millisecond)

	select {
	case data := <-player.send:
		var env Envelope
		json.Unmarshal(data, &env)
		if env.Type != EventError {
			t.Errorf("Type = %q, want %q", env.Type, EventError)
		}
	default:
		t.Error("expected error message, got none")
	}
}

func TestPlayerChoice_NoScenario(t *testing.T) {
	repo := successEventRepo()
	room, cancel := startedRoom(repo, nil)
	defer cancel()
	defer room.Stop()

	player := fakeClient("player-1", RolePlayer, room)
	room.register <- player
	time.Sleep(50 * time.Millisecond)
	drainChannel(player.send)

	msg := json.RawMessage(`{"type":"player_choice","payload":{"transition_index":0}}`)
	room.incoming <- incomingMessage{client: player, data: msg}
	time.Sleep(50 * time.Millisecond)

	select {
	case data := <-player.send:
		var env Envelope
		json.Unmarshal(data, &env)
		if env.Type != EventError {
			t.Errorf("Type = %q, want %q", env.Type, EventError)
		}
	default:
		t.Error("expected error message, got none")
	}
}

func TestPlayerChoice_NoCurrentScene(t *testing.T) {
	repo := successEventRepo()
	room, _, player, cancel := startedRoomFull(repo)
	defer cancel()
	defer room.Stop()

	// No scene has been set yet.
	msg := json.RawMessage(`{"type":"player_choice","payload":{"transition_index":0}}`)
	room.incoming <- incomingMessage{client: player, data: msg}
	time.Sleep(50 * time.Millisecond)

	select {
	case data := <-player.send:
		var env Envelope
		json.Unmarshal(data, &env)
		if env.Type != EventError {
			t.Errorf("Type = %q, want %q", env.Type, EventError)
		}
	default:
		t.Error("expected error message, got none")
	}
}

func TestPlayerChoice_TransitionIndexOutOfRange(t *testing.T) {
	repo := successEventRepo()
	room, gm, player, cancel := startedRoomFull(repo)
	defer cancel()
	defer room.Stop()

	// Set current scene to entrance.
	advMsg := json.RawMessage(`{"type":"advance_scene","payload":{"scene_id":"entrance"}}`)
	room.incoming <- incomingMessage{client: gm, data: advMsg}
	time.Sleep(50 * time.Millisecond)
	drainChannel(gm.send)
	drainChannel(player.send)

	msg := json.RawMessage(`{"type":"player_choice","payload":{"transition_index":99}}`)
	room.incoming <- incomingMessage{client: player, data: msg}
	time.Sleep(50 * time.Millisecond)

	select {
	case data := <-player.send:
		var env Envelope
		json.Unmarshal(data, &env)
		if env.Type != EventError {
			t.Errorf("Type = %q, want %q", env.Type, EventError)
		}
	default:
		t.Error("expected error message, got none")
	}
}

func TestPlayerChoice_TransitionNotPlayerChoice(t *testing.T) {
	repo := successEventRepo()
	room, gm, player, cancel := startedRoomFull(repo)
	defer cancel()
	defer room.Stop()

	// Set current scene to entrance.
	advMsg := json.RawMessage(`{"type":"advance_scene","payload":{"scene_id":"entrance"}}`)
	room.incoming <- incomingMessage{client: gm, data: advMsg}
	time.Sleep(50 * time.Millisecond)
	drainChannel(gm.send)
	drainChannel(player.send)

	// Index 1 is "gm_decision" trigger, not "player_choice".
	msg := json.RawMessage(`{"type":"player_choice","payload":{"transition_index":1}}`)
	room.incoming <- incomingMessage{client: player, data: msg}
	time.Sleep(50 * time.Millisecond)

	select {
	case data := <-player.send:
		var env Envelope
		json.Unmarshal(data, &env)
		if env.Type != EventError {
			t.Errorf("Type = %q, want %q", env.Type, EventError)
		}
	default:
		t.Error("expected error message, got none")
	}
}

func TestPlayerChoice_ChangesScene(t *testing.T) {
	repo := successEventRepo()
	room, gm, player, cancel := startedRoomFull(repo)
	defer cancel()
	defer room.Stop()

	// Set current scene to entrance.
	advMsg := json.RawMessage(`{"type":"advance_scene","payload":{"scene_id":"entrance"}}`)
	room.incoming <- incomingMessage{client: gm, data: advMsg}
	time.Sleep(50 * time.Millisecond)
	drainChannel(gm.send)
	drainChannel(player.send)

	choiceMsg := json.RawMessage(`{"type":"player_choice","payload":{"transition_index":0}}`)
	room.incoming <- incomingMessage{client: player, data: choiceMsg}
	time.Sleep(100 * time.Millisecond)

	state := room.StateSnapshot()
	if state.CurrentScene != "library" {
		t.Errorf("CurrentScene = %q, want %q", state.CurrentScene, "library")
	}
}

func TestPlayerChoice_PersistError(t *testing.T) {
	callCount := int64(0)
	repo := &mockEventRepo{
		appendEventFn: func(_ context.Context, _ string, _ int64, eventType string, _ *string, _ json.RawMessage) (*game.GameEvent, error) {
			n := atomic.AddInt64(&callCount, 1)
			// Allow game_started + advance_scene events, fail on player_choice.
			if n > 2 && eventType == EventPlayerChoice {
				return nil, errors.New("db error")
			}
			return &game.GameEvent{ID: "evt-1", Type: eventType, CreatedAt: time.Now()}, nil
		},
		listEventsSinceFn: func(_ context.Context, _ string, _ int64) ([]*game.GameEvent, error) {
			return nil, nil
		},
	}

	sc := testScenarioFull()
	room := NewRoom("sess-1", "gm-1", sc, repo, testLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go room.Run(ctx)
	defer room.Stop()

	gm := fakeClient("gm-1", RoleGM, room)
	player := fakeClient("player-1", RolePlayer, room)
	room.register <- gm
	room.register <- player
	time.Sleep(50 * time.Millisecond)

	actorID := "gm-1"
	room.BroadcastEvent(EventGameStarted, &actorID, json.RawMessage(`{}`))
	time.Sleep(50 * time.Millisecond)
	drainChannel(gm.send)
	drainChannel(player.send)

	advMsg := json.RawMessage(`{"type":"advance_scene","payload":{"scene_id":"entrance"}}`)
	room.incoming <- incomingMessage{client: gm, data: advMsg}
	time.Sleep(50 * time.Millisecond)
	drainChannel(gm.send)
	drainChannel(player.send)

	choiceMsg := json.RawMessage(`{"type":"player_choice","payload":{"transition_index":0}}`)
	room.incoming <- incomingMessage{client: player, data: choiceMsg}
	time.Sleep(50 * time.Millisecond)

	// Scene should NOT have changed due to persist error.
	state := room.StateSnapshot()
	if state.CurrentScene != "entrance" {
		t.Errorf("CurrentScene = %q, want %q (should not change on persist error)", state.CurrentScene, "entrance")
	}
}

// --- on_enter / on_exit integration tests ---

func TestAdvanceScene_OnEnter_SetVar(t *testing.T) {
	repo := successEventRepo()
	room, gm, _, cancel := startedRoomFull(repo)
	defer cancel()
	defer room.Stop()

	// Advance to entrance — has on_enter: set_var visited_entrance=true.
	advMsg := json.RawMessage(`{"type":"advance_scene","payload":{"scene_id":"entrance"}}`)
	room.incoming <- incomingMessage{client: gm, data: advMsg}
	time.Sleep(100 * time.Millisecond)

	state := room.StateSnapshot()
	if state.Variables["visited_entrance"] != true {
		t.Errorf("visited_entrance = %v, want true", state.Variables["visited_entrance"])
	}
}

func TestAdvanceScene_OnEnter_RevealItem(t *testing.T) {
	repo := successEventRepo()
	room, gm, player, cancel := startedRoomFull(repo)
	defer cancel()
	defer room.Stop()

	// First go to entrance, then library (which has on_enter: reveal_item torn_diary to current_player).
	advMsg := json.RawMessage(`{"type":"advance_scene","payload":{"scene_id":"entrance"}}`)
	room.incoming <- incomingMessage{client: gm, data: advMsg}
	time.Sleep(50 * time.Millisecond)
	drainChannel(gm.send)
	drainChannel(player.send)

	advMsg2 := json.RawMessage(`{"type":"advance_scene","payload":{"scene_id":"library"}}`)
	room.incoming <- incomingMessage{client: gm, data: advMsg2}
	time.Sleep(100 * time.Millisecond)

	// The GM is the trigger player for advance_scene, so "current_player" resolves to gm-1.
	// But gm-1 is the GM, not a player. That's expected behavior — the on_enter action
	// targets whoever triggered the transition.
	state := room.StateSnapshot()
	if !state.IsItemRevealed("gm-1", "torn_diary") {
		t.Error("expected torn_diary to be revealed for gm-1 (trigger player)")
	}
}

func TestAdvanceScene_OnEnter_RevealNPCField(t *testing.T) {
	repo := successEventRepo()
	room, gm, _, cancel := startedRoomFull(repo)
	defer cancel()
	defer room.Stop()

	advMsg := json.RawMessage(`{"type":"advance_scene","payload":{"scene_id":"entrance"}}`)
	room.incoming <- incomingMessage{client: gm, data: advMsg}
	time.Sleep(50 * time.Millisecond)
	drainChannel(gm.send)

	advMsg2 := json.RawMessage(`{"type":"advance_scene","payload":{"scene_id":"library"}}`)
	room.incoming <- incomingMessage{client: gm, data: advMsg2}
	time.Sleep(100 * time.Millisecond)

	state := room.StateSnapshot()
	fields := state.RevealedFieldsForNPC("gm-1", "ghost_child")
	if len(fields) != 1 || fields[0] != "background" {
		t.Errorf("RevealedFields for ghost_child = %v, want [background]", fields)
	}
}

func TestAdvanceScene_OnExit_SetVar(t *testing.T) {
	repo := successEventRepo()
	room, gm, _, cancel := startedRoomFull(repo)
	defer cancel()
	defer room.Stop()

	// Go to library first (it has on_exit: set_var left_library=true).
	advMsg := json.RawMessage(`{"type":"advance_scene","payload":{"scene_id":"library"}}`)
	room.incoming <- incomingMessage{client: gm, data: advMsg}
	time.Sleep(100 * time.Millisecond)
	drainChannel(gm.send)

	// Now leave library by going to kitchen.
	advMsg2 := json.RawMessage(`{"type":"advance_scene","payload":{"scene_id":"kitchen"}}`)
	room.incoming <- incomingMessage{client: gm, data: advMsg2}
	time.Sleep(100 * time.Millisecond)

	state := room.StateSnapshot()
	if state.Variables["left_library"] != true {
		t.Errorf("left_library = %v, want true", state.Variables["left_library"])
	}
}

func TestAdvanceScene_OnEnterMultipleActions(t *testing.T) {
	repo := successEventRepo()
	room, gm, _, cancel := startedRoomFull(repo)
	defer cancel()
	defer room.Stop()

	// Go to entrance first, then library (has 2 on_enter actions: reveal_item + reveal_npc_field).
	advMsg := json.RawMessage(`{"type":"advance_scene","payload":{"scene_id":"entrance"}}`)
	room.incoming <- incomingMessage{client: gm, data: advMsg}
	time.Sleep(50 * time.Millisecond)
	drainChannel(gm.send)

	advMsg2 := json.RawMessage(`{"type":"advance_scene","payload":{"scene_id":"library"}}`)
	room.incoming <- incomingMessage{client: gm, data: advMsg2}
	time.Sleep(100 * time.Millisecond)

	state := room.StateSnapshot()
	if !state.IsItemRevealed("gm-1", "torn_diary") {
		t.Error("expected torn_diary revealed")
	}
	fields := state.RevealedFieldsForNPC("gm-1", "ghost_child")
	if len(fields) != 1 || fields[0] != "background" {
		t.Errorf("RevealedFields = %v, want [background]", fields)
	}
}

func TestPlayerChoice_OnEnter_Executed(t *testing.T) {
	repo := successEventRepo()
	room, gm, player, cancel := startedRoomFull(repo)
	defer cancel()
	defer room.Stop()

	// Go to entrance.
	advMsg := json.RawMessage(`{"type":"advance_scene","payload":{"scene_id":"entrance"}}`)
	room.incoming <- incomingMessage{client: gm, data: advMsg}
	time.Sleep(100 * time.Millisecond)
	drainChannel(gm.send)
	drainChannel(player.send)

	// Player chooses "Go to library" (index 0).
	choiceMsg := json.RawMessage(`{"type":"player_choice","payload":{"transition_index":0}}`)
	room.incoming <- incomingMessage{client: player, data: choiceMsg}
	time.Sleep(150 * time.Millisecond)

	state := room.StateSnapshot()
	// Library's on_enter reveals torn_diary to "current_player" (player-1).
	if !state.IsItemRevealed("player-1", "torn_diary") {
		t.Error("expected torn_diary to be revealed for player-1 via on_enter")
	}
	// Library's on_enter also reveals ghost_child background to player-1.
	fields := state.RevealedFieldsForNPC("player-1", "ghost_child")
	if len(fields) != 1 || fields[0] != "background" {
		t.Errorf("RevealedFields = %v, want [background]", fields)
	}
}

// --- Enhanced filter tests ---

func TestFilterScenePayload_PlayerSeesOnlyRevealedItems(t *testing.T) {
	repo := successEventRepo()
	room, gm, player, cancel := startedRoomFull(repo)
	defer cancel()
	defer room.Stop()

	// Reveal rusty_key to player-1.
	revealMsg := json.RawMessage(`{"type":"reveal_item","payload":{"item_id":"rusty_key","player_ids":["player-1"]}}`)
	room.incoming <- incomingMessage{client: gm, data: revealMsg}
	time.Sleep(50 * time.Millisecond)
	drainChannel(gm.send)
	drainChannel(player.send)

	// Advance to entrance (has items_available: ["rusty_key"]).
	advMsg := json.RawMessage(`{"type":"advance_scene","payload":{"scene_id":"entrance"}}`)
	room.incoming <- incomingMessage{client: gm, data: advMsg}
	time.Sleep(100 * time.Millisecond)

	// Player should see rusty_key in the filtered scene.
	select {
	case data := <-player.send:
		var env Envelope
		json.Unmarshal(data, &env)
		if env.Type == EventSceneChanged {
			var payload map[string]json.RawMessage
			json.Unmarshal(env.Payload, &payload)
			var scene map[string]json.RawMessage
			json.Unmarshal(payload["scene"], &scene)
			var items []string
			json.Unmarshal(scene["items_available"], &items)
			if len(items) != 1 || items[0] != "rusty_key" {
				t.Errorf("items_available = %v, want [rusty_key]", items)
			}
		}
	default:
		t.Error("expected scene_changed message")
	}
}

func TestFilterScenePayload_PlayerSeesPublicNPCFields(t *testing.T) {
	repo := successEventRepo()
	room, gm, player, cancel := startedRoomFull(repo)
	defer cancel()
	defer room.Stop()

	// Don't reveal any NPC fields. Player should still see NPCs with public fields.
	advMsg := json.RawMessage(`{"type":"advance_scene","payload":{"scene_id":"entrance"}}`)
	room.incoming <- incomingMessage{client: gm, data: advMsg}
	time.Sleep(100 * time.Millisecond)

	// Player should see butler (has public "appearance" field).
	select {
	case data := <-player.send:
		var env Envelope
		json.Unmarshal(data, &env)
		if env.Type == EventSceneChanged {
			var payload map[string]json.RawMessage
			json.Unmarshal(env.Payload, &payload)
			var scene map[string]json.RawMessage
			json.Unmarshal(payload["scene"], &scene)
			var npcs []string
			json.Unmarshal(scene["npcs_present"], &npcs)
			if len(npcs) != 1 || npcs[0] != "butler" {
				t.Errorf("npcs_present = %v, want [butler]", npcs)
			}
		}
	default:
		t.Error("expected scene_changed message")
	}
}

func TestFilterScenePayload_GMSeesAll(t *testing.T) {
	repo := successEventRepo()
	room, gm, _, cancel := startedRoomFull(repo)
	defer cancel()
	defer room.Stop()

	advMsg := json.RawMessage(`{"type":"advance_scene","payload":{"scene_id":"entrance"}}`)
	room.incoming <- incomingMessage{client: gm, data: advMsg}
	time.Sleep(100 * time.Millisecond)

	select {
	case data := <-gm.send:
		var env Envelope
		json.Unmarshal(data, &env)
		if env.Type == EventSceneChanged {
			var payload map[string]json.RawMessage
			json.Unmarshal(env.Payload, &payload)
			var scene map[string]json.RawMessage
			json.Unmarshal(payload["scene"], &scene)
			// GM should see gm_notes.
			if _, ok := scene["gm_notes"]; !ok {
				t.Error("GM should see gm_notes")
			}
			// GM should see all items.
			var items []string
			json.Unmarshal(scene["items_available"], &items)
			if len(items) != 1 || items[0] != "rusty_key" {
				t.Errorf("GM items_available = %v, want [rusty_key]", items)
			}
		}
	default:
		t.Error("expected scene_changed message")
	}
}

func TestFilterScenePayload_NoRevealedItems(t *testing.T) {
	repo := successEventRepo()
	room, gm, player, cancel := startedRoomFull(repo)
	defer cancel()
	defer room.Stop()

	// Don't reveal any items.
	advMsg := json.RawMessage(`{"type":"advance_scene","payload":{"scene_id":"entrance"}}`)
	room.incoming <- incomingMessage{client: gm, data: advMsg}
	time.Sleep(100 * time.Millisecond)

	// Player should see empty items_available.
	select {
	case data := <-player.send:
		var env Envelope
		json.Unmarshal(data, &env)
		if env.Type == EventSceneChanged {
			var payload map[string]json.RawMessage
			json.Unmarshal(env.Payload, &payload)
			var scene map[string]json.RawMessage
			json.Unmarshal(payload["scene"], &scene)
			var items []string
			json.Unmarshal(scene["items_available"], &items)
			if len(items) != 0 {
				t.Errorf("items_available = %v, want empty", items)
			}
		}
	default:
		t.Error("expected scene_changed message")
	}
}

// --- broadcastFilteredPerClient tests ---

func TestBroadcastFilteredPerClient_DifferentPayloadsPerPlayer(t *testing.T) {
	repo := successEventRepo()
	room, gm, player, cancel := startedRoomFull(repo)
	defer cancel()
	defer room.Stop()

	// Reveal rusty_key to player-1 only.
	revealMsg := json.RawMessage(`{"type":"reveal_item","payload":{"item_id":"rusty_key","player_ids":["player-1"]}}`)
	room.incoming <- incomingMessage{client: gm, data: revealMsg}
	time.Sleep(50 * time.Millisecond)
	drainChannel(gm.send)
	drainChannel(player.send)

	// Advance to entrance.
	advMsg := json.RawMessage(`{"type":"advance_scene","payload":{"scene_id":"entrance"}}`)
	room.incoming <- incomingMessage{client: gm, data: advMsg}
	time.Sleep(100 * time.Millisecond)

	// GM should see rusty_key (unfiltered).
	gmData := <-gm.send
	var gmEnv Envelope
	json.Unmarshal(gmData, &gmEnv)
	var gmPayload map[string]json.RawMessage
	json.Unmarshal(gmEnv.Payload, &gmPayload)
	var gmScene map[string]json.RawMessage
	json.Unmarshal(gmPayload["scene"], &gmScene)
	var gmItems []string
	json.Unmarshal(gmScene["items_available"], &gmItems)
	if len(gmItems) != 1 || gmItems[0] != "rusty_key" {
		t.Errorf("GM items = %v, want [rusty_key]", gmItems)
	}

	// Player should also see rusty_key (it was revealed).
	playerData := <-player.send
	var playerEnv Envelope
	json.Unmarshal(playerData, &playerEnv)
	var playerPayload map[string]json.RawMessage
	json.Unmarshal(playerEnv.Payload, &playerPayload)
	var playerScene map[string]json.RawMessage
	json.Unmarshal(playerPayload["scene"], &playerScene)
	var playerItems []string
	json.Unmarshal(playerScene["items_available"], &playerItems)
	if len(playerItems) != 1 || playerItems[0] != "rusty_key" {
		t.Errorf("Player items = %v, want [rusty_key]", playerItems)
	}

	// Player should NOT see gm_notes.
	if _, ok := playerScene["gm_notes"]; ok {
		t.Error("Player should NOT see gm_notes")
	}
}
