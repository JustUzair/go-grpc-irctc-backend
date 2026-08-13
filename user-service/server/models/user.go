package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID            string         `gorm:"type:uuid;primaryKey" json:"id"`
	FirstName     string         `gorm:"not null" json:"first_name"`
	LastName      string         `gorm:"not null" json:"last_name"`
	Email         string         `gorm:"not null;unique" json:"email"`
	Password      *string        `json:"-"`
	EmailVerified bool           `gorm:"default: false; not null" json:"email_verified"`
	CreatedAt     time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	AuthProviders []AuthProvider `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE" json:"auth_providers"`
}

// Hooks

// BeforeCreate hook generates a UUID if one wasn't provided
func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == "" {
		u.ID = uuid.NewString()
	}
	return
}

func (User) TableName() string {
	return "users"
}
