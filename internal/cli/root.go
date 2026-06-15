// Package cli provides the cobra CLI framework for aiusage.
package cli

import (
	"github.com/spf13/cobra"
)

// rootCmd is the base command for aiusage.
var rootCmd = &cobra.Command{
	Use:   "aiusage",
	Short: "Analyze agent CLI token usage and costs from local data",
	Long: `aiusage reads local usage logs from Claude Code, Codex, OpenCode, Amp,
Droid, Codebuff, Hermes Agent, pi-agent, Goose, OpenClaw, Kilo, Kimi,
Qwen, GitHub Copilot CLI, and Gemini CLI to track tokens and estimate costs.`,
	Version: "1.0.0",
	RunE:    runDefault,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

// runDefault is the handler when no subcommand is given (equivalent to `aiusage all`).
func runDefault(cmd *cobra.Command, args []string) error {
	return runAllCmd(cmd, args)
}

func init() {
	// Persistent flags shared across all subcommands.
	flags := rootCmd.PersistentFlags()
	flags.Bool("json", false, "Output JSON format")
	flags.Bool("compact", false, "Force compact table layout")
	flags.Bool("breakdown", false, "Show per-model breakdown rows")
	flags.CountP("verbose", "v", "Verbose output (-v, -vv, -vvv)")
	flags.Bool("log-json", false, "JSON log output (for daemon/monitoring)")
	flags.Bool("no-color", false, "Disable ANSI color output")
	flags.Bool("single-thread", false, "Disable parallel file loading")
	flags.String("timezone", "", "Timezone for date grouping (e.g. Asia/Tokyo)")
	flags.String("since", "", "Start date filter (YYYY-MM-DD)")
	flags.String("until", "", "End date filter (YYYY-MM-DD)")
	flags.String("order", "asc", "Sort order (asc or desc)")
	flags.String("jq", "", "jq filter for JSON output")
	flags.String("project", "", "Filter to a specific project")
	flags.String("project-aliases", "", "Project name aliases (key=value,...)")
	flags.String("start-of-week", "monday", "Week start day (monday-sunday)")
	flags.String("color", "auto", "Color mode (auto, always, never)")

	// Register subcommands.
	rootCmd.AddCommand(dailyCmd)
	rootCmd.AddCommand(weeklyCmd)
	rootCmd.AddCommand(monthlyCmd)
	rootCmd.AddCommand(sessionCmd)
	rootCmd.AddCommand(blocksCmd)
	rootCmd.AddCommand(statuslineCmd)
	rootCmd.AddCommand(daemonCmd)
	rootCmd.AddCommand(updatePriceCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(allCmd)

	// Agent-specific subcommands.
	// rootCmd.AddCommand(codexCmd)
	// rootCmd.AddCommand(opencodeCmd)
	// rootCmd.AddCommand(ampCmd)
	// rootCmd.AddCommand(droidCmd)
	// rootCmd.AddCommand(codebuffCmd)
	// rootCmd.AddCommand(hermesCmd)
	// rootCmd.AddCommand(piCmd)
	// rootCmd.AddCommand(gooseCmd)
	// rootCmd.AddCommand(kiloCmd)
	// rootCmd.AddCommand(qwenCmd)
	// rootCmd.AddCommand(copilotCmd)
	// rootCmd.AddCommand(geminiCmd)
	// rootCmd.AddCommand(kimiCmd)
	// rootCmd.AddCommand(openclawCmd)
}
