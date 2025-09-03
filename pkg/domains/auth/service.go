package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/crm/pkg/constant"
	"github.com/crm/pkg/dtos"
	"github.com/crm/pkg/entities"
	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Service interface {
	Register(ctx context.Context, req dtos.DTOForUserCreate) (string, error)
	Login(ctx context.Context, req dtos.DTOForUserLogin) (string, error)
	ForgotPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, token string, newPassword string) error
}

type service struct {
	repository Repository
}

func NewService(r Repository) Service {
	return &service{
		repository: r,
	}
}

func (s *service) Register(ctx context.Context, req dtos.DTOForUserCreate) (string, error) {
	// Check if user already exists
	existingUser, err := s.repository.FindUserByEmailOrPhone(ctx, req.Email, req.Phone)
	if err != nil && err != gorm.ErrRecordNotFound {
		return "", fmt.Errorf(constant.SOMETHING_WENT_WRONG)
	}

	if existingUser.ID != 0 {
		return "", fmt.Errorf(constant.ALREADY_EXISTS, "User")
	}

	// Hash password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	// Create user (automatically verified)
	user := entities.User{
		Email:    req.Email,
		Password: string(passwordHash),
		Name:     req.Name,
		Surname:  req.Surname,
		Phone:    req.Phone,
	}

	if err := s.repository.CreateUser(ctx, user); err != nil {
		return "", err
	}

	// Generate JWT token immediately after registration
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":  user.ID,
		"exp": time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString([]byte(os.Getenv("SECRET")))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (s *service) Login(ctx context.Context, req dtos.DTOForUserLogin) (string, error) {
	// Find user by email
	user, err := s.repository.FindUserByEmail(ctx, req.Email)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", fmt.Errorf(constant.EMAIL_OR_PHONE)
		}
		return "", fmt.Errorf(constant.SOMETHING_WENT_WRONG)
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		return "", fmt.Errorf(constant.UNAUTHORIZED_ACCESS)
	}

	// Generate JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":  user.ID,
		"exp": time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString([]byte(os.Getenv("SECRET")))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (s *service) ForgotPassword(ctx context.Context, email string) error {
	// Find user by email
	user, err := s.repository.FindUserByEmail(ctx, email)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf(constant.EMAIL_OR_PHONE)
		}
		return fmt.Errorf(constant.SOMETHING_WENT_WRONG)
	}

	// Generate reset token
	token := generateResetToken()
	user.ResetToken = token
	user.ResetExpiresAt = time.Now().Add(1 * time.Hour)

	if err := s.repository.UpdateUser(ctx, user); err != nil {
		return fmt.Errorf(constant.SOMETHING_WENT_WRONG)
	}

	// For now, just return success (in real app, you'd send email)
	return nil
}

func (s *service) ResetPassword(ctx context.Context, token string, newPassword string) error {
	// Find user by reset token
	user, err := s.repository.FindUserByResetToken(ctx, token)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf(constant.INVALID_TOKEN)
		}
		return fmt.Errorf(constant.SOMETHING_WENT_WRONG)
	}

	// Check if token is expired
	if time.Now().After(user.ResetExpiresAt) {
		return fmt.Errorf(constant.TOKEN_EXPIRED)
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf(constant.SOMETHING_WENT_WRONG)
	}

	// Update user
	user.Password = string(hashedPassword)
	user.ResetToken = ""
	user.ResetExpiresAt = time.Time{}

	if err := s.repository.UpdateUser(ctx, user); err != nil {
		return fmt.Errorf(constant.SOMETHING_WENT_WRONG)
	}

	return nil
}

// Generate random reset token
func generateResetToken() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
