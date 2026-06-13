// Package shared provides common utilities for adapter implementations.
package shared

import (
	"bufio"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/archhaeondlg/aiusage/internal/types"
)

// FindJSONLFiles recursively discovers all .jsonl files under the given paths.
func FindJSONLFiles(paths []string) []string {
	var files []string
	for _, root := range paths {
		filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
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
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		if err := handler(line); err != nil {
			return err
		}
	}
	return scanner.Err()
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

// SimpleLoader provides a basic LoadEntries implementation for adapters
// that read JSONL files from well-known paths and parse them into entries.
type SimpleLoader struct {
	pathsFn     func() ([]string, error)
	parserFn    func(path string, entry []byte) (*types.LoadedEntry, error)
}

// NewSimpleLoader creates a SimpleLoader.
func NewSimpleLoader(
	pathsFn func() ([]string, error),
	parserFn func(path string, entry []byte) (*types.LoadedEntry, error),
) *SimpleLoader {
	return &SimpleLoader{pathsFn: pathsFn, parserFn: parserFn}
}
