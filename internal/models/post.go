package models

import (
	"time"

	"github.com/google/uuid"
)

// Post is a home-feed content item. Media URL slices stay empty until media ships.
type Post struct {
	ID            uuid.UUID `json:"id"`
	AuthorID      uuid.UUID `json:"author_id"`
	Body          string    `json:"body"`
	ImageURLs     []string  `json:"image_urls,omitempty"`
	VideoURLs     []string  `json:"video_urls,omitempty"`
	CommentCount  int       `json:"comment_count"`
	ReactionCount int       `json:"reaction_count"`
	Version       int       `json:"version"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty"`
}

// PostVersion is one immutable snapshot of a post body after an edit.
// posts.version points at the latest generation; this table holds history
// for audit and optimistic-locking practice.
type PostVersion struct {
	ID        uuid.UUID `json:"id"`
	PostID    uuid.UUID `json:"post_id"`
	Body      string    `json:"body"`
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
}
