package opencode

import (
	"encoding/json"
	"strings"

	"github.com/archhaeondlg/aiusage/internal/dateutil"
	"github.com/archhaeondlg/aiusage/internal/pricing"
	"github.com/archhaeondlg/aiusage/internal/types"
)

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Message JSON schema
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

type opencodeMessage struct {
	ID         string           `json:"id"`
	SessionID  string           `json:"sessionID"`
	ProviderID string           `json:"providerID"`
	ModelID    string           `json:"modelID"`
	Time       *opencodeTime    `json:"time"`
	Tokens     *opencodeTokens  `json:"tokens"`
	Cost       *float64         `json:"cost"`
}

type opencodeTime struct {
	Created int64 `json:"created"` // Unix milliseconds
}

type opencodeTokens struct {
	Input float64          `json:"input"`
	Output float64          `json:"output"`
	Total float64          `json:"total"`
	Cache *opencodeCache   `json:"cache"`
}

type opencodeCache struct {
	Read  float64 `json:"read"`
	Write float64 `json:"write"`
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Parser
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

// messageToEntry converts a raw JSON OpenCode message to a LoadedEntry.
// Equivalent to Rust message_value_to_entry.
func messageToEntry(
	value json.RawMessage,
	id, sessionID string,
	tz interface{},
	pm *pricing.PricingMap,
) *types.LoadedEntry {
	var msg opencodeMessage
	if json.Unmarshal(value, &msg) == nil {
		return messageToEntryParsed(&msg, id, sessionID, tz, pm)
	}
	// Try as raw map for edge cases.
	var raw map[string]json.RawMessage
	if json.Unmarshal(value, &raw) != nil {
		return nil
	}
	return messageToEntryRaw(raw, id, sessionID, tz, pm)
}

func messageToEntryParsed(
	msg *opencodeMessage,
	id, sessionID string,
	tz interface{},
	pm *pricing.PricingMap,
) *types.LoadedEntry {
	if msg.Tokens == nil {
		return nil
	}
	t := msg.Tokens

	cacheWrite := uint64(0)
	cacheRead := uint64(0)
	if t.Cache != nil {
		cacheWrite = uint64(t.Cache.Write)
		cacheRead = uint64(t.Cache.Read)
	}
	usage := types.TokenUsageRaw{
		InputTokens:              uint64(t.Input),
		OutputTokens:             uint64(t.Output),
		CacheCreationInputTokens: cacheWrite,
		CacheReadInputTokens:     cacheRead,
	}
	totalTokens := uint64(t.Total)
	usage, extraTotal := applyTotalTokenFallback(usage, totalTokens)

	if usage.InputTokens == 0 && usage.OutputTokens == 0 &&
		usage.CacheCreationInputTokens == 0 && usage.CacheReadInputTokens == 0 &&
		extraTotal == 0 {
		return nil
	}

	model := msg.ModelID
	if model == "" {
		return nil
	}
	provider := msg.ProviderID
	if provider == "" {
		return nil
	}

	ts := dateutil.FormatRFC3339Millis(dateutil.TimeFromMillis(msg.Time.Created))

	msgID := id
	if msgID == "" {
		msgID = msg.ID
	}
	sessID := sessionID
	if sessID == "" {
		sessID = msg.SessionID
	}
	if sessID == "" {
		sessID = "unknown"
	}

	cost := msg.Cost
	costUSD := 0.0
	if cost != nil {
		costUSD = *cost
	}

	costUsage := types.TokenUsageRaw{
		InputTokens:              usage.InputTokens,
		OutputTokens:             usage.OutputTokens + extraTotal,
		CacheCreationInputTokens: usage.CacheCreationInputTokens,
		CacheReadInputTokens:     usage.CacheReadInputTokens,
	}

	calculatedCost := calculateOpenCodeCost(model, provider, costUsage, costUSD, pm)
	missingPricing := missingOpenCodePricing(model, provider, costUsage, costUSD, pm)

	return &types.LoadedEntry{
		Data: types.UsageEntry{
			SessionID: &sessID,
			Timestamp: ts,
			Message: types.UsageMessage{
				Usage: usage,
				Model: &model,
				ID:    &msgID,
			},
			CostUSD: cost,
		},
		Timestamp:           dateutil.TimeFromMillis(msg.Time.Created),
		Date:                formatDate(ts, tz),
		Project:             "opencode",
		SessionID:           sessID,
		ProjectPath:         "OpenCode",
		Cost:                calculatedCost,
		ExtraTotalTokens:    extraTotal,
		Model:               &model,
		MissingPricingModel: missingPricing,
	}
}

func messageToEntryRaw(
	raw map[string]json.RawMessage,
	id, sessionID string,
	tz interface{},
	pm *pricing.PricingMap,
) *types.LoadedEntry {
	// Re-marshal and parse normally.
	data, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var msg opencodeMessage
	if json.Unmarshal(data, &msg) != nil {
		return nil
	}
	return messageToEntryParsed(&msg, id, sessionID, tz, pm)
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Cost calculation
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

func calculateOpenCodeCost(
	model, provider string,
	usage types.TokenUsageRaw,
	costUSD float64,
	pm *pricing.PricingMap,
) float64 {
	if costUSD > 0 {
		return costUSD
	}
	if pm == nil {
		return 0
	}
	for _, candidate := range openCodeModelCandidates(model, provider) {
		cost := pricing.CalculateCost(
			usage.InputTokens,
			usage.OutputTokens,
			usage.CacheCreationInputTokens,
			usage.CacheReadInputTokens,
			candidate,
			pm,
		)
		if cost > 0 {
			return cost
		}
	}
	return 0
}

func missingOpenCodePricing(
	model, provider string,
	usage types.TokenUsageRaw,
	costUSD float64,
	pm *pricing.PricingMap,
) *string {
	if costUSD > 0 || pm == nil {
		return nil
	}
	total := usage.InputTokens + usage.OutputTokens + usage.CacheCreationInputTokens + usage.CacheReadInputTokens
	if total == 0 {
		return nil
	}
	// Check if any candidate has pricing.
	for _, candidate := range openCodeModelCandidates(model, provider) {
		if pm.Find(candidate) != nil {
			return nil
		}
	}
	return &model
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Model name resolution (mirrors Rust)
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

func openCodeModelCandidates(model, provider string) []string {
	resolved := resolveOpenCodeModelName(model)
	normalized := normalizeOpenCodeModelName(resolved)

	var base []string
	base = append(base, resolved)
	if normalized != resolved {
		base = append(base, normalized)
	}

	var candidates []string
	candidates = append(candidates, base...)

	if provider != "unknown" {
		providerNorm := strings.ReplaceAll(provider, "-", "_")
		for _, b := range base {
			candidates = append(candidates, providerNorm+"/"+b)
		}
	}
	return dedupStrings(candidates)
}

func resolveOpenCodeModelName(model string) string {
	switch model {
	case "gemini-3-pro-high":
		return "gemini-3-pro-preview"
	case "k2p6":
		return "kimi-k2.6"
	default:
		return model
	}
}

func normalizeOpenCodeModelName(model string) string {
	for _, family := range []string{"claude-haiku-", "claude-opus-", "claude-sonnet-"} {
		if rest, ok := strings.CutPrefix(model, family); ok {
			// Try X.Y format → X-Y
			parts := strings.SplitN(rest, ".", 2)
			if len(parts) == 2 && isAllDigits(parts[0]) && len(parts[1]) > 0 && isDigit(parts[1][0]) {
				return family + parts[0] + "-" + parts[1]
			}
			// Try XY format → X-Y
			if len(rest) >= 2 && isDigit(rest[0]) && isDigit(rest[1]) {
				return family + string(rest[0]) + "-" + rest[1:]
			}
		}
	}
	return model
}

func isDigit(b byte) bool { return b >= '0' && b <= '9' }

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Token helpers
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

func applyTotalTokenFallback(usage types.TokenUsageRaw, totalTokens uint64) (types.TokenUsageRaw, uint64) {
	partsSum := usage.InputTokens + usage.OutputTokens +
		usage.CacheCreationInputTokens + usage.CacheReadInputTokens
	if partsSum > 0 || totalTokens == 0 {
		return usage, 0
	}
	// All parts are zero, put everything into output_tokens as a fallback.
	usage.OutputTokens = totalTokens
	return usage, 0
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Helpers
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

// formatDate extracts date from RFC 3339 timestamp.
func formatDate(ts string, _ interface{}) string {
	if ts == "" {
		return ""
	}
	t, err := dateutil.ParseTimestamp(ts)
	if err != nil {
		return ""
	}
	return dateutil.FormatDate(t, nil)
}

func dedupStrings(s []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, v := range s {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}

func isAllDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

