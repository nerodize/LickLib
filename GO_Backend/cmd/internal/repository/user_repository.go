package repository

import (
	models "LickLib/cmd/internal/entity"
)

type UserRepository interface {
	FindByID(id uint) (*models.User, error)

	FindByUsername(username string) (*models.User, error)

	PreloadUserByID(userId uint, user *models.User) error
}
