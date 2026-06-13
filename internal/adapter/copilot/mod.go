package copilot

import (
	"context"

	"github.com/archhaeondlg/aiusage/internal/adapter"
	"github.com/archhaeondlg/aiusage/internal/types"
)

type CopilotAdapter struct{}

func NewAdapter() *CopilotAdapter { return &CopilotAdapter{} }
func (a *CopilotAdapter) Name() string { return "copilot" }

func (a *CopilotAdapter) LoadEntries(ctx context.Context, opts adapter.LoadOptions) ([]*types.LoadedEntry, error) {
	return nil, nil
}

func (a *CopilotAdapter) Summarize(entries []*types.LoadedEntry, kind types.ReportKind) ([]*types.UsageSummary, error) {
	return nil, nil
}

func (a *CopilotAdapter) ReportJSON(rows []*types.UsageSummary, kind types.ReportKind) (any, error) {
	return map[string]any{"daily": rows, "totals": nil}, nil
}

func (a *CopilotAdapter) Paths() ([]string, error) { return nil, nil }
func (a *CopilotAdapter) IsAvailable() bool { return false }
