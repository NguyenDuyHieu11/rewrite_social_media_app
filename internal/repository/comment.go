package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/models"
	"gorm.io/gorm"
)

type CreateCommentInput struct {
	UserID         uint
	ContentID      uint
	ReplyCommentID *uint
	CommentMessage string
}

type UpdateCommentInput struct {
	CommentMessage string
}

type CommentRepository interface {
	Create(ctx context.Context, input CreateCommentInput) (models.Comment, error)
	GetByID(ctx context.Context, id uint) (models.Comment, error)
	ListByContentID(ctx context.Context, contentID uint, limit, offset int) ([]models.Comment, error)
	Update(ctx context.Context, id uint, input UpdateCommentInput) (models.Comment, error)
	Delete(ctx context.Context, id uint) error
}

type commentRepo struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) CommentRepository {
	return &commentRepo{db: db}
}

func (r *commentRepo) Create(ctx context.Context, input CreateCommentInput) (models.Comment, error) {
	comment := models.Comment{
		UserID:         input.UserID,
		ContentID:      input.ContentID,
		ReplyCommentID: input.ReplyCommentID,
		CommentMessage: input.CommentMessage,
	}
	if err := r.db.WithContext(ctx).Create(&comment).Error; err != nil {
		return models.Comment{}, fmt.Errorf("create comment: %w", err)
	}
	return comment, nil
}

func (r *commentRepo) GetByID(ctx context.Context, id uint) (models.Comment, error) {
	var comment models.Comment
	err := r.db.WithContext(ctx).First(&comment, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Comment{}, ErrNotFound
		}
		return models.Comment{}, fmt.Errorf("get comment by id %d: %w", id, err)
	}
	return comment, nil
}

func (r *commentRepo) ListByContentID(ctx context.Context, contentID uint, limit, offset int) ([]models.Comment, error) {
	if limit <= 0 {
		limit = 30
	}

	var comments []models.Comment
	err := r.db.WithContext(ctx).
		Where("content_id = ?", contentID).
		Order("creation_date ASC").
		Limit(limit).
		Offset(offset).
		Find(&comments).Error
	if err != nil {
		return nil, fmt.Errorf("list comments by content id %d: %w", contentID, err)
	}
	return comments, nil
}

func (r *commentRepo) Update(ctx context.Context, id uint, input UpdateCommentInput) (models.Comment, error) {
	q := r.db.WithContext(ctx).
		Model(&models.Comment{}).
		Where("id = ?", id).
		Update("comment_message", input.CommentMessage)
	if q.Error != nil {
		return models.Comment{}, fmt.Errorf("update comment %d: %w", id, q.Error)
	}
	if q.RowsAffected == 0 {
		return models.Comment{}, ErrNotFound
	}

	return r.GetByID(ctx, id)
}

func (r *commentRepo) Delete(ctx context.Context, id uint) error {
	q := r.db.WithContext(ctx).Delete(&models.Comment{}, id)
	if q.Error != nil {
		return fmt.Errorf("delete comment %d: %w", id, q.Error)
	}
	if q.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
