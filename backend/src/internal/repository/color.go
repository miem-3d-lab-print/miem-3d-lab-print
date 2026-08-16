package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/models"
)

type ColorRepository interface {
	ListByMaterial(materialID uuid.UUID, onlyActive bool) ([]*models.Color, error)
	FindByID(id, materialID uuid.UUID) (*models.Color, error)
	FindByName(materialID uuid.UUID, name string) (*models.Color, error)
	Create(materialID uuid.UUID, name string, isActive bool) (*models.Color, error)
	Update(id, materialID uuid.UUID, name *string, isActive *bool) (*models.Color, error)
}

type GORMColorRepository struct {
	db *gorm.DB
}

func NewGORMColorRepository(database *gorm.DB) *GORMColorRepository {
	return &GORMColorRepository{db: database}
}

func (repository *GORMColorRepository) ListByMaterial(materialID uuid.UUID, onlyActive bool) ([]*models.Color, error) {
	query := repository.db.Where("material_id = ?", materialID)
	if onlyActive {
		query = query.Where("is_active = true")
	}
	var colors []*models.Color
	err := query.Order("name").Find(&colors).Error
	return colors, err
}

func (repository *GORMColorRepository) FindByID(id, materialID uuid.UUID) (*models.Color, error) {
	var color models.Color
	result := repository.db.Where("id = ? AND material_id = ?", id, materialID).First(&color)
	return entityOrNil(&color, result)
}

func (repository *GORMColorRepository) FindByName(materialID uuid.UUID, name string) (*models.Color, error) {
	var color models.Color
	result := repository.db.Where("material_id = ? AND lower(name) = lower(?)", materialID, name).First(&color)
	return entityOrNil(&color, result)
}

func (repository *GORMColorRepository) Create(materialID uuid.UUID, name string, isActive bool) (*models.Color, error) {
	color := models.Color{ID: uuid.New(), MaterialID: materialID, Name: name, IsActive: isActive}
	if err := repository.db.Create(&color).Error; err != nil {
		return nil, err
	}
	return &color, nil
}

func (repository *GORMColorRepository) Update(id, materialID uuid.UUID, name *string, isActive *bool) (*models.Color, error) {
	updates := make(map[string]any)
	if name != nil {
		updates["name"] = *name
	}
	if isActive != nil {
		updates["is_active"] = *isActive
	}
	if len(updates) > 0 {
		result := repository.db.Model(&models.Color{}).
			Where("id = ? AND material_id = ?", id, materialID).
			Updates(updates)
		if result.Error != nil {
			return nil, result.Error
		}
	}
	return repository.FindByID(id, materialID)
}
