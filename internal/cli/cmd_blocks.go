package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/archhaeondlg/aiusage/internal/adapter"
	"github.com/archhaeondlg/aiusage/internal/adapter/claude"
	"github.com/archhaeondlg/aiusage/internal/blocks"
	"github.com/archhaeondlg/aiusage/internal/output"
)

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
