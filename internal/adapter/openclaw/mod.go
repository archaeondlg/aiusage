package openclaw

import (
	"context"

	"github.com/archhaeondlg/aiusage/internal/adapter"
	"github.com/archhaeondlg/aiusage/internal/types"
)

type OpenclawAdapter struct{}

func NewAdapter() *OpenclawAdapter { return &OpenclawAdapter{} }
func (a *OpenclawAdapter) Name() string { return "openclaw" }

func (a *OpenclawAdapter) LoadEntries(ctx context.Context, opts adapter.LoadOptions) ([]*types.LoadedEntry, error) {
	return nil, nil
}

func (a *OpenclawAdapter) Summarize(entries []*types.LoadedEntry, kind types.ReportKind) ([]*types.UsageSummary, error) {
	return nil, nil
}

func (a *OpenclawAdapter) ReportJSON(rows []*types.UsageSummary, kind types.ReportKind) (any, error) {
	return map[string]any{"daily": rows, "totals": nil}, nil
}

func (a *OpenclawAdapter) Paths() ([]string, error) { return nil, nil }
func (a *OpenclawAdapter) IsAvailable() bool { return false }
