package models

import (
	"database/sql/driver"
	"fmt"
)

type TrackStatus string

const (
	TrackStatusUploading  TrackStatus = "UPLOADING"
	TrackStatusReady      TrackStatus = "READY"
	TrackStatusFailed     TrackStatus = "FAILED"
	TrackStatusProcessing TrackStatus = "PROCESSING"
)

// Scan implementiert sql.Scanner (für GORM beim Lesen aus DB)
func (s *TrackStatus) Scan(value any) error {
	if value == nil {
		*s = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("track_status scan: not a string")
	}
	*s = TrackStatus(str)
	return nil
}

// Value implementiert driver.Valuer (für GORM beim Schreiben in DB)
func (s TrackStatus) Value() (driver.Value, error) {
	if s == "" {
		return nil, nil
	}
	return string(s), nil
}

// Helper für Pointer (wie bei Difficulty)
func PtrTrackStatus(v TrackStatus) *TrackStatus { return &v }
