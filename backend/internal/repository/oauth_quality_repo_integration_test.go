//go:build integration

package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOAuthQualitySQLScopeAndTypes(t *testing.T) {
	ctx := context.Background()
	tx, err := integrationDB.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()
	_, err = tx.ExecContext(ctx, oauthHealthFixtureSQL)
	require.NoError(t, err)
	r := newAccountRepositoryWithSQL(nil, tx, nil)
	group, ungrouped := int64(15), int64(-1)
	for _, tc := range []struct {
		name  string
		group *int64
		types []string
		want  []int64
	}{
		{"mixed group", &group, []string{"oauth", "apikey"}, []int64{1, 3}},
		{"oauth group", &group, []string{"oauth"}, []int64{1}},
		{"all", nil, []string{"oauth", "apikey"}, []int64{1, 2, 3, 4}},
		{"ungrouped", &ungrouped, []string{"oauth"}, []int64{4}},
		{"empty types", nil, []string{}, []int64{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			accounts, err := r.ListOAuthQualityAccounts(ctx, tc.group, tc.types)
			require.NoError(t, err)
			ids := []int64{}
			for _, a := range accounts {
				ids = append(ids, a.ID)
			}
			require.Equal(t, tc.want, ids)
		})
	}
	_, err = tx.ExecContext(ctx, "UPDATE accounts SET deleted_at=now() WHERE id=1")
	require.NoError(t, err)
	accounts, err := r.ListOAuthQualityAccounts(ctx, &group, []string{"oauth", "apikey"})
	require.NoError(t, err)
	require.Len(t, accounts, 1)
	require.EqualValues(t, 3, accounts[0].ID)
}
