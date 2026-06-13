package codebuff

import (
	"context"

	"github.com/archhaeondlg/aiusage/internal/adapter"
	"github.com/archhaeondlg/aiusage/internal/types"
)

type CodebuffAdapter struct{}

func NewAdapter() *CodebuffAdapter { return &CodebuffAdapter{} }
func (a *CodebuffAdapter) Name() string { return "codebuff" }

func (a *CodebuffAdapter) LoadEntries(ctx context.Context, opts adapter.LoadOptions) ([]*types.LoadedEntry, error) {
	return nil, nil
}

func (a *CodebuffAdapter) Summarize(entries []*types.LoadedEntry, kind types.ReportKind) ([]*types.UsageSummary, error) {
	return nil, nil
}

func (a *CodebuffAdapter) ReportJSON(rows []*types.UsageSummary, kind types.ReportKind) (any, error) {
	return map[string]any{"daily": rows, "totals": nil}, nil
}

func (a *CodebuffAdapter) Paths() ([]string, error) { return nil, nil }
func (a *CodebuffAdapter) IsAvailable() bool { return false }
