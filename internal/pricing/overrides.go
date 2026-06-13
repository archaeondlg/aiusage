package pricing

// PricingOverride represents user-specified pricing overrides from config.
type PricingOverride struct {
	InputCostPerToken                 *float64 `json:"inputCostPerToken"`
	OutputCostPerToken                *float64 `json:"outputCostPerToken"`
	CacheCreationInputTokenCost        *float64 `json:"cacheCreationInputTokenCost"`
	CacheReadInputTokenCost            *float64 `json:"cacheReadInputTokenCost"`
	InputCostPerTokenAbove200K         *float64 `json:"inputCostPerTokenAbove200kTokens"`
	OutputCostPerTokenAbove200K         *float64 `json:"outputCostPerTokenAbove200kTokens"`
	CacheCreationAbove200K             *float64 `json:"cacheCreationInputTokenCostAbove200kTokens"`
	CacheReadAbove200K                 *float64 `json:"cacheReadInputTokenCostAbove200kTokens"`
	FastMultiplier                     *float64 `json:"fastMultiplier"`
	MaxInputTokens                     *uint64  `json:"maxInputTokens"`
}

// applyOverrides applies user pricing overrides to existing entries.
func (m *PricingMap) applyOverrides(overrides map[string]*PricingOverride) {
	for model, o := range overrides {
		m.applyOverride(model, o)
	}
}

func (m *PricingMap) applyOverride(model string, o *PricingOverride) {
	base, exists := m.entries[model]
	if !exists {
		base = EmptyPricing()
	}

	newInput := base.Input
	if o.InputCostPerToken != nil {
		newInput = *o.InputCostPerToken
	}

	// Determine if cache values should be scaled proportionally when input changes.
	shouldScale := o.InputCostPerToken != nil && base.Input > 0 && !base.CacheReadExplicit
	scale := 1.0
	if shouldScale {
		scale = newInput / base.Input
	}

	cacheCreate := base.CacheCreate
	if o.CacheCreationInputTokenCost != nil {
		cacheCreate = *o.CacheCreationInputTokenCost
	} else if shouldScale && base.CacheCreate > 0 {
		cacheCreate = base.CacheCreate * scale
	}

	cacheRead := base.CacheRead
	if o.CacheReadInputTokenCost != nil {
		cacheRead = *o.CacheReadInputTokenCost
	} else if shouldScale && base.CacheRead > 0 {
		cacheRead = base.CacheRead * scale
	}

	cacheCreateAbove200K := base.CacheCreateAbove200K
	if o.CacheCreationAbove200K != nil {
		cacheCreateAbove200K = o.CacheCreationAbove200K
	} else if shouldScale && base.CacheCreateAbove200K != nil {
		v := *base.CacheCreateAbove200K * scale
		cacheCreateAbove200K = &v
	}

	cacheReadAbove200K := base.CacheReadAbove200K
	if o.CacheReadAbove200K != nil {
		cacheReadAbove200K = o.CacheReadAbove200K
	} else if shouldScale && base.CacheReadAbove200K != nil {
		v := *base.CacheReadAbove200K * scale
		cacheReadAbove200K = &v
	}

	fastMultiplier := base.FastMultiplier
	if o.FastMultiplier != nil {
		fastMultiplier = *o.FastMultiplier
	}

	p := Pricing{
		Input:                newInput,
		Output:               base.Output,
		CacheCreate:          cacheCreate,
		CacheRead:            cacheRead,
		CacheReadExplicit:    o.CacheReadInputTokenCost != nil || base.CacheReadExplicit,
		InputAbove200K:       base.InputAbove200K,
		OutputAbove200K:      base.OutputAbove200K,
		CacheCreateAbove200K: cacheCreateAbove200K,
		CacheReadAbove200K:   cacheReadAbove200K,
		FastMultiplier:       fastMultiplier,
	}
	if o.OutputCostPerToken != nil {
		p.Output = *o.OutputCostPerToken
	}
	if o.InputCostPerTokenAbove200K != nil {
		p.InputAbove200K = o.InputCostPerTokenAbove200K
	}
	if o.OutputCostPerTokenAbove200K != nil {
		p.OutputAbove200K = o.OutputCostPerTokenAbove200K
	}

	m.entries[model] = p
	if o.MaxInputTokens != nil {
		m.contextLimits[model] = *o.MaxInputTokens
	}
}
