package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

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
	"github.com/archhaeondlg/aiusage/internal/blocks"
	"github.com/archhaeondlg/aiusage/internal/daemon"
	"github.com/archhaeondlg/aiusage/internal/output"
	"github.com/archhaeondlg/aiusage/internal/summary"
	"github.com/archhaeondlg/aiusage/internal/types"
)

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Core report commands
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

var dailyCmd = &cobra.Command{
	Use:   "daily",
	Short: "Show daily token usage report",
	Long:  "Aggregate token usage by day for the default agent (Claude Code).",
	RunE:  runDailyCmd,
}

var weeklyCmd = &cobra.Command{
	Use:   "weekly",
	Short: "Show weekly token usage report",
	Long:  "Aggregate token usage by week for the default agent (Claude Code).",
	RunE:  runWeeklyCmd,
}

var monthlyCmd = &cobra.Command{
	Use:   "monthly",
	Short: "Show monthly token usage report",
	Long:  "Aggregate token usage by month for the default agent (Claude Code).",
	RunE:  runMonthlyCmd,
}

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Show per-session token usage report",
	Long:  "List token usage per session for the default agent (Claude Code).",
	RunE:  runSessionCmd,
}

var blocksCmd = &cobra.Command{
	Use:   "blocks",
	Short: "Show Claude Code blocks analysis",
	Long:  "Display session blocks, burn rates, and token limit projections for Claude Code.",
	RunE:  runBlocksCmd,
}

var statuslineCmd = &cobra.Command{
	Use:   "statusline",
	Short: "Show statusline integration output",
	Long:  "Output usage metrics formatted for Claude Code's statusline hook.",
	RunE:  runStatuslineCmd,
}

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Watch token usage files continuously",
	Long: `Monitor token usage log files at a configurable interval and display live stats.

The daemon polls agent log directories on each interval, re-parses
changed files, and displays current session token usage and cost.

Examples:
  aiusage daemon                    # Watch Claude Code, 30s interval
  aiusage daemon --interval 10      # 10-second polling
  aiusage daemon --agent all        # Watch all agents
  aiusage daemon --json             # JSON-line output mode`,
	RunE: runDaemonCmd,
}

var updatePriceCmd = &cobra.Command{
	Use:   "update-price",
	Short: "Update pricing from GitHub",
	Long:  "Fetch the latest config.json from GitHub and update only the pricing section in the local config.",
	RunE:  runUpdatePriceCmd,
}

func init() {
	// Blocks-specific flags.
	blocksCmd.Flags().String("token-limit", "500000", "Token limit for projections")
	blocksCmd.Flags().Float64("session-length", 5.0, "Session length in hours")
	blocksCmd.Flags().Bool("active", false, "Show active block detail only")

	// Statusline flags.
	statuslineCmd.Flags().Bool("offline", true, "Use offline mode for statusline")

	// Daemon flags.
	daemonCmd.Flags().Int("interval", 30, "Polling interval in seconds")
	daemonCmd.Flags().String("agent", "claude", "Agent to watch (claude, all, codex, opencode)")
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Combined "all" command
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

var allCmd = &cobra.Command{
	Use:   "all",
	Short: "Show usage from all detected coding agent CLIs",
	Long:  "Aggregate usage from all supported coding agent CLIs in one report.",
	RunE:  runAllCmd,
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Agent-specific commands
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

var codexCmd = newAgentCmd("codex", "Show Codex CLI usage report", "Codex")
var opencodeCmd = newAgentCmd("opencode", "Show OpenCode usage report", "OpenCode")
var ampCmd = newAgentCmd("amp", "Show Amp usage report", "Amp")
var droidCmd = newAgentCmd("droid", "Show Droid usage report", "Droid")
var codebuffCmd = newAgentCmd("codebuff", "Show Codebuff usage report", "Codebuff")
var hermesCmd = newAgentCmd("hermes", "Show Hermes Agent usage report", "Hermes")
var piCmd = newAgentCmd("pi", "Show pi-agent usage report", "Pi")
var gooseCmd = newAgentCmd("goose", "Show Goose usage report", "Goose")
var kiloCmd = newAgentCmd("kilo", "Show Kilo Code usage report", "Kilo")
var qwenCmd = newAgentCmd("qwen", "Show Qwen Code usage report", "Qwen")
var copilotCmd = newAgentCmd("copilot", "Show GitHub Copilot CLI usage report", "Copilot")
var geminiCmd = newAgentCmd("gemini", "Show Gemini CLI usage report", "Gemini")
var kimiCmd = newAgentCmd("kimi", "Show Kimi CLI usage report", "Kimi")
var openclawCmd = newAgentCmd("openclaw", "Show OpenClaw usage report", "OpenClaw")

func newAgentCmd(use, short, _ string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Long:  fmt.Sprintf("Show token usage report for %s.", short),
		RunE:  makeAgentRunner(use),
	}

	// Agent commands have a --kind flag (daily, weekly, monthly, session).
	cmd.Flags().String("kind", "daily", "Report kind (daily, weekly, monthly, session)")

	return cmd
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Command runners (stubs that dispatch to the adapter layer)
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

func runDailyCmd(cmd *cobra.Command, args []string) error {
	return runAgentReport(cmd, "claude", "daily")
}

func runWeeklyCmd(cmd *cobra.Command, args []string) error {
	return runAgentReport(cmd, "claude", "weekly")
}

func runMonthlyCmd(cmd *cobra.Command, args []string) error {
	return runAgentReport(cmd, "claude", "monthly")
}

func runSessionCmd(cmd *cobra.Command, args []string) error {
	return runAgentReport(cmd, "claude", "session")
}

func runBlocksCmd(cmd *cobra.Command, args []string) error {
	return runAgentReport(cmd, "claude", "blocks")
}

func runStatuslineCmd(cmd *cobra.Command, args []string) error {
	return runAgentReport(cmd, "claude", "statusline")
}

func runAllCmd(cmd *cobra.Command, args []string) error {
	return runAgentReport(cmd, "all", "")
}

func makeAgentRunner(agent string) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		kind, _ := cmd.Flags().GetString("kind")
		if kind == "" {
			kind = "daily"
		}
		return runAgentReport(cmd, agent, kind)
	}
}

// runAgentReport is the central dispatch function for all agent reports.
func runAgentReport(cmd *cobra.Command, agent, kind string) error {
	opts, err := parseRunOptions(cmd)
	if err != nil {
		return fmt.Errorf("parse options: %w", err)
	}
	opts.Agent = agent
	opts.Kind = kind

	// Load pricing once from config.
	pricingMap := loadPricingFromConfig()

	// "all" and dedicated adapter paths handled before the main switch.
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
	if agent == "codex" {
		return codex.Run(buildALO(), reportKindFromString(kind))
	}
	if agent == "opencode" {
		return opencode.Run(buildALO(), reportKindFromString(kind))
	}

	// Resolve agent adapter.
	var adp adapter.Adapter
	switch agent {
	case "claude":
		adp = claude.NewAdapter()
	case "codex":
		adp = codex.NewAdapter()
	case "opencode":
		adp = opencode.NewAdapter()
	case "amp":
		adp = amp.NewAdapter()
	case "droid":
		adp = droid.NewAdapter()
	case "codebuff":
		adp = codebuff.NewAdapter()
	case "hermes":
		adp = hermes.NewAdapter()
	case "pi":
		adp = pi.NewAdapter()
	case "goose":
		adp = goose.NewAdapter()
	case "kilo":
		adp = kilo.NewAdapter()
	case "kimi":
		adp = kimi.NewAdapter()
	case "qwen":
		adp = qwen.NewAdapter()
	case "copilot":
		adp = copilot.NewAdapter()
	case "gemini":
		adp = gemini.NewAdapter()
	case "openclaw":
		adp = openclaw.NewAdapter()
	default:
		fmt.Fprintf(cmd.OutOrStdout(), "Unknown agent: %s\n", agent)
		return nil
	}

	// Build adapter load options for remaining adapters.
	alo := buildALO()

	// Special handling for blocks and statusline.
	if agent == "claude" && kind == "blocks" {
		return runBlocks(cmd, alo)
	}
	if agent == "claude" && kind == "statusline" {
		return runStatusline(cmd, alo)
	}

	// Resolve report kind.
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

	// Load entries.
	ctx := context.Background()
	entries, err := adp.LoadEntries(ctx, alo)
	if err != nil {
		return fmt.Errorf("load entries: %w", err)
	}

	if len(entries) == 0 {
		fmt.Fprintln(os.Stderr, "No usage data found.")
		return nil
	}

	// Summarize.
	rows, err := adp.Summarize(entries, reportKind)
	if err != nil {
		return fmt.Errorf("summarize: %w", err)
	}

	// Apply bucket aggregation for weekly/monthly.
	if reportKind == types.ReportWeekly || reportKind == types.ReportMonthly {
		rows = summary.SummarizeByBucket(rows, reportKind, opts.WeekStartDay())
	}

	// Filter and sort.
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

	// Terminal table output.
	printUsageTable(cmd, kind, rows, opts)
	return nil
}

// printUsageTable renders the terminal table output.
func printUsageTable(cmd *cobra.Command, kind string, rows []*types.UsageSummary, opts *RunOptions) {
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

	title := "Claude Code Token Usage Report"
	output.PrintBoxTitle(title, style)

	compact := opts.Compact

	// Build table.
	var headers []string
	var aligns []output.Align
	if compact {
		headers = []string{firstCol, "Models", "Input", "Output", "Cost (USD)"}
		aligns = []output.Align{output.AlignLeft, output.AlignLeft, output.AlignRight, output.AlignRight, output.AlignRight}
	} else {
		headers = []string{firstCol, "Models", "Input", "Output", "Cache Create", "Cache Read", "Total Tokens", "Cost (USD)"}
		aligns = []output.Align{output.AlignLeft, output.AlignLeft, output.AlignRight, output.AlignRight, output.AlignRight, output.AlignRight, output.AlignRight, output.AlignRight}
	}

	t := output.NewTable(headers, aligns, style)

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
			t.Push([]string{
				label,
				models,
				output.FormatNumber(row.InputTokens),
				output.FormatNumber(row.OutputTokens),
				output.FormatCurrency(row.TotalCost),
			})
		} else {
			t.Push([]string{
				label,
				models,
				output.FormatNumber(row.InputTokens),
				output.FormatNumber(row.OutputTokens),
				output.FormatNumber(row.CacheCreation),
				output.FormatNumber(row.CacheRead),
				output.FormatNumber(row.TotalTokens()),
				output.FormatCurrency(row.TotalCost),
			})
		}
	}

	// Totals row.
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
		t.Push([]string{
			"",
			style.Colorize("Total", output.ColorYellow),
			style.Colorize(output.FormatNumber(ti), output.ColorYellow),
			style.Colorize(output.FormatNumber(to), output.ColorYellow),
			style.Colorize(output.FormatCurrency(tco), output.ColorYellow),
		})
	} else {
		t.Push([]string{
			style.Colorize("Total", output.ColorYellow),
			"",
			style.Colorize(output.FormatNumber(ti), output.ColorYellow),
			style.Colorize(output.FormatNumber(to), output.ColorYellow),
			style.Colorize(output.FormatNumber(tc), output.ColorYellow),
			style.Colorize(output.FormatNumber(tr), output.ColorYellow),
			style.Colorize(output.FormatNumber(tt), output.ColorYellow),
			style.Colorize(output.FormatCurrency(tco), output.ColorYellow),
		})
	}
	t.Print()

	// Warnings.
	for _, w := range summary.MissingPricingWarnings(rows) {
		fmt.Fprintln(os.Stderr, w)
	}
}

func runBlocks(cmd *cobra.Command, alo adapter.LoadOptions) error {
	ctx := context.Background()
	adp := claude.NewAdapter()
	entries, err := adp.LoadEntries(ctx, alo)
	if err != nil {
		return fmt.Errorf("load entries: %w", err)
	}
	if len(entries) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No usage data found.")
		return nil
	}

	bopts := blocks.BlockOptions{
		TokenLimit:    flagStr(cmd, "token-limit"),
		SessionLength: flagFloat(cmd, "session-length"),
		Active:        flagBool(cmd, "active"),
		JSON:          alo.JSON,
		NoColor:       flagBool(cmd, "no-color"),
	}

	builtBlocks, err := blocks.BuildBlocks(entries, bopts)
	if err != nil {
		return err
	}

	if alo.JSON {
		return output.PrintJSONOrJQ(blocks.BlocksJSON(builtBlocks, bopts), "", false)
	}
	blocks.PrintBlocksTable(builtBlocks, bopts)
	return nil
}

func runStatusline(cmd *cobra.Command, alo adapter.LoadOptions) error {
	ctx := context.Background()
	adp := claude.NewAdapter()
	entries, err := adp.LoadEntries(ctx, alo)
	if err != nil || len(entries) == 0 {
		fmt.Println("aiusage: no data")
		return nil
	}

	bopts := blocks.BlockOptions{
		TokenLimit:    flagStr(cmd, "token-limit"),
		SessionLength: flagFloat(cmd, "session-length"),
	}
	fmt.Println(blocks.StatuslineOutput(entries, bopts))
	return nil
}

func runDaemonCmd(cmd *cobra.Command, args []string) error {
	interval, _ := cmd.Flags().GetInt("interval")
	if interval < 1 {
		interval = 30
	}
	agent, _ := cmd.Flags().GetString("agent")
	if agent == "" {
		agent = "claude"
	}

	opts := daemon.DaemonOptions{
		Interval: time.Duration(interval) * time.Second,
		Agent:    agent,
		JSON:     flagBool(cmd, "json"),
		NoColor:  flagBool(cmd, "no-color"),
		Compact:  flagBool(cmd, "compact"),
		Timezone: flagStr(cmd, "timezone"),
	}

	ctx := context.Background()
	return daemon.Run(ctx, opts)
}

func runUpdatePriceCmd(cmd *cobra.Command, args []string) error {
	fmt.Println("→ Fetching latest pricing from GitHub...")
	if err := UpdatePricingFromGitHub(); err != nil {
		return fmt.Errorf("update pricing: %w", err)
	}
	fmt.Println("  Pricing updated successfully.")
	return nil
}

func flagStr(cmd *cobra.Command, name string) string {
	s, _ := cmd.Flags().GetString(name)
	return s
}

func flagBool(cmd *cobra.Command, name string) bool {
	b, _ := cmd.Flags().GetBool(name)
	return b
}

func reportKindFromString(kind string) types.ReportKind {
	switch kind {
	case "weekly":
		return types.ReportWeekly
	case "monthly":
		return types.ReportMonthly
	case "session":
		return types.ReportSession
	default:
		return types.ReportDaily
	}
}

// runAllAgents aggregates usage from all available adapters.
func runAllAgents(cmd *cobra.Command, alo adapter.LoadOptions) error {
	// List all registered adapters.
	allAdapters := []adapter.Adapter{
		claude.NewAdapter(),
		codex.NewAdapter(),
		opencode.NewAdapter(),
		amp.NewAdapter(),
		droid.NewAdapter(),
		codebuff.NewAdapter(),
		hermes.NewAdapter(),
		pi.NewAdapter(),
		goose.NewAdapter(),
		kilo.NewAdapter(),
		kimi.NewAdapter(),
		qwen.NewAdapter(),
		copilot.NewAdapter(),
		gemini.NewAdapter(),
		openclaw.NewAdapter(),
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
		report := map[string]any{
			"daily":  rows,
			"totals": output.TotalsJSON(rows),
		}
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

func flagFloat(cmd *cobra.Command, name string) float64 {
	f, _ := cmd.Flags().GetFloat64(name)
	return f
}
