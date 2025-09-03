package utils

import (
	"net/mail"
	"strings"
	"unicode"

	"github.com/go-playground/validator/v10"
)

type CustomValidator struct {
	Validator *validator.Validate
}

func NewCustomValidator() *CustomValidator {
	Validator := &CustomValidator{validator.New()}
	Validator.ValidatorRegistery()
	return Validator
}

func (c *CustomValidator) ValidatorRegistery() {
	c.Validator.RegisterValidation("isemail", c.IsValidEmail)
	c.Validator.RegisterValidation("isphone", c.IsValidPhone)
}

func (c *CustomValidator) IsValidEmail(fl validator.FieldLevel) bool {

	email := strings.TrimSpace(fl.Field().String())
	_, err := mail.ParseAddress(email)
	return err == nil
}

func (c *CustomValidator) IsValidPhone(fl validator.FieldLevel) bool {
	phoneNumber := strings.TrimSpace(fl.Field().String())
	if len(phoneNumber) != 11 {
		return false
	}
	for _, char := range phoneNumber {
		if !unicode.IsDigit(char) {
			return false
		}
	}
	return true
}
