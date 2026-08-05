package svgchart

import (
	"fmt"
	"strings"

	"table-tennis-backend/internal/domain/chart"
)

const (
	barChartWidth   = 600.0
	barChartHeight  = 240.0
	barSidePadding  = 12.0
	barTopPadding   = 24.0
	barBottomLabels = 28.0
	barGap          = 10.0
	barMaxWidth     = 64.0
)

func (g *SVGGenerator) BarChart(items []chart.BarItem) (string, error) {
	if len(items) == 0 {
		return "", nil
	}

	maxVal := 0.0
	for _, it := range items {
		if it.Value > maxVal {
			maxVal = it.Value
		}
	}

	plotW := barChartWidth - 2*barSidePadding
	plotH := barChartHeight - barTopPadding - barBottomLabels
	slotW := plotW / float64(len(items))
	barW := slotW - barGap
	if barW < 4 {
		barW = 4
	}
	if barW > barMaxWidth {
		barW = barMaxWidth
	}

	var bars strings.Builder
	titleParts := make([]string, 0, len(items))
	for i, it := range items {
		slotX := barSidePadding + float64(i)*slotW
		x := slotX + (slotW-barW)/2

		h := 0.0
		if maxVal > 0 {
			h = (it.Value / maxVal) * plotH
		}
		y := barTopPadding + (plotH - h)

		fmt.Fprintf(&bars, `<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" rx="3" fill="currentColor" opacity="0.85"/>`,
			x, y, barW, h)
		fmt.Fprintf(&bars, `<text x="%.2f" y="%.2f" text-anchor="middle" font-size="10" font-weight="bold" fill="currentColor">%s</text>`,
			x+barW/2, y-4, esc(formatValue(it.Value)))
		fmt.Fprintf(&bars, `<text x="%.2f" y="%.2f" text-anchor="middle" font-size="9" fill="currentColor" opacity="0.7">%s</text>`,
			x+barW/2, barChartHeight-8, esc(truncateLabel(it.Label, 12)))

		titleParts = append(titleParts, fmt.Sprintf("%s %s", it.Label, formatValue(it.Value)))
	}

	title := esc(strings.Join(titleParts, ", "))

	svg := fmt.Sprintf(
		`<svg viewBox="0 0 %g %g" preserveAspectRatio="xMidYMid meet" class="w-full h-auto" role="img"><title>%s</title>%s</svg>`,
		barChartWidth, barChartHeight, title, bars.String(),
	)
	return svg, nil
}

func truncateLabel(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}
