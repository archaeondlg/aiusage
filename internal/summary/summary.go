// Package summary provides aggregation and filtering logic for usage reports.
package summary

import (
	"sort"

	"github.com/archhaeondlg/aiusage/internal/dateutil"
	"github.com/archhaeondlg/aiusage/internal/types"
)

// SummarizeByKey groups entries by a key function and produces usage summaries.
// Equivalent to Rust summarize_by_key().
func SummarizeByKey(
	entries []*types.LoadedEntry,
	keyFn func(*types.LoadedEntry) string,
	metaFn func(string) (string, *string),
) []*types.UsageSummary {
	groups := make(map[string]*UsageAccumulator)
	var groupOrder []string

	for _, entry := range entries {
		key := keyFn(entry)
		if _, ok := groups[key]; !ok {
			groups[key] = &UsageAccumulator{
				BreakdownIdxs: make(map[string]int),
			}
			groupOrder = append(groupOrder, key)
		}
		groups[key].AddEntry(entry)
	}

	var rows []*types.UsageSummary
	for _, key := range groupOrder {
		group := groups[key]
		date, project := metaFn(key)
		summary := group.IntoSummary()
		summary.Date = &date
		summary.Project = project
		rows = append(rows, summary)
	}
	return rows
}

// UsageAccumulator aggregates entries into a UsageSummary.
type UsageAccumulator struct {
	Counts        types.TokenCounts
	Cost          float64
	ReasoningOutputTokens uint64
	Credits       *float64
	MessageCount  *uint64
	Models        []string
	Breakdowns    []types.ModelBreakdown
	BreakdownIdxs map[string]int
}

// AddEntry merges a loaded entry into the accumulator.
func (a *UsageAccumulator) AddEntry(entry *types.LoadedEntry) {
	usage := entry.Data.Message.Usage
	a.Counts.AddUsage(usage)
	a.Counts.ExtraTotalTokens += entry.ExtraTotalTokens
	a.Cost += entry.Cost
	a.ReasoningOutputTokens += entry.ReasoningOutputTokens

	if entry.Credits != nil {
		if a.Credits == nil {
			a.Credits = new(float64)
		}
		*a.Credits += *entry.Credits
	}
	if entry.MessageCount != nil {
		if a.MessageCount == nil {
			a.MessageCount = new(uint64)
		}
		*a.MessageCount += *entry.MessageCount
	}

	if entry.Model != nil {
		model := *entry.Model
		idx, ok := a.BreakdownIdxs[model]
		if !ok {
			idx = len(a.Breakdowns)
			a.BreakdownIdxs[model] = idx
			a.Models = append(a.Models, model)
			a.Breakdowns = append(a.Breakdowns, types.ModelBreakdown{
				ModelName: model,
			})
		}
		bd := &a.Breakdowns[idx]
		bd.InputTokens += usage.InputTokens
		bd.OutputTokens += usage.OutputTokens
		bd.ReasoningOutputTokens += entry.ReasoningOutputTokens
		bd.CacheCreation += usage.CacheCreationTokenCount()
		bd.CacheRead += usage.CacheReadInputTokens
		bd.ExtraTotalTokens += entry.ExtraTotalTokens
		bd.Cost += entry.Cost
		if entry.MissingPricingModel != nil {
			bd.MissingPricing = true
		}
	}
}

// IntoSummary converts the accumulator to a UsageSummary.
func (a *UsageAccumulator) IntoSummary() *types.UsageSummary {
	sort.SliceStable(a.Breakdowns, func(i, j int) bool {
		return a.Breakdowns[i].Cost > a.Breakdowns[j].Cost
	})
	return &types.UsageSummary{
		InputTokens:           a.Counts.InputTokens,
		OutputTokens:          a.Counts.OutputTokens,
		ReasoningOutputTokens: a.ReasoningOutputTokens,
		CacheCreation:         a.Counts.CacheCreation,
		CacheRead:             a.Counts.CacheRead,
		ExtraTotal:            a.Counts.ExtraTotalTokens,
		TotalCost:             a.Cost,
		Credits:               a.Credits,
		MessageCount:          a.MessageCount,
		ModelsUsed:            a.Models,
		ModelBreakdowns:       a.Breakdowns,
	}
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Session accumulator
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

// SessionAccumulator tracks per-session metadata alongside usage.
type SessionAccumulator struct {
	Usage         UsageAccumulator
	Latest        *types.LoadedEntry
	Earliest      *types.LoadedEntry
	Versions      []string
	versionSeen   map[string]bool
}

// SessionAccumulatorInit initializes a new SessionAccumulator.
func NewSessionAccumulator() *SessionAccumulator {
	return &SessionAccumulator{
		Usage: UsageAccumulator{
			BreakdownIdxs: make(map[string]int),
		},
		versionSeen: make(map[string]bool),
	}
}

// AddEntry merges an entry and updates session metadata.
func (s *SessionAccumulator) AddEntry(entry *types.LoadedEntry) {
	s.Usage.AddEntry(entry)
	if s.Latest == nil || entry.Timestamp.After(s.Latest.Timestamp) {
		s.Latest = entry
	}
	if s.Earliest == nil || entry.Timestamp.Before(s.Earliest.Timestamp) {
		s.Earliest = entry
	}
	if entry.Data.Version != nil {
		if !s.versionSeen[*entry.Data.Version] {
			s.versionSeen[*entry.Data.Version] = true
			s.Versions = append(s.Versions, *entry.Data.Version)
		}
	}
}

// IntoSummary converts to UsageSummary with session metadata.
func (s *SessionAccumulator) IntoSummary() *types.UsageSummary {
	summary := s.Usage.IntoSummary()
	if s.Latest != nil {
		sid := s.Latest.SessionID
		summary.SessionID = &sid
		pp := s.Latest.ProjectPath
		summary.ProjectPath = &pp
		la := s.Latest.Timestamp.Format(dateutil.RFC3339Z)
		summary.LastActivity = &la
	}
	if s.Earliest != nil {
		fa := s.Earliest.Timestamp.Format(dateutil.RFC3339Z)
		summary.FirstActivity = &fa
	}
	if len(s.Versions) > 0 {
		summary.Versions = s.Versions
	}
	return summary
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Bucket aggregation
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

// SummarizeByBucket rolls up summaries by month or week.
func SummarizeByBucket(rows []*types.UsageSummary, kind types.ReportKind, weekStart types.WeekDay) []*types.UsageSummary {
	groups := make(map[string][]*types.UsageSummary)
	var order []string

	for _, row := range rows {
		date := ""
		if row.Date != nil {
			date = *row.Date
		}
		if date == "" {
			continue
		}
		bucket := dateutil.BucketKey(date, kind, weekStart)
		if _, ok := groups[bucket]; !ok {
			order = append(order, bucket)
		}
		groups[bucket] = append(groups[bucket], row)
	}

	var result []*types.UsageSummary
	for _, bucket := range order {
		summary := AggregateSummaries(groups[bucket])
		switch kind {
		case types.ReportMonthly:
			summary.Month = &bucket
		case types.ReportWeekly:
			summary.Week = &bucket
		}
		result = append(result, summary)
	}
	return result
}

// AggregateSummaries merges multiple summaries into one.
func AggregateSummaries(rows []*types.UsageSummary) *types.UsageSummary {
	s := &types.UsageSummary{
		ModelsUsed:      make([]string, 0),
		ModelBreakdowns: make([]types.ModelBreakdown, 0),
	}
	seenModels := make(map[string]bool)
	breakdownIdxs := make(map[string]int)

	for _, row := range rows {
		s.InputTokens += row.InputTokens
		s.OutputTokens += row.OutputTokens
		s.ReasoningOutputTokens += row.ReasoningOutputTokens
		s.CacheCreation += row.CacheCreation
		s.CacheRead += row.CacheRead
		s.ExtraTotal += row.ExtraTotal
		s.TotalCost += row.TotalCost
		if row.Credits != nil {
			if s.Credits == nil {
				s.Credits = new(float64)
			}
			*s.Credits += *row.Credits
		}
		if row.MessageCount != nil {
			if s.MessageCount == nil {
				s.MessageCount = new(uint64)
			}
			*s.MessageCount += *row.MessageCount
		}
		for _, model := range row.ModelsUsed {
			if !seenModels[model] {
				seenModels[model] = true
				s.ModelsUsed = append(s.ModelsUsed, model)
			}
		}
		for _, item := range row.ModelBreakdowns {
			idx, ok := breakdownIdxs[item.ModelName]
			if !ok {
				idx = len(s.ModelBreakdowns)
				breakdownIdxs[item.ModelName] = idx
				s.ModelBreakdowns = append(s.ModelBreakdowns, types.ModelBreakdown{
					ModelName: item.ModelName,
				})
			}
			bd := &s.ModelBreakdowns[idx]
			bd.InputTokens += item.InputTokens
			bd.OutputTokens += item.OutputTokens
			bd.ReasoningOutputTokens += item.ReasoningOutputTokens
			bd.CacheCreation += item.CacheCreation
			bd.CacheRead += item.CacheRead
			bd.Cost += item.Cost
			bd.MissingPricing = bd.MissingPricing || item.MissingPricing
		}
	}

	sort.SliceStable(s.ModelBreakdowns, func(i, j int) bool {
		return s.ModelBreakdowns[i].Cost > s.ModelBreakdowns[j].Cost
	})
	return s
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Filter and sort
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

// FilterAndSort filters by date range and sorts summaries.
func FilterAndSort(rows []*types.UsageSummary, since, until string, order types.SortOrder, dateFn func(*types.UsageSummary) string) []*types.UsageSummary {
	var filtered []*types.UsageSummary
	sinceNorm := dateutil.NormalizeDateBound(since)
	untilNorm := dateutil.NormalizeDateBound(until)
	for _, row := range rows {
		date := dateutil.NormalizeDateBound(dateFn(row))
		if sinceNorm != "" && date < sinceNorm {
			continue
		}
		if untilNorm != "" && date > untilNorm {
			continue
		}
		filtered = append(filtered, row)
	}
	SortSummaries(filtered, order, dateFn)
	return filtered
}

// SortSummaries sorts summaries by date.
func SortSummaries(rows []*types.UsageSummary, order types.SortOrder, dateFn func(*types.UsageSummary) string) {
	sort.SliceStable(rows, func(i, j int) bool {
		a := dateFn(rows[i])
		b := dateFn(rows[j])
		if order == types.SortDesc {
			return a > b
		}
		return a < b
	})
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Warnings
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

// MissingPricingWarnings returns warning messages for models without pricing data.
func MissingPricingWarnings(rows []*types.UsageSummary) []string {
	seen := make(map[string]bool)
	var warnings []string
	for _, row := range rows {
		for _, bd := range row.ModelBreakdowns {
			if bd.MissingPricing && !seen[bd.ModelName] {
				seen[bd.ModelName] = true
				warnings = append(warnings,
					"WARN  Missing pricing for "+bd.ModelName+
						"; cost excludes this model. Add it to the pricing section in aiusage.json.")
			}
		}
	}
	return warnings
}
