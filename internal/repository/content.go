package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/models"
	"gorm.io/gorm"
)

type CreateContentInput struct {
	UserID       uint
	Type         string
	ContentURL   string
	SpanInfo     string
	TargetGender string
}

type UpdateContentInput struct {
	Type         string
	ContentURL   string
	SpanInfo     string
	TargetGender string
}

type ContentRepository interface {
	Create(ctx context.Context, input CreateContentInput) (models.Content, error)
	GetByID(ctx context.Context, id uint) (models.Content, error)
	ListByUserID(ctx context.Context, userID uint, limit, offset int) ([]models.Content, error)
	Update(ctx context.Context, id uint, input UpdateContentInput) (models.Content, error)
	Delete(ctx context.Context, id uint) error
}

type contentRepo struct {
	db *gorm.DB
}

func NewContentRepository(db *gorm.DB) ContentRepository {
	return &contentRepo{db: db}
}

func (r *contentRepo) Create(ctx context.Context, input CreateContentInput) (models.Content, error) {
	content := models.Content{
		UserID:       input.UserID,
		Type:         input.Type,
		ContentURL:   input.ContentURL,
		SpanInfo:     input.SpanInfo,
		TargetGender: input.TargetGender,
	}
	if err := r.db.WithContext(ctx).Create(&content).Error; err != nil {
		return models.Content{}, fmt.Errorf("create content: %w", err)
	}
	return content, nil
}

func (r *contentRepo) GetByID(ctx context.Context, id uint) (models.Content, error) {
	var content models.Content
	err := r.db.WithContext(ctx).First(&content, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Content{}, ErrNotFound
		}
		return models.Content{}, fmt.Errorf("get content by id %d: %w", id, err)
	}
	return content, nil
}

func (r *contentRepo) ListByUserID(ctx context.Context, userID uint, limit, offset int) ([]models.Content, error) {
	if limit <= 0 {
		limit = 20
	}

	var contents []models.Content
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("creation_date DESC").
		Limit(limit).
		Offset(offset).
		Find(&contents).Error
	if err != nil {
		return nil, fmt.Errorf("list content by user id %d: %w", userID, err)
	}
	return contents, nil
}

func (r *contentRepo) Update(ctx context.Context, id uint, input UpdateContentInput) (models.Content, error) {
	updates := map[string]any{
		"type":          input.Type,
		"content_url":   input.ContentURL,
		"span_info":     input.SpanInfo,
		"target_gender": input.TargetGender,
	}

	q := r.db.WithContext(ctx).Model(&models.Content{}).Where("id = ?", id).Updates(updates)
	if q.Error != nil {
		return models.Content{}, fmt.Errorf("update content %d: %w", id, q.Error)
	}
	if q.RowsAffected == 0 {
		return models.Content{}, ErrNotFound
	}

	return r.GetByID(ctx, id)
}

func (r *contentRepo) Delete(ctx context.Context, id uint) error {
	q := r.db.WithContext(ctx).Delete(&models.Content{}, id)
	if q.Error != nil {
		return fmt.Errorf("delete content %d: %w", id, q.Error)
	}
	if q.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
