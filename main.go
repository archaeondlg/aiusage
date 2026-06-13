// aiusage — Analyze coding agent CLI token usage and costs from local data.
//
// aiusage reads local usage logs from Claude Code, Codex, OpenCode, Amp,
// Droid, Codebuff, Hermes Agent, pi-agent, Goose, OpenClaw, Kilo, Kimi,
// Qwen, GitHub Copilot CLI, and Gemini CLI to track tokens and estimate costs.
package main

import (
	"fmt"
	"os"

	"github.com/archhaeondlg/aiusage/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "aiusage: %v\n", err)
		fmt.Fprintln(os.Stderr, "Run 'aiusage --help' for usage.")
		os.Exit(2)
	}
}
