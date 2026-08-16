package services

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/dto"
	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/models"
	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/repository"
)

type MaterialService struct {
	materialRepo repository.MaterialRepository
	colorRepo    repository.ColorRepository
}

func NewMaterialService(materialRepo repository.MaterialRepository, colorRepo repository.ColorRepository) *MaterialService {
	return &MaterialService{materialRepo: materialRepo, colorRepo: colorRepo}
}

func colorToDTO(c *models.Color, includeActive bool) *dto.ColorResponse {
	resp := &dto.ColorResponse{
		ID:   c.ID.String(),
		Name: c.Name,
	}
	if includeActive {
		resp.IsActive = &c.IsActive
	}
	return resp
}

func (s *MaterialService) buildMaterialResponse(m *models.Material, includeInactive bool) (*dto.MaterialResponse, error) {
	colors, err := s.colorRepo.ListByMaterial(m.ID, !includeInactive)
	if err != nil {
		return nil, err
	}
	colorDTOs := make([]*dto.ColorResponse, 0, len(colors))
	for _, c := range colors {
		colorDTOs = append(colorDTOs, colorToDTO(c, includeInactive))
	}
	resp := &dto.MaterialResponse{
		ID:          m.ID.String(),
		Name:        m.Name,
		Description: m.Description,
		Colors:      colorDTOs,
	}
	if includeInactive {
		resp.IsActive = &m.IsActive
	}
	return resp, nil
}

func (s *MaterialService) ListActive() ([]*dto.MaterialResponse, error) {
	materials, err := s.materialRepo.ListActive()
	if err != nil {
		return nil, fmt.Errorf("list materials: %w", err)
	}
	result := make([]*dto.MaterialResponse, 0, len(materials))
	for _, m := range materials {
		r, err := s.buildMaterialResponse(m, false)
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, nil
}

func (s *MaterialService) ListAll() ([]*dto.MaterialResponse, error) {
	materials, err := s.materialRepo.ListAll()
	if err != nil {
		return nil, fmt.Errorf("list all materials: %w", err)
	}
	result := make([]*dto.MaterialResponse, 0, len(materials))
	for _, m := range materials {
		r, err := s.buildMaterialResponse(m, true)
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, nil
}

func (s *MaterialService) Create(name, description string, isActive bool) (*dto.MaterialResponse, error) {
	existing, err := s.materialRepo.FindByName(name)
	if err != nil {
		return nil, fmt.Errorf("find by name: %w", err)
	}
	if existing != nil {
		return nil, &ErrMaterialNameExists{ExistingID: existing.ID.String()}
	}
	m, err := s.materialRepo.Create(name, description, isActive)
	if err != nil {
		return nil, fmt.Errorf("create material: %w", err)
	}
	return s.buildMaterialResponse(m, true)
}

func (s *MaterialService) Update(id uuid.UUID, name, description *string, isActive *bool) (*dto.MaterialResponse, error) {
	if name != nil {
		existing, err := s.materialRepo.FindByName(*name)
		if err != nil {
			return nil, fmt.Errorf("find by name: %w", err)
		}
		if existing != nil && existing.ID != id {
			return nil, &ErrMaterialNameExists{ExistingID: existing.ID.String()}
		}
	}
	m, err := s.materialRepo.Update(id, name, description, isActive)
	if err != nil {
		return nil, fmt.Errorf("update material: %w", err)
	}
	if m == nil {
		return nil, &ErrMaterialNotFound{}
	}
	return s.buildMaterialResponse(m, true)
}

func (s *MaterialService) CreateColor(materialID uuid.UUID, name string, isActive bool) (*dto.ColorResponse, error) {
	mat, err := s.materialRepo.FindByID(materialID)
	if err != nil {
		return nil, fmt.Errorf("find material: %w", err)
	}
	if mat == nil {
		return nil, &ErrMaterialNotFound{}
	}
	existing, err := s.colorRepo.FindByName(materialID, name)
	if err != nil {
		return nil, fmt.Errorf("find color by name: %w", err)
	}
	if existing != nil {
		return nil, &ErrColorNameExists{}
	}
	c, err := s.colorRepo.Create(materialID, name, isActive)
	if err != nil {
		return nil, fmt.Errorf("create color: %w", err)
	}
	return colorToDTO(c, true), nil
}

func (s *MaterialService) UpdateColor(materialID, colorID uuid.UUID, name *string, isActive *bool) (*dto.ColorResponse, error) {
	mat, err := s.materialRepo.FindByID(materialID)
	if err != nil {
		return nil, fmt.Errorf("find material: %w", err)
	}
	if mat == nil {
		return nil, &ErrMaterialNotFound{}
	}
	if name != nil {
		existing, err := s.colorRepo.FindByName(materialID, *name)
		if err != nil {
			return nil, fmt.Errorf("find color by name: %w", err)
		}
		if existing != nil && existing.ID != colorID {
			return nil, &ErrColorNameExists{}
		}
	}
	c, err := s.colorRepo.Update(colorID, materialID, name, isActive)
	if err != nil {
		return nil, fmt.Errorf("update color: %w", err)
	}
	if c == nil {
		return nil, &ErrColorNotFound{}
	}
	return colorToDTO(c, true), nil
}

// FindByID returns the raw material model. Used by other services to inspect material state.
func (s *MaterialService) FindByID(id uuid.UUID) (*models.Material, error) {
	return s.materialRepo.FindByID(id)
}

// GetMaterialForApplication resolves material and optional color for creating an application.
// Returns snapshot names for material and color.
func (s *MaterialService) GetMaterialForApplication(materialID uuid.UUID, colorMatters bool, colorID *uuid.UUID) (
	matName string, colorName *string, err error,
) {
	mat, err := s.materialRepo.FindByID(materialID)
	if err != nil {
		return "", nil, fmt.Errorf("find material: %w", err)
	}
	if mat == nil {
		return "", nil, &ErrMaterialNotFound{}
	}
	if !mat.IsActive {
		return "", nil, &ErrMaterialNotAvailable{}
	}
	matName = mat.Name
	if !colorMatters {
		return matName, nil, nil
	}
	if colorID == nil {
		return "", nil, &ErrColorRequired{}
	}
	color, err := s.colorRepo.FindByID(*colorID, materialID)
	if err != nil {
		return "", nil, fmt.Errorf("find color: %w", err)
	}
	if color == nil {
		return "", nil, &ErrColorNotFound{}
	}
	if !color.IsActive {
		return "", nil, &ErrColorNotAvailable{}
	}
	cn := color.Name
	return matName, &cn, nil
}
