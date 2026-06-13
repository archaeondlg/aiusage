package claude

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/archhaeondlg/aiusage/internal/adapter"
)

// ClaudePaths discovers Claude Code data directories.
// Equivalent to Rust claude_paths().
func ClaudePaths() ([]string, error) {
	var paths []string
	seen := make(map[string]bool)

	// Check CLAUDE_CONFIG_DIR environment variable first.
	if envPaths := os.Getenv("CLAUDE_CONFIG_DIR"); envPaths != "" {
		for _, raw := range strings.Split(envPaths, ",") {
			raw = strings.TrimSpace(raw)
			if raw == "" {
				continue
			}
			path := normalizeClaudeConfigPath(raw)
			projectsDir := filepath.Join(path, "projects")
			if info, err := os.Stat(projectsDir); err == nil && info.IsDir() {
				if !seen[path] {
					seen[path] = true
					paths = append(paths, path)
				}
			}
		}
		if len(paths) > 0 {
			return paths, nil
		}
		return nil, &adapter.Error{
			Message: "No valid Claude data directories found in CLAUDE_CONFIG_DIR. Expected each path to be a Claude config directory containing 'projects/', or the 'projects/' directory itself: " + envPaths,
		}
	}

	// Fall back to default locations.
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg == "" {
		xdg = filepath.Join(home, ".config")
	}

	for _, path := range []string{
		filepath.Join(xdg, "claude"),
		filepath.Join(home, ".claude"),
	} {
		projectsDir := filepath.Join(path, "projects")
		if info, err := os.Stat(projectsDir); err == nil && info.IsDir() {
			if !seen[path] {
				seen[path] = true
				paths = append(paths, path)
			}
		}
	}
	return paths, nil
}

func normalizeClaudeConfigPath(raw string) string {
	path := expandHomePath(raw)
	// If pointing directly to a 'projects' directory, use its parent.
	if filepath.Base(path) == "projects" {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return filepath.Dir(path)
		}
	}
	return path
}

func expandHomePath(raw string) string {
	if raw == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
	}
	if strings.HasPrefix(raw, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, raw[2:])
		}
	}
	return raw
}

// UsageFiles discovers all JSONL usage files under the given Claude data paths.
func UsageFiles(paths []string, projectFilter string) []string {
	var files []string
	for _, path := range paths {
		projectsDir := filepath.Join(path, "projects")
		if IsProjectPathSegment(projectFilter) {
			collectUsageFiles(filepath.Join(projectsDir, projectFilter), &files)
		} else if projectFilter != "" {
			// Non-segment filter: fall back to full discovery with post-filtering.
			collectUsageFiles(projectsDir, &files)
		} else {
			collectUsageFiles(projectsDir, &files)
		}
	}
	// Sort for deterministic output.
	sortStrings(files)
	return files
}

func sortStrings(s []string) {
	for i := 0; i < len(s); i++ {
		for j := i + 1; j < len(s); j++ {
			if s[i] > s[j] {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}

// IsProjectPathSegment validates a project filter value.
func IsProjectPathSegment(value string) bool {
	return value != "" && value != "." && value != ".." &&
		!strings.Contains(value, "/") && !strings.Contains(value, "\\")
}

func collectUsageFiles(dir string, files *[]string) {
	CollectFilesWithExtension(dir, ".jsonl", files)
}

// CollectFilesWithExtension recursively finds all files with a given extension.
func CollectFilesWithExtension(dir string, ext string, files *[]string) {
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip inaccessible files.
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ext) {
			*files = append(*files, path)
		}
		return nil
	})
}

// ExtractProject extracts the project name from a path after the "projects" component.
func ExtractProject(path string) string {
	parts := splitPath(path)
	sawProjects := false
	for _, part := range parts {
		if sawProjects {
			if strings.TrimSpace(part) == "" {
				return "unknown"
			}
			return part
		}
		if part == "projects" {
			sawProjects = true
		}
	}
	return "unknown"
}

// ExtractSessionParts extracts session ID and project path from a Claude usage file path.
func ExtractSessionParts(path string) (sessionID, projectPath string) {
	parts := splitPath(path)
	projectsIdx := -1
	for i, p := range parts {
		if p == "projects" {
			projectsIdx = i
			break
		}
	}

	var relative []string
	if projectsIdx >= 0 && projectsIdx+1 < len(parts) {
		relative = parts[projectsIdx+1:]
	} else {
		relative = parts
	}

	// Check for file-based session ID: projects/project/session.jsonl
	fileSessionID := ""
	if last := relative[len(relative)-1]; strings.HasSuffix(last, ".jsonl") {
		fileSessionID = strings.TrimSuffix(last, ".jsonl")
	}
	if len(relative) == 2 && fileSessionID != "" {
		return fileSessionID, relative[0]
	}

	// Check for subagent path: .../session/subagents/agent.jsonl
	if len(relative) >= 4 && relative[len(relative)-2] == "subagents" {
		sid := relative[len(relative)-3]
		pp := strings.Join(relative[:len(relative)-3], string(filepath.Separator))
		if pp == "" {
			pp = "Unknown Project"
		}
		return sid, pp
	}

	// Generic extraction.
	idx := len(relative) - 2
	if idx < 0 {
		idx = 0
	}
	sid := "unknown"
	if idx < len(relative) {
		sid = relative[idx]
	}
	pp := "Unknown Project"
	if idx > 0 {
		pp = strings.Join(relative[:idx], string(filepath.Separator))
	}
	return sid, pp
}

func splitPath(path string) []string {
	path = filepath.ToSlash(path)
	parts := strings.Split(path, "/")
	// Filter empty parts.
	var result []string
	for _, p := range parts {
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// LoadOptions adapts the adapter.LoadOptions to Claude-specific needs.
func convertLoadOptions(opts adapter.LoadOptions) loadOptions {
	return loadOptions{
		Pricing:       opts.Pricing,
		Timezone:      opts.Timezone,
		Since:         opts.Since,
		Until:         opts.Until,
		JSON:          opts.JSON,
		SingleThread:  opts.SingleThread,
		ProjectFilter: opts.ProjectFilter,
		Verbose:       opts.Verbose,
	}
}

type loadOptions struct {
	Pricing       interface{}
	Timezone      string
	Since         string
	Until         string
	JSON          bool
	SingleThread  bool
	ProjectFilter string
	Verbose       int
}
