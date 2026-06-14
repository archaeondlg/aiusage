package qwen

import "github.com/archhaeondlg/aiusage/internal/adapter/shared"

type QwenAdapter struct{ *shared.GenericAdapter }

func NewAdapter() *QwenAdapter {
	return &QwenAdapter{shared.NewGenericAdapter("qwen", []string{
		"~/.qwen",
		"~/.config/qwen",
	})}
}
