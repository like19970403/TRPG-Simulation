package server

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"

	"github.com/like19970403/TRPG-Simulation/internal/apperror"
	"github.com/like19970403/TRPG-Simulation/internal/auth"
	"github.com/like19970403/TRPG-Simulation/internal/realtime"
)

// isPrivateIP checks if a hostname resolves to a private/loopback IP (RFC 1918 / RFC 4193).
func isPrivateIP(hostname string) bool {
	if hostname == "localhost" {
		return true
	}
	ip := net.ParseIP(hostname)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate()
}

// newUpgrader creates a WebSocket upgrader with origin checking based on allowedOrigins.
// If allowedOrigins is empty, only same-host and localhost/private-IP origins are allowed.
func newUpgrader(allowedOrigins []string) websocket.Upgrader {
	allowed := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		o = strings.TrimSpace(o)
		o = strings.TrimRight(o, "/")
		if o != "" {
			allowed[o] = true
		}
	}

	return websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return false
			}

			parsed, err := url.Parse(origin)
			if err != nil {
				return false
			}
			originHost := parsed.Hostname()

			// Allow localhost and private/LAN IPs (dev environments).
			if isPrivateIP(originHost) {
				return true
			}

			// If explicit allow-list configured, check against it.
			if len(allowed) > 0 {
				return allowed[origin]
			}

			// Default: same-host policy (compare hostnames, ignoring port).
			reqHost := r.Host
			if h, _, found := strings.Cut(reqHost, ":"); found {
				reqHost = h
			}
			return originHost == reqHost
		},
	}
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	if s.hub == nil {
		s.writeError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "WebSocket not available", nil)
		return
	}

	id := r.PathValue("id")
	if !isValidUUID(id) {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid session ID", nil)
		return
	}

	// JWT from query param (browsers can't set Authorization header for WS).
	// NOTE: A more secure approach would use the Sec-WebSocket-Protocol header,
	// but this requires coordinated frontend changes. The JWT is short-lived (15min)
	// and access logs are internal, so this is acceptable for now.
	token := r.URL.Query().Get("token")
	if token == "" {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Missing token parameter", nil)
		return
	}

	claims, err := auth.ValidateAccessToken(token, s.jwtSecret)
	if err != nil {
		s.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or expired token", nil)
		return
	}

	// Lookup session.
	gs, err := s.sessionRepo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Session not found", nil)
			return
		}
		s.logger.Error("ws: get session", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	// Session must be active or paused.
	if gs.Status != "active" && gs.Status != "paused" {
		s.writeError(w, http.StatusForbidden, "FORBIDDEN", "Session is not active", nil)
		return
	}

	// User must be a participant.
	if !s.isSessionParticipant(r, gs.ID, claims.UserID, gs.GMID) {
		s.writeError(w, http.StatusForbidden, "FORBIDDEN", "You are not a member of this session", nil)
		return
	}

	// Determine role.
	role := realtime.RolePlayer
	if claims.UserID == gs.GMID {
		role = realtime.RoleGM
	}

	// Parse last_event_seq.
	var lastEventSeq int64
	if seqStr := r.URL.Query().Get("last_event_seq"); seqStr != "" {
		lastEventSeq, _ = strconv.ParseInt(seqStr, 10, 64)
	}

	// Upgrade to WebSocket.
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("ws: upgrade failed", "error", err)
		return // Upgrade writes its own HTTP error
	}

	// Get or create room.
	room := s.hub.GetOrCreateRoom(gs.ID, gs.GMID)

	// Create client.
	client := realtime.NewClient(conn, room, claims.UserID, claims.Username, role, s.logger)

	// Load character data for players.
	if role == realtime.RolePlayer {
		sp, err := s.sessionRepo.GetPlayer(r.Context(), gs.ID, claims.UserID)
		if err == nil && sp.CharacterID != nil {
			ch, err := s.characterRepo.GetByID(r.Context(), *sp.CharacterID)
			if err == nil {
				client.SetCharacter(ch.ID, ch.Name)
				if ch.Attributes != nil {
					var attrs map[string]any
					if err := json.Unmarshal(ch.Attributes, &attrs); err != nil {
						s.logger.Warn("ws: unmarshal character attributes", "error", err, "characterID", ch.ID)
					} else {
						client.SetAttributes(attrs)
					}
				}
			}
		}
	}

	// Register client with room.
	room.Register(client)

	// Replay missed events.
	if lastEventSeq > 0 {
		if err := room.ReplayEvents(r.Context(), client, lastEventSeq); err != nil {
			s.logger.Error("ws: replay events", "error", err, "session", gs.ID, "user", claims.UserID)
		}
	}

	// Send state_sync envelope.
	state := room.StateSnapshot()
	statePayload, err := json.Marshal(state)
	if err != nil {
		s.logger.Error("ws: marshal state snapshot", "error", err, "session", gs.ID, "user", claims.UserID)
		conn.Close()
		return
	}
	syncEnv := realtime.NewEnvelope(realtime.EventStateSync, gs.ID, "", statePayload)
	syncData, err := json.Marshal(syncEnv)
	if err != nil {
		s.logger.Error("ws: marshal state_sync envelope", "error", err, "session", gs.ID, "user", claims.UserID)
		conn.Close()
		return
	}
	client.Send(syncData)

	// Start read/write pumps.
	client.Start()
}
