package gemini

import (
	"context"

	"github.com/archhaeondlg/aiusage/internal/adapter"
	"github.com/archhaeondlg/aiusage/internal/types"
)

type GeminiAdapter struct{}

func NewAdapter() *GeminiAdapter { return &GeminiAdapter{} }
func (a *GeminiAdapter) Name() string { return "gemini" }

func (a *GeminiAdapter) LoadEntries(ctx context.Context, opts adapter.LoadOptions) ([]*types.LoadedEntry, error) {
	return nil, nil
}

func (a *GeminiAdapter) Summarize(entries []*types.LoadedEntry, kind types.ReportKind) ([]*types.UsageSummary, error) {
	return nil, nil
}

func (a *GeminiAdapter) ReportJSON(rows []*types.UsageSummary, kind types.ReportKind) (any, error) {
	return map[string]any{"daily": rows, "totals": nil}, nil
}

func (a *GeminiAdapter) Paths() ([]string, error) { return nil, nil }
func (a *GeminiAdapter) IsAvailable() bool { return false }
