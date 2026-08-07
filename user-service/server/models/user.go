package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID            string    `gorm:"type:uuid;primaryKey" json:"id"`
	FirstName     string    `gorm:"not_null" json:"first_name"`
	LastName      string    `gorm:"not_null" json:"last_name"`
	Email         string    `gorm:"not_null;unique" json:"email"`
	Password      *string   `json:"-"`
	EmailVerified bool      `gorm:"default: false; not null" json:"email_verified"`
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// Hooks

// BeforeCreate hook generates a UUID if one wasn't provided
func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == "" {
		u.ID = uuid.NewString()
	}
	return
}

// TableName override via Tabler
type Tabler interface {
	TableName() string
}

func (User) TableName() string {
	return "users"
}
