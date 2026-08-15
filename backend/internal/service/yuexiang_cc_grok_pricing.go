package service

import "strings"

const yuexiangTextBasePriceMultiplier = 3.5

// applyYuexiangCCGrokBasePricePolicy applies the public markup after the normal
// dynamic/fallback token price has resolved. Explicit channel and group prices
// are layered later and therefore remain authoritative.
func applyYuexiangCCGrokBasePricePolicy(model string, pricing *ModelPricing) *ModelPricing {
	if pricing == nil || !isYuexiangCCGrokTokenModel(model) {
		return pricing
	}

	cloned := *pricing
	cloned.InputPricePerToken *= yuexiangTextBasePriceMultiplier
	cloned.InputPricePerTokenPriority *= yuexiangTextBasePriceMultiplier
	cloned.OutputPricePerToken *= yuexiangTextBasePriceMultiplier
	cloned.OutputPricePerTokenPriority *= yuexiangTextBasePriceMultiplier
	cloned.CacheCreationPricePerToken *= yuexiangTextBasePriceMultiplier
	cloned.CacheCreationPricePerTokenPriority *= yuexiangTextBasePriceMultiplier
	cloned.CacheReadPricePerToken *= yuexiangTextBasePriceMultiplier
	cloned.CacheReadPricePerTokenPriority *= yuexiangTextBasePriceMultiplier
	cloned.CacheCreation5mPrice *= yuexiangTextBasePriceMultiplier
	cloned.CacheCreation1hPrice *= yuexiangTextBasePriceMultiplier
	return &cloned
}

func isYuexiangCCGrokTokenModel(model string) bool {
	normalized := strings.ToLower(strings.TrimSpace(model))
	return strings.Contains(normalized, "claude") || isGrokUnknownTextFamilyModel(normalized)
}
