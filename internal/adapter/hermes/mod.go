package hermes

import (
	"context"

	"github.com/archhaeondlg/aiusage/internal/adapter"
	"github.com/archhaeondlg/aiusage/internal/types"
)

type HermesAdapter struct{}

func NewAdapter() *HermesAdapter { return &HermesAdapter{} }
func (a *HermesAdapter) Name() string { return "hermes" }

func (a *HermesAdapter) LoadEntries(ctx context.Context, opts adapter.LoadOptions) ([]*types.LoadedEntry, error) {
	return nil, nil
}

func (a *HermesAdapter) Summarize(entries []*types.LoadedEntry, kind types.ReportKind) ([]*types.UsageSummary, error) {
	return nil, nil
}

func (a *HermesAdapter) ReportJSON(rows []*types.UsageSummary, kind types.ReportKind) (any, error) {
	return map[string]any{"daily": rows, "totals": nil}, nil
}

func (a *HermesAdapter) Paths() ([]string, error) { return nil, nil }
func (a *HermesAdapter) IsAvailable() bool { return false }
