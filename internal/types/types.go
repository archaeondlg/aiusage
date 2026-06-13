// Package types provides the core data structures shared across the aiusage application.
package types

import "time"

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Raw JSONL entry types (deserialized directly from agent log files)
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

// UsageEntry represents a single raw JSONL line from a usage log file.
type UsageEntry struct {
	SessionID       *string      `json:"sessionId"`
	Timestamp       string       `json:"timestamp"`
	Version         *string      `json:"version"`
	Message         UsageMessage `json:"message"`
	CostUSD         *float64     `json:"costUSD"`
	RequestID       *string      `json:"requestId"`
	IsAPIErrorMsg   *bool        `json:"isApiErrorMessage"`
	IsSidechain     *bool        `json:"isSidechain"`
}

// UsageMessage wraps token usage and model info.
type UsageMessage struct {
	Usage TokenUsageRaw `json:"usage"`
	Model *string       `json:"model"`
	ID    *string       `json:"id"`
}

// TokenUsageRaw holds raw token counts as read from JSONL.
type TokenUsageRaw struct {
	InputTokens              uint64            `json:"input_tokens"`
	OutputTokens             uint64            `json:"output_tokens"`
	CacheCreationInputTokens uint64            `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     uint64            `json:"cache_read_input_tokens"`
	Speed                    *string           `json:"speed"`          // "standard" or "fast"
	CacheCreation            *CacheCreationRaw `json:"cache_creation"` // ephemeral breakdown
}

// CacheCreationRaw breaks cache creation into ephemeral TTL buckets.
type CacheCreationRaw struct {
	Ephemeral5mInputTokens uint64 `json:"ephemeral_5m_input_tokens"`
	Ephemeral1hInputTokens uint64 `json:"ephemeral_1h_input_tokens"`
}

// CacheCreationTokenCount returns the total cache creation tokens,
// preferring the structured ephemeral breakdown when available.
func (u TokenUsageRaw) CacheCreationTokenCount() uint64 {
	if u.CacheCreation != nil {
		return u.CacheCreation.Ephemeral5mInputTokens + u.CacheCreation.Ephemeral1hInputTokens
	}
	return u.CacheCreationInputTokens
}

// Speed enum values.
const (
	SpeedStandard = "standard"
	SpeedFast     = "fast"
)

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Enriched entry (parsed + metadata attached after loading)
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

// LoadedEntry is a fully enriched usage entry after parsing.
type LoadedEntry struct {
	Data                UsageEntry
	Timestamp           time.Time
	Date                string
	Project             string
	SessionID           string
	ProjectPath         string
	Cost                float64
	ExtraTotalTokens    uint64
	Credits             *float64
	MessageCount        *uint64
	Model               *string
	UsageLimitResetTime *time.Time
	MissingPricingModel *string
}

// LoadedFile holds all entries from a single JSONL file.
type LoadedFile struct {
	Timestamp *time.Time
	Entries   []*LoadedEntry
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Aggregated token counts
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

// TokenCounts holds aggregated token counts across multiple entries.
type TokenCounts struct {
	InputTokens       uint64 `json:"inputTokens"`
	OutputTokens      uint64 `json:"outputTokens"`
	CacheCreation     uint64 `json:"cacheCreationTokens"`
	CacheRead         uint64 `json:"cacheReadTokens"`
	ExtraTotalTokens  uint64 `json:"-"`
}

// AddUsage merges usage from a raw token entry.
func (tc *TokenCounts) AddUsage(u TokenUsageRaw) {
	tc.InputTokens += u.InputTokens
	tc.OutputTokens += u.OutputTokens
	tc.CacheCreation += u.CacheCreationTokenCount()
	tc.CacheRead += u.CacheReadInputTokens
}

// Total returns the sum of all token categories.
func (tc TokenCounts) Total() uint64 {
	return tc.InputTokens + tc.OutputTokens + tc.CacheCreation + tc.CacheRead + tc.ExtraTotalTokens
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Model breakdown
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

// ModelBreakdown provides per-model token and cost detail.
type ModelBreakdown struct {
	ModelName         string  `json:"modelName"`
	InputTokens       uint64  `json:"inputTokens"`
	OutputTokens      uint64  `json:"outputTokens"`
	CacheCreation     uint64  `json:"cacheCreationTokens"`
	CacheRead         uint64  `json:"cacheReadTokens"`
	ExtraTotalTokens  uint64  `json:"-"`
	Cost              float64 `json:"cost"`
	MissingPricing    bool    `json:"-"`
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Final report summary
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

// UsageSummary is a single row in the final report output.
type UsageSummary struct {
	Date            *string          `json:"date,omitempty"`
	Month           *string          `json:"month,omitempty"`
	Week            *string          `json:"week,omitempty"`
	SessionID       *string          `json:"sessionId,omitempty"`
	ProjectPath     *string          `json:"projectPath,omitempty"`
	LastActivity    *string          `json:"lastActivity,omitempty"`
	FirstActivity   *string          `json:"firstActivity,omitempty"`
	InputTokens     uint64           `json:"inputTokens"`
	OutputTokens    uint64           `json:"outputTokens"`
	CacheCreation   uint64           `json:"cacheCreationTokens"`
	CacheRead       uint64           `json:"cacheReadTokens"`
	ExtraTotal      uint64           `json:"-"`
	TotalCost       float64          `json:"totalCost"`
	Credits         *float64         `json:"credits,omitempty"`
	MessageCount    *uint64          `json:"messageCount,omitempty"`
	ModelsUsed      []string         `json:"modelsUsed"`
	ModelBreakdowns []ModelBreakdown `json:"modelBreakdowns"`
	Project         *string          `json:"project,omitempty"`
	Versions        []string         `json:"versions,omitempty"`
}

// TotalTokens returns the sum of all token fields in the summary.
func (s UsageSummary) TotalTokens() uint64 {
	return s.InputTokens + s.OutputTokens + s.CacheCreation + s.CacheRead + s.ExtraTotal
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Blocks types (Claude-specific)
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

// SessionBlock represents an identified usage session block.
type SessionBlock struct {
	ID                   string
	StartTime            time.Time
	EndTime              time.Time
	ActualEndTime        *time.Time
	IsActive             bool
	IsGap                bool
	Entries              []*LoadedEntry
	TokenCounts          TokenCounts
	CostUSD              float64
	Models               []string
	UsageLimitResetTime  *time.Time
}

// BurnRate captures tokens-per-minute and cost-per-hour rates.
type BurnRate struct {
	TokensPerMinute          float64 `json:"tokensPerMinute"`
	TokensPerMinuteIndicator float64 `json:"tokensPerMinuteForIndicator"`
	CostPerHour              float64 `json:"costPerHour"`
}

// Projection estimates when a token limit will be exhausted.
type Projection struct {
	TotalTokens     uint64  `json:"totalTokens"`
	TotalCost       float64 `json:"totalCost"`
	RemainingMinutes uint64 `json:"remainingMinutes"`
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Codex-specific types
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

// CodexRawUsage mirrors the Rust CodexRawUsage.
type CodexRawUsage struct {
	InputTokens          uint64
	CachedInputTokens    uint64
	OutputTokens         uint64
	ReasoningOutputTokens uint64
	TotalTokens          uint64
}

// CodexTokenUsageEvent is a parsed Codex token_count event.
type CodexTokenUsageEvent struct {
	SessionID            string
	Timestamp            string
	Model                *string
	InputTokens          uint64
	CachedInputTokens    uint64
	OutputTokens         uint64
	ReasoningOutputTokens uint64
	TotalTokens          uint64
	IsFallbackModel      bool
}

// CodexModelUsage tracks usage per model for Codex.
type CodexModelUsage struct {
	InputTokens          uint64
	CachedInputTokens    uint64
	OutputTokens         uint64
	ReasoningOutputTokens uint64
	TotalTokens          uint64
	IsFallback           bool
}

// CodexGroup holds aggregated Codex usage by group key.
type CodexGroup struct {
	InputTokens          uint64
	CachedInputTokens    uint64
	OutputTokens         uint64
	ReasoningOutputTokens uint64
	TotalTokens          uint64
	Models               map[string]*CodexModelUsage
	LastActivity         *string
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Enums
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

// SortOrder for report rows.
type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)

// WeekDay for weekly bucket start-of-week.
type WeekDay int

const (
	WeekSunday    WeekDay = 0
	WeekMonday    WeekDay = 1
	WeekTuesday   WeekDay = 2
	WeekWednesday WeekDay = 3
	WeekThursday  WeekDay = 4
	WeekFriday    WeekDay = 5
	WeekSaturday  WeekDay = 6
)

// ReportKind categorizes report granularity.
type ReportKind string

const (
	ReportDaily   ReportKind = "daily"
	ReportWeekly  ReportKind = "weekly"
	ReportMonthly ReportKind = "monthly"
	ReportSession ReportKind = "session"
)
