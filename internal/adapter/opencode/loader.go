package opencode

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"github.com/archhaeondlg/aiusage/internal/pricing"
	"github.com/archhaeondlg/aiusage/internal/types"
)

// loadEntries discovers and parses all OpenCode usage data.
// Equivalent to Rust load_entries().
func loadEntries(
	pm pricing.PricingProvider,
	timezone string,
) ([]*types.LoadedEntry, error) {
	dirs, err := paths()
	if err != nil {
		return nil, nil
	}

	if pm == nil {
		pm = pricing.LoadDefaultPricing()
	}

	var allEntries []*types.LoadedEntry
	seen := make(map[string]bool)

	for _, dir := range dirs {
		entries, err := loadEntriesFromDirectory(dir, pm)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			id := ""
			if entry.Data.Message.ID != nil {
				id = *entry.Data.Message.ID
			}
			if id != "" {
				if seen[id] {
					continue
				}
				seen[id] = true
			}
			allEntries = append(allEntries, entry)
		}
	}

	return allEntries, nil
}

// loadEntriesFromDirectory loads entries from one OpenCode data directory.
func loadEntriesFromDirectory(
	opencodeDir string,
	pm pricing.PricingProvider,
) ([]*types.LoadedEntry, error) {
	var entries []*types.LoadedEntry
	seen := make(map[string]bool)

	// 1. Load from SQLite database (primary source).
	if dbPath := dbPath(opencodeDir); dbPath != "" {
		dbEntries := loadEntriesFromDatabase(dbPath, pm)
		for _, entry := range dbEntries {
			id := ""
			if entry.Data.Message.ID != nil {
				id = *entry.Data.Message.ID
			}
			if id != "" {
				seen[id] = true
			}
			entries = append(entries, entry)
		}
	}

	// 2. Load from JSON files (fallback, deduplicated against DB).
	messagesDir := filepath.Join(opencodeDir, "storage", "message")
	filepath.WalkDir(messagesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".json" {
			return nil
		}
		entry := readMessageFile(path, pm)
		if entry == nil {
			return nil
		}
		id := ""
		if entry.Data.Message.ID != nil {
			id = *entry.Data.Message.ID
		}
		if id != "" && seen[id] {
			return nil
		}
		if id != "" {
			seen[id] = true
		}
		entries = append(entries, entry)
		return nil
	})

	return entries, nil
}

// loadEntriesFromDatabase reads OpenCode SQLite database.
func loadEntriesFromDatabase(
	dbPath string,
	pm pricing.PricingProvider,
) []*types.LoadedEntry {
	db, err := sql.Open("sqlite", dbPath+"?mode=ro")
	if err != nil {
		return nil
	}
	defer db.Close()

	rows, err := db.Query("SELECT id, session_id, data FROM message")
	if err != nil {
		return nil
	}
	defer rows.Close()

	var entries []*types.LoadedEntry
	for rows.Next() {
		var id, sessionID, data string
		if err := rows.Scan(&id, &sessionID, &data); err != nil {
			continue
		}
		var raw json.RawMessage
		if json.Unmarshal([]byte(data), &raw) != nil {
			continue
		}
		entry := messageToEntry(raw, id, sessionID, nil, pm)
		if entry != nil {
			entries = append(entries, entry)
		}
	}
	return entries
}

// readMessageFile reads a single JSON message file.
func readMessageFile(
	path string,
	pm pricing.PricingProvider,
) *types.LoadedEntry {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var raw json.RawMessage
	if json.Unmarshal(data, &raw) != nil {
		return nil
	}
	return messageToEntry(raw, "", "", nil, pm)
}
