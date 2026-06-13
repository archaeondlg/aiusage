package pi

import (
	"context"

	"github.com/archhaeondlg/aiusage/internal/adapter"
	"github.com/archhaeondlg/aiusage/internal/types"
)

type PiAdapter struct{}

func NewAdapter() *PiAdapter { return &PiAdapter{} }
func (a *PiAdapter) Name() string { return "pi" }

func (a *PiAdapter) LoadEntries(ctx context.Context, opts adapter.LoadOptions) ([]*types.LoadedEntry, error) {
	return nil, nil
}

func (a *PiAdapter) Summarize(entries []*types.LoadedEntry, kind types.ReportKind) ([]*types.UsageSummary, error) {
	return nil, nil
}

func (a *PiAdapter) ReportJSON(rows []*types.UsageSummary, kind types.ReportKind) (any, error) {
	return map[string]any{"daily": rows, "totals": nil}, nil
}

func (a *PiAdapter) Paths() ([]string, error) { return nil, nil }
func (a *PiAdapter) IsAvailable() bool { return false }
