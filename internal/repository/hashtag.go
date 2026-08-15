package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/models"
	"gorm.io/gorm"
)

type CreateHashtagInput struct {
	HashtagCategoryID *uint
	Name              string
	MatchingCategory  string
	PopularityScore   int
}

type UpdateHashtagInput struct {
	HashtagCategoryID *uint
	Name              string
	MatchingCategory  string
	PopularityScore   int
}

type HashtagRepository interface {
	Create(ctx context.Context, input CreateHashtagInput) (models.Hashtag, error)
	GetByID(ctx context.Context, id uint) (models.Hashtag, error)
	GetByName(ctx context.Context, name string) (models.Hashtag, error)
	List(ctx context.Context, limit, offset int) ([]models.Hashtag, error)
	Update(ctx context.Context, id uint, input UpdateHashtagInput) (models.Hashtag, error)
	Delete(ctx context.Context, id uint) error
}

type hashtagRepo struct {
	db *gorm.DB
}

func NewHashtagRepository(db *gorm.DB) HashtagRepository {
	return &hashtagRepo{db: db}
}

func (r *hashtagRepo) Create(ctx context.Context, input CreateHashtagInput) (models.Hashtag, error) {
	row := models.Hashtag{
		HashtagCategoryID: input.HashtagCategoryID,
		Name:              input.Name,
		MatchingCategory:  input.MatchingCategory,
		PopularityScore:   input.PopularityScore,
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return models.Hashtag{}, fmt.Errorf("create hashtag: %w", err)
	}
	return row, nil
}

func (r *hashtagRepo) GetByID(ctx context.Context, id uint) (models.Hashtag, error) {
	var row models.Hashtag
	err := r.db.WithContext(ctx).First(&row, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Hashtag{}, ErrNotFound
		}
		return models.Hashtag{}, fmt.Errorf("get hashtag by id %d: %w", id, err)
	}
	return row, nil
}

func (r *hashtagRepo) GetByName(ctx context.Context, name string) (models.Hashtag, error) {
	var row models.Hashtag
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Hashtag{}, ErrNotFound
		}
		return models.Hashtag{}, fmt.Errorf("get hashtag by name %q: %w", name, err)
	}
	return row, nil
}

func (r *hashtagRepo) List(ctx context.Context, limit, offset int) ([]models.Hashtag, error) {
	if limit <= 0 {
		limit = 50
	}
	var rows []models.Hashtag
	err := r.db.WithContext(ctx).Order("popularity_score DESC, id DESC").Limit(limit).Offset(offset).Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list hashtags: %w", err)
	}
	return rows, nil
}

func (r *hashtagRepo) Update(ctx context.Context, id uint, input UpdateHashtagInput) (models.Hashtag, error) {
	updates := map[string]any{
		"hashtag_category_id": input.HashtagCategoryID,
		"name":                input.Name,
		"matching_category":   input.MatchingCategory,
		"popularity_score":    input.PopularityScore,
	}
	q := r.db.WithContext(ctx).Model(&models.Hashtag{}).Where("id = ?", id).Updates(updates)
	if q.Error != nil {
		return models.Hashtag{}, fmt.Errorf("update hashtag %d: %w", id, q.Error)
	}
	if q.RowsAffected == 0 {
		return models.Hashtag{}, ErrNotFound
	}
	return r.GetByID(ctx, id)
}

func (r *hashtagRepo) Delete(ctx context.Context, id uint) error {
	q := r.db.WithContext(ctx).Delete(&models.Hashtag{}, id)
	if q.Error != nil {
		return fmt.Errorf("delete hashtag %d: %w", id, q.Error)
	}
	if q.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
