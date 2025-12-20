package service

import (
	models "LickLib/cmd/internal/entity"
	"LickLib/cmd/internal/repository"
)

type TrackReadService struct {
	repo repository.TrackRepository
}

func NewTrackService(r repository.TrackRepository) *TrackReadService {
	return &TrackReadService{repo: r}
}

func (s *TrackReadService) GetTrackByID(id uint) (*models.Track, error) {
	track, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	return track, nil
}

// interessante Funktion: GetTracksByUsername
func (s *TrackReadService) GetTracksByUsername(username string) ([]models.Track, error) {
	tracks, err := s.repo.FindByUsername(username)
	if err != nil {
		return nil, err
	}
	return tracks, nil
}
