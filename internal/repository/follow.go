package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/models"
	"gorm.io/gorm"
)

type CreateFollowInput struct {
	FollowerID     uint
	FollowedUserID uint
	FollowStatus   string
}

type UpdateFollowInput struct {
	FollowStatus string
}

type FollowRepository interface {
	Create(ctx context.Context, input CreateFollowInput) (models.Follow, error)
	GetByID(ctx context.Context, id uint) (models.Follow, error)
	ListFollowers(ctx context.Context, followedUserID uint, limit, offset int) ([]models.Follow, error)
	ListFollowing(ctx context.Context, followerID uint, limit, offset int) ([]models.Follow, error)
	Update(ctx context.Context, id uint, input UpdateFollowInput) (models.Follow, error)
	Delete(ctx context.Context, id uint) error
}

type followRepo struct {
	db *gorm.DB
}

func NewFollowRepository(db *gorm.DB) FollowRepository {
	return &followRepo{db: db}
}

func (r *followRepo) Create(ctx context.Context, input CreateFollowInput) (models.Follow, error) {
	follow := models.Follow{
		FollowerID:     input.FollowerID,
		FollowedUserID: input.FollowedUserID,
		FollowStatus:   input.FollowStatus,
	}
	if err := r.db.WithContext(ctx).Create(&follow).Error; err != nil {
		return models.Follow{}, fmt.Errorf("create follow: %w", err)
	}
	return follow, nil
}

func (r *followRepo) GetByID(ctx context.Context, id uint) (models.Follow, error) {
	var follow models.Follow
	err := r.db.WithContext(ctx).First(&follow, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Follow{}, ErrNotFound
		}
		return models.Follow{}, fmt.Errorf("get follow by id %d: %w", id, err)
	}
	return follow, nil
}

func (r *followRepo) ListFollowers(ctx context.Context, followedUserID uint, limit, offset int) ([]models.Follow, error) {
	if limit <= 0 {
		limit = 50
	}

	var follows []models.Follow
	err := r.db.WithContext(ctx).
		Where("followed_user_id = ?", followedUserID).
		Order("follow_date DESC").
		Limit(limit).
		Offset(offset).
		Find(&follows).Error
	if err != nil {
		return nil, fmt.Errorf("list followers for user id %d: %w", followedUserID, err)
	}
	return follows, nil
}

func (r *followRepo) ListFollowing(ctx context.Context, followerID uint, limit, offset int) ([]models.Follow, error) {
	if limit <= 0 {
		limit = 50
	}

	var follows []models.Follow
	err := r.db.WithContext(ctx).
		Where("follower_id = ?", followerID).
		Order("follow_date DESC").
		Limit(limit).
		Offset(offset).
		Find(&follows).Error
	if err != nil {
		return nil, fmt.Errorf("list following for user id %d: %w", followerID, err)
	}
	return follows, nil
}

func (r *followRepo) Update(ctx context.Context, id uint, input UpdateFollowInput) (models.Follow, error) {
	q := r.db.WithContext(ctx).Model(&models.Follow{}).Where("id = ?", id).Update("follow_status", input.FollowStatus)
	if q.Error != nil {
		return models.Follow{}, fmt.Errorf("update follow %d: %w", id, q.Error)
	}
	if q.RowsAffected == 0 {
		return models.Follow{}, ErrNotFound
	}
	return r.GetByID(ctx, id)
}

func (r *followRepo) Delete(ctx context.Context, id uint) error {
	q := r.db.WithContext(ctx).Delete(&models.Follow{}, id)
	if q.Error != nil {
		return fmt.Errorf("delete follow %d: %w", id, q.Error)
	}
	if q.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
