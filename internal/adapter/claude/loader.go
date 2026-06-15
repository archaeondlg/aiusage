package claude

import (
	"bytes"
	"context"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/archhaeondlg/aiusage/internal/dateutil"
	"github.com/archhaeondlg/aiusage/internal/pricing"
	"github.com/archhaeondlg/aiusage/internal/types"
)

// LoadEntries discovers and parses all Claude Code usage files.
func LoadEntries(opts loadOptions) ([]*types.LoadedEntry, error) {
	v := opts.Verbose

	if v >= 1 {
		fmt.Fprintln(os.Stderr, "→ Discovering Claude Code data directories...")
	}
	paths, err := ClaudePaths()
	if err != nil {
		return nil, err
	}
	if v >= 2 {
		for _, p := range paths {
			fmt.Fprintf(os.Stderr, "  found: %s\n", p)
		}
	}

	if v >= 1 {
		fmt.Fprintln(os.Stderr, "→ Scanning for usage files...")
	}
	files := UsageFiles(paths, opts.ProjectFilter)
	if len(files) == 0 {
		if v >= 1 {
			fmt.Fprintln(os.Stderr, "  no usage files found")
		}
		return nil, nil
	}
	if v >= 1 {
		fmt.Fprintf(os.Stderr, "  found %d .jsonl files\n", len(files))
	}

	tz := dateutil.ParseTZ(&opts.Timezone)
	pricingMap := opts.Pricing

	if v >= 1 {
		fmt.Fprintln(os.Stderr, "→ Parsing usage files...")
	}
	startParse := time.Now()
	var loadedFiles []*types.LoadedFile
	if opts.SingleThread {
		for i, file := range files {
			if v >= 3 {
				fmt.Fprintf(os.Stderr, "  [%d/%d] %s\n", i+1, len(files), file)
			}
			lf := readUsageFile(file, tz, pricingMap)
			loadedFiles = append(loadedFiles, lf)
		}
	} else {
		if v >= 3 {
			fmt.Fprintf(os.Stderr, "  processing %d files in parallel\n", len(files))
		}
		loadedFiles = readUsageFilesParallel(files, tz, pricingMap, v)
	}

	if v >= 1 {
		totalEntries := 0
		for _, lf := range loadedFiles {
			totalEntries += len(lf.Entries)
		}
		elapsed := time.Since(startParse)
		fmt.Fprintf(os.Stderr, "  parsed %d entries from %d files in %.1fs\n", totalEntries, len(loadedFiles), elapsed.Seconds())
	}

	if v >= 1 {
		fmt.Fprintln(os.Stderr, "→ Deduplicating entries...")
	}
	result := dedupEntries(loadedFiles, opts.ProjectFilter)
	if v >= 1 {
		fmt.Fprintf(os.Stderr, "  %d unique entries after dedup\n", len(result))
	}

	return result, nil
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Parallel file loading
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

func readUsageFilesParallel(
	files []string,
	tz *time.Location,
	pricing pricing.PricingProvider,
	verbose int,
) []*types.LoadedFile {
	workers := runtime.GOMAXPROCS(0)
	if workers > len(files) {
		workers = len(files)
	}
	if workers <= 1 {
		var result []*types.LoadedFile
		for i, file := range files {
			if verbose >= 3 {
				fmt.Fprintf(os.Stderr, "  [%d/%d] %s\n", i+1, len(files), file)
			}
			result = append(result, readUsageFile(file, tz, pricing))
		}
		return result
	}

	// Sort files by size descending so big files start first.
	type fileInfo struct {
		idx  int
		path string
		size int64
	}
	sorted := make([]fileInfo, len(files))
	for i, f := range files {
		sorted[i] = fileInfo{idx: i, path: f}
		if info, err := os.Stat(f); err == nil {
			sorted[i].size = info.Size()
		}
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].size > sorted[j].size
	})

	type indexedFile struct {
		index int
		file  *types.LoadedFile
	}

	const chunkSize = 16 * 1024 * 1024 // 16MB per chunk

	// Shared atomic work queue — workers grab next file index.
	var counter atomic.Int32
	var wg sync.WaitGroup
	results := make([][]indexedFile, workers)
	var completed atomic.Int32

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(wi int) {
			defer wg.Done()
			for {
				i := int(counter.Add(1) - 1)
				if i >= len(sorted) {
					return
				}
				fi := sorted[i]
				fileStart := time.Now()

				if verbose >= 3 {
					fmt.Fprintf(os.Stderr, "  > parsing %s (%s)...\n",
						fi.path, formatSize(fi.size))
				}

				var lf *types.LoadedFile
				if fi.size > chunkSize*2 && workers > 1 {
					lf = readUsageFileChunked(fi.path, tz, pricing, chunkSize, workers)
				} else {
					lf = readUsageFile(fi.path, tz, pricing)
				}
				results[wi] = append(results[wi], indexedFile{fi.idx, lf})

				done := completed.Add(1)
				if verbose >= 3 {
					fmt.Fprintf(os.Stderr, "  [%d/%d] %s (%.1fs)\n",
						done, len(sorted), fi.path, time.Since(fileStart).Seconds())
				}
			}
		}(w)
	}
	wg.Wait()

	// Reassemble in original order.
	loadedFiles := make([]*types.LoadedFile, len(files))
	for _, chunkResults := range results {
		for _, r := range chunkResults {
			loadedFiles[r.index] = r.file
		}
	}
	return loadedFiles
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Single file reader
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

func readUsageFile(
	path string,
	tz *time.Location,
	pm pricing.PricingProvider,
) *types.LoadedFile {
	// Run with cancellation — some files cause parseLines to hang forever.
	type result struct {
		lf *types.LoadedFile
	}
	ch := make(chan result, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	go func() {
		lf := readUsageFileInner(path, tz, pm)
		select {
		case ch <- result{lf}:
		case <-ctx.Done():
		}
	}()
	select {
	case r := <-ch:
		return r.lf
	case <-ctx.Done():
		fmt.Fprintf(os.Stderr, "WARN  skipping %s: timeout\n", path)
		return &types.LoadedFile{Entries: make([]*types.LoadedEntry, 0)}
	}
}

func readUsageFileInner(
	path string,
	tz *time.Location,
	pm pricing.PricingProvider,
) *types.LoadedFile {
	project := ExtractProject(path)
	sessionID, projectPath := ExtractSessionParts(path)

	lf := &types.LoadedFile{Entries: make([]*types.LoadedEntry, 0)}

	data, err := os.ReadFile(path)
	if err != nil {
		return lf
	}

	parseLines(data, lf, tz, pm, project, sessionID, projectPath)
	return lf
}

var usageMarker = []byte(`"usage":{"`)

func parseLines(
	data []byte,
	lf *types.LoadedFile,
	tz *time.Location,
	pm pricing.PricingProvider,
	project, sessionID, projectPath string,
) {

	for len(data) > 0 {
		nl := bytes.IndexByte(data, '\n')
		var line []byte
		if nl < 0 {
			line = data
			data = nil
		} else {
			line = data[:nl]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			data = data[nl+1:]
		}

		if le := processEntry(line, pm, tz, project); le != nil {
			if lf.Timestamp == nil || le.Timestamp.Before(*lf.Timestamp) {
				lf.Timestamp = &le.Timestamp
			}
			lf.Entries = append(lf.Entries, le)
		}
	}
}

func processEntry(line []byte, pm pricing.PricingProvider, tz *time.Location, project string) *types.LoadedEntry {
	if !bytes.Contains(line, usageMarker) {
		return nil
	}
	if HasUnsupportedNullField(line) {
		return nil
	}
	entry, err := ParseUsageEntry(line)
	if err != nil || !IsValidUsageEntry(entry) {
		return nil
	}
	timestamp, err := dateutil.ParseTimestamp(entry.Timestamp)
	if err != nil {
		return nil
	}
	date := dateutil.FormatDate(timestamp, tz)
	cost := calculateEntryCost(entry, pm)
	missingModel := missingPricingModel(entry, pm)

	var model *string
	if entry.Message.Model != nil && *entry.Message.Model != "<synthetic>" {
		m := *entry.Message.Model
		if entry.Message.Usage.Speed != nil && *entry.Message.Usage.Speed == types.SpeedFast {
			m += "-fast"
		}
		model = &m
	}

	return &types.LoadedEntry{
		Data:                *entry,
		Timestamp:           timestamp,
		Date:                date,
		Project:             project,
		Cost:                cost,
		Model:               model,
		MissingPricingModel: missingModel,
	}
}

func calculateEntryCost(entry *types.UsageEntry, pm pricing.PricingProvider) float64 {
	if pm == nil {
		return 0
	}
	model := ""
	if entry.Message.Model != nil {
		model = *entry.Message.Model
	}
	return pricing.CalculateCost(
		entry.Message.Usage.InputTokens,
		entry.Message.Usage.OutputTokens,
		entry.Message.Usage.CacheCreationTokenCount(),
		entry.Message.Usage.CacheReadInputTokens,
		model,
		pm,
	)
}

func missingPricingModel(entry *types.UsageEntry, pm pricing.PricingProvider) *string {
	if pm == nil {
		return nil
	}
	model := ""
	if entry.Message.Model != nil {
		model = *entry.Message.Model
	}
	if model == "" {
		return nil
	}
	if p := pm.Find(model); p == nil {
		return &model
	}
	return nil
}

func extractUsageLimitReset(line []byte) *time.Time {
	marker := "Claude AI usage limit reached"
	idx := strings.Index(string(line), marker)
	if idx < 0 {
		return nil
	}
	rest := string(line[idx+len(marker):])
	pipeIdx := strings.IndexByte(rest, '|')
	if pipeIdx < 0 {
		return nil
	}
	timestampStr := rest[pipeIdx+1:]
	end := 0
	for end < len(timestampStr) && timestampStr[end] >= '0' && timestampStr[end] <= '9' {
		end++
	}
	if end == 0 {
		return nil
	}
	seconds, err := parseInt64(timestampStr[:end])
	if err != nil || seconds <= 0 {
		return nil
	}
	t := time.Unix(seconds, 0).UTC()
	return &t
}

func parseInt64(s string) (int64, error) {
	var n int64
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("invalid digit: %c", ch)
		}
		n = n*10 + int64(ch-'0')
	}
	return n, nil
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Chunked file reader — parallelize a single large file by byte ranges.
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

type chunkRange struct {
	start int64
	end   int64
}

type chunkResult struct {
	entries []*types.LoadedEntry
	ts      *time.Time
}

func readUsageFileChunked(
	path string,
	tz *time.Location,
	pm pricing.PricingProvider,
	chunkSize int64,
	workers int,
) *types.LoadedFile {
	f, err := os.Open(path)
	if err != nil {
		return &types.LoadedFile{Entries: nil}
	}
	fi, err := f.Stat()
	f.Close()
	if err != nil || fi.Size() <= chunkSize {
		return readUsageFile(path, tz, pm)
	}

	fileSize := fi.Size()
	numChunks := int(fileSize / chunkSize)
	if numChunks > workers {
		numChunks = workers
	}
	if numChunks < 2 {
		return readUsageFile(path, tz, pm)
	}

	chunkLen := fileSize / int64(numChunks)
	chunks := make([]chunkRange, numChunks)
	for c := 0; c < numChunks; c++ {
		chunks[c].start = int64(c) * chunkLen
		if c == numChunks-1 {
			chunks[c].end = fileSize
		} else {
			chunks[c].end = int64(c+1) * chunkLen
		}
	}

	chunkResults := make([]chunkResult, numChunks)
	var wg sync.WaitGroup
	for ci := 0; ci < numChunks; ci++ {
		wg.Add(1)
		go func(ci int, cr chunkRange) {
			defer wg.Done()
			chunkResults[ci] = readChunk(path, cr, tz, pm)
		}(ci, chunks[ci])
	}
	wg.Wait()

	lf := &types.LoadedFile{Entries: make([]*types.LoadedEntry, 0)}
	for _, cr := range chunkResults {
		lf.Entries = append(lf.Entries, cr.entries...)
		if cr.ts != nil && (lf.Timestamp == nil || cr.ts.Before(*lf.Timestamp)) {
			lf.Timestamp = cr.ts
		}
	}
	return lf
}

func readChunk(
	path string,
	cr chunkRange,
	tz *time.Location,
	pm pricing.PricingProvider,
) chunkResult {
	result := chunkResult{}

	data, err := readChunkTimeout(path, cr, 10*time.Second)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARN  skipping chunk of %s: %v\n", path, err)
		return result
	}

	// Parse lines and collect into result.
	project := ExtractProject(path)

	for len(data) > 0 {
		nl := bytes.IndexByte(data, '\n')
		var line []byte
		if nl < 0 {
			line = data
			data = nil
		} else {
			line = data[:nl]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			data = data[nl+1:]
		}

		if le := processEntry(line, pm, tz, project); le != nil {
			if result.ts == nil || le.Timestamp.Before(*result.ts) {
				result.ts = &le.Timestamp
			}
			result.entries = append(result.entries, le)
		}
	}

	return result
}

func readChunkTimeout(path string, cr chunkRange, timeout time.Duration) ([]byte, error) {
	type readResult struct {
		data []byte
		err  error
	}
	ch := make(chan readResult, 1)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	go func() {
		f, err := os.Open(path)
		if err != nil {
			ch <- readResult{nil, err}
			return
		}
		defer f.Close()

		if cr.start > 0 {
			select {
			case <-ctx.Done():
				ch <- readResult{nil, ctx.Err()}
				return
			default:
			}
			f.Seek(cr.start, 0)
			buf := make([]byte, 1)
			for {
				select {
				case <-ctx.Done():
					ch <- readResult{nil, ctx.Err()}
					return
				default:
				}
				n, err := f.Read(buf)
				if n == 0 || err != nil {
					break
				}
				cr.start++
				if buf[0] == '\n' {
					break
				}
			}
		}
		select {
		case <-ctx.Done():
			ch <- readResult{nil, ctx.Err()}
			return
		default:
		}
		data, err := io.ReadAll(io.LimitReader(f, cr.end-cr.start))
		ch <- readResult{data, err}
	}()
	select {
	case r := <-ch:
		return r.data, r.err
	case <-ctx.Done():
		return nil, fmt.Errorf("timeout after %v", timeout)
	}
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Deduplication
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

func dedupEntries(files []*types.LoadedFile, projectFilter string) []*types.LoadedEntry {
	dedupedIdxs := make(map[uint64][]int)
	var deduped []*types.LoadedEntry

	for _, lf := range files {
		for _, entry := range lf.Entries {
			if projectFilter != "" && entry.Project != projectFilter {
				continue
			}
			pushDedupedEntry(entry, dedupedIdxs, &deduped)
		}
	}
	return deduped
}

func pushDedupedEntry(
	entry *types.LoadedEntry,
	dedupedIdxs map[uint64][]int,
	deduped *[]*types.LoadedEntry,
) {
	data := &entry.Data
	messageID := ""
	if data.Message.ID != nil {
		messageID = *data.Message.ID
	}
	requestID := ""
	if data.RequestID != nil {
		requestID = *data.RequestID
	}

	var dedupeLookup *struct {
		hash        uint64
		existingIdx int
	}
	if messageID != "" {
		exactHash := usageDedupeHash(messageID, &requestID)
		existingIdx := findDedupeMatch(*deduped, dedupedIdxs, exactHash, messageID, requestID)
		if existingIdx < 0 && (data.IsSidechain != nil && *data.IsSidechain) {
			messageHash := usageDedupeHash(messageID, nil)
			for _, idx := range dedupedIdxs[messageHash] {
				if idx < len(*deduped) && matchesSidechainDedupeKey((*deduped)[idx], messageID) {
					existingIdx = idx
					break
				}
			}
		}
		dedupeLookup = &struct {
			hash        uint64
			existingIdx int
		}{exactHash, existingIdx}
	}

	if dedupeLookup != nil && dedupeLookup.existingIdx >= 0 {
		existing := (*deduped)[dedupeLookup.existingIdx]
		if shouldReplaceDedupedEntry(&entry.Data, &existing.Data) {
			(*deduped)[dedupeLookup.existingIdx] = entry
			pushDedupedIdx(dedupedIdxs, dedupeLookup.hash, dedupeLookup.existingIdx)
			if data.Message.ID != nil {
				pushDedupedIdx(dedupedIdxs, usageDedupeHash(*data.Message.ID, nil), dedupeLookup.existingIdx)
			}
		}
		return
	}

	idx := len(*deduped)
	*deduped = append(*deduped, entry)
	if dedupeLookup != nil {
		pushDedupedIdx(dedupedIdxs, dedupeLookup.hash, idx)
		if data.Message.ID != nil {
			pushDedupedIdx(dedupedIdxs, usageDedupeHash(*data.Message.ID, nil), idx)
		}
	}
}

func findDedupeMatch(
	deduped []*types.LoadedEntry,
	dedupedIdxs map[uint64][]int,
	hash uint64,
	messageID, requestID string,
) int {
	for _, idx := range dedupedIdxs[hash] {
		if idx < len(deduped) && matchesDedupeKey(deduped[idx], messageID, requestID) {
			return idx
		}
	}
	return -1
}

func matchesDedupeKey(entry *types.LoadedEntry, messageID, requestID string) bool {
	eMsgID := ""
	if entry.Data.Message.ID != nil {
		eMsgID = *entry.Data.Message.ID
	}
	eReqID := ""
	if entry.Data.RequestID != nil {
		eReqID = *entry.Data.RequestID
	}
	return eMsgID == messageID && eReqID == requestID
}

func matchesSidechainDedupeKey(entry *types.LoadedEntry, messageID string) bool {
	eMsgID := ""
	if entry.Data.Message.ID != nil {
		eMsgID = *entry.Data.Message.ID
	}
	return eMsgID == messageID &&
		(entry.Data.IsSidechain != nil && *entry.Data.IsSidechain)
}

func shouldReplaceDedupedEntry(candidate, existing *types.UsageEntry) bool {
	candSidechain := candidate.IsSidechain != nil && *candidate.IsSidechain
	existSidechain := existing.IsSidechain != nil && *existing.IsSidechain
	if candSidechain != existSidechain {
		return existSidechain
	}
	candTotal := usageEntryTotal(candidate)
	existTotal := usageEntryTotal(existing)
	if candTotal != existTotal {
		return candTotal > existTotal
	}
	candHasSpeed := candidate.Message.Usage.Speed != nil
	existHasSpeed := existing.Message.Usage.Speed != nil
	return candHasSpeed && !existHasSpeed
}

func usageEntryTotal(entry *types.UsageEntry) uint64 {
	u := entry.Message.Usage
	return u.InputTokens + u.OutputTokens + u.CacheCreationTokenCount() + u.CacheReadInputTokens
}

func usageDedupeHash(messageID string, requestID *string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(messageID))
	if requestID != nil {
		h.Write([]byte(*requestID))
	}
	return h.Sum64()
}

func pushDedupedIdx(dedupedIdxs map[uint64][]int, hash uint64, idx int) {
	for _, existing := range dedupedIdxs[hash] {
		if existing == idx {
			return
		}
	}
	dedupedIdxs[hash] = append(dedupedIdxs[hash], idx)
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Date filtering
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

func FilterLoadedEntriesByDate(entries []*types.LoadedEntry, since, until string) []*types.LoadedEntry {
	if since == "" && until == "" {
		return entries
	}
	var filtered []*types.LoadedEntry
	for _, entry := range entries {
		date := strings.ReplaceAll(entry.Date, "-", "")
		if since != "" && date < since {
			continue
		}
		if until != "" && date > until {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Helpers
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

func formatSize(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%dB", n)
	}
	if n < 1024*1024 {
		return fmt.Sprintf("%.1fKB", float64(n)/1024)
	}
	return fmt.Sprintf("%.1fMB", float64(n)/(1024*1024))
}


