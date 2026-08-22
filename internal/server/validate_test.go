package server

import (
	"math"
	"strings"
	"testing"

	"github.com/schulze/quantrisk/fair"
	"github.com/schulze/quantrisk/fair/cam"
	"github.com/schulze/quantrisk/internal/model"
)

func TestValidateRequired(t *testing.T) {
	tests := []struct {
		value   string
		wantErr bool
	}{
		{"hello", false},
		{"", true},
		{" ", true},
		{"\t\n", true},
		{" x ", false},
	}
	for _, tt := range tests {
		errs := &ValidationError{}
		validateRequired(errs, "field", tt.value)
		if tt.wantErr && !errs.HasErrors() {
			t.Errorf("validateRequired(%q): expected error", tt.value)
		}
		if !tt.wantErr && errs.HasErrors() {
			t.Errorf("validateRequired(%q): unexpected error: %s", tt.value, errs.Error())
		}
	}
}

func TestValidateMaxLen(t *testing.T) {
	errs := &ValidationError{}
	validateMaxLen(errs, "f", "abc", 5)
	if errs.HasErrors() {
		t.Error("3 chars <= 5 should pass")
	}

	errs = &ValidationError{}
	validateMaxLen(errs, "f", "abcdef", 5)
	if !errs.HasErrors() {
		t.Error("6 chars > 5 should fail")
	}

	// Test with Unicode (rune count, not byte count)
	errs = &ValidationError{}
	validateMaxLen(errs, "f", "日本語", 3)
	if errs.HasErrors() {
		t.Error("3 runes <= 3 should pass")
	}

	errs = &ValidationError{}
	validateMaxLen(errs, "f", "日本語X", 3)
	if !errs.HasErrors() {
		t.Error("4 runes > 3 should fail")
	}
}

func TestValidateEnum(t *testing.T) {
	allowed := []string{"a", "b", "c"}

	errs := &ValidationError{}
	validateEnum(errs, "f", "b", allowed)
	if errs.HasErrors() {
		t.Error("'b' should be valid")
	}

	errs = &ValidationError{}
	validateEnum(errs, "f", "d", allowed)
	if !errs.HasErrors() {
		t.Error("'d' should be invalid")
	}
	if !strings.Contains(errs.Error(), "must be one of: a, b, c") {
		t.Errorf("unexpected error message: %s", errs.Error())
	}

	// Empty string should fail
	errs = &ValidationError{}
	validateEnum(errs, "f", "", allowed)
	if !errs.HasErrors() {
		t.Error("empty should be invalid")
	}
}

func TestValidateEstimate(t *testing.T) {
	// All zeros — skip (no error)
	errs := &ValidationError{}
	validateEstimate(errs, "e", fair.Estimate{}, 0)
	if errs.HasErrors() {
		t.Error("all-zero estimate should pass")
	}

	// Valid estimate
	errs = &ValidationError{}
	validateEstimate(errs, "e", fair.Estimate{Min: 1, ML: 5, Max: 10}, 0)
	if errs.HasErrors() {
		t.Errorf("valid estimate: %s", errs.Error())
	}

	// Min > ML
	errs = &ValidationError{}
	validateEstimate(errs, "e", fair.Estimate{Min: 10, ML: 5, Max: 20}, 0)
	if !errs.HasErrors() {
		t.Error("min > ml should fail")
	}

	// ML > Max
	errs = &ValidationError{}
	validateEstimate(errs, "e", fair.Estimate{Min: 1, ML: 20, Max: 10}, 0)
	if !errs.HasErrors() {
		t.Error("ml > max should fail")
	}

	// Negative value
	errs = &ValidationError{}
	validateEstimate(errs, "e", fair.Estimate{Min: -1, ML: 5, Max: 10}, 0)
	if !errs.HasErrors() {
		t.Error("negative min should fail")
	}

	// NaN
	errs = &ValidationError{}
	validateEstimate(errs, "e", fair.Estimate{Min: math.NaN(), ML: 5, Max: 10}, 0)
	if !errs.HasErrors() {
		t.Error("NaN should fail")
	}

	// Inf
	errs = &ValidationError{}
	validateEstimate(errs, "e", fair.Estimate{Min: 1, ML: math.Inf(1), Max: 10}, 0)
	if !errs.HasErrors() {
		t.Error("Inf should fail")
	}

	// Exceeds maxValue (probability)
	errs = &ValidationError{}
	validateEstimate(errs, "e", fair.Estimate{Min: 0.1, ML: 0.5, Max: 1.5}, 1)
	if !errs.HasErrors() {
		t.Error("value > 1.0 with maxValue=1 should fail")
	}

	// Valid probability
	errs = &ValidationError{}
	validateEstimate(errs, "e", fair.Estimate{Min: 0.1, ML: 0.5, Max: 1.0}, 1)
	if errs.HasErrors() {
		t.Errorf("valid probability: %s", errs.Error())
	}

	// Long rationale
	errs = &ValidationError{}
	validateEstimate(errs, "e", fair.Estimate{Min: 1, ML: 2, Max: 3, Rationale: strings.Repeat("x", 1001)}, 0)
	if !errs.HasErrors() {
		t.Error("long rationale should fail")
	}
}

func TestValidateControlStatuses(t *testing.T) {
	for _, s := range ValidControlStatuses {
		errs := validateControlFields("status", s)
		if errs.HasErrors() {
			t.Errorf("valid status %q rejected: %s", s, errs.Error())
		}
	}
	errs := validateControlFields("status", "invalid")
	if !errs.HasErrors() {
		t.Error("invalid status should be rejected")
	}
}

func TestValidateGapSeverities(t *testing.T) {
	for _, s := range ValidGapSeverities {
		errs := validateGapFields("severity", s)
		if errs.HasErrors() {
			t.Errorf("valid severity %q rejected: %s", s, errs.Error())
		}
	}
	errs := validateGapFields("severity", "extreme")
	if !errs.HasErrors() {
		t.Error("invalid severity should be rejected")
	}
}

func TestValidateGapStatuses(t *testing.T) {
	for _, s := range ValidGapStatuses {
		errs := validateGapFields("status", s)
		if errs.HasErrors() {
			t.Errorf("valid gap status %q rejected: %s", s, errs.Error())
		}
	}
	errs := validateGapFields("status", "resolved")
	if !errs.HasErrors() {
		t.Error("invalid gap status should be rejected")
	}
}

func TestValidateRiskForm(t *testing.T) {
	// Valid direct mode risk
	r := &model.Risk{
		Scenario: "Data breach",
		LossEvent: fair.LossEvent{
			LEFMode:   fair.LEFDirect,
			DirectLEF: fair.Estimate{Min: 0.1, ML: 0.5, Max: 1.0},
			PL: fair.LossForm{
				ProdL: fair.Estimate{Min: 1000, ML: 5000, Max: 10000},
			},
		},
	}
	errs := validateRiskForm(r)
	if errs.HasErrors() {
		t.Errorf("valid risk should pass: %s", errs.Error())
	}

	// Missing scenario
	r2 := &model.Risk{
		Scenario: "",
		LossEvent: fair.LossEvent{
			LEFMode: fair.LEFDirect,
		},
	}
	errs = validateRiskForm(r2)
	if !errs.HasErrors() {
		t.Error("empty scenario should fail")
	}

	// Scenario too long
	r3 := &model.Risk{
		Scenario: strings.Repeat("x", 501),
		LossEvent: fair.LossEvent{
			LEFMode: fair.LEFDirect,
		},
	}
	errs = validateRiskForm(r3)
	if !errs.HasErrors() {
		t.Error("long scenario should fail")
	}

	// Invalid susceptibility (> 1)
	r4 := &model.Risk{
		Scenario: "Test",
		LossEvent: fair.LossEvent{
			LEFMode: fair.LEFDecomposed,
			TEF:     fair.Estimate{Min: 1, ML: 2, Max: 3},
			Susc:    fair.Estimate{Min: 0.5, ML: 0.8, Max: 1.5},
		},
	}
	errs = validateRiskForm(r4)
	if !errs.HasErrors() {
		t.Error("susceptibility > 1 should fail")
	}

	// Invalid LEF mode
	r5 := &model.Risk{
		Scenario: "Test",
		LossEvent: fair.LossEvent{
			LEFMode: fair.LEFMode(99), // invalid int value
		},
	}
	errs = validateRiskForm(r5)
	if !errs.HasErrors() {
		t.Error("invalid lef_mode should fail")
	}
}

func TestValidateControlForm(t *testing.T) {
	// Valid
	errs := validateControlForm("CTRL-001", "Firewall", "Perimeter firewall", "implemented")
	if errs.HasErrors() {
		t.Errorf("valid control should pass: %s", errs.Error())
	}

	// Missing name
	errs = validateControlForm("CTRL-001", "", "", "planned")
	if !errs.HasErrors() {
		t.Error("empty name should fail")
	}

	// Invalid status
	errs = validateControlForm("CTRL-001", "Test", "", "bogus")
	if !errs.HasErrors() {
		t.Error("invalid status should fail")
	}

	// Identifier too long
	errs = validateControlForm(strings.Repeat("X", 21), "Test", "", "planned")
	if !errs.HasErrors() {
		t.Error("long identifier should fail")
	}
}

func TestValidateGapForm(t *testing.T) {
	// Valid
	errs := validateGapForm("GAP-001", "Missing MFA", "No MFA", "high", "open")
	if errs.HasErrors() {
		t.Errorf("valid gap should pass: %s", errs.Error())
	}

	// Invalid severity
	errs = validateGapForm("GAP-001", "Test", "", "extreme", "open")
	if !errs.HasErrors() {
		t.Error("invalid severity should fail")
	}

	// Invalid status
	errs = validateGapForm("GAP-001", "Test", "", "high", "resolved")
	if !errs.HasErrors() {
		t.Error("invalid gap status should fail")
	}
}

func TestValidateRequirementForm(t *testing.T) {
	// Valid
	errs := validateRequirementForm("REQ-001", "Access Control", "RBAC", "ISO 27001")
	if errs.HasErrors() {
		t.Errorf("valid requirement should pass: %s", errs.Error())
	}

	// Missing identifier
	errs = validateRequirementForm("", "Test", "", "")
	if !errs.HasErrors() {
		t.Error("empty identifier should fail")
	}

	// Source too long
	errs = validateRequirementForm("REQ-001", "Test", "", strings.Repeat("x", 501))
	if !errs.HasErrors() {
		t.Error("long source should fail")
	}
}

func TestValidateControlFunction(t *testing.T) {
	// Valid
	errs := validateControlFunction(cam.LECAvoidance, "test notes", cam.Effectiveness{
		Capability:  fair.Estimate{Min: 0.1, ML: 0.5, Max: 0.9},
		Coverage:    fair.Estimate{Min: 0.2, ML: 0.6, Max: 1.0},
		Reliability: fair.Estimate{Min: 0.3, ML: 0.7, Max: 0.95},
	})
	if errs.HasErrors() {
		t.Errorf("valid control function should pass: %s", errs.Error())
	}

	// Invalid function
	errs = validateControlFunction(cam.Function("BOGUS"), "", cam.Effectiveness{})
	if !errs.HasErrors() {
		t.Error("invalid function should fail")
	}

	// Capability > 1
	errs = validateControlFunction(cam.LECAvoidance, "", cam.Effectiveness{
		Capability: fair.Estimate{Min: 0.1, ML: 0.5, Max: 1.5},
	})
	if !errs.HasErrors() {
		t.Error("capability > 1 should fail")
	}

	// Notes too long
	errs = validateControlFunction(cam.LECAvoidance, strings.Repeat("x", 2001), cam.Effectiveness{})
	if !errs.HasErrors() {
		t.Error("long notes should fail")
	}
}

func TestValidateAuditEntityType(t *testing.T) {
	for _, et := range ValidAuditEntityTypes {
		errs := validateAuditEntityType(et)
		if errs.HasErrors() {
			t.Errorf("valid entity type %q rejected: %s", et, errs.Error())
		}
	}
	errs := validateAuditEntityType("bogus")
	if !errs.HasErrors() {
		t.Error("invalid entity type should be rejected")
	}
	errs = validateAuditEntityType("")
	if !errs.HasErrors() {
		t.Error("empty entity type should be rejected")
	}
}

func TestValidationErrorMessage(t *testing.T) {
	errs := &ValidationError{}
	if errs.Error() != "validation failed" {
		t.Errorf("empty error message: %q", errs.Error())
	}

	errs.Add("name", "required")
	errs.Add("status", "must be one of: a, b")
	want := "name: required; status: must be one of: a, b"
	if errs.Error() != want {
		t.Errorf("error message = %q, want %q", errs.Error(), want)
	}
}

func TestValidateEffectiveness(t *testing.T) {
	// All zeros — should pass (empty)
	errs := &ValidationError{}
	validateEffectiveness(errs, "eff", cam.Effectiveness{})
	if errs.HasErrors() {
		t.Error("empty effectiveness should pass")
	}

	// Valid
	errs = &ValidationError{}
	validateEffectiveness(errs, "eff", cam.Effectiveness{
		Capability:  fair.Estimate{Min: 0.1, ML: 0.5, Max: 0.9},
		Coverage:    fair.Estimate{Min: 0.0, ML: 0.5, Max: 1.0},
		Reliability: fair.Estimate{Min: 0.2, ML: 0.6, Max: 0.8},
	})
	if errs.HasErrors() {
		t.Errorf("valid effectiveness: %s", errs.Error())
	}

	// Reliability > 1
	errs = &ValidationError{}
	validateEffectiveness(errs, "eff", cam.Effectiveness{
		Reliability: fair.Estimate{Min: 0.5, ML: 0.8, Max: 1.2},
	})
	if !errs.HasErrors() {
		t.Error("reliability > 1 should fail")
	}
}

func TestValidateLossForm(t *testing.T) {
	// All zeros — should pass
	errs := &ValidationError{}
	validateLossForm(errs, "pl", fair.LossForm{})
	if errs.HasErrors() {
		t.Error("empty loss form should pass")
	}

	// Valid
	errs = &ValidationError{}
	validateLossForm(errs, "pl", fair.LossForm{
		ProdL: fair.Estimate{Min: 1000, ML: 5000, Max: 10000},
		RespC: fair.Estimate{Min: 500, ML: 2000, Max: 5000},
	})
	if errs.HasErrors() {
		t.Errorf("valid loss form: %s", errs.Error())
	}

	// One bad field
	errs = &ValidationError{}
	validateLossForm(errs, "pl", fair.LossForm{
		ProdL: fair.Estimate{Min: 10000, ML: 5000, Max: 1000}, // inverted
	})
	if !errs.HasErrors() {
		t.Error("inverted estimate should fail")
	}
}
