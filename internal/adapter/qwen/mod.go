package qwen

import (
	"context"

	"github.com/archhaeondlg/aiusage/internal/adapter"
	"github.com/archhaeondlg/aiusage/internal/types"
)

type QwenAdapter struct{}

func NewAdapter() *QwenAdapter { return &QwenAdapter{} }
func (a *QwenAdapter) Name() string { return "qwen" }

func (a *QwenAdapter) LoadEntries(ctx context.Context, opts adapter.LoadOptions) ([]*types.LoadedEntry, error) {
	return nil, nil
}

func (a *QwenAdapter) Summarize(entries []*types.LoadedEntry, kind types.ReportKind) ([]*types.UsageSummary, error) {
	return nil, nil
}

func (a *QwenAdapter) ReportJSON(rows []*types.UsageSummary, kind types.ReportKind) (any, error) {
	return map[string]any{"daily": rows, "totals": nil}, nil
}

func (a *QwenAdapter) Paths() ([]string, error) { return nil, nil }
func (a *QwenAdapter) IsAvailable() bool { return false }
