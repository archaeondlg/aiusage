package amp

import (
	"context"

	"github.com/archhaeondlg/aiusage/internal/adapter"
	"github.com/archhaeondlg/aiusage/internal/types"
)

type AmpAdapter struct{}

func NewAdapter() *AmpAdapter { return &AmpAdapter{} }
func (a *AmpAdapter) Name() string { return "amp" }

func (a *AmpAdapter) LoadEntries(ctx context.Context, opts adapter.LoadOptions) ([]*types.LoadedEntry, error) {
	return nil, nil
}

func (a *AmpAdapter) Summarize(entries []*types.LoadedEntry, kind types.ReportKind) ([]*types.UsageSummary, error) {
	return nil, nil
}

func (a *AmpAdapter) ReportJSON(rows []*types.UsageSummary, kind types.ReportKind) (any, error) {
	return map[string]any{"daily": rows, "totals": nil}, nil
}

func (a *AmpAdapter) Paths() ([]string, error) { return nil, nil }
func (a *AmpAdapter) IsAvailable() bool { return false }
