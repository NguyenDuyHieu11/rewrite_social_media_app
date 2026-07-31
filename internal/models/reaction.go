package models

import (
	"time"

	"github.com/google/uuid"
)

// ReactionType is the kind of reaction on a post. Like is one type among several;
// each user may hold at most one reaction per post (upsert changes the type).
type ReactionType string

const (
	ReactionLike  ReactionType = "like"
	ReactionLove  ReactionType = "love"
	ReactionHaha  ReactionType = "haha"
	ReactionSad   ReactionType = "sad"
	ReactionAngry ReactionType = "angry"
)

// ValidReactionType reports whether t is allowed by the reactions.type CHECK.
func ValidReactionType(t ReactionType) bool {
	switch t {
	case ReactionLike, ReactionLove, ReactionHaha, ReactionSad, ReactionAngry:
		return true
	default:
		return false
	}
}

// Reaction is one user's reaction on a post. Composite identity is (PostID, UserID).
type Reaction struct {
	PostID    uuid.UUID    `json:"post_id"`
	UserID    uuid.UUID    `json:"user_id"`
	Type      ReactionType `json:"type"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}
