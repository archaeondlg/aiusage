// Package codex implements the Codex CLI usage log adapter.
//
// Full implementation matching the Rust version:
//   - CODEX_HOME multi-path discovery
//   - sessions/ and archived_sessions/ directories
//   - active vs archived deduplication (active preferred)
//   - subagent replay detection (thread_spawn)
//   - codex-auto-review model resolution (date-based fallback table)
//   - cumulative token diffing (total_token_usage → per-event differentials)
//   - file mtime fallback for unparseable timestamps
//   - headless format (turn.completed, data/result/response nesting)
//   - flexible field names (input_tokens/prompt_tokens/input, etc.)
package codex

import (
	"context"

	"github.com/archhaeondlg/aiusage/internal/adapter"
	"github.com/archhaeondlg/aiusage/internal/adapter/shared"
	"github.com/archhaeondlg/aiusage/internal/dateutil"
	"github.com/archhaeondlg/aiusage/internal/pricing"
	"github.com/archhaeondlg/aiusage/internal/summary"
	"github.com/archhaeondlg/aiusage/internal/types"
)

type CodexAdapter struct{}

func NewAdapter() *CodexAdapter { return &CodexAdapter{} }
func (a *CodexAdapter) Name() string { return "codex" }

func (a *CodexAdapter) LoadEntries(ctx context.Context, opts adapter.LoadOptions) ([]*types.LoadedEntry, error) {
	pm := opts.Pricing
	if pm == nil {
		pm = pricing.LoadDefaultPricing()
	}

	// Load groups directly (this is the Rust path).
	singleThread := opts.SingleThread
	groups, err := LoadGroups(singleThread)
	if err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return nil, nil
	}

	// Convert groups to LoadedEntry slice.
	tz := dateutil.ParseTZ(&opts.Timezone)
	var entries []*types.LoadedEntry
	for _, key := range sortedGroupKeys(groups) {
		g := groups[key]

		for model, usage := range g.Models {
			nonCached := nonCachedInput(usage.InputTokens, usage.CachedInputTokens)
			entry := &types.LoadedEntry{
				Data: types.UsageEntry{
					Timestamp: g.LastActivity,
					Message: types.UsageMessage{
						Usage: types.TokenUsageRaw{
							InputTokens:          nonCached,
							OutputTokens:         usage.OutputTokens,
							CacheReadInputTokens: usage.CachedInputTokens,
						},
						Model: &model,
					},
				},
				Date:                 key,
				Project:              "codex",
				SessionID:            key,
				ProjectPath:          "Codex",
				Model:                &model,
				ReasoningOutputTokens: usage.ReasoningOutputTokens,
			}

			if ts, err := dateutil.ParseTimestamp(g.LastActivity); err == nil {
				entry.Timestamp = ts
				entry.Date = dateutil.FormatDate(ts, tz)
			}

			cost := CalculateCodexModelCost(model, usage, pm, "standard")
			entry.Cost = cost
			if cost == 0 && pm.Find(model) == nil {
				entry.MissingPricingModel = &model
			}
			entries = append(entries, entry)
		}
	}

	// Date filter.
	if opts.Since != "" || opts.Until != "" {
		var filtered []*types.LoadedEntry
		since := dateutil.NormalizeDateBound(opts.Since)
		until := dateutil.NormalizeDateBound(opts.Until)
		for _, e := range entries {
			date := dateutil.NormalizeDateBound(e.Date)
			if since != "" && date < since {
				continue
			}
			if until != "" && date > until {
				continue
			}
			filtered = append(filtered, e)
		}
		entries = filtered
	}

	return entries, nil
}

func (a *CodexAdapter) Summarize(entries []*types.LoadedEntry, kind types.ReportKind) ([]*types.UsageSummary, error) {
	groups := make(map[string]*summary.UsageAccumulator)
	var order []string
	for _, e := range entries {
		key := e.Date
		if kind == types.ReportSession {
			key = e.SessionID
		}
		if _, ok := groups[key]; !ok {
			groups[key] = &summary.UsageAccumulator{BreakdownIdxs: make(map[string]int)}
			order = append(order, key)
		}
		groups[key].AddEntry(e)
	}
	var rows []*types.UsageSummary
	for _, key := range order {
		s := groups[key].IntoSummary()
		d := key
		if kind == types.ReportSession {
			s.SessionID = &d
		} else {
			s.Date = &d
		}
		rows = append(rows, s)
	}
	return rows, nil
}

func (a *CodexAdapter) ReportJSON(rows []*types.UsageSummary, kind types.ReportKind) (any, error) {
	return map[string]any{
		string(kind): rows,
		"totals":     shared.TotalsFromRows(rows),
	}, nil
}

func (a *CodexAdapter) Paths() ([]string, error) {
	sources, err := codexUsageSources()
	if err != nil {
		return nil, err
	}
	paths := make([]string, len(sources))
	for i, s := range sources {
		paths[i] = s.Dir
	}
	return paths, nil
}

func (a *CodexAdapter) IsAvailable() bool {
	sources, err := codexUsageSources()
	if err != nil || len(sources) == 0 {
		return false
	}
	for _, s := range sources {
		if len(collectCodexUsageFiles(s.Dir)) > 0 {
			return true
		}
	}
	return false
}


