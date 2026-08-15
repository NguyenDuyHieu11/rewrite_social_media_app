package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/models"
	"gorm.io/gorm"
)

type CreateReportTypeInput struct {
	CategoryName     string
	Name             string
	Importance       int
	DeadlineDuration int
}

type UpdateReportTypeInput struct {
	CategoryName     string
	Name             string
	Importance       int
	DeadlineDuration int
}

type ReportTypeRepository interface {
	Create(ctx context.Context, input CreateReportTypeInput) (models.ReportType, error)
	GetByID(ctx context.Context, id uint) (models.ReportType, error)
	List(ctx context.Context, limit, offset int) ([]models.ReportType, error)
	Update(ctx context.Context, id uint, input UpdateReportTypeInput) (models.ReportType, error)
	Delete(ctx context.Context, id uint) error
}

type reportTypeRepo struct {
	db *gorm.DB
}

func NewReportTypeRepository(db *gorm.DB) ReportTypeRepository {
	return &reportTypeRepo{db: db}
}

func (r *reportTypeRepo) Create(ctx context.Context, input CreateReportTypeInput) (models.ReportType, error) {
	row := models.ReportType{
		CategoryName:     input.CategoryName,
		Name:             input.Name,
		Importance:       input.Importance,
		DeadlineDuration: input.DeadlineDuration,
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return models.ReportType{}, fmt.Errorf("create report type: %w", err)
	}
	return row, nil
}

func (r *reportTypeRepo) GetByID(ctx context.Context, id uint) (models.ReportType, error) {
	var row models.ReportType
	err := r.db.WithContext(ctx).First(&row, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.ReportType{}, ErrNotFound
		}
		return models.ReportType{}, fmt.Errorf("get report type by id %d: %w", id, err)
	}
	return row, nil
}

func (r *reportTypeRepo) List(ctx context.Context, limit, offset int) ([]models.ReportType, error) {
	if limit <= 0 {
		limit = 50
	}
	var rows []models.ReportType
	err := r.db.WithContext(ctx).Order("importance DESC, id DESC").Limit(limit).Offset(offset).Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list report types: %w", err)
	}
	return rows, nil
}

func (r *reportTypeRepo) Update(ctx context.Context, id uint, input UpdateReportTypeInput) (models.ReportType, error) {
	updates := map[string]any{
		"category_name":     input.CategoryName,
		"name":              input.Name,
		"importance":        input.Importance,
		"deadline_duration": input.DeadlineDuration,
	}
	q := r.db.WithContext(ctx).Model(&models.ReportType{}).Where("id = ?", id).Updates(updates)
	if q.Error != nil {
		return models.ReportType{}, fmt.Errorf("update report type %d: %w", id, q.Error)
	}
	if q.RowsAffected == 0 {
		return models.ReportType{}, ErrNotFound
	}
	return r.GetByID(ctx, id)
}

func (r *reportTypeRepo) Delete(ctx context.Context, id uint) error {
	q := r.db.WithContext(ctx).Delete(&models.ReportType{}, id)
	if q.Error != nil {
		return fmt.Errorf("delete report type %d: %w", id, q.Error)
	}
	if q.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
