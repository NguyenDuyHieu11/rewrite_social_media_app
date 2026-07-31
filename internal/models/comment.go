package models

import (
	"time"

	"github.com/google/uuid"
)

// Comment is a nested reply on a post. Top-level comments have ParentID nil,
// Depth 0, and an empty Path. Replies set ParentID and build Path as the
// ancestor chain from root to parent (materialized path pattern).
//
// DeletedAt is set for soft deletes; live clients receive comment_deleted
// over SSE (Phase C). List queries should filter deleted_at IS NULL unless
// moderating.
type Comment struct {
	ID        uuid.UUID   `json:"id"`
	PostID    uuid.UUID   `json:"post_id"`
	AuthorID  uuid.UUID   `json:"author_id"`
	ParentID  *uuid.UUID  `json:"parent_id,omitempty"`
	Body      string      `json:"body"`
	Depth     int16       `json:"depth"`
	Path      []uuid.UUID `json:"path,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
	DeletedAt *time.Time  `json:"deleted_at,omitempty"`
}
