package claude

import (
	"encoding/json"

	"github.com/archhaeondlg/aiusage/internal/types"
)

// ParseUsageEntry unmarshals a JSONL line into a UsageEntry, skipping invalid lines.
func ParseUsageEntry(line []byte) (*types.UsageEntry, error) {
	if !bytesContains(line, []byte(`"usage":{"`)) {
		return nil, &SkipEntry{Reason: "no usage data"}
	}
	if HasUnsupportedNullField(line) {
		return nil, &SkipEntry{Reason: "unsupported null field"}
	}
	var entry types.UsageEntry
	if err := json.Unmarshal(line, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

// IsValidUsageEntry checks whether a parsed entry should be included.
func IsValidUsageEntry(entry *types.UsageEntry) bool {
	// Version must be empty or semver-prefixed (X.Y.Z...).
	if entry.Version != nil && !isSemverPrefix(*entry.Version) {
		return false
	}
	// Session ID must not be empty string.
	if entry.SessionID != nil && *entry.SessionID == "" {
		return false
	}
	// Request ID must not be empty string.
	if entry.RequestID != nil && *entry.RequestID == "" {
		return false
	}
	// Message ID must not be empty string.
	if entry.Message.ID != nil && *entry.Message.ID == "" {
		return false
	}
	// Model must not be empty string.
	if entry.Message.Model != nil && *entry.Message.Model == "" {
		return false
	}
	return true
}

func isSemverPrefix(s string) bool {
	// Must be: digits . digits . digit(s)[anything...]
	// Equivalent to Rust: consume_ascii_digits, expect '.', consume_ascii_digits, expect '.', expect at least one digit.
	idx := 0
	if !consumeDigits(s, &idx) || idx >= len(s) || s[idx] != '.' {
		return false
	}
	idx++
	if !consumeDigits(s, &idx) || idx >= len(s) || s[idx] != '.' {
		return false
	}
	idx++
	return idx < len(s) && isDigitByte(s[idx])
}

func consumeDigits(s string, idx *int) bool {
	start := *idx
	for *idx < len(s) && isDigitByte(s[*idx]) {
		*idx++
	}
	return *idx > start
}

func isDigitByte(b byte) bool {
	return b >= '0' && b <= '9'
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Null field detection (mirrors Rust has_unsupported_null_field)
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

// unsupportedNullFields lists fields that must NOT be null in CC usage JSONL.
var unsupportedNullFields = map[string]bool{
	"id":                        true,
	"cwd":                       true,
	"model":                     true,
	"speed":                     true,
	"costUSD":                   true,
	"version":                   true,
	"sessionId":                 true,
	"requestId":                 true,
	"isApiErrorMessage":         true,
	"cache_read_input_tokens":   true,
	"cache_creation_input_tokens": true,
}

// HasUnsupportedNullField scans a JSONL line for disallowed ":null" values.
func HasUnsupportedNullField(line []byte) bool {
	nullMarker := []byte(":null")
	offset := 0
	for {
		idx := bytesIndex(line[offset:], nullMarker)
		if idx < 0 {
			return false
		}
		nullIdx := offset + idx

		// Find the opening quote of the field name before ":null".
		fieldEnd := nullIdx - 1
		for fieldEnd > 0 && line[fieldEnd] != '"' {
			fieldEnd--
		}
		if line[fieldEnd] != '"' {
			offset = nullIdx + len(nullMarker)
			continue
		}
		fieldStart := fieldEnd - 1
		for fieldStart > 0 && line[fieldStart] != '"' {
			fieldStart--
		}
		if line[fieldStart] == '"' {
			field := string(line[fieldStart+1 : fieldEnd])
			if unsupportedNullFields[field] {
				return true
			}
		}
		offset = nullIdx + len(nullMarker)
	}
}

func bytesContains(data, sub []byte) bool {
	return bytesIndex(data, sub) >= 0
}

func bytesIndex(data, sub []byte) int {
	if len(sub) == 0 {
		return 0
	}
	first := sub[0]
	for i := 0; i <= len(data)-len(sub); i++ {
		if data[i] == first {
			match := true
			for j := 1; j < len(sub); j++ {
				if data[i+j] != sub[j] {
					match = false
					break
				}
			}
			if match {
				return i
			}
		}
	}
	return -1
}

// SkipEntry signals that a JSONL line should be skipped (not an error).
type SkipEntry struct {
	Reason string
}

func (e *SkipEntry) Error() string {
	return "skip entry: " + e.Reason
}
