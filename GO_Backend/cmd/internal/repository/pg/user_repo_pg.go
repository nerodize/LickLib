package pg

import (
	models "LickLib/cmd/internal/entity"
	"LickLib/cmd/internal/repository"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepoGorm struct {
	db *gorm.DB
}

// Explizite Prüfung ob das Interface
var _ repository.UserRepository = &UserRepoGorm{}

// NewUserRepoGorm erstellt eine neue Instanz
func NewUserRepoGorm(db *gorm.DB) *UserRepoGorm {
	return &UserRepoGorm{db: db}
}

func (r *UserRepoGorm) FindByID(id uuid.UUID) (*models.User, error) {
	var user models.User
	if err := r.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// wichtig sonst werden die Tabs und Tracks nicht mitgeschickt
func (r *UserRepoGorm) PreloadUserByID(id uuid.UUID, user *models.User) error {
	return r.db.Preload("Tracks").Preload("Notations").First(user, id).Error
}

// FindByUsername implementiert repo.UserRepository
func (r *UserRepoGorm) FindByUsername(username string) (*models.User, error) {
	username = strings.TrimSpace(username) // Leerzeichen / Newline entfernen
	var user models.User
	if err := r.db.Preload("Tracks").Preload("Notations").
		Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepoGorm) CreateUser(user *models.User) error {
	return r.db.Create(user).Error
}

func (r *UserRepoGorm) DeleteUser(id uuid.UUID) error {
	return r.db.Delete(&models.User{}, id).Error
}

func (r *UserRepoGorm) UpdateUser(id uuid.UUID, updates map[string]interface{}) error {
	return r.db.Model(&models.User{}).Where("id = ?", id).Updates(updates).Error
}
