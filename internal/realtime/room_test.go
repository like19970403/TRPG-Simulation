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
	room := NewRoom("sess-1", "gm-1", repo, testLogger())
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
	room := NewRoom("sess-1", "gm-1", repo, testLogger())
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
	room := NewRoom("sess-1", "gm-1", repo, testLogger())
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
	room := NewRoom("sess-1", "gm-1", repo, testLogger())
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
	room := NewRoom("sess-1", "gm-1", repo, testLogger())

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

func TestRoom_BroadcastEvent_PersistError_RollsBack(t *testing.T) {
	repo := &mockEventRepo{
		appendEventFn: func(_ context.Context, _ string, _ int64, _ string, _ *string, _ json.RawMessage) (*game.GameEvent, error) {
			return nil, errors.New("db error")
		},
		listEventsSinceFn: func(_ context.Context, _ string, _ int64) ([]*game.GameEvent, error) {
			return []*game.GameEvent{}, nil
		},
	}
	room := NewRoom("sess-1", "gm-1", repo, testLogger())
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
