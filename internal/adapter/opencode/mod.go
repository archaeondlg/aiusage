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
	"fmt"
	"os"

	"github.com/archhaeondlg/aiusage/internal/adapter"
	"github.com/archhaeondlg/aiusage/internal/output"
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
		for _, e := range entries {
			date := normalizeDateFast(e.Date)
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

func (a *OpenCodeAdapter) Summarize(entries []*types.LoadedEntry, kind types.ReportKind) ([]*types.UsageSummary, error) {
	return summary.SummarizeByKey(entries,
		func(e *types.LoadedEntry) string { return e.Date },
		func(key string) (string, *string) { return key, nil },
	), nil
}

func (a *OpenCodeAdapter) ReportJSON(rows []*types.UsageSummary, kind types.ReportKind) (any, error) {
	report := map[string]any{
		string(kind): rows,
		"totals":     totalsFromRows(rows),
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

// Run executes a full OpenCode report.
func Run(opts adapter.LoadOptions, kind types.ReportKind) error {
	entries, err := loadEntries(opts.Pricing, opts.Timezone)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		fmt.Println("No OpenCode usage data found.")
		return nil
	}

	rows := summary.SummarizeByKey(entries,
		func(e *types.LoadedEntry) string { return e.Date },
		func(key string) (string, *string) { return key, nil },
	)

	// Bucket aggregation.
	if kind == types.ReportWeekly || kind == types.ReportMonthly {
		rows = summary.SummarizeByBucket(rows, kind, types.WeekMonday)
	}

	// Filter and sort.
	dateFn := func(s *types.UsageSummary) string {
		if s.Date != nil {
			return *s.Date
		}
		return ""
	}
	rows = summary.FilterAndSort(rows, opts.Since, opts.Until, types.SortAsc, dateFn)

	if opts.JSON {
		report := map[string]any{
			string(kind): rows,
			"totals":     totalsFromRows(rows),
		}
		return output.PrintJSONOrJQ(report, "", false)
	}

	firstCol := "Date"
	switch kind {
	case types.ReportWeekly:
		firstCol = "Week"
	case types.ReportMonthly:
		firstCol = "Month"
	}

	printOpenCodeTable("OpenCode Token Usage Report", firstCol, rows)
	return nil
}

func totalsFromRows(rows []*types.UsageSummary) map[string]any {
	var input, output, cc, cr, extra uint64
	var cost float64
	for _, r := range rows {
		input += r.InputTokens
		output += r.OutputTokens
		cc += r.CacheCreation
		cr += r.CacheRead
		extra += r.ExtraTotal
		cost += r.TotalCost
	}
	return map[string]any{
		"inputTokens":        input,
		"outputTokens":       output,
		"cacheCreationTokens": cc,
		"cacheReadTokens":    cr,
		"totalTokens":        input + output + cc + cr + extra,
		"totalCost":          cost,
	}
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

func printOpenCodeTable(title, firstCol string, rows []*types.UsageSummary) {
	style := output.Style{Enabled: true, NoColor: false}
	output.PrintBoxTitle(title, style)
	headers := []string{firstCol, "Models", "Input", "Output", "Cache Create", "Cache Read", "Total Tokens", "Cost (USD)"}
	aligns := []output.Align{output.AlignLeft, output.AlignLeft, output.AlignRight, output.AlignRight, output.AlignRight, output.AlignRight, output.AlignRight, output.AlignRight}
	tbl := output.NewTable(headers, aligns, style)
	var ti, to, tcc, tcr, tt uint64
	var tc float64
	for _, row := range rows {
		l := ""
		if row.Date != nil { l = *row.Date } else if row.Month != nil { l = *row.Month } else if row.Week != nil { l = *row.Week }
		tbl.Push([]string{l, output.FormatModelsMultiline(row.ModelsUsed), output.FormatNumber(row.InputTokens), output.FormatNumber(row.OutputTokens), output.FormatNumber(row.CacheCreation), output.FormatNumber(row.CacheRead), output.FormatNumber(row.TotalTokens()), output.FormatCurrency(row.TotalCost)})
		ti += row.InputTokens; to += row.OutputTokens; tcc += row.CacheCreation; tcr += row.CacheRead; tt += row.TotalTokens(); tc += row.TotalCost
	}
	tbl.Separator()
	tbl.Push([]string{style.Colorize("Total", output.ColorYellow), "", style.Colorize(output.FormatNumber(ti), output.ColorYellow), style.Colorize(output.FormatNumber(to), output.ColorYellow), style.Colorize(output.FormatNumber(tcc), output.ColorYellow), style.Colorize(output.FormatNumber(tcr), output.ColorYellow), style.Colorize(output.FormatNumber(tt), output.ColorYellow), style.Colorize(output.FormatCurrency(tc), output.ColorYellow)})
	tbl.Print()
	for _, w := range summary.MissingPricingWarnings(rows) { fmt.Fprintln(os.Stderr, w) }
}

