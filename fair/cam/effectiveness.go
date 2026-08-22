package cam

import "github.com/schulze/quantrisk/fair"

// Effectiveness captures the three FAIR-CAM dimensions of control operational
// effectiveness (maturity): Capability, Coverage, and Reliability.
// Each is expressed as a three-point estimate.
type Effectiveness struct {
	// Capability is the control's inherent ability to perform its intended
	// function. Expressed as 0–1 (probability / fraction of design quality).
	Capability fair.Estimate `json:"capability"`

	// Coverage measures the extent to which the control applies to relevant
	// assets, threats, or scenarios. Expressed as 0–1.
	Coverage fair.Estimate `json:"coverage"`

	// Reliability is the likelihood the control performs consistently when
	// needed. Expressed as 0–1.
	Reliability fair.Estimate `json:"reliability"`
}

// OverallMid returns a composite midpoint effectiveness as
// Capability.Mid × Coverage.Mid × Reliability.Mid.
func (e Effectiveness) OverallMid() float64 {
	return e.Capability.Mid() * e.Coverage.Mid() * e.Reliability.Mid()
}

// ControlAssignment maps a control to the FAIR-CAM function(s) it serves
// and captures its operational effectiveness for each function.
type ControlAssignment struct {
	// ControlID is the identifier of the control (links to model.Control).
	ControlID int64 `json:"control_id"`

	// Function is the FAIR-CAM function this control serves.
	Function Function `json:"function"`

	// Effectiveness is the control's operational maturity for this function.
	Effectiveness Effectiveness `json:"effectiveness"`

	// Notes captures analyst reasoning.
	Notes string `json:"notes,omitempty"`
}
