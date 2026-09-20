package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	pluginv1 "github.com/Wei-Shaw/sub2api/pkg/pluginapi/v1"
	"github.com/stretchr/testify/require"
)

type stateAlertSourceStub struct {
	plugins      []*PluginInstallation
	status       *pluginv1.HealthResponse
	err          error
	lists, reads int
}

func (s *stateAlertSourceStub) List(context.Context) ([]*PluginInstallation, error) {
	s.lists++
	return s.plugins, s.err
}
func (s *stateAlertSourceStub) Status(context.Context, int64) (*pluginv1.HealthResponse, error) {
	s.reads++
	return s.status, s.err
}
func stateTestSource(t *testing.T, tickets ...stateAlertTicket) *stateAlertSourceStub {
	t.Helper()
	if tickets == nil {
		tickets = []stateAlertTicket{}
	}
	raw, err := json.Marshal(stateAlertSnapshot{HostReady: true, Tickets: tickets})
	require.NoError(t, err)
	return &stateAlertSourceStub{plugins: []*PluginInstallation{{ID: 1, PluginKey: stateKitPluginKey, State: PluginStateEnabled}}, status: &pluginv1.HealthResponse{Healthy: true, StatusJson: string(raw)}}
}
func stateTestTicket(state string, remaining int64) stateAlertTicket {
	return stateAlertTicket{AccountID: 184, Model: "gpt-6-astra", State: state, Remaining: remaining, ExpiresAt: time.Now().Add(time.Duration(remaining) * time.Second).UTC().Format(time.RFC3339)}
}

func TestStateTicketMetrics(t *testing.T) {
	svc := &OpsAlertEvaluatorService{}
	for _, tc := range []struct {
		state                string
		remaining            int64
		failure              string
		renewal, unavailable float64
	}{
		{"ready", 1800, "", 0, 0}, {"renewing", 600, "upstream_forbidden", 1, 0},
		{"ready", 601, "upstream_forbidden", 0, 0}, {"ready", 500, "", 0, 0},
		{"expired", 0, "attempts_exhausted", 0, 1}, {"cooldown", 0, "harvest_failed", 0, 1},
		{"harvesting", 0, "", 0, 1}, {"queued", 0, "", 0, 1},
		{"disabled", 0, "", 0, 0}, {"waiting_host", 0, "", 0, 0}, {"waiting_account", 0, "", 0, 0},
	} {
		t.Run(fmt.Sprintf("%s/%d/%s", tc.state, tc.remaining, tc.failure), func(t *testing.T) {
			ticket := stateTestTicket(tc.state, tc.remaining)
			ticket.LastError = tc.failure
			source := stateTestSource(t, ticket)
			snapshot := parseStateAlertSnapshot(source.status.StatusJson)
			require.True(t, snapshot.valid)
			for metric, want := range map[string]float64{StateTicketRenewalFailedMetric: tc.renewal, StateTicketUnavailableMetric: tc.unavailable} {
				value, description, ok := svc.stateTicketMetric(context.Background(), snapshot, &OpsAlertRule{MetricType: metric}, time.Now(), "", nil)
				require.True(t, ok)
				require.Equal(t, want, value)
				if want > 0 {
					require.Contains(t, description, "账号 #184 / gpt-6-astra")
				}
			}
		})
	}
	// Absolute expiry wins over the plugin's countdown.
	ticket := stateTestTicket("ready", 500)
	ticket.ExpiresAt = time.Now().Add(-time.Second).UTC().Format(time.RFC3339)
	snapshot := parseStateAlertSnapshot(stateTestSource(t, ticket).status.StatusJson)
	value, _, ok := svc.stateTicketMetric(context.Background(), snapshot, &OpsAlertRule{MetricType: StateTicketUnavailableMetric}, time.Now(), "", nil)
	require.True(t, ok)
	require.Equal(t, float64(1), value)
}

func TestStateSnapshotUnknownAndSanitization(t *testing.T) {
	ticket := stateTestTicket("ready", 500)
	ticket.LastError = "secret-raw-proxy-error"
	raw := stateTestSource(t, ticket).status.StatusJson
	snapshot := parseStateAlertSnapshot(raw)
	require.True(t, snapshot.valid)
	require.Equal(t, "unknown", snapshot.Tickets[0].LastError)
	for _, broken := range []string{"{}", `{"host_ready":true}`, strings.Replace(raw, `"host_ready":true`, `"host_ready":false`, 1), strings.Replace(raw, `"remaining_seconds":500,`, "", 1), strings.Replace(raw, `"remaining_seconds":500`, `"remaining_seconds":null`, 1), strings.Replace(raw, `"remaining_seconds":500`, `"remaining_seconds":-1`, 1), strings.Replace(raw, `"state":"ready"`, `"state":"future"`, 1), strings.Replace(raw, `"gpt-6-astra"`, `"<secret>"`, 1), strings.Repeat(" ", 2*1024*1024+1)} {
		require.False(t, parseStateAlertSnapshot(broken).valid)
	}
	require.False(t, parseStateAlertSnapshot(stateTestSource(t, ticket, ticket).status.StatusJson).valid)
	svc := &OpsAlertEvaluatorService{}
	_, description, ok := svc.stateTicketMetric(context.Background(), snapshot, &OpsAlertRule{MetricType: StateTicketRenewalFailedMetric}, time.Now(), "", nil)
	require.True(t, ok)
	require.NotContains(t, description, "secret-raw")
	event := &OpsAlertEvent{Description: description}
	require.Contains(t, buildOpsAlertEmailBody(&OpsAlertRule{}, event), "gpt-6-astra")
	require.NotContains(t, opsAlertEmailVariables(&OpsAlertRule{}, event)["alert_description"], "secret-raw")
}

func TestStateTicketSourceAndScopes(t *testing.T) {
	rules := []*OpsAlertRule{{Enabled: true, MetricType: StateTicketUnavailableMetric}}
	source := stateTestSource(t, stateTestTicket("expired", 0))
	require.False(t, loadStateAlertSnapshot(context.Background(), source, nil).valid)
	require.Zero(t, source.lists)
	require.True(t, loadStateAlertSnapshot(context.Background(), source, rules).valid)
	require.Equal(t, 1, source.reads)
	source.err = errors.New("offline")
	require.False(t, loadStateAlertSnapshot(context.Background(), source, rules).valid)
	source.err = nil
	source.status.Healthy = false
	require.False(t, loadStateAlertSnapshot(context.Background(), source, rules).valid)
	source.plugins[0].State = PluginStateDisabled
	require.True(t, loadStateAlertSnapshot(context.Background(), source, rules).valid)
	source.plugins = nil
	require.True(t, loadStateAlertSnapshot(context.Background(), source, rules).valid)
	snapshot := parseStateAlertSnapshot(stateTestSource(t, stateTestTicket("expired", 0)).status.StatusJson)
	groupID := int64(3)
	svc := &OpsAlertEvaluatorService{opsService: &OpsService{getAccountAvailability: func(_ context.Context, platform string, group *int64) (*OpsAccountAvailability, error) {
		require.Equal(t, "openai", platform)
		require.Equal(t, groupID, *group)
		return &OpsAccountAvailability{Accounts: map[int64]*AccountAvailability{185: {AccountID: 185}}}, nil
	}}}
	value, _, ok := svc.stateTicketMetric(context.Background(), snapshot, rules[0], time.Now(), "openai", &groupID)
	require.True(t, ok)
	require.Zero(t, value)
	value, _, ok = svc.stateTicketMetric(context.Background(), snapshot, rules[0], time.Now(), "anthropic", nil)
	require.True(t, ok)
	require.Zero(t, value)
	_, _, ok = svc.stateTicketMetric(context.Background(), stateAlertSnapshot{}, rules[0], time.Now(), "", nil)
	require.False(t, ok)
}

type stateAlertRepoStub struct {
	OpsRepository
	rules    []*OpsAlertRule
	events   []*OpsAlertEvent
	resolved int
}

func (r *stateAlertRepoStub) ListAlertRules(context.Context) ([]*OpsAlertRule, error) {
	return r.rules, nil
}
func (r *stateAlertRepoStub) GetLatestSystemMetrics(context.Context, int) (*OpsSystemMetricsSnapshot, error) {
	return nil, nil
}
func (r *stateAlertRepoStub) UpsertJobHeartbeat(context.Context, *OpsUpsertJobHeartbeatInput) error {
	return nil
}
func (r *stateAlertRepoStub) GetActiveAlertEvent(_ context.Context, id int64) (*OpsAlertEvent, error) {
	for _, e := range r.events {
		if e.RuleID == id && e.Status == OpsAlertStatusFiring {
			return e, nil
		}
	}
	return nil, nil
}
func (r *stateAlertRepoStub) GetLatestAlertEvent(_ context.Context, id int64) (*OpsAlertEvent, error) {
	for i := len(r.events) - 1; i >= 0; i-- {
		if r.events[i].RuleID == id {
			return r.events[i], nil
		}
	}
	return nil, nil
}
func (r *stateAlertRepoStub) CreateAlertEvent(_ context.Context, e *OpsAlertEvent) (*OpsAlertEvent, error) {
	e.ID = int64(len(r.events) + 1)
	r.events = append(r.events, e)
	return e, nil
}
func (r *stateAlertRepoStub) UpdateAlertEventStatus(_ context.Context, id int64, status string, at *time.Time) error {
	for _, e := range r.events {
		if e.ID == id {
			e.Status = status
			e.ResolvedAt = at
			r.resolved++
		}
	}
	return nil
}

func TestStateTicketEvaluatorLifecycle(t *testing.T) {
	r := &stateAlertRepoStub{rules: []*OpsAlertRule{
		{ID: 1, Enabled: true, MetricType: StateTicketUnavailableMetric, Operator: ">", Threshold: 0, SustainedMinutes: 2, CooldownMinutes: 30},
		{ID: 2, Enabled: true, MetricType: StateTicketRenewalFailedMetric, Operator: ">", Threshold: 0, SustainedMinutes: 2},
	}}
	source := stateTestSource(t, stateTestTicket("expired", 0))
	newService := func() *OpsAlertEvaluatorService {
		s := NewOpsAlertEvaluatorService(nil, r, nil, nil, nil, nil)
		s.stateTicketSource = source
		return s
	}
	s := newService()
	s.evaluateOnce(time.Minute)
	require.Empty(t, r.events)
	s.evaluateOnce(time.Minute)
	require.Len(t, r.events, 1)
	require.Equal(t, 2, source.lists)
	require.Equal(t, 2, source.reads)
	s.evaluateOnce(time.Minute)
	require.Len(t, r.events, 1)
	// Restart loses the in-memory consecutive counter, but must preserve firing.
	s = newService()
	s.evaluateOnce(time.Minute)
	require.Zero(t, r.resolved)
	source.err = errors.New("offline")
	s.evaluateOnce(time.Minute)
	require.Zero(t, r.resolved)
	source.err = nil
	source.status = stateTestSource(t, stateTestTicket("ready", 1800)).status
	s.evaluateOnce(time.Minute)
	require.Equal(t, 1, r.resolved)
	source.status = stateTestSource(t, stateTestTicket("expired", 0)).status
	s.evaluateOnce(time.Minute)
	s.evaluateOnce(time.Minute)
	require.Len(t, r.events, 1, "cooldown suppresses a repeated incident")
	// Renewal failures are separate from expiry and recover after successful renewal.
	ticket := stateTestTicket("renewing", 500)
	ticket.LastError = "upstream_forbidden"
	source.status = stateTestSource(t, ticket).status
	s.evaluateOnce(time.Minute)
	s.evaluateOnce(time.Minute)
	require.Len(t, r.events, 2)
	require.Equal(t, int64(2), r.events[1].RuleID)
	source.status = stateTestSource(t, stateTestTicket("ready", 1800)).status
	s.evaluateOnce(time.Minute)
	require.Equal(t, 2, r.resolved)
}
