package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/internships-backend/test-backend-bober-17/internal/model"
)

type Repo struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

func (r *Repo) GetUserByEmail(ctx context.Context, email string) (model.User, error) {
	const q = `SELECT id, email, password, role, created_at FROM users WHERE email = $1`

	var u model.User
	err := r.pool.QueryRow(ctx, q, email).Scan(&u.ID, &u.Email, &u.Password, &u.Role, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.User{}, fmt.Errorf("get user by email: %w", model.ErrInvalidCredentials)
	}
	if err != nil {
		return model.User{}, fmt.Errorf("get user by email: %w", err)
	}

	return u, nil
}

func (r *Repo) CreateUser(ctx context.Context, email, passwordHash string, role model.Role) (model.User, error) {
	const q = `
		INSERT INTO users (email, password, role)
		VALUES ($1, $2, $3)
		RETURNING id, email, password, role, created_at`

	var u model.User
	err := r.pool.QueryRow(ctx, q, email, passwordHash, string(role)).Scan(
		&u.ID, &u.Email, &u.Password, &u.Role, &u.CreatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return model.User{}, fmt.Errorf("create user: %w", model.ErrEmailTaken)
		}

		return model.User{}, fmt.Errorf("create user: %w", err)
	}

	return u, nil
}
