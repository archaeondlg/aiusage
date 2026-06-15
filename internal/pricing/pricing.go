// Package pricing provides model pricing lookup from local config with hardcoded defaults.
package pricing

import (
	"encoding/json"
	"math"
	"strings"
)

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Pricing struct
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

// Pricing holds per-model token pricing information.
type Pricing struct {
	Input               float64
	Output              float64
	CacheCreate         float64
	CacheRead           float64
	CacheReadExplicit   bool
	InputAbove200K      *float64
	OutputAbove200K     *float64
	CacheCreateAbove200K *float64
	CacheReadAbove200K  *float64
	FastMultiplier      float64
}

// EmptyPricing returns a zeroed pricing struct with FastMultiplier=1.
func EmptyPricing() Pricing {
	return Pricing{FastMultiplier: 1.0}
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// PricingMap - the central pricing registry
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

// PricingProvider abstracts model pricing lookup.
type PricingProvider interface {
	Find(model string) *Pricing
}

// PricingMap holds all known model pricing and context limits.
type PricingMap struct {
	entries       map[string]Pricing
	contextLimits map[string]uint64
}

// NewPricingMap creates an empty pricing registry.
func NewPricingMap() *PricingMap {
	return &PricingMap{
		entries:       make(map[string]Pricing),
		contextLimits: make(map[string]uint64),
	}
}

// Find looks up pricing for a model, falling back to aliases and fuzzy matching.
func (m *PricingMap) Find(model string) *Pricing {
	if p := m.findEntryOrAlias(model); p != nil {
		return p
	}
	return nil
}

// ContextLimit returns the max input tokens context window for a model.
func (m *PricingMap) ContextLimit(model string) *uint64 {
	if limit := m.contextLimitEntryOrAlias(model); limit != nil {
		return limit
	}
	return nil
}

// findEntryOrAlias checks exact match then pricing alias.
func (m *PricingMap) findEntryOrAlias(model string) *Pricing {
	if p := m.findEntry(model); p != nil {
		return p
	}
	if alias, ok := pricingAlias(model); ok {
		if p := m.findEntry(alias); p != nil {
			return p
		}
	}
	return nil
}

// findEntry does fuzzy matching against known model keys.
func (m *PricingMap) findEntry(model string) *Pricing {
	if p, ok := m.entries[model]; ok {
		return &p
	}
	normalized := normalizedPricingKey(model)
	var best *Pricing
	bestLen := 0
	for candidate, pricing := range m.entries {
		if pricingKeyMatches(candidate, model, normalized) {
			if len(candidate) > bestLen {
				bestLen = len(candidate)
				p := pricing
				best = &p
			}
		}
	}
	return best
}

func (m *PricingMap) contextLimitEntryOrAlias(model string) *uint64 {
	if limit := m.contextLimitEntry(model); limit != nil {
		return limit
	}
	if alias, ok := pricingAlias(model); ok {
		if limit := m.contextLimitEntry(alias); limit != nil {
			return limit
		}
	}
	return nil
}

func (m *PricingMap) contextLimitEntry(model string) *uint64 {
	if limit, ok := m.contextLimits[model]; ok {
		return &limit
	}
	normalized := normalizedPricingKey(model)
	var best *uint64
	bestLen := 0
	for candidate, limit := range m.contextLimits {
		if pricingKeyMatches(candidate, model, normalized) {
			if len(candidate) > bestLen {
				bestLen = len(candidate)
				l := limit
				best = &l
			}
		}
	}
	return best
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Tiered cost calculation (mirrors Rust cost::tiered_cost)
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

// TieredCost calculates cost with 200K token threshold pricing.
func TieredCost(tokens uint64, rate float64, above200k *float64) float64 {
	if tokens <= 200_000 || above200k == nil {
		return float64(tokens) * rate
	}
	return 200_000*rate + float64(tokens-200_000)*(*above200k)
}

// CalculateCost computes the estimated USD cost for a single usage message.
func CalculateCost(
	inputTokens, outputTokens, cacheCreationTokens, cacheReadTokens uint64,
	model string,
	pricing PricingProvider,
) float64 {
	if pricing == nil {
		return 0
	}
	p := pricing.Find(model)
	if p == nil {
		return 0
	}

	inputCost := TieredCost(inputTokens, p.Input, p.InputAbove200K)
	outputCost := TieredCost(outputTokens, p.Output, p.OutputAbove200K)
	cacheCreateCost := TieredCost(cacheCreationTokens, p.CacheCreate, p.CacheCreateAbove200K)
	cacheReadCost := TieredCost(cacheReadTokens, p.CacheRead, p.CacheReadAbove200K)

	total := inputCost + outputCost + cacheCreateCost + cacheReadCost

	// Apply fast multiplier rounding to nearest hundred-millionth.
	if p.FastMultiplier != 1.0 {
		total = math.Round(total*p.FastMultiplier*1e8) / 1e8
	}
	return total
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// JSON loading helpers
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

// loadJSON parses LiteLLM pricing JSON (compact or full format).
func (m *PricingMap) loadJSON(data []byte, fastOverrides *FastMultiplierOverrides) int {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return 0
	}
	loaded := 0
	for model, value := range raw {
		p, ok := parseLiteLLMPricing(value)
		if !ok || p.Input == nil || p.Output == nil {
			continue
		}
		fastMul := fastOverrides.MultiplierFor(model)
		if p.Fast != nil {
			fastMul = *p.Fast
		}
		if fastMul == 0 {
			fastMul = 1.0
		}
		cacheCreate := *p.Input * 1.25
		if p.CacheCreate != nil {
			cacheCreate = *p.CacheCreate
		}
		cacheRead := *p.Input * 0.1
		if p.CacheRead != nil {
			cacheRead = *p.CacheRead
		}
		cacheReadExplicit := p.CacheRead != nil

		pr := Pricing{
			Input:             *p.Input,
			Output:            *p.Output,
			CacheCreate:       cacheCreate,
			CacheRead:         cacheRead,
			CacheReadExplicit: cacheReadExplicit,
			InputAbove200K:    p.InputAbove200K,
			OutputAbove200K:   p.OutputAbove200K,
			CacheCreateAbove200K: p.CacheCreateAbove200K,
			CacheReadAbove200K:  p.CacheReadAbove200K,
			FastMultiplier:    fastMul,
		}
		m.entries[model] = pr
		if p.ContextLimit != nil {
			m.contextLimits[model] = *p.ContextLimit
		}
		loaded++
	}
	return loaded
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Alias mapping (mirrors Rust pricing_alias)
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

func pricingAlias(model string) (string, bool) {
	switch model {
	case "gpt-5.3-spark":
		return "gpt-5.3-codex-spark", true
	default:
		return "", false
	}
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Model key normalization (mirrors Rust normalized_pricing_key)
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

const modelDateSuffixDigits = 8

func normalizedPricingKey(value string) string {
	if strings.ContainsAny(value, ".@") {
		replacer := strings.NewReplacer(".", "-", "@", "-")
		return replacer.Replace(value)
	}
	return value
}

// pricingKeyMatches implements Rust's pricing_key_matches.
func pricingKeyMatches(candidate, model, normalizedModel string) bool {
	if containsPricingKey(model, candidate) || containsPricingKey(candidate, model) {
		return true
	}
	normalizedCandidate := normalizedPricingKey(candidate)
	return containsPricingKey(normalizedModel, normalizedCandidate) ||
		containsPricingKey(normalizedCandidate, normalizedModel)
}

func containsPricingKey(value, key string) bool {
	start := 0
	for {
		idx := strings.Index(value[start:], key)
		if idx < 0 {
			return false
		}
		idx += start

		// Check boundary before key.
		if idx > 0 && isAlphanumeric(value[idx-1]) {
			start = idx + len(key)
			continue
		}

		// Check suffix allows this match.
		suffix := value[idx+len(key):]
		if suffixAllowsPricingKeyMatch(key, suffix) {
			return true
		}
		start = idx + len(key)
	}
}

func isAlphanumeric(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

func suffixAllowsPricingKeyMatch(key, suffix string) bool {
	if len(suffix) == 0 {
		return true
	}
	if isAlphanumeric(suffix[0]) {
		return false
	}
	return !suffixStartsWithNumericModelVersion(key, suffix)
}

func suffixStartsWithNumericModelVersion(key, suffix string) bool {
	if len(key) == 0 || !isDigit(key[len(key)-1]) {
		return false
	}
	if len(suffix) == 0 || (suffix[0] != '-' && suffix[0] != '.') {
		return false
	}
	rest := suffix[1:]
	digitLen := 0
	for digitLen < len(rest) && isDigit(rest[digitLen]) {
		digitLen++
	}
	if digitLen == 0 {
		return false
	}
	if digitLen == modelDateSuffixDigits {
		afterDigits := byte(0)
		if digitLen < len(rest) {
			afterDigits = rest[digitLen]
		}
		if afterDigits == 0 || !isAlphanumeric(afterDigits) {
			return true
		}
	}
	return true
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}
