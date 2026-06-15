package codex

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/archhaeondlg/aiusage/internal/dateutil"
)

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Session file parsing
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

// isSubagentSession checks if a file contains thread_spawn markers.
func isSubagentSession(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	buf := make([]byte, 16*1024)
	n, _ := f.Read(buf)
	return strings.Contains(string(buf[:n]), "thread_spawn")
}

// detectSubagentReplaySecond detects if a subagent file has replayed parent
// history by finding two token_count events with the same second timestamp.
func detectSubagentReplaySecond(path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var firstSecond []byte
	for len(data) > 0 {
		nl := -1
		for i, b := range data { if b == '\n' { nl = i; break } }
		var line []byte
		if nl < 0 { line = data; data = nil } else { line = data[:nl]; data = data[nl+1:] }
		if len(line) > 0 && line[len(line)-1] == '\r' { line = line[:len(line)-1] }
		kind := classifyLine(line)
		if kind != kindSession {
			continue
		}

		var entry codexParsedLine
		if json.Unmarshal(line, &entry) != nil {
			continue
		}
		t := entry.Type
		if t == nil || *t != "event_msg" {
			continue
		}
		if entry.Payload == nil {
			continue
		}
		var pl codexPayload
		if json.Unmarshal(entry.Payload, &pl) != nil {
			continue
		}
		if pl.Type == nil || *pl.Type != "token_count" {
			continue
		}
		if pl.Info == nil || (pl.Info.LastTokenUsage == nil && pl.Info.TotalTokenUsage == nil) {
			continue
		}

		ts := parseTimestamp(entry.Timestamp)
		if len(ts) < 19 {
			continue
		}
		tsSecond := []byte(ts[:19])

		if firstSecond == nil {
			firstSecond = tsSecond
		} else {
			if string(firstSecond) == string(tsSecond) {
				return firstSecond
			}
			return nil
		}
	}
	return nil
}

// lineKind classifies a JSONL line.
type lineKind int

const (
	kindUnknown  lineKind = iota
	kindSession
	kindHeadless
)

func classifyLine(line []byte) lineKind {
	hasEventMsg := bytesContains(line, []byte(`"type":"event_msg"`))
	hasTokenCount := hasEventMsg && bytesContains(line, []byte(`"type":"token_count"`))
	hasTurnCtx := bytesContains(line, []byte(`"type":"turn_context"`))

	if hasTurnCtx || hasTokenCount {
		return kindSession
	}

	// Check for compact type field with nested token_count.
	hasTypeField := bytesContains(line, []byte(`"type"`))
	if !hasEventMsg && hasTypeField && len(line) < 64*1024 {
		if bytesContains(line, []byte(`"token_count"`)) {
			return kindSession
		}
	}

	if hasEventMsg || !hasTypeField {
		// Double-check with bytes-level scan.
		if bytesContains(line, []byte(`"turn_context"`)) {
			return kindSession
		}
		if bytesContains(line, []byte(`"event_msg"`)) && bytesContains(line, []byte(`"token_count"`)) {
			return kindSession
		}
	}

	if bytesContains(line, []byte(`"usage"`)) || bytesContains(line, []byte(`"input_tokens"`)) ||
		bytesContains(line, []byte(`"prompt_tokens"`)) {
		return kindHeadless
	}
	return kindUnknown
}

// visitCodexSessionFile is the Go equivalent of Rust's visit_codex_session_file.
func visitCodexSessionFile(
	sessionsDir string,
	path string,
	visit func(TokenUsageEvent) error,
) error {
	isSubagent := isSubagentSession(path)
	var replaySecond []byte
	if isSubagent {
		replaySecond = detectSubagentReplaySecond(path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	sessionID := codexSessionID(sessionsDir, path)
	var previousTotals *codexRawUsage
	var currentModel string
	var currentModelIsFallback bool
	fallbackTS := fileModifiedTimestamp(path)
	skipReplay := replaySecond != nil

	for len(data) > 0 {
		nl := -1
		for i, b := range data { if b == '\n' { nl = i; break } }
		var line []byte
		if nl < 0 { line = data; data = nil } else { line = data[:nl]; data = data[nl+1:] }
		if len(line) > 0 && line[len(line)-1] == '\r' { line = line[:len(line)-1] }
		kind := classifyLine(line)
		if kind == kindUnknown {
			continue
		}

		switch kind {
		case kindSession:
			var entry codexParsedLine
			if json.Unmarshal(line, &entry) != nil {
				continue
			}

			// Handle subagent replay skipping.
			if replaySecond != nil && skipReplay {
				t := entry.Type
				if t != nil && *t == "event_msg" {
					ts := parseTimestamp(entry.Timestamp)
					if len(ts) >= 19 && ts[:19] == string(replaySecond) {
						// Track the cumulative total from this replayed entry.
						if entry.Payload != nil {
							var pl codexPayload
							if json.Unmarshal(entry.Payload, &pl) == nil && pl.Info != nil && pl.Info.TotalTokenUsage != nil {
								prev := previousTotals
								_ = prev
								previousTotals = pl.Info.TotalTokenUsage
							}
						}
						continue
					}
					skipReplay = false
				}
			}

			visitSessionEntry(sessionID, &entry, &previousTotals, &currentModel, &currentModelIsFallback, visit)

		case kindHeadless:
			var entry codexParsedLine
			if json.Unmarshal(line, &entry) == nil {
				addHeadlessEvent(sessionID, &entry, fallbackTS, &currentModel, &currentModelIsFallback, visit)
			} else {
				// Try as raw JSON value.
				var raw map[string]json.RawMessage
				if json.Unmarshal(line, &raw) == nil {
					addHeadlessEventFromValue(sessionID, raw, fallbackTS, &currentModel, &currentModelIsFallback, visit)
				}
			}
		}
	}
	return nil
}

func visitSessionEntry(
	sessionID string,
	entry *codexParsedLine,
	previousTotals **codexRawUsage,
	currentModel *string,
	currentModelIsFallback *bool,
	visit func(TokenUsageEvent) error,
) {
	t := entry.Type
	if t == nil {
		return
	}

	if *t == "turn_context" {
		model := extractModelFromPayload(entry)
		if model != "" {
			*currentModel = model
			*currentModelIsFallback = false
		}
		return
	}
	if *t != "event_msg" {
		return
	}

	ts := parseTimestamp(entry.Timestamp)
	if ts == "" {
		return
	}

	if entry.Payload == nil {
		return
	}
	var pl codexPayload
	if json.Unmarshal(entry.Payload, &pl) != nil {
		return
	}
	if pl.Info == nil {
		return
	}
	info := pl.Info

	totalUsage := info.TotalTokenUsage
	var rawUsage *codexRawUsage
	if info.LastTokenUsage != nil {
		rawUsage = info.LastTokenUsage
	} else if totalUsage != nil {
		rawUsage = subtractUsage(totalUsage, *previousTotals)
	}
	if rawUsage == nil || rawUsage.isZero() {
		if totalUsage != nil {
			*previousTotals = totalUsage
		}
		return
	}
	if totalUsage != nil {
		*previousTotals = totalUsage
	}

	parsedModel := extractModelFromPayloadFields(
		strOrEmpty(pl.Model),
		strOrEmpty(pl.ModelName),
		extractModelFromMetadata(pl.Metadata),
	)
	if parsedModel == "" {
		parsedModel = extractModelFromInfo(info)
	}
	model, isFallback := resolveCodexUsageModel(parsedModel, ts, currentModel, currentModelIsFallback)

	cached := rawUsage.CachedInputTokens
	if cached > rawUsage.InputTokens {
		cached = rawUsage.InputTokens
	}
	visit(TokenUsageEvent{
		SessionID:           sessionID,
		Timestamp:           ts,
		Model:               &model,
		InputTokens:         rawUsage.InputTokens,
		CachedInputTokens:   cached,
		OutputTokens:        rawUsage.OutputTokens,
		ReasoningOutputTokens: rawUsage.ReasoningTokens,
		TotalTokens:         rawUsage.TotalTokens,
		IsFallbackModel:     isFallback,
	})
}

func addHeadlessEvent(
	sessionID string,
	entry *codexParsedLine,
	fallbackTS string,
	currentModel *string,
	currentModelIsFallback *bool,
	visit func(TokenUsageEvent) error,
) {
	rawUsage := normalizeHeadlessUsage(entry)
	if rawUsage == nil || rawUsage.isZero() {
		return
	}

	parsedModel := extractModelFromHeadless(entry)
	eventTS := extractTimestampFromHeadless(entry)
	if eventTS == "" {
		eventTS = fallbackTS
	}
	modelTS := extractModelTimestampFromHeadless(entry)
	if modelTS == "" {
		modelTS = fallbackTS
	}

	model, isFallback := resolveCodexUsageModel(parsedModel, modelTS, currentModel, currentModelIsFallback)

	cached := rawUsage.CachedInputTokens
	if cached > rawUsage.InputTokens {
		cached = rawUsage.InputTokens
	}
	visit(TokenUsageEvent{
		SessionID:           sessionID,
		Timestamp:           eventTS,
		Model:               &model,
		InputTokens:         rawUsage.InputTokens,
		CachedInputTokens:   cached,
		OutputTokens:        rawUsage.OutputTokens,
		ReasoningOutputTokens: rawUsage.ReasoningTokens,
		TotalTokens:         rawUsage.TotalTokens,
		IsFallbackModel:     isFallback,
	})
}

func addHeadlessEventFromValue(sessionID string, raw map[string]json.RawMessage, fallbackTS string, currentModel *string, currentModelIsFallback *bool, visit func(TokenUsageEvent) error) {
	data, _ := json.Marshal(raw)
	var entry codexParsedLine
	if json.Unmarshal(data, &entry) != nil {
		return
	}
	addHeadlessEvent(sessionID, &entry, fallbackTS, currentModel, currentModelIsFallback, visit)
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Model resolution
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

const codexAutoReview = "codex-auto-review"

func resolveCodexUsageModel(parsedModel string, timestamp string, currentModel *string, currentModelIsFallback *bool) (string, bool) {
	if parsedModel != "" {
		*currentModel = parsedModel
		*currentModelIsFallback = false
	}

	isFallback := false
	model := parsedModel
	if model == "" {
		model = *currentModel
	}
	if model == "" {
		model = "gpt-5"
		isFallback = true
		*currentModel = model
		*currentModelIsFallback = true
	}

	// Check if we should use a fallback due to auto-review model.
	if fallback := resolveAutoReviewModel(model, timestamp); fallback != "" {
		isFallback = true
		model = fallback
	}
	return model, isFallback
}

func resolveAutoReviewModel(model, timestamp string) string {
	if model != codexAutoReview {
		return ""
	}
	// Extract YYYY-MM-DD from timestamp.
	if len(timestamp) < 10 {
		return "gpt-5"
	}
	date := timestamp[:10]
	for _, fb := range autoReviewFallbacks {
		if date >= fb.ReleasedOn {
			return fb.Model
		}
	}
	return "gpt-5"
}

func extractModelFromPayload(entry *codexParsedLine) string {
	if entry == nil || entry.Payload == nil {
		return ""
	}
	var pl codexPayload
	if json.Unmarshal(entry.Payload, &pl) != nil {
		return ""
	}
	return extractModelFromPayloadFields(
		strOrEmpty(pl.Model),
		strOrEmpty(pl.ModelName),
		extractModelFromMetadata(pl.Metadata),
	)
}

func extractModelFromPayloadFields(model, modelName, metadataModel string) string {
	return nonEmpty(model, modelName, metadataModel)
}

func extractModelFromInfo(info *codexInfo) string {
	if info == nil {
		return ""
	}
	return nonEmpty(
		strOrEmpty(info.Model),
		strOrEmpty(info.ModelName),
		extractModelFromMetadata(info.Metadata),
	)
}

func extractModelFromHeadless(entry *codexParsedLine) string {
	return nonEmpty(
		strOrEmpty(entry.Model),
		strOrEmpty(entry.ModelName),
		extractModelFromMetadata(entry.Metadata),
		extractModelFromNestedFields(entry.Data),
		extractModelFromNestedFields(entry.Result),
		extractModelFromNestedFields(entry.Response),
	)
}

func extractModelFromNestedFields(raw json.RawMessage) string {
	if raw == nil {
		return ""
	}
	var rf resultFields
	if json.Unmarshal(raw, &rf) != nil {
		return ""
	}
	return nonEmpty(
		strOrEmpty(rf.Model),
		strOrEmpty(rf.ModelName),
		extractModelFromMetadata(rf.Metadata),
	)
}

func extractModelFromMetadata(raw json.RawMessage) string {
	if raw == nil {
		return ""
	}
	var m metadataModel
	if json.Unmarshal(raw, &m) != nil {
		return ""
	}
	return strOrEmpty(m.Model)
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Usage normalization
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

func normalizeHeadlessUsage(entry *codexParsedLine) *codexRawUsage {
	u := entry.Usage
	if u == nil {
		// Try nested: data.usage, result.usage, response.usage.
		for _, raw := range []json.RawMessage{entry.Data, entry.Result, entry.Response} {
			if raw == nil {
				continue
			}
			var rf resultFields
			if json.Unmarshal(raw, &rf) == nil && rf.Usage != nil {
				rf.Usage.normalize()
				return rf.Usage
			}
		}
		return nil
	}
	u.normalize()
	return u
}

func subtractUsage(current, previous *codexRawUsage) *codexRawUsage {
	if previous == nil {
		return current
	}
	return &codexRawUsage{
		InputTokens:    satSub(current.InputTokens, previous.InputTokens),
		CachedInputTokens: satSub(current.CachedInputTokens, previous.CachedInputTokens),
		OutputTokens:   satSub(current.OutputTokens, previous.OutputTokens),
		ReasoningTokens: satSub(current.ReasoningTokens, previous.ReasoningTokens),
		TotalTokens:    satSub(current.TotalTokens, previous.TotalTokens),
	}
}

func satSub(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return 0
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Timestamp parsing
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

func parseTimestamp(raw json.RawMessage) string {
	if raw == nil {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		s = strings.TrimSpace(s)
		if s != "" && isValidDatePrefix(s) {
			return s
		}
		// Try to normalize via parsing.
		if ts, err := dateutil.ParseTimestamp(s); err == nil {
			return dateutil.FormatRFC3339Millis(ts)
		}
		return ""
	}
	var n uint64
	if json.Unmarshal(raw, &n) == nil && n > 0 {
		if n > 10_000_000_000 {
			return formatRFC3339FromMillis(int64(n))
		}
		return formatRFC3339FromMillis(int64(n) * 1000)
	}
	return ""
}

func isValidDatePrefix(s string) bool {
	return len(s) >= 10 && isAllDigits(s[:4]) && s[4] == '-' &&
		isAllDigits(s[5:7]) && s[7] == '-' && isAllDigits(s[8:10])
}

func isAllDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func formatRFC3339FromMillis(ms int64) string {
	return dateutil.FormatRFC3339Millis(timeFromMillis(ms))
}

// fileModifiedTimestamp returns the file's mtime as an RFC 3339 string.
func fileModifiedTimestamp(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return "1970-01-01T00:00:00.000Z"
	}
	return dateutil.FormatRFC3339Millis(info.ModTime())
}

func codexSessionID(sessionsDir, path string) string {
	rel, err := filepath.Rel(sessionsDir, path)
	if err != nil {
		rel = path
	}
	rel = strings.TrimSuffix(rel, filepath.Ext(rel))
	return filepath.ToSlash(rel)
}

// extractTimestampFromHeadless gets the event timestamp from headless entries.
func extractTimestampFromHeadless(entry *codexParsedLine) string {
	if ts := parseTimestamp(entry.Timestamp); ts != "" {
		return ts
	}
	if ts := parseTimestamp(entry.CreatedAt); ts != "" {
		return ts
	}
	if ts := parseTimestamp(entry.CreatedAtC); ts != "" {
		return ts
	}
	for _, raw := range []json.RawMessage{entry.Data, entry.Result, entry.Response} {
		if raw == nil {
			continue
		}
		var rf resultFields
		if json.Unmarshal(raw, &rf) == nil {
			if ts := parseTimestamp(rf.Timestamp); ts != "" {
				return ts
			}
			if ts := parseTimestamp(rf.CreatedAt); ts != "" {
				return ts
			}
			if ts := parseTimestamp(rf.CreatedAtC); ts != "" {
				return ts
			}
		}
	}
	return ""
}

// extractModelTimestampFromHeadless gets the model resolution timestamp.
func extractModelTimestampFromHeadless(entry *codexParsedLine) string {
	// For model resolution, prefer the raw timestamp (not normalized).
	if s := rawTimestamp(entry.Timestamp); s != "" {
		return s
	}
	if s := rawTimestamp(entry.CreatedAt); s != "" {
		return s
	}
	if s := rawTimestamp(entry.CreatedAtC); s != "" {
		return s
	}
	for _, raw := range []json.RawMessage{entry.Data, entry.Result, entry.Response} {
		if raw == nil {
			continue
		}
		var rf resultFields
		if json.Unmarshal(raw, &rf) == nil {
			if s := rawTimestamp(rf.Timestamp); s != "" {
				return s
			}
			if s := rawTimestamp(rf.CreatedAt); s != "" {
				return s
			}
			if s := rawTimestamp(rf.CreatedAtC); s != "" {
				return s
			}
		}
	}
	return ""
}

func rawTimestamp(raw json.RawMessage) string {
	if raw == nil {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		s = strings.TrimSpace(s)
		if s != "" && isValidDatePrefix(s) {
			return s
		}
		// Try to normalize.
		if ts, err := dateutil.ParseTimestamp(s); err == nil {
			return dateutil.FormatRFC3339Millis(ts)
		}
		return ""
	}
	// For model resolution, try numeric too.
	var n uint64
	if json.Unmarshal(raw, &n) == nil && n > 0 {
		if n > 10_000_000_000 {
			return dateutil.FormatRFC3339Millis(timeFromMillis(int64(n)))
		}
		return dateutil.FormatRFC3339Millis(timeFromMillis(int64(n) * 1000))
	}
	return ""
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Helpers
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

func nonEmpty(values ...string) string {
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return ""
}

func strOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}

func timeFromMillis(ms int64) time.Time {
	return time.Unix(ms/1000, (ms%1000)*1_000_000)
}

func bytesContains(data, sub []byte) bool {
	if len(sub) == 0 {
		return true
	}
	for i := 0; i <= len(data)-len(sub); i++ {
		match := true
		for j := 0; j < len(sub); j++ {
			if data[i+j] != sub[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
