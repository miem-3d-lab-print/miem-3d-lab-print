package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/models"
)

type FileRepository interface {
	Create(tx DBTX, file *models.File) (*models.File, error)
	ListByApplication(applicationID uuid.UUID) ([]*models.File, error)
	FindByID(id, applicationID uuid.UUID) (*models.File, error)
	CountByApplication(applicationID uuid.UUID) (int, error)
	CountsByApplicationIDs(ids []uuid.UUID) (map[uuid.UUID]int, error)
	CreatePending(file *models.PendingFile) (*models.PendingFile, error)
	FindPendingByIDAndUser(id, userID uuid.UUID) (*models.PendingFile, error)
	CountPendingByUser(userID uuid.UUID) (int, error)
	ListPendingForUpdate(tx DBTX, userID uuid.UUID, ids []uuid.UUID) ([]*models.PendingFile, error)
	DeletePending(tx DBTX, ids []uuid.UUID) error
	DeletePendingByID(id uuid.UUID) error
	ListExpiredPending(limit int) ([]*models.PendingFile, error)
}

func (repository *GORMFileRepository) CreatePending(file *models.PendingFile) (*models.PendingFile, error) {
	if file.ID == uuid.Nil {
		file.ID = uuid.New()
	}
	if err := repository.db.Create(file).Error; err != nil {
		return nil, err
	}
	return file, nil
}

func (repository *GORMFileRepository) FindPendingByIDAndUser(id, userID uuid.UUID) (*models.PendingFile, error) {
	var file models.PendingFile
	result := repository.db.Where("id = ? AND user_id = ? AND expires_at > ?", id, userID, time.Now()).First(&file)
	return entityOrNil(&file, result)
}

func (repository *GORMFileRepository) CountPendingByUser(userID uuid.UUID) (int, error) {
	var count int64
	err := repository.db.Model(&models.PendingFile{}).
		Where("user_id = ? AND expires_at > ?", userID, time.Now()).
		Count(&count).Error
	return int(count), err
}

func (repository *GORMFileRepository) ListPendingForUpdate(tx DBTX, userID uuid.UUID, ids []uuid.UUID) ([]*models.PendingFile, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var files []*models.PendingFile
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ? AND id IN ? AND expires_at > ?", userID, ids, time.Now()).
		Find(&files).Error
	return files, err
}

func (repository *GORMFileRepository) DeletePending(tx DBTX, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	return tx.Where("id IN ?", ids).Delete(&models.PendingFile{}).Error
}

func (repository *GORMFileRepository) DeletePendingByID(id uuid.UUID) error {
	return repository.db.Delete(&models.PendingFile{}, "id = ?", id).Error
}

func (repository *GORMFileRepository) ListExpiredPending(limit int) ([]*models.PendingFile, error) {
	var files []*models.PendingFile
	err := repository.db.Where("expires_at <= ?", time.Now()).Order("expires_at").Limit(limit).Find(&files).Error
	return files, err
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
