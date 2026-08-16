package dto

type ColorResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	IsActive *bool  `json:"is_active,omitempty"`
}

type MaterialResponse struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	IsActive    *bool            `json:"is_active,omitempty"`
	Colors      []*ColorResponse `json:"colors"`
}

type CreateMaterialRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsActive    *bool  `json:"is_active"`
}

type PatchMaterialRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	IsActive    *bool   `json:"is_active"`
}

type CreateColorRequest struct {
	Name     string `json:"name"`
	IsActive *bool  `json:"is_active"`
}

type PatchColorRequest struct {
	Name     *string `json:"name"`
	IsActive *bool   `json:"is_active"`
}
