package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ThreadRepo provides persistence for threads.
type ThreadRepo struct {
	pool *pgxpool.Pool
}

func NewThreadRepo(pool *pgxpool.Pool) *ThreadRepo {
	return &ThreadRepo{pool: pool}
}

func (r *ThreadRepo) Create(ctx context.Context, userID, title string) (Thread, error) {
	t := Thread{}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO threads (user_id, title) VALUES ($1, $2)
		 RETURNING id, user_id, title, created_at, updated_at`,
		userID, title,
	).Scan(&t.ID, &t.UserID, &t.Title, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}

func (r *ThreadRepo) ByID(ctx context.Context, id, userID string) (Thread, error) {
	t := Thread{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, title, created_at, updated_at
		 FROM threads WHERE id = $1 AND user_id = $2`,
		id, userID,
	).Scan(&t.ID, &t.UserID, &t.Title, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}

func (r *ThreadRepo) ListByUser(ctx context.Context, userID string) ([]Thread, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, title, created_at, updated_at
		 FROM threads WHERE user_id = $1 ORDER BY updated_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	threads := []Thread{}
	for rows.Next() {
		var t Thread
		if err := rows.Scan(&t.ID, &t.UserID, &t.Title, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		threads = append(threads, t)
	}
	return threads, rows.Err()
}

func (r *ThreadRepo) Touch(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE threads SET updated_at = now() WHERE id = $1`, id)
	return err
}