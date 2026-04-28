package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("resource not found")

type CreateUserInput struct {
	Username     string
	Email        string
	PasswordHash string
}

type UsersRepository interface {
	Create(ctx context.Context, input CreateUserInput) (models.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (models.User, error)
	GetByUsername(ctx context.Context, username string) (models.User, error)
}

type usersRepo struct {
	pool *pgxpool.Pool
}

func NewUsersRepository(pool *pgxpool.Pool) UsersRepository {
	return &usersRepo{pool: pool}
}

func (r *usersRepo) Create(ctx context.Context, input CreateUserInput) (models.User, error) {
	const q = `
		INSERT INTO users (username, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, username, email, password_hash, created_at, updated_at
	`

	var u models.User
	err := r.pool.QueryRow(ctx, q, input.Username, input.Email, input.PasswordHash).Scan(
		&u.ID,
		&u.Username,
		&u.Email,
		&u.PasswordHash,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		return models.User{}, fmt.Errorf("create user: %w", err)
	}

	return u, nil
}

func (r *usersRepo) GetByID(ctx context.Context, id uuid.UUID) (models.User, error) {
	const q = `
		SELECT id, username, email, password_hash, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var u models.User
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&u.ID,
		&u.Username,
		&u.Email,
		&u.PasswordHash,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, ErrNotFound
		}
		return models.User{}, fmt.Errorf("get user by id %s: %w", id, err)
	}

	return u, nil
}

func (r *usersRepo) GetByUsername(ctx context.Context, username string) (models.User, error) {
	const q = `
		SELECT id, username, email, password_hash, created_at, updated_at
		FROM users
		WHERE username = $1
	`

	var u models.User
	err := r.pool.QueryRow(ctx, q, username).Scan(
		&u.ID,
		&u.Username,
		&u.Email,
		&u.PasswordHash,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, ErrNotFound
		}
		return models.User{}, fmt.Errorf("get user by username %q: %w", username, err)
	}

	return u, nil
}
