package svgchart

import (
	"html"
	"strconv"
)

// esc escapes a label for safe embedding in SVG text/attribute content.
func esc(s string) string {
	return html.EscapeString(s)
}

// formatValue renders a float without a trailing ".0" for whole numbers,
// since most chart values here (counts, Elo ratings) are conceptually integers.
func formatValue(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}
