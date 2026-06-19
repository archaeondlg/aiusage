// Package cli provides the cobra CLI framework for aiusage.
package cli

import (
	"github.com/spf13/cobra"
)

// rootCmd is the base command for aiusage.
var rootCmd = &cobra.Command{
	Use:   "aiusage",
	Short: "Analyze agent CLI token usage and costs from local data",
	Long: `aiusage reads local usage logs from 15 coding agent CLIs
(Claude Code, Codex, OpenCode, etc.) to track tokens and estimate costs.

Use -a/--agent to select an agent (default: claude), or "all" for all agents.
Use -m/--model to filter by model name.
Use -p/--project to filter by project.`,
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
	flags.StringP("agent", "a", "claude", "Agent to query (claude, codex, opencode, all, etc.)")
	flags.StringP("model", "m", "", "Filter by model name (fuzzy match)")
	flags.String("timezone", "", "Timezone for date grouping (e.g. Asia/Tokyo)")
	flags.StringP("since", "s", "", "Start date filter (YYYY-MM-DD or YYYY-MM-DD HH:MM:SS)")
	flags.StringP("until", "u", "", "End date filter (YYYY-MM-DD or YYYY-MM-DD HH:MM:SS)")
	flags.String("order", "asc", "Sort order (asc or desc)")
	flags.String("jq", "", "jq filter for JSON output")
	flags.StringP("project", "p", "", "Filter to a specific project")
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

}
