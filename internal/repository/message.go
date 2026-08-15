package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/models"
	"gorm.io/gorm"
)

type CreateMessageInput struct {
	ConversationID uint
	SenderUserID   uint
	Type           string
	ContentURL     string
	MessageStatus  string
	RelatedHashtag string
}

type UpdateMessageInput struct {
	Type           string
	ContentURL     string
	MessageStatus  string
	RelatedHashtag string
}

type MessageRepository interface {
	Create(ctx context.Context, input CreateMessageInput) (models.Message, error)
	GetByID(ctx context.Context, id uint) (models.Message, error)
	ListByConversationID(ctx context.Context, conversationID uint, limit, offset int) ([]models.Message, error)
	Update(ctx context.Context, id uint, input UpdateMessageInput) (models.Message, error)
	Delete(ctx context.Context, id uint) error
}

type messageRepo struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) MessageRepository {
	return &messageRepo{db: db}
}

func (r *messageRepo) Create(ctx context.Context, input CreateMessageInput) (models.Message, error) {
	row := models.Message{
		ConversationID: input.ConversationID,
		SenderUserID:   input.SenderUserID,
		Type:           input.Type,
		ContentURL:     input.ContentURL,
		MessageStatus:  input.MessageStatus,
		RelatedHashtag: input.RelatedHashtag,
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return models.Message{}, fmt.Errorf("create message: %w", err)
	}
	return row, nil
}

func (r *messageRepo) GetByID(ctx context.Context, id uint) (models.Message, error) {
	var row models.Message
	err := r.db.WithContext(ctx).First(&row, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Message{}, ErrNotFound
		}
		return models.Message{}, fmt.Errorf("get message by id %d: %w", id, err)
	}
	return row, nil
}

func (r *messageRepo) ListByConversationID(ctx context.Context, conversationID uint, limit, offset int) ([]models.Message, error) {
	if limit <= 0 {
		limit = 100
	}
	var rows []models.Message
	err := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Order("creation_date ASC").
		Limit(limit).
		Offset(offset).
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list messages by conversation id %d: %w", conversationID, err)
	}
	return rows, nil
}

func (r *messageRepo) Update(ctx context.Context, id uint, input UpdateMessageInput) (models.Message, error) {
	updates := map[string]any{
		"type":            input.Type,
		"content_url":     input.ContentURL,
		"message_status":  input.MessageStatus,
		"related_hashtag": input.RelatedHashtag,
	}
	q := r.db.WithContext(ctx).Model(&models.Message{}).Where("id = ?", id).Updates(updates)
	if q.Error != nil {
		return models.Message{}, fmt.Errorf("update message %d: %w", id, q.Error)
	}
	if q.RowsAffected == 0 {
		return models.Message{}, ErrNotFound
	}
	return r.GetByID(ctx, id)
}

func (r *messageRepo) Delete(ctx context.Context, id uint) error {
	q := r.db.WithContext(ctx).Delete(&models.Message{}, id)
	if q.Error != nil {
		return fmt.Errorf("delete message %d: %w", id, q.Error)
	}
	if q.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
