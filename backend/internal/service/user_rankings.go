package service

import (
	"context"
	"fmt"
	"math"
	"time"
)

const AdminUserRankingLimit = 10

type AdminUserSpending struct {
	UserID  int64              `json:"user_id"`
	User    *AdminOverviewUser `json:"user"`
	Cost    float64            `json:"cost"`
	CostCNY float64            `json:"cost_cny"`
}

type AdminSpendingRanking struct {
	QueriedAt   time.Time           `json:"queried_at"`
	WindowStart time.Time           `json:"window_start"`
	Users       []AdminUserSpending `json:"users"`
}

type AdminUserConcurrency struct {
	UserID             int64              `json:"user_id"`
	User               *AdminOverviewUser `json:"user"`
	CurrentConcurrency int64              `json:"current_concurrency"`
}

type AdminConcurrencyRanking struct {
	QueriedAt time.Time              `json:"queried_at"`
	Users     []AdminUserConcurrency `json:"users"`
}

// Ranking reads run only on administrator request, outside the gateway path.
type AdminUserRankingRepository interface {
	GetAdminSpendingRanking(ctx context.Context, start, end time.Time) ([]AdminUserSpending, error)
	GetAdminRankingUserIDs(ctx context.Context) ([]int64, error)
	GetAdminRankingIdentities(ctx context.Context, ids []int64) (map[int64]*AdminOverviewUser, error)
}

func (s *UserService) rankingRepository() (AdminUserRankingRepository, error) {
	repo, ok := s.userRepo.(AdminUserRankingRepository)
	if !ok {
		return nil, fmt.Errorf("user ranking repository is not configured")
	}
	return repo, nil
}

func (s *UserService) GetAdminSpendingRanking(ctx context.Context, queriedAt time.Time) (*AdminSpendingRanking, error) {
	repo, err := s.rankingRepository()
	if err != nil {
		return nil, err
	}
	end := queriedAt.UTC()
	local := end.In(time.FixedZone("UTC+8", 8*60*60))
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, local.Location()).UTC()
	users, err := repo.GetAdminSpendingRanking(ctx, start, end)
	if err != nil {
		return nil, err
	}
	if users == nil {
		users = []AdminUserSpending{}
	}
	for i := range users {
		users[i].CostCNY = math.Round(users[i].Cost/25*100) / 100
	}
	return &AdminSpendingRanking{QueriedAt: end, WindowStart: start, Users: users}, nil
}

func (s *UserService) GetAdminRankingUserIDs(ctx context.Context) ([]int64, error) {
	repo, err := s.rankingRepository()
	if err != nil {
		return nil, err
	}
	return repo.GetAdminRankingUserIDs(ctx)
}

func (s *UserService) GetAdminRankingIdentities(ctx context.Context, ids []int64) (map[int64]*AdminOverviewUser, error) {
	repo, err := s.rankingRepository()
	if err != nil {
		return nil, err
	}
	return repo.GetAdminRankingIdentities(ctx, ids)
}
