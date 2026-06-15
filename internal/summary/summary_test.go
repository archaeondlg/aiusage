package summary

import (
	"testing"
	"time"

	"github.com/archhaeondlg/aiusage/internal/types"
)

func mkEntry(t time.Time, date, project string, input, output, cacheCre, cacheRead, extra uint64, cost float64, model string) *types.LoadedEntry {
	m := model
	return &types.LoadedEntry{
		Timestamp: t,
		Date:      date,
		Project:   project,
		Cost:      cost,
		ExtraTotalTokens: extra,
		Data: types.UsageEntry{
			Message: types.UsageMessage{
				Usage: types.TokenUsageRaw{
					InputTokens:              input,
					OutputTokens:             output,
					CacheCreationInputTokens: cacheCre,
					CacheReadInputTokens:     cacheRead,
				},
				Model: &m,
			},
		},
		Model: &m,
	}
}

func mkEntryWithCredits(t time.Time, date, project string, input, output uint64, cost float64, credits float64, model string) *types.LoadedEntry {
	e := mkEntry(t, date, project, input, output, 0, 0, 0, cost, model)
	e.Credits = &credits
	return e
}

func mkEntryWithMsgCount(t time.Time, date, project string, input, output uint64, cost float64, msgCount uint64, model string) *types.LoadedEntry {
	e := mkEntry(t, date, project, input, output, 0, 0, 0, cost, model)
	e.MessageCount = &msgCount
	return e
}

func mkEntryMissingPricing(t time.Time, date, project string, cost float64, model string) *types.LoadedEntry {
	e := mkEntry(t, date, project, 0, 0, 0, 0, 0, cost, model)
	e.MissingPricingModel = &model
	return e
}

func TestUsageAccumulatorAddEntry(t *testing.T) {
	now := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	e1 := mkEntry(now, "2026-06-15", "projA", 100, 200, 10, 5, 0, 1.5, "claude-3-haiku")
	e2 := mkEntry(now, "2026-06-15", "projA", 300, 400, 20, 15, 0, 2.5, "claude-opus-4")

	acc := &UsageAccumulator{BreakdownIdxs: make(map[string]int)}
	acc.AddEntry(e1)
	acc.AddEntry(e2)

	if got := acc.Counts.InputTokens; got != 400 {
		t.Errorf("InputTokens = %d, want 400", got)
	}
	if got := acc.Counts.OutputTokens; got != 600 {
		t.Errorf("OutputTokens = %d, want 600", got)
	}
	if got := acc.Counts.CacheCreation; got != 30 {
		t.Errorf("CacheCreation = %d, want 30", got)
	}
	if got := acc.Counts.CacheRead; got != 20 {
		t.Errorf("CacheRead = %d, want 20", got)
	}
	if got := acc.Cost; got != 4.0 {
		t.Errorf("Cost = %f, want 4.0", got)
	}
}

func TestUsageAccumulatorIntoSummary(t *testing.T) {
	now := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	date := "2026-06-15"
	e1 := mkEntry(now, date, "projA", 100, 200, 10, 5, 0, 1.5, "claude-3-haiku")

	acc := &UsageAccumulator{BreakdownIdxs: make(map[string]int)}
	acc.AddEntry(e1)
	summary := acc.IntoSummary()

	if summary.InputTokens != 100 {
		t.Errorf("InputTokens = %d, want 100", summary.InputTokens)
	}
	if summary.OutputTokens != 200 {
		t.Errorf("OutputTokens = %d, want 200", summary.OutputTokens)
	}
	if summary.TotalCost != 1.5 {
		t.Errorf("TotalCost = %f, want 1.5", summary.TotalCost)
	}
}

func TestUsageAccumulatorCredits(t *testing.T) {
	now := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	e1 := mkEntryWithCredits(now, "2026-06-15", "projA", 100, 200, 1.0, 0.5, "claude-3-haiku")
	e2 := mkEntryWithCredits(now, "2026-06-15", "projA", 100, 200, 1.0, 0.3, "claude-3-haiku")

	acc := &UsageAccumulator{BreakdownIdxs: make(map[string]int)}
	acc.AddEntry(e1)
	acc.AddEntry(e2)

	if acc.Credits == nil {
		t.Fatal("Credits should not be nil")
	}
	if *acc.Credits != 0.8 {
		t.Errorf("Credits = %f, want 0.8", *acc.Credits)
	}
}

func TestUsageAccumulatorModelBreakdowns(t *testing.T) {
	now := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	e1 := mkEntry(now, "2026-06-15", "projA", 100, 200, 0, 0, 0, 1.0, "claude-3-haiku")
	e2 := mkEntry(now, "2026-06-15", "projA", 300, 400, 0, 0, 0, 2.0, "claude-opus-4")

	acc := &UsageAccumulator{BreakdownIdxs: make(map[string]int)}
	acc.AddEntry(e1)
	acc.AddEntry(e2)

	if len(acc.Breakdowns) != 2 {
		t.Fatalf("len(Breakdowns) = %d, want 2", len(acc.Breakdowns))
	}
	for _, bd := range acc.Breakdowns {
		switch bd.ModelName {
		case "claude-3-haiku":
			if bd.InputTokens != 100 || bd.OutputTokens != 200 || bd.Cost != 1.0 {
				t.Errorf("haiku: got %+v", bd)
			}
		case "claude-opus-4":
			if bd.InputTokens != 300 || bd.OutputTokens != 400 || bd.Cost != 2.0 {
				t.Errorf("opus: got %+v", bd)
			}
		default:
			t.Errorf("unexpected model: %s", bd.ModelName)
		}
	}
}

func TestSessionAccumulator(t *testing.T) {
	t1 := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	version := "1.0.0"
	e1 := &types.LoadedEntry{
		Timestamp: t1,
		Date:      "2026-06-15",
		SessionID: "sess-1",
		Cost:      1.0,
		Data: types.UsageEntry{
			Message: types.UsageMessage{
				Usage: types.TokenUsageRaw{
					InputTokens:  100,
					OutputTokens: 200,
				},
			},
			Version: &version,
		},
	}
	e2 := &types.LoadedEntry{
		Timestamp: t2,
		Date:      "2026-06-15",
		SessionID: "sess-1",
		Cost:      2.0,
		Data: types.UsageEntry{
			Message: types.UsageMessage{
				Usage: types.TokenUsageRaw{
					InputTokens:  300,
					OutputTokens: 400,
				},
			},
		},
	}

	acc := NewSessionAccumulator()
	acc.AddEntry(e1)
	acc.AddEntry(e2)

	if acc.Usage.Cost != 3.0 {
		t.Errorf("Cost = %f, want 3.0", acc.Usage.Cost)
	}
	if acc.Latest != e2 {
		t.Error("Latest should be e2")
	}
	if acc.Earliest != e1 {
		t.Error("Earliest should be e1")
	}
	if len(acc.Versions) != 1 || acc.Versions[0] != "1.0.0" {
		t.Errorf("Versions = %v, want [1.0.0]", acc.Versions)
	}

	summary := acc.IntoSummary()
	if summary.SessionID == nil || *summary.SessionID != "sess-1" {
		t.Errorf("SessionID = %v, want sess-1", summary.SessionID)
	}
	if summary.LastActivity == nil {
		t.Error("LastActivity should not be nil")
	}
	if summary.FirstActivity == nil {
		t.Error("FirstActivity should not be nil")
	}
}

func TestSummarizeByKeyByDate(t *testing.T) {
	now := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	entries := []*types.LoadedEntry{
		mkEntry(now, "2026-06-15", "projA", 100, 200, 0, 0, 0, 1.0, "claude-3-haiku"),
		mkEntry(now, "2026-06-16", "projA", 300, 400, 0, 0, 0, 2.0, "claude-3-haiku"),
	}

	rows := SummarizeByKey(entries,
		func(e *types.LoadedEntry) string { return e.Date },
		func(key string) (string, *string) { return key, nil },
	)

	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(rows))
	}
	if *rows[0].Date != "2026-06-15" || rows[0].TotalCost != 1.0 {
		t.Errorf("row0: date=%s cost=%f", *rows[0].Date, rows[0].TotalCost)
	}
	if *rows[1].Date != "2026-06-16" || rows[1].TotalCost != 2.0 {
		t.Errorf("row1: date=%s cost=%f", *rows[1].Date, rows[1].TotalCost)
	}
}

func TestSummarizeByKeyByProject(t *testing.T) {
	now := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	projA := "projA"
	projB := "projB"
	entries := []*types.LoadedEntry{
		mkEntry(now, "2026-06-15", projA, 100, 200, 0, 0, 0, 1.0, "claude-3-haiku"),
		mkEntry(now, "2026-06-15", projB, 300, 400, 0, 0, 0, 2.0, "claude-3-haiku"),
	}

	rows := SummarizeByKey(entries,
		func(e *types.LoadedEntry) string { return e.Project },
		func(key string) (string, *string) { return "", &key },
	)

	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(rows))
	}
	if *rows[0].Project != "projA" || rows[0].TotalCost != 1.0 {
		t.Errorf("row0: project=%s cost=%f", *rows[0].Project, rows[0].TotalCost)
	}
	if *rows[1].Project != "projB" || rows[1].TotalCost != 2.0 {
		t.Errorf("row1: project=%s cost=%f", *rows[1].Project, rows[1].TotalCost)
	}
}

func TestAggregateSummaries(t *testing.T) {
	now := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	e1 := mkEntry(now, "2026-06-15", "projA", 100, 200, 10, 5, 0, 1.0, "claude-3-haiku")
	e2 := mkEntry(now, "2026-06-15", "projA", 300, 400, 20, 15, 0, 2.0, "claude-opus-4")

	acc := &UsageAccumulator{BreakdownIdxs: make(map[string]int)}
	acc.AddEntry(e1)
	acc.AddEntry(e2)
	rows := []*types.UsageSummary{acc.IntoSummary()}

	result := AggregateSummaries(rows)
	if result.InputTokens != 400 || result.OutputTokens != 600 {
		t.Errorf("tokens: input=%d output=%d", result.InputTokens, result.OutputTokens)
	}
	if result.TotalCost != 3.0 {
		t.Errorf("TotalCost = %f, want 3.0", result.TotalCost)
	}
}

func TestAggregateSummariesMultipleRows(t *testing.T) {
	credits := 0.5
	msgCount := uint64(10)
	r1 := &types.UsageSummary{
		InputTokens: 100, OutputTokens: 200, TotalCost: 1.0,
		Credits: &credits, MessageCount: &msgCount,
		ModelsUsed: []string{"claude-3-haiku"},
		ModelBreakdowns: []types.ModelBreakdown{
			{ModelName: "claude-3-haiku", InputTokens: 100, OutputTokens: 200, Cost: 1.0},
		},
	}
	r2 := &types.UsageSummary{
		InputTokens: 300, OutputTokens: 400, TotalCost: 2.0,
		ModelsUsed: []string{"claude-opus-4"},
		ModelBreakdowns: []types.ModelBreakdown{
			{ModelName: "claude-opus-4", InputTokens: 300, OutputTokens: 400, Cost: 2.0},
		},
	}

	result := AggregateSummaries([]*types.UsageSummary{r1, r2})
	if result.InputTokens != 400 || result.OutputTokens != 600 || result.TotalCost != 3.0 {
		t.Errorf("aggregation: in=%d out=%d cost=%f", result.InputTokens, result.OutputTokens, result.TotalCost)
	}
	if result.Credits == nil || *result.Credits != 0.5 {
		t.Errorf("Credits = %v", result.Credits)
	}
	if result.MessageCount == nil || *result.MessageCount != 10 {
		t.Errorf("MessageCount = %v", result.MessageCount)
	}
	if len(result.ModelBreakdowns) != 2 {
		t.Fatalf("len(ModelBreakdowns) = %d, want 2", len(result.ModelBreakdowns))
	}
}

func TestAggregateSummariesMissingPricing(t *testing.T) {
	r1 := &types.UsageSummary{
		ModelBreakdowns: []types.ModelBreakdown{
			{ModelName: "claude-3-haiku", Cost: 1.0, MissingPricing: true},
		},
	}
	r2 := &types.UsageSummary{
		ModelBreakdowns: []types.ModelBreakdown{
			{ModelName: "claude-3-haiku", Cost: 2.0, MissingPricing: false},
		},
	}
	result := AggregateSummaries([]*types.UsageSummary{r1, r2})
	if len(result.ModelBreakdowns) != 1 {
		t.Fatalf("len = %d, want 1", len(result.ModelBreakdowns))
	}
	if !result.ModelBreakdowns[0].MissingPricing {
		t.Error("MissingPricing should be true (r1 had it)")
	}
	if result.ModelBreakdowns[0].Cost != 3.0 {
		t.Errorf("Cost = %f, want 3.0", result.ModelBreakdowns[0].Cost)
	}
}

func TestSummarizeByBucketMonthly(t *testing.T) {
	d1 := "2026-06-15"
	d2 := "2026-06-20"
	d3 := "2026-07-01"
	rows := []*types.UsageSummary{
		{Date: &d1, InputTokens: 100, TotalCost: 1.0},
		{Date: &d2, InputTokens: 200, TotalCost: 2.0},
		{Date: &d3, InputTokens: 300, TotalCost: 3.0},
	}

	result := SummarizeByBucket(rows, types.ReportMonthly, types.WeekMonday)
	if len(result) != 2 {
		t.Fatalf("len = %d, want 2", len(result))
	}
	if result[0].Month == nil || *result[0].Month != "2026-06" {
		t.Errorf("result[0].Month = %v", result[0].Month)
	}
	if result[0].InputTokens != 300 || result[0].TotalCost != 3.0 {
		t.Errorf("result[0]: in=%d cost=%f", result[0].InputTokens, result[0].TotalCost)
	}
	if result[1].Month == nil || *result[1].Month != "2026-07" {
		t.Errorf("result[1].Month = %v", result[1].Month)
	}
	if result[1].InputTokens != 300 || result[1].TotalCost != 3.0 {
		t.Errorf("result[1]: in=%d cost=%f", result[1].InputTokens, result[1].TotalCost)
	}
}

func TestSummarizeByBucketWeekly(t *testing.T) {
	d1 := "2026-06-15" // Monday
	d2 := "2026-06-16" // Tuesday
	d3 := "2026-06-22" // Monday
	rows := []*types.UsageSummary{
		{Date: &d1, InputTokens: 100, TotalCost: 1.0},
		{Date: &d2, InputTokens: 200, TotalCost: 2.0},
		{Date: &d3, InputTokens: 300, TotalCost: 3.0},
	}

	result := SummarizeByBucket(rows, types.ReportWeekly, types.WeekMonday)
	if len(result) != 2 {
		t.Fatalf("len = %d, want 2", len(result))
	}
	if result[0].Week == nil || *result[0].Week != "2026-06-15" {
		t.Errorf("result[0].Week = %v", result[0].Week)
	}
	if result[0].InputTokens != 300 || result[0].TotalCost != 3.0 {
		t.Errorf("result[0]: in=%d cost=%f", result[0].InputTokens, result[0].TotalCost)
	}
	if result[1].Week == nil || *result[1].Week != "2026-06-22" {
		t.Errorf("result[1].Week = %v", result[1].Week)
	}
}

func TestFilterAndSortAsc(t *testing.T) {
	d1 := "2026-06-15"
	d2 := "2026-06-16"
	d3 := "2026-06-17"
	rows := []*types.UsageSummary{
		{Date: &d1, InputTokens: 100},
		{Date: &d2, InputTokens: 200},
		{Date: &d3, InputTokens: 300},
	}

	dateFn := func(s *types.UsageSummary) string {
		if s.Date != nil {
			return *s.Date
		}
		return ""
	}
	result := FilterAndSort(rows, "2026-06-15", "2026-06-16", types.SortAsc, dateFn)
	if len(result) != 2 {
		t.Fatalf("len = %d, want 2", len(result))
	}
	if *result[0].Date != "2026-06-15" || *result[1].Date != "2026-06-16" {
		t.Errorf("order: %s %s", *result[0].Date, *result[1].Date)
	}
}

func TestFilterAndSortDesc(t *testing.T) {
	d1 := "2026-06-15"
	d2 := "2026-06-16"
	d3 := "2026-06-17"
	rows := []*types.UsageSummary{
		{Date: &d3, InputTokens: 300},
		{Date: &d1, InputTokens: 100},
		{Date: &d2, InputTokens: 200},
	}

	dateFn := func(s *types.UsageSummary) string {
		if s.Date != nil {
			return *s.Date
		}
		return ""
	}
	result := FilterAndSort(rows, "", "", types.SortDesc, dateFn)
	if len(result) != 3 {
		t.Fatalf("len = %d, want 3", len(result))
	}
	if *result[0].Date != "2026-06-17" || *result[2].Date != "2026-06-15" {
		t.Errorf("desc order: [%s %s %s]", *result[0].Date, *result[1].Date, *result[2].Date)
	}
}

func TestSortSummaries(t *testing.T) {
	d1 := "2026-06-15"
	d2 := "2026-06-16"
	d3 := "2026-06-17"
	rows := []*types.UsageSummary{
		{Date: &d2, InputTokens: 200},
		{Date: &d3, InputTokens: 300},
		{Date: &d1, InputTokens: 100},
	}
	dateFn := func(s *types.UsageSummary) string {
		if s.Date != nil {
			return *s.Date
		}
		return ""
	}

	SortSummaries(rows, types.SortAsc, dateFn)
	if *rows[0].Date != "2026-06-15" || *rows[2].Date != "2026-06-17" {
		t.Errorf("asc: [%s %s %s]", *rows[0].Date, *rows[1].Date, *rows[2].Date)
	}

	SortSummaries(rows, types.SortDesc, dateFn)
	if *rows[0].Date != "2026-06-17" || *rows[2].Date != "2026-06-15" {
		t.Errorf("desc: [%s %s %s]", *rows[0].Date, *rows[1].Date, *rows[2].Date)
	}
}

func TestFilterAndSortEmpty(t *testing.T) {
	var rows []*types.UsageSummary
	dateFn := func(s *types.UsageSummary) string { return "" }
	result := FilterAndSort(rows, "", "", types.SortAsc, dateFn)
	if len(result) != 0 {
		t.Errorf("len = %d, want 0", len(result))
	}
}

func TestMissingPricingWarnings(t *testing.T) {
	rows := []*types.UsageSummary{
		{
			ModelBreakdowns: []types.ModelBreakdown{
				{ModelName: "claude-opus-4", MissingPricing: true},
				{ModelName: "claude-3-haiku", MissingPricing: false},
			},
		},
		{
			ModelBreakdowns: []types.ModelBreakdown{
				{ModelName: "claude-opus-4", MissingPricing: true}, // duplicate
			},
		},
	}

	warnings := MissingPricingWarnings(rows)
	if len(warnings) != 1 {
		t.Fatalf("len(warnings) = %d, want 1", len(warnings))
	}
	if !contains(warnings[0], "claude-opus-4") {
		t.Errorf("warning should mention claude-opus-4: %s", warnings[0])
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestUsageAccumulatorIntoSummaryBreakdownSort(t *testing.T) {
	now := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	e1 := mkEntry(now, "2026-06-15", "projA", 0, 0, 0, 0, 0, 10.0, "model-A")
	e2 := mkEntry(now, "2026-06-15", "projA", 0, 0, 0, 0, 0, 5.0, "model-B")

	acc := &UsageAccumulator{BreakdownIdxs: make(map[string]int)}
	acc.AddEntry(e1)
	acc.AddEntry(e2)
	summary := acc.IntoSummary()

	if len(summary.ModelBreakdowns) != 2 {
		t.Fatalf("len = %d, want 2", len(summary.ModelBreakdowns))
	}
	// Sort by cost desc: model-A (10.0) first, model-B (5.0) second
	if summary.ModelBreakdowns[0].ModelName != "model-A" || summary.ModelBreakdowns[1].ModelName != "model-B" {
		t.Errorf("breakdown order: [%s %s]", summary.ModelBreakdowns[0].ModelName, summary.ModelBreakdowns[1].ModelName)
	}
}

func TestUsageAccumulatorNilModel(t *testing.T) {
	now := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	e := &types.LoadedEntry{
		Timestamp: now,
		Date:      "2026-06-15",
		Cost:      1.0,
		Data: types.UsageEntry{
			Message: types.UsageMessage{
				Usage: types.TokenUsageRaw{
					InputTokens:  100,
					OutputTokens: 200,
				},
			},
		},
	}

	acc := &UsageAccumulator{BreakdownIdxs: make(map[string]int)}
	acc.AddEntry(e)
	if len(acc.Breakdowns) != 0 {
		t.Errorf("breakdowns with nil model should be empty, got %d", len(acc.Breakdowns))
	}
}

func TestSessionAccumulatorNilLatest(t *testing.T) {
	acc := NewSessionAccumulator()
	summary := acc.IntoSummary()
	if summary.SessionID != nil {
		t.Error("SessionID should be nil for empty accumulator")
	}
	if summary.LastActivity != nil {
		t.Error("LastActivity should be nil for empty accumulator")
	}
}

func TestBucketWithNilDate(t *testing.T) {
	rows := []*types.UsageSummary{
		{Date: nil, InputTokens: 100},
		{Date: nil, InputTokens: 200},
	}
	result := SummarizeByBucket(rows, types.ReportMonthly, types.WeekMonday)
	if len(result) != 0 {
		t.Errorf("nil dates should be skipped, got %d rows", len(result))
	}
}

func TestSummarizeByKeyEmpty(t *testing.T) {
	result := SummarizeByKey(nil,
		func(e *types.LoadedEntry) string { return e.Date },
		func(key string) (string, *string) { return key, nil },
	)
	if len(result) != 0 {
		t.Errorf("empty input should produce empty output, got %d", len(result))
	}
}
