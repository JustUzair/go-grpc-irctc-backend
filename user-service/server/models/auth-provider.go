package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AuthProvider struct {
	ID         string    `gorm:"type:uuid;primaryKey" json:"id"`
	Provider   string    `gorm:"not null;index:,unique,composite:provider_external_id;index:,unique,composite:user_provider" json:"provider"`
	ProviderID string    `gorm:"not null;index:,unique,composite:provider_external_id" json:"provider_id"`
	UserID     string    `gorm:"type:uuid;not null;index:,unique,composite:user_provider" json:"user_id"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	User       *User     `gorm:"foreignKey:UserID;references:ID" json:"-"`
}

// Hooks

// BeforeCreate hook generates a UUID if one wasn't provided
func (a *AuthProvider) BeforeCreate(tx *gorm.DB) (err error) {
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	return
}

func (AuthProvider) TableName() string {
	return "auth_providers"
}
