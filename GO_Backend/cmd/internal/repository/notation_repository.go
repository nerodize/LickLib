package repository

import (
	models "LickLib/cmd/internal/entity"

	"github.com/google/uuid"
)

type NotationRepository interface {
	CreateNotation(notation *models.Notation) error
	DeleteNotation(id uuid.UUID) error
	UpdateNotation(id uuid.UUID, updates map[string]interface{}) error
	FindByID(id uuid.UUID) (*models.Notation, error)
	FindByTrackID(id uuid.UUID) ([]models.Notation, error)
}
