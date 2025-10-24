package models

import "time"

// --- models/user.go ---
type User struct {
	Username     string    `gorm:"column:username;primaryKey;type:text;not null"`
	Email        *string   `gorm:"column:email;constraint:unique"`
	PasswordHash string    `gorm:"column:password_hash;type:text;not null"`
	CreatedAt    time.Time `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt    time.Time `gorm:"column:updated_at;not null;default:now()"`
	// Has-many: KEIN constraint-Tag hier!
	Tracks []Track `gorm:"foreignKey:Username;"`
}

func (User) TableName() string { return "users" }
