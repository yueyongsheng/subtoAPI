package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

const OAuthHealthExtraKey = "oauth_health"
const OAuthHealthManualKey = "oauth_health_manual_concurrency"

const oauthHealthPolicyVersion = 2

// OAuthHealth is a dated observation, not an upstream capacity guarantee. It is
// computed only on an explicit admin action, never on the gateway request path.
type OAuthHealth struct {
	CooldownModels         []string            `json:"cooldown_models,omitempty"`
	PolicyVersion          int                 `json:"policy_version"`
	ManualConcurrency      bool                `json:"manual_concurrency,omitempty"`
	Action                 string              `json:"action,omitempty"`
	ObservationStartedAt   time.Time           `json:"observation_started_at"`
	HoldUntil              *time.Time          `json:"hold_until,omitempty"`
	RequiredModels         []string            `json:"required_models,omitempty"`
	CooldownUntil          *time.Time          `json:"cooldown_until,omitempty"`
	CooldownAlternatives   []int64             `json:"cooldown_alternatives,omitempty"`
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
	CurrentConcurrency  *int                    `json:"current_concurrency,omitempty"`
	OutputMinutes       int                     `json:"output_minutes"`
	Models              []OAuthHealthModelStats `json:"models,omitempty"`
	ObservedRequests    int                     `json:"observed_requests"`
	OutputRequests      int                     `json:"output_requests"`
	RateLimitedRequests int                     `json:"rate_limited_requests"`
	QuotaRequests       int                     `json:"quota_requests"`
	OverloadedRequests  int                     `json:"overloaded_requests"`
	AuthRequests        int                     `json:"auth_requests"`
	OtherErrorRequests  int                     `json:"other_error_requests"`
	PressureRequests    int                     `json:"pressure_requests"`
	PressureMinutes     int                     `json:"pressure_minutes"`
	LatestPressureAt    *time.Time              `json:"latest_pressure_at,omitempty"`
	P95FirstTokenMS     *float64                `json:"p95_first_token_ms,omitempty"`
}

type OAuthHealthModelStats struct {
	Model          string `json:"model"`
	OutputRequests int    `json:"output_requests"`
	ErrorRequests  int    `json:"error_requests"`
	HadError       bool   `json:"had_error"`
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
	ManualConcurrencyValue string          `json:"-"`
	RawHealth              json.RawMessage `json:"-"`
	Health                 *OAuthHealth    `json:"health"`
	Outcome                string          `json:"outcome,omitempty"`
}

type OAuthHealthObservationScope struct {
	ID           int64     `json:"id"`
	Since        time.Time `json:"since"`
	HistorySince time.Time `json:"history_since"`
}

type OAuthHealthRepository interface {
	ListOAuthHealthAccounts(context.Context, *int64) ([]OAuthHealthAccount, error)
	ObserveOAuthHealth(context.Context, []OAuthHealthObservationScope, time.Time) (map[int64]OAuthHealthStats, error)
	SaveOAuthHealth(context.Context, OAuthHealthAccount, *OAuthHealth, int, *int64) (bool, error)
	OAuthHealthCooldownAlternatives(context.Context, OAuthHealthAccount, []string, time.Time, []int64) ([]int64, error)
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
	mu                 sync.Mutex
	repo               OAuthHealthRepository
	now                func() time.Time
	monitoringEnabled  func(context.Context) bool
	currentConcurrency func(context.Context, []int64) (map[int64]int, error)
}

func NewOAuthHealthService(repo OAuthHealthRepository, ops *OpsService, concurrency *ConcurrencyService) *OAuthHealthService {
	return &OAuthHealthService{repo: repo, now: time.Now, monitoringEnabled: ops.IsMonitoringEnabled, currentConcurrency: concurrency.GetAccountConcurrencyBatch}
}

func decodeOAuthHealth(raw json.RawMessage) *OAuthHealth {
	var h OAuthHealth
	if len(raw) == 0 || json.Unmarshal(raw, &h) != nil || h.CheckedAt.IsZero() {
		return nil
	}
	return &h
}

func evaluateOAuthHealth(a OAuthHealthAccount, stats OAuthHealthStats, now time.Time) *OAuthHealth {
	h := &OAuthHealth{PolicyVersion: oauthHealthPolicyVersion, AccountID: a.ID, CheckedAt: now,
		WindowStart: now.Add(-30 * time.Minute), WindowEnd: now, ObservationStartedAt: now,
		Status: "insufficient", Reason: "insufficient", Concurrency: a.Concurrency,
		RecommendedConcurrency: a.Concurrency, Stats: stats}
	old := decodeOAuthHealth(a.RawHealth)
	if old != nil && old.AccountID == a.ID {
		h.LastChange, h.History, h.HoldUntil = old.LastChange, old.History, old.HoldUntil
		h.RequiredModels = append([]string{}, old.RequiredModels...)
		if old.PolicyVersion == oauthHealthPolicyVersion && !old.ObservationStartedAt.IsZero() {
			h.ObservationStartedAt = old.ObservationStartedAt
		}
		if old.LastChange != nil && old.LastChange.At.After(h.WindowStart) {
			h.WindowStart = old.LastChange.At
		}
	}
	required := map[string]bool{}
	for _, model := range h.RequiredModels {
		required[model] = true
	}
	for _, model := range stats.Models {
		if model.HadError {
			required[model.Model] = true
		}
	}
	h.RequiredModels = nil
	for model := range required {
		h.RequiredModels = append(h.RequiredModels, model)
	}
	sort.Strings(h.RequiredModels)
	modelsRecovered := true
	for model := range required {
		sufficient := false
		for _, m := range stats.Models {
			if m.Model == model && m.OutputRequests >= 20 && m.ErrorRequests == 0 {
				sufficient = true
			}
		}
		if !sufficient {
			modelsRecovered = false
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
	case stats.OutputRequests >= 20 && modelsRecovered:
		h.Status, h.Reason = "stable", "observed_stable"
		if len(required) > 0 || (old != nil && old.AccountID == a.ID && old.Status == "recovered") {
			h.Status = "recovered"
		}
	case stats.OutputRequests >= 20:
		h.Reason = "model_observation"
	}
	legacyLow := old != nil && old.AccountID == a.ID && old.LastChange != nil &&
		(old.LastChange.Action == "reduce" || old.LastChange.Action == "cooldown") && old.LastChange.After == a.Concurrency && a.Concurrency < 5
	manual := a.ManualConcurrencyValue != "" || (old != nil && old.AccountID == a.ID && old.ManualConcurrency) || a.Concurrency > 30 || (a.Concurrency < 5 && !legacyLow)
	if old != nil && old.AccountID == a.ID && old.Concurrency != a.Concurrency {
		manual = true
	}
	h.ManualConcurrency = manual
	pressure := stats.PressureRequests >= 5 && stats.PressureMinutes >= 3 && stats.PressureRequests*5 >= stats.ObservedRequests && stats.LatestPressureAt != nil && !stats.LatestPressureAt.Before(now.Add(-5*time.Minute))
	switch {
	case a.ParentAccountID != nil || a.HasChildren:
		h.Reason = "shared_credential"
	case a.Status != StatusActive || !a.Schedulable:
		h.Reason = "paused"
	case manual:
		h.Reason = "manual_concurrency"
	case h.Reason == "wait_reset" || h.Status == "auth_error":
	case a.TempUnschedulableUntil != nil && a.TempUnschedulableUntil.After(now):
		h.Reason = "cooldown"
		h.CooldownUntil = a.TempUnschedulableUntil
	// Failed increases can be rolled back during their observation period.
	case pressure && h.LastChange != nil && h.LastChange.Action == "increase" && h.LastChange.After == a.Concurrency:
		if h.LastChange.Before < 5 {
			h.Reason, h.Action = "cooldown_recommended", "cooldown"
		} else {
			h.Reason, h.Action = "rollback_increase", "rollback"
			h.RecommendedConcurrency = h.LastChange.Before
		}
	case h.HoldUntil != nil && h.HoldUntil.After(now):
		h.Reason = "change_cooldown"
	case h.LastChange != nil && now.Sub(h.LastChange.At) < 30*time.Minute:
		h.Reason = "change_cooldown"
	case pressure:
		if a.Concurrency > 5 {
			h.Reason, h.Action = "reduce_concurrency", "reduce"
			h.RecommendedConcurrency = max(5, (a.Concurrency/10)*5)
		} else {
			h.Reason, h.Action = "cooldown_recommended", "cooldown"
		}
	case h.Status == "stable" || h.Status == "recovered":
		switch {
		case a.Concurrency >= 30:
			h.Reason = "maximum_concurrency"
		case now.Sub(h.ObservationStartedAt) < 30*time.Minute:
			h.Reason = "recovery_observation"
		case stats.OutputMinutes < 3:
			h.Reason = "recovery_samples"
		case stats.CurrentConcurrency == nil:
			h.Reason = "load_unavailable"
		case *stats.CurrentConcurrency*5 < a.Concurrency*4:
			h.Reason = "low_demand"
		default:
			h.Reason, h.Action = "increase_concurrency", "increase"
			h.RecommendedConcurrency = min(30, a.Concurrency+5)
			if a.Concurrency < 5 {
				h.RecommendedConcurrency = 5
			}
		}
	}
	return h
}

func (s *OAuthHealthService) attachLoad(ctx context.Context, stats map[int64]OAuthHealthStats, ids []int64) {
	if s.currentConcurrency == nil {
		return
	}
	loads, err := s.currentConcurrency(ctx, ids)
	if err != nil {
		return
	}
	for id, n := range loads {
		v := stats[id]
		v.CurrentConcurrency = &n
		stats[id] = v
	}
}

func (s *OAuthHealthService) prepareCooldown(ctx context.Context, a OAuthHealthAccount, h *OAuthHealth, now time.Time, excluded []int64) error {
	if h.Action != "cooldown" {
		return nil
	}
	models := map[string]bool{}
	for _, model := range h.RequiredModels {
		models[model] = true
	}
	for _, model := range h.Stats.Models {
		models[model.Model] = true
	}
	for model := range models {
		h.CooldownModels = append(h.CooldownModels, model)
	}
	sort.Strings(h.CooldownModels)
	ids, err := s.repo.OAuthHealthCooldownAlternatives(ctx, a, h.CooldownModels, now, excluded)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		h.Action = ""
		h.Reason = "no_cooldown_alternative"
	} else {
		h.CooldownAlternatives = ids
	}
	return nil
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
	ids := make([]int64, 0, len(accounts))
	for _, a := range accounts {
		since := now.Add(-30 * time.Minute)
		if old := decodeOAuthHealth(a.RawHealth); old != nil && old.AccountID == a.ID && old.LastChange != nil && old.LastChange.At.After(since) {
			since = old.LastChange.At
		}
		historySince := since
		if old := decodeOAuthHealth(a.RawHealth); old != nil && old.AccountID == a.ID && old.LastChange != nil && old.PolicyVersion != oauthHealthPolicyVersion {
			historySince = old.LastChange.At.Add(-30 * time.Minute)
			if historySince.Before(now.Add(-24 * time.Hour)) {
				historySince = now.Add(-24 * time.Hour)
			}
		}
		scopes = append(scopes, OAuthHealthObservationScope{ID: a.ID, Since: since, HistorySince: historySince})
		ids = append(ids, a.ID)
	}
	stats, err := s.repo.ObserveOAuthHealth(ctx, scopes, now)
	if err != nil {
		return nil, err
	} // Missing logs must never become a green result.
	s.attachLoad(ctx, stats, ids)
	for i := range accounts {
		a := &accounts[i]
		a.Health = evaluateOAuthHealth(*a, stats[a.ID], now)
		if err := s.prepareCooldown(ctx, *a, a.Health, now, nil); err != nil {
			return nil, err
		}
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
	if s.monitoringEnabled == nil || !s.monitoringEnabled(ctx) {
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
		if h.PolicyVersion != oauthHealthPolicyVersion || now.Sub(h.CheckedAt) > 5*time.Minute || now.Before(h.CheckedAt) {
			result.Accounts = append(result.Accounts, a)
			continue
		}
		// Re-read evidence at apply time: a green report must not increase capacity
		// after new upstream errors arrived. This remains off the request path.
		since := now.Add(-30 * time.Minute)
		if h.LastChange != nil && h.LastChange.At.After(since) {
			since = h.LastChange.At
		}
		observed, observeErr := s.repo.ObserveOAuthHealth(ctx, []OAuthHealthObservationScope{{ID: a.ID, Since: since, HistorySince: since}}, now)
		if observeErr != nil {
			return nil, observeErr
		}
		s.attachLoad(ctx, observed, []int64{a.ID})
		fresh := evaluateOAuthHealth(a, observed[a.ID], now)
		excluded := make([]int64, 0, len(selected))
		for _, other := range selected {
			excluded = append(excluded, other.ID)
		}
		if err := s.prepareCooldown(ctx, a, fresh, now, excluded); err != nil {
			return nil, err
		}
		if fresh.Action == "" || fresh.Action != h.Action || fresh.RecommendedConcurrency != h.RecommendedConcurrency || (restore && fresh.Action != "increase") {
			result.Accounts = append(result.Accounts, a)
			continue
		}
		target, action := fresh.RecommendedConcurrency, fresh.Action
		if target < 5 && action != "cooldown" || target > 30 {
			result.Accounts = append(result.Accounts, a)
			continue
		}
		h = fresh
		hold := now.Add(30 * time.Minute)
		if action == "rollback" {
			hold = now.Add(60 * time.Minute)
		}
		h.HoldUntil = &hold
		if action == "cooldown" {
			until := now.Add(3 * time.Minute)
			h.CooldownUntil = &until
		}
		a.Health = h
		change := OAuthHealthChange{At: now, Before: a.Concurrency, After: target, Action: action, ActorID: actorID}
		h.LastChange = &change
		h.ObservationStartedAt = now
		h.History = append(h.History, change)
		if len(h.History) > 10 {
			h.History = h.History[len(h.History)-10:]
		}
		h.Concurrency, h.RecommendedConcurrency, h.Reason = target, target, "change_cooldown"
		h.Action = ""
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
