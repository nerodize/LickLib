package models

import (
	"time"

	"github.com/google/uuid"
)

type NotationType string

const (
	NotationTypeTabs  NotationType = "TABS"
	NotationTypeNotes NotationType = "NOTES"
)

type Notation struct {
	ID       uuid.UUID `gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	TrackID  uuid.UUID `gorm:"column:track_id;not null;index" json:"track_id"`
	AuthorID uuid.UUID `gorm:"column:author_id;not null;index" json:"author_id"`

	Type       NotationType `gorm:"column:type;type:text;not null" json:"type"`
	StorageKey string       `gorm:"column:storage_key;type:text;not null" json:"-"`
	FileExt    string       `gorm:"column:file_ext;type:text;not null" json:"file_ext"`
	SizeBytes  int64        `gorm:"column:size_bytes;not null" json:"size_bytes"`
	IsOfficial bool         `gorm:"column:is_official;default:false" json:"is_official"`

	CreatedAt time.Time `gorm:"column:created_at;not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;default:now()" json:"updated_at"`

	Track  *Track `gorm:"foreignKey:TrackID;references:ID" json:"track,omitempty"`
	Author *User  `gorm:"foreignKey:AuthorID;references:ID" json:"author,omitempty"`
}

func (Notation) TableName() string { return "notations" }
