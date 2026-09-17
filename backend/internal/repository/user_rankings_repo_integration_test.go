//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAdminUserRankings_SpendingScopePrecisionAndLimit(t *testing.T) {
	ctx := context.Background()
	tx, err := integrationDB.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()
	_, err = tx.ExecContext(ctx, `CREATE TEMP TABLE users (id bigint PRIMARY KEY, username text, email text, status text, deleted_at timestamptz) ON COMMIT DROP;
CREATE TEMP TABLE usage_logs (user_id bigint, created_at timestamptz, actual_cost numeric, total_cost numeric) ON COMMIT DROP;
INSERT INTO users SELECT n,'User '||n,'user'||n||'@example.test','active',NULL FROM generate_series(1,15) n;
UPDATE users SET deleted_at=now() WHERE id=15;
UPDATE users SET status='disabled' WHERE id=3;`)
	require.NoError(t, err)
	start := time.Date(2026, 9, 16, 16, 0, 0, 0, time.UTC)
	end := start.Add(12 * time.Hour)
	_, err = tx.ExecContext(ctx, `INSERT INTO usage_logs SELECT n,$1,50,9999 FROM generate_series(1,15) n`, start)
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, `INSERT INTO usage_logs VALUES (1,$1,25.001,0),(1,$1,25.001,0),(2,$1,50.001,0),
(14,$1,-50,0),(2,$2,99999,0),(3,$1::timestamptz-interval '1 microsecond',99999,0);`, start, end)
	require.NoError(t, err)
	repo := newUserRepositoryWithSQL(nil, tx)
	ranking, err := repo.GetAdminSpendingRanking(ctx, start, end)
	require.NoError(t, err)
	require.Len(t, ranking, 10)
	for i, row := range ranking {
		require.Equal(t, int64(i+1), row.UserID)
		require.Equal(t, row.UserID, row.User.ID)
	}
	require.InDelta(t, 100.002, ranking[0].Cost, 0.0000001)
	require.InDelta(t, 100.001, ranking[1].Cost, 0.0000001)
	require.InDelta(t, 50, ranking[2].Cost, 0.0000001)
	ids, err := repo.GetAdminRankingUserIDs(ctx)
	require.NoError(t, err)
	require.Len(t, ids, 14)
	identities, err := repo.GetAdminRankingIdentities(ctx, []int64{1, 3, 15, 999})
	require.NoError(t, err)
	require.Len(t, identities, 2)
	require.Equal(t, "User 3", identities[3].Username)
	_, err = tx.ExecContext(ctx, `TRUNCATE usage_logs`)
	require.NoError(t, err)
	ranking, err = repo.GetAdminSpendingRanking(ctx, start, end)
	require.NoError(t, err)
	require.Empty(t, ranking)
}
