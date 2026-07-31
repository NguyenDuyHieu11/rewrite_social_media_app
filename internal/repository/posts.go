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

type CreatePostInput struct {
	AuthorID uuid.UUID
	Body     string
}

type PostsRepository interface {
	Create(ctx context.Context, input CreatePostInput) (models.Post, error)
	GetByID(ctx context.Context, id uuid.UUID) (models.Post, error)
}

type postsRepo struct {
	pool *pgxpool.Pool
}

func NewPostsRepository(pool *pgxpool.Pool) PostsRepository {
	return &postsRepo{pool: pool}
}

const postColumns = `id, author_id, body, image_urls, video_urls,
	comment_count, reaction_count, version, created_at, updated_at`

func scanPost(row pgx.Row) (models.Post, error) {
	var p models.Post
	err := row.Scan(
		&p.ID,
		&p.AuthorID,
		&p.Body,
		&p.ImageURLs,
		&p.VideoURLs,
		&p.CommentCount,
		&p.ReactionCount,
		&p.Version,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	return p, err
}

func (r *postsRepo) Create(ctx context.Context, input CreatePostInput) (models.Post, error) {
	q := fmt.Sprintf(`
		INSERT INTO posts (author_id, body)
		VALUES ($1, $2)
		RETURNING %s
	`, postColumns)

	p, err := scanPost(r.pool.QueryRow(ctx, q, input.AuthorID, input.Body))
	if err != nil {
		return models.Post{}, fmt.Errorf("create post: %w", err)
	}
	return p, nil
}

func (r *postsRepo) GetByID(ctx context.Context, id uuid.UUID) (models.Post, error) {
	q := fmt.Sprintf(`
		SELECT %s
		FROM posts
		WHERE id = $1 AND deleted_at IS NULL
	`, postColumns)

	p, err := scanPost(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Post{}, ErrNotFound
		}
		return models.Post{}, fmt.Errorf("get post by id %s: %w", id, err)
	}
	return p, nil
}
