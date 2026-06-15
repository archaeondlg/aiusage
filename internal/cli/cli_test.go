package cli

import (
	"testing"

	"github.com/archhaeondlg/aiusage/internal/types"
)

func TestSortOrderDefault(t *testing.T) {
	o := &RunOptions{Order: ""}
	if o.SortOrder() != types.SortAsc {
		t.Errorf("default should be asc, got %s", o.SortOrder())
	}
}

func TestSortOrderAsc(t *testing.T) {
	o := &RunOptions{Order: "asc"}
	if o.SortOrder() != types.SortAsc {
		t.Errorf("got %s", o.SortOrder())
	}
}

func TestSortOrderDesc(t *testing.T) {
	o := &RunOptions{Order: "desc"}
	if o.SortOrder() != types.SortDesc {
		t.Errorf("got %s", o.SortOrder())
	}
}

func TestSortOrderCaseInsensitive(t *testing.T) {
	o := &RunOptions{Order: "DESC"}
	if o.SortOrder() != types.SortDesc {
		t.Errorf("got %s", o.SortOrder())
	}
}

func TestWeekStartDayDefault(t *testing.T) {
	o := &RunOptions{StartOfWeek: ""}
	if o.WeekStartDay() != types.WeekMonday {
		t.Errorf("default should be monday, got %d", o.WeekStartDay())
	}
}

func TestWeekStartDayMonday(t *testing.T) {
	o := &RunOptions{StartOfWeek: "monday"}
	if o.WeekStartDay() != types.WeekMonday {
		t.Errorf("got %d", o.WeekStartDay())
	}
}

func TestWeekStartDaySunday(t *testing.T) {
	o := &RunOptions{StartOfWeek: "sunday"}
	if o.WeekStartDay() != types.WeekSunday {
		t.Errorf("got %d", o.WeekStartDay())
	}
}

func TestWeekStartDayCaseInsensitive(t *testing.T) {
	o := &RunOptions{StartOfWeek: "MONDAY"}
	if o.WeekStartDay() != types.WeekMonday {
		t.Errorf("got %d", o.WeekStartDay())
	}
}

func TestNormalizeDateBound(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"2026-06-15", "20260615"},
		{"20260615", "20260615"},
		{"2026-06", "202606"},
	}
	for _, tc := range tests {
		got := normalizeDateBound(tc.input)
		if got != tc.want {
			t.Errorf("normalizeDateBound(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestValidateConfigSchemaValid(t *testing.T) {
	data := []byte(`{"defaults":{"timezone":"UTC"},"commands":{},"pricing":{"claude-opus-4":{"input":5e-6,"output":25e-6}}}`)
	// Should not panic or log.
	validateConfigSchema(data)
}

func TestValidateConfigSchemaInvalidJSON(t *testing.T) {
	data := []byte(`{invalid json}`)
	// Should not panic.
	validateConfigSchema(data)
}

func TestValidateConfigSchemaMissingPricing(t *testing.T) {
	data := []byte(`{"defaults":{},"commands":{}}`)
	// Should log warning about missing pricing but not panic.
	validateConfigSchema(data)
}

func TestTitleForAgent(t *testing.T) {
	tests := []struct {
		agent string
		want  string
	}{
		{"claude", "Claude Code Token Usage Report"},
		{"codex", "Codex Token Usage Report"},
		{"opencode", "OpenCode Token Usage Report"},
		{"amp", "amp Token Usage Report"},
		{"custom", "custom Token Usage Report"},
	}
	for _, tc := range tests {
		got := titleForAgent(tc.agent)
		if got != tc.want {
			t.Errorf("titleForAgent(%q) = %q, want %q", tc.agent, got, tc.want)
		}
	}
}

func TestReportKindFromString(t *testing.T) {
	tests := []struct {
		input string
		want  types.ReportKind
	}{
		{"daily", types.ReportDaily},
		{"weekly", types.ReportWeekly},
		{"monthly", types.ReportMonthly},
		{"session", types.ReportSession},
		{"unknown", types.ReportDaily},
		{"", types.ReportDaily},
	}
	for _, tc := range tests {
		got := reportKindFromString(tc.input)
		if got != tc.want {
			t.Errorf("reportKindFromString(%q) = %s, want %s", tc.input, got, tc.want)
		}
	}
}
