package blocks

import (
	"testing"
	"time"

	"github.com/archhaeondlg/aiusage/internal/types"
)

func tEntry(id string, ts time.Time, input, output uint64, cost float64, model string) *types.LoadedEntry {
	m := model
	return &types.LoadedEntry{
		SessionID: id,
		Timestamp: ts,
		Date:      ts.Format("2006-01-02"),
		Cost:      cost,
		Data: types.UsageEntry{
			Message: types.UsageMessage{
				Usage: types.TokenUsageRaw{
					InputTokens:  input,
					OutputTokens: output,
				},
				Model: &m,
			},
		},
		Model: &m,
	}
}

func TestNewBlock(t *testing.T) {
	now := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	e := tEntry("s1", now, 100, 200, 1.5, "claude-3-haiku")

	b := newBlock(e)
	if b.StartTime != now || b.EndTime != now {
		t.Errorf("start/end = %v / %v", b.StartTime, b.EndTime)
	}
	if b.TokenCounts.Total() != 300 {
		t.Errorf("total tokens = %d, want 300", b.TokenCounts.Total())
	}
	if b.CostUSD != 1.5 {
		t.Errorf("cost = %f, want 1.5", b.CostUSD)
	}
	if len(b.Entries) != 1 {
		t.Errorf("len(entries) = %d", len(b.Entries))
	}
	if b.IsActive {
		t.Error("new block should not be active")
	}
}

func TestIdentifyBlocksSingle(t *testing.T) {
	now := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	e := tEntry("s1", now, 100, 200, 1.0, "claude-3-haiku")

	blocks := identifyBlocks([]*types.LoadedEntry{e}, 30*time.Minute)
	if len(blocks) != 1 {
		t.Fatalf("len = %d, want 1", len(blocks))
	}
	if !blocks[0].IsActive {
		t.Error("single block should be active")
	}
	if blocks[0].TokenCounts.Total() != 300 {
		t.Errorf("total = %d", blocks[0].TokenCounts.Total())
	}
}

func TestIdentifyBlocksMergeWithinGap(t *testing.T) {
	t1 := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 6, 15, 10, 15, 0, 0, time.UTC) // 15 min gap
	e1 := tEntry("s1", t1, 100, 200, 1.0, "claude-3-haiku")
	e2 := tEntry("s1", t2, 300, 400, 2.0, "claude-opus-4")

	blocks := identifyBlocks([]*types.LoadedEntry{e1, e2}, 30*time.Minute)
	if len(blocks) != 1 {
		t.Fatalf("len = %d, want 1 (merged within session duration)", len(blocks))
	}
	if blocks[0].TokenCounts.Total() != 1000 {
		t.Errorf("total = %d, want 1000", blocks[0].TokenCounts.Total())
	}
	if blocks[0].CostUSD != 3.0 {
		t.Errorf("cost = %f, want 3.0", blocks[0].CostUSD)
	}
	if len(blocks[0].Models) != 2 {
		t.Errorf("models = %v, want 2", blocks[0].Models)
	}
}

func TestIdentifyBlocksNewBlockAfterGap(t *testing.T) {
	t1 := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 6, 15, 11, 0, 0, 0, time.UTC) // 60 min gap
	e1 := tEntry("s1", t1, 100, 200, 1.0, "claude-3-haiku")
	e2 := tEntry("s2", t2, 300, 400, 2.0, "claude-opus-4")

	blocks := identifyBlocks([]*types.LoadedEntry{e1, e2}, 30*time.Minute)
	if len(blocks) != 2 {
		t.Fatalf("len = %d, want 2", len(blocks))
	}
	if blocks[0].IsActive {
		t.Error("first block should not be active (closed)")
	}
	if !blocks[1].IsActive {
		t.Error("last block should be active")
	}
	if blocks[0].ActualEndTime == nil {
		t.Error("first block should have ActualEndTime set")
	}
}

func TestIdentifyBlocksGapInsertion(t *testing.T) {
	t1 := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC) // 2h gap > 2*30min
	e1 := tEntry("s1", t1, 100, 200, 1.0, "claude-3-haiku")
	e2 := tEntry("s2", t2, 300, 400, 2.0, "claude-opus-4")

	blocks := identifyBlocks([]*types.LoadedEntry{e1, e2}, 30*time.Minute)
	if len(blocks) != 3 {
		t.Fatalf("len = %d, want 3 (block + gap + block)", len(blocks))
	}
	if !blocks[1].IsGap {
		t.Error("middle block should be a gap")
	}
	if blocks[0].IsActive {
		t.Error("first block should be closed")
	}
	if !blocks[2].IsActive {
		t.Error("last block should be active")
	}
}

func TestIdentifyBlocksEmpty(t *testing.T) {
	blocks := identifyBlocks(nil, 30*time.Minute)
	if blocks != nil {
		t.Errorf("expected nil, got %v", blocks)
	}
}

func TestCalculateBurnRate(t *testing.T) {
	start := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 15, 11, 0, 0, 0, time.UTC) // 60 min
	block := &types.SessionBlock{
		StartTime: start,
		EndTime:   end,
		TokenCounts: types.TokenCounts{
			InputTokens:  6000,
			OutputTokens: 0,
		},
		CostUSD: 6.0,
	}

	rate := CalculateBurnRate(block)
	if rate.TokensPerMinute != 100 {
		t.Errorf("TPM = %f, want 100", rate.TokensPerMinute)
	}
	if rate.CostPerHour != 6.0 {
		t.Errorf("CPH = %f, want 6.0", rate.CostPerHour)
	}
}

func TestCalculateBurnRateShortDuration(t *testing.T) {
	start := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	end := start.Add(1 * time.Millisecond) // near-zero duration
	block := &types.SessionBlock{
		StartTime: start,
		EndTime:   end,
		TokenCounts: types.TokenCounts{
			InputTokens: 100,
		},
		CostUSD: 0.01,
	}

	rate := CalculateBurnRate(block)
	if rate.TokensPerMinute != 1000 {
		t.Errorf("TPM = %f, want 1000", rate.TokensPerMinute)
	}
}

func TestProjectLimitNotExceeded(t *testing.T) {
	start := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 15, 11, 0, 0, 0, time.UTC)
	block := &types.SessionBlock{
		StartTime: start,
		EndTime:   end,
		TokenCounts: types.TokenCounts{
			InputTokens: 100000,
		},
		CostUSD: 2.0,
	}

	proj := ProjectLimit(block, 500000)
	if proj.RemainingMinutes == 0 {
		t.Error("remaining should be > 0 when under limit")
	}
	// 100k tokens in 60 min = 1666.67 TPM, remaining 400k tokens = 240 min
	if proj.RemainingMinutes < 200 || proj.RemainingMinutes > 260 {
		t.Errorf("remaining mins = %d, expect ~240", proj.RemainingMinutes)
	}
	if proj.TotalTokens != 100000 {
		t.Errorf("total = %d", proj.TotalTokens)
	}
}

func TestProjectLimitExceeded(t *testing.T) {
	block := &types.SessionBlock{
		TokenCounts: types.TokenCounts{
			InputTokens: 600000,
		},
		CostUSD: 10.0,
	}

	proj := ProjectLimit(block, 500000)
	if proj.RemainingMinutes != 0 {
		t.Errorf("remaining = %d, want 0", proj.RemainingMinutes)
	}
}

func TestProjectLimitZeroRate(t *testing.T) {
	block := &types.SessionBlock{
		StartTime:    time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC),
		EndTime:      time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC),
		TokenCounts:  types.TokenCounts{},
		CostUSD:      0,
	}

	proj := ProjectLimit(block, 500000)
	if proj.RemainingMinutes != 0 {
		t.Errorf("remaining = %d, want 0", proj.RemainingMinutes)
	}
}

func TestParseTokenLimit(t *testing.T) {
	tests := []struct {
		input string
		want  uint64
	}{
		{"", 500000},
		{"100000", 100000},
		{"0", 500000},
		{"abc", 500000},
		{"999999999", 999999999},
	}
	for _, tc := range tests {
		got := parseTokenLimit(tc.input)
		if got != tc.want {
			t.Errorf("parseTokenLimit(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}

func TestAppendDistinct(t *testing.T) {
	s := appendDistinct(nil, "a")
	if len(s) != 1 || s[0] != "a" {
		t.Errorf("nil append = %v", s)
	}
	s2 := appendDistinct(s, "b")
	if len(s2) != 2 {
		t.Errorf("distinct append = %v", s2)
	}
	s3 := appendDistinct(s2, "a")
	if len(s3) != 2 {
		t.Errorf("duplicate append should not add: %v", s3)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{30 * time.Second, "30s"},
		{5 * time.Minute, "5m"},
		{90 * time.Minute, "1h30m"},
		{2*time.Hour + 5*time.Minute, "2h5m"},
		{0, "0s"},
	}
	for _, tc := range tests {
		got := formatDuration(tc.d)
		if got != tc.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tc.d, got, tc.want)
		}
	}
}

func TestBuildBlocksEmpty(t *testing.T) {
	blocks, err := BuildBlocks(nil, BlockOptions{})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if blocks != nil {
		t.Errorf("expected nil, got %v", blocks)
	}
}

func TestBuildBlocksSortsAndIdentifies(t *testing.T) {
	t2 := time.Date(2026, 6, 15, 10, 30, 0, 0, time.UTC)
	t1 := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC) // out of order
	entries := []*types.LoadedEntry{
		tEntry("s2", t2, 300, 400, 2.0, "claude-opus-4"),
		tEntry("s1", t1, 100, 200, 1.0, "claude-3-haiku"),
	}

	blocks, err := BuildBlocks(entries, BlockOptions{SessionLength: 30})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("len = %d, want 1", len(blocks))
	}
	if blocks[0].StartTime != t1 {
		t.Errorf("start = %v, want %v (should be sorted)", blocks[0].StartTime, t1)
	}
}

func TestBuildBlocksWithGap(t *testing.T) {
	t1 := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC) // 2h gap
	entries := []*types.LoadedEntry{
		tEntry("s1", t1, 100, 200, 1.0, "claude-3-haiku"),
		tEntry("s2", t2, 300, 400, 2.0, "claude-opus-4"),
	}

	blocks, err := BuildBlocks(entries, BlockOptions{SessionLength: 0.5}) // 30 min
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(blocks) != 3 {
		t.Fatalf("len = %d, want 3 (block + gap + block)", len(blocks))
	}
}

func TestBuildBlocksDefaultSessionLength(t *testing.T) {
	t1 := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 6, 15, 10, 30, 0, 0, time.UTC)
	entries := []*types.LoadedEntry{
		tEntry("s1", t1, 100, 200, 1.0, "claude-3-haiku"),
		tEntry("s2", t2, 300, 400, 2.0, "claude-opus-4"),
	}

	// SessionLength = 0 should use default (5h = 300min), so these merge.
	blocks, err := BuildBlocks(entries, BlockOptions{SessionLength: 0})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("len = %d, want 1 (merged with default 5h)", len(blocks))
	}
}

func TestStatuslineOutputNoData(t *testing.T) {
	for _, entries := range [][]*types.LoadedEntry{nil, {}} {
		s := StatuslineOutput(entries, BlockOptions{})
		if s != "aiusage: no data" {
			t.Errorf("got %q", s)
		}
	}
}

func TestStatuslineOutputActive(t *testing.T) {
	now := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	e := tEntry("s1", now, 100, 200, 1.5, "claude-3-haiku")

	s := StatuslineOutput([]*types.LoadedEntry{e}, BlockOptions{})
	if s == "aiusage: no data" || s == "aiusage: idle" {
		t.Fatalf("unexpected statusline: %s", s)
	}
}

func TestAppendDistinctPreservesOrder(t *testing.T) {
	s := appendDistinct(nil, "b")
	s = appendDistinct(s, "a")
	s = appendDistinct(s, "c")
	if len(s) != 3 || s[0] != "b" || s[1] != "a" || s[2] != "c" {
		t.Errorf("order not preserved: %v", s)
	}
}
