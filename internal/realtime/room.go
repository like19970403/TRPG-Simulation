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
	SaveSnapshot(ctx context.Context, sessionID string, snapshotSeq int64, state json.RawMessage) error
	LoadSnapshot(ctx context.Context, sessionID string) (snapshotSeq int64, state json.RawMessage, err error)
}

// snapshotInterval is the number of events between automatic snapshots (ADR-004).
const snapshotInterval int64 = 50

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
	state := NewGameState(sessionID)
	if scenario != nil && len(scenario.Variables) > 0 {
		state.InitVariables(scenario.Variables)
	}
	return &Room{
		sessionID:        sessionID,
		gmID:             gmID,
		scenario:         scenario,
		clients:          make(map[*Client]bool),
		incoming:         make(chan incomingMessage, 64),
		register:         make(chan *Client, 16),
		unregister:       make(chan *Client, 16),
		processEvent:     make(chan eventRequest, 16),
		queryClientCount: make(chan chan int),
		queryState:       make(chan chan GameState),
		stop:             make(chan struct{}),
		stopped:          make(chan struct{}),
		state:            state,
		eventRepo:        eventRepo,
		logger:           logger,
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

	// Snapshot check (ADR-004): save state every snapshotInterval events.
	if r.state.LastSequence > 0 && r.state.LastSequence%snapshotInterval == 0 {
		r.saveSnapshot(ctx)
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
	case "reveal_item":
		r.handleRevealItem(ctx, msg.client, action.Payload)
	case "reveal_npc_field":
		r.handleRevealNPCField(ctx, msg.client, action.Payload)
	case "player_choice":
		r.handlePlayerChoice(ctx, msg.client, action.Payload)
	case "gm_broadcast":
		r.handleGMBroadcast(ctx, msg.client, action.Payload)
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

// broadcastFilteredPerClient sends an event to all clients, applying a per-client filter function
// that receives the full Client (needed for per-player item/NPC reveal filtering).
func (r *Room) broadcastFilteredPerClient(eventType string, actorID *string, payload json.RawMessage, filterFn func(json.RawMessage, *Client) json.RawMessage) {
	senderID := ""
	if actorID != nil {
		senderID = *actorID
	}
	for client := range r.clients {
		filtered := filterFn(payload, client)
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

// filterScenePayloadPerClient is the enhanced per-player filter that removes gm_notes,
// filters items_available to only revealed items, and filters NPCs to only show
// public fields and revealed hidden fields for this specific player.
func (r *Room) filterScenePayloadPerClient(payload json.RawMessage, c *Client) json.RawMessage {
	if c.role == RoleGM {
		return payload
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(payload, &m); err != nil {
		return payload
	}

	// Remove top-level gm_notes.
	delete(m, "gm_notes")

	// Filter nested "scene" object.
	if sceneRaw, ok := m["scene"]; ok {
		var scene map[string]json.RawMessage
		if err := json.Unmarshal(sceneRaw, &scene); err == nil {
			delete(scene, "gm_notes")

			// Filter items_available to only revealed items.
			if itemsRaw, ok := scene["items_available"]; ok {
				var itemIDs []string
				if err := json.Unmarshal(itemsRaw, &itemIDs); err == nil {
					var revealed []string
					for _, itemID := range itemIDs {
						if r.state.IsItemRevealed(c.userID, itemID) {
							revealed = append(revealed, itemID)
						}
					}
					if revealed == nil {
						revealed = []string{}
					}
					itemsData, _ := json.Marshal(revealed)
					scene["items_available"] = itemsData
				}
			}

			// Filter npcs_present: only include NPCs where the player can see at least
			// one field (public or revealed hidden). For each NPC, filter fields.
			if npcsRaw, ok := scene["npcs_present"]; ok {
				var npcIDs []string
				if err := json.Unmarshal(npcsRaw, &npcIDs); err == nil {
					var visibleNPCs []string
					for _, npcID := range npcIDs {
						if r.scenario != nil {
							npc := r.scenario.FindNPC(npcID)
							if npc != nil && r.hasVisibleFieldsForPlayer(npc, c.userID) {
								visibleNPCs = append(visibleNPCs, npcID)
							}
						}
					}
					if visibleNPCs == nil {
						visibleNPCs = []string{}
					}
					npcsData, _ := json.Marshal(visibleNPCs)
					scene["npcs_present"] = npcsData
				}
			}

			sceneData, _ := json.Marshal(scene)
			m["scene"] = sceneData
		}
	}

	data, _ := json.Marshal(m)
	return data
}

// hasVisibleFieldsForPlayer returns true if the player can see at least one field on this NPC.
func (r *Room) hasVisibleFieldsForPlayer(npc *NPC, playerID string) bool {
	for _, f := range npc.Fields {
		if f.Visibility == "public" {
			return true
		}
		// Check if this hidden field has been revealed to this player.
		revealedFields := r.state.RevealedFieldsForNPC(playerID, npc.ID)
		for _, rk := range revealedFields {
			if rk == f.Key {
				return true
			}
		}
	}
	return false
}

// connectedPlayerIDs returns the user IDs of all connected clients with role Player.
func (r *Room) connectedPlayerIDs() []string {
	var ids []string
	for client := range r.clients {
		if client.role == RolePlayer {
			ids = append(ids, client.userID)
		}
	}
	return ids
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

	if r.scenario.FindScene(req.SceneID) == nil {
		r.sendError(c, fmt.Sprintf("Scene not found: %s", req.SceneID))
		return
	}

	if err := r.performSceneTransition(ctx, req.SceneID, &c.userID, c.userID); err != nil {
		r.sendError(c, "Failed to persist scene change")
	}
}

// maxTransitionChainDepth limits chained auto/condition_met transitions to prevent infinite loops.
const maxTransitionChainDepth = 10

// performSceneTransition handles the full scene transition pipeline (entry point at depth 0).
func (r *Room) performSceneTransition(ctx context.Context, targetSceneID string, actorID *string, triggerPlayerID string) error {
	return r.performSceneTransitionChained(ctx, targetSceneID, actorID, triggerPlayerID, 0)
}

// performSceneTransitionChained handles the full scene transition pipeline:
// on_exit (old scene) → scene_changed → on_enter (new scene) → broadcast → evaluate auto/condition_met.
// triggerPlayerID is used for "current_player" targeting in actions.
// depth tracks chained transitions to prevent infinite loops.
func (r *Room) performSceneTransitionChained(ctx context.Context, targetSceneID string, actorID *string, triggerPlayerID string, depth int) error {
	// Execute on_exit actions for the current scene.
	if r.state.CurrentScene != "" {
		if oldScene := r.scenario.FindScene(r.state.CurrentScene); oldScene != nil && len(oldScene.OnExit) > 0 {
			r.executeAndPersistActions(ctx, oldScene.OnExit, actorID, triggerPlayerID)
		}
	}

	// Build and persist scene_changed event.
	targetScene := r.scenario.FindScene(targetSceneID)
	previousScene := r.state.CurrentScene
	eventPayload, _ := json.Marshal(map[string]any{
		"scene_id":       targetSceneID,
		"previous_scene": previousScene,
		"scene":          targetScene,
	})

	seq := r.state.LastSequence + 1
	_, err := r.eventRepo.AppendEvent(ctx, r.sessionID, seq, EventSceneChanged, actorID, eventPayload)
	if err != nil {
		r.logger.Error("failed to persist scene_changed", "error", err, "session", r.sessionID)
		return err
	}

	if err := r.state.Apply(EventSceneChanged, seq, eventPayload); err != nil {
		r.logger.Error("failed to apply scene_changed", "error", err, "session", r.sessionID)
	}

	// Execute on_enter actions for the new scene.
	if targetScene != nil && len(targetScene.OnEnter) > 0 {
		r.executeAndPersistActions(ctx, targetScene.OnEnter, actorID, triggerPlayerID)
	}

	// Broadcast with per-client filtering.
	r.broadcastFilteredPerClient(EventSceneChanged, actorID, eventPayload, r.filterScenePayloadPerClient)

	// Evaluate auto/condition_met transitions for the new scene (chained).
	r.evaluateTransitions(ctx, actorID, triggerPlayerID, depth)

	return nil
}

// evaluateTransitions checks the current scene's transitions for auto/condition_met triggers.
// If a match is found, performs the transition and recurses (up to maxTransitionChainDepth).
func (r *Room) evaluateTransitions(ctx context.Context, actorID *string, triggerPlayerID string, depth int) {
	if depth >= maxTransitionChainDepth {
		r.logger.Warn("transition chain depth limit reached", "session", r.sessionID, "depth", depth)
		return
	}

	if r.scenario == nil || r.state.CurrentScene == "" {
		return
	}

	scene := r.scenario.FindScene(r.state.CurrentScene)
	if scene == nil {
		return
	}

	for _, t := range scene.Transitions {
		switch t.Trigger {
		case "auto":
			if r.scenario.FindScene(t.Target) == nil {
				r.logger.Warn("auto transition target not found", "target", t.Target, "session", r.sessionID)
				continue
			}
			if err := r.performSceneTransitionChained(ctx, t.Target, actorID, triggerPlayerID, depth+1); err != nil {
				r.logger.Warn("auto transition failed", "error", err, "session", r.sessionID)
			}
			return // Only one auto transition per scene entry.

		case "condition_met":
			if t.Condition == "" {
				continue
			}
			if r.scenario.FindScene(t.Target) == nil {
				r.logger.Warn("condition_met transition target not found", "target", t.Target, "session", r.sessionID)
				continue
			}
			evaluator := NewExprEvaluator(r.state, r.scenario, triggerPlayerID, r.connectedPlayerIDs())
			result, err := evaluator.EvalBool(t.Condition)
			if err != nil {
				r.logger.Warn("condition evaluation failed", "error", err, "condition", t.Condition, "session", r.sessionID)
				continue
			}
			if result {
				if err := r.performSceneTransitionChained(ctx, t.Target, actorID, triggerPlayerID, depth+1); err != nil {
					r.logger.Warn("condition_met transition failed", "error", err, "session", r.sessionID)
				}
				return // First matching condition_met wins.
			}
		}
	}
}

// executeAndPersistActions runs a list of scene actions, persisting and applying each event result.
// Failures are logged but do not abort the remaining actions (graceful degradation).
func (r *Room) executeAndPersistActions(ctx context.Context, actions []Action, actorID *string, triggerPlayerID string) {
	var evaluator *ExprEvaluator
	if r.scenario != nil {
		evaluator = NewExprEvaluator(r.state, r.scenario, triggerPlayerID, r.connectedPlayerIDs())
	}

	results, err := executeActions(actions, triggerPlayerID, r.connectedPlayerIDs(), r.state.Variables, evaluator)
	if err != nil {
		r.logger.Warn("failed to execute actions", "error", err, "session", r.sessionID)
		return
	}

	for _, res := range results {
		seq := r.state.LastSequence + 1
		_, err := r.eventRepo.AppendEvent(ctx, r.sessionID, seq, res.eventType, actorID, res.payload)
		if err != nil {
			r.logger.Warn("failed to persist action event", "error", err, "session", r.sessionID, "type", res.eventType)
			continue
		}
		if err := r.state.Apply(res.eventType, seq, res.payload); err != nil {
			r.logger.Warn("failed to apply action event", "error", err, "session", r.sessionID, "type", res.eventType)
		}

		// Snapshot check (ADR-004).
		if r.state.LastSequence > 0 && r.state.LastSequence%snapshotInterval == 0 {
			r.saveSnapshot(ctx)
		}
	}
}

// handleRevealItem processes a GM's manual item reveal request.
func (r *Room) handleRevealItem(ctx context.Context, c *Client, payload json.RawMessage) {
	// Permission check: GM only.
	if c.role != RoleGM {
		r.sendError(c, "Only the GM can reveal items")
		return
	}

	// State check: must be active.
	if r.state.Status != "active" {
		r.sendError(c, "Game is not active")
		return
	}

	// Parse payload.
	var req RevealItemPayload
	if err := json.Unmarshal(payload, &req); err != nil {
		r.sendError(c, "Invalid reveal_item payload")
		return
	}
	if req.ItemID == "" {
		r.sendError(c, "item_id is required")
		return
	}

	// Scenario check.
	if r.scenario == nil {
		r.sendError(c, "Scenario not loaded")
		return
	}
	if r.scenario.FindItem(req.ItemID) == nil {
		r.sendError(c, fmt.Sprintf("Item not found: %s", req.ItemID))
		return
	}

	// Resolve player IDs.
	playerIDs := req.PlayerIDs
	if len(playerIDs) == 0 {
		playerIDs = r.connectedPlayerIDs()
	}

	// Build event payload.
	eventPayload, _ := json.Marshal(map[string]any{
		"item_id":    req.ItemID,
		"player_ids": playerIDs,
		"method":     "gm_manual",
	})

	// Persist → apply → broadcast.
	seq := r.state.LastSequence + 1
	_, err := r.eventRepo.AppendEvent(ctx, r.sessionID, seq, EventItemRevealed, &c.userID, eventPayload)
	if err != nil {
		r.logger.Error("failed to persist item_revealed", "error", err, "session", r.sessionID)
		r.sendError(c, "Failed to persist item reveal")
		return
	}

	if err := r.state.Apply(EventItemRevealed, seq, eventPayload); err != nil {
		r.logger.Error("failed to apply item_revealed", "error", err, "session", r.sessionID)
	}

	// Broadcast to all clients.
	env := NewEnvelope(EventItemRevealed, r.sessionID, c.userID, eventPayload)
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

// handleRevealNPCField processes a GM's manual NPC field reveal request.
func (r *Room) handleRevealNPCField(ctx context.Context, c *Client, payload json.RawMessage) {
	// Permission check: GM only.
	if c.role != RoleGM {
		r.sendError(c, "Only the GM can reveal NPC fields")
		return
	}

	// State check: must be active.
	if r.state.Status != "active" {
		r.sendError(c, "Game is not active")
		return
	}

	// Parse payload.
	var req RevealNPCFieldPayload
	if err := json.Unmarshal(payload, &req); err != nil {
		r.sendError(c, "Invalid reveal_npc_field payload")
		return
	}
	if req.NPCID == "" {
		r.sendError(c, "npc_id is required")
		return
	}
	if req.FieldKey == "" {
		r.sendError(c, "field_key is required")
		return
	}

	// Scenario check.
	if r.scenario == nil {
		r.sendError(c, "Scenario not loaded")
		return
	}
	npc := r.scenario.FindNPC(req.NPCID)
	if npc == nil {
		r.sendError(c, fmt.Sprintf("NPC not found: %s", req.NPCID))
		return
	}
	if npc.FindField(req.FieldKey) == nil {
		r.sendError(c, fmt.Sprintf("Field not found: %s", req.FieldKey))
		return
	}

	// Resolve player IDs.
	playerIDs := req.PlayerIDs
	if len(playerIDs) == 0 {
		playerIDs = r.connectedPlayerIDs()
	}

	// Build event payload.
	eventPayload, _ := json.Marshal(map[string]any{
		"npc_id":     req.NPCID,
		"field_key":  req.FieldKey,
		"player_ids": playerIDs,
	})

	// Persist → apply → broadcast.
	seq := r.state.LastSequence + 1
	_, err := r.eventRepo.AppendEvent(ctx, r.sessionID, seq, EventNPCFieldRevealed, &c.userID, eventPayload)
	if err != nil {
		r.logger.Error("failed to persist npc_field_revealed", "error", err, "session", r.sessionID)
		r.sendError(c, "Failed to persist NPC field reveal")
		return
	}

	if err := r.state.Apply(EventNPCFieldRevealed, seq, eventPayload); err != nil {
		r.logger.Error("failed to apply npc_field_revealed", "error", err, "session", r.sessionID)
	}

	// Broadcast to all clients.
	env := NewEnvelope(EventNPCFieldRevealed, r.sessionID, c.userID, eventPayload)
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

// handlePlayerChoice processes a player's scene transition choice.
func (r *Room) handlePlayerChoice(ctx context.Context, c *Client, payload json.RawMessage) {
	// State check: must be active.
	if r.state.Status != "active" {
		r.sendError(c, "Game is not active")
		return
	}

	// Parse payload.
	var req PlayerChoicePayload
	if err := json.Unmarshal(payload, &req); err != nil {
		r.sendError(c, "Invalid player_choice payload")
		return
	}

	// Scenario check.
	if r.scenario == nil {
		r.sendError(c, "Scenario not loaded")
		return
	}

	// Current scene check.
	if r.state.CurrentScene == "" {
		r.sendError(c, "No current scene")
		return
	}

	currentScene := r.scenario.FindScene(r.state.CurrentScene)
	if currentScene == nil {
		r.sendError(c, "Current scene not found in scenario")
		return
	}

	// Validate transition index.
	if req.TransitionIndex < 0 || req.TransitionIndex >= len(currentScene.Transitions) {
		r.sendError(c, "Transition index out of range")
		return
	}

	transition := currentScene.Transitions[req.TransitionIndex]

	// Validate trigger type.
	if transition.Trigger != "player_choice" {
		r.sendError(c, "Transition is not a player choice")
		return
	}

	// Validate target scene exists.
	if r.scenario.FindScene(transition.Target) == nil {
		r.sendError(c, fmt.Sprintf("Target scene not found: %s", transition.Target))
		return
	}

	// Persist player_choice event (informational/audit).
	choicePayload, _ := json.Marshal(map[string]any{
		"scene_id":         r.state.CurrentScene,
		"transition_label": transition.Label,
		"target_scene":     transition.Target,
	})

	seq := r.state.LastSequence + 1
	_, err := r.eventRepo.AppendEvent(ctx, r.sessionID, seq, EventPlayerChoice, &c.userID, choicePayload)
	if err != nil {
		r.logger.Error("failed to persist player_choice", "error", err, "session", r.sessionID)
		r.sendError(c, "Failed to persist player choice")
		return
	}

	if err := r.state.Apply(EventPlayerChoice, seq, choicePayload); err != nil {
		r.logger.Error("failed to apply player_choice", "error", err, "session", r.sessionID)
	}

	// Broadcast player_choice event.
	env := NewEnvelope(EventPlayerChoice, r.sessionID, c.userID, choicePayload)
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

	// Perform the actual scene transition.
	if err := r.performSceneTransition(ctx, transition.Target, &c.userID, c.userID); err != nil {
		r.sendError(c, "Failed to persist scene change")
	}
}

// handleGMBroadcast processes a GM's text/image broadcast to specific or all players.
func (r *Room) handleGMBroadcast(ctx context.Context, c *Client, payload json.RawMessage) {
	// Permission check: GM only.
	if c.role != RoleGM {
		r.sendError(c, "Only the GM can broadcast messages")
		return
	}

	// State check: must be active.
	if r.state.Status != "active" {
		r.sendError(c, "Game is not active")
		return
	}

	// Parse payload.
	var req GMBroadcastPayload
	if err := json.Unmarshal(payload, &req); err != nil {
		r.sendError(c, "Invalid gm_broadcast payload")
		return
	}

	// Validate: at least content or image_url must be provided.
	if req.Content == "" && req.ImageURL == "" {
		r.sendError(c, "content or image_url is required")
		return
	}

	// Resolve target player IDs (empty = all connected players).
	targetPlayerIDs := req.PlayerIDs
	if len(targetPlayerIDs) == 0 {
		targetPlayerIDs = r.connectedPlayerIDs()
	}

	// Build a target set for fast lookup.
	targetSet := make(map[string]bool, len(targetPlayerIDs))
	for _, pid := range targetPlayerIDs {
		targetSet[pid] = true
	}

	// Build event payload.
	eventPayload, _ := json.Marshal(map[string]any{
		"content":    req.Content,
		"image_url":  req.ImageURL,
		"player_ids": targetPlayerIDs,
	})

	// Persist → apply → snapshot check.
	seq := r.state.LastSequence + 1
	_, err := r.eventRepo.AppendEvent(ctx, r.sessionID, seq, EventGMBroadcast, &c.userID, eventPayload)
	if err != nil {
		r.logger.Error("failed to persist gm_broadcast", "error", err, "session", r.sessionID)
		r.sendError(c, "Failed to persist GM broadcast")
		return
	}

	if err := r.state.Apply(EventGMBroadcast, seq, eventPayload); err != nil {
		r.logger.Error("failed to apply gm_broadcast", "error", err, "session", r.sessionID)
	}

	// Snapshot check (ADR-004).
	if r.state.LastSequence > 0 && r.state.LastSequence%snapshotInterval == 0 {
		r.saveSnapshot(ctx)
	}

	// Broadcast: GM always receives + targeted players only.
	env := NewEnvelope(EventGMBroadcast, r.sessionID, c.userID, eventPayload)
	data, _ := json.Marshal(env)
	for client := range r.clients {
		// GM always receives their own broadcast.
		if client.userID == r.gmID || targetSet[client.userID] {
			select {
			case client.send <- data:
			default:
				close(client.send)
				delete(r.clients, client)
				r.logger.Warn("client dropped (buffer full)", "session", r.sessionID, "user", client.userID)
			}
		}
	}
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

// saveSnapshot serializes the current GameState and persists it via the EventRepository.
// Errors are logged but not propagated (snapshot is an optimization, not critical).
func (r *Room) saveSnapshot(ctx context.Context) {
	stateData, err := json.Marshal(r.state)
	if err != nil {
		r.logger.Error("failed to marshal state for snapshot", "error", err, "session", r.sessionID)
		return
	}
	if err := r.eventRepo.SaveSnapshot(ctx, r.sessionID, r.state.LastSequence, stateData); err != nil {
		r.logger.Error("failed to save snapshot", "error", err, "session", r.sessionID, "seq", r.state.LastSequence)
	}
}

// RecoverFromSnapshot loads the latest snapshot and replays subsequent events to rebuild state.
// If no snapshot exists, replays all events from sequence 0.
func (r *Room) RecoverFromSnapshot(ctx context.Context) error {
	snapshotSeq, stateData, err := r.eventRepo.LoadSnapshot(ctx, r.sessionID)
	if err != nil {
		return fmt.Errorf("realtime: load snapshot: %w", err)
	}

	// If we have a snapshot, restore state from it.
	if stateData != nil {
		var gs GameState
		if err := json.Unmarshal(stateData, &gs); err != nil {
			return fmt.Errorf("realtime: unmarshal snapshot: %w", err)
		}
		*r.state = gs
	}

	// Replay events after the snapshot (or from 0 if no snapshot).
	events, err := r.eventRepo.ListEventsSince(ctx, r.sessionID, snapshotSeq)
	if err != nil {
		return fmt.Errorf("realtime: list events for recovery: %w", err)
	}

	for _, e := range events {
		if err := r.state.Apply(e.Type, e.Sequence, e.Payload); err != nil {
			r.logger.Warn("skip event during recovery", "error", err, "session", r.sessionID, "seq", e.Sequence)
		}
	}

	return nil
}

// SessionID returns the room's session ID.
func (r *Room) SessionID() string {
	return r.sessionID
}
