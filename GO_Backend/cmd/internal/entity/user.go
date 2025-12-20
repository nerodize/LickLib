package models

import (
	"time"
)

// --- models/user.go ---
type User struct {
	ID           int       `gorm:"column:id;primaryKey;autoIncrement"`
	Username     string    `gorm:"column:username;type:text;not null;uniqueIndex"`
	Email        *string   `gorm:"column:email;type:text;uniqueIndex"`
	PasswordHash string    `gorm:"column:password_hash;type:text;not null"`
	CreatedAt    time.Time `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt    time.Time `gorm:"column:updated_at;not null;default:now()"`
	// Has-many: KEIN constraint-Tag hier!
	Tracks    []Track    `gorm:"foreignKey:UserID;references:ID"`
	Notations []Notation `gorm:"foreignKey:AuthorID;references:ID"`
}

func (User) TableName() string { return "users" }
