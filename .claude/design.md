# 🎨 Design System

## Color Palette

### Blue (Primary) — `#4757E8` → `#002FA7`

| Name | Hex | Usage |
|------|-----|-------|
| Primary | `#4757E8` | Main brand, primary badges, headings |
| Medium | `#1E4FD9` | Secondary badges, hover states |
| Deep | `#002FA7` | Footer, dark accents, license badges |

### Red (Alert/Danger) — `#E84747` → `#A70000`

| Name | Hex | Usage |
|------|-----|-------|
| Alert | `#E84747` | Warnings, errors, destructive actions |
| Deep | `#A70000` | Critical indicators |

### Yellow (Warning/Caution) — `#E8C747` → `#A78B00`

| Name | Hex | Usage |
|------|-----|-------|
| Warn | `#E8C747` | Caution notices, rate limits |
| Deep | `#A78B00` | Persistent warnings |

## Badge Conventions (shields.io)

```markdown
<!-- Primary -->
[![Label](https://img.shields.io/badge/Text-4757E8?style=flat-square&logo=go)](/url)

<!-- Color gradient for tiered items -->
[![Tier1](https://img.shields.io/badge/Name-4757E8?style=flat-square)](/url)
[![Tier2](https://img.shields.io/badge/Name-3B4AD8?style=flat-square)](/url)
[![Tier3](https://img.shields.io/badge/Name-2E3FC8?style=flat-square)](/url)
[![Tier4](https://img.shields.io/badge/Name-2234B8?style=flat-square)](/url)
[![Tier5](https://img.shields.io/badge/Name-1E2EA8?style=flat-square)](/url)
[![Tier6](https://img.shields.io/badge/Name-192898?style=flat-square)](/url)
```

### Style parameters
- `style=flat-square` for inline badges
- `style=for-the-badge` for headers/banners
- `logo=XYZ&logoColor=white` for branded badges
- No black (`#000`) background — use dark blue (`#002FA7`) instead

## Emoji Usage

### Sections
- `## 🎯` — What It Does / Overview
- `## 🤖` — Agents / Integrations
- `## 🚀` — Quick Start / Getting Started
- `## 👁️` — Daemon / Watch / Monitor
- `## 🧱` — Blocks / Sessionization
- `## ⚡` — Statusline / One-liner
- `## ⚙️` — Configuration
- `## 🏗️` — Architecture
- `## 🧪` — Testing
- `## 🛠️` — Build
- `## 🚢` — Release
- `## 🎨` — Color Palette / Design

### Inline
- `📅` — Daily, date
- `📆` — Weekly
- `🗓️` — Monthly
- `🔗` — Session, link
- `📊` — Data, stats, totals
- `📄` — JSON, file
- `🧠` — Reasoning tokens
- `💰` — Cost, pricing
- `🔍` — Verbose, detail
- `🎯` — Target, filter
- `🌍` — Timezone
- `📏` — Compact
- `🚫` — Disable, no-color
- `🔬` — Breakdown
- `⬆️⬇️` — Sort order
- `🔄` — Active, refresh
- `⏱️` — Timeout, interval
- `🤖` — Agent select
- `📡` — All agents
- `⬆️` — Update
- `📝` — Entries
- `📥` — Input tokens
- `📤` — Output tokens
- `📌` — Note, footnote
- `💡` — Tip, info
- `📦` — Package, built-in
- `🪟` — Windows
- `🐧` — Linux
- `🍏` — macOS
- `✅` — Success, pass
- `❌` — Failure
- `🌟` — Full feature
- `⚡` — Generic, lightweight

## Design Guidelines

1. **No black backgrounds** — use deep blue `#002FA7` for dark backgrounds
2. **Emoji before section titles** — one emoji per heading
3. **Shields.io badges** for all links and labels
4. **Color gradient** for tiered/leveled lists (brighter → deeper)
5. **Tables over lists** for structured data
6. **Code blocks** for commands and configs always with language tags
7. **Separators (`---`)** between major sections
8. **Align center** for header banner and footer only
