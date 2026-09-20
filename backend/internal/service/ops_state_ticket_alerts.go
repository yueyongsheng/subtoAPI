package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	pluginv1 "github.com/Wei-Shaw/sub2api/pkg/pluginapi/v1"
)

const (
	StateTicketRenewalFailedMetric = "state_ticket_renewal_failed_count"
	StateTicketUnavailableMetric   = "state_ticket_unavailable_count"
	stateKitPluginKey              = "io.github.wangyunjeff.sub2api-state-kit"
)

type stateTicketStatusSource interface {
	List(context.Context) ([]*PluginInstallation, error)
	Status(context.Context, int64) (*pluginv1.HealthResponse, error)
}

type stateAlertTicket struct {
	AccountID int64  `json:"account_id"`
	Model     string `json:"model"`
	State     string `json:"state"`
	Remaining int64  `json:"remaining_seconds"`
	ExpiresAt string `json:"expires_at"`
	LastError string `json:"last_error"`
}

type stateAlertSnapshot struct {
	HostReady bool               `json:"host_ready"`
	Tickets   []stateAlertTicket `json:"tickets"`
	valid     bool
}

var stateAlertModel = regexp.MustCompile(`^gpt-[A-Za-z0-9][A-Za-z0-9._-]{0,94}$`)

func isStateTicketMetric(metric string) bool {
	return metric == StateTicketRenewalFailedMetric || metric == StateTicketUnavailableMetric
}

// Read once per evaluator cycle, only when an enabled rule needs it. Health is
// read-only: it neither probes models nor obtains plugin configuration/STATE.
func loadStateAlertSnapshot(ctx context.Context, source stateTicketStatusSource, rules []*OpsAlertRule) stateAlertSnapshot {
	needed := false
	for _, rule := range rules {
		if rule != nil && rule.Enabled && isStateTicketMetric(rule.MetricType) {
			needed = true
			break
		}
	}
	if !needed || source == nil {
		return stateAlertSnapshot{}
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	installed, err := source.List(ctx)
	if err != nil {
		return stateAlertSnapshot{}
	}
	for _, plugin := range installed {
		if plugin == nil || plugin.PluginKey != stateKitPluginKey {
			continue
		}
		if plugin.State == PluginStateDisabled {
			return stateAlertSnapshot{valid: true}
		}
		if plugin.State != PluginStateEnabled {
			return stateAlertSnapshot{}
		}
		status, err := source.Status(ctx, plugin.ID)
		if err != nil || status == nil || !status.Healthy {
			return stateAlertSnapshot{}
		}
		return parseStateAlertSnapshot(status.StatusJson)
	}
	// Uninstalled or deliberately disabled: stop monitoring its tickets.
	return stateAlertSnapshot{valid: true}
}

func parseStateAlertSnapshot(raw string) stateAlertSnapshot {
	var snapshot stateAlertSnapshot
	if len(raw) > 2*1024*1024 || json.Unmarshal([]byte(raw), &snapshot) != nil || !snapshot.HostReady || snapshot.Tickets == nil || len(snapshot.Tickets) > 1024 {
		return stateAlertSnapshot{}
	}
	var fields struct {
		Tickets []map[string]json.RawMessage `json:"tickets"`
	}
	if json.Unmarshal([]byte(raw), &fields) != nil {
		return stateAlertSnapshot{}
	}
	for _, ticket := range fields.Tickets {
		for _, key := range []string{"account_id", "model", "state", "remaining_seconds", "expires_at", "last_error"} {
			v := strings.TrimSpace(string(ticket[key]))
			if v == "" || v == "null" {
				return stateAlertSnapshot{}
			}
		}
	}
	seen := make(map[string]bool, len(snapshot.Tickets))
	for i := range snapshot.Tickets {
		t := &snapshot.Tickets[i]
		key := fmt.Sprintf("%d/%s", t.AccountID, t.Model)
		if t.AccountID <= 0 || !stateAlertModel.MatchString(t.Model) || t.Remaining < 0 || t.Remaining > 3600 || seen[key] {
			return stateAlertSnapshot{}
		}
		switch t.State {
		case "disabled", "waiting_host", "waiting_account", "renewing", "harvesting", "ready", "cooldown", "expired", "queued":
		default:
			return stateAlertSnapshot{}
		}
		if (t.State == "ready" || t.State == "renewing") && t.Remaining > 0 && t.ExpiresAt == "" {
			return stateAlertSnapshot{}
		}
		if t.ExpiresAt != "" {
			if _, err := time.Parse(time.RFC3339, t.ExpiresAt); err != nil {
				return stateAlertSnapshot{}
			}
		}
		seen[key] = true
		// Unknown plugin strings never reach event descriptions or email.
		t.LastError = stateAlertError(t.LastError)
	}
	snapshot.valid = true
	return snapshot
}

func stateAlertError(raw string) string {
	switch raw {
	case "", "upstream_unauthorized", "upstream_forbidden", "upstream_rate_limited", "model_mismatch", "state_312", "harvest_failed", "fixed_proxy_validation_failed", "unexpected_state_length", "identity_unavailable", "identity_changed", "invalid_dynamic_proxy", "managed_proxy_unavailable", "ticket_persistence_failed", "attempts_exhausted":
		return raw
	default:
		return "unknown"
	}
}

func (s *OpsAlertEvaluatorService) stateTicketMetric(ctx context.Context, snapshot stateAlertSnapshot, rule *OpsAlertRule, now time.Time, platform string, groupID *int64) (float64, string, bool) {
	if !snapshot.valid {
		return 0, "", false // Unknown health must not resolve an existing incident.
	}
	if platform != "" && platform != "openai" {
		return 0, "", true
	}
	var groupAccounts map[int64]*AccountAvailability
	if groupID != nil {
		if s.opsService == nil {
			return 0, "", false
		}
		availability, err := s.opsService.GetAccountAvailability(ctx, "openai", groupID)
		if err != nil || availability == nil {
			return 0, "", false
		}
		groupAccounts = availability.Accounts
	}
	var affected []stateAlertTicket
	for _, ticket := range snapshot.Tickets {
		if groupID != nil && groupAccounts[ticket.AccountID] == nil {
			continue
		}
		if ticket.State == "disabled" || ticket.State == "waiting_host" || ticket.State == "waiting_account" {
			continue
		}
		usable := (ticket.State == "ready" || ticket.State == "renewing") && ticket.Remaining > 0
		if ticket.ExpiresAt != "" {
			expires, _ := time.Parse(time.RFC3339, ticket.ExpiresAt)
			usable = usable && expires.After(now)
		}
		breached := !usable
		if rule.MetricType == StateTicketRenewalFailedMetric {
			breached = usable && ticket.LastError != "" && ticket.Remaining <= 600
		}
		if breached {
			affected = append(affected, ticket)
		}
	}
	sort.Slice(affected, func(i, j int) bool {
		if affected[i].AccountID != affected[j].AccountID {
			return affected[i].AccountID < affected[j].AccountID
		}
		return affected[i].Model < affected[j].Model
	})
	label := "STATE 票据过期或暂无有效票据"
	if rule.MetricType == StateTicketRenewalFailedMetric {
		label = "STATE 续期失败，旧票据剩余不超过 10 分钟"
	}
	lines := []string{fmt.Sprintf("%s：%d 个账号/模型。当前状态快照，非模型探测。", label, len(affected))}
	for i, ticket := range affected {
		if i == 20 {
			lines = append(lines, "更多账号请查看账号管理的 STATE 状态。")
			break
		}
		lines = append(lines, fmt.Sprintf("账号 #%d / %s；状态=%s；剩余=%ds；原因=%s", ticket.AccountID, ticket.Model, ticket.State, ticket.Remaining, ticket.LastError))
	}
	return float64(len(affected)), strings.Join(lines, "\n"), true
}
