package game

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/like19970403/TRPG-Simulation/internal/apperror"
)

// GameEvent represents a row in the game_events table.
type GameEvent struct {
	ID        string
	SessionID string
	Sequence  int64
	Type      string
	ActorID   *string
	Payload   json.RawMessage
	CreatedAt time.Time
}

const eventColumns = `id, session_id, sequence, type, actor_id, payload, created_at`

func scanEvent(row pgx.Row) (*GameEvent, error) {
	e := &GameEvent{}
	err := row.Scan(&e.ID, &e.SessionID, &e.Sequence, &e.Type, &e.ActorID, &e.Payload, &e.CreatedAt)
	if err != nil {
		return nil, err
	}
	return e, nil
}

// AppendEvent inserts a game event with the given sequence number.
func (r *Repository) AppendEvent(ctx context.Context, sessionID string, sequence int64, eventType string, actorID *string, payload json.RawMessage) (*GameEvent, error) {
	e, err := scanEvent(r.pool.QueryRow(ctx,
		`INSERT INTO game_events (session_id, sequence, type, actor_id, payload)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING `+eventColumns,
		sessionID, sequence, eventType, actorID, payload,
	))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, fmt.Errorf("game: duplicate event sequence: %w", apperror.ErrDuplicate)
		}
		return nil, fmt.Errorf("game: append event: %w", err)
	}
	return e, nil
}

// ListEventsSince returns all events for a session with sequence > afterSeq,
// ordered by sequence ascending. Used for reconnect replay.
func (r *Repository) ListEventsSince(ctx context.Context, sessionID string, afterSeq int64) ([]*GameEvent, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+eventColumns+` FROM game_events
		 WHERE session_id = $1 AND sequence > $2
		 ORDER BY sequence ASC`,
		sessionID, afterSeq,
	)
	if err != nil {
		return nil, fmt.Errorf("game: list events since: %w", err)
	}
	defer rows.Close()

	var events []*GameEvent
	for rows.Next() {
		e := &GameEvent{}
		if err := rows.Scan(&e.ID, &e.SessionID, &e.Sequence, &e.Type, &e.ActorID, &e.Payload, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("game: list events scan: %w", err)
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("game: list events rows: %w", err)
	}

	if events == nil {
		events = []*GameEvent{}
	}
	return events, nil
}

// SaveSnapshot persists a game state snapshot at the given event sequence.
// Snapshots are stored in the game_sessions table (ADR-004).
func (r *Repository) SaveSnapshot(ctx context.Context, sessionID string, snapshotSeq int64, state json.RawMessage) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE game_sessions SET state = $2, snapshot_seq = $3 WHERE id = $1`,
		sessionID, state, snapshotSeq,
	)
	if err != nil {
		return fmt.Errorf("game: save snapshot: %w", err)
	}
	return nil
}

// LoadSnapshot returns the latest snapshot for a session.
// Returns (0, nil, nil) if no snapshot exists (snapshot_seq == 0).
func (r *Repository) LoadSnapshot(ctx context.Context, sessionID string) (int64, json.RawMessage, error) {
	var snapshotSeq int64
	var state json.RawMessage
	err := r.pool.QueryRow(ctx,
		`SELECT snapshot_seq, state FROM game_sessions WHERE id = $1`,
		sessionID,
	).Scan(&snapshotSeq, &state)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, nil, nil
		}
		return 0, nil, fmt.Errorf("game: load snapshot: %w", err)
	}
	// snapshot_seq=0 means no snapshot has been saved yet.
	if snapshotSeq == 0 {
		return 0, nil, nil
	}
	return snapshotSeq, state, nil
}
