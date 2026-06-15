package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/archhaeondlg/aiusage/internal/adapter"
	_ "github.com/archhaeondlg/aiusage/internal/adapter/all"
	"github.com/archhaeondlg/aiusage/internal/output"
	"github.com/archhaeondlg/aiusage/internal/summary"
	"github.com/archhaeondlg/aiusage/internal/types"
)

// runAgentReport is the central dispatch function for all agent reports.
func runAgentReport(cmd *cobra.Command, agent, kind string) error {
	opts, err := parseRunOptions(cmd)
	if err != nil {
		return fmt.Errorf("parse options: %w", err)
	}
	opts.Agent = agent
	opts.Kind = kind

	InitLogging(opts.Verbose, flagBool(cmd, "log-json"))

	pricingMap := loadPricingFromConfig()

	buildALO := func() adapter.LoadOptions {
		return adapter.LoadOptions{
			Pricing:       pricingMap,
			Timezone:      opts.Timezone,
			Since:         opts.Since,
			Until:         opts.Until,
			JSON:          opts.JSON,
			SingleThread:  opts.SingleThread,
			ProjectFilter: opts.Project,
		}
	}
	if agent == "all" {
		return runAllAgents(cmd, buildALO())
	}

	adp, ok := adapter.GetAdapter(agent)
	if !ok {
		fmt.Fprintf(cmd.OutOrStdout(), "Unknown agent: %s\n", agent)
		return nil
	}

	alo := buildALO()

	if agent == "claude" && kind == "blocks" {
		return runBlocks(cmd, alo)
	}
	if agent == "claude" && kind == "statusline" {
		return runStatusline(cmd, alo)
	}

	var reportKind types.ReportKind
	switch kind {
	case "daily":
		reportKind = types.ReportDaily
	case "weekly":
		reportKind = types.ReportWeekly
	case "monthly":
		reportKind = types.ReportMonthly
	case "session":
		reportKind = types.ReportSession
	default:
		reportKind = types.ReportDaily
	}

	ctx := context.Background()
	entries, err := adp.LoadEntries(ctx, alo)
	if err != nil {
		return fmt.Errorf("load entries: %w", err)
	}

	if len(entries) == 0 {
		fmt.Fprintln(os.Stderr, "No usage data found.")
		return nil
	}

	rows, err := adp.Summarize(entries, reportKind)
	if err != nil {
		return fmt.Errorf("summarize: %w", err)
	}

	if reportKind == types.ReportWeekly || reportKind == types.ReportMonthly {
		rows = summary.SummarizeByBucket(rows, reportKind, opts.WeekStartDay())
	}

	dateFn := func(s *types.UsageSummary) string {
		if s.Date != nil {
			return *s.Date
		}
		if s.Week != nil {
			return *s.Week
		}
		if s.Month != nil {
			return *s.Month
		}
		return ""
	}
	rows = summary.FilterAndSort(rows, opts.Since, opts.Until, opts.SortOrder(), dateFn)

	if opts.JSON {
		report, err := adp.ReportJSON(rows, reportKind)
		if err != nil {
			return fmt.Errorf("report json: %w", err)
		}
		return output.PrintJSONOrJQ(report, opts.JQ, false)
	}

	printUsageTable(agent, kind, rows, opts)
	return nil
}

// printUsageTable renders the terminal table output for a given agent.
func printUsageTable(agent, kind string, rows []*types.UsageSummary, opts *RunOptions) {
	if len(rows) == 0 {
		fmt.Println("No usage data found.")
		return
	}

	firstCol := "Date"
	switch kind {
	case "weekly":
		firstCol = "Week"
	case "monthly":
		firstCol = "Month"
	case "session":
		firstCol = "Session"
	}

	noColor := opts.NoColor || opts.Color == "never"
	style := output.Style{Enabled: !noColor, NoColor: noColor}

	title := titleForAgent(agent)
	output.PrintBoxTitle(title, style)

	compact := opts.Compact

	var headers []string
	var aligns []output.Align
	if compact {
		headers = []string{firstCol, "Models", "Input", "Output", "Cost (USD)"}
		aligns = []output.Align{output.AlignLeft, output.AlignLeft, output.AlignRight, output.AlignRight, output.AlignRight}
	} else {
		hasReasoning := false
		for _, row := range rows {
			if row.ReasoningOutputTokens > 0 {
				hasReasoning = true
				break
			}
		}
		if hasReasoning {
			headers = []string{firstCol, "Models", "Input", "Output", "Reasoning", "Cache Create", "Cache Read", "Total Tokens", "Cost (USD)"}
			aligns = []output.Align{output.AlignLeft, output.AlignLeft, output.AlignRight, output.AlignRight, output.AlignRight, output.AlignRight, output.AlignRight, output.AlignRight, output.AlignRight}
		} else {
			headers = []string{firstCol, "Models", "Input", "Output", "Cache Create", "Cache Read", "Total Tokens", "Cost (USD)"}
			aligns = []output.Align{output.AlignLeft, output.AlignLeft, output.AlignRight, output.AlignRight, output.AlignRight, output.AlignRight, output.AlignRight, output.AlignRight}
		}
	}

	t := output.NewTable(headers, aligns, style)
	hasReasoning := len(headers) > 8

	for _, row := range rows {
		label := ""
		switch {
		case row.Date != nil:
			label = *row.Date
		case row.Week != nil:
			label = *row.Week
		case row.Month != nil:
			label = *row.Month
		case row.SessionID != nil:
			label = *row.SessionID
		}

		models := output.FormatModelsMultiline(row.ModelsUsed)

		if compact {
			t.Push([]string{label, models, output.FormatNumber(row.InputTokens), output.FormatNumber(row.OutputTokens), output.FormatCurrency(row.TotalCost)})
		} else if hasReasoning {
			t.Push([]string{label, models, output.FormatNumber(row.InputTokens), output.FormatNumber(row.OutputTokens), output.FormatNumber(row.ReasoningOutputTokens), output.FormatNumber(row.CacheCreation), output.FormatNumber(row.CacheRead), output.FormatNumber(row.TotalTokens()), output.FormatCurrency(row.TotalCost)})
		} else {
			t.Push([]string{label, models, output.FormatNumber(row.InputTokens), output.FormatNumber(row.OutputTokens), output.FormatNumber(row.CacheCreation), output.FormatNumber(row.CacheRead), output.FormatNumber(row.TotalTokens()), output.FormatCurrency(row.TotalCost)})
		}
	}

	totals := output.TotalsJSON(rows)
	ti := totals["inputTokens"].(uint64)
	to := totals["outputTokens"].(uint64)
	tc := totals["cacheCreationTokens"].(uint64)
	tr := totals["cacheReadTokens"].(uint64)
	tt := totals["totalTokens"].(uint64)
	tco := 0.0
	if v, ok := totals["totalCost"].(float64); ok {
		tco = v
	} else if v, ok := totals["totalCost"].(int64); ok {
		tco = float64(v)
	}

	t.Separator()
	if compact {
		t.Push([]string{"", style.Colorize("Total", output.ColorYellow), style.Colorize(output.FormatNumber(ti), output.ColorYellow), style.Colorize(output.FormatNumber(to), output.ColorYellow), style.Colorize(output.FormatCurrency(tco), output.ColorYellow)})
	} else if hasReasoning {
		t.Push([]string{style.Colorize("Total", output.ColorYellow), "", style.Colorize(output.FormatNumber(ti), output.ColorYellow), style.Colorize(output.FormatNumber(to), output.ColorYellow), style.Colorize(output.FormatNumber(0), output.ColorYellow), style.Colorize(output.FormatNumber(tc), output.ColorYellow), style.Colorize(output.FormatNumber(tr), output.ColorYellow), style.Colorize(output.FormatNumber(tt), output.ColorYellow), style.Colorize(output.FormatCurrency(tco), output.ColorYellow)})
	} else {
		t.Push([]string{style.Colorize("Total", output.ColorYellow), "", style.Colorize(output.FormatNumber(ti), output.ColorYellow), style.Colorize(output.FormatNumber(to), output.ColorYellow), style.Colorize(output.FormatNumber(tc), output.ColorYellow), style.Colorize(output.FormatNumber(tr), output.ColorYellow), style.Colorize(output.FormatNumber(tt), output.ColorYellow), style.Colorize(output.FormatCurrency(tco), output.ColorYellow)})
	}
	t.Print()

	for _, w := range summary.MissingPricingWarnings(rows) {
		fmt.Fprintln(os.Stderr, w)
	}
}

func titleForAgent(agent string) string {
	switch agent {
	case "codex":
		return "Codex Token Usage Report"
	case "opencode":
		return "OpenCode Token Usage Report"
	case "claude":
		return "Claude Code Token Usage Report"
	default:
		return agent + " Token Usage Report"
	}
}
