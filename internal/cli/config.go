package cli

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/archhaeondlg/aiusage/internal/pricing"
)

// Config represents the full aiusage.json configuration file.
type Config struct {
	Defaults  ConfigDefaults            `json:"defaults"`
	Commands  map[string]CommandConfig  `json:"commands"`
}

// ConfigDefaults holds default values for all commands.
type ConfigDefaults struct {
	JSON             bool                          `json:"json"`
	Mode             string                        `json:"mode"`
	Timezone         string                        `json:"timezone"`
	Breakdown        bool                          `json:"breakdown"`
	PricingOverrides map[string]PricingOverride    `json:"pricingOverrides"`
	Pricing          map[string]pricing.PricingEntry `json:"pricing"`
}

// CommandConfig holds per-command configuration overrides.
type CommandConfig struct {
	Instances       bool   `json:"instances"`
	Order           string `json:"order"`
	ProjectAliases  string `json:"projectAliases"`
	Breakdown       bool   `json:"breakdown"`
	StartOfWeek     string `json:"startOfWeek"`
	TokenLimit      string `json:"tokenLimit"`
	SessionLength   float64 `json:"sessionLength"`
	Active          bool   `json:"active"`
	Offline         bool   `json:"offline"`
}

// configPaths returns possible locations for aiusage.json.
func configPaths() []string {
	var paths []string

	// XDG config home.
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		paths = append(paths, filepath.Join(xdg, "aiusage", "aiusage.json"))
	}

	// Default config home.
	home, err := os.UserHomeDir()
	if err == nil {
		paths = append(paths,
			filepath.Join(home, ".config", "aiusage", "aiusage.json"),
			filepath.Join(home, ".aiusage.json"),
		)
	}

	// Current directory.
	paths = append(paths, "aiusage.json")

	return paths
}

// loadConfigFile loads the first found aiusage.json.
func loadConfigFile() (map[string]any, error) {
	for _, path := range configPaths() {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var cfg map[string]any
		if err := json.Unmarshal(data, &cfg); err != nil {
			continue
		}
		return cfg, nil
	}
	return nil, os.ErrNotExist
}

// loadPricingFromConfig reads aiusage.json and extracts the pricing field.
func loadPricingFromConfig() *pricing.PricingMap {
	cfg, err := loadConfigFile()
	if err != nil {
		return pricing.LoadDefaultPricing()
	}

	pricingRaw, ok := cfg["pricing"]
	if !ok {
		return pricing.LoadDefaultPricing()
	}

	// Convert from map[string]any to map[string]pricing.PricingEntry.
	// The config is loaded as map[string]any, so we need to re-marshal.
	data, err := json.Marshal(pricingRaw)
	if err != nil {
		return pricing.LoadDefaultPricing()
	}

	var entries map[string]pricing.PricingEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return pricing.LoadDefaultPricing()
	}

	return pricing.LoadPricing(entries)
}
