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
	"github.com/archhaeondlg/aiusage/internal/dateutil"
	"github.com/archhaeondlg/aiusage/internal/output"
	"github.com/archhaeondlg/aiusage/internal/pricing"
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
				Date:        key,
				Project:     "codex",
				SessionID:   key,
				ProjectPath: "Codex",
				Model:       &model,
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
		for _, e := range entries {
			date := dateutil.NormalizeDateBound(e.Date)
			if opts.Since != "" && date < opts.Since {
				continue
			}
			if opts.Until != "" && date > opts.Until {
				continue
			}
			filtered = append(filtered, e)
		}
		entries = filtered
	}

	return entries, nil
}

func (a *CodexAdapter) Summarize(entries []*types.LoadedEntry, kind types.ReportKind) ([]*types.UsageSummary, error) {
	groups := make(map[string]*summaryAccumulator)
	var order []string
	for _, e := range entries {
		key := e.Date
		if kind == types.ReportSession {
			key = e.SessionID
		}
		if _, ok := groups[key]; !ok {
			groups[key] = &summaryAccumulator{breakdownIdx: make(map[string]int)}
			order = append(order, key)
		}
		groups[key].add(e)
	}
	var rows []*types.UsageSummary
	for _, key := range order {
		s := groups[key].into()
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
		"totals":     totalsFromRows(rows),
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

// Run executes a full Codex report.
func Run(opts adapter.LoadOptions, kind types.ReportKind) error {
	pm := opts.Pricing
	if pm == nil {
		pm = pricing.LoadDefaultPricing()
	}
	singleThread := opts.SingleThread

	groups, err := LoadGroups(singleThread)
	if err != nil {
		return err
	}

	speed := string(ResolveCodexSpeed(CodexSpeedAuto))

	if opts.JSON {
		report := ReportFromGroups(groups, kind, pm, speed)
		return output.PrintJSONOrJQ(report, "", false)
	}

	PrintCodexTable(groups, kind, pm, speed, false)
	return nil
}

type summaryAccumulator struct {
	tokens       types.TokenCounts
	cost         float64
	models       []string
	breakdowns   []types.ModelBreakdown
	breakdownIdx map[string]int
}

func (a *summaryAccumulator) add(e *types.LoadedEntry) {
	u := e.Data.Message.Usage
	a.tokens.AddUsage(u)
	a.cost += e.Cost
	if e.Model != nil {
		idx, ok := a.breakdownIdx[*e.Model]
		if !ok {
			idx = len(a.breakdowns)
			a.breakdownIdx[*e.Model] = idx
			a.models = append(a.models, *e.Model)
			a.breakdowns = append(a.breakdowns, types.ModelBreakdown{ModelName: *e.Model})
		}
		bd := &a.breakdowns[idx]
		bd.InputTokens += u.InputTokens
		bd.OutputTokens += u.OutputTokens
		bd.CacheCreation += u.CacheCreationTokenCount()
		bd.CacheRead += u.CacheReadInputTokens
		bd.Cost += e.Cost
	}
}

func (a *summaryAccumulator) into() *types.UsageSummary {
	for i := 0; i < len(a.breakdowns); i++ {
		for j := i + 1; j < len(a.breakdowns); j++ {
			if a.breakdowns[j].Cost > a.breakdowns[i].Cost {
				a.breakdowns[i], a.breakdowns[j] = a.breakdowns[j], a.breakdowns[i]
			}
		}
	}
	return &types.UsageSummary{
		InputTokens:     a.tokens.InputTokens,
		OutputTokens:    a.tokens.OutputTokens,
		CacheCreation:   a.tokens.CacheCreation,
		CacheRead:       a.tokens.CacheRead,
		TotalCost:       a.cost,
		ModelsUsed:      a.models,
		ModelBreakdowns: a.breakdowns,
	}
}

func totalsFromRows(rows []*types.UsageSummary) map[string]any {
	var input, output, cc, cr uint64
	var cost float64
	for _, r := range rows {
		input += r.InputTokens
		output += r.OutputTokens
		cc += r.CacheCreation
		cr += r.CacheRead
		cost += r.TotalCost
	}
	return map[string]any{
		"inputTokens":        input,
		"outputTokens":       output,
		"cacheCreationTokens": cc,
		"cacheReadTokens":    cr,
		"totalTokens":        input + output + cc + cr,
		"totalCost":          cost,
	}
}
