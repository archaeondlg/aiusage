package gemini

import "github.com/archhaeondlg/aiusage/internal/adapter/shared"

type GeminiAdapter struct{ *shared.GenericAdapter }

func NewAdapter() *GeminiAdapter {
	return &GeminiAdapter{shared.NewGenericAdapter("gemini", []string{
		"~/.gemini",
		"~/.config/gemini",
		"~/.config/google-gemini",
	})}
}
