package codex

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// codexUsageSource represents a Codex data directory with deduplication scope.
type codexUsageSource struct {
	Dir         string
	DedupeScope string
}

// codexUsageFileGroup groups files from one source directory.
type codexUsageFileGroup struct {
	Dir   string
	Files []string
}

// codexHomePaths returns configured Codex home directories.
func codexHomePaths() ([]string, error) {
	if env := os.Getenv("CODEX_HOME"); env != "" {
		var paths []string
		for _, p := range strings.Split(env, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				paths = append(paths, p)
			}
		}
		return paths, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return []string{filepath.Join(home, ".codex")}, nil
}

// codexUsageSources discovers all Codex session directories.
func codexUsageSources() ([]codexUsageSource, error) {
	homes, err := codexHomePaths()
	if err != nil {
		return nil, err
	}
	return codexUsageSourcesFromHomes(homes), nil
}

func codexUsageSourcesFromHomes(homes []string) []codexUsageSource {
	var sources []codexUsageSource
	seen := make(map[string]bool)
	for _, path := range homes {
		sessions := filepath.Join(path, "sessions")
		archived := filepath.Join(path, "archived_sessions")
		foundUsageDir := false

		if info, err := os.Stat(sessions); err == nil && info.IsDir() {
			if !seen[sessions] {
				seen[sessions] = true
				sources = append(sources, codexUsageSource{Dir: sessions, DedupeScope: path})
			}
			foundUsageDir = true
		}
		if info, err := os.Stat(archived); err == nil && info.IsDir() {
			if !seen[archived] {
				seen[archived] = true
				sources = append(sources, codexUsageSource{Dir: archived, DedupeScope: path})
			}
			foundUsageDir = true
		}
		if !foundUsageDir && !seen[path] {
			seen[path] = true
			sources = append(sources, codexUsageSource{Dir: path, DedupeScope: path})
		}
	}
	return sources
}

// collectCodexUsageFiles finds all JSONL files in a directory.
func collectCodexUsageFiles(dir string) []string {
	var files []string
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".jsonl") {
			files = append(files, path)
		}
		return nil
	})
	sortStrings(files)
	return files
}

// collectDedupedCodexUsageFiles groups files by source, deduplicating across sources.
func collectDedupedCodexUsageFiles(sources []codexUsageSource) []codexUsageFileGroup {
	type fileKey struct {
		scope string
		rel   string
	}
	seen := make(map[fileKey]bool)
	var groups []codexUsageFileGroup

	for _, source := range sources {
		files := collectCodexUsageFiles(source.Dir)
		var filtered []string
		for _, file := range files {
			rel, err := filepath.Rel(source.Dir, file)
			if err != nil {
				rel = file
			}
			key := fileKey{source.DedupeScope, rel}
			if !seen[key] {
				seen[key] = true
				filtered = append(filtered, file)
			}
		}
		groups = append(groups, codexUsageFileGroup{Dir: source.Dir, Files: filtered})
	}
	return groups
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
