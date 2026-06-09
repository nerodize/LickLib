package models

import (
	"database/sql/driver"
	"fmt"
)

type NotationStatus string

const (
	NotationStatusUploading NotationStatus = "UPLOADING"
	NotationStatusReady     NotationStatus = "READY"
	NotationStatusFailed    NotationStatus = "FAILED"
)

func (s *NotationStatus) Scan(value any) error {
	if value == nil {
		*s = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("notation_status scan: not a string")
	}
	*s = NotationStatus(str)
	return nil
}

func (s NotationStatus) Value() (driver.Value, error) {
	if s == "" {
		return nil, nil
	}
	return string(s), nil
}
