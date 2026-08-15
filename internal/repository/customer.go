package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/models"
	"gorm.io/gorm"
)

type CreateCustomerInput struct {
	Username string
	Password string
	Email    string
}

type CustomerRepository interface {
	Create(ctx context.Context, input CreateCustomerInput) (models.Customer, error)
	GetByID(ctx context.Context, id uint) (models.Customer, error)
	GetByUsername(ctx context.Context, username string) (models.Customer, error)
}

type customerRepo struct {
	db *gorm.DB
}

func NewCustomerRepository(db *gorm.DB) CustomerRepository {
	return &customerRepo{db: db}
}

// NewcusomerRepository keeps compatibility with earlier typo'd call sites.
func NewcusomerRepository(db *gorm.DB) CustomerRepository {
	return NewCustomerRepository(db)
}

func (r *customerRepo) Create(ctx context.Context, input CreateCustomerInput) (models.Customer, error) {
	customer := models.Customer{Username: input.Username, Password: input.Password, Email: input.Email}

	if err := r.db.WithContext(ctx).Create(&customer).Error; err != nil {
		return models.Customer{}, fmt.Errorf("create customer: %w", err)
	}

	return customer, nil
}

func (r *customerRepo) GetByID(ctx context.Context, id uint) (models.Customer, error) {
	var customer models.Customer
	err := r.db.WithContext(ctx).First(&customer, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Customer{}, ErrNotFound
		}
		return models.Customer{}, fmt.Errorf("get customer by id %d: %w", id, err)
	}

	return customer, nil
}

func (r *customerRepo) GetByUsername(ctx context.Context, username string) (models.Customer, error) {
	var customer models.Customer

	err := r.db.WithContext(ctx).Where("username = ?", username).First(&customer).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Customer{}, ErrNotFound
		}
		return models.Customer{}, fmt.Errorf("get customer by username %q: %w", username, err)
	}

	return customer, nil
}
