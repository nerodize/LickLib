package repository

import (
	models "LickLib/cmd/internal/entity"
)

type TrackRepository interface {
	FindByID(id uint) (*models.Track, error)
	FindByUsername(username string) ([]models.Track, error)
	CreateTrack(track *models.Track) error
	DeleteTrack(id uint) error
	//UpdateTrack(track *models.Track) error
}
