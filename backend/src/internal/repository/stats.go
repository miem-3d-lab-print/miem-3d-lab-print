package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/models"
)

type MaterialStat struct {
	MaterialName string
	Count        int
}

type StatsRepository interface {
	CountApplications(ctx context.Context, from, to time.Time) (int, error)
	GetCompletionStats(ctx context.Context, from, to time.Time) (count int, avgHours *float64, err error)
	GroupByMaterial(ctx context.Context, from, to time.Time) ([]MaterialStat, error)
	GroupByStatus(ctx context.Context) (map[string]int, error)
}

type GORMStatsRepository struct {
	db *gorm.DB
}

func NewGORMStatsRepository(database *gorm.DB) *GORMStatsRepository {
	return &GORMStatsRepository{db: database}
}

func (repository *GORMStatsRepository) CountApplications(ctx context.Context, from, to time.Time) (int, error) {
	var count int64
	result := repository.db.WithContext(ctx).Model(&models.Application{}).
		Where("created_at >= ? AND created_at <= ?", from, to).Count(&count)
	return int(count), result.Error
}

func (repository *GORMStatsRepository) GetCompletionStats(ctx context.Context, from, to time.Time) (int, *float64, error) {
	type completionStats struct {
		Count    int
		AvgHours *float64
	}
	var stats completionStats
	err := repository.db.WithContext(ctx).Raw(`
		SELECT count(*) AS count,
		       avg(extract(epoch from (issued.created_at - new_history.created_at)) / 3600) AS avg_hours
		FROM (
			SELECT application_id, created_at
			FROM status_history
			WHERE status = 'issued' AND created_at >= ? AND created_at <= ?
		) issued
		JOIN (
			SELECT application_id, min(created_at) AS created_at
			FROM status_history WHERE status = 'new'
			GROUP BY application_id
		) new_history ON new_history.application_id = issued.application_id
	`, from, to).Scan(&stats).Error
	return stats.Count, stats.AvgHours, err
}

func (repository *GORMStatsRepository) GroupByMaterial(ctx context.Context, from, to time.Time) ([]MaterialStat, error) {
	var stats []MaterialStat
	err := repository.db.WithContext(ctx).Model(&models.Application{}).
		Select("snapshot_material_name AS material_name, count(*) AS count").
		Where("created_at >= ? AND created_at <= ?", from, to).
		Group("snapshot_material_name").
		Order("count(*) DESC").
		Scan(&stats).Error
	return stats, err
}

func (repository *GORMStatsRepository) GroupByStatus(ctx context.Context) (map[string]int, error) {
	type statusRow struct {
		Status string
		Count  int
	}
	var rows []statusRow
	err := repository.db.WithContext(ctx).Model(&models.Application{}).
		Select("status, count(*) AS count").Group("status").Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := map[string]int{
		"new": 0, "in_review": 0, "printing": 0,
		"ready": 0, "issued": 0, "rejected": 0, "cancelled": 0,
	}
	for _, row := range rows {
		result[row.Status] = row.Count
	}
	return result, nil
}
