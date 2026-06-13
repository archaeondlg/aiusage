package claude

import (
	"sort"

	"github.com/archhaeondlg/aiusage/internal/types"
)

// LoadDailySummaries loads entries and aggregates them by date.
// When groupByProject is true, entries are grouped by "date\0project".
func LoadDailySummaries(opts loadOptions, groupByProject bool) ([]*types.UsageSummary, error) {
	entries, err := LoadEntries(opts)
	if err != nil {
		return nil, err
	}
	if entries == nil {
		return nil, nil
	}

	if groupByProject {
		return summarizeByKey(entries, func(e *types.LoadedEntry) string {
			return e.Date + "\x00" + e.Project
		}, func(key string) (string, *string) {
			parts := splitByNull(key)
			date := ""
			if len(parts) > 0 {
				date = parts[0]
			}
			var project *string
			if len(parts) > 1 {
				p := parts[1]
				project = &p
			}
			return date, project
		}), nil
	}

	return summarizeByKey(entries, func(e *types.LoadedEntry) string {
		return e.Date
	}, func(key string) (string, *string) {
		return key, nil
	}), nil
}

func splitByNull(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == 0 {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Summarization helpers
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

func summarizeByKey(
	entries []*types.LoadedEntry,
	keyFn func(*types.LoadedEntry) string,
	metaFn func(string) (string, *string),
) []*types.UsageSummary {
	groups := make(map[string]*usageAccumulator)
	var groupOrder []string

	for _, entry := range entries {
		key := keyFn(entry)
		if _, ok := groups[key]; !ok {
			groups[key] = &usageAccumulator{
				breakdownIdxs: make(map[string]int),
			}
			groupOrder = append(groupOrder, key)
		}
		groups[key].addEntry(entry)
	}

	var rows []*types.UsageSummary
	for _, key := range groupOrder {
		group := groups[key]
		date, project := metaFn(key)
		summary := group.intoSummary()
		summary.Date = &date
		summary.Project = project
		rows = append(rows, summary)
	}
	return rows
}

// usageAccumulator mirrors Rust UsageAccumulator.
type usageAccumulator struct {
	counts         types.TokenCounts
	cost           float64
	credits        *float64
	messageCount   *uint64
	models         []string
	breakdowns     []types.ModelBreakdown
	breakdownIdxs  map[string]int
}

func (a *usageAccumulator) addEntry(entry *types.LoadedEntry) {
	usage := entry.Data.Message.Usage
	a.counts.AddUsage(usage)
	a.counts.ExtraTotalTokens += entry.ExtraTotalTokens
	a.cost += entry.Cost

	if entry.Credits != nil {
		if a.credits == nil {
			a.credits = new(float64)
		}
		*a.credits += *entry.Credits
	}
	if entry.MessageCount != nil {
		if a.messageCount == nil {
			a.messageCount = new(uint64)
		}
		*a.messageCount += *entry.MessageCount
	}

	if entry.Model != nil {
		model := *entry.Model
		idx, ok := a.breakdownIdxs[model]
		if !ok {
			idx = len(a.breakdowns)
			a.breakdownIdxs[model] = idx
			a.models = append(a.models, model)
			a.breakdowns = append(a.breakdowns, types.ModelBreakdown{
				ModelName: model,
			})
		}
		bd := &a.breakdowns[idx]
		bd.InputTokens += usage.InputTokens
		bd.OutputTokens += usage.OutputTokens
		bd.CacheCreation += usage.CacheCreationTokenCount()
		bd.CacheRead += usage.CacheReadInputTokens
		bd.ExtraTotalTokens += entry.ExtraTotalTokens
		bd.Cost += entry.Cost
		if entry.MissingPricingModel != nil {
			bd.MissingPricing = true
		}
	}
}

func (a *usageAccumulator) intoSummary() *types.UsageSummary {
	// Sort breakdowns by cost descending.
	sort.SliceStable(a.breakdowns, func(i, j int) bool {
		return a.breakdowns[i].Cost > a.breakdowns[j].Cost
	})

	return &types.UsageSummary{
		InputTokens:     a.counts.InputTokens,
		OutputTokens:    a.counts.OutputTokens,
		CacheCreation:   a.counts.CacheCreation,
		CacheRead:       a.counts.CacheRead,
		ExtraTotal:      a.counts.ExtraTotalTokens,
		TotalCost:       a.cost,
		Credits:         a.credits,
		MessageCount:    a.messageCount,
		ModelsUsed:      a.models,
		ModelBreakdowns: a.breakdowns,
	}
}
