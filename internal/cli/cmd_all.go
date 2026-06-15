package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/archhaeondlg/aiusage/internal/adapter"
	"github.com/archhaeondlg/aiusage/internal/adapter/amp"
	"github.com/archhaeondlg/aiusage/internal/adapter/claude"
	"github.com/archhaeondlg/aiusage/internal/adapter/codebuff"
	"github.com/archhaeondlg/aiusage/internal/adapter/codex"
	"github.com/archhaeondlg/aiusage/internal/adapter/copilot"
	"github.com/archhaeondlg/aiusage/internal/adapter/droid"
	"github.com/archhaeondlg/aiusage/internal/adapter/gemini"
	"github.com/archhaeondlg/aiusage/internal/adapter/goose"
	"github.com/archhaeondlg/aiusage/internal/adapter/hermes"
	"github.com/archhaeondlg/aiusage/internal/adapter/kilo"
	"github.com/archhaeondlg/aiusage/internal/adapter/kimi"
	"github.com/archhaeondlg/aiusage/internal/adapter/openclaw"
	"github.com/archhaeondlg/aiusage/internal/adapter/opencode"
	"github.com/archhaeondlg/aiusage/internal/adapter/pi"
	"github.com/archhaeondlg/aiusage/internal/adapter/qwen"
	"github.com/archhaeondlg/aiusage/internal/output"
	"github.com/archhaeondlg/aiusage/internal/summary"
	"github.com/archhaeondlg/aiusage/internal/types"
)

// runAllAgents aggregates usage from all available adapters.
func runAllAgents(_ *cobra.Command, alo adapter.LoadOptions) error {
	allAdapters := []adapter.Adapter{
		claude.NewAdapter(), codex.NewAdapter(), opencode.NewAdapter(),
		amp.NewAdapter(), droid.NewAdapter(), codebuff.NewAdapter(),
		hermes.NewAdapter(), pi.NewAdapter(), goose.NewAdapter(),
		kilo.NewAdapter(), kimi.NewAdapter(), qwen.NewAdapter(),
		copilot.NewAdapter(), gemini.NewAdapter(), openclaw.NewAdapter(),
	}

	var allEntries []*types.LoadedEntry
	ctx := context.Background()
	for _, adp := range allAdapters {
		if !adp.IsAvailable() {
			continue
		}
		entries, err := adp.LoadEntries(ctx, alo)
		if err != nil || len(entries) == 0 {
			continue
		}
		allEntries = append(allEntries, entries...)
	}

	if len(allEntries) == 0 {
		fmt.Fprintln(os.Stderr, "No usage data found from any coding agent CLI.")
		return nil
	}

	rows := summary.SummarizeByKey(allEntries,
		func(e *types.LoadedEntry) string { return e.Date + "|" + e.Project },
		func(key string) (string, *string) {
			parts := strings.SplitN(key, "|", 2)
			date := parts[0]
			var project *string
			if len(parts) > 1 && parts[1] != "" {
				project = &parts[1]
			}
			return date, project
		},
	)
	rows = summary.FilterAndSort(rows, alo.Since, alo.Until, types.SortAsc,
		func(s *types.UsageSummary) string {
			if s.Date != nil {
				return *s.Date
			}
			return ""
		},
	)

	if alo.JSON {
		report := map[string]any{"daily": rows, "totals": output.TotalsJSON(rows)}
		return output.PrintJSONOrJQ(report, "", false)
	}

	printAllTable(rows)
	return nil
}

func printAllTable(rows []*types.UsageSummary) {
	if len(rows) == 0 {
		fmt.Println("No usage data found.")
		return
	}
	style := output.Style{Enabled: true, NoColor: false}
	output.PrintBoxTitle("All Coding Agent CLI Token Usage Report", style)
	headers := []string{"Date", "Agent", "Models", "Input", "Output", "Cache Read", "Total Tokens", "Cost (USD)"}
	aligns := []output.Align{output.AlignLeft, output.AlignLeft, output.AlignLeft, output.AlignRight, output.AlignRight, output.AlignRight, output.AlignRight, output.AlignRight}
	tbl := output.NewTable(headers, aligns, style)
	for _, row := range rows {
		agent := "—"
		if row.Project != nil {
			agent = *row.Project
		}
		d := ""
		if row.Date != nil {
			d = *row.Date
		}
		tbl.Push([]string{d, agent, output.FormatModelsMultiline(row.ModelsUsed), output.FormatNumber(row.InputTokens), output.FormatNumber(row.OutputTokens), output.FormatNumber(row.CacheRead), output.FormatNumber(row.TotalTokens()), output.FormatCurrency(row.TotalCost)})
	}
	totals := output.TotalsJSON(rows)
	tbl.Separator()
	ti := totals["inputTokens"].(uint64)
	to := totals["outputTokens"].(uint64)
	tcr := totals["cacheReadTokens"].(uint64)
	tt := totals["totalTokens"].(uint64)
	tc := totals["totalCost"].(float64)
	tbl.Push([]string{style.Colorize("Total", output.ColorYellow), "", "", style.Colorize(output.FormatNumber(ti), output.ColorYellow), style.Colorize(output.FormatNumber(to), output.ColorYellow), style.Colorize(output.FormatNumber(tcr), output.ColorYellow), style.Colorize(output.FormatNumber(tt), output.ColorYellow), style.Colorize(output.FormatCurrency(tc), output.ColorYellow)})
	tbl.Print()
}
