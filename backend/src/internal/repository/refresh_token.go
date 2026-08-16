package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/models"
)

type RefreshTokenRepository interface {
	Create(userID uuid.UUID, tokenHash string, expiresAt time.Time) (*models.RefreshToken, error)
	FindByHash(tokenHash string) (*models.RefreshToken, error)
	Revoke(id uuid.UUID, replacedByID *uuid.UUID) error
	RevokeAllForUser(userID uuid.UUID) error
}

type GORMRefreshTokenRepository struct {
	db *gorm.DB
}

func NewGORMRefreshTokenRepository(database *gorm.DB) *GORMRefreshTokenRepository {
	return &GORMRefreshTokenRepository{db: database}
}

func (repository *GORMRefreshTokenRepository) Create(userID uuid.UUID, tokenHash string, expiresAt time.Time) (*models.RefreshToken, error) {
	token := models.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	}
	if err := repository.db.Create(&token).Error; err != nil {
		return nil, err
	}
	return &token, nil
}

func (repository *GORMRefreshTokenRepository) FindByHash(tokenHash string) (*models.RefreshToken, error) {
	var token models.RefreshToken
	return entityOrNil(&token, repository.db.Where("token_hash = ?", tokenHash).First(&token))
}

func (repository *GORMRefreshTokenRepository) Revoke(id uuid.UUID, replacedByID *uuid.UUID) error {
	return repository.db.Model(&models.RefreshToken{}).Where("id = ?", id).Updates(map[string]any{
		"revoked_at": time.Now(), "replaced_by_token_id": replacedByID,
	}).Error
}

func (repository *GORMRefreshTokenRepository) RevokeAllForUser(userID uuid.UUID) error {
	return repository.db.Model(&models.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", time.Now()).Error
}
