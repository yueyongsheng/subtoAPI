//go:build unit

package service

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/stretchr/testify/require"
)

func TestGPT62IdentityAndCapabilities(t *testing.T) {
	for _, model := range []string{"gpt-6.2-astra", "gpt-6.2-sol"} {
		t.Run(model, func(t *testing.T) {
			require.Contains(t, openai.DefaultModelIDs(), model)
			require.True(t, openai.IsPublicOpenAIModelID(model))
			require.True(t, openAIModelSupportsUltrafast(model))
			for _, effort := range []string{"low", "medium", "high", "xhigh", "max"} {
				alias := "openai/" + model + "-" + effort
				require.Equal(t, model, normalizeCodexModel(alias))
				require.Equal(t, model, normalizeKnownOpenAICodexModel(alias))
				require.Equal(t, model, NormalizeOpenAICompatRequestedModel(alias))
				require.Equal(t, effort, normalizeOpenAIReasoningEffortForModel(effort, alias))
				req := &apicompat.AnthropicRequest{Model: alias}
				applyOpenAICompatModelNormalization(req)
				converted, err := apicompat.AnthropicToResponses(req)
				require.NoError(t, err)
				require.Equal(t, model, converted.Model)
				require.Equal(t, effort, converted.Reasoning.Effort)
				require.True(t, shouldAutoInjectPromptCacheKeyForCompat(alias))
			}
			for _, effort := range []string{"none", "minimal"} {
				require.Error(t, validateGPT61CompatRequest([]byte(`{"reasoning":{"effort":"`+effort+`"}}`), model))
				_, err := apicompat.ChatCompletionsToResponses(&apicompat.ChatCompletionsRequest{Model: model, ReasoningEffort: effort})
				require.ErrorContains(t, err, "does not support reasoning effort")
			}
			require.True(t, openai.RequiresResponsesForToolCalls(model))
			converted, err := apicompat.ChatCompletionsToResponses(&apicompat.ChatCompletionsRequest{Model: model, ReasoningEffort: "max", Tools: []apicompat.ChatTool{{Type: "function", Function: &apicompat.ChatFunction{Name: "lookup"}}}})
			require.NoError(t, err)
			require.Len(t, converted.Tools, 1)
			body, err := BuildCodexModelsManifest([]string{model})
			require.NoError(t, err)
			models := decodeCodexManifestModels(t, body)
			require.Len(t, models, 1)
			require.Equal(t, model, models[0]["slug"])
			expectedContext := 500000
			if model == "gpt-6.2-astra" {
				expectedContext = 272000
			}
			require.EqualValues(t, expectedContext, models[0]["context_window"])
			require.EqualValues(t, 872000, models[0]["max_context_window"])
			require.Equal(t, []string{"low", "medium", "high", "xhigh", "max", "ultra"}, effortsFromManifestModel(t, models[0]))
			var official map[string]any
			require.NoError(t, json.Unmarshal(openai.GPT62CodexMetadata(model), &official))
			require.Equal(t, official["model_messages"].(map[string]any)["instructions_template"], openai.CodexBaseInstructionsForModel(model))
			apiKey := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{"base_url": "https://api.openai.com"}}
			caps := accountCodexToolCapabilities(apiKey, model)
			require.JSONEq(t, "false", string(caps["use_responses_lite"]))
		})
	}
	for _, model := range []string{"gpt-6.2", "gpt-6.2-luna", "gpt-6.2-astra-preview", "gpt-6.2-sol-pro", "gpt-6.2-sol-2026-11-18"} {
		require.Empty(t, openai.GPT62ModelID(model))
		require.False(t, openai.IsPublicOpenAIModelID(model))
		_, err := newTestBillingService().GetModelPricing(model)
		require.ErrorIs(t, err, ErrModelPricingUnavailable)
	}
}

func TestGPT62PricesAndUsageAcrossSources(t *testing.T) {
	embedded, err := os.ReadFile("../../resources/model-pricing/model_prices_and_context_window.json")
	require.NoError(t, err)
	for _, tc := range []struct {
		model                               string
		input, output, write, read, write12 float64
	}{
		{"gpt-6.2-astra", 28e-6, 140e-6, 35e-6, 1.4e-6, 70e-6},
		{"gpt-6.2-sol", 7e-6, 35e-6, 8.75e-6, .35e-6, 17.5e-6},
	} {
		for _, source := range []string{"unavailable", "embedded", "official-1x"} {
			t.Run(tc.model+"/"+source, func(t *testing.T) {
				var catalog *PricingService
				if source == "embedded" {
					catalog = newStubPricingServiceFromJSON(t, string(embedded))
				}
				if source == "official-1x" {
					catalog = &PricingService{pricingData: map[string]*LiteLLMModelPricing{tc.model: {InputCostPerToken: tc.input / 3.5, OutputCostPerToken: tc.output / 3.5, CacheCreationInputTokenCost: tc.write / 3.5, CacheReadInputTokenCost: tc.read / 3.5}}}
				}
				svc := NewBillingService(nil, catalog)
				p, err := svc.GetModelPricing(tc.model)
				require.NoError(t, err)
				require.InDelta(t, tc.input, p.InputPricePerToken, 1e-12)
				require.InDelta(t, tc.output, p.OutputPricePerToken, 1e-12)
				require.NotNil(t, p.CacheCreation12hPricePerToken)
				require.InDelta(t, tc.write12, *p.CacheCreation12hPricePerToken, 1e-12)
				published, err := svc.GetPublishedModelPricing(tc.model)
				require.NoError(t, err)
				require.InDelta(t, tc.input, published.Standard.Input, 1e-12)
				require.InDelta(t, tc.output, published.Standard.Output, 1e-12)
				require.InDelta(t, tc.write, published.Standard.CacheWrite, 1e-12)
				require.InDelta(t, tc.read, published.Standard.CacheRead, 1e-12)
				require.InDelta(t, tc.input*2, published.Fast.Input, 1e-12)
				for _, totalInput := range []int{272000, 272001} {
					for tier, factor := range map[string]float64{"default": 1, "priority": 2, "flex": .5} {
						tokens := UsageTokens{InputTokens: totalInput - 1300, OutputTokens: 57, CacheReadTokens: 300, CacheCreationTokens: 1000, CacheCreation30mTokens: 400, CacheCreation12hTokens: 600}
						inputFactor, outputFactor := 1.0, 1.0
						if totalInput > 272000 {
							inputFactor, outputFactor = 2, 1.5
						}
						expected := ((float64(tokens.InputTokens)*tc.input+300*tc.read+400*tc.write+600*tc.write12)*inputFactor + 57*tc.output*outputFactor) * factor
						for _, rate := range []float64{1.1, 1.88, 1.5} {
							got, err := svc.CalculateCostWithServiceTier(tc.model, tokens, rate, tier)
							require.NoError(t, err)
							require.InDelta(t, expected, got.TotalCost, 1e-9, "%s input=%d", tier, totalInput)
							require.InDelta(t, expected*rate, got.ActualCost, 1e-9)
						}
					}
				}
				boundary := time.Date(2027, 3, 1, 8, 0, 0, 0, time.UTC)
				for _, stamp := range []time.Time{boundary.Add(-time.Nanosecond), boundary} {
					card, err := svc.getModelPricingAt(tc.model, stamp)
					require.NoError(t, err)
					factor := 3.0
					if !stamp.Before(boundary) {
						factor = 6
					}
					got := svc.computeTokenBreakdown(card, UsageTokens{InputTokens: 100, OutputTokens: 10, CacheCreationTokens: 20, CacheCreation12hTokens: 20}, 1.88, "ultrafast", true)
					require.InDelta(t, (100*tc.input+10*tc.output+20*tc.write12)*factor*1.88, got.ActualCost, 1e-10)
				}
			})
		}
	}
}

func TestGPT62PricingInitializationWithoutRemote(t *testing.T) {
	svc := NewPricingService(&config.Config{Pricing: config.PricingConfig{
		DataDir: t.TempDir(), RemoteURL: "https://unavailable.example/pricing.json",
		FallbackFile: "../../resources/model-pricing/model_prices_and_context_window.json",
	}}, &failingHashRemoteClient{})
	require.NoError(t, svc.Initialize())
	t.Cleanup(svc.Stop)
	for model, input := range map[string]float64{"gpt-6.2-astra": 28e-6, "gpt-6.2-sol": 7e-6} {
		require.InDelta(t, input, svc.GetModelPricing(model).InputCostPerToken, 1e-12)
		card, err := NewBillingService(nil, svc).GetModelPricing(model)
		require.NoError(t, err)
		require.InDelta(t, input, card.InputPricePerToken, 1e-12)
		require.NotNil(t, card.CacheCreation12hPricePerToken)
	}
}
