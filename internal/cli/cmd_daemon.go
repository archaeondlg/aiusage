package cli

import (
	"context"
	"time"

	"github.com/spf13/cobra"

	"github.com/archhaeondlg/aiusage/internal/daemon"
)

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
