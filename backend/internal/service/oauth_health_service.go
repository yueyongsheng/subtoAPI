package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

const OAuthHealthExtraKey = "oauth_health"

// OAuthHealth is a dated observation, not an upstream capacity guarantee. It is
// computed only on an explicit admin action, never on the gateway request path.
type OAuthHealth struct {
	AccountID              int64               `json:"account_id"`
	CheckedAt              time.Time           `json:"checked_at"`
	WindowStart            time.Time           `json:"window_start"`
	WindowEnd              time.Time           `json:"window_end"`
	Status                 string              `json:"status"`
	Reason                 string              `json:"reason"`
	Concurrency            int                 `json:"concurrency"`
	RecommendedConcurrency int                 `json:"recommended_concurrency"`
	Stats                  OAuthHealthStats    `json:"stats"`
	LastChange             *OAuthHealthChange  `json:"last_change,omitempty"`
	History                []OAuthHealthChange `json:"history,omitempty"`
}

type OAuthHealthChange struct {
	At      time.Time `json:"at"`
	Before  int       `json:"before"`
	After   int       `json:"after"`
	Action  string    `json:"action"`
	ActorID int64     `json:"actor_id"`
}

type OAuthHealthStats struct {
	ObservedRequests    int        `json:"observed_requests"`
	OutputRequests      int        `json:"output_requests"`
	RateLimitedRequests int        `json:"rate_limited_requests"`
	QuotaRequests       int        `json:"quota_requests"`
	OverloadedRequests  int        `json:"overloaded_requests"`
	AuthRequests        int        `json:"auth_requests"`
	OtherErrorRequests  int        `json:"other_error_requests"`
	PressureRequests    int        `json:"pressure_requests"`
	PressureMinutes     int        `json:"pressure_minutes"`
	LatestPressureAt    *time.Time `json:"latest_pressure_at,omitempty"`
	P95FirstTokenMS     *float64   `json:"p95_first_token_ms,omitempty"`
}

type OAuthHealthAccount struct {
	ID                     int64           `json:"id"`
	Name                   string          `json:"name"`
	Platform               string          `json:"platform"`
	GroupIDs               []int64         `json:"group_ids"`
	Concurrency            int             `json:"concurrency"`
	Status                 string          `json:"-"`
	Schedulable            bool            `json:"schedulable"`
	RateLimitResetAt       *time.Time      `json:"-"`
	OverloadUntil          *time.Time      `json:"-"`
	TempUnschedulableUntil *time.Time      `json:"-"`
	ParentAccountID        *int64          `json:"-"`
	HasChildren            bool            `json:"-"`
	RawHealth              json.RawMessage `json:"-"`
	Health                 *OAuthHealth    `json:"health"`
	Outcome                string          `json:"outcome,omitempty"`
}

type OAuthHealthObservationScope struct {
	ID    int64     `json:"id"`
	Since time.Time `json:"since"`
}

type OAuthHealthRepository interface {
	ListOAuthHealthAccounts(context.Context, *int64) ([]OAuthHealthAccount, error)
	ObserveOAuthHealth(context.Context, []OAuthHealthObservationScope, time.Time) (map[int64]OAuthHealthStats, error)
	SaveOAuthHealth(context.Context, OAuthHealthAccount, *OAuthHealth, int, *int64) (bool, error)
}

type OAuthHealthReport struct {
	CheckedAt time.Time            `json:"checked_at"`
	GroupID   *int64               `json:"group_id"`
	Accounts  []OAuthHealthAccount `json:"accounts"`
}

type OAuthHealthSelection struct {
	ID        int64     `json:"id"`
	CheckedAt time.Time `json:"checked_at"`
}

type OAuthHealthService struct {
	mu                sync.Mutex
	repo              OAuthHealthRepository
	now               func() time.Time
	monitoringEnabled func(context.Context) bool
}

func NewOAuthHealthService(repo OAuthHealthRepository, ops *OpsService) *OAuthHealthService {
	return &OAuthHealthService{repo: repo, now: time.Now, monitoringEnabled: ops.IsMonitoringEnabled}
}

func decodeOAuthHealth(raw json.RawMessage) *OAuthHealth {
	var h OAuthHealth
	if len(raw) == 0 || json.Unmarshal(raw, &h) != nil || h.CheckedAt.IsZero() {
		return nil
	}
	return &h
}

func evaluateOAuthHealth(a OAuthHealthAccount, stats OAuthHealthStats, now time.Time) *OAuthHealth {
	h := &OAuthHealth{AccountID: a.ID, CheckedAt: now, WindowStart: now.Add(-30 * time.Minute), WindowEnd: now,
		Status: "insufficient", Reason: "insufficient", Concurrency: a.Concurrency,
		RecommendedConcurrency: a.Concurrency, Stats: stats}
	if old := decodeOAuthHealth(a.RawHealth); old != nil && old.AccountID == a.ID {
		h.LastChange, h.History = old.LastChange, old.History
		if old.LastChange != nil && old.LastChange.At.After(h.WindowStart) {
			h.WindowStart = old.LastChange.At
		}
	}
	switch {
	case stats.AuthRequests > 0:
		h.Status, h.Reason = "auth_error", "refresh_credentials"
	case stats.QuotaRequests > 0:
		h.Status, h.Reason = "quota_limited", "wait_reset"
	case a.RateLimitResetAt != nil && a.RateLimitResetAt.After(now):
		h.Status, h.Reason = "rate_limited", "wait_reset"
	case stats.RateLimitedRequests > 0:
		h.Status, h.Reason = "rate_limited", "observe"
	case stats.OverloadedRequests > 0 || (a.OverloadUntil != nil && a.OverloadUntil.After(now)):
		h.Status, h.Reason = "overloaded", "observe"
	case stats.OtherErrorRequests > 0:
		h.Status, h.Reason = "upstream_error", "inspect_errors"
	case stats.OutputRequests >= 20:
		h.Status, h.Reason = "stable", "observed_stable"
	}
	// A parent and its shadow share a credential; changing one alone would make
	// the inferred capacity misleading. Leave those accounts for manual review.
	switch {
	case a.ParentAccountID != nil || a.HasChildren:
		h.Reason = "shared_credential"
	case a.Status != StatusActive || !a.Schedulable:
		h.Reason = "paused"
	case h.Reason == "wait_reset" || h.Status == "auth_error":
	case a.TempUnschedulableUntil != nil && a.TempUnschedulableUntil.After(now):
		h.Reason = "cooldown"
	case h.LastChange != nil && now.Sub(h.LastChange.At) < 30*time.Minute:
		h.Reason = "change_cooldown"
	case a.Concurrency > 1 && stats.PressureRequests >= 5 && stats.PressureMinutes >= 3 &&
		stats.PressureRequests*5 >= stats.ObservedRequests && stats.LatestPressureAt != nil &&
		!stats.LatestPressureAt.Before(now.Add(-5*time.Minute)):
		h.Reason = "reduce_concurrency"
		h.RecommendedConcurrency = max(1, a.Concurrency/2)
	}
	return h
}

func (s *OAuthHealthService) Check(ctx context.Context, groupID *int64) (*OAuthHealthReport, error) {
	if !s.mu.TryLock() {
		return nil, errors.New("OAuth health operation in progress")
	}
	defer s.mu.Unlock()
	if s.monitoringEnabled == nil || !s.monitoringEnabled(ctx) {
		return nil, ErrOpsDisabled
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	now := s.now().UTC()
	accounts, err := s.repo.ListOAuthHealthAccounts(ctx, groupID)
	if err != nil {
		return nil, err
	}
	scopes := make([]OAuthHealthObservationScope, 0, len(accounts))
	for _, a := range accounts {
		since := now.Add(-30 * time.Minute)
		if old := decodeOAuthHealth(a.RawHealth); old != nil && old.AccountID == a.ID && old.LastChange != nil && old.LastChange.At.After(since) {
			since = old.LastChange.At
		}
		scopes = append(scopes, OAuthHealthObservationScope{ID: a.ID, Since: since})
	}
	stats, err := s.repo.ObserveOAuthHealth(ctx, scopes, now)
	if err != nil {
		return nil, err
	} // Missing logs must never become a green result.
	for i := range accounts {
		a := &accounts[i]
		a.Health = evaluateOAuthHealth(*a, stats[a.ID], now)
		ok, saveErr := s.repo.SaveOAuthHealth(ctx, *a, a.Health, a.Concurrency, groupID)
		if saveErr != nil {
			return nil, saveErr
		}
		if !ok {
			a.Outcome = "conflict"
		} else {
			a.Outcome = "checked"
		}
	}
	return &OAuthHealthReport{CheckedAt: now, GroupID: groupID, Accounts: accounts}, nil
}

func (s *OAuthHealthService) Change(ctx context.Context, groupID *int64, selected []OAuthHealthSelection, restore bool, actorID int64) (*OAuthHealthReport, error) {
	if !s.mu.TryLock() {
		return nil, errors.New("OAuth health operation in progress")
	}
	defer s.mu.Unlock()
	if !restore && (s.monitoringEnabled == nil || !s.monitoringEnabled(ctx)) {
		return nil, ErrOpsDisabled
	}
	if len(selected) == 0 || len(selected) > 2000 {
		return nil, errors.New("invalid account selection")
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	now := s.now().UTC()
	accounts, err := s.repo.ListOAuthHealthAccounts(ctx, groupID)
	if err != nil {
		return nil, err
	}
	byID := make(map[int64]OAuthHealthAccount, len(accounts))
	for _, a := range accounts {
		byID[a.ID] = a
	}
	result := &OAuthHealthReport{CheckedAt: now, GroupID: groupID, Accounts: []OAuthHealthAccount{}}
	seen := map[int64]bool{}
	for _, item := range selected {
		if seen[item.ID] {
			continue
		}
		seen[item.ID] = true
		a, ok := byID[item.ID]
		if !ok {
			result.Accounts = append(result.Accounts, OAuthHealthAccount{ID: item.ID, Outcome: "conflict"})
			continue
		}
		a.Health = decodeOAuthHealth(a.RawHealth)
		a.Outcome = "conflict"
		h := a.Health
		if h == nil || h.AccountID != a.ID || !h.CheckedAt.Equal(item.CheckedAt) || a.Concurrency != h.Concurrency {
			result.Accounts = append(result.Accounts, a)
			continue
		}
		var target int
		action := "reduce"
		if restore {
			if h.LastChange == nil || h.LastChange.Action != "reduce" || h.LastChange.After != a.Concurrency {
				result.Accounts = append(result.Accounts, a)
				continue
			}
			target, action = h.LastChange.Before, "restore"
		} else {
			fresh := evaluateOAuthHealth(a, h.Stats, now)
			if now.Sub(h.CheckedAt) > 5*time.Minute || now.Before(h.CheckedAt) ||
				fresh.Reason != "reduce_concurrency" || fresh.RecommendedConcurrency != h.RecommendedConcurrency {
				result.Accounts = append(result.Accounts, a)
				continue
			}
			target = fresh.RecommendedConcurrency
		}
		if target < 1 || target == a.Concurrency {
			result.Accounts = append(result.Accounts, a)
			continue
		}
		change := OAuthHealthChange{At: now, Before: a.Concurrency, After: target, Action: action, ActorID: actorID}
		h.LastChange = &change
		h.History = append(h.History, change)
		if len(h.History) > 10 {
			h.History = h.History[len(h.History)-10:]
		}
		h.Concurrency, h.RecommendedConcurrency, h.Reason = target, target, "change_cooldown"
		changed, saveErr := s.repo.SaveOAuthHealth(ctx, a, h, target, groupID)
		if saveErr != nil {
			return nil, fmt.Errorf("save OAuth health adjustment: %w", saveErr)
		}
		if changed {
			a.Concurrency = target
			a.Outcome = action
		} else {
			a.Health = decodeOAuthHealth(a.RawHealth)
		}
		result.Accounts = append(result.Accounts, a)
	}
	return result, nil
}
