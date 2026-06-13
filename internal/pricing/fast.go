package pricing

import (
	"strings"
)

// FastMultiplierOverrides provides pre-configured fast multipliers for specific models.
type FastMultiplierOverrides struct {
	Exact            map[string]float64 `json:"exact"`
	NormalizedPrefix map[string]float64 `json:"normalizedPrefix"`
}

func loadFastMultiplierOverrides() *FastMultiplierOverrides {
	return &FastMultiplierOverrides{
		Exact: map[string]float64{
			"gpt-5.5":       2.5,
			"gpt-5.4":       2.0,
			"gpt-5.3-codex": 2.0,
		},
		NormalizedPrefix: map[string]float64{
			"claude-opus-4-6": 6.0,
			"claude-opus-4-7": 6.0,
			"claude-opus-4-8": 2.0,
		},
	}
}

// MultiplierFor returns the fast multiplier for a model, or 0 if none.
func (o *FastMultiplierOverrides) MultiplierFor(model string) float64 {
	if mul, ok := o.Exact[model]; ok {
		return mul
	}
	normalized := strings.NewReplacer(".", "-", "@", "-").Replace(model)
	for _, part := range strings.Split(normalized, "/") {
		for _, part := range strings.Split(part, ":") {
			if mul := o.multiplierForPart(part); mul != 0 {
				return mul
			}
		}
	}
	return 0
}

func (o *FastMultiplierOverrides) multiplierForPart(part string) float64 {
	for base, mul := range o.NormalizedPrefix {
		if matchesModelSuffix(part, base) {
			return mul
		}
	}
	return 0
}

func matchesModelSuffix(part, base string) bool {
	idx := strings.LastIndex(part, base)
	if idx < 0 {
		return false
	}
	suffix := part[idx:]
	return suffix == base || (len(suffix) > len(base) && suffix[len(base)] == '-')
}
