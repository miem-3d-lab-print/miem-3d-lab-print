package repository

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/dto"
	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/models"
)

type ApplicationRepository interface {
	Create(tx DBTX, application *models.Application) (*models.Application, error)
	FindByID(id uuid.UUID) (*models.Application, error)
	FindByIDAndUser(id, userID uuid.UUID) (*models.Application, error)
	ListByUser(userID uuid.UUID, status string, page, perPage int) ([]*models.Application, int, error)
	ListAdmin(filter dto.ApplicationFilter) ([]*models.Application, int, error)
	CountActive(tx DBTX, userID uuid.UUID) (int, error)
	CancelAtomic(tx DBTX, id, userID uuid.UUID, filesDeleteAfter time.Time) (*models.Application, error)
	UpdateStatus(tx DBTX, id uuid.UUID, status string, rejectionReason *string, filesDeleteAfter *time.Time) (*models.Application, error)
	GenerateNumber(year int) (string, error)
	EnsureYearSequence(year int) error
}

type GORMApplicationRepository struct {
	db *gorm.DB
}

func NewGORMApplicationRepository(database *gorm.DB) *GORMApplicationRepository {
	return &GORMApplicationRepository{db: database}
}

func (repository *GORMApplicationRepository) EnsureYearSequence(year int) error {
	sequence := fmt.Sprintf("app_number_%d", year)
	return repository.db.Exec(fmt.Sprintf("CREATE SEQUENCE IF NOT EXISTS %s", sequence)).Error
}

func (repository *GORMApplicationRepository) GenerateNumber(year int) (string, error) {
	sequence := fmt.Sprintf("app_number_%d", year)
	var value int64
	if err := repository.db.Raw(fmt.Sprintf("SELECT nextval('%s')", sequence)).Scan(&value).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("%d-%04d", year, value), nil
}

func (repository *GORMApplicationRepository) Create(tx DBTX, application *models.Application) (*models.Application, error) {
	application.ID = uuid.New()
	application.Status = "new"
	application.DeadlineNotified = false
	if err := tx.Create(application).Error; err != nil {
		return nil, err
	}
	return application, nil
}

func (repository *GORMApplicationRepository) FindByID(id uuid.UUID) (*models.Application, error) {
	var application models.Application
	return entityOrNil(&application, repository.db.First(&application, "id = ?", id))
}

func (repository *GORMApplicationRepository) FindByIDAndUser(id, userID uuid.UUID) (*models.Application, error) {
	var application models.Application
	result := repository.db.Where("id = ? AND user_id = ?", id, userID).First(&application)
	return entityOrNil(&application, result)
}

func (repository *GORMApplicationRepository) CountActive(tx DBTX, userID uuid.UUID) (int, error) {
	var count int64
	result := tx.Model(&models.Application{}).
		Where("user_id = ? AND status IN ?", userID, []string{"new", "in_review", "printing"}).
		Count(&count)
	return int(count), result.Error
}

func (repository *GORMApplicationRepository) ListByUser(userID uuid.UUID, status string, page, perPage int) ([]*models.Application, int, error) {
	query := repository.db.Model(&models.Application{}).Where("user_id = ?", userID)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var applications []*models.Application
	err := query.Order("created_at DESC").Limit(perPage).Offset((page - 1) * perPage).Find(&applications).Error
	return applications, int(total), err
}

func (repository *GORMApplicationRepository) ListAdmin(filter dto.ApplicationFilter) ([]*models.Application, int, error) {
	query := repository.db.Model(&models.Application{})
	if len(filter.Statuses) > 0 {
		query = query.Where("status IN ?", filter.Statuses)
	}
	if filter.MaterialID != nil {
		query = query.Where("material_id = ?", *filter.MaterialID)
	}
	if filter.CreatedFrom != nil {
		query = query.Where("created_at >= ?", *filter.CreatedFrom)
	}
	if filter.CreatedTo != nil {
		query = query.Where("created_at <= ?", *filter.CreatedTo)
	}
	if filter.DesiredFrom != nil {
		query = query.Where("desired_date >= ?", *filter.DesiredFrom)
	}
	if filter.DesiredTo != nil {
		query = query.Where("desired_date <= ?", *filter.DesiredTo)
	}
	if filter.Search != "" {
		query = query.Where(
			"lower(snapshot_full_name) LIKE lower(?) OR number = ?",
			"%"+filter.Search+"%", filter.Search,
		)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var applications []*models.Application
	err := query.Order("created_at DESC").Limit(filter.PerPage).Offset((filter.Page - 1) * filter.PerPage).
		Find(&applications).Error
	return applications, int(total), err
}

func (repository *GORMApplicationRepository) CancelAtomic(tx DBTX, id, userID uuid.UUID, filesDeleteAfter time.Time) (*models.Application, error) {
	var application models.Application
	result := tx.Model(&application).Clauses(clause.Returning{}).
		Where("id = ? AND user_id = ? AND status = ?", id, userID, "new").
		Updates(map[string]any{"status": "cancelled", "files_delete_after": filesDeleteAfter})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &application, nil
}

func (repository *GORMApplicationRepository) UpdateStatus(tx DBTX, id uuid.UUID, status string, rejectionReason *string, filesDeleteAfter *time.Time) (*models.Application, error) {
	updates := map[string]any{
		"status":             status,
		"files_delete_after": filesDeleteAfter,
	}
	if rejectionReason != nil {
		updates["rejection_reason"] = *rejectionReason
	}

	var application models.Application
	result := tx.Model(&application).Clauses(clause.Returning{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &application, nil
}
