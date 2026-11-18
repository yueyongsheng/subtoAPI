package service

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// A new upstream TTL breakdown must not silently reprice deployed models.
func TestYuexiangUpgradePreservesCacheWriteCostAcrossTTLs(t *testing.T) {
	embedded, err := os.ReadFile("../../resources/model-pricing/model_prices_and_context_window.json")
	require.NoError(t, err)
	for _, source := range []string{"fallback", "embedded"} {
		var catalog *PricingService
		if source == "embedded" {
			catalog = newStubPricingServiceFromJSON(t, string(embedded))
		}
		svc := NewBillingService(nil, catalog)
		// Freeze the deployed pre-GPT-6.2 cards: new models have explicitly
		// published 12h prices and must not inherit this legacy fallback.
		for _, model := range []string{"gpt-6-astra", "gpt-6-sol", "gpt-6.1-sol", "gpt-6.1-terra", "gpt-5.6-sol", "gpt-5.5", "codex-auto-review", "gpt-5.6-terra", "gpt-5.4", "gpt-5.6-luna"} {
			for _, tier := range []string{"", "priority", "flex", "batch"} {
				for _, input := range []int{100, 272000, 272001} {
					t.Run(fmt.Sprintf("%s/%s/%s/%d", source, model, tier, input), func(t *testing.T) {
						p, err := svc.GetModelPricing(model)
						require.NoError(t, err)
						require.Nil(t, p.CacheCreation12hPricePerToken)
						legacy := UsageTokens{InputTokens: input, OutputTokens: 57, CacheReadTokens: 73, CacheCreationTokens: 1000}
						for _, rate := range []float64{1.1, 1.88, 1.5} {
							before, err := svc.CalculateCostWithServiceTier(model, legacy, rate, tier)
							require.NoError(t, err)
							for _, short := range []int{0, 400, 1000} {
								split := legacy
								split.CacheCreation30mTokens = short
								split.CacheCreation12hTokens = 1000 - short
								after, err := svc.CalculateCostWithServiceTier(model, split, rate, tier)
								require.NoError(t, err)
								require.InDelta(t, before.CacheCreationCost, after.CacheCreationCost, 1e-12)
								require.InDelta(t, before.ActualCost, after.ActualCost, 1e-12)
							}
						}
					})
				}
			}
		}
	}
}
