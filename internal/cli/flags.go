package cli

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/archhaeondlg/aiusage/internal/types"
)

// RunOptions holds all parsed CLI options for a report run.
type RunOptions struct {
	// Agent and report kind.
	Agent string
	Kind  string

	// Shared options.
	JSON          bool
	Compact       bool
	Breakdown     bool
	NoColor       bool
	SingleThread  bool
	Verbose       int
	Color         string
	Timezone      string
	Since         string
	Until         string
	Order         string
	JQ            string
	Project       string
	ProjectAliases string
	StartOfWeek   string

	// Blocks-specific.
	TokenLimit    string
	SessionLength float64
	Active        bool

	// Pricing overrides from config.
	PricingOverrides map[string]*PricingOverride
}

// PricingOverride is a local alias for config pricing overrides.
type PricingOverride struct {
	InputCostPerToken              *float64
	OutputCostPerToken             *float64
	CacheCreationInputTokenCost    *float64
	CacheReadInputTokenCost        *float64
	InputCostPerTokenAbove200K     *float64
	OutputCostPerTokenAbove200K    *float64
	CacheCreationAbove200K         *float64
	CacheReadAbove200K             *float64
	FastMultiplier                 *float64
	MaxInputTokens                 *uint64
}

// parseRunOptions extracts all run options from the cobra command.
func parseRunOptions(cmd *cobra.Command) (*RunOptions, error) {
	opts := &RunOptions{}

	// Bind cobra flags to viper.
	v := viper.New()
	v.BindPFlags(cmd.Flags())
	v.BindPFlags(cmd.PersistentFlags())

	// Load config file if present.
	if cfg, err := loadConfigFile(); err == nil {
		v.MergeConfigMap(cfg)
	}

	// Extract common options.
	opts.JSON = getBool(v, "json")
	opts.Compact = getBool(v, "compact")
	opts.Breakdown = getBool(v, "breakdown")
	opts.Verbose = getInt(v, "verbose")
	opts.NoColor = getBool(v, "no-color")
	opts.SingleThread = getBool(v, "single-thread")
	opts.Color = getString(v, "color")
	opts.Timezone = getString(v, "timezone")
	opts.Since = normalizeDateBound(getString(v, "since"))
	opts.Until = normalizeDateBound(getString(v, "until"))
	opts.Order = getString(v, "order")
	opts.JQ = getString(v, "jq")
	opts.Project = getString(v, "project")
	opts.ProjectAliases = getString(v, "project-aliases")
	opts.StartOfWeek = getString(v, "start-of-week")

	// Blocks options.
	opts.TokenLimit = getString(v, "token-limit")
	opts.SessionLength, _ = getFloat64(v, "session-length")
	opts.Active = getBool(v, "active")

	return opts, nil
}

// SortOrder converts the order string to SortOrder enum.
func (o *RunOptions) SortOrder() types.SortOrder {
	switch strings.ToLower(o.Order) {
	case "desc":
		return types.SortDesc
	default:
		return types.SortAsc
	}
}

// WeekStartDay converts StartOfWeek string to WeekDay enum.
func (o *RunOptions) WeekStartDay() types.WeekDay {
	switch strings.ToLower(o.StartOfWeek) {
	case "sunday":
		return types.WeekSunday
	case "monday":
		return types.WeekMonday
	case "tuesday":
		return types.WeekTuesday
	case "wednesday":
		return types.WeekWednesday
	case "thursday":
		return types.WeekThursday
	case "friday":
		return types.WeekFriday
	case "saturday":
		return types.WeekSaturday
	default:
		return types.WeekMonday
	}
}

func getString(v *viper.Viper, key string) string {
	return v.GetString(key)
}

func getBool(v *viper.Viper, key string) bool {
	return v.GetBool(key)
}

func getInt(v *viper.Viper, key string) int {
	return v.GetInt(key)
}

func getFloat64(v *viper.Viper, key string) (float64, bool) {
	val := v.Get(key)
	if val == nil {
		return 0, false
	}
	switch v := val.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	default:
		return 0, false
	}
}

func normalizeDateBound(s string) string {
	if s == "" {
		return ""
	}
	return strings.ReplaceAll(s, "-", "")
}
