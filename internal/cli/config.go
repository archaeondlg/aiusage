package cli

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/archhaeondlg/aiusage/internal/pricing"
)

// Config represents the full config.json configuration file.
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

// configPath returns the config file path in the executable's directory.
func configPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "config.json"
	}
	return filepath.Join(filepath.Dir(exe), "config.json")
}

// loadConfigFile loads config.json from the executable's directory.
func loadConfigFile() (map[string]any, error) {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return nil, err
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// loadPricingFromConfig reads config.json and extracts the pricing field.
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
