package models

import (
	"time"

	"github.com/google/uuid"
)

// --- models/user.go ---
// hier könnte noch validate im "json block" stehen => analog zu Hibernate
type User struct {
	ID       uuid.UUID `gorm:"column:id;primaryKey" json:"id"`
	Username string    `gorm:"column:username;type:text;not null;uniqueIndex" json:"username"`
	Email    *string   `gorm:"column:email;type:text;uniqueIndex" json:"email,omitempty"`
	//PasswordHash string     `gorm:"column:password_hash;type:text;not null" json:"-"`
	FirstName *string    `gorm:"column:first_name;type:text" json:"first_name,omitempty"`
	LastName  *string    `gorm:"column:last_name;type:text" json:"last_name,omitempty"`
	CreatedAt time.Time  `gorm:"column:created_at;not null;default:now()" json:"created_at"`
	UpdatedAt time.Time  `gorm:"column:updated_at;not null;default:now()" json:"updated_at"`
	Tracks    []Track    `gorm:"foreignKey:UserID;references:ID" json:"tracks,omitempty"`
	Notations []Notation `gorm:"foreignKey:AuthorID;references:ID" json:"notations,omitempty"`
}

func (User) TableName() string { return "users" }
