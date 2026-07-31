package repository

import (
	"context"
	"fmt"

	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CreateCommentInput struct {
	PostID   uuid.UUID
	AuthorID uuid.UUID
	Body     string
}

type CommentsRepository interface {
	// Create inserts a top-level comment and bumps the post's denormalized
	// comment_count in the same transaction. Returns ErrNotFound if the post
	// does not exist or is soft-deleted.
	Create(ctx context.Context, input CreateCommentInput) (models.Comment, error)
}

type commentsRepo struct {
	pool *pgxpool.Pool
}

func NewCommentsRepository(pool *pgxpool.Pool) CommentsRepository {
	return &commentsRepo{pool: pool}
}

func (r *commentsRepo) Create(ctx context.Context, input CreateCommentInput) (models.Comment, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return models.Comment{}, fmt.Errorf("create comment: begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after Commit

	// Bumping the counter first doubles as the existence check: zero rows
	// means no live post, so we never insert an orphan comment.
	tag, err := tx.Exec(ctx, `
		UPDATE posts
		SET comment_count = comment_count + 1, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`, input.PostID)
	if err != nil {
		return models.Comment{}, fmt.Errorf("create comment: bump comment_count: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return models.Comment{}, ErrNotFound
	}

	var c models.Comment
	err = tx.QueryRow(ctx, `
		INSERT INTO comments (post_id, author_id, body)
		VALUES ($1, $2, $3)
		RETURNING id, post_id, author_id, parent_id, body, depth, path, created_at, updated_at
	`, input.PostID, input.AuthorID, input.Body).Scan(
		&c.ID,
		&c.PostID,
		&c.AuthorID,
		&c.ParentID,
		&c.Body,
		&c.Depth,
		&c.Path,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		return models.Comment{}, fmt.Errorf("create comment: insert: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Comment{}, fmt.Errorf("create comment: commit: %w", err)
	}
	return c, nil
}
