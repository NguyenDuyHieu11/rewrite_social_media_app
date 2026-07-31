package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/channels"
	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/events"
	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/models"
	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/pubsub"
	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/repository"
	"github.com/google/uuid"
)

// ErrPostNotFound is returned when the target post does not exist or is
// soft-deleted. Handlers map it to 404.
var ErrPostNotFound = errors.New("post not found")

// FeedService owns home-feed writes: persist to Postgres first, then publish
// a realtime event on the bus so gateways can fan out to subscribed clients.
//
// Persistence is the source of truth; publishing is best-effort. A failed
// publish is logged and the write still succeeds — clients that missed the
// live event see the data on their next fetch.
type FeedService struct {
	posts     repository.PostsRepository
	comments  repository.CommentsRepository
	reactions repository.ReactionsRepository
	bus       pubsub.PubSub
	log       *slog.Logger
}

func NewFeedService(
	posts repository.PostsRepository,
	comments repository.CommentsRepository,
	reactions repository.ReactionsRepository,
	bus pubsub.PubSub,
	log *slog.Logger,
) *FeedService {
	return &FeedService{posts: posts, comments: comments, reactions: reactions, bus: bus, log: log}
}

// CreatePost persists a new text post. No realtime event fires here: nobody
// can be subscribed to a post that did not exist yet. Feed/timeline delivery
// is a later phase.
func (s *FeedService) CreatePost(ctx context.Context, authorID uuid.UUID, body string) (models.Post, error) {
	post, err := s.posts.Create(ctx, repository.CreatePostInput{AuthorID: authorID, Body: body})
	if err != nil {
		return models.Post{}, fmt.Errorf("feed: create post: %w", err)
	}
	return post, nil
}

// GetPost returns one live post.
func (s *FeedService) GetPost(ctx context.Context, id uuid.UUID) (models.Post, error) {
	post, err := s.posts.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.Post{}, ErrPostNotFound
		}
		return models.Post{}, fmt.Errorf("feed: get post: %w", err)
	}
	return post, nil
}

// CreateComment persists a comment on postID, then publishes it to
// channels.PostChannel(postID) so every gateway with viewers on that post
// delivers it over SSE.
func (s *FeedService) CreateComment(ctx context.Context, postID, authorID uuid.UUID, body string) (models.Comment, error) {
	comment, err := s.comments.Create(ctx, repository.CreateCommentInput{
		PostID:   postID,
		AuthorID: authorID,
		Body:     body,
	})
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.Comment{}, ErrPostNotFound
		}
		return models.Comment{}, fmt.Errorf("feed: create comment: %w", err)
	}

	s.publish(ctx, postID, events.TypeComment, comment)
	return comment, nil
}

// UpsertReaction persists a reaction on postID, then publishes it to
// channels.PostChannel(postID) so gateways with viewers deliver it over SSE.
func (s *FeedService) UpsertReaction(ctx context.Context, postID, userID uuid.UUID, typ models.ReactionType) (models.Reaction, error) {
	if !models.ValidReactionType(typ) {
		return models.Reaction{}, fmt.Errorf("feed: upsert reaction: invalid type %q", typ)
	}

	reaction, err := s.reactions.Upsert(ctx, repository.UpsertReactionInput{
		PostID: postID,
		UserID: userID,
		Type:   typ,
	})
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.Reaction{}, ErrPostNotFound
		}
		return models.Reaction{}, fmt.Errorf("feed: upsert reaction: %w", err)
	}

	s.publish(ctx, postID, events.TypeReaction, reaction)
	return reaction, nil
}

func (s *FeedService) publish(ctx context.Context, postID uuid.UUID, eventType string, data any) {
	payload, err := events.Marshal(eventType, data)
	if err != nil {
		s.log.Error("feed: marshal event failed", "type", eventType, "post_id", postID, "error", err)
		return
	}
	if err := s.bus.Publish(ctx, channels.PostChannel(postID.String()), payload); err != nil {
		s.log.Error("feed: publish event failed", "type", eventType, "post_id", postID, "error", err)
	}
}
