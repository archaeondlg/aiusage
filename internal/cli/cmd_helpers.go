package cli

import (
	"github.com/spf13/cobra"

	"github.com/archhaeondlg/aiusage/internal/types"
)

func flagStr(cmd *cobra.Command, name string) string {
	s, _ := cmd.Flags().GetString(name)
	return s
}

func flagBool(cmd *cobra.Command, name string) bool {
	b, _ := cmd.Flags().GetBool(name)
	return b
}

func flagFloat(cmd *cobra.Command, name string) float64 {
	f, _ := cmd.Flags().GetFloat64(name)
	return f
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
