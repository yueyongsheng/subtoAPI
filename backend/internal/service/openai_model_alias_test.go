package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeKnownOpenAICodexModel_BareGPT56RoutesToSol(t *testing.T) {
	tests := map[string]string{
		"gpt-5.6":            "gpt-5.6-sol",
		"openai/gpt-5.6":     "gpt-5.6-sol",
		"gpt5.6":             "gpt-5.6-sol",
		"gpt-5.6-high":       "gpt-5.6-sol",
		"gpt-5.6-max":        "gpt-5.6-sol",
		"gpt-5.6-2026-07-09": "gpt-5.6-sol",
		"openai/gpt-5.6-max": "gpt-5.6-sol",
	}

	for input, expected := range tests {
		t.Run(input, func(t *testing.T) {
			require.Equal(t, expected, normalizeKnownOpenAICodexModel(input))
		})
	}
}

func TestNormalizeKnownOpenAICodexModel_AstraIsExactAndUnknownGPT6FailsClosed(t *testing.T) {
	require.Equal(t, "gpt-6-astra", normalizeKnownOpenAICodexModel("gpt-6-astra"))
	require.Equal(t, "gpt-6-astra", normalizeKnownOpenAICodexModel("openai/gpt-6-astra"))
	for _, model := range []string{"gpt-6", "gpt-6-pro", "openai/gpt-6-preview", "gpt-6-astra-2026-08-01"} {
		require.Empty(t, normalizeKnownOpenAICodexModel(model), model)
	}
}

func TestSupportsOpenAIReasoningEffortMaxIncludesAstra(t *testing.T) {
	require.True(t, supportsOpenAIReasoningEffortMax("gpt-6-astra"))
	require.True(t, supportsOpenAIReasoningEffortMax("openai/gpt-6-astra"))
	require.False(t, supportsOpenAIReasoningEffortMax("gpt-6-preview"))
}

func TestUsageBillingModelCandidates_BareGPT56IncludesSol(t *testing.T) {
	require.Equal(t,
		[]string{"gpt-5.6", "gpt-5.6-sol"},
		usageBillingModelCandidates("gpt-5.6"),
	)
	require.Equal(t,
		[]string{"openai/gpt-5.6", "gpt-5.6", "gpt-5.6-sol"},
		usageBillingModelCandidates("openai/gpt-5.6"),
	)
}
