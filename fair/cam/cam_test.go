package cam

import (
	"math"
	"testing"

	"github.com/schulze/quantrisk/fair"
)

func TestCatalogCompleteness(t *testing.T) {
	// All defined function constants must be in the catalog.
	allFuncs := []Function{
		LECAvoidance, LECDeterrence, LECResistance,
		LECVisibility, LECMonitoring, LECRecognition,
		LECEventTermination, LECResilience, LECLossReduction,
		VMCReduceChangeFreq, VMCReduceVarianceProb,
		VMCThreatIntel, VMCControlMonitoring,
		VMCTreatmentSelection, VMCImplementation,
		DSCDefinedExpectations, DSCCommunication, DSCSituationalAwareness,
		DSCEnsureCapability, DSCIncentives,
		DSCIdentification, DSCCorrection,
	}

	if len(Catalog) != len(allFuncs) {
		t.Errorf("Catalog has %d entries, expected %d", len(Catalog), len(allFuncs))
	}

	for _, f := range allFuncs {
		fi, ok := Lookup(f)
		if !ok {
			t.Errorf("function %q not found in catalog", f)
			continue
		}
		if fi.Name == "" {
			t.Errorf("function %q has empty name", f)
		}
		if fi.Domain == "" {
			t.Errorf("function %q has empty domain", f)
		}
		if fi.Domain != f.Domain() {
			t.Errorf("function %q: catalog domain %q != derived domain %q", f, fi.Domain, f.Domain())
		}
	}
}

func TestFunctionDomain(t *testing.T) {
	tests := []struct {
		f    Function
		want Domain
	}{
		{LECAvoidance, DomainLEC},
		{VMCThreatIntel, DomainVMC},
		{DSCIncentives, DomainDSC},
		{Function("X"), ""},
		{Function(""), ""},
	}
	for _, tt := range tests {
		if got := tt.f.Domain(); got != tt.want {
			t.Errorf("%q.Domain() = %q, want %q", tt.f, got, tt.want)
		}
	}
}

func TestFunctionsFilter(t *testing.T) {
	lec := Functions(DomainLEC)
	if len(lec) != 9 {
		t.Errorf("LEC functions: got %d, want 9", len(lec))
	}
	vmc := Functions(DomainVMC)
	if len(vmc) != 6 {
		t.Errorf("VMC functions: got %d, want 6", len(vmc))
	}
	dsc := Functions(DomainDSC)
	if len(dsc) != 7 {
		t.Errorf("DSC functions: got %d, want 7", len(dsc))
	}
	all := Functions("")
	if len(all) != 22 {
		t.Errorf("all functions: got %d, want 22", len(all))
	}
}

func TestParents(t *testing.T) {
	lecParents := Parents(DomainLEC)
	want := []string{"LEC.Prevention", "LEC.Detection", "LEC.Response"}
	if len(lecParents) != len(want) {
		t.Fatalf("LEC parents: got %v, want %v", lecParents, want)
	}
	for i, p := range want {
		if lecParents[i] != p {
			t.Errorf("LEC parent %d: got %q, want %q", i, lecParents[i], p)
		}
	}

	vmcParents := Parents(DomainVMC)
	wantVMC := []string{"VMC.Prevention", "VMC.Identification", "VMC.Correction"}
	if len(vmcParents) != len(wantVMC) {
		t.Fatalf("VMC parents: got %v, want %v", vmcParents, wantVMC)
	}

	dscParents := Parents(DomainDSC)
	wantDSC := []string{"DSC.Prevention", "DSC.Identification", "DSC.Correction"}
	if len(dscParents) != len(wantDSC) {
		t.Fatalf("DSC parents: got %v, want %v", dscParents, wantDSC)
	}
}

func TestEffectivenessOverallMid(t *testing.T) {
	e := Effectiveness{
		Capability:  fair.Estimate{Min: 0.6, ML: 0.8, Max: 1.0},
		Coverage:    fair.Estimate{Min: 0.4, ML: 0.6, Max: 0.8},
		Reliability: fair.Estimate{Min: 0.7, ML: 0.9, Max: 1.0},
	}
	// Mid: Cap=0.8, Cov=0.6, Rel=0.85 → 0.8*0.6*0.85 = 0.408
	got := e.OverallMid()
	if math.Abs(got-0.408) > 0.001 {
		t.Errorf("OverallMid() = %v, want 0.408", got)
	}
}

func TestLookupNotFound(t *testing.T) {
	_, ok := Lookup(Function("nonexistent"))
	if ok {
		t.Error("expected Lookup to return false for nonexistent function")
	}
}

func TestSiblingRelationships(t *testing.T) {
	// Verify key relationships from the FAIR-CAM standard.
	tests := []struct {
		f    Function
		want Relationship
	}{
		// LEC Prevention = OR
		{LECAvoidance, OR},
		{LECDeterrence, OR},
		{LECResistance, OR},
		// LEC Detection = AND
		{LECVisibility, AND},
		{LECMonitoring, AND},
		{LECRecognition, AND},
		// LEC Response = WeakAND
		{LECEventTermination, WeakAND},
		{LECResilience, WeakAND},
		{LECLossReduction, WeakAND},
		// VMC Prevention = OR
		{VMCReduceChangeFreq, OR},
		{VMCReduceVarianceProb, OR},
		// VMC Identification = AND
		{VMCThreatIntel, AND},
		// VMC Correction = AND
		{VMCTreatmentSelection, AND},
		// DSC Prevention = AND
		{DSCDefinedExpectations, AND},
		{DSCIncentives, AND},
	}
	for _, tt := range tests {
		fi, ok := Lookup(tt.f)
		if !ok {
			t.Errorf("%q not found", tt.f)
			continue
		}
		if fi.SiblingRelationship != tt.want {
			t.Errorf("%q relationship = %q, want %q", tt.f, fi.SiblingRelationship, tt.want)
		}
	}
}
