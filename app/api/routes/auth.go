package routes

import (
	"fmt"

	"github.com/crm/pkg/constant"
	"github.com/crm/pkg/domains/auth"
	"github.com/crm/pkg/dtos"
	"github.com/gin-gonic/gin"
)

func AuthRoutes(r *gin.RouterGroup, s auth.Service) {
	r.POST("/register", register(s))
	r.POST("/login", login(s))
	r.POST("/forgot-password", forgotPassword(s))
	r.POST("/reset-password", resetPassword(s))
}

func register(s auth.Service) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req dtos.DTOForUserCreate
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": constant.INVALID_REQUEST})
			return
		}

		token, err := s.Register(c, req)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		c.JSON(201, gin.H{
			"message": fmt.Sprintf(constant.CREATED, "User"),
			"token":   token,
		})
	}
}

func login(s auth.Service) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req dtos.DTOForUserLogin
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": constant.INVALID_REQUEST})
			return
		}

		token, err := s.Login(c, req)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"token": token})
	}
}

func forgotPassword(s auth.Service) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req dtos.ForgotPasswordDTO

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request payload"})
			return
		}

		if err := s.ForgotPassword(c, req.Email); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"message": "Password reset email sent"})
	}
}

func resetPassword(s auth.Service) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req dtos.ResetPasswordDTO

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": constant.INVALID_REQUEST})
			return
		}

		if err := s.ResetPassword(c, req.Token, req.Password); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"message": "Password reset successfully"})
	}
}
