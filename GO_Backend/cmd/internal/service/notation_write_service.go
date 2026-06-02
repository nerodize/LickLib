package service

import (
	models "LickLib/cmd/internal/entity"
	"LickLib/cmd/internal/repository"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/google/uuid"
)

type NotationWriteService struct {
	storage      StorageClient
	notationRepo repository.NotationRepository
	trackRepo    repository.TrackRepository
}

type NotationMetadata struct {
	TrackID  uuid.UUID
	AuthorID uuid.UUID
	Type     string
	FileExt  string
}

func NewNotationWriteService(s StorageClient, n repository.NotationRepository, t repository.TrackRepository) *NotationWriteService {
	return &NotationWriteService{
		storage:      s,
		notationRepo: n,
		trackRepo:    t,
	}
}

func (s *NotationWriteService) UploadNotation(ctx context.Context, file io.Reader, size int64, data NotationMetadata) error {
	track, err := s.trackRepo.FindByID(data.TrackID)
	if err != nil {
		return fmt.Errorf("track not found: %w", err)
	}
	if track.Status != models.TrackStatusReady {
		return errors.New("track is not ready")
	}

	notationID := uuid.New()
	notation := &models.Notation{
		ID:         notationID,
		TrackID:    data.TrackID,
		AuthorID:   data.AuthorID,
		Type:       models.NotationType(data.Type), // to upper and as pointer might be missing
		StorageKey: "",
		FileExt:    data.FileExt,
		SizeBytes:  size,
		IsOfficial: data.AuthorID == track.UserID,
	}

	if err := s.notationRepo.CreateNotation(notation); err != nil {
		return fmt.Errorf("failed to create notation: %w", err)
	}

	objectName := fmt.Sprintf("users/%s/notations/%s%s",
		data.AuthorID, notationID, data.FileExt)

	if err := s.storage.Upload(ctx, objectName, file, size); err != nil {
		s.notationRepo.DeleteNotation(notationID)
		return fmt.Errorf("storage upload failed: %w", err)
	}

	return s.notationRepo.UpdateNotation(notationID, map[string]interface{}{
		"storage_key": objectName,
	})
}

func (s *NotationWriteService) DeleteNotation(ctx context.Context, notationID uuid.UUID, userID uuid.UUID) error {
	notation, err := s.notationRepo.FindByID(notationID)
	if err != nil {
		return err
	}

	if notation.AuthorID != userID {
		return errors.New("not authorized")
	}

	if err := s.storage.Delete(ctx, notation.StorageKey); err != nil {
		return fmt.Errorf("storage delete failed: %w", err)
	}

	return s.notationRepo.DeleteNotation(notationID)
}
