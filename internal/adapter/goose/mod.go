package goose

import (
	"context"

	"github.com/archhaeondlg/aiusage/internal/adapter"
	"github.com/archhaeondlg/aiusage/internal/types"
)

type GooseAdapter struct{}

func NewAdapter() *GooseAdapter { return &GooseAdapter{} }
func (a *GooseAdapter) Name() string { return "goose" }

func (a *GooseAdapter) LoadEntries(ctx context.Context, opts adapter.LoadOptions) ([]*types.LoadedEntry, error) {
	return nil, nil
}

func (a *GooseAdapter) Summarize(entries []*types.LoadedEntry, kind types.ReportKind) ([]*types.UsageSummary, error) {
	return nil, nil
}

func (a *GooseAdapter) ReportJSON(rows []*types.UsageSummary, kind types.ReportKind) (any, error) {
	return map[string]any{"daily": rows, "totals": nil}, nil
}

func (a *GooseAdapter) Paths() ([]string, error) { return nil, nil }
func (a *GooseAdapter) IsAvailable() bool { return false }
