package service

import (
	models "LickLib/cmd/internal/entity"
	"LickLib/cmd/internal/repository"
	"LickLib/cmd/storage"
	"context"

	"github.com/google/uuid"
)

// notation_read_service.go
type NotationReadService struct {
	repo    repository.NotationRepository
	storage *storage.MinioClient
}

func NewNotationReadService(r repository.NotationRepository, s *storage.MinioClient) *NotationReadService {
	return &NotationReadService{
		repo:    r,
		storage: s,
	}
}

func (s *NotationReadService) GetByTrackID(trackID uuid.UUID) ([]models.Notation, error) {
	return s.repo.FindByTrackID(trackID)
}

func (s *NotationReadService) GetByID(notationID uuid.UUID) (*models.Notation, error) {
	return s.repo.FindByID(notationID)
}

func (s *NotationReadService) GetDownloadURL(ctx context.Context, notationID uuid.UUID) (string, error) {
	notation, err := s.repo.FindByID(notationID)
	if err != nil {
		return "", err
	}
	return s.storage.GetPresignedURL(ctx, notation.StorageKey)
}
