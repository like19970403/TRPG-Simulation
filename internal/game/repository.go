package game

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GameSession represents a row in the game_sessions table.
type GameSession struct {
	ID          string
	ScenarioID  string
	GMID        string
	Status      string
	InviteCode  string
	State       json.RawMessage
	GMNotes     json.RawMessage
	SnapshotSeq int64
	CreatedAt   time.Time
	StartedAt   *time.Time
	EndedAt     *time.Time
}

// SessionPlayer represents a row in the session_players table.
type SessionPlayer struct {
	ID           string
	SessionID    string
	UserID       string
	CharacterID  *string
	CurrentScene *string
	Status       string
	Notes        string
	JoinedAt     time.Time
}

// Repository provides database operations for game session entities.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new game Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// GenerateInviteCode creates a 6-character uppercase alphanumeric invite code.
// Excludes confusable characters: 0, O, 1, I.
func GenerateInviteCode() (string, error) {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("game: generate invite code: %w", err)
	}
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b), nil
}

const sessionColumns = `id, scenario_id, gm_id, status, invite_code, state, gm_notes, snapshot_seq, created_at, started_at, ended_at`

func scanSession(row pgx.Row) (*GameSession, error) {
	gs := &GameSession{}
	err := row.Scan(
		&gs.ID, &gs.ScenarioID, &gs.GMID, &gs.Status, &gs.InviteCode,
		&gs.State, &gs.GMNotes, &gs.SnapshotSeq,
		&gs.CreatedAt, &gs.StartedAt, &gs.EndedAt,
	)
	if err != nil {
		return nil, err
	}
	return gs, nil
}

// Create inserts a new game session with an auto-generated invite code.
func (r *Repository) Create(ctx context.Context, scenarioID, gmID string) (*GameSession, error) {
	// Retry up to 3 times in case of invite code collision.
	for i := 0; i < 3; i++ {
		code, err := GenerateInviteCode()
		if err != nil {
			return nil, err
		}

		gs, err := scanSession(r.pool.QueryRow(ctx,
			`INSERT INTO game_sessions (scenario_id, gm_id, invite_code)
			 VALUES ($1, $2, $3)
			 RETURNING `+sessionColumns,
			scenarioID, gmID, code,
		))
		if err != nil {
			if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate key") {
				continue
			}
			return nil, fmt.Errorf("game: create: %w", err)
		}
		return gs, nil
	}
	return nil, fmt.Errorf("game: create: failed to generate unique invite code after 3 attempts")
}

// GetByID returns a game session by its ID.
func (r *Repository) GetByID(ctx context.Context, id string) (*GameSession, error) {
	gs, err := scanSession(r.pool.QueryRow(ctx,
		`SELECT `+sessionColumns+` FROM game_sessions WHERE id = $1`, id,
	))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("game: not found: %w", err)
		}
		return nil, fmt.Errorf("game: get: %w", err)
	}
	return gs, nil
}

// ListByGM returns paginated game sessions for a given GM.
func (r *Repository) ListByGM(ctx context.Context, gmID string, limit, offset int) ([]*GameSession, int, error) {
	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM game_sessions WHERE gm_id = $1`, gmID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("game: list count: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT `+sessionColumns+` FROM game_sessions WHERE gm_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2 OFFSET $3`,
		gmID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("game: list: %w", err)
	}
	defer rows.Close()

	var sessions []*GameSession
	for rows.Next() {
		gs := &GameSession{}
		if err := rows.Scan(
			&gs.ID, &gs.ScenarioID, &gs.GMID, &gs.Status, &gs.InviteCode,
			&gs.State, &gs.GMNotes, &gs.SnapshotSeq,
			&gs.CreatedAt, &gs.StartedAt, &gs.EndedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("game: list scan: %w", err)
		}
		sessions = append(sessions, gs)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("game: list rows: %w", err)
	}

	return sessions, total, nil
}

// UpdateStatus transitions a game session's status.
// Sets started_at when transitioning to active (if not already set).
// Sets ended_at when transitioning to completed.
func (r *Repository) UpdateStatus(ctx context.Context, id, newStatus string) (*GameSession, error) {
	gs, err := scanSession(r.pool.QueryRow(ctx,
		`UPDATE game_sessions SET
			status = $2,
			started_at = CASE WHEN $2 = 'active' AND started_at IS NULL THEN NOW() ELSE started_at END,
			ended_at = CASE WHEN $2 = 'completed' THEN NOW() ELSE ended_at END
		 WHERE id = $1
		 RETURNING `+sessionColumns,
		id, newStatus,
	))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("game: not found: %w", err)
		}
		return nil, fmt.Errorf("game: update status: %w", err)
	}
	return gs, nil
}

// GetByInviteCode returns a game session by its invite code.
func (r *Repository) GetByInviteCode(ctx context.Context, code string) (*GameSession, error) {
	gs, err := scanSession(r.pool.QueryRow(ctx,
		`SELECT `+sessionColumns+` FROM game_sessions WHERE invite_code = $1`, code,
	))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("game: not found: %w", err)
		}
		return nil, fmt.Errorf("game: get by invite code: %w", err)
	}
	return gs, nil
}

const playerColumns = `id, session_id, user_id, character_id, current_scene, status, notes, joined_at`

// AddPlayer adds a user to a game session.
func (r *Repository) AddPlayer(ctx context.Context, sessionID, userID string) (*SessionPlayer, error) {
	sp := &SessionPlayer{}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO session_players (session_id, user_id)
		 VALUES ($1, $2)
		 RETURNING `+playerColumns,
		sessionID, userID,
	).Scan(&sp.ID, &sp.SessionID, &sp.UserID, &sp.CharacterID, &sp.CurrentScene, &sp.Status, &sp.Notes, &sp.JoinedAt)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate key") {
			return nil, fmt.Errorf("game: player already joined: %w", err)
		}
		return nil, fmt.Errorf("game: add player: %w", err)
	}
	return sp, nil
}

// ListPlayers returns all players in a game session.
func (r *Repository) ListPlayers(ctx context.Context, sessionID string) ([]*SessionPlayer, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+playerColumns+` FROM session_players WHERE session_id = $1 ORDER BY joined_at ASC`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("game: list players: %w", err)
	}
	defer rows.Close()

	var players []*SessionPlayer
	for rows.Next() {
		sp := &SessionPlayer{}
		if err := rows.Scan(&sp.ID, &sp.SessionID, &sp.UserID, &sp.CharacterID, &sp.CurrentScene, &sp.Status, &sp.Notes, &sp.JoinedAt); err != nil {
			return nil, fmt.Errorf("game: list players scan: %w", err)
		}
		players = append(players, sp)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("game: list players rows: %w", err)
	}

	return players, nil
}

// RemovePlayer removes a user from a game session.
func (r *Repository) RemovePlayer(ctx context.Context, sessionID, userID string) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM session_players WHERE session_id = $1 AND user_id = $2`,
		sessionID, userID,
	)
	if err != nil {
		return fmt.Errorf("game: remove player: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("game: player not found")
	}
	return nil
}

// GetPlayer returns a specific player in a game session.
func (r *Repository) GetPlayer(ctx context.Context, sessionID, userID string) (*SessionPlayer, error) {
	sp := &SessionPlayer{}
	err := r.pool.QueryRow(ctx,
		`SELECT `+playerColumns+` FROM session_players WHERE session_id = $1 AND user_id = $2`,
		sessionID, userID,
	).Scan(&sp.ID, &sp.SessionID, &sp.UserID, &sp.CharacterID, &sp.CurrentScene, &sp.Status, &sp.Notes, &sp.JoinedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("game: player not found: %w", err)
		}
		return nil, fmt.Errorf("game: get player: %w", err)
	}
	return sp, nil
}
