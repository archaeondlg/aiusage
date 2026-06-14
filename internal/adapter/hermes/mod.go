package hermes

import "github.com/archhaeondlg/aiusage/internal/adapter/shared"

type HermesAdapter struct{ *shared.GenericAdapter }

func NewAdapter() *HermesAdapter {
	return &HermesAdapter{shared.NewGenericAdapter("hermes", []string{
		"~/.hermes",
		"~/.config/hermes",
	})}
}
