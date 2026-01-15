package models

import "time"

// --- models/track.go ---
// models/track.go

// mostly not neccessary: https://gorm.io/docs/belongs_to.html
type Track struct {
	ID     int   `gorm:"column:id;primaryKey;autoIncrement"`
	UserID int   `gorm:"column:user_id;not null;index"`
	User   *User `gorm:"foreignKey:UserID;references:ID"`

	Title       string      `gorm:"column:title;type:text;not null"`
	Description string      `gorm:"column:description;type:text;not null"`
	Difficulty  *Difficulty `gorm:"column:difficulty;type:difficulty"` // oder :text, falls ENUM noch nicht da
	FileExt     string      `gorm:"column:file_ext;type:text;not null"`
	SizeBytes   int64       `gorm:"column:size_bytes;not null"`
	StorageKey  string      `gorm:"column:storage_key;type:text;not null"`

	CreatedAt time.Time `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;default:now()"`

	Notations []Notation `gorm:"foreignKey:TrackID;references:ID"`
}

func (Track) TableName() string { return "tracks" }
