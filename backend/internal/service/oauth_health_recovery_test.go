package service

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestOAuthHealthRecoveryStepsAndManualPriority(t *testing.T) {
	now := time.Date(2026, 9, 16, 4, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		name          string
		limit, target int
		manual        bool
		reason        string
	}{
		{"five to ten", 5, 10, false, "increase_concurrency"}, {"twenty five to thirty", 25, 30, false, "increase_concurrency"},
		{"ceiling", 30, 30, false, "maximum_concurrency"}, {"manual high", 50, 50, false, "manual_concurrency"},
		{"manual in range", 10, 10, true, "manual_concurrency"}, {"manual low", 3, 3, false, "manual_concurrency"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			a := OAuthHealthAccount{ID: 1, Concurrency: tt.limit, Status: StatusActive, Schedulable: true}
			if tt.manual {
				a.ManualConcurrencyValue = "manual"
			}
			a.RawHealth, _ = json.Marshal(OAuthHealth{PolicyVersion: 2, AccountID: 1, CheckedAt: now.Add(-time.Hour), ObservationStartedAt: now.Add(-time.Hour), Concurrency: tt.limit})
			load := tt.limit
			stats := OAuthHealthStats{OutputRequests: 100, ObservedRequests: 100, OutputMinutes: 10, CurrentConcurrency: &load}
			h := evaluateOAuthHealth(a, stats, now)
			require.Equal(t, tt.reason, h.Reason)
			require.Equal(t, tt.target, h.RecommendedConcurrency)
		})
	}
}
func TestOAuthHealthRecoveryRequiresTrafficAndAffectedModels(t *testing.T) {
	now := time.Date(2026, 9, 16, 4, 0, 0, 0, time.UTC)
	a := OAuthHealthAccount{ID: 1, Concurrency: 5, Status: StatusActive, Schedulable: true}
	a.RawHealth, _ = json.Marshal(OAuthHealth{PolicyVersion: 2, AccountID: 1, CheckedAt: now.Add(-time.Hour), ObservationStartedAt: now.Add(-time.Hour), Concurrency: 5, RequiredModels: []string{"affected"}})
	load := 5
	stats := OAuthHealthStats{OutputRequests: 100, ObservedRequests: 100, OutputMinutes: 10, CurrentConcurrency: &load, Models: []OAuthHealthModelStats{{Model: "other", OutputRequests: 100}}}
	require.Equal(t, "model_observation", evaluateOAuthHealth(a, stats, now).Reason)
	stats.Models = append(stats.Models, OAuthHealthModelStats{Model: "affected", OutputRequests: 20})
	h := evaluateOAuthHealth(a, stats, now)
	require.Equal(t, "recovered", h.Status)
	require.Equal(t, "increase", h.Action)
	stats.CurrentConcurrency = nil
	require.Equal(t, "load_unavailable", evaluateOAuthHealth(a, stats, now).Reason)
	low := 1
	stats.CurrentConcurrency = &low
	require.Equal(t, "low_demand", evaluateOAuthHealth(a, stats, now).Reason)
	require.Equal(t, "insufficient", evaluateOAuthHealth(a, OAuthHealthStats{}, now).Status)
}
func TestOAuthHealthLegacyLowRecoveryAndFailedIncrease(t *testing.T) {
	now := time.Date(2026, 9, 16, 4, 0, 0, 0, time.UTC)
	a := OAuthHealthAccount{ID: 1, Concurrency: 2, Status: StatusActive, Schedulable: true}
	a.RawHealth, _ = json.Marshal(OAuthHealth{PolicyVersion: 2, AccountID: 1, CheckedAt: now.Add(-time.Hour), ObservationStartedAt: now.Add(-time.Hour), Concurrency: 2, LastChange: &OAuthHealthChange{At: now.Add(-time.Hour), Before: 5, After: 2, Action: "reduce"}})
	load := 2
	stats := OAuthHealthStats{OutputRequests: 50, ObservedRequests: 50, OutputMinutes: 5, CurrentConcurrency: &load}
	require.Equal(t, 5, evaluateOAuthHealth(a, stats, now).RecommendedConcurrency)
	a.Concurrency = 15
	hold := now.Add(time.Minute)
	a.RawHealth, _ = json.Marshal(OAuthHealth{PolicyVersion: 2, AccountID: 1, CheckedAt: now.Add(-time.Minute), Concurrency: 15, HoldUntil: &hold, LastChange: &OAuthHealthChange{At: now.Add(-time.Minute), Before: 10, After: 15, Action: "increase"}})
	recent := now.Add(-time.Second)
	stats = OAuthHealthStats{ObservedRequests: 20, PressureRequests: 5, PressureMinutes: 3, OverloadedRequests: 5, LatestPressureAt: &recent}
	h := evaluateOAuthHealth(a, stats, now)
	require.Equal(t, "rollback", h.Action)
	require.Equal(t, 10, h.RecommendedConcurrency)
}
func TestOAuthHealthApplyIncreaseRechecksErrorsAndCaps(t *testing.T) {
	for _, newError := range []bool{false, true} {
		t.Run(map[bool]string{false: "increase", true: "new error blocks"}[newError], func(t *testing.T) {
			s, r, now := healthFixture()
			r.accounts[0].Concurrency = 25
			r.accounts[0].RawHealth, _ = json.Marshal(OAuthHealth{PolicyVersion: 2, AccountID: 1, Concurrency: 25, CheckedAt: now.Add(-time.Hour), ObservationStartedAt: now.Add(-time.Hour)})
			r.stats[1] = OAuthHealthStats{OutputRequests: 50, ObservedRequests: 50, OutputMinutes: 5}
			s.currentConcurrency = func(context.Context, []int64) (map[int64]int, error) { return map[int64]int{1: 23}, nil }
			report, err := s.Check(context.Background(), nil)
			require.NoError(t, err)
			require.Equal(t, "increase", report.Accounts[0].Health.Action)
			if newError {
				x := r.stats[1]
				x.OverloadedRequests = 1
				r.stats[1] = x
			}
			changed, err := s.Change(context.Background(), nil, []OAuthHealthSelection{{ID: 1, CheckedAt: report.CheckedAt}}, false, 1)
			require.NoError(t, err)
			if newError {
				require.Equal(t, "conflict", changed.Accounts[0].Outcome)
				require.Equal(t, 25, r.accounts[0].Concurrency)
			} else {
				require.Equal(t, "increase", changed.Accounts[0].Outcome)
				require.Equal(t, 30, r.accounts[0].Concurrency)
			}
		})
	}
}
func TestOAuthHealthFloorCooldownAndAlternativeGuard(t *testing.T) {
	s, r, _ := healthFixture()
	r.accounts[0].Concurrency = 5
	report, err := s.Check(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, "no_cooldown_alternative", report.Accounts[0].Health.Reason)
	r.alternatives = []int64{2}
	report, err = s.Check(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, "cooldown", report.Accounts[0].Health.Action)
	changed, err := s.Change(context.Background(), nil, []OAuthHealthSelection{{ID: 1, CheckedAt: report.CheckedAt}}, false, 1)
	require.NoError(t, err)
	require.Equal(t, "cooldown", changed.Accounts[0].Outcome)
	require.Equal(t, 5, r.accounts[0].Concurrency)
	require.NotNil(t, r.accounts[0].TempUnschedulableUntil)
}
func TestOAuthHealthReductionFloors(t *testing.T) {
	_, r, now := healthFixture()
	for _, tt := range []struct{ before, after int }{{30, 15}, {15, 5}, {10, 5}, {6, 5}} {
		a := r.accounts[0]
		a.Concurrency = tt.before
		h := evaluateOAuthHealth(a, r.stats[1], now)
		require.Equal(t, tt.after, h.RecommendedConcurrency)
	}
}
