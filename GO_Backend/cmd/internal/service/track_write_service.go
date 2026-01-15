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

// billiges DTO
type TrackMetadata struct {
	Title       string
	Description string
	UserID      int
	Difficulty  string
	FileExt     string
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
		StorageKey:  objectName, // Hier speichern wir die MinIO-ID TODO: wth
	}

	// und hier der split für die DB => siehe hier mit create
	return s.repo.CreateTrack(trackEntity)
}

// hier dann noch die Funktion zum Track löschen
func (s *TrackWriteService) DeleteTrack(ctx context.Context, trackID uint, userID int) error {

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
