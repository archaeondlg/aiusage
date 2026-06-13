package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/archhaeondlg/aiusage/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseUsageEntry(t *testing.T) {
	line := []byte(`{"timestamp":"2025-01-10T10:00:00.000Z","version":"1.2.3","sessionId":"session-a","message":{"id":"msg_123","model":"claude-opus-4-6","usage":{"input_tokens":100,"output_tokens":25,"cache_creation_input_tokens":10,"cache_read_input_tokens":5}},"requestId":"req_456","costUSD":0.001}`)

	entry, err := ParseUsageEntry(line)
	require.NoError(t, err)
	assert.Equal(t, "2025-01-10T10:00:00.000Z", entry.Timestamp)
	require.NotNil(t, entry.SessionID)
	assert.Equal(t, "session-a", *entry.SessionID)
	assert.Equal(t, uint64(100), entry.Message.Usage.InputTokens)
	assert.Equal(t, uint64(25), entry.Message.Usage.OutputTokens)
}

func TestParseUsageEntrySynthetic(t *testing.T) {
	line := []byte(`{"timestamp":"2025-01-10T10:00:00.000Z","version":"1.2.3","message":{"id":"msg","model":"<synthetic>","usage":{"input_tokens":7,"output_tokens":3}}}`)

	entry, err := ParseUsageEntry(line)
	require.NoError(t, err)
	require.NotNil(t, entry.Message.Model)
	assert.Equal(t, "<synthetic>", *entry.Message.Model)
}

func TestIsValidUsageEntry(t *testing.T) {
	valid := &types.UsageEntry{
		Timestamp: "2026-01-01T00:00:00Z",
		Version:   strPtr("1.2.3"),
		Message: types.UsageMessage{
			ID:    strPtr("msg-1"),
			Model: strPtr("claude-opus-4-6"),
			Usage: types.TokenUsageRaw{InputTokens: 10},
		},
	}
	assert.True(t, IsValidUsageEntry(valid))

	// Empty session ID should be rejected.
	invalid := *valid
	invalid.SessionID = strPtr("")
	assert.False(t, IsValidUsageEntry(&invalid))

	// Non-semver version.
	invalid2 := *valid
	invalid2.Version = strPtr("not-semver")
	assert.False(t, IsValidUsageEntry(&invalid2))
}

func TestHasUnsupportedNullField(t *testing.T) {
	assert.True(t, HasUnsupportedNullField([]byte(`{"message":{"usage":{"speed":null}}}`)))
	assert.True(t, HasUnsupportedNullField([]byte(`{"message":{"model":null,"usage":{"input_tokens":0}}}`)))
	assert.True(t, HasUnsupportedNullField([]byte(`{"sessionId":null,"message":{"usage":{"input_tokens":0}}}`)))

	// Content can be null.
	assert.False(t, HasUnsupportedNullField([]byte(`{"message":{"content":null,"usage":{"input_tokens":0}}}`)))
}

func TestIsProjectPathSegment(t *testing.T) {
	assert.False(t, IsProjectPathSegment(""))
	assert.False(t, IsProjectPathSegment("."))
	assert.False(t, IsProjectPathSegment(".."))
	assert.False(t, IsProjectPathSegment("project/subproject"))
	assert.True(t, IsProjectPathSegment("project-a"))
}

func TestExtractProject(t *testing.T) {
	path := filepath.Join("/home", "me", ".claude", "projects", "my-project", "session.jsonl")
	assert.Equal(t, "my-project", ExtractProject(path))

	// Windows path.
	path = `C:\Users\me\.claude\projects\another-project\session.jsonl`
	assert.Equal(t, "another-project", ExtractProject(path))

	// No projects directory.
	assert.Equal(t, "unknown", ExtractProject("/tmp/data.jsonl"))
}

func TestExtractSessionParts(t *testing.T) {
	// Modern format: projects/project/session.jsonl
	sid, pp := ExtractSessionParts(
		filepath.Join("/home", "me", ".claude", "projects", "project-a", "session-a.jsonl"),
	)
	assert.Equal(t, "session-a", sid)
	assert.Equal(t, "project-a", pp)

	// Subagent path.
	sid, pp = ExtractSessionParts(
		filepath.Join("/home", "me", ".claude", "projects", "project-a", "session-a", "subagents", "worker.jsonl"),
	)
	assert.Equal(t, "session-a", sid)
	assert.Equal(t, "project-a", pp)
}

func TestLoadEntriesWithFixture(t *testing.T) {
	// Create a temp fixture directory.
	dir := t.TempDir()
	projectsDir := filepath.Join(dir, "projects", "test-project")
	require.NoError(t, os.MkdirAll(projectsDir, 0755))

	// Create a JSONL file.
	jsonl := []byte(
		`{"timestamp":"2025-01-10T10:00:00.000Z","version":"1.2.3","sessionId":"session-a","message":{"id":"msg_123","model":"claude-opus-4-6","usage":{"input_tokens":100,"output_tokens":25,"cache_creation_input_tokens":10,"cache_read_input_tokens":5}},"requestId":"req_456","costUSD":0.001}` + "\n" +
			`{"timestamp":"2025-01-10T10:00:01.000Z","version":"1.2.3","sessionId":"session-a","message":{"id":"msg_123","model":"claude-opus-4-6","usage":{"input_tokens":100,"output_tokens":250,"cache_creation_input_tokens":10,"cache_read_input_tokens":5,"speed":"standard"}},"requestId":"req_456","costUSD":0.01}` + "\n",
	)
	require.NoError(t, os.WriteFile(filepath.Join(projectsDir, "session-a.jsonl"), jsonl, 0644))

	// Test discover.
	files := UsageFiles([]string{dir}, "test-project")
	assert.Len(t, files, 1)

	// Test loading.
	loadedFiles := readUsageFilesParallel(files, nil, nil, 0)
	require.Len(t, loadedFiles, 1)

	// Dedup.
	entries := dedupEntries(loadedFiles, "")
	assert.Len(t, entries, 1)
	// Should keep the more complete entry (output_tokens=250).
	assert.Equal(t, uint64(250), entries[0].Data.Message.Usage.OutputTokens)
}

func TestExtractUsageLimitReset(t *testing.T) {
	line := []byte(`{"timestamp":"2025-01-10T10:00:00.000Z","isApiErrorMessage":true,"message":{"content":[{"text":"Claude AI usage limit reached|1736503200 remaining"}],"usage":{"input_tokens":0,"output_tokens":0}}}`)

	rt := extractUsageLimitReset(line)
	require.NotNil(t, rt)
	assert.Equal(t, int64(1736503200), rt.Unix())
}

func TestCacheCreationTokenCount(t *testing.T) {
	// Legacy field.
	u := types.TokenUsageRaw{CacheCreationInputTokens: 100}
	assert.Equal(t, uint64(100), u.CacheCreationTokenCount())

	// Structured ephemeral breakdown.
	u = types.TokenUsageRaw{
		CacheCreation: &types.CacheCreationRaw{
			Ephemeral5mInputTokens: 50,
			Ephemeral1hInputTokens: 30,
		},
		CacheCreationInputTokens: 100, // Should be ignored when CacheCreation is present.
	}
	assert.Equal(t, uint64(80), u.CacheCreationTokenCount())
}

func TestIsSemverPrefix(t *testing.T) {
	assert.True(t, isSemverPrefix("1.2.3"))
	assert.True(t, isSemverPrefix("1.2.3-beta")) // digits . digits . digit[anything...]
	assert.False(t, isSemverPrefix("not-semver"))
	assert.False(t, isSemverPrefix("1.2"))         // missing second dot
	assert.False(t, isSemverPrefix("v1.2.3"))      // starts with non-digit
}

func TestUsageDedupeHash(t *testing.T) {
	h1 := usageDedupeHash("msg-123", nil)
	h2 := usageDedupeHash("msg-123", strPtr("req-456"))
	h3 := usageDedupeHash("msg-456", nil)

	assert.NotEqual(t, h1, h2) // Different request IDs.
	assert.NotEqual(t, h1, h3) // Different message IDs.
}

func strPtr(s string) *string { return &s }

// Verify JSON round-trip.
func TestUsageEntryJSON(t *testing.T) {
	data := `{"sessionId":"s1","timestamp":"2026-01-01T00:00:00Z","message":{"model":"claude-opus-4-6","usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":10,"cache_read_input_tokens":5}}}`

	var entry types.UsageEntry
	require.NoError(t, json.Unmarshal([]byte(data), &entry))

	back, err := json.Marshal(entry)
	require.NoError(t, err)
	assert.Contains(t, string(back), `"sessionId":"s1"`)
}
