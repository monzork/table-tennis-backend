package chart

import "time"

// BarItem is a single labeled value in a bar chart.
type BarItem struct {
	Label string
	Value float64
}

// Point is a single (time, value) sample in a line chart.
type Point struct {
	X     time.Time
	Y     float64
	Label string
}

// Generator renders chart data into markup. Implementations decide the
// concrete visual format (e.g. inline SVG); callers only depend on this
// interface, never on a specific rendering technology.
type Generator interface {
	// BarChart renders items as a bar chart. Returns "" for an empty input.
	BarChart(items []BarItem) (string, error)
	// LineChart renders points as a line chart. Returns "" for fewer than 2 points.
	LineChart(points []Point) (string, error)
}
