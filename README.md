# aiusage

[![Release](https://img.shields.io/github/v/release/archaeondlg/aiusage)](https://github.com/archaeondlg/aiusage/releases/latest)
[![Go Version](https://img.shields.io/github/go-mod/go-version/archaeondlg/aiusage)](https://go.dev/)
[![License](https://img.shields.io/github/license/archaeondlg/aiusage)](LICENSE)

Analyze token usage and costs across all your coding agent CLIs — from local log data, no API keys needed.

## Supported Agents

| Agent | Default? | Description |
|-------|----------|-------------|
| **Claude Code** | ✅ | Anthropic's official CLI |
| **Codex** | | OpenAI's coding agent |
| **OpenCode** | | Open-source coding CLI |
| Amp, Codebuff, Copilot, Droid, Gemini, Goose, Hermes, Kilo, Kimi, OpenClaw, pi-agent, Qwen | | Auto-detected via `aiusage all` |

## Quick Start

```bash
# Download latest binary from GitHub Releases
# Or build from source:
go build -o aiusage .

# Place config.json next to the binary (auto-downloaded if missing)
# Generate one from model_pricing.csv or run without for built-in defaults

# Daily usage (default: Claude Code)
aiusage daily

# Verbose progress
aiusage daily -vvv

# All detected agents
aiusage all

# JSON output
aiusage daily --json
```

## Commands

| Command | Description |
|---------|-------------|
| `daily` | Daily token usage report (default: Claude Code) |
| `weekly` | Weekly aggregation |
| `monthly` | Monthly aggregation |
| `session` | Per-session breakdown |
| `blocks` | Session blocks with burn rate analysis |
| `daemon` | Watch mode — live stats at configurable interval |
| `statusline` | One-liner for Claude Code statusline hook |
| `all` | Aggregate usage from all detected agents |
| `update` | Self-update binary from GitHub Releases |
| `update-price` | Update pricing in config.json from GitHub |

## Flags

| Flag | Description |
|------|-------------|
| `--json` | JSON output |
| `-v`, `-vv`, `-vvv` | Verbose progress |
| `--since`, `--until` | Date range filter (YYYY-MM-DD) |
| `--project` | Filter to specific project |
| `--timezone` | Timezone for date grouping |
| `--compact` | Force compact table layout |
| `--no-color` | Disable ANSI colors |

## Configuration

Place `config.json` next to the binary. Auto-downloaded from GitHub on first use.

```json
{
  "pricing": {
    "deepseek-v4-pro": {
      "input": 0.000000435,
      "output": 0.00000087,
      "cacheRead": 0.0000000036,
      "contextLimit": 1048576
    }
  }
}
```

All prices in **dollars per token** ($5/1M tokens = `0.000005`).

Built-in defaults cover 100+ models. Run `aiusage update-price` to fetch the latest.

## Daemon Mode

```bash
aiusage daemon                    # Watch Claude Code, 30s interval
aiusage daemon --interval 10      # 10-second polling
aiusage daemon --agent all        # Monitor all agents
aiusage daemon --json             # JSON-line output for piping
```

## Statusline

```bash
aiusage statusline
# ⚡ deepseek-v4-pro │ $0.46 │ 60.4M │ 3h30m
```

Fields: active model, session cost, tokens used, session duration.

## Build

```bash
go build -o aiusage .
```

### Cross-compile

```bash
GOOS=windows GOARCH=amd64 go build -o aiusage.exe .
GOOS=linux   GOARCH=amd64 go build -o aiusage .
GOOS=darwin  GOARCH=arm64 go build -o aiusage .
```

## Release

Tag and push to trigger automated release via GitHub Actions + GoReleaser:

```bash
git tag v2.0.0
git push origin v2.0.0
```

Builds Windows/Linux/macOS × amd64/arm64 binaries.

## License

MIT
