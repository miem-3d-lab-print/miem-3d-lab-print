package models

import (
	"time"

	"github.com/google/uuid"
)

type Application struct {
	ID                   uuid.UUID
	Number               string
	Title                string
	UserID               uuid.UUID
	SnapshotFullName     string
	SnapshotEmail        string
	Position             string
	Purpose              string
	MaterialID           uuid.UUID
	SnapshotMaterialName string
	ColorMatters         bool
	ColorID              *uuid.UUID
	SnapshotColorName    *string
	DesiredDate          time.Time
	Comment              *string
	FileURL              *string
	Status               string
	RejectionReason      *string
	FilesDeleteAfter     *time.Time
	DeadlineNotified     bool
	CreatedAt            time.Time
}

type File struct {
	ID            uuid.UUID
	ApplicationID uuid.UUID
	Filename      string
	StoragePath   string
	Size          int
	Format        string
	DeletedAt     *time.Time
	CreatedAt     time.Time
}

type StatusHistory struct {
	ID            uuid.UUID
	ApplicationID uuid.UUID
	Status        string
	Comment       *string
	ChangedBy     uuid.UUID
	CreatedAt     time.Time
}

func (StatusHistory) TableName() string {
	return "status_history"
}
