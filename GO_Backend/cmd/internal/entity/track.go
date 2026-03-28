package models

import (
	"time"

	"github.com/google/uuid"
)

// --- models/track.go ---
// models/track.go

// mostly not neccessary: https://gorm.io/docs/belongs_to.html
type Track struct {
	ID     uuid.UUID   `gorm:"column:id;primaryKey" json:"id"`
	Status TrackStatus `gorm:"column:status;type:track_status not null" json:"status"`
	UserID uuid.UUID   `gorm:"column:user_id;not null;index" json:"user_id"`
	User   *User       `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`

	Title       string      `gorm:"column:title;type:text;not null" json:"title"`
	Description string      `gorm:"column:description;type:text;not null" json:"description"`
	Difficulty  *Difficulty `gorm:"column:difficulty;type:difficulty" json:"difficulty"`

	FileExt   string `gorm:"column:file_ext;type:text;not null" json:"file_ext"`
	SizeBytes int64  `gorm:"column:size_bytes;not null" json:"size_bytes"`

	StorageKey string `gorm:"column:storage_key;type:text;not null" json:"-"`

	CreatedAt time.Time  `gorm:"column:created_at;not null;default:now()" json:"created_at"`
	UpdatedAt time.Time  `gorm:"column:updated_at;not null;default:now()" json:"updated_at"`
	Notations []Notation `gorm:"foreignKey:TrackID;references:ID" json:"notations,omitempty"`
}

func (Track) TableName() string { return "tracks" }
