package kilo

import "github.com/archhaeondlg/aiusage/internal/adapter/shared"

type KiloAdapter struct{ *shared.GenericAdapter }

func NewAdapter() *KiloAdapter {
	return &KiloAdapter{shared.NewGenericAdapter("kilo", []string{
		"~/.kilo",
		"~/.config/kilo",
	})}
}
