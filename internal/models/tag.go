package models

import (
	"time"

	"github.com/google/uuid"
)

// Tag is a normalized hashtag label.
type Tag struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// PostTag links a post to a tag (many-to-many join table row).
type PostTag struct {
	PostID uuid.UUID `json:"post_id"`
	TagID  uuid.UUID `json:"tag_id"`
}
