package routes

import (
	"fmt"
	"io"

	"github.com/crm/pkg/constant"
	"github.com/crm/pkg/domains/whatsapp"
	"github.com/crm/pkg/dtos"
	"github.com/crm/pkg/middleware"
	"github.com/gin-gonic/gin"
)

func WhatsAppRoutes(r *gin.RouterGroup, s whatsapp.Service) {
	// Apply JWT authentication to all WhatsApp endpoints
	authGroup := r.Group("", middleware.CheckAuth())
	{
		authGroup.POST("/connect", connect(s))
		authGroup.POST("/disconnect", disconnect(s))
		authGroup.POST("/send-message", sendMessage(s))
		authGroup.POST("/send-media", sendMediaMessage(s))
		authGroup.GET("/qr-code", getQRCode(s))
		authGroup.POST("/check-connection", checkConnection(s))
		authGroup.GET("/status", getStatus(s))
		authGroup.GET("/contacts", getContacts(s))
	}
}

func connect(s whatsapp.Service) func(c *gin.Context) {
	return func(c *gin.Context) {
		if err := s.Connect(c); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"message": constant.WHATSAPP_CONNECTED,
		})
	}
}

func disconnect(s whatsapp.Service) func(c *gin.Context) {
	return func(c *gin.Context) {
		if err := s.Disconnect(c); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"message": constant.WHATSAPP_DISCONNECTED,
		})
	}
}

func sendMessage(s whatsapp.Service) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req dtos.SendMessageDTO
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": constant.INVALID_REQUEST})
			return
		}

		response, err := s.SendMessage(c, req)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"message": constant.MESSAGE_SENT,
			"data":    response,
		})
	}
}

func sendMediaMessage(s whatsapp.Service) func(c *gin.Context) {
	return func(c *gin.Context) {
		// Get form data
		phoneNumber := c.PostForm("phone_number")
		caption := c.PostForm("caption")
		mimeType := c.PostForm("mime_type")
		height := c.PostForm("height")
		width := c.PostForm("width")

		if phoneNumber == "" || mimeType == "" {
			c.JSON(400, gin.H{"error": "phone_number and mime_type are required"})
			return
		}

		// Get uploaded file
		file, header, err := c.Request.FormFile("media")
		if err != nil {
			c.JSON(400, gin.H{"error": "Failed to get uploaded file"})
			return
		}
		defer file.Close()

		// Read file data
		mediaData, err := io.ReadAll(file)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to read file data"})
			return
		}

		// Create DTO
		req := dtos.SendMediaMessageDTO{
			PhoneNumber: phoneNumber,
			Caption:     caption,
			MediaData:   mediaData,
			MimeType:    mimeType,
		}

		// Parse height and width if provided
		if height != "" {
			if h, err := fmt.Sscanf(height, "%d", &req.Height); err != nil || h == 0 {
				c.JSON(400, gin.H{"error": "Invalid height value"})
				return
			}
		}
		if width != "" {
			if w, err := fmt.Sscanf(width, "%d", &req.Width); err != nil || w == 0 {
				c.JSON(400, gin.H{"error": "Invalid width value"})
				return
			}
		}

		response, err := s.SendMediaMessage(c, req)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"message": fmt.Sprintf("%s. File: %s", constant.MEDIA_SENT, header.Filename),
			"data":    response,
		})
	}
}

func getQRCode(s whatsapp.Service) func(c *gin.Context) {
	return func(c *gin.Context) {
		qrCode, err := s.GetQRCode(c)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"qr_code": qrCode,
			"message": "Scan this QR code with WhatsApp mobile app",
		})
	}
}

func checkConnection(s whatsapp.Service) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req struct {
			PhoneNumber string `json:"phone_number" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": constant.INVALID_REQUEST})
			return
		}

		connected, err := s.CheckConnection(c, req.PhoneNumber)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, dtos.CheckConnectionDTO{
			PhoneNumber: req.PhoneNumber,
			Connected:   connected,
		})
	}
}

func getStatus(s whatsapp.Service) func(c *gin.Context) {
	return func(c *gin.Context) {
		status, err := s.GetStatus(c)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, dtos.WhatsAppStatusDTO{
			Status: status,
		})
	}
}

func getContacts(s whatsapp.Service) func(c *gin.Context) {
	return func(c *gin.Context) {
		contacts, err := s.GetContacts(c)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		var contactDTOs []dtos.ContactInfoDTO
		for jid, contact := range contacts {
			contactDTOs = append(contactDTOs, dtos.ContactInfoDTO{
				JID:          jid.String(), // Proper JID string
				Name:         contact.PushName,
				FirstName:    contact.FirstName,
				FullName:     contact.FullName,
				BusinessName: contact.BusinessName, // Business contact support
				Found:        contact.Found,        // Contact found status
			})
		}

		c.JSON(200, gin.H{
			"contacts": contactDTOs,
		})
	}
}
