package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/models"
)

type StatusHistoryRepository interface {
	Create(tx DBTX, history *models.StatusHistory) (*models.StatusHistory, error)
	ListByApplication(applicationID uuid.UUID) ([]*models.StatusHistory, error)
}

type GORMStatusHistoryRepository struct {
	db *gorm.DB
}

func NewGORMStatusHistoryRepository(database *gorm.DB) *GORMStatusHistoryRepository {
	return &GORMStatusHistoryRepository{db: database}
}

func (repository *GORMStatusHistoryRepository) Create(tx DBTX, history *models.StatusHistory) (*models.StatusHistory, error) {
	history.ID = uuid.New()
	if err := tx.Create(history).Error; err != nil {
		return nil, err
	}
	return history, nil
}

func (repository *GORMStatusHistoryRepository) ListByApplication(applicationID uuid.UUID) ([]*models.StatusHistory, error) {
	var history []*models.StatusHistory
	err := repository.db.Where("application_id = ?", applicationID).Order("created_at").Find(&history).Error
	return history, err
}
