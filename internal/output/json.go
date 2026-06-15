package output

import (
	"encoding/json"
	"io"
	"math"
	"os"
	"os/exec"

	"github.com/archhaeondlg/aiusage/internal/types"
)

// PrintJSONOrJQ outputs JSON, optionally piping through jq.
func PrintJSONOrJQ(value any, jq string, noCost bool) error {
	if noCost {
		stripCostJSON(value)
	}
	// If jq filter specified, pipe through jq.
	if jq != "" {
		return pipeToJQ(value, jq)
	}
	// Otherwise pretty-print to stdout.
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		return err
	}
	return nil
}

func pipeToJQ(value any, filter string) error {
	cmd := exec.Command("jq", filter)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	go func() {
		defer stdin.Close()
		data, _ := json.Marshal(value)
		stdin.Write(data)
		if data[len(data)-1] != '\n' {
			stdin.Write([]byte("\n"))
		}
	}()

	return cmd.Run()
}

// SummaryJSON builds a JSON object from a UsageSummary.
func SummaryJSON(row *types.UsageSummary) map[string]any {
	m := map[string]any{
		"inputTokens":       row.InputTokens,
		"outputTokens":      row.OutputTokens,
		"cacheCreationTokens": row.CacheCreation,
		"cacheReadTokens":   row.CacheRead,
		"totalTokens":       row.TotalTokens(),
		"totalCost":         jsonFloat(row.TotalCost),
		"modelsUsed":        row.ModelsUsed,
		"modelBreakdowns":   row.ModelBreakdowns,
	}
	if row.Date != nil {
		m["date"] = *row.Date
	}
	if row.Month != nil {
		m["month"] = *row.Month
	}
	if row.Week != nil {
		m["week"] = *row.Week
	}
	if row.Project != nil {
		m["project"] = *row.Project
	}
	if row.Credits != nil {
		m["credits"] = *row.Credits
	}
	return m
}

// SessionSummaryJSON builds a JSON object for session views.
func SessionSummaryJSON(row *types.UsageSummary) map[string]any {
	m := map[string]any{
		"sessionId":          row.SessionID,
		"inputTokens":        row.InputTokens,
		"outputTokens":       row.OutputTokens,
		"cacheCreationTokens": row.CacheCreation,
		"cacheReadTokens":    row.CacheRead,
		"totalTokens":        row.TotalTokens(),
		"totalCost":          jsonFloat(row.TotalCost),
		"lastActivity":       row.LastActivity,
		"firstActivity":      row.FirstActivity,
		"modelsUsed":         row.ModelsUsed,
		"modelBreakdowns":    row.ModelBreakdowns,
		"projectPath":        row.ProjectPath,
	}
	if row.Credits != nil {
		m["credits"] = *row.Credits
	}
	return m
}

// TotalsJSON builds aggregate totals from a slice of summaries.
func TotalsJSON(rows []*types.UsageSummary) map[string]any {
	var input, output, cacheCreate, cacheRead, extra uint64
	var totalCost, credits float64

	for _, row := range rows {
		input += row.InputTokens
		output += row.OutputTokens
		cacheCreate += row.CacheCreation
		cacheRead += row.CacheRead
		extra += row.ExtraTotal
		totalCost += row.TotalCost
		if row.Credits != nil {
			credits += *row.Credits
		}
	}

	m := map[string]any{
		"inputTokens":        input,
		"outputTokens":       output,
		"cacheCreationTokens": cacheCreate,
		"cacheReadTokens":    cacheRead,
		"totalTokens":        input + output + cacheCreate + cacheRead + extra,
		"totalCost":          jsonFloat(totalCost),
	}
	if credits > 0 {
		m["credits"] = credits
	}
	return m
}

// GroupProjectOutput groups summaries by project name.
func GroupProjectOutput(rows []*types.UsageSummary) map[string][]map[string]any {
	projects := make(map[string][]map[string]any)
	for _, row := range rows {
		proj := "unknown"
		if row.Project != nil {
			proj = *row.Project
		}
		projects[proj] = append(projects[proj], SummaryJSON(row))
	}
	return projects
}

// jsonFloat truncates a float64 to 9 decimal places, avoiding integer overflow risk.
func jsonFloat(value float64) any {
	if value == 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0.0
	}
	return math.Trunc(value*1e9) / 1e9
}

// stripCostJSON recursively removes cost fields from a value.
func stripCostJSON(v any) {
	stripCostJSONDepth(v, 0, 64)
}

func stripCostJSONDepth(v any, depth, maxDepth int) {
	if depth >= maxDepth {
		return
	}
	switch val := v.(type) {
	case map[string]any:
		delete(val, "totalCost")
		delete(val, "costUSD")
		delete(val, "cost")
		for _, child := range val {
			stripCostJSONDepth(child, depth+1, maxDepth)
		}
	case []any:
		for _, child := range val {
			stripCostJSONDepth(child, depth+1, maxDepth)
		}
	}
}

// WriteJSON marshals a value as indented JSON to a writer.
func WriteJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
