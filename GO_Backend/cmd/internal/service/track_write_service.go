package service

import (
	models "LickLib/cmd/internal/entity"
	"LickLib/cmd/internal/repository"
	"LickLib/cmd/storage"
	"context"
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
		//StorageKey:  objectName, // Hier speichern wir die MinIO-ID
	}

	return s.repo.CreateTrack(trackEntity)
}

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
