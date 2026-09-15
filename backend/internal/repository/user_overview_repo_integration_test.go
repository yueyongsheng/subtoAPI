//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAdminUserOverview_PositiveBalancesAndDistinctWindow(t *testing.T) {
	ctx := context.Background()
	tx, err := integrationDB.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()
	_, err = tx.ExecContext(ctx, `
CREATE TEMP TABLE users (id bigint, balance numeric(20,8), deleted_at timestamptz) ON COMMIT DROP;
CREATE TEMP TABLE usage_logs (user_id bigint, created_at timestamptz) ON COMMIT DROP;
INSERT INTO users VALUES (1,100.25,NULL),(2,25.50,NULL),(3,-999,NULL),(4,0,NULL),(5,50000,now());`)
	require.NoError(t, err)
	end := time.Date(2026, 9, 15, 2, 5, 0, 0, time.UTC)
	start := end.Add(-10 * time.Minute)
	_, err = tx.ExecContext(ctx, `INSERT INTO usage_logs VALUES
 (1,$1::timestamptz + interval '1 second'),(1,$1::timestamptz + interval '2 seconds'),
 (2,$2),(2,$1::timestamptz - interval '1 microsecond'),(3,$1),(4,$1),(5,$1)`, start, end)
	require.NoError(t, err)
	repo := newUserRepositoryWithSQL(nil, tx)
	stats, ids, err := repo.GetAdminUserOverview(ctx, start, end)
	require.NoError(t, err)
	require.Equal(t, int64(4), stats.TotalUsers)
	require.Equal(t, int64(2), stats.PositiveBalanceUsers)
	require.InDelta(t, 125.75, stats.TotalBalance, 0.000001)
	require.Equal(t, int64(3), stats.ActiveUsers10m)
	require.ElementsMatch(t, []int64{1, 2, 3, 4}, ids)
	_, err = tx.ExecContext(ctx, "TRUNCATE users, usage_logs")
	require.NoError(t, err)
	stats, ids, err = repo.GetAdminUserOverview(ctx, start, end)
	require.NoError(t, err)
	require.Zero(t, stats.TotalBalance)
	require.Zero(t, stats.TotalUsers)
	require.Zero(t, stats.ActiveUsers10m)
	require.Empty(t, ids)
}
