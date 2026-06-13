# aiusage

Analyze coding agent CLI token usage and costs from local data.

## Supported Agents

| Agent | Command | Status |
|-------|---------|--------|
| Claude Code | `aiusage` (default) | ✅ Full |
| Codex | `aiusage codex` | ✅ Full |
| OpenCode | `aiusage opencode` | ✅ Full |

## Quick Start

```bash
# Install
go install github.com/archhaeondlg/aiusage@latest

# Daily report (default: Claude Code)
aiusage daily

# JSON output
aiusage daily --json

# Specific agent
aiusage codex --kind weekly
aiusage opencode --json

# All detected agents
aiusage all
```

## Build

```bash
go build -o aiusage .

# Cross-compile all platforms
bash scripts/build-all.sh
```

## Features

- Daily, weekly, monthly, and per-session reports
- JSON output with jq support
- Responsive terminal tables (compact/wide)
- Embedded model pricing (offline mode)
- Timezone-aware date grouping
- Per-model cost breakdowns
- Claude Code blocks and statusline integration

## License

MIT
