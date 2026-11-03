//go:build unit

package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestGetModelPricingAppliesYuexiangMarkupToGPT61Terra(t *testing.T) {
	// An upstream source may still carry the official 1x card. Billing must
	// expose the 3.5x Yuexiang card regardless of which source is loaded.
	pricingService := &PricingService{pricingData: map[string]*LiteLLMModelPricing{
		"gpt-6.1-terra": {
			InputCostPerToken:                   1e-6,
			InputCostPerTokenPriority:           2e-6,
			OutputCostPerToken:                  5e-6,
			OutputCostPerTokenPriority:          10e-6,
			CacheCreationInputTokenCost:         1.25e-6,
			CacheCreationInputTokenCostPriority: 2.5e-6,
			CacheReadInputTokenCost:             0.05e-6,
			CacheReadInputTokenCostPriority:     0.1e-6,
		},
	}}
	svc := NewBillingService(&config.Config{}, pricingService)

	got, err := svc.GetModelPricing("gpt-6.1-terra")
	require.NoError(t, err)
	require.InDelta(t, 3.5e-6, got.InputPricePerToken, 1e-12)
	require.InDelta(t, 7e-6, got.InputPricePerTokenPriority, 1e-12)
	require.InDelta(t, 17.5e-6, got.OutputPricePerToken, 1e-12)
	require.InDelta(t, 35e-6, got.OutputPricePerTokenPriority, 1e-12)
	require.InDelta(t, 4.375e-6, got.CacheCreationPricePerToken, 1e-12)
	require.InDelta(t, 8.75e-6, got.CacheCreationPricePerTokenPriority, 1e-12)
	require.InDelta(t, 0.175e-6, got.CacheReadPricePerToken, 1e-12)
	require.InDelta(t, 0.35e-6, got.CacheReadPricePerTokenPriority, 1e-12)

	published, err := svc.GetPublishedModelPricing("gpt-6.1-terra")
	require.NoError(t, err)
	require.InDelta(t, got.InputPricePerToken, published.Standard.Input, 1e-12)
	require.InDelta(t, got.OutputPricePerToken, published.Standard.Output, 1e-12)
	require.InDelta(t, got.CacheCreationPricePerToken, published.Standard.CacheWrite, 1e-12)
	require.InDelta(t, got.CacheReadPricePerToken, published.Standard.CacheRead, 1e-12)
	require.InDelta(t, 7e-6, published.Fast.Input, 1e-12)
	require.InDelta(t, 35e-6, published.Fast.Output, 1e-12)
	require.InDelta(t, 0.35e-6, published.Fast.CacheRead, 1e-12)
	require.Equal(t, 272000, published.LongContextThreshold)
	require.InDelta(t, 2.0, published.LongContextInputRate, 1e-12)
	require.InDelta(t, 1.5, published.LongContextOutputRate, 1e-12)
}

func TestGetModelPricingWithChannelPreservesGPT61TerraOverride(t *testing.T) {
	svc := newTestBillingService()
	input, output, cacheWrite, cacheRead := 2e-6, 9e-6, 2.5e-6, 0.2e-6

	got, err := svc.GetModelPricingWithChannel("gpt-6.1-terra", &ChannelModelPricing{
		InputPrice:      &input,
		OutputPrice:     &output,
		CacheWritePrice: &cacheWrite,
		CacheReadPrice:  &cacheRead,
	})
	require.NoError(t, err)
	require.InDelta(t, input, got.InputPricePerToken, 1e-12)
	require.InDelta(t, output, got.OutputPricePerToken, 1e-12)
	require.InDelta(t, cacheWrite, got.CacheCreationPricePerToken, 1e-12)
	require.InDelta(t, cacheRead, got.CacheReadPricePerToken, 1e-12)
}
