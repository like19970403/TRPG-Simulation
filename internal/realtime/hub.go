package realtime

import (
	"context"
	"log/slog"
	"sync"
)

// ScenarioLoader loads scenario content for a session (consumer-side interface).
type ScenarioLoader interface {
	LoadScenarioForSession(ctx context.Context, sessionID string) (*ScenarioContent, error)
}

// Hub manages all active Rooms.
type Hub struct {
	rooms          map[string]*Room
	mu             sync.RWMutex
	eventRepo      EventRepository
	scenarioLoader ScenarioLoader
	logger         *slog.Logger
	ctx            context.Context
	cancel         context.CancelFunc
}

// NewHub creates a Hub. scenarioLoader may be nil (graceful degradation).
func NewHub(eventRepo EventRepository, scenarioLoader ScenarioLoader, logger *slog.Logger) *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	return &Hub{
		rooms:          make(map[string]*Room),
		eventRepo:      eventRepo,
		scenarioLoader: scenarioLoader,
		logger:         logger,
		ctx:            ctx,
		cancel:         cancel,
	}
}

// GetOrCreateRoom returns the Room for a session, creating one if it doesn't exist.
// Loads scenario content for scene validation; gracefully degrades if loading fails.
func (h *Hub) GetOrCreateRoom(sessionID, gmID string) *Room {
	h.mu.Lock()
	defer h.mu.Unlock()

	if room, ok := h.rooms[sessionID]; ok {
		return room
	}

	// Load scenario content (graceful degradation if it fails).
	var scenario *ScenarioContent
	if h.scenarioLoader != nil {
		sc, err := h.scenarioLoader.LoadScenarioForSession(h.ctx, sessionID)
		if err != nil {
			h.logger.Warn("failed to load scenario for session", "session", sessionID, "error", err)
		} else {
			scenario = sc
		}
	}

	room := NewRoom(sessionID, gmID, scenario, h.eventRepo, h.logger)

	// Recover state from DB snapshot + event replay (graceful degradation).
	if err := room.RecoverFromSnapshot(h.ctx); err != nil {
		h.logger.Warn("failed to recover room state", "session", sessionID, "error", err)
	}

	h.rooms[sessionID] = room
	go room.Run(h.ctx)
	h.logger.Info("room created", "session", sessionID)
	return room
}

// GetRoom returns the Room for a session, or nil if it doesn't exist.
func (h *Hub) GetRoom(sessionID string) *Room {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.rooms[sessionID]
}

// RemoveRoom stops and removes a Room.
func (h *Hub) RemoveRoom(sessionID string) {
	h.mu.Lock()
	room, ok := h.rooms[sessionID]
	if ok {
		delete(h.rooms, sessionID)
	}
	h.mu.Unlock()

	if ok {
		room.Stop()
		h.logger.Info("room removed", "session", sessionID)
	}
}

// Stop shuts down all rooms and the hub.
func (h *Hub) Stop() {
	h.cancel()
	h.mu.Lock()
	defer h.mu.Unlock()

	for id, room := range h.rooms {
		room.Stop()
		delete(h.rooms, id)
	}
	h.logger.Info("hub stopped")
}

// RoomCount returns the number of active rooms (for testing).
func (h *Hub) RoomCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms)
}
