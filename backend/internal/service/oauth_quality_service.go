package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"
)

const (
	oauthQualityMaxAccounts  = 50
	oauthQualityProbeTimeout = 90 * time.Second
	oauthQualityPreviewBytes = 32000
)

// OAuthQualityRunner is the account-test transport reused by the quality
// probes. Keeping this interface small makes the scoring layer independent of
// HTTP details and keeps the tests deterministic.
type OAuthQualityRunner interface {
	RunTestBackgroundWithPrompt(context.Context, int64, string, string) (*ScheduledTestResult, error)
}

type OAuthQualityRepository interface {
	ListOAuthQualityAccounts(context.Context, *int64, []string) ([]OAuthQualityAccount, error)
}

var ErrInvalidQualitySelection = errors.New("invalid quality selection")

func ValidQualityTypes(types []string) bool {
	if len(types) == 0 || len(types) > 4 {
		return false
	}
	for _, kind := range types {
		switch kind {
		case "oauth", "apikey", "setup-token", "bedrock":
		default:
			return false
		}
	}
	return true
}

type OAuthQualityCustomProbe struct {
	Prompt   string `json:"prompt"`
	Expected string `json:"expected"`
}

func selectQualityProbes(keys []string, custom *OAuthQualityCustomProbe) ([]OAuthQualityProbe, error) {
	if len(keys) == 0 || len(keys) > len(oauthQualityProbes)+1 {
		return nil, ErrInvalidQualitySelection
	}
	selected := make([]OAuthQualityProbe, 0, len(keys))
	seen := map[string]bool{}
	for _, key := range keys {
		if key == "custom" {
			if custom == nil || strings.TrimSpace(custom.Prompt) == "" || len([]rune(custom.Prompt)) > 8000 || len([]rune(custom.Expected)) > 2000 {
				return nil, ErrInvalidQualitySelection
			}
			if !seen[key] {
				selected = append(selected, OAuthQualityProbe{Key: key, Label: "自定义问题", Prompt: strings.TrimSpace(custom.Prompt)})
				seen[key] = true
			}
			continue
		}
		found := false
		for _, probe := range oauthQualityProbes {
			if key == probe.Key {
				found = true
				if !seen[key] {
					selected = append(selected, probe)
					seen[key] = true
				}
				break
			}
		}
		if !found {
			return nil, ErrInvalidQualitySelection
		}
	}
	return selected, nil
}

type OAuthQualityProbe struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Prompt string `json:"-"`
}

type OAuthQualityProbeResult struct {
	Key           string `json:"key"`
	Label         string `json:"label"`
	Status        string `json:"status"`
	Summary       string `json:"summary"`
	OutputPreview string `json:"output_preview,omitempty"`
	LatencyMS     int64  `json:"latency_ms,omitempty"`
}

type OAuthQualityAccount struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Platform    string  `json:"platform"`
	Type        string  `json:"type"`
	GroupIDs    []int64 `json:"group_ids"`
	Status      string  `json:"status"`
	Schedulable bool    `json:"schedulable"`
}

type OAuthQualityAccountResult struct {
	OAuthQualityAccount
	Status  string                    `json:"status"`
	Summary string                    `json:"summary"`
	Passed  int                       `json:"passed"`
	Total   int                       `json:"total"`
	Probes  []OAuthQualityProbeResult `json:"probes"`
}

type OAuthQualityReport struct {
	CheckedAt    time.Time                   `json:"checked_at"`
	GroupID      *int64                      `json:"group_id"`
	ModelID      string                      `json:"model_id"`
	AccountTypes []string                    `json:"account_types"`
	ProbeKeys    []string                    `json:"probe_keys"`
	CustomProbe  *OAuthQualityCustomProbe    `json:"custom_probe,omitempty"`
	Accounts     []OAuthQualityAccountResult `json:"accounts"`
}

type OAuthQualityService struct {
	mu     sync.Mutex
	repo   OAuthQualityRepository
	runner OAuthQualityRunner
	now    func() time.Time
}

func NewOAuthQualityService(repo OAuthQualityRepository, runner OAuthQualityRunner, _ *OpsService) *OAuthQualityService {
	return &OAuthQualityService{repo: repo, runner: runner, now: time.Now}
}

var oauthQualityProbes = []OAuthQualityProbe{
	{
		Key:    "reasoning_exact",
		Label:  "馒头推理题",
		Prompt: "黑色袋子里有苹果、桃子、西瓜三种口味的馒头，每种有圆形和五角星形，形状能靠手感分辨，可以按形状选择取出的数量，但不能靠手感分辨口味。数量表（列依次为苹果、桃子、西瓜）：圆形 7、9、8；五角星形 7、6、4。活动前决定摸出的总数和两种形状各自的数量。最少取出多少个才能保证同时拥有圆形苹果与五角星桃子，或者圆形桃子与五角星苹果？不要调用工具、联网或读取文件。只输出最终整数，不要解释。",
	},
	{
		Key:    "structured_output",
		Label:  "结构化指令",
		Prompt: "只输出一个严格 JSON 对象，不要 Markdown，不要解释，格式必须是：{\"answer\":42,\"ok\":true}。",
	},
	{
		Key:    "svg_html",
		Label:  "鹈鹕骑自行车",
		Prompt: "创建一个 HTML，内容是用 SVG 绘制一个鹈鹕骑自行车的 2D 动画。不要进行任何测试，不要调用 skills，不要网络检索，不要调用子智能体，直接生成完整 HTML 源码。",
	},
	{
		Key:    "english_knowledge",
		Label:  "英文人物问答",
		Prompt: "don't search the internet, do you know Thibault Sottiaux on X. answer yes or no",
	},
}

func OAuthQualityProbes() []OAuthQualityProbe {
	result := make([]OAuthQualityProbe, len(oauthQualityProbes))
	copy(result, oauthQualityProbes)
	for i := range result {
		result[i].Prompt = ""
	}
	return result
}

func (s *OAuthQualityService) List(ctx context.Context, groupID *int64, types []string) ([]OAuthQualityAccount, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("OAuth quality service unavailable")
	}
	if !ValidQualityTypes(types) || (groupID != nil && *groupID != -1 && *groupID <= 0) {
		return nil, ErrInvalidQualitySelection
	}
	return s.repo.ListOAuthQualityAccounts(ctx, groupID, types)
}

func (s *OAuthQualityService) Run(ctx context.Context, groupID *int64, ids []int64, modelID string, types, probeKeys []string, custom *OAuthQualityCustomProbe) (*OAuthQualityReport, error) {
	if s == nil || s.repo == nil || s.runner == nil {
		return nil, errors.New("OAuth quality service unavailable")
	}
	if !s.mu.TryLock() {
		return nil, errors.New("OAuth quality operation in progress")
	}
	defer s.mu.Unlock()
	modelID = strings.TrimSpace(modelID)
	if modelID == "" || len(modelID) > 200 {
		return nil, ErrInvalidQualitySelection
	}
	probes, err := selectQualityProbes(probeKeys, custom)
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		if id <= 0 {
			return nil, ErrInvalidQualitySelection
		}
	}
	ids = uniquePositiveIDs(ids)
	if len(ids) == 0 || len(ids) > oauthQualityMaxAccounts {
		return nil, ErrInvalidQualitySelection
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()
	accounts, err := s.List(ctx, groupID, types)
	if err != nil {
		return nil, err
	}
	byID := make(map[int64]OAuthQualityAccount, len(accounts))
	for _, account := range accounts {
		byID[account.ID] = account
	}
	selected := make([]OAuthQualityAccount, 0, len(ids))
	for _, id := range ids {
		account, ok := byID[id]
		if !ok {
			return nil, ErrInvalidQualitySelection
		}
		selected = append(selected, account)
	}

	report := &OAuthQualityReport{CheckedAt: s.now().UTC(), GroupID: groupID, ModelID: modelID, Accounts: make([]OAuthQualityAccountResult, 0, len(selected))}
	report.AccountTypes = append([]string{}, types...)
	for _, probe := range probes {
		report.ProbeKeys = append(report.ProbeKeys, probe.Key)
	}
	for _, probe := range probes {
		if probe.Key == "custom" {
			report.CustomProbe = custom
		}
	}
	for _, account := range selected {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		result := OAuthQualityAccountResult{OAuthQualityAccount: account, Status: "failed", Total: len(probes), Probes: make([]OAuthQualityProbeResult, 0, len(probes))}
		for _, probe := range probes {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			probeCtx, probeCancel := context.WithTimeout(ctx, oauthQualityProbeTimeout)
			test, runErr := s.runner.RunTestBackgroundWithPrompt(probeCtx, account.ID, modelID, probe.Prompt)
			probeCancel()
			item := OAuthQualityProbeResult{Key: probe.Key, Label: probe.Label}
			if runErr != nil || test == nil || test.Status != "success" {
				item.Status = "failed"
				item.Summary = "传输或上游请求失败"
				if test != nil {
					item.LatencyMS, item.OutputPreview = test.LatencyMs, previewQualityOutput(test.ResponseText)
				}
				result.Probes = append(result.Probes, item)
				continue
			}
			item.LatencyMS = test.LatencyMs
			item.OutputPreview = previewQualityOutput(test.ResponseText)
			item.Status, item.Summary = scoreOAuthQualityProbe(probe.Key, test.ResponseText)
			if probe.Key == "custom" && strings.TrimSpace(test.ResponseText) != "" {
				switch {
				case strings.TrimSpace(custom.Expected) == "":
					item.Status, item.Summary = "review", "未设置预期答案，请人工对比输出"
				case strings.TrimSpace(test.ResponseText) == strings.TrimSpace(custom.Expected):
					item.Status, item.Summary = "passed", "与预期答案一致（忽略首尾空白）"
				default:
					item.Status, item.Summary = "review", "与预期答案不一致，请人工复核"
				}
			}
			result.Probes = append(result.Probes, item)
		}
		result.Passed = 0
		for _, probe := range result.Probes {
			if probe.Status == "passed" {
				result.Passed++
			}
		}
		switch {
		case allQualityProbesFailed(result.Probes):
			result.Status, result.Summary = "failed", "所选检测均未获得有效输出，请检查请求状态后重试"
		case result.Passed == result.Total:
			result.Status, result.Summary = "passed", "所选检测符合规则；单次结果仅供参考"
		case anyQualityProbeFailed(result.Probes):
			result.Status, result.Summary = "review", "存在失败或不完整的探针，建议人工复核"
		default:
			result.Status, result.Summary = "review", "部分探针通过，建议人工复核"
		}
		report.Accounts = append(report.Accounts, result)
	}
	return report, nil
}

func uniquePositiveIDs(ids []int64) []int64 {
	seen := make(map[int64]bool, len(ids))
	result := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id > 0 && !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}
	return result
}

func anyQualityProbeFailed(probes []OAuthQualityProbeResult) bool {
	for _, probe := range probes {
		if probe.Status == "failed" {
			return true
		}
	}
	return false
}

func allQualityProbesFailed(probes []OAuthQualityProbeResult) bool {
	for _, probe := range probes {
		if probe.Status != "failed" {
			return false
		}
	}
	return len(probes) > 0
}

func scoreOAuthQualityProbe(key, output string) (string, string) {
	text := strings.TrimSpace(output)
	if text == "" {
		return "failed", "没有收到可评分的文本输出"
	}
	switch key {
	case "reasoning_exact":
		if text == "21" {
			return "passed", "最终答案 21 正确：可取 9 个圆形和 12 个五角星形"
		}
		return "review", "按形状选择数量时预期仅输出 21，请复核答案与格式"
	case "english_knowledge":
		if strings.ToLower(text) == "yes" {
			return "passed", "与参考回答 yes 一致；不代表账号能力鉴定"
		}
		return "review", "与参考回答 yes 不一致；人物知识和单次回答不构成降智证据"
	case "structured_output":
		var value map[string]any
		if json.Unmarshal([]byte(text), &value) == nil && len(value) == 2 && value["answer"] == float64(42) && value["ok"] == true {
			return "passed", "JSON 字段和类型符合预期"
		}
		return "review", "JSON 结构或字段值不符合预期"
	case "svg_html":
		lower := strings.ToLower(text)
		checks := []bool{
			strings.Contains(lower, "<svg"),
			strings.Contains(lower, "</svg>"),
			strings.Contains(lower, "pelican") || strings.Contains(text, "鹈鹕"),
			strings.Contains(lower, "bicycle") || strings.Contains(lower, "bike") || strings.Contains(text, "自行车"),
			strings.Contains(lower, "animate") || strings.Contains(lower, "animation") || strings.Contains(lower, "@keyframes"),
		}
		for _, ok := range checks {
			if !ok {
				return "review", "HTML/SVG 缺少主题元素或动画定义"
			}
		}
		return "review", "检测到 SVG、主题元素和动画定义；画面质量仍需人工复核"
	default:
		return "review", "未知探针版本"
	}
}

func previewQualityOutput(value string) string {
	value = strings.TrimSpace(value)
	if len([]rune(value)) <= oauthQualityPreviewBytes {
		return value
	}
	runes := []rune(value)
	return string(runes[:oauthQualityPreviewBytes]) + "…"
}
