//go:build unit

package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestApplyYuexiangCCGrokBasePricePolicyScalesEveryTokenPrice(t *testing.T) {
	base := &ModelPricing{
		InputPricePerToken:                 1,
		InputPricePerTokenPriority:         2,
		OutputPricePerToken:                3,
		OutputPricePerTokenPriority:        4,
		CacheCreationPricePerToken:         5,
		CacheCreationPricePerTokenPriority: 6,
		CacheReadPricePerToken:             7,
		CacheReadPricePerTokenPriority:     8,
		CacheCreation5mPrice:               9,
		CacheCreation1hPrice:               10,
		ImageInputPricePerToken:            11,
		ImageOutputPricePerToken:           12,
		SupportsCacheBreakdown:             true,
		LongContextInputThreshold:          200000,
		LongContextThresholdInclusive:      true,
		LongContextInputMultiplier:         2,
		LongContextOutputMultiplier:        1.5,
	}

	for _, model := range []string{
		"claude-sonnet-6-preview",
		"projects/p/publishers/anthropic/models/claude-opus-6@20270101",
		"grok-5-preview",
		"xai/grok-5-preview",
	} {
		t.Run(model, func(t *testing.T) {
			got := applyYuexiangCCGrokBasePricePolicy(model, base)
			require.NotSame(t, base, got)
			require.Equal(t, 3.5, got.InputPricePerToken)
			require.Equal(t, 7.0, got.InputPricePerTokenPriority)
			require.Equal(t, 10.5, got.OutputPricePerToken)
			require.Equal(t, 14.0, got.OutputPricePerTokenPriority)
			require.Equal(t, 17.5, got.CacheCreationPricePerToken)
			require.Equal(t, 21.0, got.CacheCreationPricePerTokenPriority)
			require.Equal(t, 24.5, got.CacheReadPricePerToken)
			require.Equal(t, 28.0, got.CacheReadPricePerTokenPriority)
			require.Equal(t, 31.5, got.CacheCreation5mPrice)
			require.Equal(t, 35.0, got.CacheCreation1hPrice)
			require.Equal(t, 11.0, got.ImageInputPricePerToken)
			require.Equal(t, 12.0, got.ImageOutputPricePerToken)
			require.Equal(t, 200000, got.LongContextInputThreshold)
			require.True(t, got.LongContextThresholdInclusive)
			require.Equal(t, 2.0, got.LongContextInputMultiplier)
			require.Equal(t, 1.5, got.LongContextOutputMultiplier)
		})
	}

	// The source price is shared by dynamic and fallback catalogs and must not mutate.
	require.Equal(t, 1.0, base.InputPricePerToken)
	require.Equal(t, 10.0, base.CacheCreation1hPrice)
}

func TestApplyYuexiangCCGrokBasePricePolicyExcludesNonTextModels(t *testing.T) {
	base := &ModelPricing{InputPricePerToken: 1, OutputPricePerToken: 2}
	for _, model := range []string{
		"gpt-5.4",
		"grok-imagine-image-quality",
		"grok-imagine-video-1.5",
		"grok-voice-latest",
		"grok-web-search",
		"grok-x-search",
	} {
		t.Run(model, func(t *testing.T) {
			require.Same(t, base, applyYuexiangCCGrokBasePricePolicy(model, base))
		})
	}
	require.Nil(t, applyYuexiangCCGrokBasePricePolicy("claude-sonnet-6", nil))
}

func TestGetModelPricingAppliesYuexiangPolicyToDynamicClaudeAndGrok(t *testing.T) {
	for _, model := range []string{"claude-sonnet-6-preview", "grok-5-preview"} {
		t.Run(model, func(t *testing.T) {
			remote := &LiteLLMModelPricing{
				InputCostPerToken:                   1e-6,
				InputCostPerTokenPriority:           2e-6,
				OutputCostPerToken:                  3e-6,
				OutputCostPerTokenPriority:          4e-6,
				CacheCreationInputTokenCost:         1.25e-6,
				CacheCreationInputTokenCostPriority: 2.5e-6,
				CacheCreationInputTokenCostAbove1hr: 2e-6,
				CacheReadInputTokenCost:             0.1e-6,
				CacheReadInputTokenCostPriority:     0.2e-6,
				LongContextInputTokenThreshold:      200000,
				LongContextInputCostMultiplier:      2,
				LongContextOutputCostMultiplier:     1.5,
				InputCostPerImageToken:              7e-6,
				OutputCostPerImageToken:             8e-6,
			}
			pricingService := &PricingService{pricingData: map[string]*LiteLLMModelPricing{model: remote}}
			svc := NewBillingService(&config.Config{}, pricingService)

			got, err := svc.GetModelPricing(model)
			require.NoError(t, err)
			require.InDelta(t, 3.5e-6, got.InputPricePerToken, 1e-12)
			require.InDelta(t, 7e-6, got.InputPricePerTokenPriority, 1e-12)
			require.InDelta(t, 10.5e-6, got.OutputPricePerToken, 1e-12)
			require.InDelta(t, 14e-6, got.OutputPricePerTokenPriority, 1e-12)
			require.InDelta(t, 4.375e-6, got.CacheCreationPricePerToken, 1e-12)
			require.InDelta(t, 8.75e-6, got.CacheCreationPricePerTokenPriority, 1e-12)
			require.InDelta(t, 7e-6, got.CacheCreation1hPrice, 1e-12)
			require.InDelta(t, 0.35e-6, got.CacheReadPricePerToken, 1e-12)
			require.InDelta(t, 0.7e-6, got.CacheReadPricePerTokenPriority, 1e-12)
			require.InDelta(t, 7e-6, got.ImageInputPricePerToken, 1e-12)
			require.InDelta(t, 8e-6, got.ImageOutputPricePerToken, 1e-12)
			require.Equal(t, 200000, got.LongContextInputThreshold)
			require.InDelta(t, 2.0, got.LongContextInputMultiplier, 1e-12)
			require.InDelta(t, 1.5, got.LongContextOutputMultiplier, 1e-12)

			// Remote price updates remain the 1x source and cannot bypass the policy.
			remote.InputCostPerToken = 2e-6
			updated, err := svc.GetModelPricing(model)
			require.NoError(t, err)
			require.InDelta(t, 7e-6, updated.InputPricePerToken, 1e-12)
		})
	}
}

func TestGetModelPricingAppliesYuexiangPolicyToFallbacks(t *testing.T) {
	svc := newTestBillingService()
	tests := []struct {
		model      string
		input      float64
		output     float64
		cacheWrite float64
		cacheRead  float64
	}{
		{"claude-sonnet-4", 10.5e-6, 52.5e-6, 13.125e-6, 1.05e-6},
		{"claude-opus-4.6", 17.5e-6, 87.5e-6, 21.875e-6, 1.75e-6},
		{"claude-3-haiku", 0.875e-6, 4.375e-6, 1.05e-6, 0.105e-6},
		{"grok-4.5", 7e-6, 21e-6, 0, 1.05e-6},
		{"grok-4.6", 7e-6, 21e-6, 0, 1.75e-6},
		{"grok-5-preview", 7e-6, 21e-6, 0, 1.05e-6},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got, err := svc.GetModelPricing(tt.model)
			require.NoError(t, err)
			require.InDelta(t, tt.input, got.InputPricePerToken, 1e-12)
			require.InDelta(t, tt.output, got.OutputPricePerToken, 1e-12)
			require.InDelta(t, tt.cacheWrite, got.CacheCreationPricePerToken, 1e-12)
			require.InDelta(t, tt.cacheRead, got.CacheReadPricePerToken, 1e-12)
		})
	}
}

func TestYuexiangCCGrokChannelOverridesAndPublishedPricesUseLiveBilling(t *testing.T) {
	remote := &LiteLLMModelPricing{
		InputCostPerToken:           2e-6,
		OutputCostPerToken:          6e-6,
		CacheCreationInputTokenCost: 2.5e-6,
		CacheReadInputTokenCost:     0.5e-6,
	}
	pricingService := &PricingService{pricingData: map[string]*LiteLLMModelPricing{"claude-sonnet-6-preview": remote}}
	svc := NewBillingService(&config.Config{}, pricingService)

	live, err := svc.GetModelPricing("claude-sonnet-6-preview")
	require.NoError(t, err)
	published, err := svc.GetPublishedModelPricing("claude-sonnet-6-preview")
	require.NoError(t, err)
	require.InDelta(t, live.InputPricePerToken, published.Standard.Input, 1e-12)
	require.InDelta(t, live.OutputPricePerToken, published.Standard.Output, 1e-12)
	require.InDelta(t, live.CacheCreationPricePerToken, published.Standard.CacheWrite, 1e-12)
	require.InDelta(t, live.CacheReadPricePerToken, published.Standard.CacheRead, 1e-12)

	input, output, cacheWrite, cacheRead := 1e-6, 2e-6, 3e-6, 4e-6
	overridden, err := svc.GetModelPricingWithChannel("claude-sonnet-6-preview", &ChannelModelPricing{
		InputPrice:      &input,
		OutputPrice:     &output,
		CacheWritePrice: &cacheWrite,
		CacheReadPrice:  &cacheRead,
	})
	require.NoError(t, err)
	require.InDelta(t, input, overridden.InputPricePerToken, 1e-12)
	require.InDelta(t, output, overridden.OutputPricePerToken, 1e-12)
	require.InDelta(t, cacheWrite, overridden.CacheCreationPricePerToken, 1e-12)
	require.InDelta(t, cacheRead, overridden.CacheReadPricePerToken, 1e-12)

	// GPT's existing authoritative card is independent from this policy.
	gpt, err := svc.GetModelPricing("gpt-5.4")
	require.NoError(t, err)
	require.InDelta(t, 8.75e-6, gpt.InputPricePerToken, 1e-12)
	require.InDelta(t, 52.5e-6, gpt.OutputPricePerToken, 1e-12)
}
