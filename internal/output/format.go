// Package output provides terminal table rendering and JSON output for usage reports.
package output

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

// FormatNumber formats large numbers with B/M/K suffix, small numbers with commas.
// < 10000: "1,234"  ≥ 10000: "12.3K"  ≥ 1M: "1.2M"  ≥ 1B: "1.2B".
func FormatNumber(value uint64) string {
	switch {
	case value >= 1_000_000_000:
		return formatScaled(float64(value)/1_000_000_000, "B")
	case value >= 1_000_000:
		return formatScaled(float64(value)/1_000_000, "M")
	case value >= 10_000:
		return formatScaled(float64(value)/1_000, "K")
	default:
		s := fmt.Sprintf("%d", value)
		var result strings.Builder
		result.Grow(len(s) + len(s)/3)
		for i, ch := range s {
			if i > 0 && (len(s)-i)%3 == 0 {
				result.WriteByte(',')
			}
			result.WriteRune(ch)
		}
		return result.String()
	}
}

func formatScaled(v float64, suffix string) string {
	var s string
	switch {
	case v >= 100:
		s = fmt.Sprintf("%.0f%s", v, suffix)
	case v >= 10:
		s = fmt.Sprintf("%.1f%s", v, suffix)
	default:
		s = fmt.Sprintf("%.2f%s", v, suffix)
	}
	// Remove trailing ".0" if present before the suffix.
	s = strings.Replace(s, ".0"+suffix, suffix, 1)
	return s
}

// FormatCurrency formats a float as USD: 0.25 → "$0.25".
func FormatCurrency(value float64) string {
	return fmt.Sprintf("$%.2f", value)
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// ANSI terminal styling
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

// Color represents an ANSI terminal color.
type Color int

const (
	ColorNone Color = iota
	ColorYellow
	ColorBlue
	ColorGrey
	ColorGreen
	ColorRed
)

// Align controls column text alignment.
type Align int

const (
	AlignLeft Align = iota
	AlignRight
	AlignCenter
)

// Style controls terminal output styling.
type Style struct {
	Enabled bool
	NoColor bool
}

// Colorize wraps text in ANSI color codes when color is enabled.
func (s Style) Colorize(text string, c Color) string {
	if s.NoColor || !s.Enabled {
		return text
	}
	code := ansiColorCode(c)
	if code == "" {
		return text
	}
	return fmt.Sprintf("\x1b[%sm%s\x1b[0m", code, text)
}

func ansiColorCode(c Color) string {
	switch c {
	case ColorYellow:
		return "33"
	case ColorBlue:
		return "34"
	case ColorGrey:
		return "90"
	case ColorGreen:
		return "32"
	case ColorRed:
		return "31"
	default:
		return ""
	}
}

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// visibleLen returns the display width of a string after stripping ANSI codes.
func visibleLen(s string) int {
	return utf8.RuneCountInString(ansiRE.ReplaceAllString(s, ""))
}

// padVisible pads a string to the given visual width, preserving ANSI codes.
func padVisible(s string, width int, align Align) string {
	visLen := visibleLen(s)
	if visLen > width {
		return truncateVisible(s, width)
	}
	pad := width - visLen
	switch align {
	case AlignRight:
		return strings.Repeat(" ", pad) + s
	case AlignCenter:
		left := pad / 2
		right := pad - left
		return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
	default: // AlignLeft
		return s + strings.Repeat(" ", pad)
	}
}

// truncateVisible truncates a string with ANSI codes to the given visual width.
func truncateVisible(s string, visWidth int) string {
	var buf strings.Builder
	visible := 0
	i := 0
	hasANSI := false
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			end := i + 2
			for end < len(s) && s[end] != 'm' {
				end++
			}
			if end < len(s) {
				end++ // include 'm'
			}
			buf.WriteString(s[i:end])
			i = end
			hasANSI = true
			continue
		}
		if visible >= visWidth {
			break
		}
		_, size := utf8.DecodeRuneInString(s[i:])
		buf.WriteString(s[i : i+size])
		visible++
		i += size
	}
	if hasANSI {
		buf.WriteString("\x1b[0m")
	}
	return buf.String()
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Table rendering
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

// SimpleTable is a responsive terminal table supporting compact and wide layouts.
type SimpleTable struct {
	headers        []string
	aligns         []Align
	rows           [][]string
	style          Style
	termWidth      int
	compactWidth   int
	dateCompaction bool
}

// NewTable creates a new SimpleTable.
func NewTable(headers []string, aligns []Align, style Style) *SimpleTable {
	return &SimpleTable{
		headers:      headers,
		aligns:       aligns,
		style:        style,
		termWidth:    120,
		compactWidth: 100,
	}
}

// WithTerminalWidth sets the terminal width for layout decisions.
func (t *SimpleTable) WithTerminalWidth(width int) *SimpleTable {
	t.termWidth = width
	return t
}

// WithCompactWidth sets the threshold below which compact mode activates.
func (t *SimpleTable) WithCompactWidth(width int) *SimpleTable {
	t.compactWidth = width
	return t
}

// WithDateCompaction enables date shortening in narrow tables.
func (t *SimpleTable) WithDateCompaction(on bool) *SimpleTable {
	t.dateCompaction = on
	return t
}

// ColumnCount returns the number of columns.
func (t *SimpleTable) ColumnCount() int {
	return len(t.headers)
}

// Push appends a row of string values.
func (t *SimpleTable) Push(values []string) {
	t.rows = append(t.rows, values)
}

// Separator adds a visual separator row.
func (t *SimpleTable) Separator() {
	sep := make([]string, len(t.headers))
	for i := range sep {
		sep[i] = "-"
	}
	t.rows = append(t.rows, sep)
}

// Render builds the full table as a string.
func (t *SimpleTable) Render() string {
	if len(t.rows) == 0 {
		return ""
	}

	compact := t.termWidth < t.compactWidth

	// Split all cells into sub-lines and calculate max column widths (visual).
	rowLines := make([][][]string, len(t.rows))
	colWidths := make([]int, len(t.headers))
	for i, h := range t.headers {
		colWidths[i] = visibleLen(h)
	}
	for ri, row := range t.rows {
		rowLines[ri] = make([][]string, len(row))
		for ci, cell := range row {
			lines := strings.Split(cell, "\n")
			rowLines[ri][ci] = lines
			for _, line := range lines {
				if ci < len(colWidths) && visibleLen(line) > colWidths[ci] {
					colWidths[ci] = visibleLen(line)
				}
			}
		}
	}

	// Cap column widths in compact mode.
	if compact {
		maxWidth := (t.termWidth - (len(colWidths)-1)*3 - 4) / len(colWidths)
		if maxWidth < 8 {
			maxWidth = 8
		}
		for i := range colWidths {
			if colWidths[i] > maxWidth {
				colWidths[i] = maxWidth
			}
		}
	}

	var buf strings.Builder

	// Header.
	for i, h := range t.headers {
		if i > 0 {
			buf.WriteString(" │ ")
		}
		buf.WriteString(padVisible(h, colWidths[i], AlignLeft))
	}
	buf.WriteByte('\n')

	// Header separator.
	for i, w := range colWidths {
		if i > 0 {
			buf.WriteString("─┼─")
		}
		buf.WriteString(strings.Repeat("─", w))
	}
	buf.WriteByte('\n')

	// Data rows.
	for _, cells := range rowLines {
		maxLines := 0
		for _, lines := range cells {
			if len(lines) > maxLines {
				maxLines = len(lines)
			}
		}
		for li := 0; li < maxLines; li++ {
			for ci := 0; ci < len(t.headers); ci++ {
				if ci > 0 {
					buf.WriteString(" │ ")
				}
				line := ""
				if ci < len(cells) && li < len(cells[ci]) {
					line = cells[ci][li]
				}
				// Expand separator dash to full column width.
				if line == "-" {
					line = strings.Repeat("-", colWidths[ci])
				}
				align := AlignLeft
				if ci < len(t.aligns) {
					align = t.aligns[ci]
				}
				s := padVisible(line, colWidths[ci], align)
				buf.WriteString(s)
			}
			buf.WriteByte('\n')
		}
	}

	return buf.String()
}

// Print renders and prints the table to stdout.
func (t *SimpleTable) Print() {
	fmt.Print(t.Render())
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Box title
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

// PrintBoxTitle prints a centered boxed title.
func PrintBoxTitle(title string, style Style) {
	width := 60
	border := strings.Repeat("═", width)
	fmt.Printf("\n%s\n", style.Colorize(border, ColorBlue))
	fmt.Printf("  %s\n", style.Colorize(title, ColorYellow))
	fmt.Printf("%s\n\n", style.Colorize(border, ColorBlue))
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Model name formatting
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

// ShortModelName truncates common model name prefixes for display.
func ShortModelName(model string) string {
	prefixes := []string{
		"anthropic.", "us.anthropic.", "eu.anthropic.", "global.anthropic.",
		"jp.anthropic.", "au.anthropic.", "anthropic/",
		"openrouter/", "openrouter/anthropic/",
		"vertex_ai/", "bedrock/", "openai/", "azure/",
	}
	result := model
	for {
		changed := false
		for _, p := range prefixes {
			if strings.HasPrefix(result, p) {
				result = result[len(p):]
				changed = true
				break
			}
		}
		if !changed {
			break
		}
	}
	return result
}

// FormatModelsMultiline formats model names as a sorted, deduplicated bullet list.
func FormatModelsMultiline(models []string) string {
	seen := make(map[string]bool)
	var unique []string
	for _, m := range models {
		short := ShortModelName(m)
		if !seen[short] {
			seen[short] = true
			unique = append(unique, short)
		}
	}
	sort.Strings(unique)
	var parts []string
	for _, m := range unique {
		parts = append(parts, "- "+m)
	}
	return strings.Join(parts, "\n")
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Project name formatting
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

// FormatProjectName extracts a short project name from a path or alias map.
func FormatProjectName(path string, aliases map[string]string) string {
	if alias, ok := aliases[path]; ok {
		return alias
	}
	// Also check the normalized path.
	norm := strings.ReplaceAll(path, "\\", "/")
	if alias, ok := aliases[norm]; ok && norm != path {
		return alias
	}
	// Extract last path component.
	path = strings.ReplaceAll(path, "\\", "/")
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}

// ParseProjectAliases parses a key=value,key2=value2 string into a map.
func ParseProjectAliases(raw string) map[string]string {
	if raw == "" {
		return nil
	}
	result := make(map[string]string)
	pairs := strings.Split(raw, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(kv) == 2 {
			result[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return result
}
