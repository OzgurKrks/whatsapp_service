package database

import (
	"github.com/crm/pkg/entities"
	"gorm.io/gorm"
)

// AutoMigrate runs database migrations
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&entities.User{},
		&entities.WhatsAppSession{},
		&entities.WhatsAppDevice{},
		&entities.WhatsAppMessage{},
	)
}
