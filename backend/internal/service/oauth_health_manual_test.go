//go:build unit

package service

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestOAuthHealthManualEditKeepsReportAndLocksRecommendations(t *testing.T) {
	a := &Account{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Concurrency: 10, Extra: map[string]any{OAuthHealthExtraKey: map[string]any{"account_id": 1}}}
	repo := &accountRepoStubForBulkUpdate{getByIDAccounts: map[int64]*Account{1: a}}
	svc := &adminServiceImpl{accountRepo: repo}
	limit := 50
	got, err := svc.UpdateAccount(context.Background(), 1, &UpdateAccountInput{Concurrency: &limit, Extra: map[string]any{OAuthHealthExtraKey: "injected"}})
	require.NoError(t, err)
	require.Equal(t, 50, got.Concurrency)
	require.NotEmpty(t, got.Extra[OAuthHealthManualKey])
	require.Equal(t, map[string]any{"account_id": 1}, got.Extra[OAuthHealthExtraKey])
	marker := got.Extra[OAuthHealthManualKey]
	got, err = svc.UpdateAccount(context.Background(), 1, &UpdateAccountInput{Name: "renamed", Concurrency: &limit, Extra: map[string]any{}})
	require.NoError(t, err)
	require.Equal(t, marker, got.Extra[OAuthHealthManualKey])
}

func TestOAuthHealthBulkConcurrencyMarksManual(t *testing.T) {
	repo := &accountRepoStubForBulkUpdate{}
	svc := &adminServiceImpl{accountRepo: repo}
	limit := 10
	_, err := svc.BulkUpdateAccounts(context.Background(), &BulkUpdateAccountsInput{AccountIDs: []int64{1}, Concurrency: &limit, Extra: map[string]any{OAuthHealthExtraKey: "injected", OAuthHealthManualKey: "injected"}})
	require.NoError(t, err)
	require.NotEmpty(t, repo.lastBulkUpdate.Extra[OAuthHealthManualKey])
	require.NotEqual(t, "injected", repo.lastBulkUpdate.Extra[OAuthHealthManualKey])
	require.NotContains(t, repo.lastBulkUpdate.Extra, OAuthHealthExtraKey)
}
