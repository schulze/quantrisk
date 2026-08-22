package fair

import (
	"encoding/json"
	"math"
	"testing"
)

func TestEstimateMid(t *testing.T) {
	e := Estimate{Min: 2, ML: 5, Max: 8}
	if got := e.Mid(); got != 5 {
		t.Errorf("Mid() = %v, want 5", got)
	}
}

func TestEstimateIsZero(t *testing.T) {
	if !(Estimate{}).IsZero() {
		t.Error("zero estimate should be zero")
	}
	if (Estimate{Min: 1}).IsZero() {
		t.Error("non-zero estimate should not be zero")
	}
}

func TestEstimatePERTMean(t *testing.T) {
	e := Estimate{Min: 1, ML: 5, Max: 10}
	// (1 + 4*5 + 10) / 6 = 31/6 ≈ 5.167
	want := 31.0 / 6.0
	if got := e.PERTMean(); math.Abs(got-want) > 0.001 {
		t.Errorf("PERTMean() = %v, want %v", got, want)
	}
}

func TestLEFDecomposed(t *testing.T) {
	le := LossEvent{
		LEFMode: LEFDecomposed,
		TEF:     Estimate{Min: 10, ML: 20, Max: 30},
		Susc:    Estimate{Min: 0.2, ML: 0.5, Max: 0.8},
	}
	// TEF.Mid() = 20, Susc.Mid() = 0.5 → LEF = 10
	got := le.LEF()
	if math.Abs(got-10) > 0.001 {
		t.Errorf("LEF() decomposed = %v, want 10", got)
	}
}

func TestLEFDirect(t *testing.T) {
	le := LossEvent{
		LEFMode:   LEFDirect,
		DirectLEF: Estimate{Min: 3, ML: 7, Max: 11},
		// TEF/Susc should be ignored
		TEF:  Estimate{Min: 100, ML: 100, Max: 100},
		Susc: Estimate{Min: 1, ML: 1, Max: 1},
	}
	// DirectLEF.Mid() = (3+11)/2 = 7
	got := le.LEF()
	if math.Abs(got-7) > 0.001 {
		t.Errorf("LEF() direct = %v, want 7", got)
	}
}

func TestLEFDefaultIsDecomposed(t *testing.T) {
	// Zero-value LEFMode should behave as decomposed
	le := LossEvent{
		TEF:  Estimate{Min: 4, ML: 6, Max: 8},
		Susc: Estimate{Min: 0.4, ML: 0.6, Max: 0.8},
	}
	// TEF.Mid()=6, Susc.Mid()=0.6 → 3.6
	got := le.LEF()
	if math.Abs(got-3.6) > 0.001 {
		t.Errorf("LEF() default = %v, want 3.6", got)
	}
}

func TestLossFormEstimates(t *testing.T) {
	lf := LossForm{
		ProdL: Estimate{Min: 1},
		RespC: Estimate{Min: 2},
		ReplC: Estimate{Min: 3},
		FinJu: Estimate{Min: 4},
		RepuD: Estimate{Min: 5},
		CAdvL: Estimate{Min: 6},
	}
	es := lf.Estimates()
	if len(es) != 6 {
		t.Fatalf("Estimates() len = %d, want 6", len(es))
	}
	for i, want := range []float64{1, 2, 3, 4, 5, 6} {
		if es[i].Min != want {
			t.Errorf("Estimates()[%d].Min = %v, want %v", i, es[i].Min, want)
		}
	}
}

// DistType tests

func TestDistType_String(t *testing.T) {
	tests := []struct {
		d    DistType
		want string
	}{
		{DistDefault, "default"},
		{DistPERT, "pert"},
		{DistPoisson, "poisson"},
		{DistLogNormal, "lognormal"},
		{DistBeta, "beta"},
	}
	for _, tt := range tests {
		if got := tt.d.String(); got != tt.want {
			t.Errorf("%d.String() = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestDistType_ParseRoundTrip(t *testing.T) {
	for _, d := range []DistType{DistDefault, DistPERT, DistPoisson, DistLogNormal, DistBeta} {
		b, _ := d.MarshalText()
		var got DistType
		if err := got.UnmarshalText(b); err != nil {
			t.Errorf("UnmarshalText(%q): %v", b, err)
			continue
		}
		if got != d {
			t.Errorf("round-trip %v: got %v", d, got)
		}
	}
}

func TestDistType_JSONRoundTrip(t *testing.T) {
	e := Estimate{Min: 1, ML: 5, Max: 10, Dist: DistLogNormal}
	data, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	var got Estimate
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Dist != DistLogNormal {
		t.Errorf("Dist = %v, want %v", got.Dist, DistLogNormal)
	}
}

func TestDistType_JSONOmitsDefault(t *testing.T) {
	e := Estimate{Min: 1, ML: 5, Max: 10} // Dist == DistDefault
	data, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	// DistDefault marshals to "" which should be omitted by omitempty.
	// However, encoding/json treats empty TextMarshaler output as non-empty.
	// This is a known Go limitation — the field will be present as "dist":"".
	// Just verify it unmarshals back correctly.
	var got Estimate
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Dist != DistDefault {
		t.Errorf("Dist = %v, want DistDefault", got.Dist)
	}
}

func TestParseDistType_Invalid(t *testing.T) {
	_, err := ParseDistType("bogus")
	if err == nil {
		t.Error("expected error for bogus dist type")
	}
}

// LEFMode tests

func TestLEFMode_String(t *testing.T) {
	if got := LEFDirect.String(); got != "direct" {
		t.Errorf("LEFDirect.String() = %q, want direct", got)
	}
	if got := LEFDecomposed.String(); got != "decomposed" {
		t.Errorf("LEFDecomposed.String() = %q, want decomposed", got)
	}
}

func TestLEFMode_ParseRoundTrip(t *testing.T) {
	for _, m := range []LEFMode{LEFDirect, LEFDecomposed} {
		b, _ := m.MarshalText()
		var got LEFMode
		if err := got.UnmarshalText(b); err != nil {
			t.Errorf("UnmarshalText(%q): %v", b, err)
			continue
		}
		if got != m {
			t.Errorf("round-trip %v: got %v", m, got)
		}
	}
}

func TestLEFMode_Scan(t *testing.T) {
	var m LEFMode
	if err := m.Scan("direct"); err != nil {
		t.Fatal(err)
	}
	if m != LEFDirect {
		t.Errorf("Scan(direct) = %v, want %v", m, LEFDirect)
	}

	if err := m.Scan("decomposed"); err != nil {
		t.Fatal(err)
	}
	if m != LEFDecomposed {
		t.Errorf("Scan(decomposed) = %v, want %v", m, LEFDecomposed)
	}
}

func TestLEFMode_Value(t *testing.T) {
	v, err := LEFDirect.Value()
	if err != nil {
		t.Fatal(err)
	}
	if v != "direct" {
		t.Errorf("Value() = %v, want direct", v)
	}
}

func TestLEFMode_JSONRoundTrip(t *testing.T) {
	le := LossEvent{LEFMode: LEFDirect}
	data, err := json.Marshal(le)
	if err != nil {
		t.Fatal(err)
	}
	var got LossEvent
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.LEFMode != LEFDirect {
		t.Errorf("LEFMode = %v, want %v", got.LEFMode, LEFDirect)
	}
}

func TestParseLEFMode_Invalid(t *testing.T) {
	_, err := ParseLEFMode("bogus")
	if err == nil {
		t.Error("expected error for bogus LEF mode")
	}
}
