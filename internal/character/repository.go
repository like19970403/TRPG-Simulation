package character

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Character represents a row in the characters table.
type Character struct {
	ID         string
	UserID     string
	Name       string
	Attributes json.RawMessage
	Inventory  json.RawMessage
	Notes      string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Repository provides database operations for character entities.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new character Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const characterColumns = `id, user_id, name, attributes, inventory, notes, created_at, updated_at`

func scanCharacter(row pgx.Row) (*Character, error) {
	c := &Character{}
	err := row.Scan(&c.ID, &c.UserID, &c.Name, &c.Attributes, &c.Inventory, &c.Notes, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// Create inserts a new character.
func (r *Repository) Create(ctx context.Context, userID, name string, attributes, inventory json.RawMessage, notes string) (*Character, error) {
	c, err := scanCharacter(r.pool.QueryRow(ctx,
		`INSERT INTO characters (user_id, name, attributes, inventory, notes)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING `+characterColumns,
		userID, name, attributes, inventory, notes,
	))
	if err != nil {
		return nil, fmt.Errorf("character: create: %w", err)
	}
	return c, nil
}

// GetByID returns a character by its ID.
func (r *Repository) GetByID(ctx context.Context, id string) (*Character, error) {
	c, err := scanCharacter(r.pool.QueryRow(ctx,
		`SELECT `+characterColumns+` FROM characters WHERE id = $1`, id,
	))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("character: not found: %w", err)
		}
		return nil, fmt.Errorf("character: get: %w", err)
	}
	return c, nil
}

// ListByUser returns characters owned by a user with pagination.
func (r *Repository) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*Character, int, error) {
	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM characters WHERE user_id = $1`, userID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("character: list count: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT `+characterColumns+` FROM characters WHERE user_id = $1
		 ORDER BY updated_at DESC
		 LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("character: list: %w", err)
	}
	defer rows.Close()

	var characters []*Character
	for rows.Next() {
		c := &Character{}
		if err := rows.Scan(&c.ID, &c.UserID, &c.Name, &c.Attributes, &c.Inventory, &c.Notes, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("character: list scan: %w", err)
		}
		characters = append(characters, c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("character: list rows: %w", err)
	}

	return characters, total, nil
}

// Update modifies an existing character.
func (r *Repository) Update(ctx context.Context, id, name string, attributes, inventory json.RawMessage, notes string) (*Character, error) {
	c, err := scanCharacter(r.pool.QueryRow(ctx,
		`UPDATE characters
		 SET name = $2, attributes = $3, inventory = $4, notes = $5, updated_at = NOW()
		 WHERE id = $1
		 RETURNING `+characterColumns,
		id, name, attributes, inventory, notes,
	))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("character: not found: %w", err)
		}
		return nil, fmt.Errorf("character: update: %w", err)
	}
	return c, nil
}

// Delete removes a character by its ID.
func (r *Repository) Delete(ctx context.Context, id string) error {
	result, err := r.pool.Exec(ctx,
		`DELETE FROM characters WHERE id = $1`, id,
	)
	if err != nil {
		return fmt.Errorf("character: delete: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("character: not found")
	}
	return nil
}

// IsLinkedToSession returns true if the character is assigned to any session player.
func (r *Repository) IsLinkedToSession(ctx context.Context, id string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM session_players WHERE character_id = $1)`, id,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("character: check session link: %w", err)
	}
	return exists, nil
}
