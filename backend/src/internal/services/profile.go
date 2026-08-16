package services

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/dto"
	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/models"
	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/repository"
)

type ProfileService struct {
	userRepo repository.UserRepository
}

func NewProfileService(userRepo repository.UserRepository) *ProfileService {
	return &ProfileService{userRepo: userRepo}
}

func userToProfileResponse(u *models.User) dto.ProfileResponse {
	return dto.ProfileResponse{
		ID:           u.ID.String(),
		Email:        u.Email,
		FullName:     u.FullName,
		Telegram:     u.Telegram,
		Max:          u.Max,
		Role:         u.Role,
		ConsentGiven: u.ConsentGiven,
		ConsentDate:  u.ConsentGivenAt,
		CreatedAt:    u.CreatedAt,
	}
}

func (s *ProfileService) GetProfile(userID uuid.UUID) (dto.ProfileResponse, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return dto.ProfileResponse{}, fmt.Errorf("find user: %w", err)
	}
	if user == nil {
		return dto.ProfileResponse{}, &ErrProfileNotFound{}
	}
	return userToProfileResponse(user), nil
}

func (s *ProfileService) PatchProfile(userID uuid.UUID, body json.RawMessage) (dto.ProfileResponse, error) {
	// Check for forbidden fields
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return dto.ProfileResponse{}, &ErrInvalidFullName{}
	}
	for _, forbidden := range []string{"email", "role", "id", "consent_given", "created_at", "updated_at"} {
		if _, ok := raw[forbidden]; ok {
			return dto.ProfileResponse{}, &ErrFieldNotEditable{Field: forbidden}
		}
	}

	var req dto.PatchProfileRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return dto.ProfileResponse{}, &ErrInvalidFullName{}
	}

	// Validate full_name
	if req.FullName != nil && strings.TrimSpace(*req.FullName) == "" {
		return dto.ProfileResponse{}, &ErrInvalidFullName{}
	}

	// Load current user to check contact invariant
	current, err := s.userRepo.FindByID(userID)
	if err != nil || current == nil {
		return dto.ProfileResponse{}, &ErrProfileNotFound{}
	}

	// Compute resulting contacts
	resultTelegram := current.Telegram
	resultMax := current.Max
	if _, ok := raw["telegram"]; ok {
		resultTelegram = req.Telegram
	}
	if _, ok := raw["max"]; ok {
		resultMax = req.Max
	}
	if resultTelegram == nil && resultMax == nil {
		return dto.ProfileResponse{}, &ErrContactRequired{}
	}

	// Determine actual new values to write
	var newFullName *string
	if _, ok := raw["full_name"]; ok {
		newFullName = req.FullName
	}

	user, err := s.userRepo.UpdateProfile(userID, newFullName, resultTelegram, resultMax)
	if err != nil {
		return dto.ProfileResponse{}, fmt.Errorf("update profile: %w", err)
	}
	return userToProfileResponse(user), nil
}

func (s *ProfileService) GiveConsent(userID uuid.UUID) (dto.ConsentResponse, error) {
	user, err := s.userRepo.GiveConsent(userID)
	if err != nil {
		return dto.ConsentResponse{}, fmt.Errorf("give consent: %w", err)
	}
	return dto.ConsentResponse{
		ConsentGiven: user.ConsentGiven,
		ConsentDate:  user.ConsentGivenAt,
	}, nil
}
