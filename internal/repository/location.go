package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/models"
	"gorm.io/gorm"
)

type CreateLocationInput struct {
	Combination string
	Country     string
	City        string
}

type UpdateLocationInput struct {
	Combination string
	Country     string
	City        string
}

type LocationRepository interface {
	Create(ctx context.Context, input CreateLocationInput) (models.Location, error)
	GetByID(ctx context.Context, id uint) (models.Location, error)
	List(ctx context.Context, limit, offset int) ([]models.Location, error)
	Update(ctx context.Context, id uint, input UpdateLocationInput) (models.Location, error)
	Delete(ctx context.Context, id uint) error
}

type locationRepo struct {
	db *gorm.DB
}

func NewLocationRepository(db *gorm.DB) LocationRepository {
	return &locationRepo{db: db}
}

func (r *locationRepo) Create(ctx context.Context, input CreateLocationInput) (models.Location, error) {
	row := models.Location{Combination: input.Combination, Country: input.Country, City: input.City}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return models.Location{}, fmt.Errorf("create location: %w", err)
	}
	return row, nil
}

func (r *locationRepo) GetByID(ctx context.Context, id uint) (models.Location, error) {
	var row models.Location
	err := r.db.WithContext(ctx).First(&row, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Location{}, ErrNotFound
		}
		return models.Location{}, fmt.Errorf("get location by id %d: %w", id, err)
	}
	return row, nil
}

func (r *locationRepo) List(ctx context.Context, limit, offset int) ([]models.Location, error) {
	if limit <= 0 {
		limit = 100
	}
	var rows []models.Location
	err := r.db.WithContext(ctx).Order("id DESC").Limit(limit).Offset(offset).Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list locations: %w", err)
	}
	return rows, nil
}

func (r *locationRepo) Update(ctx context.Context, id uint, input UpdateLocationInput) (models.Location, error) {
	updates := map[string]any{"combination": input.Combination, "country": input.Country, "city": input.City}
	q := r.db.WithContext(ctx).Model(&models.Location{}).Where("id = ?", id).Updates(updates)
	if q.Error != nil {
		return models.Location{}, fmt.Errorf("update location %d: %w", id, q.Error)
	}
	if q.RowsAffected == 0 {
		return models.Location{}, ErrNotFound
	}
	return r.GetByID(ctx, id)
}

func (r *locationRepo) Delete(ctx context.Context, id uint) error {
	q := r.db.WithContext(ctx).Delete(&models.Location{}, id)
	if q.Error != nil {
		return fmt.Errorf("delete location %d: %w", id, q.Error)
	}
	if q.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
