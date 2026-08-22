package chart

import (
	"bytes"
	"math"
	"strings"
	"testing"
)

func TestExceedancePointsFrom_Length(t *testing.T) {
	losses := make([]float64, 1000)
	for i := range losses {
		losses[i] = float64(i)
	}
	points := ExceedancePointsFrom(losses, 50)
	if len(points) != 50 {
		t.Fatalf("expected 50 points, got %d", len(points))
	}
}

func TestExceedancePointsFrom_DecreasingProbability(t *testing.T) {
	losses := make([]float64, 5000)
	for i := range losses {
		losses[i] = float64(i)
	}
	points := ExceedancePointsFrom(losses, 99)
	for i := 1; i < len(points); i++ {
		if points[i].Probability >= points[i-1].Probability {
			t.Fatalf("probability not decreasing at index %d: %v >= %v",
				i, points[i].Probability, points[i-1].Probability)
		}
	}
}

func TestExceedancePointsFrom_IncreasingLoss(t *testing.T) {
	losses := make([]float64, 5000)
	for i := range losses {
		losses[i] = float64(i)
	}
	points := ExceedancePointsFrom(losses, 99)
	for i := 1; i < len(points); i++ {
		if points[i].Loss < points[i-1].Loss {
			t.Fatalf("loss not increasing at index %d: %v < %v",
				i, points[i].Loss, points[i-1].Loss)
		}
	}
}

func TestRenderLEC_SingleCurve(t *testing.T) {
	points := []ExceedancePoint{
		{Loss: 100, Probability: 0.9},
		{Loss: 500, Probability: 0.5},
		{Loss: 1000, Probability: 0.1},
	}
	curves := []NamedCurve{{Label: "Test", Points: points}}
	var buf bytes.Buffer
	RenderLEC(curves, "Test LEC", &buf)
	svg := buf.String()
	if !strings.HasPrefix(svg, "<svg") {
		t.Fatalf("output does not start with <svg: got %q...", svg[:min(len(svg), 40)])
	}
	if !strings.Contains(svg, "</svg>") {
		t.Fatal("output does not contain closing </svg> tag")
	}
}

func TestRenderLEC_MultiCurve(t *testing.T) {
	points := []ExceedancePoint{
		{Loss: 100, Probability: 0.9},
		{Loss: 1000, Probability: 0.1},
	}
	curves := []NamedCurve{
		{Label: "Scenario A", Points: points},
		{Label: "Scenario B", Points: points},
		{Label: "Aggregate", Points: points},
	}
	var buf bytes.Buffer
	RenderLEC(curves, "Multi LEC", &buf)
	svg := buf.String()
	if !strings.Contains(svg, "Aggregate") {
		t.Fatal("multi-curve SVG should contain Aggregate label")
	}
}

func TestFormatLoss(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{0, "0"},
		{500, "500"},
		{1000, "1K"},
		{1500, "2K"},
		{1000000, "1M"},
		{1000000000, "1B"},
	}
	for _, tc := range tests {
		got := formatLoss(tc.input)
		if got != tc.want {
			t.Errorf("formatLoss(%v) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestAutoXTicks_Linear(t *testing.T) {
	curves := []NamedCurve{{
		Label: "test",
		Points: []ExceedancePoint{
			{Loss: 100, Probability: 0.9},
			{Loss: 50000, Probability: 0.1},
		},
	}}
	xMin, xMax, ticks := autoXTicks(curves)
	if xMin != 0 {
		t.Errorf("xMin = %v, want 0", xMin)
	}
	if xMax < 50000 {
		t.Errorf("xMax = %v, want >= 50000", xMax)
	}
	if len(ticks) < 2 {
		t.Fatalf("expected at least 2 ticks, got %d", len(ticks))
	}
	// First tick should be 0
	if ticks[0].val != 0 {
		t.Errorf("first tick = %v, want 0", ticks[0].val)
	}
	// Ticks should be evenly spaced
	step := ticks[1].val - ticks[0].val
	for i := 2; i < len(ticks); i++ {
		got := ticks[i].val - ticks[i-1].val
		if math.Abs(got-step) > 0.01 {
			t.Errorf("tick spacing not uniform: tick[%d]-tick[%d] = %v, want %v", i, i-1, got, step)
		}
	}
}

func TestNiceMax(t *testing.T) {
	tests := []struct {
		input float64
		want  float64
	}{
		{0, 1000},
		{750, 1000},
		{1200, 2000},
		{4500, 5000},
		{7500, 10000},
		{15000, 20000},
		{3e6, 5e6},
	}
	for _, tc := range tests {
		got := niceMax(tc.input)
		if got != tc.want {
			t.Errorf("niceMax(%v) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestNiceStep(t *testing.T) {
	tests := []struct {
		max  float64
		n    int
		want float64
	}{
		{1000, 6, 200},
		{5000, 6, 1000},
		{50000, 6, 10000},
		{2000000, 6, 500000},
	}
	for _, tc := range tests {
		got := niceStep(tc.max, tc.n)
		if got != tc.want {
			t.Errorf("niceStep(%v, %d) = %v, want %v", tc.max, tc.n, got, tc.want)
		}
	}
}
