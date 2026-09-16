//go:build integration

package repository

import (
	"context"
	_ "embed"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/oauth_group_availability.sql
var oauthGroupAvailabilityFixtureSQL string

func TestOAuthGroupAvailabilitySchedulingSnapshot(t *testing.T) {
	ctx := context.Background()
	tx, err := integrationDB.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()
	_, err = tx.ExecContext(ctx, oauthGroupAvailabilityFixtureSQL)
	require.NoError(t, err)
	r := newAccountRepositoryWithSQL(nil, tx, nil)
	now := time.Date(2026, 9, 17, 0, 0, 0, 0, time.UTC)
	groups, err := r.GetOAuthGroupAvailability(ctx, now)
	require.NoError(t, err)
	require.Equal(t, []service.OAuthGroupAvailability{
		{GroupID: 20, GroupName: "Shared", AvailableAccounts: 1},
		{GroupID: 10, GroupName: "Enterprise Pro", AvailableAccounts: 6},
		{GroupID: 30, GroupName: "Unavailable", AvailableAccounts: 0},
	}, groups)
	// Scheduling changes refresh both shared groups; past 503 metadata is not a pause.
	_, err = tx.ExecContext(ctx, "UPDATE accounts SET temp_unschedulable_until=$1 WHERE id=1", now.Add(time.Minute))
	require.NoError(t, err)
	groups, err = r.GetOAuthGroupAvailability(ctx, now)
	require.NoError(t, err)
	require.Zero(t, groups[0].AvailableAccounts)
	require.EqualValues(t, 5, groups[1].AvailableAccounts)
	groups, err = r.GetOAuthGroupAvailability(ctx, now.Add(time.Minute))
	require.NoError(t, err)
	require.EqualValues(t, 1, groups[0].AvailableAccounts)
	require.EqualValues(t, 6, groups[1].AvailableAccounts)
}
