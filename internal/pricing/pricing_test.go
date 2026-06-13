package pricing

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDefault(t *testing.T) {
	m := LoadDefaultPricing()
	assert.Greater(t, len(m.entries), 0, "default pricing should not be empty")
}

func TestFindClaudeModels(t *testing.T) {
	m := LoadDefaultPricing()

	tests := []struct {
		model string
		want  bool
	}{
		{"claude-sonnet-4-6", true},
		{"claude-opus-4-6", true},
		{"claude-haiku-4-5", true},
		{"gpt-5.4", true},
		{"gpt-5.5", true},
		{"nonexistent-model-xyz", false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			p := m.Find(tt.model)
			if tt.want {
				assert.NotNil(t, p, "expected pricing for %s", tt.model)
			} else {
				assert.Nil(t, p, "expected no pricing for %s", tt.model)
			}
		})
	}
}

func TestBuiltinPricingValues(t *testing.T) {
	m := LoadDefaultPricing()

	p := m.Find("claude-opus-4-8")
	require.NotNil(t, p)
	assert.Equal(t, 5e-6, p.Input)
	assert.Equal(t, 25e-6, p.Output)
	assert.Equal(t, 6.25e-6, p.CacheCreate)
	assert.True(t, p.CacheReadExplicit)

	p = m.Find("gpt-5.4")
	require.NotNil(t, p)
	assert.Equal(t, 2.5e-6, p.Input)
}

func TestContextLimits(t *testing.T) {
	m := LoadDefaultPricing()

	tests := []struct {
		model string
		limit uint64
	}{
		{"claude-opus-4-8", 1_000_000},
		{"claude-sonnet-4", 200_000},
		{"gpt-5.5", 1_050_000},
		{"moonshot/kimi-k2.5", 262_144},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			limit := m.ContextLimit(tt.model)
			require.NotNil(t, limit)
			assert.Equal(t, tt.limit, *limit)
		})
	}
}

func TestTieredCost(t *testing.T) {
	// Below threshold.
	cost := TieredCost(100_000, 1e-6, nil)
	assert.InDelta(t, 0.1, cost, 0.0001)

	// Above threshold.
	above200k := 2e-6
	cost = TieredCost(300_000, 1e-6, &above200k)
	// 200000 * 1e-6 + 100000 * 2e-6 = 0.2 + 0.2 = 0.4
	assert.InDelta(t, 0.4, cost, 0.0001)
}

func TestPricingKeyMatches(t *testing.T) {
	// Exact match.
	assert.True(t, pricingKeyMatches("claude-opus-4-6", "claude-opus-4-6", "claude-opus-4-6"))

	// Candidate contains model key.
	assert.True(t, pricingKeyMatches("us.anthropic.claude-opus-4-6-v1:0", "claude-opus-4-6", "claude-opus-4-6"))

	// Candidate contains model at boundary.
	assert.True(t, pricingKeyMatches("us.anthropic.claude-sonnet-4", "claude-sonnet-4", "claude-sonnet-4"))

	// Different version.
	assert.False(t, pricingKeyMatches("claude-opus-4-8", "claude-opus-4-6", "claude-opus-4-6"))
}

func TestLoadJSONCompactFormat(t *testing.T) {
	m := NewPricingMap()
	count := m.loadJSON([]byte(`{"test-model":{"i":0.000001,"o":0.00001,"cc":0.00000125,"cr":0.0000001,"ctx":123456}}`), loadFastMultiplierOverrides())
	assert.Equal(t, 1, count)

	p := m.Find("test-model")
	require.NotNil(t, p)
	assert.Equal(t, 1e-6, p.Input)
	assert.Equal(t, 10e-6, p.Output)
	assert.Equal(t, 1.25e-6, p.CacheCreate)
	assert.Equal(t, 0.1e-6, p.CacheRead)
	assert.True(t, p.CacheReadExplicit)

	limit := m.ContextLimit("test-model")
	require.NotNil(t, limit)
	assert.Equal(t, uint64(123456), *limit)
}

func TestLoadJSONFullFormat(t *testing.T) {
	m := NewPricingMap()
	count := m.loadJSON([]byte(`{"test-full":{"input_cost_per_token":0.000002,"output_cost_per_token":0.00002,"cache_read_input_token_cost":0.0000002,"max_input_tokens":654321}}`), loadFastMultiplierOverrides())
	assert.Equal(t, 1, count)

	p := m.Find("test-full")
	require.NotNil(t, p)
	assert.Equal(t, 2e-6, p.Input)
	assert.Equal(t, 20e-6, p.Output)
	assert.True(t, p.CacheReadExplicit)
}

func TestPricingOverrides(t *testing.T) {
	m := NewPricingMap()

	// Add base entry.
	m.loadJSON([]byte(`{"base-model":{"i":0.000001,"o":0.00001}}`), loadFastMultiplierOverrides())

	// Apply override.
	inputOverride := 5e-6
	outputOverride := 20e-6
	m.applyOverrides(map[string]*PricingOverride{
		"base-model": {
			InputCostPerToken:  &inputOverride,
			OutputCostPerToken: &outputOverride,
		},
	})

	p := m.Find("base-model")
	require.NotNil(t, p)
	assert.Equal(t, 5e-6, p.Input)
	assert.Equal(t, 20e-6, p.Output)
}

func TestCalculateCost(t *testing.T) {
	m := LoadDefaultPricing()

	cost := CalculateCost(100, 50, 10, 5, "claude-sonnet-4-6", m)
	// input: 100 * 3e-6 = 0.0003
	// output: 50 * 15e-6 = 0.00075
	// cache_create: 10 * 3.75e-6 = 0.0000375
	// cache_read: 5 * 0.3e-6 = 0.0000015
	// total = 0.001089
	assert.InDelta(t, 0.001089, cost, 0.000001)

	// Unknown model should return 0.
	cost = CalculateCost(100, 50, 10, 5, "nonexistent-model", m)
	assert.Equal(t, 0.0, cost)
}

func TestAliasResolve(t *testing.T) {
	alias, ok := pricingAlias("gpt-5.3-spark")
	assert.True(t, ok)
	assert.Equal(t, "gpt-5.3-codex-spark", alias)

	_, ok = pricingAlias("nonexistent")
	assert.False(t, ok)
}
