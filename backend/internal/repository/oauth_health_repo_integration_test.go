//go:build integration

package repository

import (
	"context"
	_ "embed"
	"encoding/json"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/oauth_health.sql
var oauthHealthFixtureSQL string

func TestOAuthHealthSQLScopeDedupAndAtomicAdjustment(t *testing.T) {
	ctx := context.Background()
	tx, err := integrationDB.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()
	_, err = tx.ExecContext(ctx, oauthHealthFixtureSQL)
	require.NoError(t, err)
	r := newAccountRepositoryWithSQL(nil, tx, nil)
	group := int64(15)
	accounts, err := r.ListOAuthHealthAccounts(ctx, &group)
	require.NoError(t, err)
	require.Len(t, accounts, 1)
	require.EqualValues(t, 1, accounts[0].ID)
	all, err := r.ListOAuthHealthAccounts(ctx, nil)
	require.NoError(t, err)
	require.Len(t, all, 3)
	ungrouped := int64(-1)
	ungroupedAccounts, err := r.ListOAuthHealthAccounts(ctx, &ungrouped)
	require.NoError(t, err)
	require.Len(t, ungroupedAccounts, 1)
	require.EqualValues(t, 4, ungroupedAccounts[0].ID)
	end := time.Date(2026, 9, 16, 4, 0, 0, 0, time.UTC)
	scopes := []service.OAuthHealthObservationScope{{ID: 1, Since: end.Add(-30 * time.Minute)}, {ID: 2, Since: end.Add(-30 * time.Minute)}}
	stats, err := r.ObserveOAuthHealth(ctx, scopes, end)
	require.NoError(t, err)
	require.Equal(t, 4, stats[1].ObservedRequests)    // r1, r2, quota, r3; no local 503
	require.Equal(t, 1, stats[1].RateLimitedRequests) // retries and duplicated terminal log deduplicated
	require.Equal(t, 1, stats[1].QuotaRequests)
	require.Equal(t, 1, stats[1].OverloadedRequests)
	require.Equal(t, 2, stats[1].OutputRequests)
	require.Equal(t, 2, stats[1].PressureRequests)
	require.Equal(t, 1, stats[2].OverloadedRequests) // failing-over account attribution
	require.InDelta(t, 2150, *stats[1].P95FirstTokenMS, 0.01)
	// Array attribution wins even when every failed account is outside the scope.
	finalAccount, err := r.ObserveOAuthHealth(ctx, []service.OAuthHealthObservationScope{{ID: 4, Since: end.Add(-30 * time.Minute)}}, end)
	require.NoError(t, err)
	require.Zero(t, finalAccount[4].ObservedRequests)
	newEvidence, err := r.ObserveOAuthHealth(ctx, []service.OAuthHealthObservationScope{{ID: 1, Since: end.Add(-90 * time.Second)}}, end)
	require.NoError(t, err)
	require.Zero(t, newEvidence[1].PressureRequests)
	a := accounts[0]
	h := &service.OAuthHealth{AccountID: a.ID, CheckedAt: end, Concurrency: 5, RecommendedConcurrency: 5}
	saved, err := r.SaveOAuthHealth(ctx, a, h, 5, &group)
	require.NoError(t, err)
	require.True(t, saved)
	var concurrency, notifications int
	var extra string
	require.NoError(t, tx.QueryRowContext(ctx, "SELECT concurrency,extra->>'unrelated' FROM accounts WHERE id=1").Scan(&concurrency, &extra))
	require.Equal(t, 5, concurrency)
	require.Equal(t, "preserved", extra)
	require.NoError(t, tx.QueryRowContext(ctx, "SELECT count(*) FROM scheduler_outbox").Scan(&notifications))
	require.Equal(t, 1, notifications)
	saved, err = r.SaveOAuthHealth(ctx, a, h, 2, &group)
	require.NoError(t, err)
	require.False(t, saved) // stale CAS
	a.Concurrency = 5
	a.RawHealth, _ = json.Marshal(h)
	h.CheckedAt = end.Add(time.Minute)
	saved, err = r.SaveOAuthHealth(ctx, a, h, 5, &group)
	require.NoError(t, err)
	require.True(t, saved)
	require.NoError(t, tx.QueryRowContext(ctx, "SELECT count(*) FROM scheduler_outbox").Scan(&notifications))
	require.Equal(t, 1, notifications) // report-only writes don't churn scheduler
	a.RawHealth, _ = json.Marshal(h)
	_, err = tx.ExecContext(ctx, "DELETE FROM account_groups WHERE account_id=1 AND group_id=15")
	require.NoError(t, err)
	saved, err = r.SaveOAuthHealth(ctx, a, h, 2, &group)
	require.NoError(t, err)
	require.False(t, saved) // scope changed
}
