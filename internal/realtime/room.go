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
	scenario     *ScenarioContent
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

// NewRoom creates a Room. scenario may be nil (graceful degradation).
// Call Run() as a goroutine to start the event loop.
func NewRoom(sessionID, gmID string, scenario *ScenarioContent, eventRepo EventRepository, logger *slog.Logger) *Room {
	return &Room{
		sessionID:    sessionID,
		gmID:         gmID,
		scenario:     scenario,
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
			r.handleIncoming(ctx, msg)

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
	seq := r.state.LastSequence + 1

	// Persist to DB.
	_, err := r.eventRepo.AppendEvent(ctx, r.sessionID, seq, req.eventType, req.actorID, req.payload)
	if err != nil {
		r.logger.Error("failed to persist event", "error", err, "session", r.sessionID, "type", req.eventType)
		return
	}

	// Apply to in-memory state (also advances LastSequence).
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

// handleIncoming dispatches an incoming WebSocket message to the appropriate handler.
func (r *Room) handleIncoming(ctx context.Context, msg incomingMessage) {
	var action IncomingAction
	if err := json.Unmarshal(msg.data, &action); err != nil {
		r.sendError(msg.client, "Invalid message format")
		return
	}

	switch action.Type {
	case "advance_scene":
		r.handleAdvanceScene(ctx, msg.client, action.Payload)
	case "dice_roll":
		r.handleDiceRoll(ctx, msg.client, action.Payload)
	default:
		r.sendError(msg.client, fmt.Sprintf("Unknown action type: %q", action.Type))
	}
}

// sendError sends an error envelope to a single client.
func (r *Room) sendError(c *Client, message string) {
	payload, _ := json.Marshal(map[string]string{"message": message})
	env := NewEnvelope(EventError, r.sessionID, "", payload)
	data, _ := json.Marshal(env)
	select {
	case c.send <- data:
	default:
		// Buffer full; will be disconnected by room eventually.
	}
}

// broadcastFiltered sends an event to all clients, applying a per-client filter function.
// The filterFn receives the original payload and client role, returning the filtered payload.
func (r *Room) broadcastFiltered(eventType string, actorID *string, payload json.RawMessage, filterFn func(json.RawMessage, SenderRole) json.RawMessage) {
	senderID := ""
	if actorID != nil {
		senderID = *actorID
	}
	for client := range r.clients {
		filtered := filterFn(payload, client.role)
		env := NewEnvelope(eventType, r.sessionID, senderID, filtered)
		data, _ := json.Marshal(env)
		select {
		case client.send <- data:
		default:
			close(client.send)
			delete(r.clients, client)
			r.logger.Warn("client dropped (buffer full)", "session", r.sessionID, "user", client.userID)
		}
	}
}

// filterScenePayload removes gm_notes from the payload for non-GM clients.
func filterScenePayload(payload json.RawMessage, role SenderRole) json.RawMessage {
	if role == RoleGM {
		return payload
	}
	// Parse, remove gm_notes from the nested scene object, re-marshal.
	var m map[string]json.RawMessage
	if err := json.Unmarshal(payload, &m); err != nil {
		return payload
	}
	// Remove top-level gm_notes if present.
	delete(m, "gm_notes")

	// Also remove gm_notes from nested "scene" object if present.
	if sceneRaw, ok := m["scene"]; ok {
		var scene map[string]json.RawMessage
		if err := json.Unmarshal(sceneRaw, &scene); err == nil {
			delete(scene, "gm_notes")
			sceneData, _ := json.Marshal(scene)
			m["scene"] = sceneData
		}
	}

	data, _ := json.Marshal(m)
	return data
}

// handleAdvanceScene processes a GM's scene switch request.
func (r *Room) handleAdvanceScene(ctx context.Context, c *Client, payload json.RawMessage) {
	// Permission check: GM only.
	if c.role != RoleGM {
		r.sendError(c, "Only the GM can advance the scene")
		return
	}

	// State check: must be active.
	if r.state.Status != "active" {
		r.sendError(c, "Game is not active")
		return
	}

	// Parse payload.
	var req AdvanceScenePayload
	if err := json.Unmarshal(payload, &req); err != nil {
		r.sendError(c, "Invalid advance_scene payload")
		return
	}
	if req.SceneID == "" {
		r.sendError(c, "scene_id is required")
		return
	}

	// Scenario check.
	if r.scenario == nil {
		r.sendError(c, "Scenario not loaded")
		return
	}

	scene := r.scenario.FindScene(req.SceneID)
	if scene == nil {
		r.sendError(c, fmt.Sprintf("Scene not found: %s", req.SceneID))
		return
	}

	// Build event payload with full scene data.
	previousScene := r.state.CurrentScene
	eventPayload, _ := json.Marshal(map[string]any{
		"scene_id":       req.SceneID,
		"previous_scene": previousScene,
		"scene":          scene,
	})

	// Serialize: assign seq → persist → apply → broadcastFiltered.
	seq := r.state.LastSequence + 1

	_, err := r.eventRepo.AppendEvent(ctx, r.sessionID, seq, EventSceneChanged, &c.userID, eventPayload)
	if err != nil {
		r.logger.Error("failed to persist scene_changed", "error", err, "session", r.sessionID)
		r.sendError(c, "Failed to persist scene change")
		return
	}

	if err := r.state.Apply(EventSceneChanged, seq, eventPayload); err != nil {
		r.logger.Error("failed to apply scene_changed", "error", err, "session", r.sessionID)
	}

	r.broadcastFiltered(EventSceneChanged, &c.userID, eventPayload, filterScenePayload)
}

// handleDiceRoll processes a dice roll request from any participant (GM or Player).
func (r *Room) handleDiceRoll(ctx context.Context, c *Client, payload json.RawMessage) {
	// State check: must be active.
	if r.state.Status != "active" {
		r.sendError(c, "Game is not active")
		return
	}

	// Parse payload.
	var req DiceRollPayload
	if err := json.Unmarshal(payload, &req); err != nil {
		r.sendError(c, "Invalid dice_roll payload")
		return
	}
	if req.Formula == "" {
		r.sendError(c, "formula is required")
		return
	}

	// Roll dice.
	result, err := RollDice(req.Formula)
	if err != nil {
		r.sendError(c, fmt.Sprintf("Invalid dice formula: %s", err.Error()))
		return
	}

	// Build event payload.
	eventPayload, _ := json.Marshal(map[string]any{
		"roller_id": c.userID,
		"formula":   result.Formula,
		"results":   result.Results,
		"modifier":  result.Modifier,
		"total":     result.Total,
		"purpose":   req.Purpose,
	})

	// Serialize: assign seq → persist → apply → broadcast.
	seq := r.state.LastSequence + 1

	_, err = r.eventRepo.AppendEvent(ctx, r.sessionID, seq, EventDiceRolled, &c.userID, eventPayload)
	if err != nil {
		r.logger.Error("failed to persist dice_rolled", "error", err, "session", r.sessionID)
		r.sendError(c, "Failed to persist dice roll")
		return
	}

	if err := r.state.Apply(EventDiceRolled, seq, eventPayload); err != nil {
		r.logger.Error("failed to apply dice_rolled", "error", err, "session", r.sessionID)
	}

	// Dice rolls are broadcast to everyone without filtering.
	env := NewEnvelope(EventDiceRolled, r.sessionID, c.userID, eventPayload)
	data, _ := json.Marshal(env)
	for client := range r.clients {
		select {
		case client.send <- data:
		default:
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
