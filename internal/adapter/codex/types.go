package codex

import (
	_ "embed"
	"encoding/json"
)

// RawUsage mirrors the Rust CodexRawUsage.
type RawUsage struct {
	InputTokens           uint64
	CachedInputTokens     uint64
	OutputTokens          uint64
	ReasoningOutputTokens uint64
	TotalTokens           uint64
}

// TokenUsageEvent is a parsed Codex token_count event.
type TokenUsageEvent struct {
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

// ModelUsage tracks usage per model for Codex.
type ModelUsage struct {
	InputTokens           uint64
	CachedInputTokens     uint64
	OutputTokens          uint64
	ReasoningOutputTokens uint64
	TotalTokens           uint64
	IsFallback            bool
}

// Group holds aggregated Codex usage by group key.
type Group struct {
	InputTokens           uint64
	CachedInputTokens     uint64
	OutputTokens          uint64
	ReasoningOutputTokens uint64
	TotalTokens           uint64
	Models                map[string]*ModelUsage
	LastActivity          *string
}

// Embedded Codex auto-review model fallback table.
// Maps release dates to concrete model names.
//
//go:embed codex-auto-review-fallbacks.json
var codexAutoReviewFallbacksJSON []byte

type autoReviewFallback struct {
	ReleasedOn string `json:"releasedOn"`
	Model      string `json:"model"`
}

var autoReviewFallbacks []autoReviewFallback

func init() {
	if err := json.Unmarshal(codexAutoReviewFallbacksJSON, &autoReviewFallbacks); err != nil {
		panic("failed to parse embedded codex-auto-review-fallbacks.json: " + err.Error())
	}
}

const codexAutoReviewModel = "codex-auto-review"

// codexParsedLine holds the raw fields extracted from a Codex JSONL line.
type codexParsedLine struct {
	Timestamp  json.RawMessage `json:"timestamp"`
	CreatedAt  json.RawMessage `json:"created_at"`
	CreatedAtC json.RawMessage `json:"createdAt"`
	Type       *string         `json:"type"`
	Payload    json.RawMessage `json:"payload"`
	Model      *string         `json:"model"`
	ModelName  *string         `json:"model_name"`
	Metadata   json.RawMessage `json:"metadata"`
	Data       json.RawMessage `json:"data"`
	Result     json.RawMessage `json:"result"`
	Response   json.RawMessage `json:"response"`
	Usage      *codexRawUsage  `json:"usage"`
}

// codexPayload mirrors CodexPayload from Rust.
type codexPayload struct {
	Type      *string        `json:"type"`
	Info      *codexInfo     `json:"info"`
	Model     *string        `json:"model"`
	ModelName *string        `json:"model_name"`
	Metadata  json.RawMessage `json:"metadata"`
}

// codexInfo mirrors CodexInfo from Rust.
type codexInfo struct {
	LastTokenUsage  *codexRawUsage  `json:"last_token_usage"`
	TotalTokenUsage *codexRawUsage  `json:"total_token_usage"`
	Model           *string         `json:"model"`
	ModelName       *string         `json:"model_name"`
	Metadata        json.RawMessage  `json:"metadata"`
}

// codexRawUsage handles the flexible field names in Codex usage JSON.
type codexRawUsage struct {
	InputTokens          uint64 `json:"input_tokens"`
	PromptTokens         *uint64 `json:"prompt_tokens"`
	InputField           *uint64 `json:"input"`
	CachedInputTokens    uint64 `json:"cached_input_tokens"`
	CacheReadTokens      *uint64 `json:"cache_read_input_tokens"`
	CachedTokens         *uint64 `json:"cached_tokens"`
	OutputTokens         uint64 `json:"output_tokens"`
	CompletionTokens     *uint64 `json:"completion_tokens"`
	OutputField          *uint64 `json:"output"`
	ReasoningTokens      uint64 `json:"reasoning_output_tokens"`
	ReasoningField       *uint64 `json:"reasoning_tokens"`
	TotalTokens          uint64 `json:"total_tokens"`
}

// Normalize merges alternative field names into the canonical fields.
func (u *codexRawUsage) normalize() {
	// Input: prefer input_tokens, fall back to prompt_tokens or input field.
	if u.InputTokens == 0 {
		if u.PromptTokens != nil {
			u.InputTokens = *u.PromptTokens
		} else if u.InputField != nil {
			u.InputTokens = *u.InputField
		}
	}
	// Output: prefer output_tokens, fall back to completion_tokens or output field.
	if u.OutputTokens == 0 {
		if u.CompletionTokens != nil {
			u.OutputTokens = *u.CompletionTokens
		} else if u.OutputField != nil {
			u.OutputTokens = *u.OutputField
		}
	}
	// Reasoning.
	if u.ReasoningTokens == 0 && u.ReasoningField != nil {
		u.ReasoningTokens = *u.ReasoningField
	}
	// Cached: prefer cached_input_tokens, then cache_read_input_tokens, then cached_tokens.
	if u.CachedInputTokens == 0 {
		if u.CacheReadTokens != nil {
			u.CachedInputTokens = *u.CacheReadTokens
		} else if u.CachedTokens != nil {
			u.CachedInputTokens = *u.CachedTokens
		}
	}
	// TotalTokens: if 0, compute from parts.
	if u.TotalTokens == 0 {
		u.TotalTokens = u.InputTokens + u.OutputTokens + u.ReasoningTokens
	}
}

func (u codexRawUsage) isZero() bool {
	return u.InputTokens == 0 && u.CachedInputTokens == 0 &&
		u.OutputTokens == 0 && u.ReasoningTokens == 0 && u.TotalTokens == 0
}

// resultFields mirrors CodexResultFields from Rust.
type resultFields struct {
	Timestamp  json.RawMessage `json:"timestamp"`
	CreatedAt  json.RawMessage `json:"created_at"`
	CreatedAtC json.RawMessage `json:"createdAt"`
	Usage      *codexRawUsage  `json:"usage"`
	Model      *string         `json:"model"`
	ModelName  *string         `json:"model_name"`
	Metadata   json.RawMessage `json:"metadata"`
}

// metadataModel is CodexModelMetadata.
type metadataModel struct {
	Model *string `json:"model"`
}
