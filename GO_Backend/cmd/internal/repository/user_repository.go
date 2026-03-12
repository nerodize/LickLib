package repository

import (
	models "LickLib/cmd/internal/entity"

	"github.com/google/uuid"
)

type UserRepository interface {
	FindByID(id uuid.UUID) (*models.User, error)
	FindByUsername(username string) (*models.User, error)
	PreloadUserByID(userId uuid.UUID, user *models.User) error

	CreateUser(track *models.User) error
	DeleteUser(id uuid.UUID) error
	ExistsByUsernameOrEmail(username string, email string) (bool, error)
	// string: any
	UpdateUser(id uuid.UUID, updates map[string]interface{}) error
}
