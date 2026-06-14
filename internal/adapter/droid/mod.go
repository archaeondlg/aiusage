package droid

import "github.com/archhaeondlg/aiusage/internal/adapter/shared"

type DroidAdapter struct{ *shared.GenericAdapter }

func NewAdapter() *DroidAdapter {
	return &DroidAdapter{shared.NewGenericAdapter("droid", []string{
		"~/.droid",
		"~/.config/droid",
	})}
}
