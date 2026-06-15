package shared

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/archhaeondlg/aiusage/internal/adapter"
	"github.com/archhaeondlg/aiusage/internal/pricing"
	"github.com/archhaeondlg/aiusage/internal/summary"
	"github.com/archhaeondlg/aiusage/internal/types"
)

// GenericAdapter implements the Adapter interface for any agent that stores
// JSONL usage logs in a known directory.
type GenericAdapter struct {
	name  string
	dirs  []string // known data directories (checked for existence)
}

// NewGenericAdapter creates a GenericAdapter.
func NewGenericAdapter(name string, dirs []string) *GenericAdapter {
	return &GenericAdapter{name: name, dirs: dirs}
}

func (a *GenericAdapter) Name() string { return a.name }

func (a *GenericAdapter) LoadEntries(ctx context.Context, opts adapter.LoadOptions) ([]*types.LoadedEntry, error) {
	pm := opts.Pricing
	if pm == nil {
		pm = pricing.LoadDefaultPricing()
	}

	paths, err := a.Paths()
	if err != nil {
		slog.Debug("adapter paths error", "adapter", a.name, "error", err)
	}
	files := FindJSONLFiles(paths)
	if len(files) == 0 {
		return nil, nil
	}

	var entries []*types.LoadedEntry
	for _, file := range files {
		fileEntries := parseJSONLFile(file, pm)
		entries = append(entries, fileEntries...)
	}
	return entries, nil
}

func (a *GenericAdapter) Summarize(entries []*types.LoadedEntry, kind types.ReportKind) ([]*types.UsageSummary, error) {
	if kind == types.ReportSession {
		return summary.SummarizeByKey(entries,
			func(e *types.LoadedEntry) string { return e.SessionID },
			func(key string) (string, *string) { return "", &key },
		), nil
	}
	return summary.SummarizeByKey(entries,
		func(e *types.LoadedEntry) string { return e.Date },
		func(key string) (string, *string) { return key, nil },
	), nil
}

func (a *GenericAdapter) ReportJSON(rows []*types.UsageSummary, kind types.ReportKind) (any, error) {
	return map[string]any{
		string(kind): rows,
		"totals":     TotalsFromRows(rows),
	}, nil
}

func (a *GenericAdapter) Paths() ([]string, error) {
	var found []string
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home dir: %w", err)
	}
	for _, dir := range a.dirs {
		// Expand ~/ to home directory.
		d := dir
		if len(d) > 1 && d[:2] == "~/" {
			d = filepath.Join(home, d[2:])
		}
		if info, err := os.Stat(d); err == nil && info.IsDir() {
			found = append(found, d)
		}
	}
	return found, nil
}

func (a *GenericAdapter) IsAvailable() bool {
	paths, _ := a.Paths()
	return len(paths) > 0 && len(FindJSONLFiles(paths)) > 0
}

// TotalsFromRows computes total metrics across all rows.
func TotalsFromRows(rows []*types.UsageSummary) map[string]any {
	var input, output, cc, cr, extra uint64
	var cost float64
	for _, r := range rows {
		input += r.InputTokens
		output += r.OutputTokens
		cc += r.CacheCreation
		cr += r.CacheRead
		extra += r.ExtraTotal
		cost += r.TotalCost
	}
	return map[string]any{
		"inputTokens":         input,
		"outputTokens":        output,
		"cacheCreationTokens": cc,
		"cacheReadTokens":     cr,
		"totalTokens":         input + output + cc + cr + extra,
		"totalCost":           cost,
	}
}

func parseJSONLFile(path string, pm pricing.PricingProvider) []*types.LoadedEntry {
	var entries []*types.LoadedEntry
	if err := ReadJSONLLines(path, func(line []byte) error {
		entry := ParseGenericEntry(line, pm)
		if entry != nil {
			entries = append(entries, entry)
		}
		return nil
	}); err != nil {
		slog.Debug("skipping malformed JSONL file", "path", path, "error", err)
	}
	return entries
}
