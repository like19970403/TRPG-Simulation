package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/like19970403/TRPG-Simulation/internal/game"
)

// EventRepository is a consumer-side interface for event persistence.
type EventRepository interface {
	AppendEvent(ctx context.Context, sessionID string, sequence int64, eventType string, actorID *string, payload json.RawMessage) (*game.GameEvent, error)
	ListEventsSince(ctx context.Context, sessionID string, afterSeq int64) ([]*game.GameEvent, error)
}

// incomingMessage wraps a raw message with sender info for the Room goroutine.
type incomingMessage struct {
	client *Client
	data   []byte
}

// eventRequest is sent to the Room goroutine for serialized event processing.
type eventRequest struct {
	eventType string
	actorID   *string
	payload   json.RawMessage
}

// Room manages one active game session's real-time communications.
// One goroutine per Room; all state access is serialized via channels.
type Room struct {
	sessionID    string
	gmID         string
	clients      map[*Client]bool
	incoming     chan incomingMessage
	register     chan *Client
	unregister   chan *Client
	processEvent     chan eventRequest
	queryClientCount chan chan int
	queryState       chan chan GameState
	stop             chan struct{}
	stopOnce     sync.Once
	stopped      chan struct{}
	state        *GameState
	eventRepo    EventRepository
	logger       *slog.Logger
}

// NewRoom creates a Room. Call Run() as a goroutine to start the event loop.
func NewRoom(sessionID, gmID string, eventRepo EventRepository, logger *slog.Logger) *Room {
	return &Room{
		sessionID:    sessionID,
		gmID:         gmID,
		clients:      make(map[*Client]bool),
		incoming:     make(chan incomingMessage, 64),
		register:     make(chan *Client, 16),
		unregister:   make(chan *Client, 16),
		processEvent:     make(chan eventRequest, 16),
		queryClientCount: make(chan chan int),
		queryState:       make(chan chan GameState),
		stop:             make(chan struct{}),
		stopped:      make(chan struct{}),
		state:        NewGameState(sessionID),
		eventRepo:    eventRepo,
		logger:       logger,
	}
}

// Run starts the Room's event loop. Blocks until Stop() is called or ctx is cancelled.
func (r *Room) Run(ctx context.Context) {
	defer close(r.stopped)
	for {
		select {
		case client := <-r.register:
			r.clients[client] = true
			r.logger.Info("client registered", "session", r.sessionID, "user", client.userID, "role", client.role)

		case client := <-r.unregister:
			if _, ok := r.clients[client]; ok {
				delete(r.clients, client)
				close(client.send)
				r.logger.Info("client unregistered", "session", r.sessionID, "user", client.userID)
			}

		case msg := <-r.incoming:
			// In SPEC-005, incoming client messages are not processed.
			// Future SPECs will handle scene_change, dice_roll, etc.
			r.logger.Debug("incoming message (not yet handled)", "session", r.sessionID, "user", msg.client.userID)

		case req := <-r.processEvent:
			r.handleEvent(ctx, req)

		case ch := <-r.queryClientCount:
			ch <- len(r.clients)

		case ch := <-r.queryState:
			ch <- *r.state

		case <-r.stop:
			r.closeAllClients()
			return

		case <-ctx.Done():
			r.closeAllClients()
			return
		}
	}
}

func (r *Room) handleEvent(ctx context.Context, req eventRequest) {
	r.state.LastSequence++
	seq := r.state.LastSequence

	// Persist to DB.
	_, err := r.eventRepo.AppendEvent(ctx, r.sessionID, seq, req.eventType, req.actorID, req.payload)
	if err != nil {
		r.logger.Error("failed to persist event", "error", err, "session", r.sessionID, "type", req.eventType)
		r.state.LastSequence-- // rollback sequence
		return
	}

	// Apply to in-memory state.
	if err := r.state.Apply(req.eventType, seq, req.payload); err != nil {
		r.logger.Error("failed to apply event", "error", err, "session", r.sessionID, "type", req.eventType)
	}

	// Create envelope and broadcast.
	env := NewEnvelope(req.eventType, r.sessionID, "", req.payload)
	if req.actorID != nil {
		env.SenderID = *req.actorID
	}
	data, _ := json.Marshal(env)

	for client := range r.clients {
		select {
		case client.send <- data:
		default:
			// Client send buffer full; disconnect.
			close(client.send)
			delete(r.clients, client)
			r.logger.Warn("client dropped (buffer full)", "session", r.sessionID, "user", client.userID)
		}
	}
}

func (r *Room) closeAllClients() {
	for client := range r.clients {
		close(client.send)
		delete(r.clients, client)
	}
}

// Stop signals the Room to shut down and disconnect all clients.
// Blocks until the Room goroutine has fully exited.
func (r *Room) Stop() {
	r.stopOnce.Do(func() {
		close(r.stop)
	})
	<-r.stopped
}

// BroadcastEvent sends an event to the Room goroutine for serialized processing.
// Safe to call from any goroutine.
func (r *Room) BroadcastEvent(eventType string, actorID *string, payload json.RawMessage) {
	select {
	case r.processEvent <- eventRequest{
		eventType: eventType,
		actorID:   actorID,
		payload:   payload,
	}:
	case <-r.stopped:
		// Room already stopped; discard.
	}
}

// ReplayEvents sends all events since lastSeq to a specific client.
func (r *Room) ReplayEvents(ctx context.Context, c *Client, lastSeq int64) error {
	events, err := r.eventRepo.ListEventsSince(ctx, r.sessionID, lastSeq)
	if err != nil {
		return fmt.Errorf("realtime: replay events: %w", err)
	}
	for _, e := range events {
		senderID := ""
		if e.ActorID != nil {
			senderID = *e.ActorID
		}
		env := Envelope{
			Type:      e.Type,
			SessionID: e.SessionID,
			SenderID:  senderID,
			Payload:   e.Payload,
			Timestamp: e.CreatedAt.Unix(),
		}
		data, _ := json.Marshal(env)
		select {
		case c.send <- data:
		default:
			return fmt.Errorf("realtime: client send buffer full during replay")
		}
	}
	return nil
}

// ClientCount returns the number of connected clients (goroutine-safe).
func (r *Room) ClientCount() int {
	ch := make(chan int, 1)
	select {
	case r.queryClientCount <- ch:
		return <-ch
	case <-r.stopped:
		return 0
	}
}

// StateSnapshot returns a copy of the room's current GameState (goroutine-safe).
func (r *Room) StateSnapshot() GameState {
	ch := make(chan GameState, 1)
	select {
	case r.queryState <- ch:
		return <-ch
	case <-r.stopped:
		return GameState{}
	}
}

// Register adds a client to the room (goroutine-safe).
func (r *Room) Register(c *Client) {
	r.register <- c
}

// SessionID returns the room's session ID.
func (r *Room) SessionID() string {
	return r.sessionID
}
