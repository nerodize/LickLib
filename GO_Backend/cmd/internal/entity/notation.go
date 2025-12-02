package models

import "time"

type NotationType string

const (
	NotationTypeTabs  NotationType = "TABS"
	NotationTypeNotes NotationType = "NOTES"
)

type Notation struct {
	ID int `gorm:"column:id;primaryKey;autoIncrement"`

	TrackID int   `gorm:"column:track_id;not null;index"`
	Track   Track `gorm:"foreignKey:TrackID;references:ID"`

	AuthorID int  `gorm:"column:author_id;not null;index"`
	Author   User `gorm:"foreignKey:AuthorID;references:ID"`

	Type    NotationType `gorm:"column:type;type:text;not null"`    // "TABS" | "NOTES"
	Content string       `gorm:"column:content;type:text;not null"` // Tab oder Noten als Text/JSON

	FileExt   string `gorm:"column:file_ext;type:text;not null"`
	SizeBytes int64  `gorm:"column:size_bytes;not null"`

	CreatedAt time.Time `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;default:now()"`
}

func (Notation) TableName() string { return "notations" }
