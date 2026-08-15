package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/models"
	"gorm.io/gorm"
)

type CreateReportInput struct {
	ReportTypeID     uint
	ReporterID       uint
	Description      string
	RelatedUserID    *uint
	RelatedContentID *uint
	RelatedMessageID *uint
	Status           string
}

type UpdateReportInput struct {
	Description      string
	RelatedUserID    *uint
	RelatedContentID *uint
	RelatedMessageID *uint
	Status           string
}

type ReportRepository interface {
	Create(ctx context.Context, input CreateReportInput) (models.Report, error)
	GetByID(ctx context.Context, id uint) (models.Report, error)
	List(ctx context.Context, limit, offset int) ([]models.Report, error)
	Update(ctx context.Context, id uint, input UpdateReportInput) (models.Report, error)
	Delete(ctx context.Context, id uint) error
}

type reportRepo struct {
	db *gorm.DB
}

func NewReportRepository(db *gorm.DB) ReportRepository {
	return &reportRepo{db: db}
}

func (r *reportRepo) Create(ctx context.Context, input CreateReportInput) (models.Report, error) {
	row := models.Report{
		ReportTypeID:     input.ReportTypeID,
		ReporterID:       input.ReporterID,
		Description:      input.Description,
		RelatedUserID:    input.RelatedUserID,
		RelatedContentID: input.RelatedContentID,
		RelatedMessageID: input.RelatedMessageID,
		Status:           input.Status,
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return models.Report{}, fmt.Errorf("create report: %w", err)
	}
	return row, nil
}

func (r *reportRepo) GetByID(ctx context.Context, id uint) (models.Report, error) {
	var row models.Report
	err := r.db.WithContext(ctx).First(&row, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Report{}, ErrNotFound
		}
		return models.Report{}, fmt.Errorf("get report by id %d: %w", id, err)
	}
	return row, nil
}

func (r *reportRepo) List(ctx context.Context, limit, offset int) ([]models.Report, error) {
	if limit <= 0 {
		limit = 50
	}
	var rows []models.Report
	err := r.db.WithContext(ctx).Order("creation_date DESC").Limit(limit).Offset(offset).Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list reports: %w", err)
	}
	return rows, nil
}

func (r *reportRepo) Update(ctx context.Context, id uint, input UpdateReportInput) (models.Report, error) {
	updates := map[string]any{
		"description":        input.Description,
		"related_user_id":    input.RelatedUserID,
		"related_content_id": input.RelatedContentID,
		"related_message_id": input.RelatedMessageID,
		"status":             input.Status,
	}
	q := r.db.WithContext(ctx).Model(&models.Report{}).Where("id = ?", id).Updates(updates)
	if q.Error != nil {
		return models.Report{}, fmt.Errorf("update report %d: %w", id, q.Error)
	}
	if q.RowsAffected == 0 {
		return models.Report{}, ErrNotFound
	}
	return r.GetByID(ctx, id)
}

func (r *reportRepo) Delete(ctx context.Context, id uint) error {
	q := r.db.WithContext(ctx).Delete(&models.Report{}, id)
	if q.Error != nil {
		return fmt.Errorf("delete report %d: %w", id, q.Error)
	}
	if q.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
