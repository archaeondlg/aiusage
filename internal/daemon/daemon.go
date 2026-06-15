// Package daemon provides the watch/polling mode for aiusage.
//
// The daemon continuously monitors token usage log files at a configurable
// interval, printing live usage stats to the terminal or emitting JSON lines.
package daemon

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/archhaeondlg/aiusage/internal/adapter"
	_ "github.com/archhaeondlg/aiusage/internal/adapter/all"
	"github.com/archhaeondlg/aiusage/internal/output"
	"github.com/archhaeondlg/aiusage/internal/pricing"
	"github.com/archhaeondlg/aiusage/internal/summary"
	"github.com/archhaeondlg/aiusage/internal/types"
)

// DaemonOptions configures the daemon polling loop.
type DaemonOptions struct {
	Interval time.Duration // polling interval (default 30s)
	Agent    string        // agent filter: "claude", "all", "codex", etc.
	JSON     bool          // emit JSON lines instead of terminal output
	NoColor  bool          // disable ANSI color
	Compact  bool          // compact table style
	Timezone string        // timezone for date display
}

// agentEntry maps an agent key to its adapter and display name.
type agentEntry struct {
	adapter     adapter.Adapter
	displayName string
}

var agentDisplayNames = map[string]string{
	"claude":   "Claude Code",
	"codex":    "Codex CLI",
	"opencode": "OpenCode",
	"amp":      "Amp",
	"codebuff": "Codebuff",
	"copilot":  "GitHub Copilot CLI",
	"droid":    "Droid",
	"gemini":   "Gemini CLI",
	"goose":    "Goose",
	"hermes":   "Hermes Agent",
	"kilo":     "Kilo Code",
	"kimi":     "Kimi CLI",
	"openclaw": "OpenClaw",
	"pi":       "pi-agent",
	"qwen":     "Qwen Code",
}

// agentRegistry returns all registered adapters.
func agentRegistry() []agentEntry {
	var result []agentEntry
	for _, a := range adapter.AllAdapters() {
		name := a.Name()
		display := agentDisplayNames[name]
		if display == "" {
			display = name
		}
		result = append(result, agentEntry{a, display})
	}
	return result
}

// resolveAdapter returns one or more adapters based on the agent filter.
func resolveAdapter(filter string) ([]agentEntry, error) {
	if filter == "all" {
		var available []agentEntry
		for _, ae := range agentRegistry() {
			if ae.adapter.IsAvailable() {
				available = append(available, ae)
			}
		}
		if len(available) == 0 {
			return nil, fmt.Errorf("no coding agent log data found")
		}
		return available, nil
	}

	for _, ae := range agentRegistry() {
		if ae.adapter.Name() == filter {
			if !ae.adapter.IsAvailable() {
				return nil, fmt.Errorf("agent %q is not available (no log data found)", filter)
			}
			return []agentEntry{ae}, nil
		}
	}
	return nil, fmt.Errorf("unknown agent: %q", filter)
}

// Run starts the daemon polling loop. It blocks until ctx is cancelled.
func Run(ctx context.Context, opts DaemonOptions) error {
	// Set up signal handling.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Resolve adapters.
	agents, err := resolveAdapter(opts.Agent)
	if err != nil {
		return err
	}

	// Load pricing from config with builtin defaults.
	pricingMap := pricing.NewCachedProvider(pricing.LoadDefaultPricing())

	style := output.Style{Enabled: !opts.NoColor, NoColor: opts.NoColor}

	ticker := time.NewTicker(opts.Interval)
	defer ticker.Stop()

	// Prevent overlapping polls.
	var polling atomic.Bool

	// Print header once in terminal mode.
	if !opts.JSON {
		printDaemonHeader(opts, style)
	}

	// Run first poll immediately.
	if err := pollAndPrint(ctx, agents, pricingMap, style, opts); err != nil {
		if ctx.Err() != nil {
			return nil
		}
		fmt.Fprintf(os.Stderr, "daemon: poll error: %v\n", err)
	}

	// Main loop.
	for {
		select {
		case <-ctx.Done():
			if !opts.JSON {
				fmt.Println()
				output.PrintBoxTitle("aiusage daemon stopped", style)
			}
			return nil
		case <-ticker.C:
			if !polling.CompareAndSwap(false, true) {
				slog.Warn("daemon: skipped poll — previous cycle still running")
				continue
			}
			if err := pollAndPrint(ctx, agents, pricingMap, style, opts); err != nil {
				if ctx.Err() != nil {
					polling.Store(false)
					return nil
				}
				fmt.Fprintf(os.Stderr, "daemon: poll error: %v\n", err)
			}
			polling.Store(false)
		}
	}
}

// pollAndPrint runs one poll cycle and prints the result.
func pollAndPrint(
	ctx context.Context,
	agents []agentEntry,
	pricingMap pricing.PricingProvider,
	style output.Style,
	opts DaemonOptions,
) error {
	now := time.Now()

	// Collect entries from all matched adapters.
	var allEntries []*types.LoadedEntry
	for _, ae := range agents {
		alo := adapter.LoadOptions{
			Pricing:      pricingMap,
			Timezone:     opts.Timezone,
			SingleThread: false,
		}

		entries, err := ae.adapter.LoadEntries(ctx, alo)
		if err != nil || len(entries) == 0 {
			continue
		}
		allEntries = append(allEntries, entries...)
	}

	if opts.JSON {
		return printJSONCycle(allEntries, now)
	}

	// Clear screen and re-print header.
	clearScreen()
	printDaemonHeader(opts, style)

	if len(allEntries) == 0 {
		fmt.Println("  No usage data found. Waiting...")
		return nil
	}

	printDaemonStats(allEntries, now, style, opts)
	return nil
}

func printDaemonHeader(opts DaemonOptions, style output.Style) {
	agentLabel := opts.Agent
	if agentLabel == "all" {
		agentLabel = "all agents"
	}
	title := fmt.Sprintf("aiusage daemon · %s · every %s · Ctrl+C to quit",
		agentLabel, formatDuration(opts.Interval))
	output.PrintBoxTitle(title, style)
}

func printDaemonStats(entries []*types.LoadedEntry, now time.Time, style output.Style, opts DaemonOptions) {
	// Aggregate stats.
	rows := summary.SummarizeByKey(entries,
		func(e *types.LoadedEntry) string { return e.Date },
		func(key string) (string, *string) { return key, nil },
	)
	if len(rows) == 0 {
		fmt.Println("  No usage data found. Waiting...")
		return
	}

	totalInput := uint64(0)
	totalOutput := uint64(0)
	totalCost := 0.0
	allModels := make(map[string]bool)
	allProjects := make(map[string]bool)

	for _, row := range rows {
		totalInput += row.InputTokens
		totalOutput += row.OutputTokens
		totalCost += row.TotalCost
		for _, m := range row.ModelsUsed {
			allModels[m] = true
		}
		if row.Project != nil {
			allProjects[*row.Project] = true
		}
	}

	modelList := make([]string, 0, len(allModels))
	for m := range allModels {
		modelList = append(modelList, m)
	}

	// Compact single-line output for small stats.
	totalTokens := totalInput + totalOutput

	// Print stats block.
	fmt.Printf("  %-14s %s\n", "Updated:", style.Colorize(now.Format("2006-01-02 15:04:05"), output.ColorBlue))
	if len(allProjects) > 0 {
		projectList := make([]string, 0, len(allProjects))
		for p := range allProjects {
			projectList = append(projectList, p)
		}
		fmt.Printf("  %-14s %s\n", "Projects:", strings.Join(projectList, ", "))
	}
	fmt.Printf("  %-14s %s\n", "Models:", strings.Join(modelList, ", "))
	fmt.Printf("  %-14s %s\n", "Input:", style.Colorize(output.FormatNumber(totalInput), output.ColorYellow))
	fmt.Printf("  %-14s %s\n", "Output:", style.Colorize(output.FormatNumber(totalOutput), output.ColorYellow))
	fmt.Printf("  %-14s %s\n", "Total Tokens:", style.Colorize(output.FormatNumber(totalTokens), output.ColorGreen))

	costStr := output.FormatCurrency(totalCost)
	fmt.Printf("  %-14s %s\n", "Cost:", style.Colorize(costStr, output.ColorGreen))
	fmt.Printf("  %-14s %d\n", "Entries:", len(entries))

	// Print warnings.
	for _, w := range summary.MissingPricingWarnings(rows) {
		fmt.Fprintln(os.Stderr, w)
	}
}

func printJSONCycle(entries []*types.LoadedEntry, now time.Time) error {
	totalInput := uint64(0)
	totalOutput := uint64(0)
	totalCost := 0.0
	models := make(map[string]bool)
	projects := make(map[string]bool)

	for _, e := range entries {
		totalInput += e.Data.Message.Usage.InputTokens
		totalOutput += e.Data.Message.Usage.OutputTokens
		totalCost += e.Cost
		if e.Model != nil {
			models[*e.Model] = true
		}
		projects[e.Project] = true
	}

	modelList := make([]string, 0, len(models))
	for m := range models {
		modelList = append(modelList, m)
	}
	projectList := make([]string, 0, len(projects))
	for p := range projects {
		projectList = append(projectList, p)
	}

	data := map[string]any{
		"timestamp":    now.Format(time.RFC3339),
		"inputTokens":  totalInput,
		"outputTokens": totalOutput,
		"totalTokens":  totalInput + totalOutput,
		"totalCost":    totalCost,
		"models":       modelList,
		"projects":     projectList,
		"entries":      len(entries),
	}
	return output.PrintJSONOrJQ(data, "", false)
}

func clearScreen() {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		_ = cmd.Run()
	} else {
		fmt.Print("\033[2J\033[H")
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}
