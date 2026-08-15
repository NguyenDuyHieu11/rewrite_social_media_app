package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/models"
	"gorm.io/gorm"
)

type CreateConversationInput struct {
	FirstUserID         uint
	SecondUserID        uint
	LastMessageID       *uint
	LastMessageContent  string
	LastMessageSentTime *time.Time
}

type UpdateConversationInput struct {
	LastMessageID       *uint
	LastMessageContent  string
	LastMessageSentTime *time.Time
}

type ConversationRepository interface {
	Create(ctx context.Context, input CreateConversationInput) (models.Conversation, error)
	GetByID(ctx context.Context, id uint) (models.Conversation, error)
	ListByUserID(ctx context.Context, userID uint, limit, offset int) ([]models.Conversation, error)
	Update(ctx context.Context, id uint, input UpdateConversationInput) (models.Conversation, error)
	Delete(ctx context.Context, id uint) error
}

type conversationRepo struct {
	db *gorm.DB
}

func NewConversationRepository(db *gorm.DB) ConversationRepository {
	return &conversationRepo{db: db}
}

func (r *conversationRepo) Create(ctx context.Context, input CreateConversationInput) (models.Conversation, error) {
	row := models.Conversation{
		FirstUserID:         input.FirstUserID,
		SecondUserID:        input.SecondUserID,
		LastMessageID:       input.LastMessageID,
		LastMessageContent:  input.LastMessageContent,
		LastMessageSentTime: input.LastMessageSentTime,
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return models.Conversation{}, fmt.Errorf("create conversation: %w", err)
	}
	return row, nil
}

func (r *conversationRepo) GetByID(ctx context.Context, id uint) (models.Conversation, error) {
	var row models.Conversation
	err := r.db.WithContext(ctx).First(&row, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Conversation{}, ErrNotFound
		}
		return models.Conversation{}, fmt.Errorf("get conversation by id %d: %w", id, err)
	}
	return row, nil
}

func (r *conversationRepo) ListByUserID(ctx context.Context, userID uint, limit, offset int) ([]models.Conversation, error) {
	if limit <= 0 {
		limit = 50
	}
	var rows []models.Conversation
	err := r.db.WithContext(ctx).
		Where("first_user_id = ? OR second_user_id = ?", userID, userID).
		Order("updated_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list conversations by user id %d: %w", userID, err)
	}
	return rows, nil
}

func (r *conversationRepo) Update(ctx context.Context, id uint, input UpdateConversationInput) (models.Conversation, error) {
	updates := map[string]any{
		"last_message_id":        input.LastMessageID,
		"last_message_content":   input.LastMessageContent,
		"last_message_sent_time": input.LastMessageSentTime,
	}
	q := r.db.WithContext(ctx).Model(&models.Conversation{}).Where("id = ?", id).Updates(updates)
	if q.Error != nil {
		return models.Conversation{}, fmt.Errorf("update conversation %d: %w", id, q.Error)
	}
	if q.RowsAffected == 0 {
		return models.Conversation{}, ErrNotFound
	}
	return r.GetByID(ctx, id)
}

func (r *conversationRepo) Delete(ctx context.Context, id uint) error {
	q := r.db.WithContext(ctx).Delete(&models.Conversation{}, id)
	if q.Error != nil {
		return fmt.Errorf("delete conversation %d: %w", id, q.Error)
	}
	if q.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
