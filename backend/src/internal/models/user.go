package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	UserRoleUser  = "user"
	UserRoleAdmin = "admin"
)

type User struct {
	ID                       uuid.UUID
	Email                    string
	FullName                 *string
	Telegram                 *string
	Max                      *string
	TelegramID               *int64
	Role                     string
	ApplicationNotifications bool
	ConsentGiven             bool
	ConsentGivenAt           *time.Time
	CreatedAt                time.Time
	UpdatedAt                time.Time
}
