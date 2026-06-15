package codex

import (
	"fmt"
	"os"
	"sort"

	"github.com/archhaeondlg/aiusage/internal/output"
	"github.com/archhaeondlg/aiusage/internal/pricing"
	"github.com/archhaeondlg/aiusage/internal/types"
)

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// JSON report
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

// ReportFromGroups builds a JSON report from aggregated groups.
func ReportFromGroups(groups map[string]*codexGroupData, kind types.ReportKind, pricing pricing.PricingProvider, speed string) map[string]any {
	var rows []map[string]any
	for _, key := range sortedGroupKeys(groups) {
		group := groups[key]
		rows = append(rows, groupJSON(key, group, kind, pricing, speed))
	}
	return map[string]any{
		rowsKey(kind): rows,
		"totals":      totalsJSON(groups, pricing, speed),
	}
}

func rowsKey(kind types.ReportKind) string {
	switch kind {
	case types.ReportWeekly:
		return "weekly"
	case types.ReportMonthly:
		return "monthly"
	case types.ReportSession:
		return "sessions"
	default:
		return "daily"
	}
}

func periodKey(kind types.ReportKind) string {
	switch kind {
	case types.ReportWeekly:
		return "week"
	case types.ReportMonthly:
		return "month"
	case types.ReportSession:
		return "sessionId"
	default:
		return "date"
	}
}

func groupJSON(period string, group *codexGroupData, kind types.ReportKind, pm pricing.PricingProvider, speed string) map[string]any {
	cost := calculateGroupCost(group, pm, speed)
	input := nonCachedInput(group.InputTokens, group.CachedInputTokens)

	models := make(map[string]any)
	modelKeys := sortedStringKeys(group.Models)
	for _, model := range modelKeys {
		usage := group.Models[model]
		models[model] = map[string]any{
			"inputTokens":          nonCachedInput(usage.InputTokens, usage.CachedInputTokens),
			"cacheCreationTokens":  0,
			"cacheReadTokens":      usage.CachedInputTokens,
			"outputTokens":         usage.OutputTokens,
			"reasoningOutputTokens": usage.ReasoningOutputTokens,
			"totalTokens":          usage.TotalTokens,
			"isFallback":           usage.IsFallback,
		}
	}

	row := map[string]any{
		periodKey(kind):         period,
		"inputTokens":           input,
		"cacheCreationTokens":   0,
		"cacheReadTokens":       group.CachedInputTokens,
		"outputTokens":          group.OutputTokens,
		"reasoningOutputTokens": group.ReasoningOutputTokens,
		"totalTokens":           group.TotalTokens,
		"costUSD":               cost,
		"models":                models,
	}
	if kind == types.ReportSession {
		row["lastActivity"] = group.LastActivity
	}
	return row
}

func nonCachedInput(input, cached uint64) uint64 {
	if input > cached {
		return input - cached
	}
	return 0
}

func totalsJSON(groups map[string]*codexGroupData, pm pricing.PricingProvider, speed string) map[string]any {
	var input, cached, output, reasoning, total uint64
	var cost float64
	for _, g := range groups {
		input += nonCachedInput(g.InputTokens, g.CachedInputTokens)
		cached += g.CachedInputTokens
		output += g.OutputTokens
		reasoning += g.ReasoningOutputTokens
		total += g.TotalTokens
		cost += calculateGroupCost(g, pm, speed)
	}
	return map[string]any{
		"inputTokens":          input,
		"cacheCreationTokens":  0,
		"cacheReadTokens":      cached,
		"outputTokens":         output,
		"reasoningOutputTokens": reasoning,
		"totalTokens":          total,
		"costUSD":              cost,
	}
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Cost calculation (mirrors Rust calculate_codex_model_cost)
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

// CalculateCodexModelCost computes the USD cost for a Codex model's usage.
func CalculateCodexModelCost(model string, usage *codexModelUsageData, pm pricing.PricingProvider, speed string) float64 {
	p := pm.Find(model)
	if p == nil {
		return 0
	}
	nonCached := nonCachedInput(usage.InputTokens, usage.CachedInputTokens)

	multiplier := 1.0
	if speed == "fast" {
		if p.FastMultiplier == 1.0 {
			multiplier = 2.0
		} else {
			multiplier = p.FastMultiplier
		}
	}

	cacheReadRate := p.CacheRead
	if !p.CacheReadExplicit {
		cacheReadRate = p.Input
	}

	return (float64(nonCached)*p.Input +
		float64(usage.CachedInputTokens)*cacheReadRate +
		float64(usage.OutputTokens)*p.Output) * multiplier
}

func calculateGroupCost(group *codexGroupData, pm pricing.PricingProvider, speed string) float64 {
	var cost float64
	for model, usage := range group.Models {
		cs := string(CodexSpeedStandard)
		if speed == "fast" {
			cs = string(CodexSpeedFast)
		}
		cost += CalculateCodexModelCost(model, usage, pm, cs)
	}
	return cost
}


// MissingPricingModels returns models without pricing data.
func MissingPricingModels(groups map[string]*codexGroupData, pm pricing.PricingProvider) []string {
	seen := make(map[string]bool)
	for _, g := range groups {
		for model := range g.Models {
			if pm.Find(model) == nil {
				seen[model] = true
			}
		}
	}
	models := make([]string, 0, len(seen))
	for m := range seen {
		models = append(models, m)
	}
	sort.Strings(models)
	return models
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Table output
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

// PrintCodexTable prints a terminal table from aggregated groups.
func PrintCodexTable(groups map[string]*codexGroupData, kind types.ReportKind, pm pricing.PricingProvider, speed string, compact bool) {
	if len(groups) == 0 {
		fmt.Println("No Codex usage data found.")
		return
	}

	firstCol := "Date"
	switch kind {
	case types.ReportWeekly:
		firstCol = "Week"
	case types.ReportMonthly:
		firstCol = "Month"
	case types.ReportSession:
		firstCol = "Session"
	}

	style := output.Style{Enabled: true, NoColor: false}
	kindName := string(kind[0]-32) + string(kind[1:]) // Title case.
	output.PrintBoxTitle("Codex Token Usage Report - "+kindName, style)

	headers := []string{firstCol, "Models", "Input", "Output", "Reasoning", "Cache Read", "Total Tokens", "Cost (USD)"}
	aligns := []output.Align{
		output.AlignLeft, output.AlignLeft,
		output.AlignRight, output.AlignRight, output.AlignRight,
		output.AlignRight, output.AlignRight, output.AlignRight,
	}

	tbl := output.NewTable(headers, aligns, style)

	var totalInput, totalCached, totalOutput, totalReasoning, totalTokens uint64
	var totalCost float64

	for _, key := range sortedGroupKeys(groups) {
		g := groups[key]
		input := nonCachedInput(g.InputTokens, g.CachedInputTokens)
		cost := calculateGroupCost(g, pm, speed)

		totalInput += input
		totalCached += g.CachedInputTokens
		totalOutput += g.OutputTokens
		totalReasoning += g.ReasoningOutputTokens
		totalTokens += g.TotalTokens
		totalCost += cost

		models := output.FormatModelsMultiline(sortedStringKeys(g.Models))

		tbl.Push([]string{
			key,
			models,
			output.FormatNumber(input),
			output.FormatNumber(g.OutputTokens),
			output.FormatNumber(g.ReasoningOutputTokens),
			output.FormatNumber(g.CachedInputTokens),
			output.FormatNumber(g.TotalTokens),
			output.FormatCurrency(cost),
		})
	}

	tbl.Separator()
	tbl.Push([]string{
		style.Colorize("Total", output.ColorYellow),
		"",
		style.Colorize(output.FormatNumber(totalInput), output.ColorYellow),
		style.Colorize(output.FormatNumber(totalOutput), output.ColorYellow),
		style.Colorize(output.FormatNumber(totalReasoning), output.ColorYellow),
		style.Colorize(output.FormatNumber(totalCached), output.ColorYellow),
		style.Colorize(output.FormatNumber(totalTokens), output.ColorYellow),
		style.Colorize(output.FormatCurrency(totalCost), output.ColorYellow),
	})
	tbl.Print()

	missing := MissingPricingModels(groups, pm)
	for _, m := range missing {
		fmt.Fprintf(osStderr(), "WARN  Missing pricing for %s; cost excludes this model.\n", m)
	}
}

func sortedStringKeys(m map[string]*codexModelUsageData) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func osStderr() *os.File { return os.Stderr }
