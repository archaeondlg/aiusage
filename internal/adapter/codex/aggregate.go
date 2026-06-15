package codex

import (
	"fmt"
	"hash/fnv"
	"sort"
	"strings"
	"sync"

	"github.com/archhaeondlg/aiusage/internal/dateutil"
	"github.com/archhaeondlg/aiusage/internal/types"
)

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Group loading
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

// LoadGroups loads Codex events and aggregates them into groups by period.
func LoadGroups(singleThread bool) (map[string]*codexGroupData, error) {
	sources, err := codexUsageSources()
	if err != nil {
		return nil, err
	}
	if len(sources) == 1 {
		return loadGroupsFromDirectory(sources[0].Dir, singleThread)
	}
	return loadGroupsFromSources(sources, singleThread), nil
}

func loadGroupsFromDirectory(sessionsDir string, singleThread bool) (map[string]*codexGroupData, error) {
	files := collectCodexUsageFiles(sessionsDir)
	if singleThread {
		return aggregateFiles(sessionsDir, files), nil
	}
	return aggregateFilesParallel(sessionsDir, files), nil
}

func loadGroupsFromSources(sources []codexUsageSource, singleThread bool) map[string]*codexGroupData {
	groups := make(map[string]*codexGroupData)
	for _, group := range collectDedupedCodexUsageFiles(sources) {
		fileGroups := aggregateFiles(group.Dir, group.Files)
		mergeGroups(groups, fileGroups)
	}
	return groups
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Aggregation
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

// codexGroupData accumulates Codex usage per time period.
type codexGroupData struct {
	InputTokens          uint64
	CachedInputTokens    uint64
	OutputTokens         uint64
	ReasoningOutputTokens uint64
	TotalTokens          uint64
	Models               map[string]*codexModelUsageData
	LastActivity         string
}

type codexModelUsageData struct {
	InputTokens          uint64
	CachedInputTokens    uint64
	OutputTokens         uint64
	ReasoningOutputTokens uint64
	TotalTokens          uint64
	IsFallback           bool
}

func aggregateFiles(sessionsDir string, files []string) map[string]*codexGroupData {
	groups := make(map[string]*codexGroupData)
	for _, file := range files {
		aggregateFile(sessionsDir, file, groups)
	}
	return groups
}

func aggregateFilesParallel(sessionsDir string, files []string) map[string]*codexGroupData {
	workers := 4
	if len(files) < workers {
		workers = len(files)
	}
	if workers <= 1 {
		return aggregateFiles(sessionsDir, files)
	}

	chunks := chunkStringSlice(files, workers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	groups := make(map[string]*codexGroupData)

	for _, chunk := range chunks {
		wg.Add(1)
		go func(chunkFiles []string) {
			defer wg.Done()
			local := make(map[string]*codexGroupData)
			for _, file := range chunkFiles {
				aggregateFile(sessionsDir, file, local)
			}
			mu.Lock()
			mergeGroups(groups, local)
			mu.Unlock()
		}(chunk)
	}
	wg.Wait()
	return groups
}

func aggregateFile(sessionsDir, file string, groups map[string]*codexGroupData) {
	visitCodexSessionFile(sessionsDir, file, func(event TokenUsageEvent) error {
		model := ""
		if event.Model != nil {
			model = *event.Model
		}
		if model == "" {
			return nil
		}

		ts, err := dateutil.ParseTimestamp(event.Timestamp)
		if err != nil {
			return nil
		}
		date := dateutil.FormatDate(ts, nil)

		// Group by date (daily is default).
		period := date

		g, ok := groups[period]
		if !ok {
			g = &codexGroupData{Models: make(map[string]*codexModelUsageData)}
			groups[period] = g
		}

		g.InputTokens += event.InputTokens
		g.CachedInputTokens += event.CachedInputTokens
		g.OutputTokens += event.OutputTokens
		g.ReasoningOutputTokens += event.ReasoningOutputTokens
		g.TotalTokens += event.TotalTokens

		if g.LastActivity == "" || event.Timestamp > g.LastActivity {
			g.LastActivity = event.Timestamp
		}

		mu, ok := g.Models[model]
		if !ok {
			mu = &codexModelUsageData{}
			g.Models[model] = mu
		}
		mu.InputTokens += event.InputTokens
		mu.CachedInputTokens += event.CachedInputTokens
		mu.OutputTokens += event.OutputTokens
		mu.ReasoningOutputTokens += event.ReasoningOutputTokens
		mu.TotalTokens += event.TotalTokens
		mu.IsFallback = mu.IsFallback || event.IsFallbackModel
		return nil
	})
}

// AggregateEvents aggregates pre-loaded events into groups.
func AggregateEvents(events []TokenUsageEvent, kind types.ReportKind, timezone string) map[string]*codexGroupData {
	groups := make(map[string]*codexGroupData)
	tz := dateutil.ParseTZ(&timezone)

	for _, event := range events {
		model := ""
		if event.Model != nil {
			model = *event.Model
		}
		if model == "" {
			continue
		}
		ts, err := dateutil.ParseTimestamp(event.Timestamp)
		if err != nil {
			continue
		}
		date := dateutil.FormatDate(ts, tz)

		var period string
		switch kind {
		case types.ReportDaily:
			period = date
		case types.ReportWeekly:
			period, _ = dateutil.WeekStart(date, types.WeekMonday)
		case types.ReportMonthly:
			if len(date) >= 7 {
				period = date[:7]
			} else {
				period = date
			}
		case types.ReportSession:
			period = event.SessionID
		}

		g, ok := groups[period]
		if !ok {
			g = &codexGroupData{Models: make(map[string]*codexModelUsageData)}
			groups[period] = g
		}
		g.InputTokens += event.InputTokens
		g.CachedInputTokens += event.CachedInputTokens
		g.OutputTokens += event.OutputTokens
		g.ReasoningOutputTokens += event.ReasoningOutputTokens
		g.TotalTokens += event.TotalTokens
		if g.LastActivity == "" || event.Timestamp > g.LastActivity {
			g.LastActivity = event.Timestamp
		}
		mu, ok := g.Models[model]
		if !ok {
			mu = &codexModelUsageData{}
			g.Models[model] = mu
		}
		mu.InputTokens += event.InputTokens
		mu.CachedInputTokens += event.CachedInputTokens
		mu.OutputTokens += event.OutputTokens
		mu.ReasoningOutputTokens += event.ReasoningOutputTokens
		mu.TotalTokens += event.TotalTokens
		mu.IsFallback = mu.IsFallback || event.IsFallbackModel
	}
	return groups
}

// FilterEventsByDate filters events by since/until date bounds.
func FilterEventsByDate(events []TokenUsageEvent, since, until string, timezone string) []TokenUsageEvent {
	if since == "" && until == "" {
		return events
	}
	tz := dateutil.ParseTZ(&timezone)
	var filtered []TokenUsageEvent
	for _, event := range events {
		ts, err := dateutil.ParseTimestamp(event.Timestamp)
		if err != nil {
			continue
		}
		date := strings.ReplaceAll(dateutil.FormatDate(ts, tz), "-", "")
		if since != "" && date < since {
			continue
		}
		if until != "" && date > until {
			continue
		}
		filtered = append(filtered, event)
	}
	return filtered
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Group merging
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

func mergeGroups(target, source map[string]*codexGroupData) {
	for period, srcGroup := range source {
		tgt, ok := target[period]
		if !ok {
			tgt = &codexGroupData{Models: make(map[string]*codexModelUsageData)}
			target[period] = tgt
		}
		tgt.InputTokens += srcGroup.InputTokens
		tgt.CachedInputTokens += srcGroup.CachedInputTokens
		tgt.OutputTokens += srcGroup.OutputTokens
		tgt.ReasoningOutputTokens += srcGroup.ReasoningOutputTokens
		tgt.TotalTokens += srcGroup.TotalTokens
		if srcGroup.LastActivity > tgt.LastActivity {
			tgt.LastActivity = srcGroup.LastActivity
		}
		for model, srcMu := range srcGroup.Models {
			tgtMu, ok := tgt.Models[model]
			if !ok {
				tgtMu = &codexModelUsageData{}
				tgt.Models[model] = tgtMu
			}
			tgtMu.InputTokens += srcMu.InputTokens
			tgtMu.CachedInputTokens += srcMu.CachedInputTokens
			tgtMu.OutputTokens += srcMu.OutputTokens
			tgtMu.ReasoningOutputTokens += srcMu.ReasoningOutputTokens
			tgtMu.TotalTokens += srcMu.TotalTokens
			tgtMu.IsFallback = tgtMu.IsFallback || srcMu.IsFallback
		}
	}
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Deduplication (parallel-safe via sharding)
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

type eventKey struct {
	SessionHash         uint64
	Timestamp           int64
	ModelHash           uint64
	InputTokens         uint64
	CachedInputTokens   uint64
	OutputTokens        uint64
	ReasoningTokens     uint64
	TotalTokens         uint64
}

type dedupeShard struct {
	mu   sync.Mutex
	seen map[eventKey]bool
}

func newDedupeShards(count int) []*dedupeShard {
	shards := make([]*dedupeShard, count)
	for i := range shards {
		shards[i] = &dedupeShard{seen: make(map[eventKey]bool)}
	}
	return shards
}

func dedupeShardInsert(shards []*dedupeShard, event TokenUsageEvent, timestamp string) bool {
	model := ""
	if event.Model != nil {
		model = *event.Model
	}
	ts, _ := dateutil.ParseTimestamp(timestamp)
	key := eventKey{
		Timestamp:         ts.UnixMilli(),
		ModelHash:         hashString(model),
		InputTokens:       event.InputTokens,
		CachedInputTokens: event.CachedInputTokens,
		OutputTokens:      event.OutputTokens,
		ReasoningTokens:   event.ReasoningOutputTokens,
		TotalTokens:       event.TotalTokens,
	}
	h := fnv.New64a()
	h.Write([]byte(fmt.Sprintf("%d/%d/%d/%d/%d/%d/%d", key.Timestamp, key.ModelHash, key.InputTokens, key.CachedInputTokens, key.OutputTokens, key.ReasoningTokens, key.TotalTokens)))
	shardIdx := h.Sum64() % uint64(len(shards))
	shard := shards[shardIdx]
	shard.mu.Lock()
	defer shard.mu.Unlock()
	if shard.seen[key] {
		return false
	}
	shard.seen[key] = true
	return true
}

func hashString(s string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64()
}

func chunkStringSlice(s []string, n int) [][]string {
	var chunks [][]string
	for i := 0; i < len(s); i += (len(s) + n - 1) / n {
		end := i + (len(s)+n-1)/n
		if end > len(s) {
			end = len(s)
		}
		chunks = append(chunks, s[i:end])
	}
	return chunks
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Sorted groups helper
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

// sortedGroupKeys returns sorted keys for deterministic output.
func sortedGroupKeys(groups map[string]*codexGroupData) []string {
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
