package service

import (
	"context"
	"fmt"
	"math"
	"time"
)

// AdminUserOverview is a manually refreshed, global snapshot, independent of list filters.
type AdminUserOverview struct {
	TotalUsers           int64     `json:"total_users"`
	PositiveBalanceUsers int64     `json:"positive_balance_users"`
	TotalBalance         float64   `json:"total_balance"`
	BalanceCNY           float64   `json:"balance_cny"`
	CurrentConcurrency   *int64    `json:"current_concurrency"`
	MaxUserConcurrency   *int64    `json:"max_user_concurrency"`
	ActiveUsers10m       int64     `json:"active_users_10m"`
	TodayUserCost        float64   `json:"today_user_cost"`
	TodayUserCostCNY     float64   `json:"today_user_cost_cny"`
	QueriedAt            time.Time `json:"queried_at"`
	WindowStart          time.Time `json:"window_start"`
}

// AdminUserOverviewRepository keeps this reporting query separate from user mutation APIs.
type AdminUserOverviewRepository interface {
	GetAdminUserOverview(ctx context.Context, start, end, todayStart time.Time) (*AdminUserOverview, []int64, error)
}

func (s *UserService) GetAdminUserOverview(ctx context.Context, queriedAt time.Time) (*AdminUserOverview, []int64, error) {
	repo, ok := s.userRepo.(AdminUserOverviewRepository)
	if !ok {
		return nil, nil, fmt.Errorf("user overview repository is not configured")
	}
	end := queriedAt.UTC()
	start := end.Add(-10 * time.Minute)
	localEnd := end.In(time.FixedZone("UTC+8", 8*60*60))
	todayStart := time.Date(localEnd.Year(), localEnd.Month(), localEnd.Day(), 0, 0, 0, 0, localEnd.Location()).UTC()
	stats, ids, err := repo.GetAdminUserOverview(ctx, start, end, todayStart)
	if err != nil {
		return nil, nil, fmt.Errorf("get user overview: %w", err)
	}
	stats.QueriedAt = end
	stats.WindowStart = start
	stats.BalanceCNY = math.Round(stats.TotalBalance/25*100) / 100
	stats.TodayUserCostCNY = math.Round(stats.TodayUserCost/25*100) / 100
	return stats, ids, nil
}
