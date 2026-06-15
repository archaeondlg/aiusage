package output

import (
	"testing"

	"github.com/archhaeondlg/aiusage/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		input uint64
		want  string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1,000"},
		{9_999, "9,999"},
		{10_000, "10K"},
		{1_234_567, "1.23M"},
		{10_000_000, "10M"},
		{100_000_000, "100M"},
		{1_000_000_000, "1.00B"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, FormatNumber(tt.input))
	}
}

func TestFormatCurrency(t *testing.T) {
	assert.Equal(t, "$0.00", FormatCurrency(0))
	assert.Equal(t, "$0.25", FormatCurrency(0.25))
	assert.Equal(t, "$1.50", FormatCurrency(1.5))
	assert.Equal(t, "$1234.56", FormatCurrency(1234.56))
}

func TestShortModelName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"claude-sonnet-4-20250514", "claude-sonnet-4-20250514"},
		{"anthropic.claude-sonnet-4-20250514", "claude-sonnet-4-20250514"},
		{"us.anthropic.claude-opus-4-6", "claude-opus-4-6"},
		{"openrouter/anthropic/claude-sonnet-4", "claude-sonnet-4"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, ShortModelName(tt.input))
	}
}

func TestFormatModelsMultiline(t *testing.T) {
	models := []string{
		"anthropic.claude-sonnet-4-20250514",
		"gpt-5.2-codex",
		"anthropic.claude-sonnet-4-20250514",
		"unknown",
	}
	result := FormatModelsMultiline(models)
	assert.Contains(t, result, "- claude-sonnet-4-20250514")
	assert.Contains(t, result, "- gpt-5.2-codex")
	assert.Contains(t, result, "- unknown")
	// No duplicates.
	assert.Equal(t, 3, countLines(result))
}

func TestParseProjectAliases(t *testing.T) {
	result := ParseProjectAliases("aiusage=Usage Tracker,long-project=Short")
	assert.Equal(t, "Usage Tracker", result["aiusage"])
	assert.Equal(t, "Short", result["long-project"])

	result = ParseProjectAliases("")
	assert.Nil(t, result)
}

func TestFormatProjectName(t *testing.T) {
	aliases := map[string]string{"D--Project-aiusage": "AIUsage"}
	assert.Equal(t, "AIUsage", FormatProjectName("D--Project-aiusage", aliases))

	// Windows-style path.
	assert.Equal(t, "aiusage", FormatProjectName(`C:\Users\dev\Development\aiusage`, nil))
}

func TestTableBasic(t *testing.T) {
	style := Style{Enabled: false, NoColor: true}
	tbl := NewTable(
		[]string{"Date", "Cost"},
		[]Align{AlignLeft, AlignRight},
		style,
	)
	tbl.Push([]string{"2026-01-02", "$0.25"})
	tbl.Push([]string{"2026-01-03", "$1.00"})
	output := tbl.Render()
	assert.Contains(t, output, "Date")
	assert.Contains(t, output, "Cost")
	assert.Contains(t, output, "2026-01-02")
	assert.Contains(t, output, "$0.25")
}

func TestTotalsJSON(t *testing.T) {
	rows := []*types.UsageSummary{
		{
			InputTokens:   100,
			OutputTokens:  50,
			CacheCreation: 10,
			CacheRead:     5,
			TotalCost:     0.25,
		},
		{
			InputTokens:   200,
			OutputTokens:  100,
			CacheCreation: 20,
			CacheRead:     10,
			TotalCost:     0.50,
		},
	}
	totals := TotalsJSON(rows)
	assert.Equal(t, uint64(300), totals["inputTokens"])
	assert.Equal(t, uint64(150), totals["outputTokens"])
	assert.Equal(t, uint64(30), totals["cacheCreationTokens"])
	assert.Equal(t, uint64(15), totals["cacheReadTokens"])
	assert.Equal(t, uint64(495), totals["totalTokens"])
	assert.Equal(t, 0.75, totals["totalCost"]) // 0.75 is not a whole number, stays float64.
}

func TestJSONFloat(t *testing.T) {
	assert.Equal(t, 42.0, jsonFloat(42.0))
	assert.Equal(t, 0.0, jsonFloat(0.0))
	assert.Equal(t, 0.25, jsonFloat(0.25))
	assert.Equal(t, 42.123456789, jsonFloat(42.123456789))
	assert.Equal(t, 42.123456789, jsonFloat(42.1234567891)) // truncated to 9 decimals
}

func countLines(s string) int {
	n := 0
	for _, ch := range s {
		if ch == '\n' {
			n++
		}
	}
	return n + 1
}
