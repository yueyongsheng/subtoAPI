package service

// PublishedTokenPrices is the public per-million-token price card shape.
type PublishedTokenPrices struct {
	Input      float64
	Output     float64
	CacheWrite float64
	CacheRead  float64
}

type PublishedModelPricing struct {
	Standard              PublishedTokenPrices
	Fast                  PublishedTokenPrices
	LongContextThreshold  int
	LongContextInputRate  float64
	LongContextOutputRate float64
}

func publishedTokenPricesForTier(pricing *ModelPricing, tier string) PublishedTokenPrices {
	if pricing == nil {
		return PublishedTokenPrices{}
	}
	// Price one token of each kind through the same tier selection as billing.
	cost := (&BillingService{}).computeTokenBreakdown(pricing, UsageTokens{
		InputTokens: 1, OutputTokens: 1, CacheCreationTokens: 1, CacheReadTokens: 1,
	}, 1, tier, false)
	return PublishedTokenPrices{
		Input: cost.InputCost, Output: cost.OutputCost,
		CacheWrite: cost.CacheCreationCost, CacheRead: cost.CacheReadCost,
	}
}

func (s *BillingService) GetPublishedModelPricing(model string) (*PublishedModelPricing, error) {
	pricing, err := s.GetModelPricing(model)
	if err != nil {
		return nil, err
	}
	return &PublishedModelPricing{
		Standard:              publishedTokenPricesForTier(pricing, "standard"),
		Fast:                  publishedTokenPricesForTier(pricing, "priority"),
		LongContextThreshold:  pricing.LongContextInputThreshold,
		LongContextInputRate:  pricing.LongContextInputMultiplier,
		LongContextOutputRate: pricing.LongContextOutputMultiplier,
	}, nil
}
