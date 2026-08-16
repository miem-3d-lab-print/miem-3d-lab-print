package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/dto"
	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/models"
	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/repository"
)

type AdminService struct {
	logger   *slog.Logger
	txMgr    repository.TxManager
	userRepo repository.UserRepository
}

func NewAdminService(logger *slog.Logger, txMgr repository.TxManager, userRepo repository.UserRepository) *AdminService {
	return &AdminService{logger: logger, txMgr: txMgr, userRepo: userRepo}
}

func (s *AdminService) SearchUsers(query string) (*dto.AdminUsersResponse, error) {
	if len(query) < 3 {
		return nil, &ErrQueryTooShort{}
	}
	users, err := s.userRepo.SearchByEmail(query, 20)
	if err != nil {
		return nil, fmt.Errorf("search users: %w", err)
	}
	items := make([]*dto.AdminUserItem, 0, len(users))
	for _, u := range users {
		items = append(items, &dto.AdminUserItem{
			ID:        u.ID.String(),
			Email:     u.Email,
			FullName:  u.FullName,
			Role:      u.Role,
			CreatedAt: u.CreatedAt,
		})
	}
	return &dto.AdminUsersResponse{Items: items}, nil
}

func (s *AdminService) SetRole(targetID uuid.UUID, role string) (*dto.SetRoleResponse, error) {
	if role != "admin" && role != "user" {
		return nil, &ErrInvalidRole{}
	}

	target, err := s.userRepo.FindByID(targetID)
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}
	if target == nil {
		return nil, &ErrUserNotFound{}
	}
	if target.Role == role {
		return &dto.SetRoleResponse{ID: target.ID.String(), Email: target.Email, Role: target.Role}, nil
	}

	var updated *models.User

	if target.Role == "admin" && role == "user" {
		// CountAdmins locks admin rows inside the transaction so concurrent
		// demotions cannot both pass the last-admin check.
		if err := s.txMgr.RunInTx(context.Background(), func(tx repository.DBTX) error {
			count, err := s.userRepo.CountAdmins(tx)
			if err != nil {
				return fmt.Errorf("count admins: %w", err)
			}
			if count <= 1 {
				return &ErrLastAdmin{}
			}
			updated, err = s.userRepo.SetRole(tx, targetID, role)
			return err
		}); err != nil {
			return nil, err
		}
	} else {
		var txErr error
		if err := s.txMgr.RunInTx(context.Background(), func(tx repository.DBTX) error {
			updated, txErr = s.userRepo.SetRole(tx, targetID, role)
			return txErr
		}); err != nil {
			return nil, fmt.Errorf("set role: %w", err)
		}
	}

	s.logger.Info("role changed", "user_id", targetID, "new_role", role)
	return &dto.SetRoleResponse{ID: updated.ID.String(), Email: updated.Email, Role: updated.Role}, nil
}

// StatsService assembles statistics from the StatsRepository.
// No SQL lives here — only business logic (default period, response shaping).
type StatsService struct {
	statsRepo repository.StatsRepository
}

func NewStatsService(statsRepo repository.StatsRepository) *StatsService {
	return &StatsService{statsRepo: statsRepo}
}

func (s *StatsService) GetStats(dateFrom, dateTo *time.Time) (*dto.StatsResponse, error) {
	now := time.Now()
	if dateFrom == nil {
		t := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		dateFrom = &t
	}
	if dateTo == nil {
		t := time.Date(now.Year(), now.Month()+1, 0, 23, 59, 59, 0, time.UTC)
		dateTo = &t
	}
	if dateFrom.After(*dateTo) {
		return nil, &ErrInvalidPeriod{}
	}

	ctx := context.Background()

	total, err := s.statsRepo.CountApplications(ctx, *dateFrom, *dateTo)
	if err != nil {
		return nil, fmt.Errorf("count total: %w", err)
	}

	completedCount, avgHours, err := s.statsRepo.GetCompletionStats(ctx, *dateFrom, *dateTo)
	if err != nil {
		return nil, fmt.Errorf("avg completion: %w", err)
	}

	materialStats, err := s.statsRepo.GroupByMaterial(ctx, *dateFrom, *dateTo)
	if err != nil {
		return nil, fmt.Errorf("by material: %w", err)
	}

	byStatus, err := s.statsRepo.GroupByStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("by status: %w", err)
	}

	byMaterial := make([]*dto.ByMaterialStat, 0, len(materialStats))
	for _, m := range materialStats {
		byMaterial = append(byMaterial, &dto.ByMaterialStat{
			MaterialName: m.MaterialName,
			Count:        m.Count,
		})
	}

	resp := &dto.StatsResponse{
		TotalApplications:  total,
		AvgCompletionHours: avgHours,
		CompletedCount:     completedCount,
		ByMaterial:         byMaterial,
		ByStatusCurrent:    byStatus,
	}
	resp.Period.DateFrom = dateFrom.Format("2006-01-02")
	resp.Period.DateTo = dateTo.Format("2006-01-02")

	return resp, nil
}
