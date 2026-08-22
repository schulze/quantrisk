package server

import (
	"math"
	"testing"

	"github.com/schulze/quantrisk/fair"
	"github.com/schulze/quantrisk/internal/model"
)

func TestRiskToScenario(t *testing.T) {
	r := model.Risk{
		Identifier: "RISK-001",
		Scenario:   "Data breach",
		LossEvent: fair.LossEvent{
			LEFMode:   fair.LEFDirect,
			DirectLEF: fair.Estimate{Min: 1, ML: 2, Max: 3},
			PL: fair.LossForm{
				ProdL: fair.Estimate{Min: 10000, ML: 50000, Max: 100000},
			},
		},
	}
	sc := riskToScenario(r)
	if sc.Identifier != "RISK-001" {
		t.Errorf("Identifier = %q", sc.Identifier)
	}
	if sc.Name != "Data breach" {
		t.Errorf("Name = %q", sc.Name)
	}
	if sc.LEFMode != fair.LEFDirect {
		t.Errorf("LEFMode = %q", sc.LEFMode)
	}
	if sc.Label() != "RISK-001: Data breach" {
		t.Errorf("Label() = %q", sc.Label())
	}
}

func TestRiskToScenario_Simulatable(t *testing.T) {
	r := model.Risk{
		Identifier: "TEST-001",
		Scenario:   "Test scenario",
		LossEvent: fair.LossEvent{
			LEFMode:   fair.LEFDirect,
			DirectLEF: fair.Estimate{Min: 0.5, ML: 1.0, Max: 2.0},
			PL: fair.LossForm{
				ProdL: fair.Estimate{Min: 10000, ML: 50000, Max: 100000},
			},
			SL: fair.LossForm{
				RespC: fair.Estimate{Min: 5000, ML: 20000, Max: 50000},
			},
		},
	}
	sc := riskToScenario(r)

	// Should be simulatable without error
	result, err := fair.SimulateLossEvent(sc.LossEvent, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.YearlyLosses) != 1000 {
		t.Fatalf("got %d years, want 1000", len(result.YearlyLosses))
	}
	if result.Mean() <= 0 {
		t.Errorf("mean = %v, want > 0", result.Mean())
	}
}

func TestRiskToScenario_Decomposed(t *testing.T) {
	r := model.Risk{
		Identifier: "TEST-002",
		Scenario:   "Decomposed",
		LossEvent: fair.LossEvent{
			LEFMode: fair.LEFDecomposed,
			TEF:     fair.Estimate{Min: 5, ML: 10, Max: 20},
			Susc:    fair.Estimate{Min: 0.1, ML: 0.3, Max: 0.5},
			PL: fair.LossForm{
				ProdL: fair.Estimate{Min: 1000, ML: 5000, Max: 10000},
			},
		},
	}
	sc := riskToScenario(r)

	al, err := fair.AnnualizedLoss(sc.LossEvent)
	if err != nil {
		t.Fatal(err)
	}
	if al <= 0 {
		t.Fatalf("AnnualizedLoss = %v, want > 0", al)
	}
}

func TestRiskToScenario_ZeroValues(t *testing.T) {
	// Risk with all zeros should not panic
	r := model.Risk{
		Identifier: "ZERO",
		Scenario:   "Zero",
		LossEvent:  fair.LossEvent{LEFMode: fair.LEFDirect},
	}
	sc := riskToScenario(r)
	_, err := fair.AnnualizedLoss(sc.LossEvent)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRiskToScenario_CombinesPLandSL(t *testing.T) {
	r := model.Risk{
		Identifier: "COMBO",
		Scenario:   "Combined",
		LossEvent: fair.LossEvent{
			LEFMode:   fair.LEFDirect,
			DirectLEF: fair.Estimate{Min: 1, ML: 1, Max: 1},
			PL: fair.LossForm{
				ProdL: fair.Estimate{Min: 100, ML: 200, Max: 300},
				RespC: fair.Estimate{Min: 50, ML: 100, Max: 150},
			},
			SL: fair.LossForm{
				RepuD: fair.Estimate{Min: 1000, ML: 2000, Max: 3000},
			},
		},
	}
	sc := riskToScenario(r)

	// Simulate and verify multiple loss forms contribute
	result, err := fair.SimulateLossEvent(sc.LossEvent, 5000)
	if err != nil {
		t.Fatal(err)
	}

	al, _ := fair.AnnualizedLoss(sc.LossEvent)

	// Simulated mean should be in the neighborhood of the analytical
	if math.Abs(result.Mean()-al)/al > 0.5 {
		t.Errorf("simulated mean %v far from analytical %v", result.Mean(), al)
	}
}
