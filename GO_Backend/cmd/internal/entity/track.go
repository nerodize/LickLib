package models

import "time"

// --- models/track.go ---
// models/track.go
type Track struct {
	ID          int         `gorm:"column:id;primaryKey;autoIncrement"`
	Username    string      `gorm:"column:username;uniqueIndex:idx_user_title"`
	Title       string      `gorm:"column:title;uniqueIndex:idx_user_title"`
	Description string      `gorm:"column:description;type:text;not null"`
	Difficulty  *Difficulty `gorm:"column:difficulty;type:difficulty"` // oder :text, falls ENUM noch nicht da
	FileExt     string      `gorm:"column:file_ext;type:text;not null"`
	SizeBytes   int64       `gorm:"column:size_bytes;not null"`
	CreatedAt   time.Time   `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt   time.Time   `gorm:"column:updated_at;not null;default:now()"`
	// Nur HIER das Constraint setzen:
	User *User `gorm:"foreignKey:Username;references:Username"`
}

func (Track) TableName() string { return "tracks" }
