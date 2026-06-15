package claude

import (
	"github.com/archhaeondlg/aiusage/internal/summary"
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
	groups := make(map[string]*summary.UsageAccumulator)
	var groupOrder []string

	for _, entry := range entries {
		key := keyFn(entry)
		if _, ok := groups[key]; !ok {
			groups[key] = &summary.UsageAccumulator{
				BreakdownIdxs: make(map[string]int),
			}
			groupOrder = append(groupOrder, key)
		}
		groups[key].AddEntry(entry)
	}

	var rows []*types.UsageSummary
	for _, key := range groupOrder {
		group := groups[key]
		date, project := metaFn(key)
		row := group.IntoSummary()
		row.Date = &date
		row.Project = project
		rows = append(rows, row)
	}
	return rows
}
