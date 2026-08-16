package dto

import (
	"time"

	"github.com/google/uuid"
)

// ApplicationFilter is the input to the admin application list use case.
// It is defined here (not in repository) so that both the service and the
// handler layer can reference it without importing the infrastructure package.
type ApplicationFilter struct {
	Statuses    []string
	MaterialID  *uuid.UUID
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	DesiredFrom *time.Time
	DesiredTo   *time.Time
	Search      string
	Page        int
	PerPage     int
}

type PaginationMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type ApplicationListItem struct {
	ID           string    `json:"id"`
	Number       string    `json:"number"`
	Status       string    `json:"status"`
	MaterialName string    `json:"material_name"`
	ColorName    *string   `json:"color_name"`
	DesiredDate  string    `json:"desired_date"`
	CreatedAt    time.Time `json:"created_at"`
	FilesCount   int       `json:"files_count"`
}

type ApplicationListResponse struct {
	Items []*ApplicationListItem `json:"items"`
	Meta  PaginationMeta         `json:"meta"`
}

type FileItem struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	Size     int    `json:"size"`
	Format   string `json:"format"`
}

type StatusHistoryItemUser struct {
	Status    string    `json:"status"`
	Comment   *string   `json:"comment"`
	ChangedBy string    `json:"changed_by_role"`
	CreatedAt time.Time `json:"created_at"`
}

type MaterialRef struct {
	ID           string `json:"id"`
	SnapshotName string `json:"snapshot_name"`
}

type ColorRef struct {
	ID           string `json:"id"`
	SnapshotName string `json:"snapshot_name"`
}

type ApplicationDetailUser struct {
	ID               string                   `json:"id"`
	Number           string                   `json:"number"`
	Status           string                   `json:"status"`
	RejectionReason  *string                  `json:"rejection_reason"`
	Position         string                   `json:"position"`
	Purpose          string                   `json:"purpose"`
	Material         MaterialRef              `json:"material"`
	ColorMatters     bool                     `json:"color_matters"`
	Color            *ColorRef                `json:"color"`
	DesiredDate      string                   `json:"desired_date"`
	Comment          *string                  `json:"comment"`
	Files            []*FileItem              `json:"files"`
	FilesDeleteAfter *time.Time               `json:"files_delete_after"`
	StatusHistory    []*StatusHistoryItemUser `json:"status_history"`
	CreatedAt        time.Time                `json:"created_at"`
}

type CreateApplicationResponse struct {
	ID        string    `json:"id"`
	Number    string    `json:"number"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type CancelApplicationResponse struct {
	ID               string     `json:"id"`
	Status           string     `json:"status"`
	FilesDeleteAfter *time.Time `json:"files_delete_after"`
}

// Admin DTOs

type AdminApplicant struct {
	UserID           string  `json:"user_id"`
	SnapshotFullName string  `json:"snapshot_full_name"`
	SnapshotEmail    string  `json:"snapshot_email"`
	Telegram         *string `json:"telegram"`
	Max              *string `json:"max"`
}

type MaterialRefAdmin struct {
	ID           string `json:"id"`
	SnapshotName string `json:"snapshot_name"`
	IsActiveNow  bool   `json:"is_active_now"`
}

type StatusHistoryChangedBy struct {
	ID       string `json:"id"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
}

type StatusHistoryItemAdmin struct {
	Status    string                  `json:"status"`
	Comment   *string                 `json:"comment"`
	ChangedBy *StatusHistoryChangedBy `json:"changed_by"`
	CreatedAt time.Time               `json:"created_at"`
}

type ApplicationDetailAdmin struct {
	ID              string                    `json:"id"`
	Number          string                    `json:"number"`
	Applicant       AdminApplicant            `json:"applicant"`
	Position        string                    `json:"position"`
	Purpose         string                    `json:"purpose"`
	Material        MaterialRefAdmin          `json:"material"`
	ColorMatters    bool                      `json:"color_matters"`
	Color           *ColorRef                 `json:"color"`
	DesiredDate     string                    `json:"desired_date"`
	Comment         *string                   `json:"comment"`
	Status          string                    `json:"status"`
	RejectionReason *string                   `json:"rejection_reason"`
	Files           []*FileItem               `json:"files"`
	StatusHistory   []*StatusHistoryItemAdmin `json:"status_history"`
	CreatedAt       time.Time                 `json:"created_at"`
}

type AdminApplicationListItem struct {
	ID           string    `json:"id"`
	Number       string    `json:"number"`
	FullName     string    `json:"full_name"`
	CreatedAt    time.Time `json:"created_at"`
	DesiredDate  string    `json:"desired_date"`
	MaterialName string    `json:"material_name"`
	Status       string    `json:"status"`
	DeadlineSoon bool      `json:"deadline_soon"`
}

type AdminApplicationListResponse struct {
	Items []*AdminApplicationListItem `json:"items"`
	Meta  PaginationMeta              `json:"meta"`
}

type ChangeStatusRequest struct {
	Status          string  `json:"status"`
	Comment         *string `json:"comment"`
	RejectionReason *string `json:"rejection_reason"`
}

type ChangeStatusResponse struct {
	ID               string     `json:"id"`
	Status           string     `json:"status"`
	FilesDeleteAfter *time.Time `json:"files_delete_after"`
}
