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

## 🎯 What It Does

| | |
|---|---|
| 📅 **Daily / Weekly / Monthly** | Aggregated token & cost reports |
| 🔗 **Per-Session** | Breakdown by coding session |
| 🧱 **Blocks + Burn Rate** | Session blocks with tokens/min projection |
| 👁️ **Daemon Mode** | Live watch with auto-refresh |
| ⚡ **Statusline** | One-liner for Claude Code hook |
| 📄 **JSON Output** | Machine-readable for piping |
| 🧠 **Reasoning Tokens** | Extended thinking tracking (Claude, Codex) |

---

## 🤖 Supported Agents

| Level | Agents |
|-------|--------|
| 🌟 <b>Full</b> | <img src="https://img.shields.io/badge/Claude_Code-4757E8?style=flat-square&logo=anthropic" height="20"> &nbsp; <img src="https://img.shields.io/badge/Codex_CLI-1E4FD9?style=flat-square&logo=openai" height="20"> &nbsp; <img src="https://img.shields.io/badge/OpenCode-002FA7?style=flat-square&logo=openai" height="20"> |
| ⚡ <b>Generic</b> | <img src="https://img.shields.io/badge/Amp-4757E8?style=flat-square" height="20"> &nbsp; <img src="https://img.shields.io/badge/Codebuff-3B4AD8?style=flat-square" height="20"> &nbsp; <img src="https://img.shields.io/badge/Copilot-2E3FC8?style=flat-square" height="20"> &nbsp; <img src="https://img.shields.io/badge/Droid-2234B8?style=flat-square" height="20"> &nbsp; <img src="https://img.shields.io/badge/Gemini-1E2EA8?style=flat-square" height="20"> &nbsp; <img src="https://img.shields.io/badge/Goose-192898?style=flat-square" height="20"> |
| ⚡ <b>Generic</b> | <img src="https://img.shields.io/badge/Hermes-4757E8?style=flat-square" height="20"> &nbsp; <img src="https://img.shields.io/badge/Kilo_Code-3B4AD8?style=flat-square" height="20"> &nbsp; <img src="https://img.shields.io/badge/Kimi_CLI-2E3FC8?style=flat-square" height="20"> &nbsp; <img src="https://img.shields.io/badge/OpenClaw-2234B8?style=flat-square" height="20"> &nbsp; <img src="https://img.shields.io/badge/pi--agent-1E2EA8?style=flat-square" height="20"> &nbsp; <img src="https://img.shields.io/badge/Qwen-192898?style=flat-square" height="20"> |

📌 <b>Full</b> — Custom parser: reasoning tokens, session dedup, subagent replay  
📌 <b>Generic</b> — Auto-detected JSONL reader via `shared.GenericAdapter`

---

## 🚀 Quick Start

```bash
# 🏗️ Build
go build -o aiusage .

# 📅 Daily usage (default: Claude Code)
aiusage daily

# 🎯 Specific agent
aiusage daily --agent codex

# 📡 All agents combined
aiusage all

# 🔍 Verbose progress
aiusage daily -vvv

# 📄 JSON output
aiusage daily --json | jq '.totals'
```

### 📋 Commands

| Command | Description |
|---------|-------------|
| `aiusage daily` | 📅 Daily breakdown |
| `aiusage weekly` | 📆 Weekly aggregation |
| `aiusage monthly` | 🗓️ Monthly aggregation |
| `aiusage session` | 🔗 Per-session breakdown |
| `aiusage blocks` | 🧱 Session blocks + burn rate + projection |
| `aiusage statusline` | ⚡ One-liner for Claude Code statusline |
| `aiusage all` | 📡 All agents combined |
| `aiusage daemon` | 👁️ Live watch mode |
| `aiusage update` | ⬆️ Self-update |
| `aiusage update-price` | 💰 Sync pricing from GitHub |

### 🏁 Flags

| Flag | Description |
|------|-------------|
| `--json` | 📄 JSON output |
| `-v`, `-vv`, `-vvv` | 🔍 Verbosity level |
| `--since`, `--until` | 📅 Date range (`YYYY-MM-DD`) |
| `--project` | 🎯 Filter by project |
| `--agent` | 🤖 Select agent |
| `--timezone` | 🌍 Timezone for grouping |
| `--compact` | 📏 Compact table layout |
| `--no-color` | 🚫 Disable ANSI |
| `--breakdown` | 🔬 Per-model breakdown |
| `--order` | ⬆️⬇️ Sort (`asc`/`desc`) |
| `--token-limit` | 🎯 Projection target |
| `--session-length` | ⏱️ Session timeout (min) |
| `--active` | 🔄 Active blocks only |

---

## 👁️ Daemon Mode

```bash
aiusage daemon                    # 👀 Watch Claude Code, 30s interval
aiusage daemon --interval 10      # ⏱️ 10s polling
aiusage daemon --agent all        # 📡 All agents
aiusage daemon --json             # 📄 JSON-line output
```

```
┌── aiusage daemon · claude · every 30s · Ctrl+C to quit ──┐
│ 🕐 Updated:     2026-06-15 23:45:30                       │
│ 🧠 Models:      claude-sonnet-4-5, claude-opus-4          │
│ 📥 Input:       12,345                                    │
│ 📤 Output:      6,789                                     │
│ 📊 Total:       19,134                                    │
│ 💰 Cost:        $0.42                                     │
│ 📝 Entries:     143                                       │
└───────────────────────────────────────────────────────────┘
```

## 🧱 Blocks Mode

```bash
aiusage blocks                     # 📊 Today's session blocks
aiusage blocks --token-limit 500K  # 🎯 Project when you'll hit 500K tokens
aiusage blocks --active            # 🔄 Only active blocks
```

📊 Shows: block start → end, duration (HH:MM), tokens used, burn rate (tokens/min), projected limit hit date.

## ⚡ Statusline

```bash
aiusage statusline
# ⚡ claude-sonnet-4-5 │ $0.46 │ 60.4M │ 3h30m
```

📌 Fields: active model · session cost · tokens used · session duration

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
  },
  "defaults": {
    "timezone": "Asia/Shanghai",
    "interval": "30s"
  }
}
```

💡 Prices in **dollars per token**. `$3/M tokens` → `0.000003`  
📦 Built-in defaults: **100+ models**. Run `aiusage update-price` to sync.

---

## 🏗️ Architecture

```
aiusage
├── main.go                  🚪 Entry → cli.Execute()
├── internal/
│   ├── adapter/             🔌 Agent adapters
│   │   ├── claude/          🌟 JSONL parser, dedup, sessions, subagents
│   │   ├── codex/           🌟 Group parser, archived sessions, dedup
│   │   ├── opencode/        🌟 SQLite + JSON message reader
│   │   ├── shared/          ⚡ Generic JSONL adapter (12 agents)
│   │   └── all/             📋 Central init() registry
│   ├── blocks/              🧱 Sessionization, burn rate, projection
│   ├── cli/                 🎮 Cobra commands, config, flags
│   ├── daemon/              👁️ Watch/poll loop
│   ├── dateutil/            📅 Time parsing & formatting
│   ├── output/              🎨 Tables, JSON, ANSI styling
│   ├── pricing/             💰 Model pricing (built-in + GitHub)
│   ├── summary/             📊 Aggregation, filtering, sorting
│   ├── types/               📦 Core data structures
│   └── update/              ⬆️ Self-update
```

---

## 🧪 Testing

```bash
go test ./...                 # ✅ 139 tests · 10 packages
go test -cover ./...          # 📊 With coverage
go generate ./...             # 🔄 Regenerate adapter wrappers
```

| Package | 🧪 Tests | | Package | 🧪 Tests |
|---------|--------|-|---------|--------|
| `adapter/claude` (loader) | 15 | | `adapter/claude` (parser) | 13 |
| `adapter/shared` | 22 | | `blocks` | 21 |
| `cli` | 14 | | `daemon` | 4 |
| `dateutil` | 8 | | `output` | 9 |
| `pricing` | 11 | | `summary` | 22 |

---

## 🛠️ Build

```bash
go build -o aiusage .
```

### Cross-Compile

| Platform | Command |
|----------|---------|
| 🪟 Windows amd64 | `GOOS=windows GOARCH=amd64 go build -o aiusage.exe .` |
| 🐧 Linux amd64 | `GOOS=linux GOARCH=amd64 go build -o aiusage .` |
| 🐧 Linux arm64 | `GOOS=linux GOARCH=arm64 go build -o aiusage .` |
| 🍏 macOS amd64 | `GOOS=darwin GOARCH=amd64 go build -o aiusage .` |
| 🍏 macOS arm64 | `GOOS=darwin GOARCH=arm64 go build -o aiusage .` |

### 🚢 Release

```bash
git tag v3.0.0
git push origin v3.0.0
```

⬆️ Triggers GitHub Actions + GoReleaser — builds all 5 targets automatically.

---

## 🎨 Color Palette

| Role | Hex | Preview |
|------|-----|---------|
| 🟦 Primary | `#4757E8` | ![#4757E8](https://via.placeholder.com/15/4757E8/000000?text=+) |
| 🟦 Medium | `#1E4FD9` | ![#1E4FD9](https://via.placeholder.com/15/1E4FD9/000000?text=+) |
| 🟦 Deep | `#002FA7` | ![#002FA7](https://via.placeholder.com/15/002FA7/000000?text=+) |
| 🔴 Alert | `#E84747` | ![#E84747](https://via.placeholder.com/15/E84747/000000?text=+) |
| 🔴 Deep | `#A70000` | ![#A70000](https://via.placeholder.com/15/A70000/000000?text=+) |
| 🟡 Warn | `#E8C747` | ![#E8C747](https://via.placeholder.com/15/E8C747/000000?text=+) |
| 🟡 Deep | `#A78B00` | ![#A78B00](https://via.placeholder.com/15/A78B00/000000?text=+) |

---

<p align="center">
  <a href="LICENSE">📄 MIT License</a> &nbsp;·&nbsp;
  <a href="https://github.com/archaeondlg/aiusage">🐙 github.com/archaeondlg/aiusage</a>
</p>
