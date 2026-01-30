package pg

import (
	models "LickLib/cmd/internal/entity"
	"LickLib/cmd/internal/repository"
	"strings"

	"gorm.io/gorm"
)

type UserRepoGorm struct {
	db *gorm.DB
}

var _ repository.UserRepository = &UserRepoGorm{}

// NewUserRepoGorm erstellt eine neue Instanz
func NewUserRepoGorm(db *gorm.DB) *UserRepoGorm {
	return &UserRepoGorm{db: db}
}

func (r *UserRepoGorm) FindByID(id uint) (*models.User, error) {
	var user models.User
	if err := r.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// wichtig sonst werden die Tabs und Tracks nicht mitgeschickt
func (r *UserRepoGorm) PreloadUserByID(id uint, user *models.User) error {
	return r.db.Preload("Tracks").Preload("Notations").First(user, id).Error
}

// FindByUsername implementiert repo.UserRepository
func (r *UserRepoGorm) FindByUsername(username string) (*models.User, error) {
	username = strings.TrimSpace(username) // Leerzeichen / Newline entfernen
	var user models.User
	// TODO: warum nochmal preload wichtig?
	if err := r.db.Preload("Tracks").Preload("Notations").
		Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepoGorm) CreateUser(user *models.User) error {
	return r.db.Create(user).Error
}
