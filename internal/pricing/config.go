package pricing

// PricingEntry is the user-facing config format for model pricing in aiusage.json.
// All price values are in dollars per token (e.g. 5e-6 = $5 per 1M input tokens).
type PricingEntry struct {
	Input           float64  `json:"input"`
	Output          float64  `json:"output"`
	CacheCreate     *float64 `json:"cacheCreate,omitempty"`
	CacheRead       *float64 `json:"cacheRead,omitempty"`
	InputAbove200K  *float64 `json:"inputAbove200K,omitempty"`
	OutputAbove200K *float64 `json:"outputAbove200K,omitempty"`
	FastMultiplier  *float64 `json:"fastMultiplier,omitempty"`
	ContextLimit    *uint64  `json:"contextLimit,omitempty"`
}

// toPricing converts a PricingEntry to the internal Pricing struct.
func (e PricingEntry) toPricing(fastMul float64) Pricing {
	cacheCreate := e.Input * 1.25
	if e.CacheCreate != nil {
		cacheCreate = *e.CacheCreate
	}
	cacheRead := e.Input * 0.1
	if e.CacheRead != nil {
		cacheRead = *e.CacheRead
	}
	cacheReadExplicit := e.CacheRead != nil

	fm := fastMul
	if e.FastMultiplier != nil {
		fm = *e.FastMultiplier
	}
	if fm == 0 {
		fm = 1.0
	}

	return Pricing{
		Input:                e.Input,
		Output:               e.Output,
		CacheCreate:          cacheCreate,
		CacheRead:            cacheRead,
		CacheReadExplicit:    cacheReadExplicit,
		InputAbove200K:       e.InputAbove200K,
		OutputAbove200K:      e.OutputAbove200K,
		CacheCreateAbove200K: nil,
		CacheReadAbove200K:   nil,
		FastMultiplier:       fm,
	}
}

// LoadPricing creates a PricingMap from user config pricing, falling back to
// hardcoded builtin defaults for any model not present in the config.
func LoadPricing(configPricing map[string]PricingEntry) *PricingMap {
	m := NewPricingMap()

	// 1. Load from user config file.
	fastOverrides := loadFastMultiplierOverrides()
	for model, entry := range configPricing {
		fastMul := fastOverrides.MultiplierFor(model)
		if entry.FastMultiplier != nil {
			fastMul = *entry.FastMultiplier
		}
		p := entry.toPricing(fastMul)
		m.entries[model] = p
		if entry.ContextLimit != nil {
			m.contextLimits[model] = *entry.ContextLimit
		}
	}

	// 2. Fill gaps with builtin defaults (only for models NOT in config).
	defaults := defaultPricingMap(fastOverrides)
	for model, p := range defaults.entries {
		if _, exists := m.entries[model]; !exists {
			m.entries[model] = p
		}
	}
	for model, limit := range defaults.contextLimits {
		if _, exists := m.contextLimits[model]; !exists {
			m.contextLimits[model] = limit
		}
	}

	return m
}

// LoadDefaultPricing creates a PricingMap using only builtin defaults (no config file).
func LoadDefaultPricing() *PricingMap {
	return defaultPricingMap(loadFastMultiplierOverrides())
}
