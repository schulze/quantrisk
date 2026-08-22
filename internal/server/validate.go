package server

import (
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/schulze/quantrisk/fair"
	"github.com/schulze/quantrisk/fair/cam"
	"github.com/schulze/quantrisk/internal/model"
)

// ValidationError collects multiple field-level validation errors.
type ValidationError struct {
	Errors []FieldError
}

type FieldError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if len(e.Errors) == 0 {
		return "validation failed"
	}
	msgs := make([]string, len(e.Errors))
	for i, fe := range e.Errors {
		msgs[i] = fe.Field + ": " + fe.Message
	}
	return strings.Join(msgs, "; ")
}

func (e *ValidationError) Add(field, message string) {
	e.Errors = append(e.Errors, FieldError{Field: field, Message: message})
}

func (e *ValidationError) HasErrors() bool {
	return len(e.Errors) > 0
}

// validateRequired checks that a string is non-empty after trimming.
func validateRequired(errs *ValidationError, field, value string) {
	if strings.TrimSpace(value) == "" {
		errs.Add(field, "required")
	}
}

// validateMaxLen checks that a string does not exceed maxLen runes.
func validateMaxLen(errs *ValidationError, field, value string, maxLen int) {
	if len([]rune(value)) > maxLen {
		errs.Add(field, fmt.Sprintf("must be at most %d characters", maxLen))
	}
}

// Allowed values for enum fields.
var (
	ValidControlStatuses  = []string{"planned", "implemented", "verified", "ineffective"}
	ValidGapSeverities    = []string{"low", "medium", "high", "critical"}
	ValidGapStatuses      = []string{"open", "mitigated", "accepted", "closed"}
	ValidLEFModes         = []string{fair.LEFDirect.String(), fair.LEFDecomposed.String()}
	ValidAuditEntityTypes = []string{"risk", "control", "requirement", "gap"}
)

// validateEnum checks that value is one of the allowed values.
func validateEnum(errs *ValidationError, field, value string, allowed []string) {
	for _, a := range allowed {
		if value == a {
			return
		}
	}
	errs.Add(field, fmt.Sprintf("must be one of: %s", strings.Join(allowed, ", ")))
}

// validateEstimate validates a FAIR three-point estimate.
// If the estimate is all zeros it's treated as empty/optional and skipped.
// If maxValue > 0, values must be <= maxValue (for probabilities 0-1).
func validateEstimate(errs *ValidationError, prefix string, e fair.Estimate, maxValue float64) {
	// Skip all-zero estimates (empty/unset)
	if e.Min == 0 && e.ML == 0 && e.Max == 0 {
		return
	}

	for _, check := range []struct {
		name string
		val  float64
	}{
		{"min", e.Min}, {"ml", e.ML}, {"max", e.Max},
	} {
		field := prefix + "." + check.name
		if math.IsNaN(check.val) || math.IsInf(check.val, 0) {
			errs.Add(field, "must be a finite number")
			return // can't do further checks with non-finite values
		}
		if check.val < 0 {
			errs.Add(field, "must not be negative")
		}
		if maxValue > 0 && check.val > maxValue {
			errs.Add(field, fmt.Sprintf("must not exceed %.1f", maxValue))
		}
	}

	if e.Min > e.ML {
		errs.Add(prefix+".min", "must not exceed most likely value")
	}
	if e.ML > e.Max {
		errs.Add(prefix+".ml", "must not exceed maximum value")
	}

	// Rationale length
	validateMaxLen(errs, prefix+".rationale", e.Rationale, 1000)
}

// validateLossForm validates all six estimate fields in a FAIR LossForm.
func validateLossForm(errs *ValidationError, prefix string, lf fair.LossForm) {
	validateEstimate(errs, prefix+".prodl", lf.ProdL, 0)
	validateEstimate(errs, prefix+".respc", lf.RespC, 0)
	validateEstimate(errs, prefix+".replc", lf.ReplC, 0)
	validateEstimate(errs, prefix+".finju", lf.FinJu, 0)
	validateEstimate(errs, prefix+".repud", lf.RepuD, 0)
	validateEstimate(errs, prefix+".cadvl", lf.CAdvL, 0)
}

// validateEffectiveness validates FAIR-CAM effectiveness estimates (all 0-1).
func validateEffectiveness(errs *ValidationError, prefix string, eff cam.Effectiveness) {
	validateEstimate(errs, prefix+".capability", eff.Capability, 1)
	validateEstimate(errs, prefix+".coverage", eff.Coverage, 1)
	validateEstimate(errs, prefix+".reliability", eff.Reliability, 1)
}

// validateRiskForm validates a risk model from form input.
// The model.Risk is already parsed; this validates the values.
func validateRiskForm(r *model.Risk) *ValidationError {
	errs := &ValidationError{}
	validateRequired(errs, "scenario", r.Scenario)
	validateMaxLen(errs, "scenario", r.Scenario, 500)
	validateEnum(errs, "lef_mode", r.LEFMode.String(), ValidLEFModes)

	if r.LEFMode == fair.LEFDirect {
		validateEstimate(errs, "lef", r.DirectLEF, 0)
	} else {
		validateEstimate(errs, "tef", r.TEF, 0)
		validateEstimate(errs, "susc", r.Susc, 1)
	}

	validateLossForm(errs, "pl", r.PL)
	validateLossForm(errs, "sl", r.SL)

	return errs
}

// validateControlFields validates a control's patchable fields.
func validateControlFields(field, value string) *ValidationError {
	errs := &ValidationError{}
	switch field {
	case "name":
		validateRequired(errs, "name", value)
		validateMaxLen(errs, "name", value, 200)
	case "description":
		validateMaxLen(errs, "description", value, 5000)
	case "status":
		validateEnum(errs, "status", value, ValidControlStatuses)
	case "identifier":
		validateRequired(errs, "identifier", value)
		validateMaxLen(errs, "identifier", value, 20)
	}
	return errs
}

// validateControlForm validates a full control update.
func validateControlForm(identifier, name, description, status string) *ValidationError {
	errs := &ValidationError{}
	validateRequired(errs, "identifier", identifier)
	validateMaxLen(errs, "identifier", identifier, 20)
	validateRequired(errs, "name", name)
	validateMaxLen(errs, "name", name, 200)
	validateMaxLen(errs, "description", description, 5000)
	validateEnum(errs, "status", status, ValidControlStatuses)
	return errs
}

// validateRequirementFields validates a requirement's patchable fields.
func validateRequirementFields(field, value string) *ValidationError {
	errs := &ValidationError{}
	switch field {
	case "name":
		validateRequired(errs, "name", value)
		validateMaxLen(errs, "name", value, 200)
	case "description":
		validateMaxLen(errs, "description", value, 5000)
	case "source":
		validateMaxLen(errs, "source", value, 500)
	}
	return errs
}

// validateRequirementForm validates a full requirement update.
func validateRequirementForm(identifier, name, description, source string) *ValidationError {
	errs := &ValidationError{}
	validateRequired(errs, "identifier", identifier)
	validateMaxLen(errs, "identifier", identifier, 20)
	validateRequired(errs, "name", name)
	validateMaxLen(errs, "name", name, 200)
	validateMaxLen(errs, "description", description, 5000)
	validateMaxLen(errs, "source", source, 500)
	return errs
}

// validateGapFields validates a gap's patchable fields.
func validateGapFields(field, value string) *ValidationError {
	errs := &ValidationError{}
	switch field {
	case "name":
		validateRequired(errs, "name", value)
		validateMaxLen(errs, "name", value, 200)
	case "description":
		validateMaxLen(errs, "description", value, 5000)
	case "severity":
		validateEnum(errs, "severity", value, ValidGapSeverities)
	case "status":
		validateEnum(errs, "status", value, ValidGapStatuses)
	}
	return errs
}

// validateGapForm validates a full gap update.
func validateGapForm(identifier, name, description, severity, status string) *ValidationError {
	errs := &ValidationError{}
	validateRequired(errs, "identifier", identifier)
	validateMaxLen(errs, "identifier", identifier, 20)
	validateRequired(errs, "name", name)
	validateMaxLen(errs, "name", name, 200)
	validateMaxLen(errs, "description", description, 5000)
	validateEnum(errs, "severity", severity, ValidGapSeverities)
	validateEnum(errs, "status", status, ValidGapStatuses)
	return errs
}

// validateRiskPatchField validates a risk's patchable field value.
func validateRiskPatchField(field, value string) *ValidationError {
	errs := &ValidationError{}
	switch field {
	case "scenario":
		validateRequired(errs, "scenario", value)
		validateMaxLen(errs, "scenario", value, 500)
	}
	return errs
}

// validateControlFunction validates a control function creation.
func validateControlFunction(fn cam.Function, notes string, eff cam.Effectiveness) *ValidationError {
	errs := &ValidationError{}
	if _, ok := cam.Lookup(fn); !ok {
		errs.Add("function", "unknown FAIR-CAM function")
	}
	validateMaxLen(errs, "notes", notes, 2000)
	validateEffectiveness(errs, "effectiveness", eff)
	return errs
}

// validateAuditEntityType validates an audit log entity type.
func validateAuditEntityType(entityType string) *ValidationError {
	errs := &ValidationError{}
	validateEnum(errs, "entityType", entityType, ValidAuditEntityTypes)
	return errs
}

// validationFailed writes a 422 Unprocessable Entity response with the validation error.
func validationFailed(w http.ResponseWriter, errs *ValidationError) {
	http.Error(w, errs.Error(), http.StatusUnprocessableEntity)
}
