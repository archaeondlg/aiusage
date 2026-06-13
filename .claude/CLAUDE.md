# aiusage (Go)

Analyze coding agent CLI token usage and costs from local data.

## Build & Test

```bash
go build -o aiusage .
go test ./...
```

## Architecture

```
main.go              → entry point
internal/
  adapter/           → adapter interface + 15 agent implementations
  cli/               → cobra + viper CLI framework
  output/            → terminal table + JSON output
  pricing/           → embedded pricing engine (go:embed)
  summary/           → aggregation & filtering
  blocks/            → Claude Code blocks/burn rate
  dateutil/          → timezone-aware date utilities
  types/             → shared data types
```
