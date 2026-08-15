package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/models"
	"gorm.io/gorm"
)

type CreateHashtagCategoryInput struct {
	Name            string
	PopularityScore int
}

type UpdateHashtagCategoryInput struct {
	Name            string
	PopularityScore int
}

type HashtagCategoryRepository interface {
	Create(ctx context.Context, input CreateHashtagCategoryInput) (models.HashtagCategory, error)
	GetByID(ctx context.Context, id uint) (models.HashtagCategory, error)
	List(ctx context.Context, limit, offset int) ([]models.HashtagCategory, error)
	Update(ctx context.Context, id uint, input UpdateHashtagCategoryInput) (models.HashtagCategory, error)
	Delete(ctx context.Context, id uint) error
}

type hashtagCategoryRepo struct {
	db *gorm.DB
}

func NewHashtagCategoryRepository(db *gorm.DB) HashtagCategoryRepository {
	return &hashtagCategoryRepo{db: db}
}

func (r *hashtagCategoryRepo) Create(ctx context.Context, input CreateHashtagCategoryInput) (models.HashtagCategory, error) {
	row := models.HashtagCategory{Name: input.Name, PopularityScore: input.PopularityScore}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return models.HashtagCategory{}, fmt.Errorf("create hashtag category: %w", err)
	}
	return row, nil
}

func (r *hashtagCategoryRepo) GetByID(ctx context.Context, id uint) (models.HashtagCategory, error) {
	var row models.HashtagCategory
	err := r.db.WithContext(ctx).First(&row, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.HashtagCategory{}, ErrNotFound
		}
		return models.HashtagCategory{}, fmt.Errorf("get hashtag category by id %d: %w", id, err)
	}
	return row, nil
}

func (r *hashtagCategoryRepo) List(ctx context.Context, limit, offset int) ([]models.HashtagCategory, error) {
	if limit <= 0 {
		limit = 50
	}
	var rows []models.HashtagCategory
	err := r.db.WithContext(ctx).Order("id DESC").Limit(limit).Offset(offset).Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list hashtag categories: %w", err)
	}
	return rows, nil
}

func (r *hashtagCategoryRepo) Update(ctx context.Context, id uint, input UpdateHashtagCategoryInput) (models.HashtagCategory, error) {
	updates := map[string]any{"name": input.Name, "popularity_score": input.PopularityScore}
	q := r.db.WithContext(ctx).Model(&models.HashtagCategory{}).Where("id = ?", id).Updates(updates)
	if q.Error != nil {
		return models.HashtagCategory{}, fmt.Errorf("update hashtag category %d: %w", id, q.Error)
	}
	if q.RowsAffected == 0 {
		return models.HashtagCategory{}, ErrNotFound
	}
	return r.GetByID(ctx, id)
}

func (r *hashtagCategoryRepo) Delete(ctx context.Context, id uint) error {
	q := r.db.WithContext(ctx).Delete(&models.HashtagCategory{}, id)
	if q.Error != nil {
		return fmt.Errorf("delete hashtag category %d: %w", id, q.Error)
	}
	if q.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
