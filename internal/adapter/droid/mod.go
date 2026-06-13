package droid

import (
	"context"

	"github.com/archhaeondlg/aiusage/internal/adapter"
	"github.com/archhaeondlg/aiusage/internal/types"
)

type DroidAdapter struct{}

func NewAdapter() *DroidAdapter { return &DroidAdapter{} }
func (a *DroidAdapter) Name() string { return "droid" }

func (a *DroidAdapter) LoadEntries(ctx context.Context, opts adapter.LoadOptions) ([]*types.LoadedEntry, error) {
	return nil, nil
}

func (a *DroidAdapter) Summarize(entries []*types.LoadedEntry, kind types.ReportKind) ([]*types.UsageSummary, error) {
	return nil, nil
}

func (a *DroidAdapter) ReportJSON(rows []*types.UsageSummary, kind types.ReportKind) (any, error) {
	return map[string]any{"daily": rows, "totals": nil}, nil
}

func (a *DroidAdapter) Paths() ([]string, error) { return nil, nil }
func (a *DroidAdapter) IsAvailable() bool { return false }
