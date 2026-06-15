// Package all registers all known coding agent adapters with the adapter registry.
package all

import (
	"github.com/archhaeondlg/aiusage/internal/adapter"
	"github.com/archhaeondlg/aiusage/internal/adapter/amp"
	"github.com/archhaeondlg/aiusage/internal/adapter/claude"
	"github.com/archhaeondlg/aiusage/internal/adapter/codebuff"
	"github.com/archhaeondlg/aiusage/internal/adapter/codex"
	"github.com/archhaeondlg/aiusage/internal/adapter/copilot"
	"github.com/archhaeondlg/aiusage/internal/adapter/droid"
	"github.com/archhaeondlg/aiusage/internal/adapter/gemini"
	"github.com/archhaeondlg/aiusage/internal/adapter/goose"
	"github.com/archhaeondlg/aiusage/internal/adapter/hermes"
	"github.com/archhaeondlg/aiusage/internal/adapter/kilo"
	"github.com/archhaeondlg/aiusage/internal/adapter/kimi"
	"github.com/archhaeondlg/aiusage/internal/adapter/openclaw"
	"github.com/archhaeondlg/aiusage/internal/adapter/opencode"
	"github.com/archhaeondlg/aiusage/internal/adapter/pi"
	"github.com/archhaeondlg/aiusage/internal/adapter/qwen"
)

func init() {
	adapter.Register("claude", func() adapter.Adapter { return claude.NewAdapter() })
	adapter.Register("codex", func() adapter.Adapter { return codex.NewAdapter() })
	adapter.Register("opencode", func() adapter.Adapter { return opencode.NewAdapter() })
	adapter.Register("amp", func() adapter.Adapter { return amp.NewAdapter() })
	adapter.Register("codebuff", func() adapter.Adapter { return codebuff.NewAdapter() })
	adapter.Register("copilot", func() adapter.Adapter { return copilot.NewAdapter() })
	adapter.Register("droid", func() adapter.Adapter { return droid.NewAdapter() })
	adapter.Register("gemini", func() adapter.Adapter { return gemini.NewAdapter() })
	adapter.Register("goose", func() adapter.Adapter { return goose.NewAdapter() })
	adapter.Register("hermes", func() adapter.Adapter { return hermes.NewAdapter() })
	adapter.Register("kilo", func() adapter.Adapter { return kilo.NewAdapter() })
	adapter.Register("kimi", func() adapter.Adapter { return kimi.NewAdapter() })
	adapter.Register("openclaw", func() adapter.Adapter { return openclaw.NewAdapter() })
	adapter.Register("pi", func() adapter.Adapter { return pi.NewAdapter() })
	adapter.Register("qwen", func() adapter.Adapter { return qwen.NewAdapter() })
}
