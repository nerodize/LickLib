package repository

import (
	models "LickLib/cmd/internal/entity"

	"github.com/google/uuid"
)

type TrackRepository interface {
	FindByID(id uuid.UUID) (*models.Track, error)
	FindByUsername(username string) ([]models.Track, error)
	FindByUserID(userID uuid.UUID) ([]models.Track, error)
	CreateTrack(track *models.Track) error
	DeleteTrack(id uuid.UUID) error
	UpdateTrack(id uuid.UUID, updates map[string]interface{}) error

	DeleteFailedTracksByTitle(userID uuid.UUID, title string) error
}
