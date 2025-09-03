package dtos

type SendMessageDTO struct {
	PhoneNumber string `json:"phone_number" binding:"required"`
	Message     string `json:"message" binding:"required"`
}

type SendMediaMessageDTO struct {
	PhoneNumber string `json:"phone_number" binding:"required"`
	Caption     string `json:"caption"`
	MediaData   []byte `json:"media_data" binding:"required"`
	MimeType    string `json:"mime_type" binding:"required"`
	Height      uint32 `json:"height"`
	Width       uint32 `json:"width"`
}

type WhatsAppStatusDTO struct {
	Status string `json:"status"`
}

type QRCodeDTO struct {
	PhoneNumber string `json:"phone_number"`
	QRCode      string `json:"qr_code"`
}

type CheckConnectionDTO struct {
	PhoneNumber string `json:"phone_number"`
	Connected   bool   `json:"connected"`
}

type ContactInfoDTO struct {
	JID          string `json:"jid"`
	Name         string `json:"name"` // PushName
	FirstName    string `json:"first_name"`
	FullName     string `json:"full_name"`
	BusinessName string `json:"business_name"` // For business contacts
	Found        bool   `json:"found"`         // Whether contact was found
}

type MessageResponseDTO struct {
	MessageID string `json:"message_id"`
	Timestamp string `json:"timestamp"`
	Status    string `json:"status"`
	To        string `json:"to"`
}
