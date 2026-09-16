package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOAuthHealthClassificationAndGuards(t *testing.T) {
	now := time.Date(2026, 9, 16, 4, 0, 0, 0, time.UTC)
	recent := now.Add(-time.Minute)
	future := now.Add(time.Hour)
	old := now.Add(-6 * time.Minute)
	base := OAuthHealthAccount{ID: 1, Concurrency: 10, Status: StatusActive, Schedulable: true}
	pressure := OAuthHealthStats{ObservedRequests: 20, PressureRequests: 5, PressureMinutes: 3, RateLimitedRequests: 5, LatestPressureAt: &recent}
	cases := []struct {
		name, status, reason string
		target               int
		edit                 func(*OAuthHealthAccount, *OAuthHealthStats)
	}{
		{"repeated 429", "rate_limited", "reduce_concurrency", 5, func(*OAuthHealthAccount, *OAuthHealthStats) {}},
		{"repeated 503", "overloaded", "reduce_concurrency", 5, func(_ *OAuthHealthAccount, s *OAuthHealthStats) { s.RateLimitedRequests = 0; s.OverloadedRequests = 5 }},
		{"single request retries", "rate_limited", "observe", 10, func(_ *OAuthHealthAccount, s *OAuthHealthStats) { s.PressureRequests = 1 }},
		{"burst only", "rate_limited", "observe", 10, func(_ *OAuthHealthAccount, s *OAuthHealthStats) { s.PressureMinutes = 1 }},
		{"rare errors", "rate_limited", "observe", 10, func(_ *OAuthHealthAccount, s *OAuthHealthStats) { s.ObservedRequests = 1000 }},
		{"old pressure", "rate_limited", "observe", 10, func(_ *OAuthHealthAccount, s *OAuthHealthStats) { s.LatestPressureAt = &old }},
		{"quota exhaustion", "quota_limited", "wait_reset", 10, func(_ *OAuthHealthAccount, s *OAuthHealthStats) { s.QuotaRequests = 1 }},
		{"reset time", "rate_limited", "wait_reset", 10, func(a *OAuthHealthAccount, _ *OAuthHealthStats) { a.RateLimitResetAt = &future }},
		{"credentials", "auth_error", "refresh_credentials", 10, func(_ *OAuthHealthAccount, s *OAuthHealthStats) { s.AuthRequests = 1 }},
		{"disabled", "rate_limited", "paused", 10, func(a *OAuthHealthAccount, _ *OAuthHealthStats) { a.Schedulable = false }},
		{"inactive", "rate_limited", "paused", 10, func(a *OAuthHealthAccount, _ *OAuthHealthStats) { a.Status = "inactive" }},
		{"temporary cooldown", "rate_limited", "cooldown", 10, func(a *OAuthHealthAccount, _ *OAuthHealthStats) { a.TempUnschedulableUntil = &future }},
		{"shadow", "rate_limited", "shared_credential", 10, func(a *OAuthHealthAccount, _ *OAuthHealthStats) { id := int64(99); a.ParentAccountID = &id }},
		{"parent", "rate_limited", "shared_credential", 10, func(a *OAuthHealthAccount, _ *OAuthHealthStats) { a.HasChildren = true }},
		{"floor", "rate_limited", "manual_concurrency", 1, func(a *OAuthHealthAccount, _ *OAuthHealthStats) { a.Concurrency = 1 }},
		{"empty", "insufficient", "insufficient", 10, func(_ *OAuthHealthAccount, s *OAuthHealthStats) { *s = OAuthHealthStats{} }},
		{"stable", "stable", "recovery_observation", 10, func(_ *OAuthHealthAccount, s *OAuthHealthStats) {
			*s = OAuthHealthStats{OutputRequests: 20, ObservedRequests: 20}
		}},
		{"incomplete output", "insufficient", "insufficient", 10, func(_ *OAuthHealthAccount, s *OAuthHealthStats) { *s = OAuthHealthStats{ObservedRequests: 30} }},
		{"other upstream errors", "upstream_error", "inspect_errors", 10, func(_ *OAuthHealthAccount, s *OAuthHealthStats) {
			*s = OAuthHealthStats{OutputRequests: 30, OtherErrorRequests: 1}
		}},
		{"change cooldown", "rate_limited", "change_cooldown", 10, func(a *OAuthHealthAccount, _ *OAuthHealthStats) {
			a.RawHealth, _ = json.Marshal(OAuthHealth{AccountID: a.ID, Concurrency: a.Concurrency, CheckedAt: now, LastChange: &OAuthHealthChange{At: recent}})
		}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			a, s := base, pressure
			tt.edit(&a, &s)
			h := evaluateOAuthHealth(a, s, now)
			require.Equal(t, tt.status, h.Status)
			require.Equal(t, tt.reason, h.Reason)
			require.Equal(t, tt.target, h.RecommendedConcurrency)
		})
	}
}

type healthMemoryRepo struct {
	accounts     []OAuthHealthAccount
	stats        map[int64]OAuthHealthStats
	scopes       []OAuthHealthObservationScope
	group        *int64
	failObserve  bool
	conflict     bool
	saves        int
	alternatives []int64
}

func (r *healthMemoryRepo) GetOAuthGroupAvailability(context.Context, time.Time) ([]OAuthGroupAvailability, error) {
	return nil, nil
}

func (r *healthMemoryRepo) OAuthHealthCooldownAlternatives(_ context.Context, _ OAuthHealthAccount, _ []string, _ time.Time, excluded []int64) ([]int64, error) {
	result := []int64{}
	for _, id := range r.alternatives {
		found := false
		for _, x := range excluded {
			if id == x {
				found = true
			}
		}
		if !found {
			result = append(result, id)
		}
	}
	return result, nil
}

func (r *healthMemoryRepo) ListOAuthHealthAccounts(_ context.Context, g *int64) ([]OAuthHealthAccount, error) {
	r.group = g
	return append([]OAuthHealthAccount{}, r.accounts...), nil
}
func (r *healthMemoryRepo) ObserveOAuthHealth(_ context.Context, s []OAuthHealthObservationScope, _ time.Time) (map[int64]OAuthHealthStats, error) {
	r.scopes = s
	if r.failObserve {
		return nil, errors.New("logs offline")
	}
	return r.stats, nil
}
func (r *healthMemoryRepo) SaveOAuthHealth(_ context.Context, a OAuthHealthAccount, h *OAuthHealth, target int, _ *int64) (bool, error) {
	r.saves++
	if r.conflict {
		return false, nil
	}
	for i := range r.accounts {
		if r.accounts[i].ID == a.ID {
			r.accounts[i].RawHealth, _ = json.Marshal(h)
			r.accounts[i].Concurrency = target
			if h.LastChange != nil && h.LastChange.Action == "cooldown" {
				r.accounts[i].TempUnschedulableUntil = h.CooldownUntil
			}
		}
	}
	return true, nil
}
func healthFixture() (*OAuthHealthService, *healthMemoryRepo, time.Time) {
	now := time.Date(2026, 9, 16, 4, 0, 0, 0, time.UTC)
	recent := now.Add(-time.Minute)
	r := &healthMemoryRepo{accounts: []OAuthHealthAccount{{ID: 1, Name: "fixture", Concurrency: 10, Status: StatusActive, Schedulable: true}}, stats: map[int64]OAuthHealthStats{1: {ObservedRequests: 20, PressureRequests: 5, PressureMinutes: 3, RateLimitedRequests: 5, LatestPressureAt: &recent}}}
	return &OAuthHealthService{repo: r, now: func() time.Time { return now }, monitoringEnabled: func(context.Context) bool { return true }}, r, now
}
func TestOAuthHealthCheckApplyRepeatRestore(t *testing.T) {
	s, r, now := healthFixture()
	ctx := context.Background()
	group := int64(15)
	report, err := s.Check(ctx, &group)
	require.NoError(t, err)
	require.Equal(t, &group, r.group)
	require.Equal(t, 10, r.accounts[0].Concurrency)
	selection := []OAuthHealthSelection{{ID: 1, CheckedAt: report.Accounts[0].Health.CheckedAt}, {ID: 1, CheckedAt: now}}
	changed, err := s.Change(ctx, &group, selection, false, 42)
	require.NoError(t, err)
	require.Len(t, changed.Accounts, 1)
	require.Equal(t, "reduce", changed.Accounts[0].Outcome)
	require.Equal(t, 5, r.accounts[0].Concurrency)
	repeated, err := s.Change(ctx, &group, selection, false, 42)
	require.NoError(t, err)
	require.Equal(t, "conflict", repeated.Accounts[0].Outcome)
	require.Equal(t, 5, r.accounts[0].Concurrency)
	s.now = func() time.Time { return now.Add(time.Minute) }
	report, err = s.Check(ctx, nil)
	require.NoError(t, err)
	require.Nil(t, r.group)
	require.Equal(t, now, r.scopes[0].Since)
	require.Equal(t, "change_cooldown", report.Accounts[0].Health.Reason)
	restored, err := s.Change(ctx, nil, []OAuthHealthSelection{{ID: 1, CheckedAt: report.Accounts[0].Health.CheckedAt}}, true, 42)
	require.NoError(t, err)
	require.Equal(t, "conflict", restored.Accounts[0].Outcome)
	require.Equal(t, 5, r.accounts[0].Concurrency)
	require.Len(t, restored.Accounts[0].Health.History, 1)
}
func TestOAuthHealthStaleConcurrentAndDisabled(t *testing.T) {
	for _, mode := range []string{"stale", "manual change", "paused", "rescan", "cas conflict", "outside scope", "monitoring disabled"} {
		t.Run(mode, func(t *testing.T) {
			s, r, now := healthFixture()
			ctx := context.Background()
			report, err := s.Check(ctx, nil)
			require.NoError(t, err)
			sel := []OAuthHealthSelection{{ID: 1, CheckedAt: report.Accounts[0].Health.CheckedAt}}
			switch mode {
			case "stale":
				s.now = func() time.Time { return now.Add(6 * time.Minute) }
			case "manual change":
				r.accounts[0].Concurrency = 7
			case "paused":
				r.accounts[0].Schedulable = false
			case "rescan":
				sel[0].CheckedAt = now.Add(-time.Second)
			case "cas conflict":
				r.conflict = true
			case "outside scope":
				sel[0].ID = 99
			case "monitoring disabled":
				s.monitoringEnabled = func(context.Context) bool { return false }
			}
			before := r.accounts[0].Concurrency
			result, err := s.Change(ctx, nil, sel, false, 42)
			if mode == "monitoring disabled" {
				require.ErrorIs(t, err, ErrOpsDisabled)
			} else {
				require.NoError(t, err)
				require.Equal(t, "conflict", result.Accounts[0].Outcome)
			}
			require.Equal(t, before, r.accounts[0].Concurrency)
		})
	}
}
func TestOAuthHealthMissingLogsDoesNotPersistHealthy(t *testing.T) {
	s, r, _ := healthFixture()
	r.failObserve = true
	_, err := s.Check(context.Background(), nil)
	require.Error(t, err)
	require.Zero(t, r.saves)
}
