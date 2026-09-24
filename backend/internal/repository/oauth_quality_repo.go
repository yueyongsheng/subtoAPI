package repository

import (
	"context"
	"database/sql"
	"errors"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

func NewOAuthQualityRepository(client *dbent.Client, db *sql.DB, cache service.SchedulerCache) service.OAuthQualityRepository {
	return newAccountRepositoryWithSQL(client, db, cache)
}

// Read only the account metadata needed for selection, across all pages. No
// credentials or health/concurrency mutations are involved in this query.
const oauthQualityAccountSQL = `
SELECT a.id,a.name,a.platform,a.type,a.status,a.schedulable,
 ARRAY(SELECT ag.group_id FROM account_groups ag WHERE ag.account_id=a.id ORDER BY ag.group_id)
FROM accounts a WHERE a.deleted_at IS NULL AND a.type=ANY($2::text[])
 AND ($1::bigint IS NULL OR ($1=-1 AND NOT EXISTS(SELECT 1 FROM account_groups ag WHERE ag.account_id=a.id))
 OR EXISTS(SELECT 1 FROM account_groups ag WHERE ag.account_id=a.id AND ag.group_id=$1))
ORDER BY a.id LIMIT 2001`

func (r *accountRepository) ListOAuthQualityAccounts(ctx context.Context, groupID *int64, types []string) ([]service.OAuthQualityAccount, error) {
	rows, err := r.sql.QueryContext(ctx, oauthQualityAccountSQL, groupID, pq.Array(types))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	result := []service.OAuthQualityAccount{}
	for rows.Next() {
		var a service.OAuthQualityAccount
		if err := rows.Scan(&a.ID, &a.Name, &a.Platform, &a.Type, &a.Status, &a.Schedulable, pq.Array(&a.GroupIDs)); err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	if len(result) > 2000 {
		return nil, errors.New("select a smaller account group (maximum 2000)")
	}
	return result, rows.Err()
}
