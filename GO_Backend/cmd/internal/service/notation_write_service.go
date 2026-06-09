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

	if stale, err := s.notationRepo.FindFailedNotations(data.TrackID, data.AuthorID); err != nil {
		log.Printf("Warning: Could not find stale notations for track %s: %v", data.TrackID, err)
	} else {
		for _, n := range stale {
			if n.StorageKey != "" {
				if err := s.storage.Delete(ctx, n.StorageKey); err != nil {
					log.Printf("Warning: Could not delete stale notation %s from storage: %v", n.ID, err)
				}
			}
		}
		if err := s.notationRepo.DeleteFailedNotations(data.TrackID, data.AuthorID); err != nil {
			log.Printf("Warning: Could not delete stale notations from DB for track %s: %v", data.TrackID, err)
		}
	}

	notationID := uuid.New()
	notation := &models.Notation{
		ID:         notationID,
		TrackID:    data.TrackID,
		AuthorID:   data.AuthorID,
		Status:     models.NotationStatusUploading,
		Type:       models.NotationType(strings.ToUpper(data.Type)),
		StorageKey: "",
		FileExt:    data.FileExt,
		SizeBytes:  size,
		IsOfficial: data.AuthorID == track.UserID,
	}

	if err := s.notationRepo.CreateNotation(notation); err != nil {
		return fmt.Errorf("failed to create notation: %w", err)
	}

	objectName := generateNotationKey(data.AuthorID, notationID, notation.FileExt)

	if err := s.storage.Upload(ctx, objectName, file, size); err != nil {
		s.notationRepo.UpdateNotation(notationID, map[string]interface{}{
			"status": models.NotationStatusFailed,
		})
		return fmt.Errorf("storage upload failed: %w", err)
	}

	return s.notationRepo.UpdateNotation(notationID, map[string]interface{}{
		"status":      models.NotationStatusReady,
		"storage_key": objectName,
	})
}

func (s *NotationWriteService) DeleteNotation(ctx context.Context, notationID uuid.UUID, userID uuid.UUID) error {
	notation, err := s.notationRepo.FindByID(notationID)
	if err != nil {
		return err
	}

	track, err := s.trackRepo.FindByID(notation.TrackID)
	if err != nil {
		return err
	}

	if notation.AuthorID != userID && track.UserID != userID {
		return errors.New("not authorized")
	}

	if err := s.storage.Delete(ctx, notation.StorageKey); err != nil {
		return fmt.Errorf("storage delete failed: %w", err)
	}

	return s.notationRepo.DeleteNotation(notationID)
}

var allowedExtensions = map[string]bool{
	".pdf": true,
	".xml": true,
	".mxl": true,
	".gp":  true,
	".gpx": true,
	".gp5": true,
	".txt": true,
}

func validateNotationFile(file io.Reader, size int64, ext string) error {
	const maxSize = 20 * 1024 * 1024 // 20MB reicht für Notation
	if size > maxSize {
		return errors.New("file exceeds 20MB limit")
	}

	if !allowedExtensions[strings.ToLower(ext)] {
		return errors.New("invalid file type, allowed: pdf, xml, mxl, gp, gpx, gp5, txt")
	}

	// Magic Bytes nur für PDF und GP Formate
	header := make([]byte, 4)
	n, err := file.Read(header)
	if err != nil || n < 4 {
		return errors.New("cannot read file header")
	}

	if seeker, ok := file.(io.Seeker); ok { // "implementiert dieses interf. (file also reader) Seek?"
		seeker.Seek(0, io.SeekStart) // als Art reset sonst, unvollständig im Upload()...
	} else {
		return errors.New("file does not support seeking")
	}

	switch strings.ToLower(ext) {
	case ".pdf":
		// PDF fängt immer mit %PDF an
		if !bytes.HasPrefix(header, []byte("%PDF")) {
			return errors.New("invalid PDF file")
		}
	case ".gp", ".gpx", ".gp5":
		// Guitar Pro Dateien fangen mit FICHIER an (ältere Versionen)
		// oder mit einer Versionsnummer — hier reicht Extension-Check
	case ".xml", ".mxl", ".txt":
		// Textformate — Extension-Check reicht
	}

	return nil
}

func validateNotationMetadata(data NotationMetadata) error {
	// Type prüfen
	t := strings.ToUpper(data.Type)
	if t != string(models.NotationTypeTabs) && t != string(models.NotationTypeNotes) {
		return errors.New("invalid type, must be TABS or NOTES")
	}

	// FileExt prüfen
	if !allowedExtensions[strings.ToLower(data.FileExt)] {
		return errors.New("invalid file extension")
	}

	// Type/Extension Kombination prüfen
	if t == string(models.NotationTypeTabs) {
		tabExtensions := map[string]bool{
			".gp": true, ".gpx": true, ".gp5": true, ".txt": true,
		}
		if !tabExtensions[strings.ToLower(data.FileExt)] {
			return errors.New("TABS must be .gp/.gpx/.gp5/.txt")
		}
	}

	if t == string(models.NotationTypeNotes) {
		noteExtensions := map[string]bool{
			".pdf": true, ".xml": true, ".mxl": true,
		}
		if !noteExtensions[strings.ToLower(data.FileExt)] {
			return errors.New("NOTES must be .pdf/.xml/.mxl")
		}
	}

	return nil
}

func generateNotationKey(authorID, notationID uuid.UUID, ext string) string {
	return fmt.Sprintf("users/%s/notations/%s%s", authorID, notationID, ext)
}
