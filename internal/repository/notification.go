package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/models"
	"gorm.io/gorm"
)

type CreateNotificationInput struct {
	UserID             uint
	ContentType        string
	DirectedContentURL string
	RelatedUserID      *uint
	RelatedMessageID   *uint
	ReadStatus         bool
}

type UpdateNotificationInput struct {
	ContentType        string
	DirectedContentURL string
	RelatedUserID      *uint
	RelatedMessageID   *uint
	ReadStatus         bool
}

type NotificationRepository interface {
	Create(ctx context.Context, input CreateNotificationInput) (models.Notification, error)
	GetByID(ctx context.Context, id uint) (models.Notification, error)
	ListByUserID(ctx context.Context, userID uint, limit, offset int) ([]models.Notification, error)
	Update(ctx context.Context, id uint, input UpdateNotificationInput) (models.Notification, error)
	MarkRead(ctx context.Context, id uint) (models.Notification, error)
	Delete(ctx context.Context, id uint) error
}

type notificationRepo struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) NotificationRepository {
	return &notificationRepo{db: db}
}

func (r *notificationRepo) Create(ctx context.Context, input CreateNotificationInput) (models.Notification, error) {
	row := models.Notification{
		UserID:             input.UserID,
		ContentType:        input.ContentType,
		DirectedContentURL: input.DirectedContentURL,
		RelatedUserID:      input.RelatedUserID,
		RelatedMessageID:   input.RelatedMessageID,
		ReadStatus:         input.ReadStatus,
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return models.Notification{}, fmt.Errorf("create notification: %w", err)
	}
	return row, nil
}

func (r *notificationRepo) GetByID(ctx context.Context, id uint) (models.Notification, error) {
	var row models.Notification
	err := r.db.WithContext(ctx).First(&row, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Notification{}, ErrNotFound
		}
		return models.Notification{}, fmt.Errorf("get notification by id %d: %w", id, err)
	}
	return row, nil
}

func (r *notificationRepo) ListByUserID(ctx context.Context, userID uint, limit, offset int) ([]models.Notification, error) {
	if limit <= 0 {
		limit = 50
	}
	var rows []models.Notification
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("creation_date DESC").
		Limit(limit).
		Offset(offset).
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list notifications by user id %d: %w", userID, err)
	}
	return rows, nil
}

func (r *notificationRepo) Update(ctx context.Context, id uint, input UpdateNotificationInput) (models.Notification, error) {
	updates := map[string]any{
		"content_type":         input.ContentType,
		"directed_content_url": input.DirectedContentURL,
		"related_user_id":      input.RelatedUserID,
		"related_message_id":   input.RelatedMessageID,
		"read_status":          input.ReadStatus,
	}
	q := r.db.WithContext(ctx).Model(&models.Notification{}).Where("id = ?", id).Updates(updates)
	if q.Error != nil {
		return models.Notification{}, fmt.Errorf("update notification %d: %w", id, q.Error)
	}
	if q.RowsAffected == 0 {
		return models.Notification{}, ErrNotFound
	}
	return r.GetByID(ctx, id)
}

func (r *notificationRepo) MarkRead(ctx context.Context, id uint) (models.Notification, error) {
	q := r.db.WithContext(ctx).Model(&models.Notification{}).Where("id = ?", id).Update("read_status", true)
	if q.Error != nil {
		return models.Notification{}, fmt.Errorf("mark notification %d as read: %w", id, q.Error)
	}
	if q.RowsAffected == 0 {
		return models.Notification{}, ErrNotFound
	}
	return r.GetByID(ctx, id)
}

func (r *notificationRepo) Delete(ctx context.Context, id uint) error {
	q := r.db.WithContext(ctx).Delete(&models.Notification{}, id)
	if q.Error != nil {
		return fmt.Errorf("delete notification %d: %w", id, q.Error)
	}
	if q.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
