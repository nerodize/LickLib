package service

import (
	"context"
	"fmt"
	"time"

	models "LickLib/cmd/internal/entity"
	"LickLib/cmd/internal/repository"
	dto "LickLib/cmd/internal/service/dto"
)

type TrackWriteService struct {
	trackRepo repository.TrackRepository
	userRepo  repository.UserRepository
}

func NewTrackWriteService(r repository.TrackRepository) *TrackWriteService {
	if r == nil {
		panic("TrackWriteService: repo cannot be nil")
	}
	return &TrackWriteService{trackRepo: r}
}
func (s *TrackWriteService) CreateTrack(ctx context.Context, dto dto.TrackDTO) (*models.Track, error) {
	// Validierung: user existiert?
	if _, err := s.userRepo.FindByID(uint(dto.UserID)); err != nil {
		return nil, err
	}

	if s.trackRepo == nil {
		return nil, fmt.Errorf("track repository not initialized")
	}

	t := models.Track{
		UserID:      int(dto.UserID),
		Title:       dto.Title,
		Description: dto.Description,
		FileExt:     dto.FileExt,
		SizeBytes:   dto.SizeBytes,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
		FileURL:     dto.FileURL,
	}

	// Wenn dein models.Track ein FileURL-Feld hat:
	// t.FileURL = dto.FileURL

	if err := s.trackRepo.Create(&t); err != nil {
		return nil, err
	}
	return &t, nil
}
