package opencode

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// opencodeDataDirEnv is the environment variable for OpenCode data directory.
const opencodeDataDirEnv = "OPENCODE_DATA_DIR"

// paths returns all discovered OpenCode data directories.
func paths() ([]string, error) {
	var result []string
	seen := make(map[string]bool)

	if envPaths := os.Getenv(opencodeDataDirEnv); envPaths != "" {
		for _, raw := range strings.Split(envPaths, ",") {
			raw = strings.TrimSpace(raw)
			if raw == "" {
				continue
			}
			if info, err := os.Stat(raw); err == nil && info.IsDir() && !seen[raw] {
				seen[raw] = true
				result = append(result, raw)
			}
		}
		return result, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(home, ".local", "share", "opencode")
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		result = append(result, path)
	}
	return result, nil
}

// dbPath returns the primary SQLite database path.
func dbPath(opencodeDir string) string {
	defaultDB := filepath.Join(opencodeDir, "opencode.db")
	if _, err := os.Stat(defaultDB); err == nil {
		return defaultDB
	}
	// Find channel-specific DBs like opencode-beta.db, opencode-stable.db
	entries, err := os.ReadDir(opencodeDir)
	if err != nil {
		return ""
	}
	var candidates []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if isChannelDBName(name) {
			candidates = append(candidates, filepath.Join(opencodeDir, name))
		}
	}
	if len(candidates) == 0 {
		return ""
	}
	sort.Strings(candidates)
	return candidates[0]
}

func isChannelDBName(name string) bool {
	prefix := "opencode-"
	suffix := ".db"
	if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, suffix) {
		return false
	}
	middle := name[len(prefix) : len(name)-len(suffix)]
	for _, ch := range middle {
		if !isAlphanumeric(byte(ch)) && ch != '_' && ch != '-' {
			return false
		}
	}
	return true
}

func isAlphanumeric(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}
