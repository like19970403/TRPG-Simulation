package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/like19970403/TRPG-Simulation/internal/game"
)

type mockEventRepo struct {
	appendEventFn     func(ctx context.Context, sessionID string, sequence int64, eventType string, actorID *string, payload json.RawMessage) (*game.GameEvent, error)
	listEventsSinceFn func(ctx context.Context, sessionID string, afterSeq int64) ([]*game.GameEvent, error)
	saveSnapshotFn    func(ctx context.Context, sessionID string, snapshotSeq int64, state json.RawMessage) error
	loadSnapshotFn    func(ctx context.Context, sessionID string) (int64, json.RawMessage, error)
}

func (m *mockEventRepo) AppendEvent(ctx context.Context, sessionID string, sequence int64, eventType string, actorID *string, payload json.RawMessage) (*game.GameEvent, error) {
	return m.appendEventFn(ctx, sessionID, sequence, eventType, actorID, payload)
}

func (m *mockEventRepo) ListEventsSince(ctx context.Context, sessionID string, afterSeq int64) ([]*game.GameEvent, error) {
	return m.listEventsSinceFn(ctx, sessionID, afterSeq)
}

func (m *mockEventRepo) SaveSnapshot(ctx context.Context, sessionID string, snapshotSeq int64, state json.RawMessage) error {
	if m.saveSnapshotFn != nil {
		return m.saveSnapshotFn(ctx, sessionID, snapshotSeq, state)
	}
	return nil // nil-safe default: silently succeed
}

func (m *mockEventRepo) LoadSnapshot(ctx context.Context, sessionID string) (int64, json.RawMessage, error) {
	if m.loadSnapshotFn != nil {
		return m.loadSnapshotFn(ctx, sessionID)
	}
	return 0, nil, nil // nil-safe default: no snapshot
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
		send:     make(chan []byte, sendBufferSize),
		userID:   userID,
		username: userID, // default username = userID
		role:     role,
		room:     room,
		logger:   testLogger(),
		done:     make(chan struct{}),
	}
}

// drainPlayerEvents drains any player_joined/player_left messages from a client's send channel.
func drainPlayerEvents(c *Client) {
	for {
		select {
		case msg := <-c.send:
			var env Envelope
			json.Unmarshal(msg, &env)
			if env.Type != EventPlayerJoined && env.Type != EventPlayerLeft {
				// Put it back — this is the message the test wants.
				c.send <- msg
				return
			}
		default:
			return
		}
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
	drainPlayerEvents(c)

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
	drainPlayerEvents(c1)
	drainPlayerEvents(c2)

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
	drainPlayerEvents(c)

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
	drainPlayerEvents(c)

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
	drainPlayerEvents(c)

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
	drainPlayerEvents(gm)
	drainPlayerEvents(player)

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
	drainPlayerEvents(player)

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
	drainPlayerEvents(gm)
	drainPlayerEvents(player)

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
	drainPlayerEvents(player)

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
	drainPlayerEvents(player)

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
	drainPlayerEvents(player)

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

	player := fakeClient("user-1", RolePlayer, room)
	room.register <- player
	time.Sleep(50 * time.Millisecond)
	drainPlayerEvents(player)

	seqBefore := room.StateSnapshot().LastSequence

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
	drainPlayerEvents(player)

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
	drainPlayerEvents(player)

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
	drainPlayerEvents(player)

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

// lastMessage reads the last available message from a channel, returning nil if empty.
func lastMessage(ch chan []byte) []byte {
	var last []byte
	for {
		select {
		case msg := <-ch:
			last = msg
		default:
			return last
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
	// Use a room without scenario so CurrentScene remains empty.
	room := NewRoom("sess-1", "gm-1", nil, repo, testLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go room.Run(ctx)
	defer room.Stop()

	player := fakeClient("player-1", RolePlayer, room)
	room.register <- player
	time.Sleep(50 * time.Millisecond)

	actorID := "gm-1"
	room.BroadcastEvent(EventGameStarted, &actorID, json.RawMessage(`{}`))
	time.Sleep(50 * time.Millisecond)
	drainChannel(player.send)

	// No scene has been set (no scenario → no start_scene).
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

// --- condition_met + auto transition tests ---

// testScenarioWithTransitions creates a scenario with auto, condition_met, and chain transitions.
func testScenarioWithTransitions() *ScenarioContent {
	return &ScenarioContent{
		ID:         "test-transitions",
		Title:      "Transition Tests",
		StartScene: "entrance",
		Scenes: []Scene{
			{
				ID: "entrance", Name: "Entrance", Content: "You enter.",
				Transitions: []Transition{
					{Target: "library", Trigger: "player_choice", Label: "Go to library"},
					{Target: "secret_room", Trigger: "condition_met", Condition: "has_item('rusty_key') && var('found_passage') == true", Label: "Secret"},
				},
				OnEnter: []Action{
					{SetVar: &SetVarAction{Name: "visited_entrance", Value: true}},
				},
			},
			{
				ID: "library", Name: "Library", Content: "Books.",
				Transitions: []Transition{
					{Target: "entrance", Trigger: "player_choice", Label: "Back"},
					{Target: "discovery", Trigger: "condition_met", Condition: "var('visited_entrance') == true", Label: "Discovery"},
				},
			},
			{ID: "discovery", Name: "Discovery", Content: "Found it!"},
			{ID: "secret_room", Name: "Secret Room", Content: "Dark."},
			{
				ID: "auto_start", Name: "Auto Start", Content: "Brief.",
				Transitions: []Transition{
					{Target: "auto_dest", Trigger: "auto"},
				},
			},
			{ID: "auto_dest", Name: "Auto Destination", Content: "Arrived."},
			{
				ID: "chain_a", Name: "Chain A", Content: "A.",
				Transitions: []Transition{
					{Target: "chain_b", Trigger: "auto"},
				},
			},
			{
				ID: "chain_b", Name: "Chain B", Content: "B.",
				Transitions: []Transition{
					{Target: "chain_c", Trigger: "auto"},
				},
			},
			{ID: "chain_c", Name: "Chain C", Content: "C."},
			{
				ID: "cond_false", Name: "Cond False", Content: "Nope.",
				Transitions: []Transition{
					{Target: "discovery", Trigger: "condition_met", Condition: "var('nonexistent') == true", Label: "Never"},
				},
			},
			{
				ID: "cond_invalid", Name: "Cond Invalid", Content: "Bad.",
				Transitions: []Transition{
					{Target: "discovery", Trigger: "condition_met", Condition: "((( invalid", Label: "Bad"},
				},
			},
			{
				ID: "cond_empty", Name: "Cond Empty", Content: "Empty.",
				Transitions: []Transition{
					{Target: "discovery", Trigger: "condition_met", Condition: "", Label: "Empty cond"},
				},
			},
		},
		Items: []Item{
			{ID: "rusty_key", Name: "Key", Type: "item", Description: "A key"},
		},
		Variables: []Variable{
			{Name: "visited_entrance", Type: "bool", Default: false},
			{Name: "found_passage", Type: "bool", Default: false},
		},
	}
}

func startedRoomWithTransitions(repo *mockEventRepo) (*Room, *Client, *Client, context.CancelFunc) {
	sc := testScenarioWithTransitions()
	room := NewRoom("sess-1", "gm-1", sc, repo, testLogger())
	ctx, cancel := context.WithCancel(context.Background())
	go room.Run(ctx)

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

	return room, gm, player, cancel
}

func TestAutoTransition_ImmediateTransition(t *testing.T) {
	repo := successEventRepo()
	room, gm, player, cancel := startedRoomWithTransitions(repo)
	defer cancel()
	defer room.Stop()

	// Advance to auto_start — should auto-transition to auto_dest.
	msg := json.RawMessage(`{"type":"advance_scene","payload":{"scene_id":"auto_start"}}`)
	room.incoming <- incomingMessage{client: gm, data: msg}
	time.Sleep(100 * time.Millisecond)
	drainChannel(gm.send)
	drainChannel(player.send)

	state := room.StateSnapshot()
	if state.CurrentScene != "auto_dest" {
		t.Errorf("CurrentScene = %q, want 'auto_dest'", state.CurrentScene)
	}
}

func TestAutoTransition_ChainDepth_TwoLevels(t *testing.T) {
	repo := successEventRepo()
	room, gm, player, cancel := startedRoomWithTransitions(repo)
	defer cancel()
	defer room.Stop()

	// chain_a → auto → chain_b → auto → chain_c.
	msg := json.RawMessage(`{"type":"advance_scene","payload":{"scene_id":"chain_a"}}`)
	room.incoming <- incomingMessage{client: gm, data: msg}
	time.Sleep(100 * time.Millisecond)
	drainChannel(gm.send)
	drainChannel(player.send)

	state := room.StateSnapshot()
	if state.CurrentScene != "chain_c" {
		t.Errorf("CurrentScene = %q, want 'chain_c'", state.CurrentScene)
	}
}

func TestConditionMet_TrueCondition_AutoTransitions(t *testing.T) {
	repo := successEventRepo()
	room, gm, player, cancel := startedRoomWithTransitions(repo)
	defer cancel()
	defer room.Stop()

	// Advance to entrance first (sets visited_entrance=true via on_enter).
	msg := json.RawMessage(`{"type":"advance_scene","payload":{"scene_id":"entrance"}}`)
	room.incoming <- incomingMessage{client: gm, data: msg}
	time.Sleep(100 * time.Millisecond)
	drainChannel(gm.send)
	drainChannel(player.send)

	// Now advance to library — condition_met checks var('visited_entrance')==true → should auto-transition to discovery.
	msg2 := json.RawMessage(`{"type":"advance_scene","payload":{"scene_id":"library"}}`)
	room.incoming <- incomingMessage{client: gm, data: msg2}
	time.Sleep(100 * time.Millisecond)
	drainChannel(gm.send)
	drainChannel(player.send)

	state := room.StateSnapshot()
	if state.CurrentScene != "discovery" {
		t.Errorf("CurrentScene = %q, want 'discovery'", state.CurrentScene)
	}
}

func TestConditionMet_FalseCondition_NoTransition(t *testing.T) {
	repo := successEventRepo()
	room, gm, player, cancel := startedRoomWithTransitions(repo)
	defer cancel()
	defer room.Stop()

	// Advance to cond_false — condition "var('nonexistent') == true" is false.
	msg := json.RawMessage(`{"type":"advance_scene","payload":{"scene_id":"cond_false"}}`)
	room.incoming <- incomingMessage{client: gm, data: msg}
	time.Sleep(100 * time.Millisecond)
	drainChannel(gm.send)
	drainChannel(player.send)

	state := room.StateSnapshot()
	if state.CurrentScene != "cond_false" {
		t.Errorf("CurrentScene = %q, want 'cond_false'", state.CurrentScene)
	}
}

func TestConditionMet_InvalidCondition_LogsAndContinues(t *testing.T) {
	repo := successEventRepo()
	room, gm, player, cancel := startedRoomWithTransitions(repo)
	defer cancel()
	defer room.Stop()

	// Advance to cond_invalid — condition "((( invalid" fails to compile.
	msg := json.RawMessage(`{"type":"advance_scene","payload":{"scene_id":"cond_invalid"}}`)
	room.incoming <- incomingMessage{client: gm, data: msg}
	time.Sleep(100 * time.Millisecond)
	drainChannel(gm.send)
	drainChannel(player.send)

	state := room.StateSnapshot()
	if state.CurrentScene != "cond_invalid" {
		t.Errorf("CurrentScene = %q, want 'cond_invalid'", state.CurrentScene)
	}
}

func TestConditionMet_EmptyCondition_Skipped(t *testing.T) {
	repo := successEventRepo()
	room, gm, player, cancel := startedRoomWithTransitions(repo)
	defer cancel()
	defer room.Stop()

	// Advance to cond_empty — empty condition is skipped.
	msg := json.RawMessage(`{"type":"advance_scene","payload":{"scene_id":"cond_empty"}}`)
	room.incoming <- incomingMessage{client: gm, data: msg}
	time.Sleep(100 * time.Millisecond)
	drainChannel(gm.send)
	drainChannel(player.send)

	state := room.StateSnapshot()
	if state.CurrentScene != "cond_empty" {
		t.Errorf("CurrentScene = %q, want 'cond_empty'", state.CurrentScene)
	}
}

func TestConditionMet_HasItemAndVar_True(t *testing.T) {
	repo := successEventRepo()
	room, gm, player, cancel := startedRoomWithTransitions(repo)
	defer cancel()
	defer room.Stop()

	// Reveal rusty_key to all players (including GM's triggerPlayerID for condition_met).
	revealMsg := json.RawMessage(`{"type":"reveal_item","payload":{"item_id":"rusty_key"}}`)
	room.incoming <- incomingMessage{client: gm, data: revealMsg}
	time.Sleep(50 * time.Millisecond)
	drainChannel(gm.send)
	drainChannel(player.send)

	// Set found_passage = true.
	actorID := "gm-1"
	varPayload, _ := json.Marshal(map[string]any{"name": "found_passage", "new_value": true})
	room.BroadcastEvent(EventVariableChanged, &actorID, varPayload)
	time.Sleep(50 * time.Millisecond)
	drainChannel(gm.send)
	drainChannel(player.send)

	// Advance to entrance — condition_met checks has_item('rusty_key') && var('found_passage') == true.
	// The GM is the triggerPlayerID. Since item was revealed to all (including GM's connected player IDs),
	// has_item checks against the GM user ID. We need the item revealed to the GM too.
	// Empty player_ids in reveal_item means all connected players, which includes player-1 but NOT gm-1 (GM role).
	// So we also reveal to GM explicitly.
	revealMsg2 := json.RawMessage(`{"type":"reveal_item","payload":{"item_id":"rusty_key","player_ids":["gm-1"]}}`)
	room.incoming <- incomingMessage{client: gm, data: revealMsg2}
	time.Sleep(50 * time.Millisecond)
	drainChannel(gm.send)
	drainChannel(player.send)

	msg := json.RawMessage(`{"type":"advance_scene","payload":{"scene_id":"entrance"}}`)
	room.incoming <- incomingMessage{client: gm, data: msg}
	time.Sleep(100 * time.Millisecond)
	drainChannel(gm.send)
	drainChannel(player.send)

	state := room.StateSnapshot()
	if state.CurrentScene != "secret_room" {
		t.Errorf("CurrentScene = %q, want 'secret_room'", state.CurrentScene)
	}
}

func TestConditionMet_HasItemAndVar_False(t *testing.T) {
	repo := successEventRepo()
	room, gm, player, cancel := startedRoomWithTransitions(repo)
	defer cancel()
	defer room.Stop()

	// Don't reveal key or set passage. Advance to entrance — condition_met should fail, stay on entrance.
	msg := json.RawMessage(`{"type":"advance_scene","payload":{"scene_id":"entrance"}}`)
	room.incoming <- incomingMessage{client: gm, data: msg}
	time.Sleep(100 * time.Millisecond)
	drainChannel(gm.send)
	drainChannel(player.send)

	state := room.StateSnapshot()
	if state.CurrentScene != "entrance" {
		t.Errorf("CurrentScene = %q, want 'entrance'", state.CurrentScene)
	}
}

func TestConditionMet_AfterPlayerChoice(t *testing.T) {
	repo := successEventRepo()
	room, gm, player, cancel := startedRoomWithTransitions(repo)
	defer cancel()
	defer room.Stop()

	// First go to entrance (sets visited_entrance=true).
	msg := json.RawMessage(`{"type":"advance_scene","payload":{"scene_id":"entrance"}}`)
	room.incoming <- incomingMessage{client: gm, data: msg}
	time.Sleep(100 * time.Millisecond)
	drainChannel(gm.send)
	drainChannel(player.send)

	// Player chooses "Go to library" (index 0) — library has condition_met(visited_entrance==true) → discovery.
	choiceMsg := json.RawMessage(`{"type":"player_choice","payload":{"transition_index":0}}`)
	room.incoming <- incomingMessage{client: player, data: choiceMsg}
	time.Sleep(100 * time.Millisecond)
	drainChannel(gm.send)
	drainChannel(player.send)

	state := room.StateSnapshot()
	if state.CurrentScene != "discovery" {
		t.Errorf("CurrentScene = %q, want 'discovery'", state.CurrentScene)
	}
}

func TestConditionMet_OnEnterActionsTriggerTransition(t *testing.T) {
	// entrance on_enter sets visited_entrance=true; entrance has condition_met on has_item && found_passage.
	// Without the item, condition_met stays on entrance even though on_enter ran.
	repo := successEventRepo()
	room, gm, player, cancel := startedRoomWithTransitions(repo)
	defer cancel()
	defer room.Stop()

	msg := json.RawMessage(`{"type":"advance_scene","payload":{"scene_id":"entrance"}}`)
	room.incoming <- incomingMessage{client: gm, data: msg}
	time.Sleep(100 * time.Millisecond)
	drainChannel(gm.send)
	drainChannel(player.send)

	state := room.StateSnapshot()
	// on_enter set visited_entrance=true, but condition_met requires has_item('rusty_key') too.
	if state.CurrentScene != "entrance" {
		t.Errorf("CurrentScene = %q, want 'entrance'", state.CurrentScene)
	}
	if state.Variables["visited_entrance"] != true {
		t.Error("expected visited_entrance to be true after on_enter")
	}
}

func TestAutoTransition_OnEnterActionsExecuted(t *testing.T) {
	// auto_start → auto → auto_dest; verify the scene_changed events happen.
	repo := successEventRepo()
	room, gm, _, cancel := startedRoomWithTransitions(repo)
	defer cancel()
	defer room.Stop()

	msg := json.RawMessage(`{"type":"advance_scene","payload":{"scene_id":"auto_start"}}`)
	room.incoming <- incomingMessage{client: gm, data: msg}
	time.Sleep(100 * time.Millisecond)

	// GM should receive at least 2 scene_changed events (auto_start + auto_dest).
	count := 0
	for {
		select {
		case data := <-gm.send:
			var env Envelope
			json.Unmarshal(data, &env)
			if env.Type == EventSceneChanged {
				count++
			}
		default:
			goto done
		}
	}
done:
	if count < 2 {
		t.Errorf("scene_changed count = %d, want >= 2", count)
	}
}

func TestConditionMet_MultipleConditions_FirstMatchWins(t *testing.T) {
	// Library has two transitions: player_choice (index 0) and condition_met (index 1).
	// When condition is true, the first matching condition_met wins.
	repo := successEventRepo()
	room, gm, player, cancel := startedRoomWithTransitions(repo)
	defer cancel()
	defer room.Stop()

	// Set visited_entrance=true directly.
	actorID := "gm-1"
	varPayload, _ := json.Marshal(map[string]any{"name": "visited_entrance", "new_value": true})
	room.BroadcastEvent(EventVariableChanged, &actorID, varPayload)
	time.Sleep(50 * time.Millisecond)
	drainChannel(gm.send)
	drainChannel(player.send)

	// Advance to library — condition_met should trigger to discovery.
	msg := json.RawMessage(`{"type":"advance_scene","payload":{"scene_id":"library"}}`)
	room.incoming <- incomingMessage{client: gm, data: msg}
	time.Sleep(100 * time.Millisecond)
	drainChannel(gm.send)
	drainChannel(player.send)

	state := room.StateSnapshot()
	if state.CurrentScene != "discovery" {
		t.Errorf("CurrentScene = %q, want 'discovery'", state.CurrentScene)
	}
}

// --- Snapshot tests (ADR-004) ---

func TestSnapshot_SavedAtInterval(t *testing.T) {
	var mu sync.Mutex
	var savedSnapshots []int64
	repo := successEventRepo()
	repo.saveSnapshotFn = func(_ context.Context, _ string, snapshotSeq int64, _ json.RawMessage) error {
		mu.Lock()
		savedSnapshots = append(savedSnapshots, snapshotSeq)
		mu.Unlock()
		return nil
	}

	room := NewRoom("sess-snap", "gm-1", nil, repo, testLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go room.Run(ctx)
	defer room.Stop()

	// Start game.
	startPayload, _ := json.Marshal(map[string]string{"status": "active"})
	room.BroadcastEvent(EventGameStarted, nil, startPayload)
	time.Sleep(50 * time.Millisecond)

	// Fire events up to exactly snapshotInterval (50).
	// We already have seq=1 from game_started. Need 49 more to reach seq=50.
	for i := int64(0); i < snapshotInterval-1; i++ {
		payload, _ := json.Marshal(map[string]any{
			"roller_id": "p1",
			"formula":   "1d6",
			"results":   []int{3},
			"modifier":  0,
			"total":     3,
			"purpose":   "test",
		})
		room.BroadcastEvent(EventDiceRolled, nil, payload)
	}
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(savedSnapshots) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(savedSnapshots))
	}
	if savedSnapshots[0] != snapshotInterval {
		t.Errorf("snapshot seq = %d, want %d", savedSnapshots[0], snapshotInterval)
	}
}

func TestSnapshot_NotSavedBeforeInterval(t *testing.T) {
	var snapshotCount atomic.Int32
	repo := successEventRepo()
	repo.saveSnapshotFn = func(_ context.Context, _ string, _ int64, _ json.RawMessage) error {
		snapshotCount.Add(1)
		return nil
	}

	room := NewRoom("sess-snap2", "gm-1", nil, repo, testLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go room.Run(ctx)
	defer room.Stop()

	// Fire 10 events (well below snapshot interval of 50).
	for i := 0; i < 10; i++ {
		payload, _ := json.Marshal(map[string]any{
			"roller_id": "p1",
			"formula":   "1d6",
			"results":   []int{3},
			"modifier":  0,
			"total":     3,
			"purpose":   "test",
		})
		room.BroadcastEvent(EventDiceRolled, nil, payload)
	}
	time.Sleep(100 * time.Millisecond)

	if snapshotCount.Load() != 0 {
		t.Errorf("expected 0 snapshots, got %d", snapshotCount.Load())
	}
}

func TestSnapshot_SaveErrorDoesNotCrash(t *testing.T) {
	repo := successEventRepo()
	repo.saveSnapshotFn = func(_ context.Context, _ string, _ int64, _ json.RawMessage) error {
		return errors.New("snapshot write failed")
	}

	room := NewRoom("sess-snap3", "gm-1", nil, repo, testLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go room.Run(ctx)
	defer room.Stop()

	// Fire exactly snapshotInterval events — snapshot will fail but room should not crash.
	for i := int64(0); i < snapshotInterval; i++ {
		payload, _ := json.Marshal(map[string]any{
			"roller_id": "p1",
			"formula":   "1d6",
			"results":   []int{3},
			"modifier":  0,
			"total":     3,
			"purpose":   "test",
		})
		room.BroadcastEvent(EventDiceRolled, nil, payload)
	}
	time.Sleep(200 * time.Millisecond)

	// Room should still be functional.
	state := room.StateSnapshot()
	if state.LastSequence != snapshotInterval {
		t.Errorf("LastSequence = %d, want %d", state.LastSequence, snapshotInterval)
	}
}

func TestSnapshot_ContainsValidGameState(t *testing.T) {
	var mu sync.Mutex
	var capturedState json.RawMessage
	repo := successEventRepo()
	repo.saveSnapshotFn = func(_ context.Context, _ string, _ int64, state json.RawMessage) error {
		mu.Lock()
		capturedState = make(json.RawMessage, len(state))
		copy(capturedState, state)
		mu.Unlock()
		return nil
	}

	room := NewRoom("sess-snap4", "gm-1", nil, repo, testLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go room.Run(ctx)
	defer room.Stop()

	// Fire exactly snapshotInterval events.
	for i := int64(0); i < snapshotInterval; i++ {
		payload, _ := json.Marshal(map[string]any{
			"roller_id": "p1",
			"formula":   "1d6",
			"results":   []int{3},
			"modifier":  0,
			"total":     3,
			"purpose":   "test",
		})
		room.BroadcastEvent(EventDiceRolled, nil, payload)
	}
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if capturedState == nil {
		t.Fatal("snapshot state was not captured")
	}

	// Verify the snapshot deserializes to a valid GameState.
	var gs GameState
	if err := json.Unmarshal(capturedState, &gs); err != nil {
		t.Fatalf("failed to unmarshal snapshot: %v", err)
	}
	if gs.SessionID != "sess-snap4" {
		t.Errorf("SessionID = %q, want 'sess-snap4'", gs.SessionID)
	}
	if gs.LastSequence != snapshotInterval {
		t.Errorf("LastSequence = %d, want %d", gs.LastSequence, snapshotInterval)
	}
}

func TestRecoverFromSnapshot_WithSnapshot(t *testing.T) {
	// Build a snapshot at seq=50 with a known state.
	snapshotState := &GameState{
		SessionID:    "sess-recover",
		Status:       "active",
		CurrentScene: "library",
		LastSequence: 50,
		Variables:    map[string]any{"anger": float64(3)},
	}
	snapshotData, _ := json.Marshal(snapshotState)

	// Events after snapshot: seq 51 = variable_changed (anger→5), seq 52 = scene_changed.
	event51Payload, _ := json.Marshal(map[string]any{"name": "anger", "new_value": float64(5)})
	event52Payload, _ := json.Marshal(map[string]any{"scene_id": "dungeon"})

	repo := successEventRepo()
	repo.loadSnapshotFn = func(_ context.Context, _ string) (int64, json.RawMessage, error) {
		return 50, snapshotData, nil
	}
	repo.listEventsSinceFn = func(_ context.Context, _ string, afterSeq int64) ([]*game.GameEvent, error) {
		if afterSeq != 50 {
			t.Errorf("ListEventsSince afterSeq = %d, want 50", afterSeq)
		}
		return []*game.GameEvent{
			{ID: "e51", SessionID: "sess-recover", Sequence: 51, Type: EventVariableChanged, Payload: event51Payload},
			{ID: "e52", SessionID: "sess-recover", Sequence: 52, Type: EventSceneChanged, Payload: event52Payload},
		}, nil
	}

	room := NewRoom("sess-recover", "gm-1", nil, repo, testLogger())

	if err := room.RecoverFromSnapshot(context.Background()); err != nil {
		t.Fatalf("RecoverFromSnapshot failed: %v", err)
	}

	state := room.state
	if state.LastSequence != 52 {
		t.Errorf("LastSequence = %d, want 52", state.LastSequence)
	}
	if state.CurrentScene != "dungeon" {
		t.Errorf("CurrentScene = %q, want 'dungeon'", state.CurrentScene)
	}
	if state.Variables["anger"] != float64(5) {
		t.Errorf("anger = %v, want 5", state.Variables["anger"])
	}
}

func TestRecoverFromSnapshot_NoSnapshot(t *testing.T) {
	// No snapshot — should replay all events from seq 0.
	event1Payload, _ := json.Marshal(map[string]string{"status": "active"})
	event2Payload, _ := json.Marshal(map[string]any{"scene_id": "intro"})

	repo := successEventRepo()
	repo.loadSnapshotFn = func(_ context.Context, _ string) (int64, json.RawMessage, error) {
		return 0, nil, nil
	}
	repo.listEventsSinceFn = func(_ context.Context, _ string, afterSeq int64) ([]*game.GameEvent, error) {
		if afterSeq != 0 {
			t.Errorf("ListEventsSince afterSeq = %d, want 0", afterSeq)
		}
		return []*game.GameEvent{
			{ID: "e1", SessionID: "sess-nsnap", Sequence: 1, Type: EventGameStarted, Payload: event1Payload},
			{ID: "e2", SessionID: "sess-nsnap", Sequence: 2, Type: EventSceneChanged, Payload: event2Payload},
		}, nil
	}

	room := NewRoom("sess-nsnap", "gm-1", nil, repo, testLogger())

	if err := room.RecoverFromSnapshot(context.Background()); err != nil {
		t.Fatalf("RecoverFromSnapshot failed: %v", err)
	}

	state := room.state
	if state.LastSequence != 2 {
		t.Errorf("LastSequence = %d, want 2", state.LastSequence)
	}
	if state.CurrentScene != "intro" {
		t.Errorf("CurrentScene = %q, want 'intro'", state.CurrentScene)
	}
}

func TestRecoverFromSnapshot_LoadError(t *testing.T) {
	repo := successEventRepo()
	repo.loadSnapshotFn = func(_ context.Context, _ string) (int64, json.RawMessage, error) {
		return 0, nil, errors.New("db connection failed")
	}

	room := NewRoom("sess-err", "gm-1", nil, repo, testLogger())

	err := room.RecoverFromSnapshot(context.Background())
	if err == nil {
		t.Fatal("expected error from RecoverFromSnapshot")
	}
}

func TestRecoverFromSnapshot_ReplayError(t *testing.T) {
	repo := successEventRepo()
	repo.loadSnapshotFn = func(_ context.Context, _ string) (int64, json.RawMessage, error) {
		return 0, nil, nil
	}
	repo.listEventsSinceFn = func(_ context.Context, _ string, _ int64) ([]*game.GameEvent, error) {
		return nil, errors.New("list events failed")
	}

	room := NewRoom("sess-rerr", "gm-1", nil, repo, testLogger())

	err := room.RecoverFromSnapshot(context.Background())
	if err == nil {
		t.Fatal("expected error from RecoverFromSnapshot")
	}
}

func TestSnapshot_SavedFromActionsAtInterval(t *testing.T) {
	// Verify that snapshots are also triggered from executeAndPersistActions.
	var mu sync.Mutex
	var snapshotSeqs []int64
	repo := successEventRepo()
	repo.saveSnapshotFn = func(_ context.Context, _ string, seq int64, _ json.RawMessage) error {
		mu.Lock()
		snapshotSeqs = append(snapshotSeqs, seq)
		mu.Unlock()
		return nil
	}

	// Create a scenario with on_enter actions that produce events.
	scenario := &ScenarioContent{
		Scenes: []Scene{
			{
				ID: "start",
				OnEnter: []Action{
					{SetVar: &SetVarAction{Name: "visited", Value: true}},
				},
			},
		},
	}

	room := NewRoom("sess-asnap", "gm-1", scenario, repo, testLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go room.Run(ctx)
	defer room.Stop()

	gm := fakeClient("gm-1", RoleGM, room)
	room.Register(gm)
	time.Sleep(50 * time.Millisecond)

	// Start game.
	startPayload, _ := json.Marshal(map[string]string{"status": "active"})
	room.BroadcastEvent(EventGameStarted, nil, startPayload)
	time.Sleep(50 * time.Millisecond)

	// Bring sequence close to snapshotInterval by firing events.
	// After game_started seq=1. We need seq to reach 50.
	// advance_scene will create: scene_changed event + on_enter set_var event = 2 events.
	// So fire 47 dice rolls (seq 2..48), then advance_scene creates seq 49 (scene_changed) + 50 (set_var).
	for i := 0; i < 47; i++ {
		payload, _ := json.Marshal(map[string]any{
			"roller_id": "p1",
			"formula":   "1d6",
			"results":   []int{3},
			"modifier":  0,
			"total":     3,
			"purpose":   "test",
		})
		room.BroadcastEvent(EventDiceRolled, nil, payload)
	}
	time.Sleep(200 * time.Millisecond)

	// Now advance scene — this should produce seq 49 (scene_changed) and seq 50 (set_var action).
	msg := json.RawMessage(`{"type":"advance_scene","payload":{"scene_id":"start"}}`)
	room.incoming <- incomingMessage{client: gm, data: msg}
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(snapshotSeqs) != 1 {
		t.Fatalf("expected 1 snapshot from action, got %d (seqs: %v)", len(snapshotSeqs), snapshotSeqs)
	}
	if snapshotSeqs[0] != snapshotInterval {
		t.Errorf("snapshot seq = %d, want %d", snapshotSeqs[0], snapshotInterval)
	}
}

func TestRecoverFromSnapshot_SkipsStaleEvents(t *testing.T) {
	// Snapshot at seq=10. Replay should work correctly.
	snapshotState := &GameState{
		SessionID:    "sess-stale",
		Status:       "active",
		CurrentScene: "intro",
		LastSequence: 10,
	}
	snapshotData, _ := json.Marshal(snapshotState)

	event11Payload, _ := json.Marshal(map[string]any{"scene_id": "forest"})

	repo := successEventRepo()
	repo.loadSnapshotFn = func(_ context.Context, _ string) (int64, json.RawMessage, error) {
		return 10, snapshotData, nil
	}
	repo.listEventsSinceFn = func(_ context.Context, _ string, _ int64) ([]*game.GameEvent, error) {
		return []*game.GameEvent{
			{ID: "e11", SessionID: "sess-stale", Sequence: 11, Type: EventSceneChanged, Payload: event11Payload},
		}, nil
	}

	room := NewRoom("sess-stale", "gm-1", nil, repo, testLogger())

	if err := room.RecoverFromSnapshot(context.Background()); err != nil {
		t.Fatalf("RecoverFromSnapshot failed: %v", err)
	}

	if room.state.LastSequence != 11 {
		t.Errorf("LastSequence = %d, want 11", room.state.LastSequence)
	}
	if room.state.CurrentScene != "forest" {
		t.Errorf("CurrentScene = %q, want 'forest'", room.state.CurrentScene)
	}
}

// --- GM Broadcast tests ---

func TestGMBroadcast_ContentToAll(t *testing.T) {
	repo := successEventRepo()
	room, cancel := startedRoom(repo, nil)
	defer cancel()
	defer room.Stop()

	gm := fakeClient("gm-1", RoleGM, room)
	player1 := fakeClient("player-1", RolePlayer, room)
	player2 := fakeClient("player-2", RolePlayer, room)
	room.Register(gm)
	room.Register(player1)
	room.Register(player2)
	time.Sleep(50 * time.Millisecond)
	drainChannel(gm.send)
	drainChannel(player1.send)
	drainChannel(player2.send)

	// GM broadcasts content to all.
	msg := json.RawMessage(`{"type":"gm_broadcast","payload":{"content":"A storm is approaching!"}}`)
	room.incoming <- incomingMessage{client: gm, data: msg}
	time.Sleep(100 * time.Millisecond)

	// All should receive the broadcast.
	gmMsg := lastMessage(gm.send)
	p1Msg := lastMessage(player1.send)
	p2Msg := lastMessage(player2.send)

	if gmMsg == nil {
		t.Fatal("GM did not receive broadcast")
	}
	if p1Msg == nil {
		t.Fatal("player-1 did not receive broadcast")
	}
	if p2Msg == nil {
		t.Fatal("player-2 did not receive broadcast")
	}

	var env Envelope
	json.Unmarshal(gmMsg, &env)
	if env.Type != EventGMBroadcast {
		t.Errorf("type = %q, want %q", env.Type, EventGMBroadcast)
	}
}

func TestGMBroadcast_ContentToSpecificPlayer(t *testing.T) {
	repo := successEventRepo()
	room, cancel := startedRoom(repo, nil)
	defer cancel()
	defer room.Stop()

	gm := fakeClient("gm-1", RoleGM, room)
	player1 := fakeClient("player-1", RolePlayer, room)
	player2 := fakeClient("player-2", RolePlayer, room)
	room.Register(gm)
	room.Register(player1)
	room.Register(player2)
	time.Sleep(50 * time.Millisecond)
	drainChannel(gm.send)
	drainChannel(player1.send)
	drainChannel(player2.send)

	// GM broadcasts only to player-1.
	msg := json.RawMessage(`{"type":"gm_broadcast","payload":{"content":"You hear a whisper...","player_ids":["player-1"]}}`)
	room.incoming <- incomingMessage{client: gm, data: msg}
	time.Sleep(100 * time.Millisecond)

	// GM and player-1 should receive; player-2 should NOT.
	gmMsg := lastMessage(gm.send)
	p1Msg := lastMessage(player1.send)
	p2Msg := lastMessage(player2.send)

	if gmMsg == nil {
		t.Fatal("GM did not receive own broadcast")
	}
	if p1Msg == nil {
		t.Fatal("player-1 did not receive targeted broadcast")
	}
	if p2Msg != nil {
		t.Error("player-2 should NOT have received the broadcast")
	}
}

func TestGMBroadcast_ImageURL(t *testing.T) {
	repo := successEventRepo()
	room, cancel := startedRoom(repo, nil)
	defer cancel()
	defer room.Stop()

	gm := fakeClient("gm-1", RoleGM, room)
	room.Register(gm)
	time.Sleep(50 * time.Millisecond)
	drainChannel(gm.send)

	msg := json.RawMessage(`{"type":"gm_broadcast","payload":{"image_url":"https://example.com/map.png"}}`)
	room.incoming <- incomingMessage{client: gm, data: msg}
	time.Sleep(100 * time.Millisecond)

	gmMsg := lastMessage(gm.send)
	if gmMsg == nil {
		t.Fatal("GM did not receive broadcast")
	}

	var env Envelope
	json.Unmarshal(gmMsg, &env)
	var m map[string]any
	json.Unmarshal(env.Payload, &m)
	if m["image_url"] != "https://example.com/map.png" {
		t.Errorf("image_url = %v, want 'https://example.com/map.png'", m["image_url"])
	}
}

func TestGMBroadcast_MissingContentAndImage(t *testing.T) {
	repo := successEventRepo()
	room, cancel := startedRoom(repo, nil)
	defer cancel()
	defer room.Stop()

	gm := fakeClient("gm-1", RoleGM, room)
	room.Register(gm)
	time.Sleep(50 * time.Millisecond)
	drainChannel(gm.send)

	// Neither content nor image_url → error.
	msg := json.RawMessage(`{"type":"gm_broadcast","payload":{}}`)
	room.incoming <- incomingMessage{client: gm, data: msg}
	time.Sleep(100 * time.Millisecond)

	gmMsg := lastMessage(gm.send)
	if gmMsg == nil {
		t.Fatal("expected error response")
	}
	var env Envelope
	json.Unmarshal(gmMsg, &env)
	if env.Type != EventError {
		t.Errorf("type = %q, want %q", env.Type, EventError)
	}
}

func TestGMBroadcast_PlayerRejected(t *testing.T) {
	repo := successEventRepo()
	room, cancel := startedRoom(repo, nil)
	defer cancel()
	defer room.Stop()

	player := fakeClient("player-1", RolePlayer, room)
	room.Register(player)
	time.Sleep(50 * time.Millisecond)
	drainChannel(player.send)

	// Player tries to broadcast → should be rejected.
	msg := json.RawMessage(`{"type":"gm_broadcast","payload":{"content":"I am a player"}}`)
	room.incoming <- incomingMessage{client: player, data: msg}
	time.Sleep(100 * time.Millisecond)

	pMsg := lastMessage(player.send)
	if pMsg == nil {
		t.Fatal("expected error response")
	}
	var env Envelope
	json.Unmarshal(pMsg, &env)
	if env.Type != EventError {
		t.Errorf("type = %q, want %q", env.Type, EventError)
	}
}

func TestGMBroadcast_GameNotActive(t *testing.T) {
	repo := successEventRepo()
	room := NewRoom("sess-gm-na", "gm-1", nil, repo, testLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go room.Run(ctx)
	defer room.Stop()

	// Do NOT start the game — status remains "active" from NewGameState default.
	// Instead, end the game to make it non-active.
	actorID := "gm-1"
	startPayload, _ := json.Marshal(map[string]string{"status": "active"})
	room.BroadcastEvent(EventGameStarted, &actorID, startPayload)
	time.Sleep(50 * time.Millisecond)
	endPayload, _ := json.Marshal(map[string]string{"status": "completed"})
	room.BroadcastEvent(EventGameEnded, &actorID, endPayload)
	time.Sleep(50 * time.Millisecond)

	gm := fakeClient("gm-1", RoleGM, room)
	room.Register(gm)
	time.Sleep(50 * time.Millisecond)
	drainChannel(gm.send)

	msg := json.RawMessage(`{"type":"gm_broadcast","payload":{"content":"Hello"}}`)
	room.incoming <- incomingMessage{client: gm, data: msg}
	time.Sleep(100 * time.Millisecond)

	gmMsg := lastMessage(gm.send)
	if gmMsg == nil {
		t.Fatal("expected error response")
	}
	var env Envelope
	json.Unmarshal(gmMsg, &env)
	if env.Type != EventError {
		t.Errorf("type = %q, want %q", env.Type, EventError)
	}
}

func TestGMBroadcast_PersistError(t *testing.T) {
	callCount := 0
	repo := &mockEventRepo{
		appendEventFn: func(_ context.Context, sessionID string, sequence int64, eventType string, actorID *string, payload json.RawMessage) (*game.GameEvent, error) {
			callCount++
			if callCount == 1 {
				// Allow game_started to succeed.
				return &game.GameEvent{
					ID: "evt-1", SessionID: sessionID, Sequence: sequence,
					Type: eventType, ActorID: actorID, Payload: payload,
				}, nil
			}
			// Fail gm_broadcast persist.
			return nil, errors.New("db write failed")
		},
		listEventsSinceFn: func(_ context.Context, _ string, _ int64) ([]*game.GameEvent, error) {
			return []*game.GameEvent{}, nil
		},
	}

	room := NewRoom("sess-gm-perr", "gm-1", nil, repo, testLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go room.Run(ctx)
	defer room.Stop()

	actorID := "gm-1"
	startPayload, _ := json.Marshal(map[string]string{"status": "active"})
	room.BroadcastEvent(EventGameStarted, &actorID, startPayload)
	time.Sleep(50 * time.Millisecond)

	gm := fakeClient("gm-1", RoleGM, room)
	room.Register(gm)
	time.Sleep(50 * time.Millisecond)
	drainChannel(gm.send)

	msg := json.RawMessage(`{"type":"gm_broadcast","payload":{"content":"Hello"}}`)
	room.incoming <- incomingMessage{client: gm, data: msg}
	time.Sleep(100 * time.Millisecond)

	gmMsg := lastMessage(gm.send)
	if gmMsg == nil {
		t.Fatal("expected error response")
	}
	var env Envelope
	json.Unmarshal(gmMsg, &env)
	if env.Type != EventError {
		t.Errorf("type = %q, want %q", env.Type, EventError)
	}
}

func TestGMBroadcast_ContentAndImage(t *testing.T) {
	repo := successEventRepo()
	room, cancel := startedRoom(repo, nil)
	defer cancel()
	defer room.Stop()

	gm := fakeClient("gm-1", RoleGM, room)
	room.Register(gm)
	time.Sleep(50 * time.Millisecond)
	drainChannel(gm.send)

	// Both content and image_url provided.
	msg := json.RawMessage(`{"type":"gm_broadcast","payload":{"content":"Look at this map!","image_url":"https://example.com/map.png"}}`)
	room.incoming <- incomingMessage{client: gm, data: msg}
	time.Sleep(100 * time.Millisecond)

	gmMsg := lastMessage(gm.send)
	if gmMsg == nil {
		t.Fatal("GM did not receive broadcast")
	}

	var env Envelope
	json.Unmarshal(gmMsg, &env)
	var m map[string]any
	json.Unmarshal(env.Payload, &m)
	if m["content"] != "Look at this map!" {
		t.Errorf("content = %v, want 'Look at this map!'", m["content"])
	}
	if m["image_url"] != "https://example.com/map.png" {
		t.Errorf("image_url = %v, want 'https://example.com/map.png'", m["image_url"])
	}
}

func TestGMBroadcast_EventPersisted(t *testing.T) {
	var mu sync.Mutex
	var persistedType string
	repo := successEventRepo()
	origAppend := repo.appendEventFn
	repo.appendEventFn = func(ctx context.Context, sessionID string, sequence int64, eventType string, actorID *string, payload json.RawMessage) (*game.GameEvent, error) {
		if eventType == EventGMBroadcast {
			mu.Lock()
			persistedType = eventType
			mu.Unlock()
		}
		return origAppend(ctx, sessionID, sequence, eventType, actorID, payload)
	}

	room, cancel := startedRoom(repo, nil)
	defer cancel()
	defer room.Stop()

	gm := fakeClient("gm-1", RoleGM, room)
	room.Register(gm)
	time.Sleep(50 * time.Millisecond)
	drainChannel(gm.send)

	msg := json.RawMessage(`{"type":"gm_broadcast","payload":{"content":"Test persist"}}`)
	room.incoming <- incomingMessage{client: gm, data: msg}
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if persistedType != EventGMBroadcast {
		t.Errorf("persistedType = %q, want %q", persistedType, EventGMBroadcast)
	}
}

func TestGMBroadcast_SequenceAdvances(t *testing.T) {
	repo := successEventRepo()
	room, cancel := startedRoom(repo, nil)
	defer cancel()
	defer room.Stop()

	gm := fakeClient("gm-1", RoleGM, room)
	room.Register(gm)
	time.Sleep(50 * time.Millisecond)
	drainChannel(gm.send)

	seqBefore := room.StateSnapshot().LastSequence

	msg := json.RawMessage(`{"type":"gm_broadcast","payload":{"content":"Test"}}`)
	room.incoming <- incomingMessage{client: gm, data: msg}
	time.Sleep(100 * time.Millisecond)

	seqAfter := room.StateSnapshot().LastSequence
	if seqAfter != seqBefore+1 {
		t.Errorf("LastSequence = %d, want %d", seqAfter, seqBefore+1)
	}
}
