package scenario

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/like19970403/TRPG-Simulation/internal/apperror"
)

// Scenario represents a row in the scenarios table.
type Scenario struct {
	ID          string
	AuthorID    string
	Title       string
	Description string
	Version     int
	Status      string
	Content     json.RawMessage
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Repository provides database operations for scenario entities.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new scenario Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create inserts a new scenario and returns the created record.
func (r *Repository) Create(ctx context.Context, authorID, title, description string, content json.RawMessage) (*Scenario, error) {
	sc := &Scenario{}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO scenarios (author_id, title, description, content)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, author_id, title, description, version, status, content, created_at, updated_at`,
		authorID, title, description, content,
	).Scan(&sc.ID, &sc.AuthorID, &sc.Title, &sc.Description, &sc.Version, &sc.Status, &sc.Content, &sc.CreatedAt, &sc.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("scenario: create: %w", err)
	}
	return sc, nil
}

// ListByAuthor returns paginated scenarios for a given author.
func (r *Repository) ListByAuthor(ctx context.Context, authorID string, limit, offset int) ([]*Scenario, int, error) {
	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM scenarios WHERE author_id = $1`,
		authorID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("scenario: list count: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, author_id, title, description, version, status, content, created_at, updated_at
		 FROM scenarios WHERE author_id = $1
		 ORDER BY updated_at DESC
		 LIMIT $2 OFFSET $3`,
		authorID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("scenario: list: %w", err)
	}
	defer rows.Close()

	var scenarios []*Scenario
	for rows.Next() {
		sc := &Scenario{}
		if err := rows.Scan(&sc.ID, &sc.AuthorID, &sc.Title, &sc.Description, &sc.Version, &sc.Status, &sc.Content, &sc.CreatedAt, &sc.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scenario: list scan: %w", err)
		}
		scenarios = append(scenarios, sc)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("scenario: list rows: %w", err)
	}

	return scenarios, total, nil
}

// GetByID returns a scenario by its ID.
func (r *Repository) GetByID(ctx context.Context, id string) (*Scenario, error) {
	sc := &Scenario{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, author_id, title, description, version, status, content, created_at, updated_at
		 FROM scenarios WHERE id = $1`,
		id,
	).Scan(&sc.ID, &sc.AuthorID, &sc.Title, &sc.Description, &sc.Version, &sc.Status, &sc.Content, &sc.CreatedAt, &sc.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("scenario: get: %w", apperror.ErrNotFound)
		}
		return nil, fmt.Errorf("scenario: get: %w", err)
	}
	return sc, nil
}

// Update updates a scenario's title, description, and content.
func (r *Repository) Update(ctx context.Context, id, title, description string, content json.RawMessage) (*Scenario, error) {
	sc := &Scenario{}
	err := r.pool.QueryRow(ctx,
		`UPDATE scenarios SET title = $2, description = $3, content = $4, updated_at = NOW()
		 WHERE id = $1
		 RETURNING id, author_id, title, description, version, status, content, created_at, updated_at`,
		id, title, description, content,
	).Scan(&sc.ID, &sc.AuthorID, &sc.Title, &sc.Description, &sc.Version, &sc.Status, &sc.Content, &sc.CreatedAt, &sc.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("scenario: update: %w", apperror.ErrNotFound)
		}
		return nil, fmt.Errorf("scenario: update: %w", err)
	}
	return sc, nil
}

// Delete removes a scenario by ID.
func (r *Repository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM scenarios WHERE id = $1`,
		id,
	)
	if err != nil {
		return fmt.Errorf("scenario: delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("scenario: delete: %w", apperror.ErrNotFound)
	}
	return nil
}

// UpdateStatus transitions a scenario's status.
func (r *Repository) UpdateStatus(ctx context.Context, id, newStatus string) (*Scenario, error) {
	sc := &Scenario{}
	err := r.pool.QueryRow(ctx,
		`UPDATE scenarios SET status = $2, updated_at = NOW()
		 WHERE id = $1
		 RETURNING id, author_id, title, description, version, status, content, created_at, updated_at`,
		id, newStatus,
	).Scan(&sc.ID, &sc.AuthorID, &sc.Title, &sc.Description, &sc.Version, &sc.Status, &sc.Content, &sc.CreatedAt, &sc.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("scenario: update status: %w", apperror.ErrNotFound)
		}
		return nil, fmt.Errorf("scenario: update status: %w", err)
	}
	return sc, nil
}
