package repository

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/models"
)

type UserRepository interface {
	FindByEmail(email string) (*models.User, error)
	FindByID(id uuid.UUID) (*models.User, error)
	FindByIDs(ids []uuid.UUID) (map[uuid.UUID]*models.User, error)
	Upsert(email string) (*models.User, error)
	UpdateProfile(id uuid.UUID, fullName, telegram, max *string) (*models.User, error)
	GiveConsent(id uuid.UUID) (*models.User, error)
	SearchByEmail(query string, limit int) ([]*models.User, error)
	ListAdmins() ([]*models.User, error)
	ListApplicationNotificationRecipients() ([]*models.User, error)
	SetRole(tx DBTX, id uuid.UUID, role string) (*models.User, error)
	SetApplicationNotifications(tx DBTX, id uuid.UUID, enabled bool) (*models.User, error)
	CountAdmins(tx DBTX) (int, error)
}

type GORMUserRepository struct {
	db *gorm.DB
}

func NewGORMUserRepository(database *gorm.DB) *GORMUserRepository {
	return &GORMUserRepository{db: database}
}

func (repository *GORMUserRepository) FindByEmail(email string) (*models.User, error) {
	var user models.User
	result := repository.db.Where("lower(email) = lower(?)", email).First(&user)
	return entityOrNil(&user, result)
}

func (repository *GORMUserRepository) FindByID(id uuid.UUID) (*models.User, error) {
	var user models.User
	result := repository.db.First(&user, "id = ?", id)
	return entityOrNil(&user, result)
}

func (repository *GORMUserRepository) FindByIDs(ids []uuid.UUID) (map[uuid.UUID]*models.User, error) {
	result := make(map[uuid.UUID]*models.User, len(ids))
	if len(ids) == 0 {
		return result, nil
	}

	var users []*models.User
	if err := repository.db.Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	for _, user := range users {
		result[user.ID] = user
	}
	return result, nil
}

func (repository *GORMUserRepository) Upsert(email string) (*models.User, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	user := models.User{
		ID:           uuid.New(),
		Email:        normalizedEmail,
		Role:         models.UserRoleUser,
		ConsentGiven: false,
	}
	if err := repository.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&user).Error; err != nil {
		return nil, err
	}
	return repository.FindByEmail(normalizedEmail)
}

func (repository *GORMUserRepository) UpdateProfile(id uuid.UUID, fullName, telegram, max *string) (*models.User, error) {
	updates := map[string]any{
		"telegram":   telegram,
		"max":        max,
		"updated_at": time.Now(),
	}
	if fullName != nil {
		updates["full_name"] = fullName
	}

	result := repository.db.Model(&models.User{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	return repository.FindByID(id)
}

func (repository *GORMUserRepository) GiveConsent(id uuid.UUID) (*models.User, error) {
	result := repository.db.Model(&models.User{}).Where("id = ?", id).Updates(map[string]any{
		"consent_given":    true,
		"consent_given_at": gorm.Expr("COALESCE(consent_given_at, now())"),
		"updated_at":       time.Now(),
	})
	if result.Error != nil {
		return nil, result.Error
	}
	return repository.FindByID(id)
}

func (repository *GORMUserRepository) SearchByEmail(query string, limit int) ([]*models.User, error) {
	var users []*models.User
	err := repository.db.Where("lower(email) LIKE lower(?)", "%"+query+"%").Order("email").Limit(limit).Find(&users).Error
	return users, err
}

func (repository *GORMUserRepository) ListAdmins() ([]*models.User, error) {
	var users []*models.User
	err := repository.db.Where("role = ?", models.UserRoleAdmin).Order("email").Find(&users).Error
	return users, err
}

func (repository *GORMUserRepository) ListApplicationNotificationRecipients() ([]*models.User, error) {
	var users []*models.User
	err := repository.db.
		Where("role = ? AND application_notifications = ?", models.UserRoleAdmin, true).
		Order("email").
		Find(&users).Error
	return users, err
}

func (repository *GORMUserRepository) SetRole(tx DBTX, id uuid.UUID, role string) (*models.User, error) {
	updates := map[string]any{"role": role, "updated_at": time.Now()}
	if role != models.UserRoleAdmin {
		updates["application_notifications"] = false
	}
	if err := tx.Model(&models.User{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}

	var user models.User
	return entityOrNil(&user, tx.First(&user, "id = ?", id))
}

func (repository *GORMUserRepository) SetApplicationNotifications(tx DBTX, id uuid.UUID, enabled bool) (*models.User, error) {
	result := tx.Model(&models.User{}).
		Where("id = ? AND role = ?", id, models.UserRoleAdmin).
		Updates(map[string]any{
			"application_notifications": enabled,
			"updated_at":                time.Now(),
		})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}

	var user models.User
	return entityOrNil(&user, tx.First(&user, "id = ?", id))
}

func (repository *GORMUserRepository) CountAdmins(tx DBTX) (int, error) {
	var adminIDs []uuid.UUID
	result := tx.Model(&models.User{}).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("role = ?", models.UserRoleAdmin).
		Pluck("id", &adminIDs)
	return len(adminIDs), result.Error
}
