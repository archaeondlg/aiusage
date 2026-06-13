package pricing

import (
	"encoding/json"
)

// Raw pricing structures for LiteLLM JSON deserialization.

// LiteLLmPricingRaw is the full-format LiteLLM JSON entry.
type LiteLLmPricingRaw struct {
	InputCostPerToken               *float64 `json:"input_cost_per_token"`
	OutputCostPerToken              *float64 `json:"output_cost_per_token"`
	CacheCreationInputTokenCost      *float64 `json:"cache_creation_input_token_cost"`
	CacheReadInputTokenCost          *float64 `json:"cache_read_input_token_cost"`
	InputCostPerTokenAbove200K       *float64 `json:"input_cost_per_token_above_200k_tokens"`
	OutputCostPerTokenAbove200K       *float64 `json:"output_cost_per_token_above_200k_tokens"`
	CacheCreationAbove200K           *float64 `json:"cache_creation_input_token_cost_above_200k_tokens"`
	CacheReadAbove200K               *float64 `json:"cache_read_input_token_cost_above_200k_tokens"`
	MaxInputTokens                   *uint64  `json:"max_input_tokens"`
	ProviderSpecificEntry            *struct {
		Fast *float64 `json:"fast"`
	} `json:"provider_specific_entry"`
	// Compact format fields (single-letter keys)
	I   *float64 `json:"i"`
	O   *float64 `json:"o"`
	CC  *float64 `json:"cc"`
	CR  *float64 `json:"cr"`
	IA  *float64 `json:"ia"`
	OA  *float64 `json:"oa"`
	CCA *float64 `json:"cca"`
	CRA *float64 `json:"cra"`
	Ctx *uint64  `json:"ctx"`
	FastC *float64 `json:"fast"`
}

// Parsed pricing data extracted from either format.
type parsedPricing struct {
	Input              *float64
	Output             *float64
	CacheCreate        *float64
	CacheRead          *float64
	InputAbove200K     *float64
	OutputAbove200K    *float64
	CacheCreateAbove200K *float64
	CacheReadAbove200K *float64
	ContextLimit       *uint64
	Fast               *float64
}

// parseLiteLLMPricing handles both compact and full LiteLLM JSON formats.
func parseLiteLLMPricing(raw json.RawMessage) (*parsedPricing, bool) {
	var entry LiteLLmPricingRaw
	if err := json.Unmarshal(raw, &entry); err != nil {
		return nil, false
	}

	// Detect compact format: has both "i" and "o" fields.
	if entry.I != nil && entry.O != nil {
		p := &parsedPricing{
			Input:              entry.I,
			Output:             entry.O,
			CacheCreate:        entry.CC,
			CacheRead:          entry.CR,
			InputAbove200K:     entry.IA,
			OutputAbove200K:    entry.OA,
			CacheCreateAbove200K: entry.CCA,
			CacheReadAbove200K: entry.CRA,
			ContextLimit:       entry.Ctx,
			Fast:               entry.FastC,
		}
		if p.Input != nil && p.Output != nil {
			return p, true
		}
		return nil, false
	}

	// Full format.
	if entry.InputCostPerToken == nil || entry.OutputCostPerToken == nil {
		return nil, false
	}
	var fast *float64
	if entry.ProviderSpecificEntry != nil {
		fast = entry.ProviderSpecificEntry.Fast
	}
	return &parsedPricing{
		Input:              entry.InputCostPerToken,
		Output:             entry.OutputCostPerToken,
		CacheCreate:        entry.CacheCreationInputTokenCost,
		CacheRead:          entry.CacheReadInputTokenCost,
		InputAbove200K:     entry.InputCostPerTokenAbove200K,
		OutputAbove200K:    entry.OutputCostPerTokenAbove200K,
		CacheCreateAbove200K: entry.CacheCreationAbove200K,
		CacheReadAbove200K: entry.CacheReadAbove200K,
		ContextLimit:       entry.MaxInputTokens,
		Fast:               fast,
	}, true
}
