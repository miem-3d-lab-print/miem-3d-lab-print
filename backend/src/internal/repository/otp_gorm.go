package repository

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/models"
)

type GORMOTPRepository struct {
	db *gorm.DB
}

func NewGORMOTPRepository(database *gorm.DB) *GORMOTPRepository {
	return &GORMOTPRepository{db: database}
}

func (repository *GORMOTPRepository) FindActive(email string) (*models.OTPCode, error) {
	var otp models.OTPCode
	result := repository.db.Where(
		"email = ? AND is_used = false AND expires_at > ?",
		strings.ToLower(email), time.Now(),
	).Order("created_at DESC").First(&otp)
	return entityOrNil(&otp, result)
}

func (repository *GORMOTPRepository) FindByEmail(email string) (*models.OTPCode, error) {
	var otp models.OTPCode
	result := repository.db.Where("email = ? AND is_used = false", strings.ToLower(email)).
		Order("created_at DESC").First(&otp)
	return entityOrNil(&otp, result)
}

func (repository *GORMOTPRepository) Create(email, codeHash string, expiresAt time.Time) error {
	otp := models.OTPCode{
		ID:        uuid.New(),
		Email:     strings.ToLower(email),
		CodeHash:  codeHash,
		ExpiresAt: expiresAt,
	}
	return repository.db.Create(&otp).Error
}

func (repository *GORMOTPRepository) Invalidate(id uuid.UUID) error {
	return repository.db.Model(&models.OTPCode{}).Where("id = ?", id).Update("is_used", true).Error
}

func (repository *GORMOTPRepository) UpdateAttempts(id uuid.UUID, attempts int16, blockedUntil *time.Time) error {
	return repository.db.Model(&models.OTPCode{}).Where("id = ?", id).Updates(map[string]any{
		"attempts": attempts, "blocked_until": blockedUntil,
	}).Error
}

func (repository *GORMOTPRepository) MarkUsed(id uuid.UUID) error {
	return repository.Invalidate(id)
}
