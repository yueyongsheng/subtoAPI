//go:build integration

package repository

import (
	"context"
	"database/sql"
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
CREATE TEMP TABLE usage_logs (user_id bigint, created_at timestamptz, actual_cost numeric) ON COMMIT DROP;
INSERT INTO users VALUES (1,100.25,NULL),(2,25.50,NULL),(3,-999,NULL),(4,0,NULL),(5,50000,now());`)
	require.NoError(t, err)
	end := time.Date(2026, 9, 15, 2, 5, 0, 0, time.UTC)
	start := end.Add(-10 * time.Minute)
	todayStart := time.Date(2026, 9, 14, 16, 0, 0, 0, time.UTC)
	_, err = tx.ExecContext(ctx, `INSERT INTO usage_logs (user_id, created_at, actual_cost) VALUES
	 (1,$1::timestamptz + interval '1 second',1.25),(1,$1::timestamptz + interval '2 seconds',2.50),
	 (2,$2,4.00),(2,$1::timestamptz - interval '1 microsecond',8.00),(3,$1,3.00),(4,$1,4.00),(5,$1,100.00),
	 (1,$3,5.25),(1,$3::timestamptz - interval '1 microsecond',999.00)`, start, end, todayStart)
	require.NoError(t, err)
	repo := newUserRepositoryWithSQL(nil, tx)
	stats, ids, err := repo.GetAdminUserOverview(ctx, start, end, todayStart)
	require.NoError(t, err)
	require.Equal(t, int64(4), stats.TotalUsers)
	require.Equal(t, int64(2), stats.PositiveBalanceUsers)
	require.InDelta(t, 125.75, stats.TotalBalance, 0.000001)
	require.Equal(t, int64(3), stats.ActiveUsers10m)
	require.InDelta(t, 24.00, stats.TodayUserCost, 0.000001)
	require.ElementsMatch(t, []int64{1, 2, 3, 4}, ids)
	_, err = tx.ExecContext(ctx, "TRUNCATE users, usage_logs")
	require.NoError(t, err)
	stats, ids, err = repo.GetAdminUserOverview(ctx, start, end, todayStart)
	require.NoError(t, err)
	require.Zero(t, stats.TotalBalance)
	require.Zero(t, stats.TotalUsers)
	require.Zero(t, stats.ActiveUsers10m)
	require.Zero(t, stats.TodayUserCost)
	require.Empty(t, ids)
}

func TestAdminOverviewUser_IdentityAndDeletedUser(t *testing.T) {
	ctx := context.Background()
	tx, err := integrationDB.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()
	_, err = tx.ExecContext(ctx, `CREATE TEMP TABLE users (id bigint PRIMARY KEY, username text, email text, deleted_at timestamptz) ON COMMIT DROP;
INSERT INTO users VALUES (1,'Example','example@example.test',NULL),(2,NULL,NULL,NULL),(3,'Deleted','deleted@example.test',now());`)
	require.NoError(t, err)
	repo := newUserRepositoryWithSQL(nil, tx)
	u, err := repo.GetAdminOverviewUser(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, "Example", u.Username)
	require.Equal(t, "example@example.test", u.Email)
	u, err = repo.GetAdminOverviewUser(ctx, 2)
	require.NoError(t, err)
	require.Empty(t, u.Username)
	require.Empty(t, u.Email)
	for _, id := range []int64{3, 999} {
		_, err = repo.GetAdminOverviewUser(ctx, id)
		require.ErrorIs(t, err, sql.ErrNoRows)
	}
}
