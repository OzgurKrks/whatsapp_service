package entities

import (
	"time"

	"gorm.io/gorm"
)

// WhatsAppSession stores WhatsApp session data for each user
type WhatsAppSession struct {
	gorm.Model
	UserID         uint      `json:"user_id" gorm:"uniqueIndex;not null"`
	SessionData    []byte    `json:"session_data" gorm:"type:bytea"`
	IsConnected    bool      `json:"is_connected" gorm:"default:false"`
	IsLoggedIn     bool      `json:"is_logged_in" gorm:"default:false"`
	PhoneNumber    string    `json:"phone_number" gorm:"type:varchar(20)"`
	LastActiveAt   time.Time `json:"last_active_at"`
	
	// Relations
	User User `json:"user" gorm:"foreignKey:UserID"`
}

// WhatsAppDevice stores device information for WhatsApp sessions
type WhatsAppDevice struct {
	gorm.Model
	UserID       uint   `json:"user_id" gorm:"uniqueIndex;not null"`
	JID          string `json:"jid" gorm:"type:varchar(255)"`
	Registration []byte `json:"registration" gorm:"type:bytea"`
	NoiseKey     []byte `json:"noise_key" gorm:"type:bytea"`
	IdentityKey  []byte `json:"identity_key" gorm:"type:bytea"`
	SignedPreKey []byte `json:"signed_pre_key" gorm:"type:bytea"`
	
	// Relations
	User User `json:"user" gorm:"foreignKey:UserID"`
}

// WhatsAppMessage stores WhatsApp message logs
type WhatsAppMessage struct {
	gorm.Model
	UserID      uint      `json:"user_id" gorm:"not null"`
	MessageID   string    `json:"message_id" gorm:"type:varchar(255);not null"`
	FromJID     string    `json:"from_jid" gorm:"type:varchar(255);not null"`
	ToJID       string    `json:"to_jid" gorm:"type:varchar(255);not null"`
	Content     string    `json:"content" gorm:"type:text"`
	MessageType string    `json:"message_type" gorm:"type:varchar(50)"`
	Timestamp   time.Time `json:"timestamp"`
	IsIncoming  bool      `json:"is_incoming" gorm:"default:false"`
	
	// Relations
	User User `json:"user" gorm:"foreignKey:UserID"`
}
