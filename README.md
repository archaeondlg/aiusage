<p align="center">
  <img src="https://img.shields.io/badge/aiusage-Token_Usage_Analyzer-4757E8?style=for-the-badge&logo=go&logoColor=white" alt="aiusage">
</p>

<p align="center">
  <a href="https://github.com/archaeondlg/aiusage/releases/latest"><img src="https://img.shields.io/github/v/release/archaeondlg/aiusage?style=flat-square&logo=github&label=release&color=4757E8" alt="Release"></a>
  <a href="https://go.dev/"><img src="https://img.shields.io/github/go-mod/go-version/archaeondlg/aiusage?style=flat-square&logo=go&label=go&color=1E4FD9" alt="Go"></a>
  <a href="https://goreportcard.com/report/github.com/archaeondlg/aiusage"><img src="https://goreportcard.com/badge/github.com/archaeondlg/aiusage?style=flat-square" alt="Go Report"></a>
  <a href="LICENSE"><img src="https://img.shields.io/github/license/archaeondlg/aiusage?style=flat-square&label=license&color=002FA7" alt="License"></a>
</p>

<p align="center">
  🔍 <b>Parse</b> &nbsp;·&nbsp; 📊 <b>Analyze</b> &nbsp;·&nbsp; 💰 <b>Track costs</b><br>
  Token usage for <b>15 coding agent CLIs</b> — from local logs, no API keys needed.
</p>

---

## 🚀 Quick Start

```bash
go build -o aiusage .

# Claude Code (default agent)
aiusage daily

# Specific agent
aiusage daily -a codex
aiusage weekly -a opencode

# All agents
aiusage all

# Filter by model name (fuzzy match)
aiusage daily -m deepseek
aiusage weekly -a all -m opus

# Filter by project
aiusage daily -p D--Project-aiusage

# Date range (multiple formats supported)
aiusage daily -s 2026-01-02 -u '2026-06-15 15:04:05'

# Verbose progress + JSON
aiusage daily -vvv --json | jq '.totals'
```

### 📋 Commands

| Command | Description |
|---------|-------------|
| `daily` | Daily breakdown (default) |
| `weekly` | Weekly aggregation |
| `monthly` | Monthly aggregation |
| `session` | Per-session breakdown |
| `blocks` | Session blocks + burn rate + projection |
| `statusline` | One-liner for Claude Code hook |
| `all` | All agents combined |
| `daemon` | Live watch mode |
| `update` | Self-update binary |
| `update-price` | Sync pricing from GitHub |

### 🏁 Flags

| Flag | Description |
|------|-------------|
| `-a`, `--agent` | Agent (default `claude`: codex, opencode, all, etc.) |
| `-m`, `--model` | Filter by model name, fuzzy match |
| `-p`, `--project` | Filter by project |
| `-s`, `--since` | Start date (`YYYY-MM-DD` or `YYYY-MM-DD HH:MM:SS`) |
| `-u`, `--until` | End date (same formats) |
| `-v`, `-vv`, `-vvv` | Verbosity |
| `--json` | JSON output |
| `--timezone` | Timezone for grouping |
| `--compact` | Compact table layout |
| `--no-color` | Disable ANSI colors |
| `--breakdown` | Per-model breakdown rows |
| `--order` | Sort (`asc`/`desc`) |
| `--token-limit` | Projection target (blocks) |
| `--session-length` | Session timeout in hours (blocks) |

---

## 🤖 Supported Agents

| Level | Agents |
|-------|--------|
| 🌟 Full | Claude Code, Codex CLI, OpenCode |
| ⚡ Generic | Amp, Codebuff, Copilot, Droid, Gemini, Goose, Hermes, Kilo, Kimi, OpenClaw, pi-agent, Qwen |

---

## 👁️ Daemon Mode

```bash
aiusage daemon                    # Watch Claude Code, 30s interval
aiusage daemon -i 10              # 10s polling
aiusage daemon -a all             # All agents
aiusage daemon --json             # JSON-line output
```

## 🧱 Blocks Mode

```bash
aiusage blocks                     # Today's session blocks
aiusage blocks --token-limit 500K  # Project when you'll hit 500K
aiusage blocks --active            # Only active blocks
```

## ⚡ Statusline

```bash
aiusage statusline
# ⚡ deepseek-v4-pro │ $0.46 │ 60.4M │ 3h30m
```

Fields: active model · session cost · tokens used · session duration

---

## ⚙️ Configuration

Place `config.json` next to the binary. Auto-downloaded from GitHub on first use.

```json
{
  "pricing": {
    "claude-sonnet-4-5": {
      "input": 0.000003,
      "output": 0.000015,
      "cacheRead": 0.0000003,
      "cacheCreation": 0.000003
    }
  }
}
```

Prices in **dollars per token** (`$3/M tokens` → `0.000003`). Built-in defaults cover 100+ models. Run `aiusage update-price` to sync latest.

---

## 🏗️ Architecture

```
aiusage
├── main.go
├── internal/
│   ├── adapter/           Agent adapters (claude, codex, opencode + 12 generic)
│   ├── blocks/            Sessionization, burn rate, projection
│   ├── cli/               Cobra commands, config, flags
│   ├── daemon/            Watch/poll loop
│   ├── dateutil/          Time parsing & formatting
│   ├── output/            Tables, JSON, ANSI styling
│   ├── pricing/           Model pricing (built-in + GitHub)
│   ├── summary/           Aggregation, filtering, sorting
│   ├── types/             Core data structures
│   └── update/            Self-update
```

---

## 🧪 Testing

```bash
go test ./...                 # 139 tests · 10 packages
go test -cover ./...          # With coverage
```

---

## 🛠️ Build

```bash
go build -o aiusage .

# Cross-compile
GOOS=windows GOARCH=amd64 go build -o aiusage.exe .
GOOS=linux   GOARCH=amd64 go build -o aiusage .
GOOS=darwin  GOARCH=arm64 go build -o aiusage .
```

### 🚢 Release

```bash
git tag v3.0.0
git push origin v3.0.0
```

Triggers GitHub Actions + GoReleaser — builds Windows/Linux/macOS × amd64/arm64.

---

<p align="center">
  <a href="LICENSE">📄 MIT License</a> &nbsp;·&nbsp;
  <a href="https://github.com/archaeondlg/aiusage">🐙 github.com/archaeondlg/aiusage</a>
</p>
