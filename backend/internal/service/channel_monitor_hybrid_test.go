//go:build unit

package service

import (
	"testing"
	"time"
)

func TestTrafficBucketStatus(t *testing.T) {
	previousFailure := &ChannelMonitorHistoryEntry{
		Source:       MonitorSourceRealTraffic,
		FailureCount: 1,
	}
	tests := []struct {
		name     string
		bucket   *channelObserverBucket
		previous *ChannelMonitorHistoryEntry
		want     string
	}{
		{name: "success", bucket: &channelObserverBucket{samples: 1, successes: 1}, want: MonitorStatusOperational},
		{name: "slow success", bucket: &channelObserverBucket{samples: 1, successes: 1, slow: 1}, want: MonitorStatusDegraded},
		{name: "recovered upstream failure", bucket: &channelObserverBucket{samples: 1, successes: 1, recoveredErrors: 1}, want: MonitorStatusDegraded},
		{name: "mixed final outcomes", bucket: &channelObserverBucket{samples: 2, successes: 1, failures: 1}, want: MonitorStatusDegraded},
		{name: "first final failure", bucket: &channelObserverBucket{samples: 1, failures: 1}, want: MonitorStatusDegraded},
		{name: "second failure bucket", bucket: &channelObserverBucket{samples: 1, failures: 1}, previous: previousFailure, want: MonitorStatusError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := trafficBucketStatus(tt.bucket, tt.previous); got != tt.want {
				t.Fatalf("trafficBucketStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestObserveChannelHealth_FiltersProbeKeysAndIsolatesGroupModel(t *testing.T) {
	svc := &ChannelMonitorService{}
	svc.observer.init()
	svc.observer.routes[channelObserverRouteKey{groupID: 2, model: "gpt-5.5"}] = []channelObserverTarget{{monitorID: 10, model: "gpt-5.5"}}
	svc.observer.probeKeys[99] = struct{}{}

	svc.ObserveChannelHealth(ChannelHealthObservation{
		GroupID: 2, APIKeyID: 7, Model: "gpt-5.5", Success: true,
		Duration: 7 * time.Second, ObservedAt: time.Now(),
	})
	select {
	case event := <-svc.observer.queue:
		if event.target.monitorID != 10 || !event.success || !event.slow {
			t.Fatalf("unexpected routed event: %+v", event)
		}
	default:
		t.Fatal("matching customer traffic was not queued")
	}

	ignored := []ChannelHealthObservation{
		{GroupID: 2, APIKeyID: 99, Model: "gpt-5.5", Success: true},
		{GroupID: 3, APIKeyID: 7, Model: "gpt-5.5", Success: true},
		{GroupID: 2, APIKeyID: 7, Model: "gpt-5.6-sol", Success: true},
		{GroupID: 2, APIKeyID: 7, Model: "gpt-5.5"},
	}
	for _, observation := range ignored {
		svc.ObserveChannelHealth(observation)
	}
	if got := len(svc.observer.queue); got != 0 {
		t.Fatalf("ignored observations queued %d event(s)", got)
	}
}

func TestHybridModelProbeDue_IdleAndRecovery(t *testing.T) {
	now := time.Date(2026, 7, 25, 10, 0, 0, 0, time.UTC)
	if !hybridModelProbeDue(now, time.Hour, nil) {
		t.Fatal("model without history should be probed on its first scheduler tick")
	}
	recentSuccess := []*ChannelMonitorHistoryEntry{{
		Source: MonitorSourceRealTraffic, Status: MonitorStatusOperational,
		SuccessCount: 1, CheckedAt: now.Add(-59 * time.Minute),
	}}
	if hybridModelProbeDue(now, time.Hour, recentSuccess) {
		t.Fatal("recent real success should suppress active probing")
	}
	recentSuccess[0].CheckedAt = now.Add(-time.Hour)
	if !hybridModelProbeDue(now, time.Hour, recentSuccess) {
		t.Fatal("one idle hour should make the model due")
	}

	recovered := []*ChannelMonitorHistoryEntry{{
		Source: MonitorSourceRealTraffic, Status: MonitorStatusDegraded,
		SuccessCount: 1, RecoveredErrorCount: 1, CheckedAt: now.Add(-time.Minute),
	}}
	if hybridModelProbeDue(now, time.Hour, recovered) {
		t.Fatal("a recovered successful request should reset the idle timer")
	}
}

func TestHybridModelProbeDue_FailureBackoff(t *testing.T) {
	now := time.Date(2026, 7, 25, 10, 0, 0, 0, time.UTC)
	entry := func(checkedAt time.Time) *ChannelMonitorHistoryEntry {
		return &ChannelMonitorHistoryEntry{
			Source: MonitorSourceActiveProbe, Status: MonitorStatusError,
			FailureCount: 1, CheckedAt: checkedAt,
		}
	}
	tests := []struct {
		name    string
		streak  int
		elapsed time.Duration
		want    bool
	}{
		{name: "first waits five", streak: 1, elapsed: 4 * time.Minute, want: false},
		{name: "first due at five", streak: 1, elapsed: 5 * time.Minute, want: true},
		{name: "second waits fifteen", streak: 2, elapsed: 14 * time.Minute, want: false},
		{name: "second due at fifteen", streak: 2, elapsed: 15 * time.Minute, want: true},
		{name: "third due at thirty", streak: 3, elapsed: 30 * time.Minute, want: true},
		{name: "fourth due at sixty", streak: 4, elapsed: 60 * time.Minute, want: true},
		{name: "later stays hourly", streak: 7, elapsed: 59 * time.Minute, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			history := make([]*ChannelMonitorHistoryEntry, tt.streak)
			for i := range history {
				history[i] = entry(now.Add(-tt.elapsed - time.Duration(i)*time.Second))
			}
			if got := hybridModelProbeDue(now, time.Hour, history); got != tt.want {
				t.Fatalf("hybridModelProbeDue() = %v, want %v", got, tt.want)
			}
		})
	}
}
