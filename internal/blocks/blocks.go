// Package blocks implements Claude Code session block detection, burn rate calculation, and token limit projection.
package blocks

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/archhaeondlg/aiusage/internal/dateutil"
	"github.com/archhaeondlg/aiusage/internal/output"
	"github.com/archhaeondlg/aiusage/internal/types"
)

const (
	defaultTokenLimit     = 500_000
	defaultSessionHours   = 5.0
	warningThreshold      = 0.8
)

// BlockOptions configures block analysis.
type BlockOptions struct {
	TokenLimit    string
	SessionLength float64
	Active        bool
	JSON          bool
	NoColor       bool
	Color         string
}

// BuildBlocks groups loaded entries into session blocks.
func BuildBlocks(entries []*types.LoadedEntry, opts BlockOptions) ([]*types.SessionBlock, error) {
	if len(entries) == 0 {
		return nil, nil
	}

	sessionHours := opts.SessionLength
	if sessionHours <= 0 {
		sessionHours = defaultSessionHours
	}

	// Sort entries by timestamp.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.Before(entries[j].Timestamp)
	})

	limit := parseTokenLimit(opts.TokenLimit)
	blocks := identifyBlocks(entries, time.Duration(sessionHours*float64(time.Hour)))
	applyTokenLimits(blocks, limit)

	return blocks, nil
}

func identifyBlocks(entries []*types.LoadedEntry, sessionDuration time.Duration) []*types.SessionBlock {
	if len(entries) == 0 {
		return nil
	}

	var blocks []*types.SessionBlock
	var currentBlock *types.SessionBlock

	for _, entry := range entries {
		if currentBlock == nil {
			currentBlock = newBlock(entry)
			continue
		}

		// Check if this entry belongs in the current block based on time gap.
		gap := entry.Timestamp.Sub(currentBlock.EndTime)
		if gap > sessionDuration {
			// Entry starts a new block.
			currentBlock.ActualEndTime = &currentBlock.EndTime
			blocks = append(blocks, currentBlock)
			_ = currentBlock // prevent misuse
			// Check for gap block.
			if gap > 2*sessionDuration {
				gapBlock := &types.SessionBlock{
					ID:        fmt.Sprintf("gap-%s", currentBlock.EndTime.Format(dateutil.RFC3339Z)),
					StartTime: currentBlock.EndTime,
					EndTime:   entry.Timestamp,
					IsGap:     true,
					Models:    nil,
				}
				blocks = append(blocks, gapBlock)
			}
			currentBlock = newBlock(entry)
			continue
		}

		// Extend current block.
		currentBlock.EndTime = entry.Timestamp
		currentBlock.Entries = append(currentBlock.Entries, entry)
		currentBlock.TokenCounts.AddUsage(entry.Data.Message.Usage)
		currentBlock.TokenCounts.ExtraTotalTokens += entry.ExtraTotalTokens
		currentBlock.CostUSD += entry.Cost
		if entry.Model != nil {
			currentBlock.Models = appendDistinct(currentBlock.Models, *entry.Model)
		}
	}

	if currentBlock != nil {
		currentBlock.IsActive = true
		currentBlock.ActualEndTime = nil
		blocks = append(blocks, currentBlock)
	}

	return blocks
}

func newBlock(entry *types.LoadedEntry) *types.SessionBlock {
	b := &types.SessionBlock{
		ID:        entry.Timestamp.Format(dateutil.RFC3339Z),
		StartTime: entry.Timestamp,
		EndTime:   entry.Timestamp,
		Entries:   []*types.LoadedEntry{entry},
	}
	b.TokenCounts.AddUsage(entry.Data.Message.Usage)
	b.TokenCounts.ExtraTotalTokens += entry.ExtraTotalTokens
	b.CostUSD = entry.Cost
	if entry.Model != nil {
		b.Models = append(b.Models, *entry.Model)
	}
	return b
}

func applyTokenLimits(blocks []*types.SessionBlock, limit uint64) {
	// Mark blocks that exceed the warning threshold.
}

// CalculateBurnRate computes the tokens-per-minute and cost-per-hour rates.
func CalculateBurnRate(block *types.SessionBlock) *types.BurnRate {
	duration := block.EndTime.Sub(block.StartTime)
	minutes := duration.Minutes()
	if minutes < 0.1 {
		minutes = 0.1
	}

	tpm := float64(block.TokenCounts.Total()) / minutes
	cph := (block.CostUSD / minutes) * 60

	return &types.BurnRate{
		TokensPerMinute:          tpm,
		TokensPerMinuteIndicator: tpm,
		CostPerHour:              cph,
	}
}

// ProjectLimit estimates when the token limit will be reached.
func ProjectLimit(block *types.SessionBlock, limit uint64) *types.Projection {
	rate := CalculateBurnRate(block)
	if rate.TokensPerMinute <= 0 {
		return &types.Projection{RemainingMinutes: 0}
	}
	total := block.TokenCounts.Total()
	if total >= limit {
		return &types.Projection{
			TotalTokens:      total,
			TotalCost:        block.CostUSD,
			RemainingMinutes: 0,
		}
	}
	remaining := limit - total
	remainingMins := uint64(float64(remaining) / rate.TokensPerMinute)
	return &types.Projection{
		TotalTokens:      block.TokenCounts.Total(),
		TotalCost:        block.CostUSD,
		RemainingMinutes: remainingMins,
	}
}

// PrintBlocksTable prints a formatted blocks table.
func PrintBlocksTable(blocks []*types.SessionBlock, opts BlockOptions) {
	if len(blocks) == 0 {
		fmt.Println("No blocks found.")
		return
	}

	limit := parseTokenLimit(opts.TokenLimit)

	noColor := opts.NoColor || opts.Color == "never"
	style := output.Style{Enabled: !noColor, NoColor: noColor}

	headers := []string{"Block Start", "Tokens", "Cost (USD)", "Models", "Burn Rate", "Status"}
	aligns := []output.Align{
		output.AlignLeft, output.AlignRight, output.AlignRight,
		output.AlignLeft, output.AlignRight, output.AlignLeft,
	}

	tbl := output.NewTable(headers, aligns, style)

	for _, block := range blocks {
		status := "Active"
		if !block.IsActive {
			status = "Complete"
		}
		if block.IsGap {
			tbl.Push([]string{"—", "—", "—", "—", "—", "Gap"})
			continue
		}
		rate := CalculateBurnRate(block)
		burnStr := fmt.Sprintf("%.0f t/m", rate.TokensPerMinute)

		models := "—"
		if len(block.Models) > 0 {
			models = output.FormatModelsMultiline(block.Models)
		}
		tbl.Push([]string{
			block.StartTime.Format("2006-01-02 15:04"),
			output.FormatNumber(block.TokenCounts.Total()),
			output.FormatCurrency(block.CostUSD),
			models,
			burnStr,
			status,
		})
	}

	// Show projection for active block.
	for _, block := range blocks {
		if block.IsActive && !block.IsGap {
			proj := ProjectLimit(block, limit)
			minLeft := proj.RemainingMinutes
			fmt.Printf("\nActive block (limit: %s tokens): ~%d min remaining\n",
				output.FormatNumber(limit), minLeft)
			break
		}
	}

	tbl.Print()
}

// BlocksJSON builds a JSON representation of blocks.
func BlocksJSON(blocks []*types.SessionBlock, opts BlockOptions) map[string]any {
	limit := parseTokenLimit(opts.TokenLimit)
	var data []map[string]any
	for _, block := range blocks {
		rate := CalculateBurnRate(block)
		proj := ProjectLimit(block, limit)
		data = append(data, map[string]any{
			"id":              block.ID,
			"startTime":       block.StartTime.Format(dateutil.RFC3339Z),
			"endTime":         block.EndTime.Format(dateutil.RFC3339Z),
			"isActive":        block.IsActive,
			"isGap":           block.IsGap,
			"inputTokens":     block.TokenCounts.InputTokens,
			"outputTokens":    block.TokenCounts.OutputTokens,
			"cacheCreationTokens": block.TokenCounts.CacheCreation,
			"cacheReadTokens": block.TokenCounts.CacheRead,
			"totalTokens":     block.TokenCounts.Total(),
			"costUSD":         block.CostUSD,
			"models":          block.Models,
			"burnRate":        rate,
			"projection":      proj,
		})
	}
	return map[string]any{
		"blocks":     data,
		"tokenLimit": limit,
	}
}

// StatuslineOutput produces the short format for Claude Code's statusline hook.
func StatuslineOutput(entries []*types.LoadedEntry, opts BlockOptions) string {
	if len(entries) == 0 {
		return "aiusage: no data"
	}
	blocks, _ := BuildBlocks(entries, opts)
	if len(blocks) == 0 {
		return "aiusage: no blocks"
	}

	// Find active block.
	for _, block := range blocks {
		if block.IsActive {
			dur := time.Since(block.StartTime)
			durStr := formatDuration(dur)
			return fmt.Sprintf("⚡ %s │ $%.2f │ %s │ %s",
				strings.Join(block.Models, ","),
				block.CostUSD,
				output.FormatNumber(block.TokenCounts.Total()),
				durStr,
			)
		}
	}
	return "aiusage: idle"
}

func parseTokenLimit(s string) uint64 {
	if s == "" {
		return defaultTokenLimit
	}
	var n uint64
	fmt.Sscanf(s, "%d", &n)
	if n == 0 {
		return defaultTokenLimit
	}
	return n
}

func appendDistinct(s []string, v string) []string {
	for _, x := range s {
		if x == v {
			return s
		}
	}
	return append(s, v)
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}
