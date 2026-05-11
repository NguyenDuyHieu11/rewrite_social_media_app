package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CreateRefreshTokenInput struct {
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
}

type RefreshTokensRepository interface {
	Create(ctx context.Context, input CreateRefreshTokenInput) (models.RefreshToken, error)
	GetByHash(ctx context.Context, tokenHash string) (models.RefreshToken, error)
	Revoke(ctx context.Context, id uuid.UUID, replacedBy *uuid.UUID) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
}

type refreshTokensRepo struct {
	pool *pgxpool.Pool
}

func NewRefreshTokensRepository(pool *pgxpool.Pool) RefreshTokensRepository {
	return &refreshTokensRepo{pool: pool}
}

func (r *refreshTokensRepo) Create(ctx context.Context, input CreateRefreshTokenInput) (models.RefreshToken, error) {
	const q = `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, token_hash, issued_at, expires_at, revoked_at, replaced_by
	`

	var t models.RefreshToken
	err := r.pool.QueryRow(ctx, q, input.UserID, input.TokenHash, input.ExpiresAt).Scan(
		&t.ID,
		&t.UserID,
		&t.TokenHash,
		&t.IssuedAt,
		&t.ExpiresAt,
		&t.RevokedAt,
		&t.ReplacedBy,
	)
	if err != nil {
		return models.RefreshToken{}, fmt.Errorf("create refresh token: %w", err)
	}

	return t, nil
}

func (r *refreshTokensRepo) GetByHash(ctx context.Context, tokenHash string) (models.RefreshToken, error) {
	const q = `
		SELECT id, user_id, token_hash, issued_at, expires_at, revoked_at, replaced_by
		FROM refresh_tokens
		WHERE token_hash = $1
	`

	var t models.RefreshToken
	err := r.pool.QueryRow(ctx, q, tokenHash).Scan(
		&t.ID,
		&t.UserID,
		&t.TokenHash,
		&t.IssuedAt,
		&t.ExpiresAt,
		&t.RevokedAt,
		&t.ReplacedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.RefreshToken{}, ErrNotFound
		}
		return models.RefreshToken{}, fmt.Errorf("get refresh token by hash: %w", err)
	}

	return t, nil
}

// Revoke marks a refresh token as revoked. If replacedBy is non-nil, it is
// stored as the rotation pointer so the chain can be audited.
func (r *refreshTokensRepo) Revoke(ctx context.Context, id uuid.UUID, replacedBy *uuid.UUID) error {
	const q = `
		UPDATE refresh_tokens
		SET revoked_at = NOW(), replaced_by = $2
		WHERE id = $1 AND revoked_at IS NULL
	`

	tag, err := r.pool.Exec(ctx, q, id, replacedBy)
	if err != nil {
		return fmt.Errorf("revoke refresh token %s: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// RevokeAllForUser revokes every active refresh token belonging to a user.
// Used on logout-all and on suspected token theft.
func (r *refreshTokensRepo) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	const q = `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL
	`

	if _, err := r.pool.Exec(ctx, q, userID); err != nil {
		return fmt.Errorf("revoke all refresh tokens for user %s: %w", userID, err)
	}
	return nil
}
