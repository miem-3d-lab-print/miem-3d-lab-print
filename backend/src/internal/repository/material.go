package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/models"
)

type MaterialRepository interface {
	ListActive() ([]*models.Material, error)
	ListAll() ([]*models.Material, error)
	FindByID(id uuid.UUID) (*models.Material, error)
	FindByName(name string) (*models.Material, error)
	Create(name, description string, isActive bool) (*models.Material, error)
	Update(id uuid.UUID, name, description *string, isActive *bool) (*models.Material, error)
}

type GORMMaterialRepository struct {
	db *gorm.DB
}

func NewGORMMaterialRepository(database *gorm.DB) *GORMMaterialRepository {
	return &GORMMaterialRepository{db: database}
}

func (repository *GORMMaterialRepository) ListActive() ([]*models.Material, error) {
	var materials []*models.Material
	err := repository.db.Where("is_active = true").Order("name").Find(&materials).Error
	return materials, err
}

func (repository *GORMMaterialRepository) ListAll() ([]*models.Material, error) {
	var materials []*models.Material
	err := repository.db.Order("name").Find(&materials).Error
	return materials, err
}

func (repository *GORMMaterialRepository) FindByID(id uuid.UUID) (*models.Material, error) {
	var material models.Material
	return entityOrNil(&material, repository.db.First(&material, "id = ?", id))
}

func (repository *GORMMaterialRepository) FindByName(name string) (*models.Material, error) {
	var material models.Material
	return entityOrNil(&material, repository.db.Where("lower(name) = lower(?)", name).First(&material))
}

func (repository *GORMMaterialRepository) Create(name, description string, isActive bool) (*models.Material, error) {
	material := models.Material{ID: uuid.New(), Name: name, Description: description, IsActive: isActive}
	if err := repository.db.Create(&material).Error; err != nil {
		return nil, err
	}
	return &material, nil
}

func (repository *GORMMaterialRepository) Update(id uuid.UUID, name, description *string, isActive *bool) (*models.Material, error) {
	updates := make(map[string]any)
	if name != nil {
		updates["name"] = *name
	}
	if description != nil {
		updates["description"] = *description
	}
	if isActive != nil {
		updates["is_active"] = *isActive
	}
	if len(updates) > 0 {
		if err := repository.db.Model(&models.Material{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			return nil, err
		}
	}
	return repository.FindByID(id)
}
