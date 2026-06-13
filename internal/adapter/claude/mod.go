// Package claude implements the Claude Code usage log adapter.
//
// Claude Code stores usage data as JSONL files under:
//
//	$CLAUDE_CONFIG_DIR/projects/<project>/<session>.jsonl
//	$CLAUDE_CONFIG_DIR/projects/<project>/<session>/chat.jsonl
//	$CLAUDE_CONFIG_DIR/projects/<project>/<session>/subagents/<agent>.jsonl
package claude

import (
	"context"

	"github.com/archhaeondlg/aiusage/internal/adapter"
	"github.com/archhaeondlg/aiusage/internal/pricing"
	"github.com/archhaeondlg/aiusage/internal/types"
)

// ClaudeAdapter implements the Adapter interface for Claude Code.
type ClaudeAdapter struct{}

// NewAdapter creates a new Claude adapter.
func NewAdapter() *ClaudeAdapter {
	return &ClaudeAdapter{}
}

// Name returns the adapter identifier.
func (a *ClaudeAdapter) Name() string {
	return "claude"
}

// LoadEntries discovers and parses Claude Code usage files.
func (a *ClaudeAdapter) LoadEntries(ctx context.Context, opts adapter.LoadOptions) ([]*types.LoadedEntry, error) {
	pricingMap := opts.Pricing
	if pricingMap == nil {
		pricingMap = pricing.LoadDefaultPricing()
	}

	clo := loadOptions{
		Pricing:       pricingMap,
		Timezone:      opts.Timezone,
		Since:         opts.Since,
		Until:         opts.Until,
		JSON:          opts.JSON,
		SingleThread:  opts.SingleThread,
		ProjectFilter: opts.ProjectFilter,
		Verbose:       opts.Verbose,
	}

	entries, err := LoadEntries(clo)
	if err != nil {
		return nil, err
	}
	return FilterLoadedEntriesByDate(entries, clo.Since, clo.Until), nil
}

// Summarize aggregates loaded entries into usage summaries.
func (a *ClaudeAdapter) Summarize(entries []*types.LoadedEntry, kind types.ReportKind) ([]*types.UsageSummary, error) {
	switch kind {
	case types.ReportDaily, types.ReportMonthly, types.ReportWeekly:
		return summarizeByKey(entries,
			func(e *types.LoadedEntry) string { return e.Date },
			func(key string) (string, *string) { return key, nil },
		), nil
	case types.ReportSession:
		// Session mode: group by session_id.
		return sessionSummaries(entries), nil
	default:
		return nil, nil
	}
}

// ReportJSON builds the JSON report structure.
func (a *ClaudeAdapter) ReportJSON(rows []*types.UsageSummary, kind types.ReportKind) (any, error) {
	report := map[string]any{
		"totals": totalsFromRows(rows),
	}
	switch kind {
	case types.ReportDaily:
		report["daily"] = rows
	case types.ReportWeekly:
		report["weekly"] = rows
	case types.ReportMonthly:
		report["monthly"] = rows
	case types.ReportSession:
		report["sessions"] = rows
	}
	return report, nil
}

// Paths returns the Claude Code data directories.
func (a *ClaudeAdapter) Paths() ([]string, error) {
	return ClaudePaths()
}

// IsAvailable checks if Claude Code data exists on the system.
func (a *ClaudeAdapter) IsAvailable() bool {
	paths, err := ClaudePaths()
	if err != nil || len(paths) == 0 {
		return false
	}
	files := UsageFiles(paths, "")
	return len(files) > 0
}

// Helper for session-level summarization.
func sessionSummaries(entries []*types.LoadedEntry) []*types.UsageSummary {
	groups := make(map[string]*sessionAccumulator)
	var order []string

	for _, entry := range entries {
		sid := entry.SessionID
		if _, ok := groups[sid]; !ok {
			groups[sid] = &sessionAccumulator{
				accum: usageAccumulator{
					breakdownIdxs: make(map[string]int),
				},
			}
			order = append(order, sid)
		}
		groups[sid].add(entry)
	}

	var rows []*types.UsageSummary
	for _, sid := range order {
		s := groups[sid]
		row := s.accum.intoSummary()
		row.SessionID = &sid
		row.LastActivity = &s.lastActivity
		if s.firstActivity != "" {
			row.FirstActivity = &s.firstActivity
		}
		row.ProjectPath = &s.projectPath
		if len(s.versions) > 0 {
			row.Versions = s.versions
		}
		rows = append(rows, row)
	}
	return rows
}

type sessionAccumulator struct {
	accum         usageAccumulator
	lastActivity  string
	firstActivity  string
	projectPath   string
	versions      []string
	versionSeen   map[string]bool
}

func (s *sessionAccumulator) add(entry *types.LoadedEntry) {
	s.accum.addEntry(entry)
	s.lastActivity = entry.Timestamp.Format("2006-01-02T15:04:05.000Z")
	if s.firstActivity == "" || entry.Timestamp.Format("2006-01-02T15:04:05.000Z") < s.firstActivity {
		s.firstActivity = entry.Timestamp.Format("2006-01-02T15:04:05.000Z")
	}
	s.projectPath = entry.ProjectPath
	if entry.Data.Version != nil {
		if s.versionSeen == nil {
			s.versionSeen = make(map[string]bool)
		}
		if !s.versionSeen[*entry.Data.Version] {
			s.versionSeen[*entry.Data.Version] = true
			s.versions = append(s.versions, *entry.Data.Version)
		}
	}
}

func totalsFromRows(rows []*types.UsageSummary) map[string]any {
	var input, output, cacheCreate, cacheRead, extra uint64
	var totalCost, credits float64
	for _, row := range rows {
		input += row.InputTokens
		output += row.OutputTokens
		cacheCreate += row.CacheCreation
		cacheRead += row.CacheRead
		extra += row.ExtraTotal
		totalCost += row.TotalCost
		if row.Credits != nil {
			credits += *row.Credits
		}
	}
	m := map[string]any{
		"inputTokens":        input,
		"outputTokens":       output,
		"cacheCreationTokens": cacheCreate,
		"cacheReadTokens":    cacheRead,
		"totalTokens":        input + output + cacheCreate + cacheRead + extra,
		"totalCost":          totalCost,
	}
	if credits > 0 {
		m["credits"] = credits
	}
	return m
}
