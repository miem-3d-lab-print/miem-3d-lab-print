package dto

import "time"

type AdminUserItem struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FullName  *string   `json:"full_name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type AdminUsersResponse struct {
	Items []*AdminUserItem `json:"items"`
}

type SetRoleRequest struct {
	Role string `json:"role"`
}

type SetRoleResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type ByMaterialStat struct {
	MaterialName string `json:"material_name"`
	Count        int    `json:"count"`
}

type StatsResponse struct {
	Period struct {
		DateFrom string `json:"date_from"`
		DateTo   string `json:"date_to"`
	} `json:"period"`
	TotalApplications  int               `json:"total_applications"`
	AvgCompletionHours *float64          `json:"avg_completion_hours"`
	CompletedCount     int               `json:"completed_count"`
	ByMaterial         []*ByMaterialStat `json:"by_material"`
	ByStatusCurrent    map[string]int    `json:"by_status_current"`
}

type UploadFileResponse struct {
	ID        string    `json:"id"`
	Filename  string    `json:"filename"`
	Size      int       `json:"size"`
	Format    string    `json:"format"`
	CreatedAt time.Time `json:"created_at"`
}
