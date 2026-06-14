package kimi

import "github.com/archhaeondlg/aiusage/internal/adapter/shared"

type KimiAdapter struct{ *shared.GenericAdapter }

func NewAdapter() *KimiAdapter {
	return &KimiAdapter{shared.NewGenericAdapter("kimi", []string{
		"~/.kimi",
		"~/.config/kimi",
	})}
}
