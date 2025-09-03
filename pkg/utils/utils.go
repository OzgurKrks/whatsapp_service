package utils

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"math"
	"math/big"

	"github.com/crm/pkg/constant"
	"github.com/joho/godotenv"

	"gorm.io/gorm"
)

func LoadEnv() {
	err := godotenv.Load()
	if err != nil {
		// Don't fail if .env file doesn't exist
		// Environment variables can be provided via Docker Compose or system
		log.Println("Info: .env file not found, using system environment variables")
	}
}

func Pagination(item interface{}, pageNumber int, db *gorm.DB, c context.Context, query interface{}, args ...interface{}) (int, error) {
	limit := 10
	offset := 0

	var totalCount int64
	if err := db.WithContext(c).Model(item).Where(query, args...).Count(&totalCount).Error; err != nil {
		return 0, err
	}

	// Calculate total pages
	totalPages := int(math.Ceil(float64(totalCount) / float64(limit)))

	if pageNumber > totalPages || pageNumber <= 0 {
		return 0, errors.New(constant.PAGE_NUMBER_OUT_OF_RANGE)
	}

	// Check if pageNumber is provided and valid
	if pageNumber > 0 {
		offset = (pageNumber - 1) * limit
	}

	// Get items with pagination
	if err := db.WithContext(c).Limit(limit).Offset(offset).Where(query, args...).Find(item).Error; err != nil {
		return 0, err
	}
	return totalPages, nil
}

func GenerateVerificationCode() string {
	const length = 4
	numbers := "0123456789"
	code := make([]byte, length)

	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(numbers))))
		if err != nil {
			panic(fmt.Sprintf("failed to generate random number: %v", err))
		}
		code[i] = numbers[num.Int64()]
	}

	return string(code)
}
