package server

import (
	"testing"

	"github.com/schulze/quantrisk/fair"
)

func TestFmtNum(t *testing.T) {
	tests := []struct {
		in   float64
		want string
	}{
		{0, "0"},
		{1, "1"},
		{50000, "50000"},
		{1000000, "1000000"},
		{0.5, "0.5"},
		{0.123, "0.123"},
		{3.14159, "3.14159"},
	}
	for _, tt := range tests {
		if got := fmtNum(tt.in); got != tt.want {
			t.Errorf("fmtNum(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestFmtEstimate(t *testing.T) {
	tests := []struct {
		name string
		in   fair.Estimate
		want string
	}{
		{"zero", fair.Estimate{}, ""},
		{"integers", fair.Estimate{Min: 100000, ML: 300000, Max: 800000}, "100000 / 300000 / 800000"},
		{"decimals", fair.Estimate{Min: 0.1, ML: 0.5, Max: 0.9}, "0.1 / 0.5 / 0.9"},
		{"mixed", fair.Estimate{Min: 1, ML: 2.5, Max: 10}, "1 / 2.5 / 10"},
	}
	for _, tt := range tests {
		if got := fmtEstimate(tt.in); got != tt.want {
			t.Errorf("fmtEstimate(%s) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestDiffLossEvent(t *testing.T) {
	old := fair.LossEvent{
		LEFMode:   fair.LEFDirect,
		DirectLEF: fair.Estimate{Min: 1, ML: 2, Max: 10},
		PL: fair.LossForm{
			ProdL: fair.Estimate{Min: 50000, ML: 150000, Max: 500000, Rationale: "downtime"},
		},
	}

	// Change LEF mode, LEF values, and one PL rationale.
	new := fair.LossEvent{
		LEFMode: fair.LEFDecomposed,
		TEF:     fair.Estimate{Min: 1, ML: 3, Max: 12},
		Susc:    fair.Estimate{Min: 0.3, ML: 0.5, Max: 0.8},
		PL: fair.LossForm{
			ProdL: fair.Estimate{Min: 50000, ML: 150000, Max: 500000, Rationale: "extended downtime"},
		},
	}

	diffs := diffLossEvent(old, new)

	// Collect field names.
	fields := make(map[string]bool)
	for _, d := range diffs {
		fields[d.field] = true
	}

	// LEF mode changed.
	if !fields["LEF Mode"] {
		t.Error("expected LEF Mode diff")
	}
	// Direct LEF cleared (was 1/2/10, now zero).
	if !fields["LEF"] {
		t.Error("expected LEF diff")
	}
	// TEF added.
	if !fields["TEF"] {
		t.Error("expected TEF diff")
	}
	// Susc added.
	if !fields["Susceptibility"] {
		t.Error("expected Susceptibility diff")
	}
	// PL ProdL rationale changed.
	if !fields["PL Productivity Loss rationale"] {
		t.Error("expected PL Productivity Loss rationale diff")
	}
	// PL ProdL values did NOT change.
	if fields["PL Productivity Loss"] {
		t.Error("PL Productivity Loss values should not differ")
	}
}

func TestDiffLossEventNoChanges(t *testing.T) {
	le := fair.LossEvent{
		LEFMode:   fair.LEFDirect,
		DirectLEF: fair.Estimate{Min: 1, ML: 2, Max: 10},
	}
	diffs := diffLossEvent(le, le)
	if len(diffs) != 0 {
		t.Errorf("expected 0 diffs for identical LossEvents, got %d", len(diffs))
	}
}
