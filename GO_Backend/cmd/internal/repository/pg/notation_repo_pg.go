package pg

import (
	models "LickLib/cmd/internal/entity"
	"LickLib/cmd/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NotationRepoGorm struct {
	db *gorm.DB
}

var _ repository.NotationRepository = &NotationRepoGorm{}

func NewNotationRepoGorm(db *gorm.DB) *NotationRepoGorm {
	return &NotationRepoGorm{db: db}
}

func (r *NotationRepoGorm) CreateNotation(notation *models.Notation) error {
	return r.db.Create(notation).Error
}

func (r *NotationRepoGorm) UpdateNotation(id uuid.UUID, updates map[string]interface{}) error {
	return r.db.Model(&models.Notation{}).Where("id = ?", id).Updates(updates).Error
}

func (r *NotationRepoGorm) DeleteNotation(id uuid.UUID) error {
	return r.db.Delete(&models.Notation{}, id).Error
}

func (r *NotationRepoGorm) FindByID(id uuid.UUID) (*models.Notation, error) {
	var notation models.Notation
	if err := r.db.Preload("Author").First(&notation, id).Error; err != nil {
		return nil, err
	}
	return &notation, nil
}

func (r *NotationRepoGorm) FindByTrackID(trackID uuid.UUID) ([]models.Notation, error) {
	var notations []models.Notation
	if err := r.db.Preload("Author").
		Where("track_id = ?", trackID).
		Order("created_at desc").
		Find(&notations).Error; err != nil {
		return nil, err
	}
	return notations, nil
}
