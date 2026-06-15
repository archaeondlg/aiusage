package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/archhaeondlg/aiusage/internal/update"
)

func runUpdatePriceCmd(cmd *cobra.Command, args []string) error {
	fmt.Println("→ Fetching latest pricing from GitHub...")
	if err := UpdatePricingFromGitHub(); err != nil {
		return fmt.Errorf("update pricing: %w", err)
	}
	fmt.Println("  Pricing updated successfully.")
	return nil
}

func runUpdateCmd(cmd *cobra.Command, args []string) error {
	ver, err := update.SelfUpdate()
	if err != nil {
		return fmt.Errorf("update: %w", err)
	}
	fmt.Printf("  Updated to %s. Restart to use the new version.\n", ver)
	return nil
}
