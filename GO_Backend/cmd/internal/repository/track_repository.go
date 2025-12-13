package repository

import (
	models "LickLib/cmd/internal/entity"
)

type TrackRepository interface {
	FindByID(id uint) (*models.Track, error)
	//FIndByUserID()...
	FindByUsername(username string) ([]models.Track, error)
	Create(t *models.Track) error
}
