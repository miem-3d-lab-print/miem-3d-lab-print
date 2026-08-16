package models

import (
	"time"

	"github.com/google/uuid"
)

type OTPCode struct {
	ID           uuid.UUID  `db:"id"`
	Email        string     `db:"email"`
	CodeHash     string     `db:"code_hash"`
	Attempts     int16      `db:"attempts"`
	ExpiresAt    time.Time  `db:"expires_at"`
	BlockedUntil *time.Time `db:"blocked_until"`
	IsUsed       bool       `db:"is_used"`
	CreatedAt    time.Time  `db:"created_at"`
}
