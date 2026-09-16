package repository

import (
	"context"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// Reuse the existing group availability definition at one snapshot time. An
// account shared by groups counts once in each; groups without OAuth are omitted.
var oauthGroupAvailabilitySQL = `SELECT g.id,g.name,
 COUNT(DISTINCT a.id) FILTER (WHERE ` + strings.ReplaceAll(groupAccountAvailableSQL, "NOW()", "$1::timestamptz") + `
 AND (a.parent_account_id IS NULL OR EXISTS (
  SELECT 1 FROM accounts p WHERE p.id=a.parent_account_id AND p.deleted_at IS NULL AND p.status='active'
  AND (p.expires_at IS NULL OR p.expires_at>$1 OR p.auto_pause_on_expired=FALSE)
  AND (p.temp_unschedulable_until IS NULL OR p.temp_unschedulable_until<=$1)
 ))) AS available_accounts
 FROM groups g JOIN account_groups ag ON ag.group_id=g.id
 JOIN accounts a ON a.id=ag.account_id AND a.type='oauth' AND a.deleted_at IS NULL
 WHERE g.deleted_at IS NULL AND g.status='active'
 GROUP BY g.id,g.name,g.sort_order ORDER BY g.sort_order,g.id`

func (r *accountRepository) GetOAuthGroupAvailability(ctx context.Context, now time.Time) ([]service.OAuthGroupAvailability, error) {
	rows, err := r.sql.QueryContext(ctx, oauthGroupAvailabilitySQL, now)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	result := []service.OAuthGroupAvailability{}
	for rows.Next() {
		var group service.OAuthGroupAvailability
		if err := rows.Scan(&group.GroupID, &group.GroupName, &group.AvailableAccounts); err != nil {
			return nil, err
		}
		result = append(result, group)
	}
	return result, rows.Err()
}
