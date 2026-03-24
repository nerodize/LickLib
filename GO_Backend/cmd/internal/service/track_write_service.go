package service

import (
	models "LickLib/cmd/internal/entity"
	"LickLib/cmd/internal/repository"
	"LickLib/cmd/storage"
	"bytes"
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

	// TODO: should be refactored and should belong to minio.
	if err := s.validateTrack(data); err != nil {
		return err
	}

	//objName := GenerateUniqueName(data)

	trackID := uuid.New()
	objectName := fmt.Sprintf("users/%s/tracks/%s%s",
		data.UserID,
		trackID.String(),
		data.FileExt)

	if err := s.storage.Upload(ctx, objectName, file, size); err != nil {
		return err
	}

	trackEntity := &models.Track{
		ID:          trackID, // hat gefehlt
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
func (s *TrackWriteService) DeleteTrack(ctx context.Context, trackID uuid.UUID, userID uuid.UUID) error {

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

func (s *TrackWriteService) UpdateTrack(ctx context.Context, trackID uuid.UUID, userID uuid.UUID, req UpdateTrackRequest) error {
	track, err := s.repo.FindByID(trackID)
	if err != nil {
		return err
	}
	if track.UserID != userID {
		return errors.New("nicht autorisiert")
	}

	updates := make(map[string]interface{})
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}

	return s.repo.UpdateTrack(trackID, updates)
}

func GenerateUniqueName(metadata TrackMetadata) string {
	newID := uuid.New().String()

	ext := metadata.FileExt
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}

	// Ergebnis: "550e8400-e29b-11d4-a716-446655440000.mp3 || wav"
	return fmt.Sprintf("%s%s", newID, ext)
}

func (s *TrackWriteService) validateTrack(data TrackMetadata) error {
	if strings.TrimSpace(data.Title) == "" {
		return errors.New("title cannot be empty")
	}

	if len(data.Title) > 200 {
		return errors.New("title too long (max 200 chars)")
	}

	// Description
	if len(strings.TrimSpace(data.Description)) < 10 {
		return errors.New("description must be at least 10 characters")
	}
	if len(data.Description) > 2000 {
		return errors.New("description too long (max 2000 chars)")
	}

	// Difficulty (optional field)
	if data.Difficulty != "" {
		validDifficulties := map[string]bool{
			"EASY": true, "MEDIUM": true, "HARD": true, "GOGGINS": true,
		}
		if !validDifficulties[strings.ToUpper(data.Difficulty)] {
			return errors.New("invalid difficulty (must be EASY/MEDIUM/HARD/GOGGINS)")
		}
	}

	allowedExtensions := map[string]bool{".mp3": true, ".wav": true, ".flac": true}
	if !allowedExtensions[strings.ToLower(data.FileExt)] { // schöner code imo
		return fmt.Errorf("file type %s is not supported", data.FileExt)
	}

	return nil
}

func (s *TrackWriteService) validateAudioFile(file io.Reader, size int64) error {
	// Size (nochmal, Defense in Depth)
	const maxSize = 100 * 1024 * 1024 // 100MB
	if size > maxSize {
		return errors.New("file exceeds maximum size")
	}

	// Magic Bytes Check
	header := make([]byte, 12)
	n, err := file.Read(header)
	if err != nil || n < 12 {
		return errors.New("cannot read file header")
	}

	// MP3
	if bytes.HasPrefix(header, []byte("ID3")) {
		return nil // Valid MP3 with ID3 tag
	}
	if header[0] == 0xFF && (header[1]&0xE0) == 0xE0 {
		return nil // Valid MP3 MPEG frame
	}

	// WAV
	if bytes.HasPrefix(header, []byte("RIFF")) &&
		bytes.Contains(header[8:12], []byte("WAVE")) {
		return nil
	}

	// FLAC
	if bytes.HasPrefix(header, []byte("fLaC")) {
		return nil
	}

	return errors.New("not a valid audio file (MP3/WAV/FLAC)")
}
