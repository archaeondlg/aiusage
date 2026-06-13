package kimi

import (
	"context"

	"github.com/archhaeondlg/aiusage/internal/adapter"
	"github.com/archhaeondlg/aiusage/internal/types"
)

type KimiAdapter struct{}

func NewAdapter() *KimiAdapter { return &KimiAdapter{} }
func (a *KimiAdapter) Name() string { return "kimi" }

func (a *KimiAdapter) LoadEntries(ctx context.Context, opts adapter.LoadOptions) ([]*types.LoadedEntry, error) {
	return nil, nil
}

func (a *KimiAdapter) Summarize(entries []*types.LoadedEntry, kind types.ReportKind) ([]*types.UsageSummary, error) {
	return nil, nil
}

func (a *KimiAdapter) ReportJSON(rows []*types.UsageSummary, kind types.ReportKind) (any, error) {
	return map[string]any{"daily": rows, "totals": nil}, nil
}

func (a *KimiAdapter) Paths() ([]string, error) { return nil, nil }
func (a *KimiAdapter) IsAvailable() bool { return false }
