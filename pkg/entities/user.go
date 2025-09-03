package entities

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Email          string    `json:"email" gorm:"unique;not null"`
	Password       string    `json:"password" gorm:"not null"`
	Name           string    `json:"name" gorm:"type:varchar(255);not null"`
	Surname        string    `json:"surname" gorm:"type:varchar(255);not null"`
	Phone          string    `json:"phone" gorm:"type:varchar(20)"`
	ResetToken     string    `json:"reset_token" gorm:"type:varchar(255)"`
	ResetExpiresAt time.Time `json:"reset_expires_at"`
}
