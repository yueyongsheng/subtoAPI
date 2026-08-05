//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetPublishedModelPricing_OpenAITiersMatchLiveBilling(t *testing.T) {
	svc := newTestBillingService()

	tests := []struct {
		model         string
		input         float64
		output        float64
		cacheWrite    float64
		cacheRead     float64
		fastInput     float64
		fastOutput    float64
		fastCacheRead float64
	}{
		{"gpt-5.6-sol", 17.5e-6, 105e-6, 21.875e-6, 1.75e-6, 35e-6, 210e-6, 3.5e-6},
		{"gpt-5.6-terra", 7e-6, 42e-6, 8.75e-6, 0.7e-6, 14e-6, 84e-6, 1.4e-6},
		{"gpt-5.6-luna", 0.7e-6, 4.2e-6, 0.875e-6, 0.07e-6, 1.4e-6, 8.4e-6, 0.14e-6},
		{"gpt-5.5", 17.5e-6, 105e-6, 21.875e-6, 1.75e-6, 35e-6, 210e-6, 3.5e-6},
		{"codex-auto-review", 17.5e-6, 105e-6, 21.875e-6, 1.75e-6, 35e-6, 210e-6, 3.5e-6},
		{"gpt-5.4", 8.75e-6, 52.5e-6, 8.75e-6, 0.875e-6, 17.5e-6, 105e-6, 1.75e-6},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got, err := svc.GetPublishedModelPricing(tt.model)
			require.NoError(t, err)
			require.InDelta(t, tt.input, got.Standard.Input, 1e-12)
			require.InDelta(t, tt.output, got.Standard.Output, 1e-12)
			require.InDelta(t, tt.cacheWrite, got.Standard.CacheWrite, 1e-12)
			require.InDelta(t, tt.cacheRead, got.Standard.CacheRead, 1e-12)
			require.InDelta(t, tt.fastInput, got.Fast.Input, 1e-12)
			require.InDelta(t, tt.fastOutput, got.Fast.Output, 1e-12)
			require.InDelta(t, tt.fastCacheRead, got.Fast.CacheRead, 1e-12)
			require.Equal(t, 272000, got.LongContextThreshold)
			require.InDelta(t, 2.0, got.LongContextInputRate, 1e-12)
			require.InDelta(t, 1.5, got.LongContextOutputRate, 1e-12)
		})
	}
}

func TestGetPublishedModelPricing_PriorityFallbackMatchesBillingMultiplier(t *testing.T) {
	svc := newTestBillingService()

	got, err := svc.GetPublishedModelPricing("claude-sonnet-4")
	require.NoError(t, err)
	require.InDelta(t, got.Standard.Input*2, got.Fast.Input, 1e-12)
	require.InDelta(t, got.Standard.Output*2, got.Fast.Output, 1e-12)
	require.InDelta(t, got.Standard.CacheWrite*2, got.Fast.CacheWrite, 1e-12)
	require.InDelta(t, got.Standard.CacheRead*2, got.Fast.CacheRead, 1e-12)
}

func TestPublishedTokenPricesForTier_CacheBreakdownMatchesLivePrioritySelection(t *testing.T) {
	pricing := &ModelPricing{
		InputPricePerToken:                 1e-6,
		InputPricePerTokenPriority:         2e-6,
		CacheCreationPricePerToken:         1.25e-6,
		CacheCreationPricePerTokenPriority: 2.5e-6,
		CacheCreation5mPrice:               1.4e-6,
		CacheCreation1hPrice:               2.8e-6,
		SupportsCacheBreakdown:             true,
	}

	got := publishedTokenPricesForTier(pricing, "priority")

	require.InDelta(t, 2e-6, got.Input, 1e-12)
	require.InDelta(t, 1.4e-6, got.CacheWrite, 1e-12)
}
