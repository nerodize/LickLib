package repository

import (
	"context"

	"gorm.io/gorm"

	models "LickLib/cmd/internal/entity"
)

type UserRepo interface {
	Create(ctx context.Context, u *models.User) error
	ByUsername(ctx context.Context, username string) (*models.User, error)
}

type userRepo struct{ db *gorm.DB }

func NewUserRepo(db *gorm.DB) UserRepo { return &userRepo{db: db} }

func (r *userRepo) Create(ctx context.Context, u *models.User) error {
	return r.db.WithContext(ctx).Create(u).Error
}

func (r *userRepo) ByUsername(ctx context.Context, username string) (*models.User, error) {
	var u models.User
	if err := r.db.WithContext(ctx).First(&u, "username = ?", username).Error; err != nil {
		return nil, err
	}
	return &u, nil
}
