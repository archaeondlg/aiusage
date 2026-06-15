// Package opencode implements the OpenCode usage log adapter.
//
// OpenCode stores usage data in two locations:
//   - SQLite database: opencode.db or opencode-{channel}.db
//   - JSON files: storage/message/*.json
//
// Messages include providerID, modelID, token counts with cache,
// and pre-computed cost. Model name normalization handles
// claude-sonnet-4.5 → claude-sonnet-4-5 and provider-prefixed
// pricing lookups (e.g., github_copilot/claude-sonnet-4-5).
package opencode

import (
	"context"
	"os"

	"github.com/archhaeondlg/aiusage/internal/adapter"
	"github.com/archhaeondlg/aiusage/internal/adapter/shared"
	"github.com/archhaeondlg/aiusage/internal/summary"
	"github.com/archhaeondlg/aiusage/internal/types"
)

type OpenCodeAdapter struct{}

func NewAdapter() *OpenCodeAdapter { return &OpenCodeAdapter{} }
func (a *OpenCodeAdapter) Name() string { return "opencode" }

func (a *OpenCodeAdapter) LoadEntries(ctx context.Context, opts adapter.LoadOptions) ([]*types.LoadedEntry, error) {
	entries, err := loadEntries(opts.Pricing, opts.Timezone)
	if err != nil {
		return nil, err
	}
	// Date filter.
	if opts.Since != "" || opts.Until != "" {
		var filtered []*types.LoadedEntry
		since := normalizeDateFast(opts.Since)
		until := normalizeDateFast(opts.Until)
		for _, e := range entries {
			date := normalizeDateFast(e.Date)
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

func (a *OpenCodeAdapter) Summarize(entries []*types.LoadedEntry, kind types.ReportKind) ([]*types.UsageSummary, error) {
	return summary.SummarizeByKey(entries,
		func(e *types.LoadedEntry) string { return e.Date },
		func(key string) (string, *string) { return key, nil },
	), nil
}

func (a *OpenCodeAdapter) ReportJSON(rows []*types.UsageSummary, kind types.ReportKind) (any, error) {
	report := map[string]any{
		string(kind): rows,
		"totals":     shared.TotalsFromRows(rows),
	}
	return report, nil
}

func (a *OpenCodeAdapter) Paths() ([]string, error) {
	return paths()
}

func (a *OpenCodeAdapter) IsAvailable() bool {
	dirs, err := paths()
	if err != nil || len(dirs) == 0 {
		return false
	}
	for _, dir := range dirs {
		// Check for DB or message files.
		if dbPath(dir) != "" {
			return true
		}
		msgDir := dir + "/storage/message"
		if info, err := os.Stat(msgDir); err == nil && info.IsDir() {
			return true
		}
	}
	return false
}



func normalizeDateFast(date string) string {
	result := ""
	for _, ch := range date {
		if ch != '-' {
			result += string(ch)
		}
	}
	return result
}

