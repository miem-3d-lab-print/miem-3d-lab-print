package repository

import (
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/models"
)

type OTPRepository interface {
	FindActive(email string) (*models.OTPCode, error)
	FindByEmail(email string) (*models.OTPCode, error)
	Create(email, codeHash string, expiresAt time.Time) error
	Invalidate(id uuid.UUID) error
	UpdateAttempts(id uuid.UUID, attempts int16, blockedUntil *time.Time) error
	MarkUsed(id uuid.UUID) error
}

type InMemoryOTPRepository struct {
	mu      sync.Mutex
	records []*models.OTPCode
}

func NewInMemoryOTPRepository() *InMemoryOTPRepository {
	return &InMemoryOTPRepository{}
}

func (r *InMemoryOTPRepository) FindActive(email string) (*models.OTPCode, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	for _, rec := range r.records {
		if rec.Email == email && !rec.IsUsed && rec.ExpiresAt.After(now) {
			return rec, nil
		}
	}
	return nil, nil
}

func (r *InMemoryOTPRepository) FindByEmail(email string) (*models.OTPCode, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var latest *models.OTPCode
	for _, rec := range r.records {
		if rec.Email == email && !rec.IsUsed {
			if latest == nil || rec.CreatedAt.After(latest.CreatedAt) {
				latest = rec
			}
		}
	}
	return latest, nil
}

func (r *InMemoryOTPRepository) Create(email, codeHash string, expiresAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = append(r.records, &models.OTPCode{
		ID:        uuid.New(),
		Email:     email,
		CodeHash:  codeHash,
		Attempts:  0,
		ExpiresAt: expiresAt,
		IsUsed:    false,
		CreatedAt: time.Now(),
	})
	return nil
}

func (r *InMemoryOTPRepository) Invalidate(id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, rec := range r.records {
		if rec.ID == id {
			rec.IsUsed = true
			return nil
		}
	}
	return nil
}

func (r *InMemoryOTPRepository) UpdateAttempts(id uuid.UUID, attempts int16, blockedUntil *time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, rec := range r.records {
		if rec.ID == id {
			rec.Attempts = attempts
			rec.BlockedUntil = blockedUntil
			return nil
		}
	}
	return nil
}

func (r *InMemoryOTPRepository) MarkUsed(id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, rec := range r.records {
		if rec.ID == id {
			rec.IsUsed = true
			return nil
		}
	}
	return nil
}
