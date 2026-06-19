package cli

import (
	"github.com/spf13/cobra"
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

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Self-update aiusage from GitHub Releases",
	Long:  "Download the latest aiusage binary from GitHub Releases and replace the current executable.",
	RunE:  runUpdateCmd,
}

func init() {
	blocksCmd.Flags().String("token-limit", "500000", "Token limit for projections")
	blocksCmd.Flags().Float64("session-length", 5.0, "Session length in hours")
	blocksCmd.Flags().Bool("active", false, "Show active block detail only")

	statuslineCmd.Flags().Bool("offline", true, "Use offline mode for statusline")

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
// Command dispatchers
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

func runDailyCmd(cmd *cobra.Command, args []string) error {
	return runReportCmd(cmd, "daily")
}
func runWeeklyCmd(cmd *cobra.Command, args []string) error {
	return runReportCmd(cmd, "weekly")
}
func runMonthlyCmd(cmd *cobra.Command, args []string) error {
	return runReportCmd(cmd, "monthly")
}
func runSessionCmd(cmd *cobra.Command, args []string) error {
	return runReportCmd(cmd, "session")
}
func runBlocksCmd(cmd *cobra.Command, args []string) error {
	return runReportCmd(cmd, "blocks")
}
func runStatuslineCmd(cmd *cobra.Command, args []string) error {
	return runReportCmd(cmd, "statusline")
}
func runAllCmd(cmd *cobra.Command, args []string) error {
	return runReportCmd(cmd, "all")
}

// runReportCmd reads --agent from flags and dispatches to runAgentReport.
func runReportCmd(cmd *cobra.Command, kind string) error {
	agent := flagStr(cmd, "agent")
	if kind == "all" {
		agent = "all"
	}
	return runAgentReport(cmd, agent, kind)
}

