package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/models"
)

type FileRepository interface {
	Create(tx DBTX, file *models.File) (*models.File, error)
	ListByApplication(applicationID uuid.UUID) ([]*models.File, error)
	FindByID(id, applicationID uuid.UUID) (*models.File, error)
	CountByApplication(applicationID uuid.UUID) (int, error)
	CountsByApplicationIDs(ids []uuid.UUID) (map[uuid.UUID]int, error)
}

type GORMFileRepository struct {
	db *gorm.DB
}

func NewGORMFileRepository(database *gorm.DB) *GORMFileRepository {
	return &GORMFileRepository{db: database}
}

func (repository *GORMFileRepository) Create(tx DBTX, file *models.File) (*models.File, error) {
	file.ID = uuid.New()
	if err := tx.Create(file).Error; err != nil {
		return nil, err
	}
	return file, nil
}

func (repository *GORMFileRepository) ListByApplication(applicationID uuid.UUID) ([]*models.File, error) {
	var files []*models.File
	err := repository.db.Where("application_id = ?", applicationID).Order("created_at").Find(&files).Error
	return files, err
}

func (repository *GORMFileRepository) FindByID(id, applicationID uuid.UUID) (*models.File, error) {
	var file models.File
	result := repository.db.Where("id = ? AND application_id = ?", id, applicationID).First(&file)
	return entityOrNil(&file, result)
}

func (repository *GORMFileRepository) CountByApplication(applicationID uuid.UUID) (int, error) {
	var count int64
	result := repository.db.Model(&models.File{}).Where("application_id = ?", applicationID).Count(&count)
	return int(count), result.Error
}

func (repository *GORMFileRepository) CountsByApplicationIDs(ids []uuid.UUID) (map[uuid.UUID]int, error) {
	result := make(map[uuid.UUID]int, len(ids))
	if len(ids) == 0 {
		return result, nil
	}

	type countRow struct {
		ApplicationID uuid.UUID
		Count         int
	}
	var rows []countRow
	err := repository.db.Model(&models.File{}).
		Select("application_id, count(*) AS count").
		Where("application_id IN ?", ids).
		Group("application_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.ApplicationID] = row.Count
	}
	return result, nil
}
