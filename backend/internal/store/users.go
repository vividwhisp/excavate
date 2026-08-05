package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepo provides persistence for users.
type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

const userColumns = `id, email, password_hash, created_at`

func (r *UserRepo) Create(ctx context.Context, email, passwordHash string) (User, error) {
	u := User{}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING `+userColumns,
		email, passwordHash,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	return u, err
}

func (r *UserRepo) ByEmail(ctx context.Context, email string) (User, error) {
	u := User{}
	err := r.pool.QueryRow(ctx,
		`SELECT `+userColumns+` FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	return u, err
}

func (r *UserRepo) ByID(ctx context.Context, id string) (User, error) {
	u := User{}
	err := r.pool.QueryRow(ctx,
		`SELECT `+userColumns+` FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	return u, err
}