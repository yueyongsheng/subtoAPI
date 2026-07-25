package service

import (
	"context"
	"strings"
	"sync"
	"time"
)

var monitorRecoveryBackoff = [...]time.Duration{
	5 * time.Minute,
	15 * time.Minute,
	30 * time.Minute,
	60 * time.Minute,
}

func (s *ChannelMonitorService) RunCheckModel(ctx context.Context, id int64, model string) (*CheckResult, error) {
	m, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if m.APIKeyDecryptFailed {
		return nil, ErrChannelMonitorAPIKeyDecryptFailed
	}
	model = strings.TrimSpace(model)
	if !monitorContainsModel(m, model) {
		return nil, ErrChannelMonitorModelNotConfigured
	}
	pingMs := pingEndpointOrigin(ctx, m.Endpoint)
	result := runCheckForModel(ctx, m.Provider, m.Endpoint, m.APIKey, model, &CheckOptions{
		APIMode:          m.APIMode,
		ExtraHeaders:     m.ExtraHeaders,
		BodyOverrideMode: m.BodyOverrideMode,
		BodyOverride:     m.BodyOverride,
	})
	result.PingLatencyMs = pingMs
	s.persistCheckResults(ctx, m, []*CheckResult{result})
	return result, nil
}

func (s *ChannelMonitorService) RunHybridCycle(ctx context.Context, id int64) error {
	m, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if !m.Enabled || m.Mode != MonitorModeHybrid {
		return nil
	}
	now := time.Now()
	if m.LastPingAt == nil || now.Sub(*m.LastPingAt) >= monitorHybridPingInterval {
		ping := pingEndpointOrigin(ctx, m.Endpoint)
		if err := s.repo.MarkPing(ctx, m.ID, ping, now); err != nil {
			return err
		}
	}
	models := append([]string{m.PrimaryModel}, m.ExtraModels...)
	due := make([]string, 0, len(models))
	for _, model := range models {
		model = strings.TrimSpace(model)
		if model == "" {
			continue
		}
		history, err := s.repo.ListHistory(ctx, m.ID, model, 20)
		if err != nil {
			return err
		}
		if hybridModelProbeDue(now, time.Duration(m.IntervalSeconds)*time.Second, history) {
			due = append(due, model)
		}
	}
	var wg sync.WaitGroup
	var firstErr error
	var errMu sync.Mutex
	for _, model := range due {
		model := model
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.RunCheckModel(ctx, m.ID, model); err != nil {
				errMu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				errMu.Unlock()
			}
		}()
	}
	wg.Wait()
	return firstErr
}

func monitorContainsModel(m *ChannelMonitor, model string) bool {
	if m == nil || model == "" {
		return false
	}
	if m.PrimaryModel == model {
		return true
	}
	for _, extra := range m.ExtraModels {
		if extra == model {
			return true
		}
	}
	return false
}

func hybridModelProbeDue(now time.Time, idleInterval time.Duration, history []*ChannelMonitorHistoryEntry) bool {
	if idleInterval <= 0 {
		idleInterval = monitorHybridIdleInterval
	}
	if len(history) == 0 {
		return true
	}
	latest := history[0]
	if monitorObservationSucceeded(latest) {
		return now.Sub(latest.CheckedAt) >= idleInterval
	}
	failureStreak := 0
	for _, entry := range history {
		if monitorObservationSucceeded(entry) {
			break
		}
		if monitorObservationFailed(entry) {
			failureStreak++
			continue
		}
		break
	}
	if failureStreak == 0 {
		return now.Sub(latest.CheckedAt) >= idleInterval
	}
	index := failureStreak - 1
	if index >= len(monitorRecoveryBackoff) {
		index = len(monitorRecoveryBackoff) - 1
	}
	return now.Sub(latest.CheckedAt) >= monitorRecoveryBackoff[index]
}

func monitorObservationSucceeded(entry *ChannelMonitorHistoryEntry) bool {
	if entry == nil {
		return false
	}
	if entry.Status == MonitorStatusOperational || entry.SuccessCount > 0 {
		return true
	}
	// Rows created before the hybrid migration have zeroed counters. A degraded
	// active probe without an explicit failure was a successful slow probe.
	return entry.Source == MonitorSourceActiveProbe && entry.Status == MonitorStatusDegraded && entry.FailureCount == 0
}

func monitorObservationFailed(entry *ChannelMonitorHistoryEntry) bool {
	if entry == nil || entry.SuccessCount > 0 {
		return false
	}
	return entry.FailureCount > 0 || entry.Status == MonitorStatusError || entry.Status == MonitorStatusFailed
}
