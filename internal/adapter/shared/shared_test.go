package shared

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/archhaeondlg/aiusage/internal/pricing"
	"github.com/archhaeondlg/aiusage/internal/types"
)

func TestTotalsFromRowsEmpty(t *testing.T) {
	result := TotalsFromRows(nil)
	if result["totalTokens"].(uint64) != 0 {
		t.Errorf("total = %d", result["totalTokens"])
	}
}

func TestTotalsFromRowsSingle(t *testing.T) {
	rows := []*types.UsageSummary{
		{InputTokens: 100, OutputTokens: 200, CacheCreation: 10, CacheRead: 5, ExtraTotal: 0, TotalCost: 1.5},
	}
	result := TotalsFromRows(rows)
	if result["inputTokens"].(uint64) != 100 {
		t.Errorf("input = %d", result["inputTokens"])
	}
	if result["totalTokens"].(uint64) != 315 {
		t.Errorf("total = %d", result["totalTokens"])
	}
}

func TestTotalsFromRowsMultiple(t *testing.T) {
	rows := []*types.UsageSummary{
		{InputTokens: 100, OutputTokens: 200, ExtraTotal: 0, TotalCost: 1.0},
		{InputTokens: 300, OutputTokens: 400, CacheCreation: 50, CacheRead: 25, ExtraTotal: 10, TotalCost: 2.0},
	}
	result := TotalsFromRows(rows)
	if result["inputTokens"].(uint64) != 400 {
		t.Errorf("input = %d", result["inputTokens"])
	}
	if result["outputTokens"].(uint64) != 600 {
		t.Errorf("output = %d", result["outputTokens"])
	}
	if result["cacheCreationTokens"].(uint64) != 50 {
		t.Errorf("cache_creation = %d", result["cacheCreationTokens"])
	}
	if result["totalTokens"].(uint64) != 1085 {
		t.Errorf("total = %d", result["totalTokens"])
	}
	if result["totalCost"].(float64) != 3.0 {
		t.Errorf("cost = %f", result["totalCost"])
	}
}

func TestReadJSONLLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")
	content := `{"a":1}
{"a":2}
{"a":3}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	var count int
	err := ReadJSONLLines(path, func(line []byte) error {
		count++
		return nil
	})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if count != 3 {
		t.Errorf("count = %d, want 3", count)
	}
}

func TestReadJSONLLinesEmptyLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")
	content := "{\"a\":1}\n\n{\"a\":2}\n\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	var count int
	err := ReadJSONLLines(path, func(line []byte) error {
		count++
		return nil
	})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2 (skip empty lines)", count)
	}
}

func TestReadJSONLLinesNoTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")
	content := `{"a":1}
{"a":2}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	var count int
	err := ReadJSONLLines(path, func(line []byte) error {
		count++
		return nil
	})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestReadJSONLLinesWindowsCRLF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")
	content := "{\"a\":1}\r\n{\"a\":2}\r\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	var lines [][]byte
	err := ReadJSONLLines(path, func(line []byte) error {
		lines = append(lines, line)
		return nil
	})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("count = %d, want 2", len(lines))
	}
	for i, l := range lines {
		if len(l) > 0 && l[len(l)-1] == '\r' {
			t.Errorf("line %d still has CR", i)
		}
	}
}

func TestReadJSONLLinesFileNotFound(t *testing.T) {
	err := ReadJSONLLines("nonexistent.jsonl", func(line []byte) error {
		return nil
	})
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestParseJSONLFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")
	content := `{"name":"alice"}
{"name":"bob"}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	type entry struct {
		Name string `json:"name"`
	}
	entries, err := ParseJSONLFile[entry](path)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len = %d, want 2", len(entries))
	}
	if entries[0].Name != "alice" || entries[1].Name != "bob" {
		t.Errorf("names = %v", entries)
	}
}

func TestParseJSONLFileSkipsBadLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")
	content := `{"name":"alice"}
not-json
{"name":"bob"}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	type entry struct {
		Name string `json:"name"`
	}
	entries, err := ParseJSONLFile[entry](path)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len = %d, want 2 (skip unparseable)", len(entries))
	}
}

func TestGenericAdapterName(t *testing.T) {
	a := NewGenericAdapter("test-agent", nil)
	if a.Name() != "test-agent" {
		t.Errorf("name = %q", a.Name())
	}
}

func TestGenericAdapterIsAvailableNoDirs(t *testing.T) {
	a := NewGenericAdapter("test", nil)
	if a.IsAvailable() {
		t.Error("should not be available with nil dirs")
	}
}

func TestFindJSONLFiles(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	os.Mkdir(sub, 0755)

	os.WriteFile(filepath.Join(dir, "a.jsonl"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(sub, "b.jsonl"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(dir, "c.txt"), []byte(""), 0644)

	files := FindJSONLFiles([]string{dir})
	if len(files) != 2 {
		t.Fatalf("found %d JSONL files, want 2: %v", len(files), files)
	}
}

func TestFindJSONLFilesEmptyPaths(t *testing.T) {
	files := FindJSONLFiles(nil)
	if len(files) != 0 {
		t.Errorf("expected empty, got %v", files)
	}
}

func TestNewGenericAdapter(t *testing.T) {
	a := NewGenericAdapter("custom", []string{"~/data"})
	if a.Name() != "custom" {
		t.Errorf("name = %q", a.Name())
	}
}

func TestParseGenericEntry(t *testing.T) {
	pm := pricing.LoadDefaultPricing()
	line := `{"timestamp":"2026-06-15T10:00:00Z","sessionId":"s1","message":{"model":"claude-sonnet-4-6","usage":{"input_tokens":100,"output_tokens":200,"cache_creation_input_tokens":10,"cache_read_input_tokens":5}}}`
	entry := ParseGenericEntry([]byte(line), pm)
	if entry == nil {
		t.Fatal("expected entry, got nil")
	}
	if entry.Timestamp.IsZero() {
		t.Error("timestamp should be set")
	}
	if entry.SessionID != "s1" {
		t.Errorf("session = %q", entry.SessionID)
	}
	if entry.Model == nil || *entry.Model != "claude-sonnet-4-6" {
		t.Errorf("model = %v", entry.Model)
	}
	if entry.Cost <= 0 {
		t.Errorf("cost should be > 0, got %f", entry.Cost)
	}
	if entry.MissingPricingModel != nil {
		t.Errorf("should have pricing: %v", entry.MissingPricingModel)
	}
}

func TestParseGenericEntryWithCostUSD(t *testing.T) {
	line := `{"timestamp":"2026-06-15T10:00:00Z","sessionId":"s2","message":{"model":"claude-sonnet-4-6","usage":{"input_tokens":100,"output_tokens":200}},"costUSD":0.05}`
	entry := ParseGenericEntry([]byte(line), nil)
	if entry == nil {
		t.Fatal("expected entry, got nil")
	}
	if entry.Cost != 0.05 {
		t.Errorf("cost = %f, want 0.05", entry.Cost)
	}
}

func TestParseGenericEntryNoUsage(t *testing.T) {
	line := `{"timestamp":"2026-06-15T10:00:00Z","message":{}}`
	entry := ParseGenericEntry([]byte(line), nil)
	if entry != nil {
		t.Error("expected nil for no usage")
	}
}

func TestParseGenericEntryZeroTokens(t *testing.T) {
	line := `{"timestamp":"2026-06-15T10:00:00Z","message":{"usage":{"input_tokens":0,"output_tokens":0}}}`
	entry := ParseGenericEntry([]byte(line), nil)
	if entry != nil {
		t.Error("expected nil for zero tokens")
	}
}

func TestParseGenericEntryShortLine(t *testing.T) {
	entry := ParseGenericEntry([]byte("short"), nil)
	if entry != nil {
		t.Error("expected nil for short line")
	}
}

func TestParseGenericEntryMissingPricing(t *testing.T) {
	pm := pricing.LoadDefaultPricing()
	line := `{"timestamp":"2026-06-15T10:00:00Z","sessionId":"s1","message":{"model":"unknown-model-xyz","usage":{"input_tokens":100,"output_tokens":200}}}`
	entry := ParseGenericEntry([]byte(line), pm)
	if entry == nil {
		t.Fatal("expected entry, got nil")
	}
	if entry.MissingPricingModel == nil {
		t.Error("expected MissingPricingModel to be set")
	}
}

func TestParseGenericEntryNoSessionID(t *testing.T) {
	line := `{"timestamp":"2026-06-15T10:00:00Z","message":{"model":"claude-sonnet-4-5","usage":{"input_tokens":100,"output_tokens":200}}}`
	entry := ParseGenericEntry([]byte(line), nil)
	if entry == nil {
		t.Fatal("expected entry, got nil")
	}
	if entry.SessionID != "unknown" {
		t.Errorf("session = %q, want unknown", entry.SessionID)
	}
}
