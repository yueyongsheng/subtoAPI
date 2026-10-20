//go:build unit

package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// 同时覆盖远程目录更新、静态兜底、旧计费入口和分组入口，校验真实扣款与账号成本。
func TestOpenAIGatewayServiceRecordUsage_GPT61SolSubscriptionLongContext(t *testing.T) {
	for _, catalog := range []bool{false, true} {
		for _, groupMode := range []string{"无分组", "分组关闭", "分组开启"} {
			for _, accountEnabled := range []bool{false, true} {
				for _, accountType := range []string{AccountTypeOAuth, AccountTypeSetupToken, AccountTypeAPIKey} {
					for _, tier := range []string{"", "priority"} {
						name := fmt.Sprintf("目录=%t/%s/账号开关=%t/%s/%s", catalog, groupMode, accountEnabled, accountType, tier)
						t.Run(name, func(t *testing.T) {
							usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
							userRepo := &openAIRecordUsageUserRepoStub{}
							svc := newOpenAIRecordUsageServiceForTest(usageRepo, userRepo, &openAIRecordUsageSubRepoStub{}, nil)
							if catalog {
								svc.billingService = NewBillingService(svc.cfg, newStubPricingServiceFromJSON(t, `{
									"gpt-6.1-sol": {"litellm_provider": "openai", "mode": "chat",
									"input_cost_per_token": 2e-6, "output_cost_per_token": 10e-6,
									"cache_read_input_token_cost": 0.1e-6, "cache_creation_input_token_cost": 2.5e-6,
									"input_cost_per_token_above_272k_tokens": 4e-6,
									"output_cost_per_token_above_272k_tokens": 15e-6}
								}`))
							}
							apiKey := &APIKey{ID: 1}
							if groupMode != "无分组" {
								apiKey = openAIRecordUsageAPIKeyWithGroup(svc, 1, groupMode == "分组开启")
								apiKey.GroupID = i64p(1)
								apiKey.Group.RateMultiplier = 1.1
								svc.channelService = newTestChannelServiceForStats(t, &Channel{ID: 1, Status: StatusActive}, 1, PlatformOpenAI)
							}
							account := &Account{ID: 3, Platform: PlatformOpenAI, Type: accountType,
								Extra: map[string]any{openAILongContextBillingEnabledKey: accountEnabled}}
							err := svc.RecordUsage(context.Background(), &OpenAIRecordUsageInput{
								Result: &OpenAIForwardResult{
									RequestID: name, Model: "gpt-6.1-sol", ServiceTier: &tier, Duration: time.Second,
									// 普通输入只有 200k，由缓存使总上下文越过 272k。
									Usage: OpenAIUsage{InputTokens: 300000, OutputTokens: 2000, CacheReadInputTokens: 80000, CacheCreationInputTokens: 20000},
								},
								APIKey: apiKey, User: &User{ID: 2}, Account: account,
							})
							require.NoError(t, err)
							wantLong := accountEnabled || groupMode == "分组开启"
							inputFactor, outputFactor, tierFactor := 1.0, 1.0, 1.0
							if wantLong {
								inputFactor, outputFactor = 2, 1.5
							}
							if tier == "priority" {
								tierFactor = 2
							}
							log := usageRepo.lastLog
							require.NotNil(t, log)
							require.Equal(t, wantLong, log.LongContextBillingApplied)
							require.InDelta(t, 200000*2e-6*inputFactor*tierFactor*3.5, log.InputCost, 1e-12)
							require.InDelta(t, 80000*0.1e-6*inputFactor*tierFactor*3.5, log.CacheReadCost, 1e-12)
							require.InDelta(t, 20000*2.5e-6*inputFactor*tierFactor*3.5, log.CacheCreationCost, 1e-12)
							require.InDelta(t, 2000*10e-6*outputFactor*tierFactor*3.5, log.OutputCost, 1e-12)
							wantTotal := (0.458*inputFactor + 0.02*outputFactor) * tierFactor * 3.5
							require.InDelta(t, wantTotal, log.TotalCost, 1e-12)
							require.InDelta(t, wantTotal*1.1, log.ActualCost, 1e-12)
							require.Equal(t, 1, userRepo.deductCalls)
							require.InDelta(t, wantTotal*1.1, userRepo.lastAmount, 1e-12)
							if groupMode != "无分组" {
								wantStats := 0.478 * tierFactor * 3.5
								if accountEnabled {
									wantStats = (0.458*2 + 0.02*1.5) * tierFactor * 3.5
								}
								require.NotNil(t, log.AccountStatsCost)
								require.InDelta(t, wantStats, *log.AccountStatsCost, 1e-12)
							}
						})
					}
				}
			}
		}
	}
}

func TestOpenAIGatewayServiceRecordUsage_GPT61SolUsesCredentialAndUpstreamModel(t *testing.T) {
	for _, tt := range []struct {
		name       string
		model      string
		upstream   string
		parentType string
		wantTotal  float64
		wantLong   bool
	}{
		{name: "客户别名映射到目标", model: "custom", upstream: "gpt-6.1-sol", parentType: AccountTypeOAuth, wantTotal: 1.23, wantLong: true},
		{name: "请求目标映射到旧模型", model: "gpt-6.1-sol", upstream: "gpt-6-sol", parentType: AccountTypeOAuth, wantTotal: 1.23, wantLong: true},
		{name: "其他订阅模型", model: "gpt-6-sol", parentType: AccountTypeOAuth, wantTotal: 1.23, wantLong: true},
		{name: "订阅母账号", model: "gpt-6.1-sol", parentType: AccountTypeOAuth, wantTotal: 1.23, wantLong: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
			svc := newOpenAIRecordUsageServiceForTest(usageRepo, &openAIRecordUsageUserRepoStub{}, &openAIRecordUsageSubRepoStub{}, nil)
			parentID := int64(4)
			svc.accountRepo = &openAIRecordUsageAccountRepoStub{account: &Account{
				ID: parentID, Platform: PlatformOpenAI, Type: tt.parentType,
				Extra: map[string]any{openAILongContextBillingEnabledKey: true},
			}}
			err := svc.RecordUsage(context.Background(), &OpenAIRecordUsageInput{
				Result: &OpenAIForwardResult{RequestID: tt.name, Model: tt.model, UpstreamModel: tt.upstream,
					Usage: OpenAIUsage{InputTokens: 300000, OutputTokens: 2000}, Duration: time.Second},
				APIKey: openAIRecordUsageAPIKeyWithGroup(svc, 1, true), User: &User{ID: 2},
				Account: &Account{ID: 3, Platform: PlatformOpenAI, Type: AccountTypeOAuth, ParentAccountID: &parentID, QuotaDimension: QuotaDimensionSpark},
			})
			require.NoError(t, err)
			require.Equal(t, tt.wantLong, usageRepo.lastLog.LongContextBillingApplied)
			require.InDelta(t, tt.wantTotal*3.5, usageRepo.lastLog.TotalCost, 1e-12)
		})
	}
}

func TestOpenAIGatewayServiceRecordUsage_GPT61SolLongContextBoundary(t *testing.T) {
	for _, totalInput := range []int{271999, 272000, 272001} {
		for _, accountType := range []string{AccountTypeOAuth, AccountTypeAPIKey} {
			t.Run(fmt.Sprintf("%s/%d", accountType, totalInput), func(t *testing.T) {
				usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
				svc := newOpenAIRecordUsageServiceForTest(usageRepo, &openAIRecordUsageUserRepoStub{}, &openAIRecordUsageSubRepoStub{}, nil)
				err := svc.RecordUsage(context.Background(), &OpenAIRecordUsageInput{
					Result: &OpenAIForwardResult{RequestID: "boundary", Model: "gpt-6.1-sol", Duration: time.Second,
						Usage: OpenAIUsage{InputTokens: totalInput, OutputTokens: 1000}},
					APIKey: openAIRecordUsageAPIKeyWithGroup(svc, 1, true), User: &User{ID: 2},
					Account: &Account{ID: 3, Platform: PlatformOpenAI, Type: accountType,
						Extra: map[string]any{openAILongContextBillingEnabledKey: true}},
				})
				require.NoError(t, err)
				wantLong := totalInput > 272000
				wantInput, wantOutput := float64(totalInput)*7e-6, 0.035
				if wantLong {
					wantInput *= 2
					wantOutput *= 1.5
				}
				require.Equal(t, wantLong, usageRepo.lastLog.LongContextBillingApplied)
				require.InDelta(t, wantInput, usageRepo.lastLog.InputCost, 1e-12)
				require.InDelta(t, wantOutput, usageRepo.lastLog.OutputCost, 1e-12)
			})
		}
	}
}

func TestOpenAIGatewayServiceRecordUsage_GPT61SolRetailLongContextSurvivesRepricing(t *testing.T) {
	usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
	userRepo := &openAIRecordUsageUserRepoStub{}
	svc := newOpenAIRecordUsageServiceForTest(usageRepo, userRepo, &openAIRecordUsageSubRepoStub{}, nil)
	apiKey := openAIRecordUsageAPIKeyWithGroup(svc, 1, true)
	apiKey.GroupID = i64p(1)
	apiKey.Group.Platform = PlatformOpenAI
	apiKey.Group.RateMultiplier = 1.1
	apiKey.Group.FreeOpenAIFast = true
	svc.channelService = newTestChannelServiceForStats(t, &Channel{ID: 1, Status: StatusActive}, 1, PlatformOpenAI)
	tier := "priority"
	err := svc.RecordUsage(context.Background(), &OpenAIRecordUsageInput{
		Result: &OpenAIForwardResult{
			RequestID: "repricing", Model: "gpt-6-sol", UpstreamModel: "gpt-6.1-sol", BillingModel: "gpt-6-sol",
			UpstreamResponseModel: "gpt-6.1-sol", ServiceTier: &tier, Duration: time.Second,
			Usage: OpenAIUsage{InputTokens: 300000, OutputTokens: 2000, CacheReadInputTokens: 80000},
		},
		APIKey: apiKey, User: &User{ID: 2},
		Account: &Account{ID: 3, Platform: PlatformOpenAI, Type: AccountTypeOAuth,
			Extra: map[string]any{openAILongContextBillingEnabledKey: true}},
		ChannelUsageFields: ChannelUsageFields{BillingModelSource: BillingModelSourceResponse},
	})
	require.NoError(t, err)
	// Response repricing and free Fast retain the established retail long-context policy.
	const standardCost = (220000*2e-6*2 + 80000*0.1e-6*2 + 2000*10e-6*1.5) * 3.5
	require.InDelta(t, standardCost*2, usageRepo.lastLog.TotalCost, 1e-12)
	require.InDelta(t, standardCost*1.1, usageRepo.lastLog.ActualCost, 1e-12)
	require.InDelta(t, standardCost*1.1, userRepo.lastAmount, 1e-12)
	require.True(t, usageRepo.lastLog.LongContextBillingApplied)
	require.NotNil(t, usageRepo.lastLog.AccountStatsCost)
	require.InDelta(t, standardCost*2, *usageRepo.lastLog.AccountStatsCost, 1e-12)
}
