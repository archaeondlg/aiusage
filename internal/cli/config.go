package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/archhaeondlg/aiusage/internal/pricing"
)

const configURL = "https://raw.githubusercontent.com/archaeondlg/aiusage/master/config.json"

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
// If the file doesn't exist, downloads a default from GitHub.
func loadConfigFile() (map[string]any, error) {
	data, err := os.ReadFile(configPath())
	if err == nil {
		var cfg map[string]any
		if err := json.Unmarshal(data, &cfg); err == nil {
			return cfg, nil
		}
	}
	// Not found locally — download from GitHub.
	fmt.Fprintln(os.Stderr, "→ Downloading default config.json from GitHub...")
	if err := downloadConfig(); err != nil {
		return nil, fmt.Errorf("download config: %w", err)
	}
	fmt.Fprintln(os.Stderr, "  config.json saved")
	data, err = os.ReadFile(configPath())
	if err != nil {
		return nil, err
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// downloadConfig fetches the default config.json from GitHub.
func downloadConfig() error {
	resp, err := http.Get(configURL)
	if err != nil {
		return fmt.Errorf("fetch %s: %w", configURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), data, 0644)
}

// UpdatePricingFromGitHub fetches the latest config.json from GitHub
// and merges only the pricing section into the local config.
func UpdatePricingFromGitHub() error {
	resp, err := http.Get(configURL)
	if err != nil {
		return fmt.Errorf("fetch pricing: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var remoteCfg map[string]any
	if err := json.Unmarshal(data, &remoteCfg); err != nil {
		return fmt.Errorf("parse remote config: %w", err)
	}
	remotePricing, ok := remoteCfg["pricing"]
	if !ok {
		return fmt.Errorf("no pricing field in remote config")
	}

	// Read local config (or start fresh).
	localCfg := map[string]any{}
	if localData, err := os.ReadFile(configPath()); err == nil {
		json.Unmarshal(localData, &localCfg)
	}

	// Update only the pricing field.
	localCfg["pricing"] = remotePricing

	out, err := json.MarshalIndent(localCfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), out, 0644)
}

// loadPricingFromConfig reads config.json and extracts the pricing field.
func loadPricingFromConfig() pricing.PricingProvider {
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

	return pricing.NewCachedProvider(pricing.LoadPricing(entries))
}
