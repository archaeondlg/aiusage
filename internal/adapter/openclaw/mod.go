package openclaw

import "github.com/archhaeondlg/aiusage/internal/adapter/shared"

type OpenclawAdapter struct{ *shared.GenericAdapter }

func NewAdapter() *OpenclawAdapter {
	return &OpenclawAdapter{shared.NewGenericAdapter("openclaw", []string{
		"~/.openclaw",
		"~/.config/openclaw",
	})}
}
