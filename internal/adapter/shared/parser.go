package shared

import (
	"encoding/json"
	"time"

	"github.com/archhaeondlg/aiusage/internal/pricing"
	"github.com/archhaeondlg/aiusage/internal/types"
)

// ParseGenericEntry parses a JSONL line into a LoadedEntry using the common
// Claude Code / coding agent format.
func ParseGenericEntry(line []byte, pm *pricing.PricingMap) *types.LoadedEntry {
	// Skip lines without usage data.
	if len(line) < 20 {
		return nil
	}

	var raw struct {
		Timestamp  string  `json:"timestamp"`
		SessionID  *string `json:"sessionId"`
		Message    struct {
			Model *string `json:"model"`
			Usage *struct {
				InputTokens              int64 `json:"input_tokens"`
				OutputTokens             int64 `json:"output_tokens"`
				CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
				CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
			} `json:"usage"`
		} `json:"message"`
		CostUSD *float64 `json:"costUSD"`
	}

	if err := json.Unmarshal(line, &raw); err != nil {
		return nil
	}
	if raw.Message.Usage == nil {
		return nil
	}
	u := raw.Message.Usage
	if u.InputTokens == 0 && u.OutputTokens == 0 &&
		u.CacheCreationInputTokens == 0 && u.CacheReadInputTokens == 0 {
		return nil
	}

	ts, err := time.Parse(time.RFC3339, raw.Timestamp)
	if err != nil {
		return nil
	}
	date := ts.Format("2006-01-02")

	sessionID := "unknown"
	if raw.SessionID != nil {
		sessionID = *raw.SessionID
	}

	cost := 0.0
	if raw.CostUSD != nil {
		cost = *raw.CostUSD
	}
	if cost == 0 && pm != nil && raw.Message.Model != nil {
		cost = pricing.CalculateCost(
			uint64(u.InputTokens),
			uint64(u.OutputTokens),
			uint64(u.CacheCreationInputTokens),
			uint64(u.CacheReadInputTokens),
			*raw.Message.Model,
			pm,
		)
	}

	var missingModel *string
	if pm != nil && raw.Message.Model != nil && pm.Find(*raw.Message.Model) == nil {
		missingModel = raw.Message.Model
	}

	return &types.LoadedEntry{
		Timestamp: ts,
		Date:      date,
		SessionID: sessionID,
		Cost:      cost,
		Model:     raw.Message.Model,
		Data: types.UsageEntry{
			Timestamp: raw.Timestamp,
			SessionID: raw.SessionID,
			CostUSD:   raw.CostUSD,
			Message: types.UsageMessage{
				Model: raw.Message.Model,
				Usage: types.TokenUsageRaw{
					InputTokens:              uint64(u.InputTokens),
					OutputTokens:             uint64(u.OutputTokens),
					CacheCreationInputTokens: uint64(u.CacheCreationInputTokens),
					CacheReadInputTokens:     uint64(u.CacheReadInputTokens),
				},
			},
		},
		MissingPricingModel: missingModel,
	}
}
