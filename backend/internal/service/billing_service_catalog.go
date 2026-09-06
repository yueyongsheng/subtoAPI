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
	fast := tier == "priority" || tier == "fast"
	input, output := pricing.InputPricePerToken, pricing.OutputPricePerToken
	cacheWrite, cacheRead := pricing.CacheCreationPricePerToken, pricing.CacheReadPricePerToken
	if fast {
		if pricing.InputPricePerTokenPriority > 0 {
			input = pricing.InputPricePerTokenPriority
		} else {
			input *= 2
		}
		if pricing.OutputPricePerTokenPriority > 0 {
			output = pricing.OutputPricePerTokenPriority
		} else {
			output *= 2
		}
		if pricing.CacheCreationPricePerTokenPriority > 0 {
			cacheWrite = pricing.CacheCreationPricePerTokenPriority
		} else {
			cacheWrite *= 2
		}
		if pricing.CacheReadPricePerTokenPriority > 0 {
			cacheRead = pricing.CacheReadPricePerTokenPriority
		} else {
			cacheRead *= 2
		}
	}
	return PublishedTokenPrices{Input: input, Output: output, CacheWrite: cacheWrite, CacheRead: cacheRead}
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
