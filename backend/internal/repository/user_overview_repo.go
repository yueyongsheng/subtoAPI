package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

var _ service.AdminUserOverviewRepository = (*userRepository)(nil)

// Keep the balance, user cohort and usage window in one PostgreSQL statement snapshot.
// Debt is excluded before summation, never subtracted from other users' unused funds.
const adminUserOverviewSQL = `
WITH live_users AS (
    SELECT id, balance FROM users WHERE deleted_at IS NULL
)
SELECT COUNT(*),
       COUNT(*) FILTER (WHERE balance > 0),
       COALESCE(SUM(balance) FILTER (WHERE balance > 0), 0),
       COALESCE(array_agg(id), ARRAY[]::bigint[]),
       (SELECT COUNT(DISTINCT l.user_id)
          FROM usage_logs l JOIN live_users u ON u.id = l.user_id
         WHERE l.created_at >= $1 AND l.created_at < $2)
FROM live_users`

func (r *userRepository) GetAdminUserOverview(ctx context.Context, start, end time.Time) (*service.AdminUserOverview, []int64, error) {
	rows, err := r.sql.QueryContext(ctx, adminUserOverviewSQL, start, end)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, nil, err
		}
		return nil, nil, sql.ErrNoRows
	}
	var result service.AdminUserOverview
	var ids pq.Int64Array
	if err := rows.Scan(&result.TotalUsers, &result.PositiveBalanceUsers, &result.TotalBalance, &ids, &result.ActiveUsers10m); err != nil {
		return nil, nil, err
	}
	return &result, []int64(ids), rows.Err()
}
