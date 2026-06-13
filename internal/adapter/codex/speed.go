package codex

import (
	"os"
	"path/filepath"
	"strings"
)

type CodexSpeedMode string

const (
	CodexSpeedAuto     CodexSpeedMode = "auto"
	CodexSpeedStandard CodexSpeedMode = "standard"
	CodexSpeedFast     CodexSpeedMode = "fast"
)

// ResolveCodexSpeed determines the effective speed tier.
func ResolveCodexSpeed(requested CodexSpeedMode) CodexSpeedMode {
	if requested == CodexSpeedAuto {
		if detectFastServiceTier() {
			return CodexSpeedFast
		}
		return CodexSpeedStandard
	}
	return requested
}

func detectFastServiceTier() bool {
	homes, err := codexHomePaths()
	if err != nil {
		return false
	}
	for _, home := range homes {
		configPath := filepath.Join(home, "config.toml")
		data, err := os.ReadFile(configPath)
		if err != nil {
			continue
		}
		if configRequestsFastServiceTier(string(data)) {
			return true
		}
	}
	return false
}

func configRequestsFastServiceTier(content string) bool {
	for _, line := range strings.Split(content, "\n") {
		// Strip comments.
		if idx := strings.IndexByte(line, '#'); idx >= 0 {
			line = line[:idx]
		}
		setting := strings.TrimSpace(line)
		parts := strings.SplitN(setting, "=", 2)
		if len(parts) != 2 {
			continue
		}
		if strings.TrimSpace(parts[0]) != "service_tier" {
			continue
		}
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, `"'`)
		if value == "fast" || value == "priority" {
			return true
		}
	}
	return false
}
