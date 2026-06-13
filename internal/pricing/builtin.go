package pricing

// defaultPricingMap returns a PricingMap pre-populated with hardcoded model pricing.
// These serve as fallback defaults when no user config is provided.
func defaultPricingMap(fastOverrides *FastMultiplierOverrides) *PricingMap {
	m := NewPricingMap()
	// Claude Opus family.
	m.entries["claude-opus-4-5"] = Pricing{Input: 5e-6, Output: 25e-6, CacheCreate: 6.25e-6, CacheRead: 0.5e-6, CacheReadExplicit: true, FastMultiplier: 1.0}
	m.entries["claude-opus-4-6"] = Pricing{Input: 5e-6, Output: 25e-6, CacheCreate: 6.25e-6, CacheRead: 0.5e-6, CacheReadExplicit: true, FastMultiplier: fastOverrides.MultiplierFor("claude-opus-4-6")}
	m.entries["claude-opus-4-7"] = Pricing{Input: 5e-6, Output: 25e-6, CacheCreate: 6.25e-6, CacheRead: 0.5e-6, CacheReadExplicit: true, FastMultiplier: fastOverrides.MultiplierFor("claude-opus-4-7")}
	m.entries["claude-opus-4-8"] = Pricing{Input: 5e-6, Output: 25e-6, CacheCreate: 6.25e-6, CacheRead: 0.5e-6, CacheReadExplicit: true, FastMultiplier: fastOverrides.MultiplierFor("claude-opus-4-8")}

	// Claude Haiku.
	m.entries["claude-haiku-4-5"] = Pricing{Input: 1e-6, Output: 5e-6, CacheCreate: 1.25e-6, CacheRead: 0.1e-6, CacheReadExplicit: true, FastMultiplier: 1.0}

	// Claude Opus 4.
	m.entries["claude-opus-4"] = Pricing{Input: 15e-6, Output: 75e-6, CacheCreate: 18.75e-6, CacheRead: 1.5e-6, CacheReadExplicit: true, FastMultiplier: 1.0}

	// Claude Sonnet.
	m.entries["claude-sonnet-4-6"] = Pricing{Input: 3e-6, Output: 15e-6, CacheCreate: 3.75e-6, CacheRead: 0.3e-6, CacheReadExplicit: true, FastMultiplier: 1.0}
	m.entries["claude-sonnet-4"] = Pricing{
		Input: 3e-6, Output: 15e-6, CacheCreate: 3.75e-6, CacheRead: 0.3e-6, CacheReadExplicit: true,
		InputAbove200K: ptr(6e-6), OutputAbove200K: ptr(22.5e-6), CacheCreateAbove200K: ptr(7.5e-6), CacheReadAbove200K: ptr(0.6e-6),
		FastMultiplier: 1.0,
	}

	// Claude 3.5 Haiku.
	claude35Haiku := Pricing{Input: 0.8e-6, Output: 4e-6, CacheCreate: 1.0e-6, CacheRead: 0.08e-6, CacheReadExplicit: true, FastMultiplier: 1.0}
	m.entries["claude-3-5-haiku"] = claude35Haiku
	m.entries["claude-3-5-haiku-20241022"] = claude35Haiku

	// Claude 3 models.
	m.entries["claude-3-opus"] = Pricing{Input: 15e-6, Output: 75e-6, CacheCreate: 18.75e-6, CacheRead: 1.5e-6, CacheReadExplicit: true, FastMultiplier: 1.0}
	m.entries["claude-3-sonnet"] = Pricing{Input: 3e-6, Output: 15e-6, CacheCreate: 3.75e-6, CacheRead: 0.3e-6, CacheReadExplicit: true, FastMultiplier: 1.0}
	m.entries["claude-3-haiku"] = Pricing{Input: 0.25e-6, Output: 1.25e-6, CacheCreate: 0.3e-6, CacheRead: 0.03e-6, CacheReadExplicit: true, FastMultiplier: 1.0}

	// GPT-5 family.
	m.entries["gpt-5"] = Pricing{Input: 1.25e-6, Output: 10e-6, CacheCreate: 1.25e-6, CacheRead: 0.125e-6, CacheReadExplicit: true, FastMultiplier: 1.0}
	m.entries["gpt-5.5"] = Pricing{Input: 5e-6, Output: 30e-6, CacheCreate: 5e-6, CacheRead: 0.5e-6, CacheReadExplicit: true, FastMultiplier: fastOverrides.MultiplierFor("gpt-5.5")}

	// Grok.
	m.entries["grok-4.3"] = Pricing{Input: 1.25e-6, Output: 2.5e-6, CacheCreate: 1.25e-6, CacheRead: 0.125e-6, CacheReadExplicit: false, FastMultiplier: 1.0}

	// Kimi models.
	m.entries["moonshot/kimi-k2.5"] = Pricing{Input: 0.6e-6, Output: 3e-6, CacheCreate: 0.75e-6, CacheRead: 0.1e-6, CacheReadExplicit: true, FastMultiplier: 1.0}
	m.entries["moonshot/kimi-k2.6"] = Pricing{Input: 0.95e-6, Output: 4e-6, CacheCreate: 1.1875e-6, CacheRead: 0.16e-6, CacheReadExplicit: true, FastMultiplier: 1.0}

	// GPT-5.1 models.
	gpt51 := Pricing{Input: 1.25e-6, Output: 10e-6, CacheCreate: 1.25e-6, CacheRead: 0.125e-6, CacheReadExplicit: true, FastMultiplier: 1.0}
	m.entries["gpt-5.1"] = gpt51
	m.entries["gpt-5.1-codex"] = gpt51

	// GPT-5.2/5.3 Codex.
	gpt5Codex := Pricing{Input: 1.75e-6, Output: 14e-6, CacheCreate: 1.75e-6, CacheRead: 0.175e-6, CacheReadExplicit: true, FastMultiplier: 1.0}
	m.entries["gpt-5.2-codex"] = gpt5Codex
	m.entries["gpt-5.2"] = gpt5Codex
	m.entries["gpt-5.3-codex"] = Pricing{Input: 1.75e-6, Output: 14e-6, CacheCreate: 1.75e-6, CacheRead: 0.175e-6, CacheReadExplicit: true, FastMultiplier: fastOverrides.MultiplierFor("gpt-5.3-codex")}

	// GPT-5.4.
	m.entries["gpt-5.4"] = Pricing{Input: 2.5e-6, Output: 15e-6, CacheCreate: 2.5e-6, CacheRead: 0.25e-6, CacheReadExplicit: true, FastMultiplier: fastOverrides.MultiplierFor("gpt-5.4")}
	m.entries["gpt-5.4-mini"] = Pricing{Input: 0.75e-6, Output: 4.5e-6, CacheCreate: 0.75e-6, CacheRead: 0.075e-6, CacheReadExplicit: true, FastMultiplier: 1.0}
	m.entries["gpt-5.4-nano"] = Pricing{Input: 0.2e-6, Output: 1.25e-6, CacheCreate: 0.2e-6, CacheRead: 0.02e-6, CacheReadExplicit: true, FastMultiplier: 1.0}

	// GLM models (Z.AI).
	glmBase := Pricing{Input: 0.6e-6, Output: 2.2e-6, CacheCreate: 0.0, CacheRead: 0.11e-6, CacheReadExplicit: true, FastMultiplier: 1.0}
	m.entries["glm-4.5"] = glmBase
	m.entries["zai/glm-4.5"] = glmBase
	m.entries["zai/glm-4.5-x"] = Pricing{Input: 2.2e-6, Output: 8.9e-6, CacheCreate: 0.0, CacheRead: 0.45e-6, CacheReadExplicit: true, FastMultiplier: 1.0}
	m.entries["zai/glm-4.5-air"] = Pricing{Input: 0.2e-6, Output: 1.1e-6, CacheCreate: 0.0, CacheRead: 0.03e-6, CacheReadExplicit: true, FastMultiplier: 1.0}
	m.entries["zai/glm-4.5-airx"] = Pricing{Input: 1.1e-6, Output: 4.5e-6, CacheCreate: 0.0, CacheRead: 0.22e-6, CacheReadExplicit: true, FastMultiplier: 1.0}
	m.entries["zai/glm-4.5v"] = Pricing{Input: 0.6e-6, Output: 1.8e-6, CacheCreate: 0.0, CacheRead: 0.11e-6, CacheReadExplicit: true, FastMultiplier: 1.0}
	m.entries["zai/glm-4-32b-0414-128k"] = Pricing{Input: 0.1e-6, Output: 0.1e-6, CacheCreate: 0.0, CacheRead: 0.0, CacheReadExplicit: true, FastMultiplier: 1.0}
	m.entries["zai/glm-4.5-flash"] = Pricing{Input: 0.0, Output: 0.0, CacheCreate: 0.0, CacheRead: 0.0, CacheReadExplicit: true, FastMultiplier: 1.0}
	m.entries["glm-4.6"] = glmBase
	m.entries["glm-4.7"] = glmBase
	m.entries["glm-5"] = Pricing{Input: 1.0e-6, Output: 3.2e-6, CacheCreate: 0.0, CacheRead: 0.2e-6, CacheReadExplicit: true, FastMultiplier: 1.0}
	m.entries["glm-5-turbo"] = Pricing{Input: 1.2e-6, Output: 4.0e-6, CacheCreate: 0.0, CacheRead: 0.24e-6, CacheReadExplicit: true, FastMultiplier: 1.0}
	m.entries["glm-5.1"] = Pricing{Input: 1.4e-6, Output: 4.4e-6, CacheCreate: 0.0, CacheRead: 0.26e-6, CacheReadExplicit: true, FastMultiplier: 1.0}

	// Context limits.
	m.contextLimits["gpt-5.5"] = 1_050_000
	m.contextLimits["grok-4.3"] = 1_000_000
	m.contextLimits["gpt-5.4"] = 1_050_000
	for _, model := range []string{"claude-opus-4-8", "claude-opus-4-7", "claude-opus-4-6", "claude-sonnet-4-6"} {
		m.contextLimits[model] = 1_000_000
	}
	m.contextLimits["moonshot/kimi-k2.5"] = 262_144
	m.contextLimits["moonshot/kimi-k2.6"] = 262_144
	for _, model := range []string{
		"claude-opus-4-5", "claude-haiku-4-5", "claude-opus-4", "claude-sonnet-4",
		"claude-3-5-haiku", "claude-3-5-haiku-20241022", "claude-3-opus", "claude-3-sonnet", "claude-3-haiku",
	} {
		m.contextLimits[model] = 200_000
	}

	// DeepSeek models.
	m.entries["deepseek-v4-pro"] = Pricing{
		Input: 4.35e-7, Output: 8.7e-7,
		CacheCreate: 5.4375e-7, CacheRead: 3.625e-9,
		CacheReadExplicit: true, FastMultiplier: 1.0,
	}
	m.contextLimits["deepseek-v4-pro"] = 1_048_576

	// Qwen3.6 models.
	qwen36plus := Pricing{
		Input: 3.25e-7, Output: 1.95e-6,
		CacheCreate: 3.5e-7, CacheRead: 2.8e-8,
		CacheReadExplicit: true, FastMultiplier: 1.0,
	}
	m.entries["qwen3.6-plus"] = qwen36plus
	m.entries["qwen3.6-plus-2026-04-02"] = qwen36plus
	m.contextLimits["qwen3.6-plus"] = 1_000_000
	m.contextLimits["qwen3.6-plus-2026-04-02"] = 1_000_000

	// Qwen3 Coder models.
	m.entries["qwen3-coder-plus"] = Pricing{
		Input: 1.0e-6, Output: 5.0e-6,
		CacheCreate: 1.25e-6, CacheRead: 1.0e-7,
		CacheReadExplicit: true, FastMultiplier: 1.0,
	}
	m.contextLimits["qwen3-coder-plus"] = 1_000_000

	m.entries["qwen/qwen3-coder-480b-a35b-instruct"] = Pricing{
		Input: 2.9e-7, Output: 1.2e-6,
		CacheCreate: 3.625e-7, CacheRead: 2.9e-8,
		CacheReadExplicit: true, FastMultiplier: 1.0,
	}
	m.contextLimits["qwen/qwen3-coder-480b-a35b-instruct"] = 262_144

	return m
}

func ptr[T any](v T) *T { return &v }
