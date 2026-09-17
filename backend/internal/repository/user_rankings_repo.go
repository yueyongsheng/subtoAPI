package repository

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

var _ service.AdminUserRankingRepository = (*userRepository)(nil)

// Sum actual user charges before rounding. Match the overview's live-user cohort
// and UTC+8 half-open day window; ties use ID for deterministic top-ten membership.
const adminSpendingRankingSQL = `
WITH ranked AS (
    SELECT l.user_id, SUM(l.actual_cost) AS cost
      FROM usage_logs l JOIN users u ON u.id = l.user_id AND u.deleted_at IS NULL
     WHERE l.created_at >= $1 AND l.created_at < $2
     GROUP BY l.user_id
    HAVING SUM(l.actual_cost) > 0
     ORDER BY cost DESC, l.user_id ASC
     LIMIT 10
)
SELECT r.user_id, COALESCE(u.username, '') AS username, COALESCE(u.email, '') AS email, r.cost
  FROM ranked r JOIN users u ON u.id = r.user_id
 ORDER BY r.cost DESC, r.user_id ASC`

func (r *userRepository) GetAdminSpendingRanking(ctx context.Context, start, end time.Time) ([]service.AdminUserSpending, error) {
	rows, err := r.sql.QueryContext(ctx, adminSpendingRankingSQL, start, end)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	result := make([]service.AdminUserSpending, 0, service.AdminUserRankingLimit)
	for rows.Next() {
		var item service.AdminUserSpending
		user := &service.AdminOverviewUser{}
		if err := rows.Scan(&item.UserID, &user.Username, &user.Email, &item.Cost); err != nil {
			return nil, err
		}
		user.ID, item.User = item.UserID, user
		result = append(result, item)
	}
	return result, rows.Err()
}

func (r *userRepository) GetAdminRankingUserIDs(ctx context.Context) ([]int64, error) {
	rows, err := r.sql.QueryContext(ctx, `SELECT id FROM users WHERE deleted_at IS NULL ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// At most ten identities are fetched in one query, never one query per user.
func (r *userRepository) GetAdminRankingIdentities(ctx context.Context, ids []int64) (map[int64]*service.AdminOverviewUser, error) {
	result := make(map[int64]*service.AdminOverviewUser, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	rows, err := r.sql.QueryContext(ctx, `SELECT id, COALESCE(username, ''), COALESCE(email, '') FROM users WHERE id = ANY($1) AND deleted_at IS NULL`, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		user := &service.AdminOverviewUser{}
		if err := rows.Scan(&user.ID, &user.Username, &user.Email); err != nil {
			return nil, err
		}
		result[user.ID] = user
	}
	return result, rows.Err()
}
