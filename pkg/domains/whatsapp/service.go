package whatsapp

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/crm/pkg/constant"
	"github.com/crm/pkg/database"
	"github.com/crm/pkg/dtos"
	"github.com/crm/pkg/entities"
	"github.com/crm/pkg/state"
	"github.com/gin-gonic/gin"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waTypes "go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

type Service interface {
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	SendMessage(ctx context.Context, req dtos.SendMessageDTO) (*dtos.MessageResponseDTO, error)
	SendMediaMessage(ctx context.Context, req dtos.SendMediaMessageDTO) (*dtos.MessageResponseDTO, error)
	GetQRCode(ctx context.Context) (string, error)
	CheckConnection(ctx context.Context, phoneNumber string) (bool, error)
	GetStatus(ctx context.Context) (string, error)
	GetContacts(ctx context.Context) (map[types.JID]types.ContactInfo, error)
}

// UserSession represents a WhatsApp session for a specific user
type UserSession struct {
	UserID      uint
	Client      *whatsmeow.Client
	DB          *sqlstore.Container
	EventChan   chan *events.Message
	IsConnected bool
	Ctx         context.Context
	Cancel      context.CancelFunc
}

type service struct {
	sessions map[uint]*UserSession // Map of user ID to their WhatsApp session
	mutex    sync.RWMutex          // Mutex to protect concurrent access to sessions
}

func NewService() Service {
	s := &service{
		sessions: make(map[uint]*UserSession),
		mutex:    sync.RWMutex{},
	}

	return s
}

// getUserSession gets or creates a WhatsApp session for the user
func (s *service) getUserSession(userID uint) (*UserSession, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check if session already exists
	if session, exists := s.sessions[userID]; exists {
		return session, nil
	}

	// Create new session for this user
	ctx, cancel := context.WithCancel(context.Background())
	session := &UserSession{
		UserID:    userID,
		EventChan: make(chan *events.Message, 100),
		Ctx:       ctx,
		Cancel:    cancel,
	}

	// Initialize the session
	if err := s.initializeUserClient(session); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize client for user %d: %v", userID, err)
	}

	// Start event processor for this user
	go s.eventProcessor(session)

	// Store the session
	s.sessions[userID] = session

	return session, nil
}

// removeUserSession removes a user's WhatsApp session
func (s *service) removeUserSession(userID uint) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if session, exists := s.sessions[userID]; exists {
		// Stop event processor
		if session.Cancel != nil {
			session.Cancel()
		}

		// Close event channel
		if session.EventChan != nil {
			close(session.EventChan)
		}

		// Disconnect client
		if session.Client != nil {
			session.Client.Disconnect()
		}

		// Close database connection
		if session.DB != nil {
			session.DB.Close()
		}

		// Remove from map
		delete(s.sessions, userID)
		log.Printf("Removed WhatsApp session for user %d", userID)
	}
}

// getUserIDFromContext extracts user ID from Gin context
func (s *service) getUserIDFromContext(ctx context.Context) (uint, error) {
	if ginCtx, ok := ctx.(*gin.Context); ok {
		userID, exists := ginCtx.Get(state.CurrentUserId)
		if !exists {
			return 0, fmt.Errorf("user ID not found in context")
		}

		if uid, ok := userID.(uint); ok {
			return uid, nil
		}
		return 0, fmt.Errorf("invalid user ID type in context")
	}
	return 0, fmt.Errorf("invalid context type")
}

// eventProcessor processes incoming WhatsApp events in background for a specific user
func (s *service) eventProcessor(session *UserSession) {
	for {
		select {
		case event := <-session.EventChan:
			// Process incoming message - MessageInfo struct'ını doğru kullanım
			sender := event.Info.SourceString() // Better source identification
			var messageText string

			// Safe message content extraction
			if event.Message != nil && event.Message.GetConversation() != "" {
				messageText = event.Message.GetConversation()
			} else if event.Message != nil && event.Message.GetExtendedTextMessage() != nil {
				messageText = event.Message.GetExtendedTextMessage().GetText()
			} else {
				messageText = "[Media or unsupported message type]"
			}

			log.Printf("📱 WhatsApp Message [User %d] - From: %s | Content: %s | Timestamp: %v",
				session.UserID, sender, messageText, event.Info.Timestamp)

			// Here you can add your custom message processing logic
			// For example: save to database, trigger webhooks, auto-reply, etc.
		case <-session.Ctx.Done():
			log.Printf("Event processor stopped for user %d", session.UserID)
			return
		}
	}
}

// formatPhoneNumber converts phone number to WhatsApp JID format using proper whatsmeow functions
func (s *service) formatPhoneNumber(phoneNumber string) (waTypes.JID, error) {
	// Remove all non-numeric characters except +
	re := regexp.MustCompile(`[^\d+]`)
	cleanPhone := re.ReplaceAllString(phoneNumber, "")

	// Remove leading + if present
	cleanPhone = strings.TrimPrefix(cleanPhone, "+")

	// Validate phone number (must be at least 10 digits)
	if len(cleanPhone) < 10 {
		return waTypes.JID{}, fmt.Errorf("invalid phone number: too short")
	}

	// Use proper NewJID function from whatsmeow types
	jid := waTypes.NewJID(cleanPhone, waTypes.DefaultUserServer)

	return jid, nil
}

func (s *service) Connect(ctx context.Context) error {
	// Get user ID from context
	userID, err := s.getUserIDFromContext(ctx)
	if err != nil {
		return fmt.Errorf("authentication required: %v", err)
	}

	// Get user session (don't create new one, check existing)
	s.mutex.RLock()
	session, exists := s.sessions[userID]
	s.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("no active session found. Please scan QR code first")
	}

	// Check if already connected and logged in
	if session.IsConnected && session.Client != nil && session.Client.IsConnected() && session.Client.Store.ID != nil {
		log.Printf("User %d is already connected", userID)
		return nil // Already connected and logged in
	}

	// Check if we have a valid session but not connected
	if session.Client != nil && session.Client.Store.ID != nil {
		// User is logged in but websocket not connected
		if !session.Client.IsConnected() {
			if err := session.Client.Connect(); err != nil {
				return fmt.Errorf("failed to connect: %v", err)
			}
		}
		session.IsConnected = true
		s.updateSessionStatus(userID, true, true)
		log.Printf("WhatsApp client reconnected successfully for user %d", userID)
		return nil
	}

	// If not logged in, return error
	return fmt.Errorf("not logged in to WhatsApp. Please scan QR code first")
}

func (s *service) Disconnect(ctx context.Context) error {
	// Get user ID from context
	userID, err := s.getUserIDFromContext(ctx)
	if err != nil {
		return fmt.Errorf("authentication required: %v", err)
	}

	log.Printf("Starting graceful shutdown of WhatsApp client for user %d", userID)

	// Remove user session (this handles all cleanup)
	s.removeUserSession(userID)

	log.Printf("WhatsApp service shutdown completed for user %d", userID)
	return nil
}

func (s *service) SendMessage(ctx context.Context, req dtos.SendMessageDTO) (*dtos.MessageResponseDTO, error) {
	// Get user ID from context
	userID, err := s.getUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("authentication required: %v", err)
	}

	// Get user session
	s.mutex.RLock()
	session, exists := s.sessions[userID]
	s.mutex.RUnlock()

	if !exists || session.Client == nil {
		return nil, fmt.Errorf(constant.WHATSAPP_NOT_CONNECTED)
	}

	// Check if client is logged in and connected
	if session.Client.Store.ID == nil {
		return nil, fmt.Errorf("not logged in to WhatsApp. Please scan QR code first")
	}

	if !session.Client.IsConnected() {
		return nil, fmt.Errorf("WhatsApp websocket not connected. Please call /connect first")
	}

	if !session.IsConnected {
		return nil, fmt.Errorf("session not marked as connected. Please call /connect first")
	}

	// Format phone number to JID
	recipient, err := s.formatPhoneNumber(req.PhoneNumber)
	if err != nil {
		return nil, fmt.Errorf(constant.INVALID_PHONE_NUMBER+": %v", err)
	}

	// Create message
	msg := &waProto.Message{
		Conversation: proto.String(req.Message),
	}

	// Send message
	resp, err := session.Client.SendMessage(ctx, recipient, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %v", err)
	}

	// Create response DTO
	response := &dtos.MessageResponseDTO{
		MessageID: resp.ID,
		Timestamp: resp.Timestamp.Format(time.RFC3339),
		Status:    "sent",
		To:        req.PhoneNumber,
	}

	log.Printf("Message sent successfully by user %d. ID: %s, Timestamp: %s", userID, resp.ID, resp.Timestamp)
	return response, nil
}

func (s *service) SendMediaMessage(ctx context.Context, req dtos.SendMediaMessageDTO) (*dtos.MessageResponseDTO, error) {
	// Get user ID from context
	userID, err := s.getUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("authentication required: %v", err)
	}

	// Get user session
	s.mutex.RLock()
	session, exists := s.sessions[userID]
	s.mutex.RUnlock()

	if !exists || session.Client == nil {
		return nil, fmt.Errorf(constant.WHATSAPP_NOT_CONNECTED)
	}

	// Check if client is logged in and connected
	if session.Client.Store.ID == nil {
		return nil, fmt.Errorf("not logged in to WhatsApp. Please scan QR code first")
	}

	if !session.Client.IsConnected() {
		return nil, fmt.Errorf("WhatsApp websocket not connected. Please call /connect first")
	}

	if !session.IsConnected {
		return nil, fmt.Errorf("session not marked as connected. Please call /connect first")
	}

	// Format phone number to JID
	recipient, err := s.formatPhoneNumber(req.PhoneNumber)
	if err != nil {
		return nil, fmt.Errorf(constant.INVALID_PHONE_NUMBER+": %v", err)
	}

	// Determine media type based on MIME type
	var mediaType whatsmeow.MediaType
	switch {
	case strings.HasPrefix(req.MimeType, "image/"):
		mediaType = whatsmeow.MediaImage
	case strings.HasPrefix(req.MimeType, "video/"):
		mediaType = whatsmeow.MediaVideo
	case strings.HasPrefix(req.MimeType, "audio/"):
		mediaType = whatsmeow.MediaAudio
	default:
		mediaType = whatsmeow.MediaDocument
	}

	// Upload media
	uploaded, err := session.Client.Upload(ctx, req.MediaData, mediaType)
	if err != nil {
		return nil, fmt.Errorf(constant.MEDIA_UPLOAD_FAILED+": %v", err)
	}

	var msg *waProto.Message

	// Create appropriate message based on media type
	switch mediaType {
	case whatsmeow.MediaImage:
		msg = &waProto.Message{
			ImageMessage: &waProto.ImageMessage{
				URL:           &uploaded.URL,
				Mimetype:      &req.MimeType,
				Caption:       &req.Caption,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    &uploaded.FileLength,
				Height:        &req.Height,
				Width:         &req.Width,
				DirectPath:    &uploaded.DirectPath,
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
			},
		}
	case whatsmeow.MediaVideo:
		msg = &waProto.Message{
			VideoMessage: &waProto.VideoMessage{
				URL:           &uploaded.URL,
				Mimetype:      &req.MimeType,
				Caption:       &req.Caption,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    &uploaded.FileLength,
				DirectPath:    &uploaded.DirectPath,
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
			},
		}
	case whatsmeow.MediaAudio:
		msg = &waProto.Message{
			AudioMessage: &waProto.AudioMessage{
				URL:           &uploaded.URL,
				Mimetype:      &req.MimeType,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    &uploaded.FileLength,
				DirectPath:    &uploaded.DirectPath,
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
			},
		}
	default: // Document
		msg = &waProto.Message{
			DocumentMessage: &waProto.DocumentMessage{
				URL:           &uploaded.URL,
				Mimetype:      &req.MimeType,
				Title:         &req.Caption,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    &uploaded.FileLength,
				DirectPath:    &uploaded.DirectPath,
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
			},
		}
	}

	// Send message
	resp, err := session.Client.SendMessage(ctx, recipient, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send media message: %v", err)
	}

	// Create response DTO
	response := &dtos.MessageResponseDTO{
		MessageID: resp.ID,
		Timestamp: resp.Timestamp.Format(time.RFC3339),
		Status:    "sent",
		To:        req.PhoneNumber,
	}

	log.Printf("Media message sent successfully by user %d. ID: %s, Type: %s", userID, resp.ID, mediaType)
	return response, nil
}

func (s *service) GetQRCode(ctx context.Context) (string, error) {
	// Get user ID from context
	userID, err := s.getUserIDFromContext(ctx)
	if err != nil {
		return "", fmt.Errorf("authentication required: %v", err)
	}

	// Check if user already has a session and if it's logged in
	s.mutex.RLock()
	existingSession, exists := s.sessions[userID]
	s.mutex.RUnlock()

	if exists && existingSession.Client != nil && existingSession.Client.Store.ID != nil {
		return fmt.Sprintf("User %d already logged in to WhatsApp", userID), nil
	}

	// If session exists but not logged in, return appropriate message
	if exists && existingSession.Client != nil {
		log.Printf("Session exists for user %d but not logged in", userID)

		// Check if already logged in
		if existingSession.Client.Store.ID != nil {
			return fmt.Sprintf("User %d already logged in to WhatsApp", userID), nil
		}

		// Return message that QR code is already being generated
		return fmt.Sprintf("User %d already has a QR code session. Please wait for the current QR code to be scanned or expired.", userID), nil
	}

	// Create new user session
	session, err := s.getUserSession(userID)
	if err != nil {
		return "", fmt.Errorf("failed to get user session: %v", err)
	}

	// Check if already logged in (after session creation)
	if session.Client.Store.ID != nil {
		return fmt.Sprintf("User %d already logged in", userID), nil
	}

	// Get QR channel BEFORE connecting (as per documentation)
	qrChan, err := session.Client.GetQRChannel(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get QR channel: %v", err)
	}

	// Connect to start QR generation
	err = session.Client.Connect()
	if err != nil {
		return "", fmt.Errorf("failed to connect: %v", err)
	}

	log.Printf("Generating QR code for user %d", userID)

	// Listen for QR events
	for evt := range qrChan {
		switch evt.Event {
		case "code":
			log.Printf("QR code generated for user %d", userID)
			s.updateSessionStatus(userID, true, false)
			return evt.Code, nil
		case "success":
			session.IsConnected = true
			log.Printf("User %d successfully connected via QR code", userID)
			s.updateSessionStatus(userID, true, true)
			// QR kod başarılı olduğunda session'ı kaydet
			s.mutex.Lock()
			s.sessions[userID] = session
			s.mutex.Unlock()
			return fmt.Sprintf("User %d successfully connected", userID), nil
		case "timeout":
			log.Printf("QR code timeout for user %d", userID)
			return "", fmt.Errorf("QR code expired")
		case "error":
			log.Printf("QR code error for user %d: %v", userID, evt.Error)
			return "", fmt.Errorf("QR code error: %v", evt.Error)
		default:
			log.Printf("Unknown QR event for user %d: %s", userID, evt.Event)
		}
	}

	return "", fmt.Errorf("QR channel closed unexpectedly")
}

func (s *service) initializeUserClient(session *UserSession) error {
	log.Printf("Starting WhatsApp client initialization for user %d", session.UserID)

	// Use in-memory store for WhatsApp session data
	clientLog := waLog.Stdout(fmt.Sprintf("WhatsApp_User_%d", session.UserID), "INFO", true)
	log.Printf("Created logger for user %d", session.UserID)

	// Create in-memory database for whatsmeow with foreign keys enabled
	log.Printf("Creating in-memory SQLite database for user %d", session.UserID)
	db, err := sqlstore.New(session.Ctx, "sqlite", ":memory:?_pragma=foreign_keys(1)", clientLog)
	if err != nil {
		log.Printf("Failed to create in-memory database for user %d: %v", session.UserID, err)
		return fmt.Errorf("failed to create in-memory database: %v", err)
	}
	log.Printf("Successfully created in-memory database for user %d", session.UserID)

	session.DB = db

	// Get device store
	log.Printf("Getting device store for user %d", session.UserID)
	deviceStore, err := session.DB.GetFirstDevice(session.Ctx)
	if err != nil {
		log.Printf("Failed to get device store for user %d: %v", session.UserID, err)
		return fmt.Errorf("failed to get device: %v", err)
	}
	log.Printf("Successfully got device store for user %d", session.UserID)

	// Create client
	log.Printf("Creating WhatsApp client for user %d", session.UserID)
	session.Client = whatsmeow.NewClient(deviceStore, clientLog)
	log.Printf("Successfully created WhatsApp client for user %d", session.UserID)

	// Register event handlers
	session.Client.AddEventHandler(func(evt interface{}) {
		s.handleEvents(session, evt)
	})
	log.Printf("Registered event handlers for user %d", session.UserID)

	// Update session status in PostgreSQL
	s.updateSessionStatus(session.UserID, false, false)

	log.Printf("Successfully initialized WhatsApp client for user %d (in-memory + PostgreSQL tracking)", session.UserID)
	return nil
}

// updateSessionStatus updates the session status in PostgreSQL
func (s *service) updateSessionStatus(userID uint, isConnected, isLoggedIn bool) {
	db := database.DBClient()

	var session entities.WhatsAppSession
	err := db.Where("user_id = ?", userID).First(&session).Error
	if err == gorm.ErrRecordNotFound {
		session = entities.WhatsAppSession{
			UserID:       userID,
			IsConnected:  isConnected,
			IsLoggedIn:   isLoggedIn,
			LastActiveAt: time.Now(),
		}
		db.Create(&session)
	} else if err == nil {
		session.IsConnected = isConnected
		session.IsLoggedIn = isLoggedIn
		session.LastActiveAt = time.Now()
		db.Save(&session)
	}
}

func (s *service) CheckConnection(ctx context.Context, phoneNumber string) (bool, error) {
	// Get user ID from context
	userID, err := s.getUserIDFromContext(ctx)
	if err != nil {
		return false, fmt.Errorf("authentication required: %v", err)
	}

	// Get user session
	s.mutex.RLock()
	session, exists := s.sessions[userID]
	s.mutex.RUnlock()

	if !exists || session.Client == nil {
		return false, nil
	}

	// Check if client is connected and logged in
	if !session.Client.IsConnected() {
		return false, nil
	}

	// Check if we have a valid session
	if session.Client.Store.ID == nil {
		return false, nil
	}

	return true, nil
}

func (s *service) GetStatus(ctx context.Context) (string, error) {
	// Get user ID from context
	userID, err := s.getUserIDFromContext(ctx)
	if err != nil {
		return "", fmt.Errorf("authentication required: %v", err)
	}

	// Get user session from memory first
	s.mutex.RLock()
	session, exists := s.sessions[userID]
	s.mutex.RUnlock()

	// If memory session exists, check real-time status
	if exists && session.Client != nil {
		// Check if user is logged in (has valid Store ID)
		if session.Client.Store.ID != nil {
			if session.IsConnected && session.Client.IsConnected() {
				s.updateSessionStatus(userID, true, true)
				return "Connected and logged in", nil
			} else if session.Client.IsConnected() {
				s.updateSessionStatus(userID, true, true)
				return "Logged in but session not marked as connected", nil
			} else {
				s.updateSessionStatus(userID, false, true)
				return "Logged in but websocket disconnected", nil
			}
		}

		// Check if client is connected but not logged in
		if session.Client.IsConnected() {
			s.updateSessionStatus(userID, true, false)
			return "Connected but not logged in", nil
		}

		// Session exists but not connected
		s.updateSessionStatus(userID, false, false)
		return "Session exists but disconnected", nil
	}

	// Check PostgreSQL for session status
	db := database.DBClient()
	var dbSession entities.WhatsAppSession
	err = db.Where("user_id = ?", userID).First(&dbSession).Error
	if err == gorm.ErrRecordNotFound {
		return "Not initialized", nil
	} else if err != nil {
		return "", fmt.Errorf("failed to get session status: %v", err)
	}

	if dbSession.IsLoggedIn {
		return "Logged in but session expired (restart needed)", nil
	} else if dbSession.IsConnected {
		return "Previous session existed but expired", nil
	}
	return "Not initialized", nil
}

func (s *service) GetContacts(ctx context.Context) (map[types.JID]types.ContactInfo, error) {
	// Get user ID from context
	userID, err := s.getUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("authentication required: %v", err)
	}

	// Get user session
	s.mutex.RLock()
	session, exists := s.sessions[userID]
	s.mutex.RUnlock()

	if !exists || session.Client == nil {
		return nil, fmt.Errorf(constant.WHATSAPP_NOT_CONNECTED)
	}

	// Check if client is connected and logged in
	if !session.IsConnected || !session.Client.IsConnected() || session.Client.Store.ID == nil {
		return nil, fmt.Errorf("WhatsApp not connected or not logged in. Please connect first")
	}

	contacts, err := session.Client.Store.Contacts.GetAllContacts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get contacts: %v", err)
	}

	return contacts, nil
}

func (s *service) handleEvents(session *UserSession, evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		// Handle incoming messages
		session.EventChan <- v
	case *events.Receipt:
		// Handle message receipts
		// You can implement delivery status tracking here
		log.Printf("Message receipt for user %d: %v", session.UserID, v)
	}
}
