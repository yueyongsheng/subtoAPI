package openai

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultModelsIncludeBareGPT56Alias(t *testing.T) {
	require.Contains(t, DefaultModelIDs(), "gpt-5.6")
}

func TestDefaultModelsIncludeGPT6Astra(t *testing.T) {
	byID := make(map[string]Model, len(DefaultModels))
	for _, model := range DefaultModels {
		byID[model.ID] = model
	}

	model, ok := byID["gpt-6-astra"]
	require.True(t, ok)
	require.Equal(t, "GPT-6 Astra", model.DisplayName)
	require.Equal(t, "openai", model.OwnedBy)
}

func TestDefaultModelsExposeOnlyExactGPT6AstraID(t *testing.T) {
	ids := DefaultModelIDs()
	require.Contains(t, ids, "gpt-6-astra")
	require.NotContains(t, ids, "gpt-6")
	require.NotContains(t, ids, "astra")
}

func TestIsPublicOpenAIModelIDRequiresExactAstraID(t *testing.T) {
	require.True(t, IsPublicOpenAIModelID("gpt-6-astra"))
	for _, id := range []string{
		"gpt-6", "astra", "GPT-6-ASTRA", "gpt-6-astra-preview",
		"gpt-6-other", "openai/gpt-6-astra", "my-astra-alias",
	} {
		require.False(t, IsPublicOpenAIModelID(id), "model ID %q must stay private", id)
	}
	require.True(t, IsPublicOpenAIModelID("gpt-5.6-sol"))
}

func TestDefaultModelsPreferConcreteGPT56SolForAccountTests(t *testing.T) {
	require.NotEmpty(t, DefaultModels)
	require.Equal(t, "gpt-5.6-sol", DefaultModels[0].ID)
}

func TestDefaultModelsIncludeGPTImage25(t *testing.T) {
	require.Contains(t, DefaultModelIDs(), "gpt-image-2.5-flare")
	require.Contains(t, DefaultModelIDs(), "gpt-image-2.5-sunburst")
}

func TestGPT6AstraCreatedTimestampRemainsUnknown(t *testing.T) {
	for _, model := range DefaultModels {
		if model.ID == "gpt-6-astra" {
			require.Zero(t, model.Created)
			return
		}
	}
	t.Fatal("gpt-6-astra is missing from the default model catalog")
}
