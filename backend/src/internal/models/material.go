package models

import (
	"time"

	"github.com/google/uuid"
)

type Material struct {
	ID          uuid.UUID
	Name        string
	Description string
	IsActive    bool
	CreatedAt   time.Time
}

type Color struct {
	ID         uuid.UUID
	MaterialID uuid.UUID
	Name       string
	IsActive   bool
	CreatedAt  time.Time
}
