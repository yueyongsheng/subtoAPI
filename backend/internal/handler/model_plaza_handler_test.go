//go:build unit

package handler

import (
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestScalePublishedTier_AppliesPerMillionAndGroupRate(t *testing.T) {
	got := scalePublishedTier(service.PublishedTokenPrices{
		Input:      5e-6,
		Output:     30e-6,
		CacheWrite: 6.25e-6,
		CacheRead:  0.5e-6,
	}, 0.12)

	require.InDelta(t, 0.6, got.Input, 1e-12)
	require.InDelta(t, 3.6, got.Output, 1e-12)
	require.InDelta(t, 0.75, got.CacheWrite, 1e-12)
	require.InDelta(t, 0.06, got.CacheRead, 1e-12)
}

func TestUserModelPlazaResponse_DoesNotExposeChannelOrAccountFields(t *testing.T) {
	payload, err := json.Marshal(userModelPlazaResponse{
		Currency: "USD",
		Unit:     "per_million_tokens",
		Groups: []userModelPlazaGroup{{
			ID:             1,
			Name:           "plus-quarantine",
			Platform:       "openai",
			RateMultiplier: 0.12,
		}},
		Models: []userModelPlazaModel{{
			Name:         "gpt-5.6-sol",
			Platform:     "openai",
			BaseStandard: userModelPlazaTier{Input: 5, Output: 30, CacheWrite: 6.25, CacheRead: 0.5},
			BaseFast:     userModelPlazaTier{Input: 10, Output: 60, CacheWrite: 12.5, CacheRead: 1},
			Prices: []userModelPlazaGroupPrice{{
				GroupID:  1,
				Standard: userModelPlazaTier{Input: 0.6, Output: 3.6},
				Fast:     userModelPlazaTier{Input: 1.2, Output: 7.2},
			}},
		}},
	})
	require.NoError(t, err)

	serialized := string(payload)
	require.NotContains(t, serialized, "channel")
	require.NotContains(t, serialized, "account")
	require.NotContains(t, serialized, "cost_multiplier")
	require.Contains(t, serialized, "rate_multiplier")
	require.Contains(t, serialized, "base_standard")
	require.Contains(t, serialized, "base_fast")
}

func TestUserModelPlazaResponse_BaseAndFinalPricesUseSamePublishedTier(t *testing.T) {
	published := service.PublishedTokenPrices{
		Input:      2e-6,
		Output:     12e-6,
		CacheWrite: 2.5e-6,
		CacheRead:  0.2e-6,
	}

	base := scalePublishedTier(published, 1)
	final := scalePublishedTier(published, 0.12)

	require.InDelta(t, 2, base.Input, 1e-12)
	require.InDelta(t, 12, base.Output, 1e-12)
	require.InDelta(t, 0.24, final.Input, 1e-12)
	require.InDelta(t, 1.44, final.Output, 1e-12)
	require.InDelta(t, base.CacheRead*0.12, final.CacheRead, 1e-12)
}
