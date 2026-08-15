package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/models"
	"gorm.io/gorm"
)

type CreateLikedInput struct {
	UserID          uint
	LikedContentID  *uint
	LikedCommentID  *uint
	HashtagID       *uint
	TimeSpent       int
	ActionFrequency int
}

type UpdateLikedInput struct {
	TimeSpent       int
	ActionFrequency int
}

type LikedRepository interface {
	Create(ctx context.Context, input CreateLikedInput) (models.Liked, error)
	GetByID(ctx context.Context, id uint) (models.Liked, error)
	ListByUserID(ctx context.Context, userID uint, limit, offset int) ([]models.Liked, error)
	Update(ctx context.Context, id uint, input UpdateLikedInput) (models.Liked, error)
	Delete(ctx context.Context, id uint) error
}

type likedRepo struct {
	db *gorm.DB
}

func NewLikedRepository(db *gorm.DB) LikedRepository {
	return &likedRepo{db: db}
}

func (r *likedRepo) Create(ctx context.Context, input CreateLikedInput) (models.Liked, error) {
	liked := models.Liked{
		UserID:          input.UserID,
		LikedContentID:  input.LikedContentID,
		LikedCommentID:  input.LikedCommentID,
		HashtagID:       input.HashtagID,
		TimeSpent:       input.TimeSpent,
		ActionFrequency: input.ActionFrequency,
	}
	if err := r.db.WithContext(ctx).Create(&liked).Error; err != nil {
		return models.Liked{}, fmt.Errorf("create liked: %w", err)
	}
	return liked, nil
}

func (r *likedRepo) GetByID(ctx context.Context, id uint) (models.Liked, error) {
	var liked models.Liked
	err := r.db.WithContext(ctx).First(&liked, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Liked{}, ErrNotFound
		}
		return models.Liked{}, fmt.Errorf("get liked by id %d: %w", id, err)
	}
	return liked, nil
}

func (r *likedRepo) ListByUserID(ctx context.Context, userID uint, limit, offset int) ([]models.Liked, error) {
	if limit <= 0 {
		limit = 50
	}

	var likes []models.Liked
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("creation_date DESC").
		Limit(limit).
		Offset(offset).
		Find(&likes).Error
	if err != nil {
		return nil, fmt.Errorf("list likes by user id %d: %w", userID, err)
	}
	return likes, nil
}

func (r *likedRepo) Update(ctx context.Context, id uint, input UpdateLikedInput) (models.Liked, error) {
	updates := map[string]any{
		"time_spent":       input.TimeSpent,
		"action_frequency": input.ActionFrequency,
	}

	q := r.db.WithContext(ctx).Model(&models.Liked{}).Where("id = ?", id).Updates(updates)
	if q.Error != nil {
		return models.Liked{}, fmt.Errorf("update liked %d: %w", id, q.Error)
	}
	if q.RowsAffected == 0 {
		return models.Liked{}, ErrNotFound
	}

	return r.GetByID(ctx, id)
}

func (r *likedRepo) Delete(ctx context.Context, id uint) error {
	q := r.db.WithContext(ctx).Delete(&models.Liked{}, id)
	if q.Error != nil {
		return fmt.Errorf("delete liked %d: %w", id, q.Error)
	}
	if q.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
