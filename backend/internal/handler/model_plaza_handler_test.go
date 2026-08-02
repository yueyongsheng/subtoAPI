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
			Name:     "gpt-5.6-sol",
			Platform: "openai",
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
}
