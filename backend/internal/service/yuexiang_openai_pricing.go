package service

var yuexiangOpenAIChargedModels = [...]string{
	"gpt-6-astra",
	"gpt-5.6-sol",
	"gpt-5.5",
	"codex-auto-review",
	"gpt-5.6-terra",
	"gpt-5.4",
	"gpt-5.6-luna",
}

// yuexiangOpenAIModelPricing is the authoritative base card for the explicitly
// approved charged OpenAI models. Group model_pricing and channel
// overrides are applied later by ModelPricingResolver.
func yuexiangOpenAIModelPricing(model string) (*ModelPricing, bool) {
	var input, output, cacheWrite, cacheRead float64
	switch model {
	case "gpt-6-astra":
		input, output, cacheWrite, cacheRead = 35e-6, 175e-6, 43.75e-6, 3.5e-6
	case "gpt-5.6-sol", "gpt-5.5", "codex-auto-review":
		input, output, cacheWrite, cacheRead = 17.5e-6, 105e-6, 21.875e-6, 1.75e-6
	case "gpt-5.6-terra":
		input, output, cacheWrite, cacheRead = 7e-6, 42e-6, 8.75e-6, 0.7e-6
	case "gpt-5.4":
		input, output, cacheWrite, cacheRead = 8.75e-6, 52.5e-6, 8.75e-6, 0.875e-6
	case "gpt-5.6-luna":
		input, output, cacheWrite, cacheRead = 0.7e-6, 4.2e-6, 0.875e-6, 0.07e-6
	default:
		return nil, false
	}
	return &ModelPricing{
		InputPricePerToken:                 input,
		InputPricePerTokenPriority:         input * 2,
		OutputPricePerToken:                output,
		OutputPricePerTokenPriority:        output * 2,
		CacheCreationPricePerToken:         cacheWrite,
		CacheCreationPricePerTokenPriority: cacheWrite * 2,
		CacheReadPricePerToken:             cacheRead,
		CacheReadPricePerTokenPriority:     cacheRead * 2,
		LongContextInputThreshold:          openAIGPT54LongContextInputThreshold,
		LongContextInputMultiplier:         openAIGPT54LongContextInputMultiplier,
		LongContextOutputMultiplier:        openAIGPT54LongContextOutputMultiplier,
	}, true
}
