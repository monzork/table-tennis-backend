// Package svgchart implements chart.Generator by rendering chart data as
// inline SVG markup: no JS, no external assets, no client-side rendering
// cost. Colors are always `currentColor`, so charts inherit the wrapping
// element's existing (already themed) text color for free.
package svgchart

type SVGGenerator struct{}

func NewSVGGenerator() *SVGGenerator {
	return &SVGGenerator{}
}
