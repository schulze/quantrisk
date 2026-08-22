// Package chart renders Loss Exceedance Curve (LEC) SVG charts.
//
// It accepts pre-computed exceedance data and produces self-contained SVG.
// All simulation logic lives in the fair/ package; this package only renders.
package chart

import (
	"fmt"
	"io"
	"math"
	"sort"
	"strings"

	"gonum.org/v1/gonum/stat"
)

// ExceedancePoint represents a single point on a loss exceedance curve.
type ExceedancePoint struct {
	Loss        float64
	Probability float64
}

// NamedCurve pairs a label with exceedance points for chart rendering.
type NamedCurve struct {
	Label  string
	Points []ExceedancePoint
}

// ExceedancePointsFrom computes n percentile points from pre-simulated yearly losses.
func ExceedancePointsFrom(losses []float64, n int) []ExceedancePoint {
	sorted := make([]float64, len(losses))
	copy(sorted, losses)
	sort.Float64s(sorted)

	points := make([]ExceedancePoint, n)
	for i := range points {
		x := float64(i + 1)
		points[i] = ExceedancePoint{
			Loss:        stat.Quantile(x/float64(n+1), stat.LinInterp, sorted, nil) + 0.1,
			Probability: 1 - x/float64(n+1),
		}
	}
	return points
}

// RenderLEC writes a self-contained SVG chart with the given labeled curves.
func RenderLEC(curves []NamedCurve, title string, w io.Writer) {
	renderMultiLECSVG(curves, title, w)
}

// SVG rendering

// curveColors defines distinct colors for up to 10 curves, plus aggregate.
var curveColors = []string{
	"#2563eb", // blue
	"#dc2626", // red
	"#16a34a", // green
	"#d97706", // amber
	"#7c3aed", // violet
	"#0891b2", // cyan
	"#be185d", // pink
	"#65a30d", // lime
	"#ea580c", // orange
	"#4338ca", // indigo
}

const aggregateColor = "#111827" // near-black for aggregate

type tick struct {
	val   float64
	label string
}

// formatLoss returns a human-readable label for a loss value.
func formatLoss(v float64) string {
	switch {
	case v >= 1e9:
		return fmt.Sprintf("%.0fB", v/1e9)
	case v >= 1e6:
		return fmt.Sprintf("%.0fM", v/1e6)
	case v >= 1e3:
		return fmt.Sprintf("%.0fK", v/1e3)
	default:
		return fmt.Sprintf("%.0f", v)
	}
}

// autoXTicks generates linear x-axis ticks spanning the data range.
func autoXTicks(curves []NamedCurve) (xMin, xMax float64, ticks []tick) {
	minLoss := math.MaxFloat64
	maxLoss := 0.0
	for _, c := range curves {
		for _, p := range c.Points {
			if p.Loss < minLoss {
				minLoss = p.Loss
			}
			if p.Loss > maxLoss {
				maxLoss = p.Loss
			}
		}
	}
	if minLoss >= maxLoss {
		minLoss = 0
		maxLoss = 1000
	}

	xMin = 0
	xMax = niceMax(maxLoss)

	step := niceStep(xMax, 6)
	for v := xMin; v <= xMax+step*0.01; v += step {
		ticks = append(ticks, tick{val: v, label: formatLoss(v)})
	}
	return
}

// niceMax rounds a value up to a clean human-readable number.
func niceMax(v float64) float64 {
	if v <= 0 {
		return 1000
	}
	exponent := math.Floor(math.Log10(v))
	fraction := v / math.Pow(10, exponent)
	var nice float64
	switch {
	case fraction <= 1:
		nice = 1
	case fraction <= 2:
		nice = 2
	case fraction <= 5:
		nice = 5
	default:
		nice = 10
	}
	return nice * math.Pow(10, exponent)
}

// niceStep picks a clean tick interval that yields roughly n ticks.
func niceStep(max float64, n int) float64 {
	rough := max / float64(n)
	exponent := math.Floor(math.Log10(rough))
	fraction := rough / math.Pow(10, exponent)
	var nice float64
	switch {
	case fraction <= 1.5:
		nice = 1
	case fraction <= 3:
		nice = 2
	case fraction <= 7:
		nice = 5
	default:
		nice = 10
	}
	return nice * math.Pow(10, exponent)
}

// autoYMax finds the maximum probability across all curves.
func autoYMax(curves []NamedCurve) float64 {
	max := 0.0
	for _, c := range curves {
		for _, p := range c.Points {
			if p.Probability > max {
				max = p.Probability
			}
		}
	}
	// Round up to nearest 0.05
	max = math.Ceil(max*20) / 20
	if max < 0.05 {
		max = 0.05
	}
	return max
}

// renderMultiLECSVG writes a self-contained SVG chart with multiple labeled curves.
func renderMultiLECSVG(curves []NamedCurve, title string, w io.Writer) {
	legendH := len(curves) * 20
	const (
		svgWidth = 700
		marginL  = 80
		marginR  = 30
		marginT  = 50
		marginB  = 60
	)
	svgHeight := 500 + legendH

	xMin, xMax, xTicks := autoXTicks(curves)
	yMax := autoYMax(curves)

	plotW := float64(svgWidth - marginL - marginR)
	plotH := float64(svgHeight - legendH - marginT - marginB)

	mapX := func(loss float64) float64 {
		if xMax == xMin {
			return float64(marginL)
		}
		frac := (loss - xMin) / (xMax - xMin)
		return float64(marginL) + frac*plotW
	}
	mapY := func(prob float64) float64 {
		frac := prob / yMax
		return float64(marginT) + plotH - frac*plotH
	}

	// Y-axis ticks
	yStep := 0.05
	if yMax <= 0.1 {
		yStep = 0.02
	} else if yMax > 0.5 {
		yStep = 0.1
	}
	var yTicks []tick
	for v := 0.0; v <= yMax+0.001; v += yStep {
		yTicks = append(yTicks, tick{val: v, label: fmt.Sprintf("%.2f", v)})
	}

	fmt.Fprintf(w, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" font-family="sans-serif" font-size="12">`, svgWidth, svgHeight)
	fmt.Fprintln(w)

	fmt.Fprintf(w, `<rect width="%d" height="%d" fill="white"/>`, svgWidth, svgHeight)
	fmt.Fprintln(w)

	fmt.Fprintf(w, `<text x="%d" y="%d" text-anchor="middle" font-size="16" font-weight="bold">%s</text>`,
		svgWidth/2, 28, title)
	fmt.Fprintln(w)

	// X grid + labels
	for _, t := range xTicks {
		x := mapX(t.val)
		fmt.Fprintf(w, `<line x1="%.1f" y1="%d" x2="%.1f" y2="%.1f" stroke="#ddd" stroke-width="1"/>`,
			x, marginT, x, float64(marginT)+plotH)
		fmt.Fprintf(w, `<text x="%.1f" y="%.1f" text-anchor="middle" font-size="11">%s</text>`,
			x, float64(marginT)+plotH+18, t.label)
		fmt.Fprintln(w)
	}

	// Y grid + labels
	for _, t := range yTicks {
		y := mapY(t.val)
		fmt.Fprintf(w, `<line x1="%d" y1="%.1f" x2="%.1f" y2="%.1f" stroke="#ddd" stroke-width="1"/>`,
			marginL, y, float64(svgWidth-marginR), y)
		fmt.Fprintf(w, `<text x="%d" y="%.1f" text-anchor="end" font-size="11" dominant-baseline="middle">%s</text>`,
			marginL-8, y, t.label)
		fmt.Fprintln(w)
	}

	// Axes box
	fmt.Fprintf(w, `<rect x="%d" y="%d" width="%.0f" height="%.0f" fill="none" stroke="#333" stroke-width="1"/>`,
		marginL, marginT, plotW, plotH)
	fmt.Fprintln(w)

	// Axis labels
	xLabelY := int(float64(marginT) + plotH + 42)
	fmt.Fprintf(w, `<text x="%d" y="%d" text-anchor="middle" font-size="13">Loss ($)</text>`,
		svgWidth/2, xLabelY)
	yLabelY := marginT + int(plotH)/2
	fmt.Fprintf(w, `<text x="18" y="%d" text-anchor="middle" font-size="13" transform="rotate(-90,18,%d)">Probability of Exceedance</text>`,
		yLabelY, yLabelY)
	fmt.Fprintln(w)

	// Draw curves
	for ci, curve := range curves {
		color := aggregateColor
		strokeW := "2.5"
		if ci < len(curves)-1 || len(curves) == 1 {
			color = curveColors[ci%len(curveColors)]
			strokeW = "1.5"
		}
		// For single-curve charts, use heavier stroke
		if len(curves) == 1 {
			strokeW = "2"
		}

		var polyPts strings.Builder
		for i, pt := range curve.Points {
			x := math.Max(float64(marginL), math.Min(mapX(pt.Loss), float64(svgWidth-marginR)))
			y := math.Max(float64(marginT), math.Min(mapY(pt.Probability), float64(marginT)+plotH))
			if i > 0 {
				polyPts.WriteByte(' ')
			}
			fmt.Fprintf(&polyPts, "%.1f,%.1f", x, y)
		}
		fmt.Fprintf(w, `<polyline points="%s" fill="none" stroke="%s" stroke-width="%s"/>`,
			polyPts.String(), color, strokeW)
		fmt.Fprintln(w)
	}

	// Legend below chart
	legendY := marginT + int(plotH) + 55
	for ci, curve := range curves {
		color := aggregateColor
		if ci < len(curves)-1 || len(curves) == 1 {
			color = curveColors[ci%len(curveColors)]
		}
		y := legendY + ci*20
		fmt.Fprintf(w, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="2"/>`,
			marginL, y, marginL+20, y, color)
		// Truncate long labels
		label := curve.Label
		if len(label) > 60 {
			label = label[:57] + "…"
		}
		fmt.Fprintf(w, `<text x="%d" y="%d" font-size="11" dominant-baseline="middle">%s</text>`,
			marginL+26, y, label)
		fmt.Fprintln(w)
	}

	fmt.Fprintln(w, `</svg>`)
}
