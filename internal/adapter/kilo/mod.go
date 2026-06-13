package kilo

import (
	"context"

	"github.com/archhaeondlg/aiusage/internal/adapter"
	"github.com/archhaeondlg/aiusage/internal/types"
)

type KiloAdapter struct{}

func NewAdapter() *KiloAdapter { return &KiloAdapter{} }
func (a *KiloAdapter) Name() string { return "kilo" }

func (a *KiloAdapter) LoadEntries(ctx context.Context, opts adapter.LoadOptions) ([]*types.LoadedEntry, error) {
	return nil, nil
}

func (a *KiloAdapter) Summarize(entries []*types.LoadedEntry, kind types.ReportKind) ([]*types.UsageSummary, error) {
	return nil, nil
}

func (a *KiloAdapter) ReportJSON(rows []*types.UsageSummary, kind types.ReportKind) (any, error) {
	return map[string]any{"daily": rows, "totals": nil}, nil
}

func (a *KiloAdapter) Paths() ([]string, error) { return nil, nil }
func (a *KiloAdapter) IsAvailable() bool { return false }
