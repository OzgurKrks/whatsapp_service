package auth

import (
	"context"

	"github.com/crm/pkg/entities"
	"gorm.io/gorm"
)

type Repository interface {
	CreateUser(ctx context.Context, user entities.User) error
	FindUserByEmail(ctx context.Context, email string) (entities.User, error)
	FindUserByEmailOrPhone(ctx context.Context, email string, phone string) (entities.User, error)
	UpdateUser(ctx context.Context, user entities.User) error
	FindUserByResetToken(ctx context.Context, token string) (entities.User, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepo(db *gorm.DB) Repository {
	return &repository{
		db: db,
	}
}

func (r *repository) CreateUser(ctx context.Context, user entities.User) error {
	return r.db.WithContext(ctx).Create(&user).Error
}

func (r *repository) FindUserByEmail(ctx context.Context, email string) (entities.User, error) {
	var user entities.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	return user, err
}

func (r *repository) FindUserByEmailOrPhone(ctx context.Context, email string, phone string) (entities.User, error) {
	var user entities.User
	err := r.db.WithContext(ctx).Where("email = ? OR phone = ?", email, phone).First(&user).Error
	return user, err
}

func (r *repository) UpdateUser(ctx context.Context, user entities.User) error {
	return r.db.WithContext(ctx).Save(&user).Error
}

func (r *repository) FindUserByResetToken(ctx context.Context, token string) (entities.User, error) {
	var user entities.User
	err := r.db.WithContext(ctx).Where("reset_token = ?", token).First(&user).Error
	return user, err
}
