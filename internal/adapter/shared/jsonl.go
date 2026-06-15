// Package shared provides common utilities for adapter implementations.
package shared

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// FindJSONLFiles recursively discovers all .jsonl files under the given paths.
func FindJSONLFiles(paths []string) []string {
	var files []string
	for _, root := range paths {
		filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				slog.Warn("walk error", "path", path, "err", err)
				return nil
			}
			if !d.IsDir() && strings.HasSuffix(d.Name(), ".jsonl") {
				files = append(files, path)
			}
			return nil
		})
	}
	return files
}

// ReadJSONLLines reads a JSONL file and calls the handler for each parsed line.
func ReadJSONLLines(path string, handler func([]byte) error) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	for len(data) > 0 {
		nl := bytes.IndexByte(data, '\n')
		var line []byte
		if nl < 0 {
			line = data
			data = nil
		} else {
			line = data[:nl]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			data = data[nl+1:]
		}
		if len(line) == 0 {
			continue
		}
		if err := handler(line); err != nil {
			return err
		}
	}
	return nil
}

// ParseJSONLFile reads and parses all JSONL entries from a file into the given type.
func ParseJSONLFile[T any](path string) ([]T, error) {
	var results []T
	err := ReadJSONLLines(path, func(line []byte) error {
		var entry T
		if err := json.Unmarshal(line, &entry); err != nil {
			return nil // Skip unparseable lines.
		}
		results = append(results, entry)
		return nil
	})
	return results, err
}


