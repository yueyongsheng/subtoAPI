package service

import (
	"context"
	"errors"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

type qualityMemoryRepo struct {
	accounts []OAuthQualityAccount
	group    *int64
	types    []string
}

func (r *qualityMemoryRepo) ListOAuthQualityAccounts(_ context.Context, group *int64, types []string) ([]OAuthQualityAccount, error) {
	r.group, r.types = group, types
	result := []OAuthQualityAccount{}
	for _, a := range r.accounts {
		matchType, matchGroup := false, group == nil || (*group == -1 && len(a.GroupIDs) == 0)
		for _, kind := range types {
			if a.Type == kind {
				matchType = true
			}
		}
		for _, id := range a.GroupIDs {
			if group != nil && id == *group {
				matchGroup = true
			}
		}
		if matchType && matchGroup {
			result = append(result, a)
		}
	}
	return result, nil
}

type qualityRunnerCall struct {
	accountID       int64
	modelID, prompt string
}
type qualityMemoryRunner struct {
	calls  []qualityRunnerCall
	output map[string]string
	err    error
}

func (r *qualityMemoryRunner) RunTestBackgroundWithPrompt(_ context.Context, accountID int64, modelID, prompt string) (*ScheduledTestResult, error) {
	r.calls = append(r.calls, qualityRunnerCall{accountID, modelID, prompt})
	if r.err != nil {
		return nil, r.err
	}
	return &ScheduledTestResult{Status: "success", ResponseText: r.output[prompt], LatencyMs: 12}, nil
}
func qualityFixtureRunner() *qualityMemoryRunner {
	r := &qualityMemoryRunner{output: map[string]string{}}
	for _, probe := range oauthQualityProbes {
		switch probe.Key {
		case "reasoning_exact":
			r.output[probe.Prompt] = "21"
		case "structured_output":
			r.output[probe.Prompt] = `{"answer":42,"ok":true}`
		case "svg_html":
			r.output[probe.Prompt] = `<html><svg><text>pelican bicycle</text><animate attributeName="x" /></svg></html>`
		case "english_knowledge":
			r.output[probe.Prompt] = "yes"
		}
	}
	return r
}
func qualityTestService() (*OAuthQualityService, *qualityMemoryRunner) {
	repo := &qualityMemoryRepo{accounts: []OAuthQualityAccount{
		{ID: 1, Name: "oauth", Type: "oauth", GroupIDs: []int64{7}},
		{ID: 2, Name: "key", Type: "apikey", GroupIDs: []int64{7}},
		{ID: 3, Name: "other group", Type: "oauth", GroupIDs: []int64{8}},
		{ID: 4, Name: "ungrouped", Type: "apikey", GroupIDs: []int64{}},
	}}
	runner := qualityFixtureRunner()
	return &OAuthQualityService{repo: repo, runner: runner, now: time.Now}, runner
}
func TestScoreOAuthQualityProbe(t *testing.T) {
	tests := []struct{ key, output, status string }{
		{"reasoning_exact", "21", "passed"}, {"reasoning_exact", "29", "review"}, {"reasoning_exact", "21 or 22", "review"},
		{"structured_output", `{"answer":42,"ok":true}`, "passed"},
		{"structured_output", `{"answer":42,"ok":true,"extra":1}`, "review"},
		{"structured_output", "\x60\x60\x60json\n{\"answer\":42,\"ok\":true}\n\x60\x60\x60", "review"},
		{"svg_html", "<svg>pelican bicycle <animate /></svg>", "review"},
		{"english_knowledge", "YES", "passed"}, {"english_knowledge", "no", "review"},
		{"english_knowledge", "yes, I know him", "review"}, {"custom", "", "failed"},
	}
	for _, test := range tests {
		status, _ := scoreOAuthQualityProbe(test.key, test.output)
		require.Equal(t, test.status, status, test.output)
	}
}
func TestQualityBunWorstCase(t *testing.T) {
	// Exhaustively enumerate samples with no desired pair, keeping shape counts.
	var unsafe [25][18]bool
	for ra := 0; ra <= 7; ra++ {
		for rp := 0; rp <= 9; rp++ {
			for sa := 0; sa <= 7; sa++ {
				for sp := 0; sp <= 6; sp++ {
					if (ra > 0 && sp > 0) || (rp > 0 && sa > 0) {
						continue
					}
					for rw := 0; rw <= 8; rw++ {
						for sw := 0; sw <= 4; sw++ {
							unsafe[ra+rp+rw][sa+sp+sw] = true
						}
					}
				}
			}
		}
	}
	best := 42
	for round := 0; round <= 24; round++ {
		for star := 0; star <= 17; star++ {
			if !unsafe[round][star] && round+star < best {
				best = round + star
			}
		}
	}
	require.Equal(t, 21, best)
	require.False(t, unsafe[9][12])
}

func TestOAuthQualityServiceUsesSelectedMethodsAndBothTypes(t *testing.T) {
	svc, runner := qualityTestService()
	group := int64(7)
	report, err := svc.Run(context.Background(), &group, []int64{1, 1, 2}, "gpt-6-astra", []string{"oauth", "apikey"}, []string{"english_knowledge", "structured_output", "english_knowledge"}, nil)
	require.NoError(t, err)
	require.Len(t, report.Accounts, 2)
	require.Len(t, runner.calls, 4)
	require.Equal(t, []string{"english_knowledge", "structured_output"}, report.ProbeKeys)
	for _, account := range report.Accounts {
		require.Equal(t, 2, account.Total)
		require.Equal(t, 2, account.Passed)
	}
	for _, call := range runner.calls {
		require.Equal(t, "gpt-6-astra", call.modelID)
		require.NotContains(t, call.prompt, "鹈鹕")
	}
}
func TestOAuthQualityRejectsOutOfScopeAndInvalidOptionsBeforeCallingModel(t *testing.T) {
	svc, runner := qualityTestService()
	group := int64(7)
	for _, ids := range [][]int64{{3}, {4}, {99}, {0}, {-1}, nil} {
		_, err := svc.Run(context.Background(), &group, ids, "gpt-6-astra", []string{"oauth", "apikey"}, []string{"english_knowledge"}, nil)
		require.Error(t, err)
	}
	_, err := svc.Run(context.Background(), &group, []int64{2}, "gpt-6-astra", []string{"oauth"}, []string{"english_knowledge"}, nil)
	require.Error(t, err)
	for _, types := range [][]string{nil, {"invalid"}} {
		_, err = svc.Run(context.Background(), nil, []int64{1}, "gpt-6-astra", types, []string{"english_knowledge"}, nil)
		require.Error(t, err)
	}
	for _, keys := range [][]string{nil, {"invalid"}, {"custom"}} {
		_, err = svc.Run(context.Background(), nil, []int64{1}, "gpt-6-astra", []string{"oauth"}, keys, nil)
		require.Error(t, err)
	}
	_, err = svc.Run(context.Background(), nil, []int64{1}, " ", []string{"oauth"}, []string{"english_knowledge"}, nil)
	require.Error(t, err)
	svc.mu.Lock()
	_, err = svc.Run(context.Background(), nil, []int64{1}, "gpt-6-astra", []string{"oauth"}, []string{"english_knowledge"}, nil)
	svc.mu.Unlock()
	require.Error(t, err)
	require.Empty(t, runner.calls)
}
func TestOAuthQualityCustomQuestion(t *testing.T) {
	svc, runner := qualityTestService()
	custom := &OAuthQualityCustomProbe{Prompt: "Return the number of days in a week.", Expected: "7"}
	runner.output[custom.Prompt] = "7"
	report, err := svc.Run(context.Background(), nil, []int64{2}, "gpt-6-astra", []string{"apikey"}, []string{"custom"}, custom)
	require.NoError(t, err)
	require.Equal(t, "passed", report.Accounts[0].Status)
	require.Equal(t, custom, report.CustomProbe)
	require.Len(t, runner.calls, 1)
	require.Equal(t, custom.Prompt, runner.calls[0].prompt)
	custom.Expected = ""
	report, err = svc.Run(context.Background(), nil, []int64{2}, "gpt-6-astra", []string{"apikey"}, []string{"custom"}, custom)
	require.NoError(t, err)
	require.Equal(t, "review", report.Accounts[0].Status)
	custom.Expected = "8"
	report, err = svc.Run(context.Background(), nil, []int64{2}, "gpt-6-astra", []string{"apikey"}, []string{"custom"}, custom)
	require.NoError(t, err)
	require.Equal(t, "review", report.Accounts[0].Status)
	for _, invalid := range []*OAuthQualityCustomProbe{{Prompt: " "}, {Prompt: strings.Repeat("题", 8001)}, {Prompt: "hi", Expected: strings.Repeat("答", 2001)}} {
		_, err = svc.Run(context.Background(), nil, []int64{2}, "gpt-6-astra", []string{"apikey"}, []string{"custom"}, invalid)
		require.Error(t, err)
	}
}
func TestOAuthQualityUngroupedAndFailures(t *testing.T) {
	svc, runner := qualityTestService()
	group := int64(-1)
	list, err := svc.List(context.Background(), &group, []string{"apikey"})
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, int64(4), list[0].ID)
	runner.err = errors.New("upstream error containing private details")
	report, err := svc.Run(context.Background(), &group, []int64{4}, "gpt-6-astra", []string{"apikey"}, []string{"english_knowledge"}, nil)
	require.NoError(t, err)
	require.Equal(t, "failed", report.Accounts[0].Status)
	require.NotContains(t, report.Accounts[0].Probes[0].Summary, "private details")
}
func TestPreviewQualityOutputKeepsUTF8Boundaries(t *testing.T) {
	value := strings.Repeat("鹈", oauthQualityPreviewBytes+10)
	require.Equal(t, oauthQualityPreviewBytes+1, len([]rune(previewQualityOutput(value))))
}
