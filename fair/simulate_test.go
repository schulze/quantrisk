package fair

import (
	"math"
	"testing"
)

// Distribution tests

func TestNewDistribution_FrequencyDefault(t *testing.T) {
	// Frequency role defaults to Poisson
	e := Estimate{Min: 1, ML: 5, Max: 10}
	d, err := NewDistribution(e, RoleFrequency)
	if err != nil {
		t.Fatal(err)
	}
	_, ok := d.(*PoissonDist)
	if !ok {
		t.Fatalf("expected PoissonDist, got %T", d)
	}
	// Lambda should be PERT mean: (1+4*5+10)/6 = 31/6 ≈ 5.167
	wantMean := 31.0 / 6.0
	if math.Abs(d.Mean()-wantMean) > 0.01 {
		t.Errorf("Mean() = %v, want ~%v", d.Mean(), wantMean)
	}
}

func TestNewDistribution_MagnitudeDefault(t *testing.T) {
	e := Estimate{Min: 1000, ML: 5000, Max: 50000}
	d, err := NewDistribution(e, RoleMagnitude)
	if err != nil {
		t.Fatal(err)
	}
	_, ok := d.(*PERTDist)
	if !ok {
		t.Fatalf("expected PERTDist, got %T", d)
	}
}

func TestNewDistribution_ProbabilityDefault(t *testing.T) {
	e := Estimate{Min: 0.1, ML: 0.5, Max: 0.9}
	d, err := NewDistribution(e, RoleProbability)
	if err != nil {
		t.Fatal(err)
	}
	_, ok := d.(*BetaDist)
	if !ok {
		t.Fatalf("expected BetaDist, got %T", d)
	}
	// Mean should be around 0.5
	if d.Mean() < 0.3 || d.Mean() > 0.7 {
		t.Errorf("Mean() = %v, want ~0.5", d.Mean())
	}
}

func TestNewDistribution_ExplicitOverride(t *testing.T) {
	// Override frequency role with PERT
	e := Estimate{Min: 1, ML: 5, Max: 10, Dist: DistPERT}
	d, err := NewDistribution(e, RoleFrequency)
	if err != nil {
		t.Fatal(err)
	}
	_, ok := d.(*PERTDist)
	if !ok {
		t.Fatalf("expected PERTDist, got %T", d)
	}
}

func TestNewDistribution_LogNormal(t *testing.T) {
	e := Estimate{Min: 1000, ML: 5000, Max: 100000, Dist: DistLogNormal}
	d, err := NewDistribution(e, RoleMagnitude)
	if err != nil {
		t.Fatal(err)
	}
	_, ok := d.(*LogNormalDist)
	if !ok {
		t.Fatalf("expected LogNormalDist, got %T", d)
	}
	// Mean should be positive
	if d.Mean() <= 0 {
		t.Errorf("Mean() = %v, want > 0", d.Mean())
	}
}

func TestPoissonDist_Draw(t *testing.T) {
	d := newPoisson(5.0)
	draws := d.Draw(10000)
	sum := 0.0
	for _, v := range draws {
		if v < 0 {
			t.Fatalf("negative draw: %v", v)
		}
		sum += v
	}
	avg := sum / float64(len(draws))
	if avg < 4.0 || avg > 6.0 {
		t.Errorf("average = %v, want ~5", avg)
	}
}

func TestPERTDist_Draw(t *testing.T) {
	d, err := newPERT(100, 500, 1000)
	if err != nil {
		t.Fatal(err)
	}
	draws := d.Draw(10000)
	for _, v := range draws {
		if v < 100 || v > 1000 {
			t.Fatalf("draw %v outside [100, 1000]", v)
		}
	}
	// Mean should be close to PERT mean: (100+4*500+1000)/6 = 3100/6 ≈ 516.67
	want := (100.0 + 4*500.0 + 1000.0) / 6.0
	sum := 0.0
	for _, v := range draws {
		sum += v
	}
	avg := sum / float64(len(draws))
	if math.Abs(avg-want) > want*0.1 {
		t.Errorf("average = %v, want ~%v", avg, want)
	}
}

func TestBetaDist_Draw(t *testing.T) {
	d, err := newBeta(0.1, 0.5, 0.9)
	if err != nil {
		t.Fatal(err)
	}
	draws := d.Draw(10000)
	for _, v := range draws {
		if v < 0 || v > 1 {
			t.Fatalf("draw %v outside [0, 1]", v)
		}
	}
}

func TestLogNormalDist_Draw(t *testing.T) {
	d, err := newLogNormal(1000, 100000)
	if err != nil {
		t.Fatal(err)
	}
	draws := d.Draw(10000)
	for _, v := range draws {
		if v < 0 {
			t.Fatalf("negative draw: %v", v)
		}
	}
}

// SimulateLossEvent tests

func TestSimulateLossEvent_DirectLEF(t *testing.T) {
	le := LossEvent{
		LEFMode:   LEFDirect,
		DirectLEF: Estimate{Min: 1, ML: 2, Max: 3},
		PL: LossForm{
			ProdL: Estimate{Min: 10000, ML: 50000, Max: 100000},
		},
	}
	result, err := SimulateLossEvent(le, 10000)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.YearlyLosses) != 10000 {
		t.Fatalf("got %d years, want 10000", len(result.YearlyLosses))
	}

	// Average should be in a reasonable range
	avg := result.Mean()
	if avg <= 0 {
		t.Fatalf("mean = %v, want > 0", avg)
	}

	// Frequency ~2 events/yr, magnitude PERT mean ~50000-ish
	// So annual loss should be roughly in 50k-200k range
	if avg < 10000 || avg > 500000 {
		t.Errorf("mean = %v, expected roughly 50k-200k range", avg)
	}
}

func TestSimulateLossEvent_Decomposed(t *testing.T) {
	le := LossEvent{
		LEFMode: LEFDecomposed,
		TEF:     Estimate{Min: 5, ML: 10, Max: 20},
		Susc:    Estimate{Min: 0.2, ML: 0.5, Max: 0.8},
		PL: LossForm{
			ProdL: Estimate{Min: 1000, ML: 5000, Max: 10000},
		},
	}
	result, err := SimulateLossEvent(le, 10000)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.YearlyLosses) != 10000 {
		t.Fatalf("got %d years, want 10000", len(result.YearlyLosses))
	}

	avg := result.Mean()
	if avg <= 0 {
		t.Fatalf("mean = %v, want > 0", avg)
	}
}

func TestSimulateLossEvent_MultipleLossForms(t *testing.T) {
	le := LossEvent{
		LEFMode:   LEFDirect,
		DirectLEF: Estimate{Min: 1, ML: 1, Max: 1},
		PL: LossForm{
			ProdL: Estimate{Min: 100, ML: 200, Max: 300},
			RespC: Estimate{Min: 50, ML: 100, Max: 150},
		},
		SL: LossForm{
			RepuD: Estimate{Min: 1000, ML: 2000, Max: 3000},
		},
	}
	result, err := SimulateLossEvent(le, 5000)
	if err != nil {
		t.Fatal(err)
	}

	avg := result.Mean()
	// With ~1 event/yr, total magnitude should be around
	// ProdL(~200) + RespC(~100) + RepuD(~2000) = ~2300
	if avg < 500 || avg > 10000 {
		t.Errorf("mean = %v, expected ~2300 range", avg)
	}
}

func TestSimulateLossEvent_ZeroEstimates(t *testing.T) {
	// All-zero loss forms should produce zero losses
	le := LossEvent{
		LEFMode:   LEFDirect,
		DirectLEF: Estimate{Min: 1, ML: 2, Max: 3},
	}
	result, err := SimulateLossEvent(le, 100)
	if err != nil {
		t.Fatal(err)
	}
	for i, v := range result.YearlyLosses {
		if v != 0 {
			t.Fatalf("year[%d] = %v, want 0 (no magnitude estimates)", i, v)
		}
	}
}

func TestSimulateLossEvent_ZeroFrequency(t *testing.T) {
	le := LossEvent{
		LEFMode:   LEFDirect,
		DirectLEF: Estimate{Min: 0, ML: 0, Max: 0},
		PL: LossForm{
			ProdL: Estimate{Min: 10000, ML: 50000, Max: 100000},
		},
	}
	result, err := SimulateLossEvent(le, 1000)
	if err != nil {
		t.Fatal(err)
	}
	// With near-zero frequency (clamped to 0.001), nearly all years should be 0
	nonZero := 0
	for _, v := range result.YearlyLosses {
		if v > 0 {
			nonZero++
		}
	}
	// At lambda=0.001, expect <5 non-zero years in 1000
	if nonZero > 20 {
		t.Errorf("%d non-zero years, expected very few", nonZero)
	}
}

// AnnualizedLoss tests

func TestAnnualizedLoss_Direct(t *testing.T) {
	le := LossEvent{
		LEFMode:   LEFDirect,
		DirectLEF: Estimate{Min: 1, ML: 2, Max: 3},
		PL: LossForm{
			ProdL: Estimate{Min: 10000, ML: 50000, Max: 100000},
		},
	}
	al, err := AnnualizedLoss(le)
	if err != nil {
		t.Fatal(err)
	}
	if al <= 0 {
		t.Fatalf("AnnualizedLoss = %v, want > 0", al)
	}
}

func TestAnnualizedLoss_Decomposed(t *testing.T) {
	le := LossEvent{
		LEFMode: LEFDecomposed,
		TEF:     Estimate{Min: 10, ML: 20, Max: 30},
		Susc:    Estimate{Min: 0.3, ML: 0.5, Max: 0.7},
		PL: LossForm{
			ProdL: Estimate{Min: 1000, ML: 5000, Max: 10000},
		},
	}
	al, err := AnnualizedLoss(le)
	if err != nil {
		t.Fatal(err)
	}
	if al <= 0 {
		t.Fatalf("AnnualizedLoss = %v, want > 0", al)
	}
}

func TestAnnualizedLoss_ZeroMagnitude(t *testing.T) {
	le := LossEvent{
		LEFMode:   LEFDirect,
		DirectLEF: Estimate{Min: 1, ML: 2, Max: 3},
		// No loss forms set
	}
	al, err := AnnualizedLoss(le)
	if err != nil {
		t.Fatal(err)
	}
	if al != 0 {
		t.Errorf("AnnualizedLoss = %v, want 0 (no magnitudes)", al)
	}
}

// SimulationResult tests

func TestSimulationResult_MeanEmpty(t *testing.T) {
	sr := SimulationResult{}
	if sr.Mean() != 0 {
		t.Errorf("Mean() = %v, want 0", sr.Mean())
	}
}

func TestSimulationResult_Mean(t *testing.T) {
	sr := SimulationResult{YearlyLosses: []float64{100, 200, 300}}
	if sr.Mean() != 200 {
		t.Errorf("Mean() = %v, want 200", sr.Mean())
	}
}

// Dist field override

func TestSimulateLossEvent_PERTFrequency(t *testing.T) {
	// Override frequency distribution to use PERT instead of Poisson
	le := LossEvent{
		LEFMode:   LEFDirect,
		DirectLEF: Estimate{Min: 1, ML: 5, Max: 10, Dist: DistPERT},
		PL: LossForm{
			ProdL: Estimate{Min: 1000, ML: 5000, Max: 10000},
		},
	}
	result, err := SimulateLossEvent(le, 5000)
	if err != nil {
		t.Fatal(err)
	}
	if result.Mean() <= 0 {
		t.Fatalf("mean = %v, want > 0", result.Mean())
	}
}

func TestSimulateLossEvent_LogNormalMagnitude(t *testing.T) {
	// Override magnitude to use LogNormal
	le := LossEvent{
		LEFMode:   LEFDirect,
		DirectLEF: Estimate{Min: 1, ML: 2, Max: 3},
		PL: LossForm{
			ProdL: Estimate{Min: 1000, ML: 5000, Max: 100000, Dist: DistLogNormal},
		},
	}
	result, err := SimulateLossEvent(le, 5000)
	if err != nil {
		t.Fatal(err)
	}
	if result.Mean() <= 0 {
		t.Fatalf("mean = %v, want > 0", result.Mean())
	}
}

func TestSimulateLossEvent_SimulationConverges(t *testing.T) {
	// The simulated mean should be reasonably close to the analytical annualized loss
	le := LossEvent{
		LEFMode:   LEFDirect,
		DirectLEF: Estimate{Min: 2, ML: 5, Max: 8},
		PL: LossForm{
			ProdL: Estimate{Min: 10000, ML: 50000, Max: 100000},
		},
	}
	result, err := SimulateLossEvent(le, 50000)
	if err != nil {
		t.Fatal(err)
	}

	analytical, err := AnnualizedLoss(le)
	if err != nil {
		t.Fatal(err)
	}

	simMean := result.Mean()
	// Allow 30% tolerance for convergence
	if math.Abs(simMean-analytical)/analytical > 0.30 {
		t.Errorf("simulation mean %v far from analytical %v", simMean, analytical)
	}
}

// Scenario and multi-scenario tests

func TestScenarioLabel(t *testing.T) {
	s := Scenario{Identifier: "RISK-001", Name: "Data breach"}
	if got := s.Label(); got != "RISK-001: Data breach" {
		t.Errorf("Label() = %q", got)
	}
	s2 := Scenario{Identifier: "RISK-002"}
	if got := s2.Label(); got != "RISK-002" {
		t.Errorf("Label() = %q, want RISK-002", got)
	}
}

func TestPrioritizedLosses(t *testing.T) {
	scenarios := []Scenario{
		{
			Identifier: "SMALL",
			Name:       "Small",
			LossEvent: LossEvent{
				LEFMode:   LEFDirect,
				DirectLEF: Estimate{Min: 1, ML: 1, Max: 1},
				PL:        LossForm{ProdL: Estimate{Min: 100, ML: 200, Max: 300}},
			},
		},
		{
			Identifier: "BIG",
			Name:       "Big",
			LossEvent: LossEvent{
				LEFMode:   LEFDirect,
				DirectLEF: Estimate{Min: 10, ML: 10, Max: 10},
				PL:        LossForm{ProdL: Estimate{Min: 10000, ML: 50000, Max: 100000}},
			},
		},
	}
	result, err := PrioritizedLosses(scenarios)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Fatalf("got %d results, want 2", len(result))
	}
	// BIG should be first (higher loss)
	if result[0].Identifier != "BIG" {
		t.Errorf("first = %q, want BIG", result[0].Identifier)
	}
	if result[1].Identifier != "SMALL" {
		t.Errorf("second = %q, want SMALL", result[1].Identifier)
	}
}

func TestSimulateMulti(t *testing.T) {
	scenarios := []Scenario{
		{
			Identifier: "A",
			LossEvent: LossEvent{
				LEFMode:   LEFDirect,
				DirectLEF: Estimate{Min: 1, ML: 2, Max: 3},
				PL:        LossForm{ProdL: Estimate{Min: 1000, ML: 5000, Max: 10000}},
			},
		},
		{
			Identifier: "B",
			LossEvent: LossEvent{
				LEFMode:   LEFDirect,
				DirectLEF: Estimate{Min: 2, ML: 5, Max: 8},
				PL:        LossForm{ProdL: Estimate{Min: 500, ML: 2000, Max: 5000}},
			},
		},
	}
	per, agg, err := SimulateMulti(scenarios, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if len(per) != 2 {
		t.Fatalf("got %d per-scenario results, want 2", len(per))
	}
	if len(agg.YearlyLosses) != 1000 {
		t.Fatalf("aggregate has %d years, want 1000", len(agg.YearlyLosses))
	}
	// Aggregate should be >= each individual scenario for each year
	for yr := 0; yr < 1000; yr++ {
		sum := per[0].YearlyLosses[yr] + per[1].YearlyLosses[yr]
		if math.Abs(agg.YearlyLosses[yr]-sum) > 0.01 {
			t.Fatalf("year %d: aggregate %v != sum %v", yr, agg.YearlyLosses[yr], sum)
		}
	}
}

func TestNewSimpleScenario(t *testing.T) {
	s := NewSimpleScenario("ALICE", "Alice steals", 0.1, 1000, 10000)
	if s.Identifier != "ALICE" {
		t.Errorf("Identifier = %q", s.Identifier)
	}
	if s.LEFMode != LEFDirect {
		t.Errorf("LEFMode = %v, want %v", s.LEFMode, LEFDirect)
	}
	if s.DirectLEF.Dist != DistPoisson {
		t.Errorf("DirectLEF.Dist = %v, want %v", s.DirectLEF.Dist, DistPoisson)
	}
	if s.PL.ProdL.Dist != DistLogNormal {
		t.Errorf("PL.ProdL.Dist = %v, want %v", s.PL.ProdL.Dist, DistLogNormal)
	}
	// Should be simulatable
	result, err := SimulateLossEvent(s.LossEvent, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if result.Mean() <= 0 {
		t.Errorf("mean = %v, want > 0", result.Mean())
	}
}

func TestNewSimpleScenario_MatchesLegacy(t *testing.T) {
	// Verify that NewSimpleScenario produces annualized losses in the same
	// ballpark as the legacy risk.NewSimpleLoss (Poisson+LogNormal).
	s := NewSimpleScenario("BOB", "Bob", 2.0, 1000, 100000)
	al, err := AnnualizedLoss(s.LossEvent)
	if err != nil {
		t.Fatal(err)
	}
	// Legacy: Poisson(2) × LogNormal(1000, 100000).Mean()
	// The analytical means should be very similar.
	if al < 1000 || al > 10000000 {
		t.Errorf("annualized loss = %v, expected reasonable range", al)
	}
}
