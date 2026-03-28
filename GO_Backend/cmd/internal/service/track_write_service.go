package service

import (
	models "LickLib/cmd/internal/entity"
	"LickLib/cmd/internal/repository"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/google/uuid"
)

type TrackWriteService struct {
	storage StorageClient
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

func NewTrackWriteService(s StorageClient, r repository.TrackRepository) *TrackWriteService {
	return &TrackWriteService{storage: s, repo: r}
}

// could be seen as create
func (s *TrackWriteService) UploadTrack(ctx context.Context, file io.Reader, size int64, data TrackMetadata) error {

	if err := s.validateMetadata(data); err != nil {
		return fmt.Errorf("metadata validation: %w", err)
	}

	if err := s.validateAudioFile(file, size); err != nil {
		return fmt.Errorf("file validation: %w", err)
	}

	if err := s.repo.DeleteFailedTracksByTitle(data.UserID, data.Title); err != nil {
		log.Printf("Warning: Could not cleanup old failed tracks for %s: %v", data.Title, data.UserID)
	}

	//objName := GenerateUniqueName(data)

	trackID := uuid.New()

	var difficultyPtr *models.Difficulty
	if data.Difficulty != "" {
		diff := models.Difficulty(strings.ToUpper(data.Difficulty))
		difficultyPtr = &diff
	}

	trackEntity := &models.Track{
		ID:          trackID, // hat gefehlt
		Title:       data.Title,
		Description: data.Description,
		UserID:      data.UserID,
		FileExt:     data.FileExt,
		SizeBytes:   size,
		Difficulty:  difficultyPtr,
		Status:      models.TrackStatusUploading,
		StorageKey:  "", // Minio ID
	}

	if err := s.repo.CreateTrack(trackEntity); err != nil {
		return fmt.Errorf("failed to create track: %w", err)
	}

	objectName := s.storage.GenerateTrackKey(data.UserID, trackID, data.FileExt)

	if err := s.storage.Upload(ctx, objectName, file, size); err != nil {
		// ROLLBACK: Status auf FAILED setzen
		s.repo.UpdateTrack(trackID, map[string]interface{}{
			"status": models.TrackStatusFailed,
		})
		return fmt.Errorf("storage upload failed: %w", err)
	}

	if err := s.repo.UpdateTrack(trackID, map[string]interface{}{
		"status":      models.TrackStatusReady,
		"storage_key": objectName,
	}); err != nil {
		// TODO: help
		// Hier haben wir ein Problem: File ist in MinIO, aber Status falsch
		// In Production: Queue für Cleanup-Job
		return fmt.Errorf("failed to update track status: %w", err)
	}

	return nil

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

// ===== VALIDIERUNGEN =====

func (s *TrackWriteService) validateMetadata(data TrackMetadata) error {
	// Title
	title := strings.TrimSpace(data.Title)
	if len(title) < 3 {
		return errors.New("title must be at least 3 characters")
	}
	if len(title) > 200 {
		return errors.New("title too long (max 200 chars)")
	}

	// Description
	desc := strings.TrimSpace(data.Description)
	if len(desc) < 10 {
		return errors.New("description must be at least 10 characters")
	}
	if len(desc) > 2000 {
		return errors.New("description too long (max 2000 chars)")
	}

	// Difficulty (optional)
	if data.Difficulty != "" {
		validDifficulties := map[string]bool{
			"EASY": true, "MEDIUM": true, "HARD": true, "GOGGINS": true,
		}
		if !validDifficulties[strings.ToUpper(data.Difficulty)] {
			return errors.New("invalid difficulty (must be EASY/MEDIUM/HARD/GOGGINS)")
		}
	}

	// Extension
	ext := strings.ToLower(data.FileExt)
	if ext != ".mp3" && ext != ".wav" && ext != ".flac" {
		return errors.New("invalid file extension (must be .mp3/.wav/.flac)")
	}

	return nil
}

func (s *TrackWriteService) validateAudioFile(file io.Reader, size int64) error {
	// Size-Check (Defense in Depth)
	const maxSize = 100 * 1024 * 1024 // 100MB
	if size > maxSize {
		return errors.New("file exceeds 100MB limit")
	}

	// Magic Bytes Check
	header := make([]byte, 12)
	n, err := file.Read(header)
	if err != nil || n < 12 {
		return errors.New("cannot read file header")
	}

	if seeker, ok := file.(io.Seeker); ok {
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			return fmt.Errorf("failed to reset file position: %w", err)
		}
	} else {
		return errors.New("file does not support seeking")
	}

	// MP3: ID3 tag oder MPEG frame sync
	if bytes.HasPrefix(header, []byte("ID3")) {
		return nil
	}
	if header[0] == 0xFF && (header[1]&0xE0) == 0xE0 {
		return nil
	}

	// WAV: RIFF...WAVE
	if bytes.HasPrefix(header, []byte("RIFF")) && bytes.Contains(header[8:12], []byte("WAVE")) {
		return nil
	}

	// FLAC: fLaC
	if bytes.HasPrefix(header, []byte("fLaC")) {
		return nil
	}

	return errors.New("not a valid audio file (MP3/WAV/FLAC expected)")
}
