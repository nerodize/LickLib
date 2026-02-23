package service

import (
	models "LickLib/cmd/internal/entity"
	"LickLib/cmd/internal/repository"
	"LickLib/cmd/storage"
	"context"

	"github.com/google/uuid"
)

type TrackReadService struct {
	repo    repository.TrackRepository
	storage *storage.MinioClient
}

func NewTrackService(r repository.TrackRepository, storage *storage.MinioClient) *TrackReadService {
	return &TrackReadService{
		repo:    r,
		storage: storage,
	}
}

func (s *TrackReadService) GetTrackByID(id uuid.UUID) (*models.Track, error) {
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

func (s *TrackReadService) GetPlaybackURL(ctx context.Context, trackID uuid.UUID) (string, error) {
	track, err := s.repo.FindByID(trackID)
	if err != nil {
		return "", err
	}

	// Die Logik für den Presigned Link liegt jetzt im Service/Storage-Layer
	return s.storage.GetPresignedURL(ctx, track.StorageKey)
}
