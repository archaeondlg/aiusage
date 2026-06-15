// Package adapter provides the adapter interface for coding agent CLI usage log parsing.
package adapter

import (
	"context"

	"github.com/archhaeondlg/aiusage/internal/pricing"
	"github.com/archhaeondlg/aiusage/internal/types"
)

// LoadOptions configures how an adapter loads and processes usage data.
type LoadOptions struct {
	Pricing       pricing.PricingProvider
	Timezone      string
	Since         string
	Until         string
	JSON          bool
	SingleThread  bool
	ProjectFilter string
	Verbose       int
}

// Error represents an adapter-level error with a user-facing message.
type Error struct {
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

// Adapter is the interface that all coding agent adapters must implement.
// Each adapter handles one specific agent's log format.
type Adapter interface {
	// Name returns the adapter identifier (e.g., "claude", "codex").
	Name() string

	// LoadEntries discovers and parses usage log files from disk.
	LoadEntries(ctx context.Context, opts LoadOptions) ([]*types.LoadedEntry, error)

	// Summarize aggregates loaded entries into report rows.
	Summarize(entries []*types.LoadedEntry, kind types.ReportKind) ([]*types.UsageSummary, error)

	// ReportJSON builds the final structured JSON report.
	ReportJSON(rows []*types.UsageSummary, kind types.ReportKind) (any, error)

	// Paths returns the data directories where this agent stores logs.
	Paths() ([]string, error)

	// IsAvailable returns true if this agent has log data on the system.
	IsAvailable() bool
}
