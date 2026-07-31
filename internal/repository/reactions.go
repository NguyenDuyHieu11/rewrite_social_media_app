package repository

import (
	"context"
	"fmt"

	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UpsertReactionInput struct {
	PostID uuid.UUID
	UserID uuid.UUID
	Type   models.ReactionType
}

type ReactionsRepository interface {
	// Upsert inserts or updates one reaction per (post, user). On first insert
	// it bumps posts.reaction_count. Returns ErrNotFound if the post is missing
	// or soft-deleted.
	Upsert(ctx context.Context, input UpsertReactionInput) (models.Reaction, error)
}

type reactionsRepo struct {
	pool *pgxpool.Pool
}

func NewReactionsRepository(pool *pgxpool.Pool) ReactionsRepository {
	return &reactionsRepo{pool: pool}
}

func (r *reactionsRepo) Upsert(ctx context.Context, input UpsertReactionInput) (models.Reaction, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return models.Reaction{}, fmt.Errorf("upsert reaction: begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after Commit

	var exists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM posts WHERE id = $1 AND deleted_at IS NULL)
	`, input.PostID).Scan(&exists); err != nil {
		return models.Reaction{}, fmt.Errorf("upsert reaction: check post: %w", err)
	}
	if !exists {
		return models.Reaction{}, ErrNotFound
	}

	var reaction models.Reaction
	var inserted bool
	err = tx.QueryRow(ctx, `
		INSERT INTO reactions (post_id, user_id, type)
		VALUES ($1, $2, $3)
		ON CONFLICT (post_id, user_id) DO UPDATE
			SET type = EXCLUDED.type,
			    updated_at = NOW()
		RETURNING post_id, user_id, type, created_at, updated_at, (xmax = 0) AS inserted
	`, input.PostID, input.UserID, input.Type).Scan(
		&reaction.PostID,
		&reaction.UserID,
		&reaction.Type,
		&reaction.CreatedAt,
		&reaction.UpdatedAt,
		&inserted,
	)
	if err != nil {
		return models.Reaction{}, fmt.Errorf("upsert reaction: upsert: %w", err)
	}

	if inserted {
		if _, err := tx.Exec(ctx, `
			UPDATE posts
			SET reaction_count = reaction_count + 1, updated_at = NOW()
			WHERE id = $1 AND deleted_at IS NULL
		`, input.PostID); err != nil {
			return models.Reaction{}, fmt.Errorf("upsert reaction: bump reaction_count: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Reaction{}, fmt.Errorf("upsert reaction: commit: %w", err)
	}
	return reaction, nil
}
