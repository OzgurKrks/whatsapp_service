# Golang Gin Boilerplate

Bu proje Go ve GORM kullanılarak geliştirilmiş bir Golang Gin Boilerplate'idir.

## Özellikler

- **User Authentication**: Kullanıcı kayıt, giriş, doğrulama ve şifre sıfırlama
- **JWT Token**: Güvenli kimlik doğrulama
- **Email Verification**: Email doğrulama sistemi
- **Password Reset**: Şifre sıfırlama özelliği
- **GORM**: PostgreSQL veritabanı ORM
- **Gin**: HTTP web framework
- **Swagger**: API dokümantasyonu

## Kurulum

### Gereksinimler

- Go 1.23+
- PostgreSQL
- Redis (opsiyonel)

### Environment Variables

`.env` dosyası oluşturun:

```env
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=crm_db

# App
APP_HOST=localhost
APP_PORT=8000
APP_NAME=crm

# JWT Secret
SECRET=your_jwt_secret_key

# SMTP (Email)
SMTP_HOST=smtp.hostinger.com
SMTP_PORT=587
SMTP_EMAIL=your_email@domain.com
SMTP_PASSWORD=your_email_password
```

### Veritabanı Kurulumu

```bash
# PostgreSQL'de veritabanı oluşturun
createdb crm_db

# Migration'ları çalıştırın
go run main.go
```

## API Endpoints

### Authentication

#### 1. User Registration

```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "password123",
  "name": "John",
  "surname": "Doe",
  "phone": "+905551234567"
}
```

#### 2. User Login

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "password123"
}
```

#### 3. Verify Email Code

```http
POST /api/v1/auth/verify-code
Content-Type: application/json

{
  "email": "user@example.com",
  "code": "1234"
}
```

#### 4. Forgot Password

```http
POST /api/v1/auth/forgot-password
Content-Type: application/json

{
  "email": "user@example.com"
}
```

#### 5. Reset Password

```http
POST /api/v1/auth/reset-password
Content-Type: application/json

{
  "token": "reset_token_here",
  "password": "newpassword123"
}
```

## Proje Yapısı

```
├── app/
│   ├── api/
│   │   └── routes/
│   │       └── auth.go          # Auth route handlers
│   └── cmd/
│       └── micro.go             # Application entry point
├── pkg/
│   ├── config/
│   │   └── config.go            # Configuration management
│   ├── constant/
│   │   └── auth.go              # Constants
│   ├── database/
│   │   ├── pg.go                # PostgreSQL connection
│   │   └── migrations.go        # Database migrations
│   ├── domains/
│   │   └── auth/
│   │       ├── repository.go    # Data access layer
│   │       └── service.go       # Business logic
│   ├── dtos/
│   │   └── user.go              # Data transfer objects
│   ├── entities/
│   │   └── user.go              # User model
│   ├── middleware/
│   │   └── middleware.go        # HTTP middleware
│   ├── server/
│   │   └── http.go              # HTTP server setup
│   └── utils/
│       └── utils.go             # Utility functions
├── main.go                      # Main application file
├── go.mod                       # Go modules
└── README.md                    # This file
```

## Çalıştırma

```bash
# Bağımlılıkları yükleyin
go mod tidy

# Uygulamayı çalıştırın
go run main.go
```

## Swagger Dokümantasyonu

API dokümantasyonuna `http://localhost:8000/docs/` adresinden erişebilirsiniz.

## Veritabanı Şeması

### User Table

- `id`: Primary key (auto-increment)
- `email`: Unique email address
- `password`: Hashed password
- `name`: User's first name
- `surname`: User's last name
- `phone`: Phone number
- `is_verified`: Email verification status
- `verification_code`: Email verification code
- `code_expires_at`: Verification code expiration
- `reset_token`: Password reset token
- `reset_expires_at`: Reset token expiration
- `created_at`, `updated_at`, `deleted_at`: Timestamps

## Güvenlik

- Passwords bcrypt ile hash'lenir
- JWT token'lar 24 saat geçerlidir
- Email verification code'lar 4 dakika geçerlidir
- Password reset token'lar 1 saat geçerlidir
- CORS yapılandırılmış
- Input validation mevcut
