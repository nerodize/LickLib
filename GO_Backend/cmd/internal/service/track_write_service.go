package service

import (
	models "LickLib/cmd/internal/entity"
	"LickLib/cmd/internal/repository"
	"LickLib/cmd/storage"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
)

type TrackWriteService struct {
	storage *storage.MinioClient
	repo    repository.TrackRepository
}

// billiges DTO => auslagern?
type TrackMetadata struct {
	Title       string
	Description string
	UserID      uuid.UUID
	Difficulty  string
	FileExt     string
}

type UpdateTrackRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
}

func NewTrackWriteService(s *storage.MinioClient, r repository.TrackRepository) *TrackWriteService {
	return &TrackWriteService{storage: s, repo: r}
}

// could be seen as create
func (s *TrackWriteService) UploadTrack(ctx context.Context, file io.Reader, size int64, data TrackMetadata) error {
	objectName := GenerateUniqueName(data)

	// MinIO Upload...
	if err := s.storage.Upload(ctx, objectName, file, size); err != nil {
		return err
	}

	// MAPPER: DTO -> Entity
	trackEntity := &models.Track{
		Title:       data.Title,
		Description: data.Description,
		UserID:      data.UserID,
		FileExt:     data.FileExt,
		SizeBytes:   size,
		StorageKey:  objectName, // Minio ID
	}

	// und hier der split für die DB => siehe hier mit create
	return s.repo.CreateTrack(trackEntity)
}

// hier dann noch die Funktion zum Track löschen
func (s *TrackWriteService) DeleteTrack(ctx context.Context, trackID uint, userID uuid.UUID) error {

	track, err := s.repo.FindByID(trackID)
	if err != nil {
		return err
	}

	if track.UserID != userID {
		return errors.New("Authorization ERROR, not owner of the Track")
	}

	if err := s.storage.Delete(ctx, track.StorageKey); err != nil {
		return fmt.Errorf("failed to delete file from storage: %w", err)
	}

	return s.repo.DeleteTrack(trackID)
}

func (s *TrackWriteService) UpdateTrack(ctx context.Context, trackID uint, userID uuid.UUID, req UpdateTrackRequest) error {
	// 1. Track laden & Owner checken
	track, err := s.repo.FindByID(trackID)
	if err != nil {
		return err
	}
	if track.UserID != userID {
		return errors.New("nicht autorisiert")
	}

	// 2. Nur die Felder vorbereiten, die wirklich geändert werden sollen
	updates := make(map[string]interface{})
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}

	// 3. In der DB speichern
	return s.repo.UpdateTrack(trackID, updates)
}

// macht wohl eher nicht viel Sinn, müsste noch ggf. ByID gelöscht werden.

func GenerateUniqueName(metadata TrackMetadata) string {
	// Erstellt eine ID wie: 550e8400-e29b-11d4-a716-446655440000
	newID := uuid.New().String()

	// Wir nehmen die Endung vom Original (z.B. .mp3)
	ext := metadata.FileExt
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}

	// Ergebnis: "550e8400-e29b-11d4-a716-446655440000.mp3"
	return fmt.Sprintf("%s%s", newID, ext)
}
