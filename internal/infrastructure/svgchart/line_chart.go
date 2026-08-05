package svgchart

import (
	"fmt"
	"strings"

	"table-tennis-backend/internal/domain/chart"
)

const (
	lineChartWidth  = 600.0
	lineChartHeight = 200.0
	lineSidePad     = 12.0
	lineTopPad      = 20.0
	lineBottomPad   = 24.0
)

func (g *SVGGenerator) LineChart(points []chart.Point) (string, error) {
	if len(points) < 2 {
		return "", nil
	}

	minY, maxY := points[0].Y, points[0].Y
	for _, p := range points {
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}
	if minY == maxY {
		minY--
		maxY++
	} else {
		pad := (maxY - minY) * 0.1
		minY -= pad
		maxY += pad
	}

	plotW := lineChartWidth - 2*lineSidePad
	plotH := lineChartHeight - lineTopPad - lineBottomPad

	scaleX := func(i int) float64 {
		if len(points) == 1 {
			return lineSidePad + plotW/2
		}
		return lineSidePad + (plotW/float64(len(points)-1))*float64(i)
	}
	scaleY := func(y float64) float64 {
		return lineTopPad + plotH - ((y-minY)/(maxY-minY))*plotH
	}

	var coords []string
	var marks strings.Builder
	for i, p := range points {
		x, y := scaleX(i), scaleY(p.Y)
		coords = append(coords, fmt.Sprintf("%.2f,%.2f", x, y))
		fmt.Fprintf(&marks, `<circle cx="%.2f" cy="%.2f" r="3.5" fill="currentColor"/>`, x, y)

		if i == 0 || i == len(points)-1 {
			anchor := "start"
			if i == len(points)-1 {
				anchor = "end"
			}
			fmt.Fprintf(&marks, `<text x="%.2f" y="%.2f" text-anchor="%s" font-size="10" font-weight="bold" fill="currentColor">%s</text>`,
				x, y-8, anchor, esc(formatValue(p.Y)))
		}
	}

	polyline := fmt.Sprintf(`<polyline points="%s" fill="none" stroke="currentColor" stroke-width="2" stroke-linejoin="round" stroke-linecap="round"/>`,
		strings.Join(coords, " "))

	first, last := points[0], points[len(points)-1]
	title := esc(fmt.Sprintf("Trend from %s to %s, %s to %s",
		labelOrDash(first.Label), labelOrDash(last.Label), formatValue(first.Y), formatValue(last.Y)))

	svg := fmt.Sprintf(
		`<svg viewBox="0 0 %g %g" preserveAspectRatio="xMidYMid meet" class="w-full h-auto" role="img"><title>%s</title>%s%s</svg>`,
		lineChartWidth, lineChartHeight, title, polyline, marks.String(),
	)
	return svg, nil
}

func labelOrDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
