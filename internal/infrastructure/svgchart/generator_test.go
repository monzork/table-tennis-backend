package svgchart_test

import (
	"strings"
	"testing"
	"time"

	"table-tennis-backend/internal/domain/chart"
	"table-tennis-backend/internal/infrastructure/svgchart"
)

func TestSVGGenerator_BarChart(t *testing.T) {
	g := svgchart.NewSVGGenerator()

	t.Run("empty input returns empty string", func(t *testing.T) {
		svg, err := g.BarChart(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if svg != "" {
			t.Errorf("expected empty string, got %q", svg)
		}
	})

	t.Run("renders valid svg with a rect per item", func(t *testing.T) {
		items := []chart.BarItem{
			{Label: "Primera", Value: 8},
			{Label: "Segunda", Value: 12},
		}
		svg, err := g.BarChart(items)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.HasPrefix(svg, "<svg") || !strings.HasSuffix(svg, "</svg>") {
			t.Fatalf("expected a well-formed svg wrapper, got %q", svg)
		}
		if strings.Count(svg, "<rect") != 2 {
			t.Errorf("expected 2 bars, got svg=%s", svg)
		}
		if !strings.Contains(svg, "role=\"img\"") || !strings.Contains(svg, "<title>") {
			t.Errorf("expected accessible role+title, got %s", svg)
		}
	})

	t.Run("single item does not panic or divide by zero", func(t *testing.T) {
		svg, err := g.BarChart([]chart.BarItem{{Label: "Only", Value: 5}})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(svg, "<rect") {
			t.Errorf("expected a bar to be rendered, got %s", svg)
		}
	})

	t.Run("all-zero values do not divide by zero", func(t *testing.T) {
		items := []chart.BarItem{{Label: "A", Value: 0}, {Label: "B", Value: 0}}
		svg, err := g.BarChart(items)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Count(svg, "<rect") != 2 {
			t.Errorf("expected 2 bars even at zero height, got %s", svg)
		}
	})

	t.Run("labels are escaped", func(t *testing.T) {
		items := []chart.BarItem{{Label: "<script>", Value: 1}}
		svg, err := g.BarChart(items)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(svg, "<script>") {
			t.Errorf("expected label to be escaped, got %s", svg)
		}
	})
}

func TestSVGGenerator_LineChart(t *testing.T) {
	g := svgchart.NewSVGGenerator()

	t.Run("fewer than 2 points returns empty string", func(t *testing.T) {
		svg, err := g.LineChart(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if svg != "" {
			t.Errorf("expected empty string for 0 points, got %q", svg)
		}

		svg, err = g.LineChart([]chart.Point{{X: time.Now(), Y: 1000}})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if svg != "" {
			t.Errorf("expected empty string for 1 point, got %q", svg)
		}
	})

	t.Run("renders a polyline with a vertex per point", func(t *testing.T) {
		points := []chart.Point{
			{X: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Y: 1000, Label: "First"},
			{X: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), Y: 1050, Label: "Second"},
			{X: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), Y: 1020, Label: "Third"},
		}
		svg, err := g.LineChart(points)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(svg, "<polyline") {
			t.Errorf("expected a polyline, got %s", svg)
		}
		if strings.Count(svg, "<circle") != 3 {
			t.Errorf("expected 3 vertex circles, got %s", svg)
		}
	})

	t.Run("flat line (all equal values) does not divide by zero", func(t *testing.T) {
		points := []chart.Point{
			{X: time.Now(), Y: 1000},
			{X: time.Now().Add(time.Hour), Y: 1000},
		}
		svg, err := g.LineChart(points)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(svg, "<polyline") {
			t.Errorf("expected a flat polyline, got %s", svg)
		}
	})
}
