package claude

import (
	"testing"

	"github.com/archhaeondlg/aiusage/internal/types"
	"github.com/stretchr/testify/assert"
)

func boolPtr(b bool) *bool { return &b }

func entry(msgID string, input, output uint64, sidechain bool, speed *string) *types.LoadedEntry {
	return &types.LoadedEntry{
		Data: types.UsageEntry{
			Message: types.UsageMessage{
				ID:    &msgID,
				Model: strPtr("claude-opus-4-6"),
				Usage: types.TokenUsageRaw{
					InputTokens:  input,
					OutputTokens: output,
					Speed:        speed,
				},
			},
			RequestID:   strPtr("req-" + msgID),
			IsSidechain: &sidechain,
		},
	}
}

func entryWithReq(msgID, reqID string, input, output uint64) *types.LoadedEntry {
	return &types.LoadedEntry{
		Data: types.UsageEntry{
			Message: types.UsageMessage{
				ID:    &msgID,
				Model: strPtr("claude-opus-4-6"),
				Usage: types.TokenUsageRaw{
					InputTokens:  input,
					OutputTokens: output,
				},
			},
			RequestID: &reqID,
		},
	}
}

func TestShouldReplaceDedupedEntry_sidechainPreferred(t *testing.T) {
	// Prefer non-sidechain over sidechain.
	cand := &types.UsageEntry{IsSidechain: boolPtr(true)}
	exist := &types.UsageEntry{IsSidechain: boolPtr(false)}
	assert.False(t, shouldReplaceDedupedEntry(cand, exist),
		"should not replace non-sidechain with sidechain")

	// Replace sidechain with non-sidechain.
	cand2 := &types.UsageEntry{IsSidechain: boolPtr(false)}
	exist2 := &types.UsageEntry{IsSidechain: boolPtr(true)}
	assert.True(t, shouldReplaceDedupedEntry(cand2, exist2),
		"should replace sidechain with non-sidechain")
}

func TestShouldReplaceDedupedEntry_moreTokensPreferred(t *testing.T) {
	// More total tokens wins.
	cand := &types.UsageEntry{
		Message: types.UsageMessage{
			Usage: types.TokenUsageRaw{InputTokens: 100, OutputTokens: 200},
		},
	}
	exist := &types.UsageEntry{
		Message: types.UsageMessage{
			Usage: types.TokenUsageRaw{InputTokens: 50, OutputTokens: 50},
		},
	}
	assert.True(t, shouldReplaceDedupedEntry(cand, exist))
	assert.False(t, shouldReplaceDedupedEntry(exist, cand))
}

func TestShouldReplaceDedupedEntry_speedPreferred(t *testing.T) {
	// When token counts equal, prefer entry with speed data.
	base := types.TokenUsageRaw{InputTokens: 100, OutputTokens: 100}

	cand := &types.UsageEntry{
		Message: types.UsageMessage{
			Usage: types.TokenUsageRaw{
				InputTokens:  base.InputTokens,
				OutputTokens: base.OutputTokens,
				Speed:        strPtr("standard"),
			},
		},
	}
	exist := &types.UsageEntry{
		Message: types.UsageMessage{
			Usage: base,
		},
	}
	assert.True(t, shouldReplaceDedupedEntry(cand, exist),
		"should replace entry without speed data")
	assert.False(t, shouldReplaceDedupedEntry(exist, cand),
		"should not replace entry that already has speed data")
}

func TestMatchesDedupeKey(t *testing.T) {
	e := entryWithReq("msg-1", "req-1", 100, 50)

	assert.True(t, matchesDedupeKey(e, "msg-1", "req-1"))
	assert.False(t, matchesDedupeKey(e, "msg-1", "req-2"))
	assert.False(t, matchesDedupeKey(e, "msg-2", "req-1"))
}

func TestMatchesSidechainDedupeKey(t *testing.T) {
	e := entry("msg-1", 100, 50, true, nil)

	assert.True(t, matchesSidechainDedupeKey(e, "msg-1"))

	e2 := entry("msg-2", 100, 50, false, nil)
	assert.False(t, matchesSidechainDedupeKey(e2, "msg-2"),
		"non-sidechain entry should not match sidechain key")
}

func TestDedupEntries_dedupByIdentity(t *testing.T) {
	e1 := entry("msg-1", 100, 50, false, nil)
	e2 := entry("msg-1", 100, 50, false, nil) // Same identity.

	lf := &types.LoadedFile{Entries: []*types.LoadedEntry{e1, e2}}
	result := dedupEntries([]*types.LoadedFile{lf}, "")
	assert.Len(t, result, 1)
}

func TestDedupEntries_differentEntriesKept(t *testing.T) {
	e1 := entry("msg-1", 100, 50, false, nil)
	e2 := entry("msg-2", 200, 100, false, nil) // Different message ID.

	lf := &types.LoadedFile{Entries: []*types.LoadedEntry{e1, e2}}
	result := dedupEntries([]*types.LoadedFile{lf}, "")
	assert.Len(t, result, 2)
}

func TestDedupEntries_prefersMoreTokens(t *testing.T) {
	e1 := entry("msg-1", 100, 50, false, nil)   // 150 total
	e2 := entry("msg-1", 200, 100, false, nil)   // 300 total, should win

	lf := &types.LoadedFile{Entries: []*types.LoadedEntry{e1, e2}}
	result := dedupEntries([]*types.LoadedFile{lf}, "")
	assert.Len(t, result, 1)
	assert.Equal(t, uint64(200), result[0].Data.Message.Usage.InputTokens)
}

func TestDedupEntries_sidechainReplacedByNonSidechain(t *testing.T) {
	side := entry("msg-1", 100, 50, true, nil)
	normal := entry("msg-1", 80, 40, false, nil) // Fewer tokens but non-sidechain.

	lf := &types.LoadedFile{Entries: []*types.LoadedEntry{side, normal}}
	result := dedupEntries([]*types.LoadedFile{lf}, "")
	assert.Len(t, result, 1)
	// Non-sidechain wins even with fewer tokens.
	assert.False(t, *result[0].Data.IsSidechain)
}

func TestDedupEntries_projectFilter(t *testing.T) {
	e1 := entry("msg-1", 100, 50, false, nil)
	e1.Project = "project-a"
	e2 := entry("msg-2", 200, 100, false, nil)
	e2.Project = "project-b"

	lf := &types.LoadedFile{Entries: []*types.LoadedEntry{e1, e2}}
	result := dedupEntries([]*types.LoadedFile{lf}, "project-a")
	assert.Len(t, result, 1)
	assert.Equal(t, "project-a", result[0].Project)
}

func TestDedupEntries_multipleFiles(t *testing.T) {
	e1 := entry("msg-1", 100, 50, false, nil)
	e2 := entry("msg-1", 200, 100, false, nil) // Duplicate in other file.

	lf1 := &types.LoadedFile{Entries: []*types.LoadedEntry{e1}}
	lf2 := &types.LoadedFile{Entries: []*types.LoadedEntry{e2}}
	result := dedupEntries([]*types.LoadedFile{lf1, lf2}, "")
	assert.Len(t, result, 1)
	assert.Equal(t, uint64(200), result[0].Data.Message.Usage.InputTokens)
}

func TestUsageEntryTotal(t *testing.T) {
	e := &types.UsageEntry{
		Message: types.UsageMessage{
			Usage: types.TokenUsageRaw{
				InputTokens:              100,
				OutputTokens:             50,
				CacheCreationInputTokens: 20,
				CacheReadInputTokens:     10,
			},
		},
	}
	assert.Equal(t, uint64(180), usageEntryTotal(e))
}

func TestPushDedupedIdx_noDuplicate(t *testing.T) {
	m := make(map[uint64][]int)
	pushDedupedIdx(m, 42, 0)
	assert.Equal(t, []int{0}, m[42])
}

func TestPushDedupedIdx_skipsDuplicate(t *testing.T) {
	m := make(map[uint64][]int)
	m[42] = []int{0}
	pushDedupedIdx(m, 42, 0) // Same idx, should be skipped.
	assert.Equal(t, []int{0}, m[42])
}

func TestFilterLoadedEntriesByDate(t *testing.T) {
	entries := []*types.LoadedEntry{
		{Date: "2025-01-10"},
		{Date: "2025-02-15"},
		{Date: "2025-03-20"},
	}

	// No filter.
	result := FilterLoadedEntriesByDate(entries, "", "")
	assert.Len(t, result, 3)

	// Since filter.
	result = FilterLoadedEntriesByDate(entries, "20250201", "")
	assert.Len(t, result, 2)
	assert.Equal(t, "2025-02-15", result[0].Date)
	assert.Equal(t, "2025-03-20", result[1].Date)

	// Until filter.
	result = FilterLoadedEntriesByDate(entries, "", "20250201")
	assert.Len(t, result, 1)
	assert.Equal(t, "2025-01-10", result[0].Date)
}
