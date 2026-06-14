package codebuff

import "github.com/archhaeondlg/aiusage/internal/adapter/shared"

type CodebuffAdapter struct{ *shared.GenericAdapter }

func NewAdapter() *CodebuffAdapter {
	return &CodebuffAdapter{shared.NewGenericAdapter("codebuff", []string{
		"~/.codebuff",
	})}
}
