// Package fair implements data types and Monte Carlo simulation for the
// Factor Analysis of Information Risk (FAIR) model, version 3.0.
// See https://www.fairinstitute.org/.
package fair

import (
	"database/sql/driver"
	"fmt"
)

//go:generate stringer -type=DistType,LEFMode -linecomment

// Distribution type

// DistType identifies the probability distribution used to sample an Estimate
// during Monte Carlo simulation.
type DistType int

const (
	// DistDefault lets the simulation engine choose based on the
	// estimate's role (Poisson for frequencies, Beta for probabilities,
	// PERT for magnitudes). This is the zero value.
	DistDefault DistType = iota // default

	// DistPERT uses a four-parameter Beta (PERT) distribution fitted
	// to Min, ML (mode), and Max with kurtosis=4.
	DistPERT // pert

	// DistPoisson uses a Poisson distribution whose lambda is the
	// PERT mean of (Min, ML, Max). Draws are integer-valued event
	// counts.
	DistPoisson // poisson

	// DistLogNormal fits a log-normal distribution to a 90% confidence
	// interval [Min, Max]. ML is ignored for fitting; Min and Max map
	// to the 5th and 95th percentiles.
	DistLogNormal // lognormal

	// DistBeta uses a Beta distribution on [0, 1] parameterized from
	// a three-point estimate. Suitable for probabilities like
	// Susceptibility.
	DistBeta // beta
)

var distNames = map[string]DistType{
	"":          DistDefault,
	"default":   DistDefault,
	"pert":      DistPERT,
	"poisson":   DistPoisson,
	"lognormal": DistLogNormal,
	"beta":      DistBeta,
}

// ParseDistType returns the DistType for the given string, or an error if
// the string is not a recognized distribution name.
func ParseDistType(s string) (DistType, error) {
	if dt, ok := distNames[s]; ok {
		return dt, nil
	}
	return DistDefault, fmt.Errorf("unknown distribution type: %q", s)
}

// MarshalText implements encoding.TextMarshaler so DistType serializes
// as its lowercase name in JSON.
func (d DistType) MarshalText() ([]byte, error) {
	if d == DistDefault {
		return []byte(""), nil
	}
	return []byte(d.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler for JSON decoding.
func (d *DistType) UnmarshalText(b []byte) error {
	dt, err := ParseDistType(string(b))
	if err != nil {
		return err
	}
	*d = dt
	return nil
}

// LEF mode

// LEFMode controls how Loss Event Frequency is determined.
type LEFMode int

const (
	// LEFDecomposed means LEF is computed from TEF × Susceptibility.
	// This is the zero value (default for new risks).
	LEFDecomposed LEFMode = iota // decomposed

	// LEFDirect means LEF is estimated directly as a three-point estimate.
	LEFDirect // direct
)

var lefNames = map[string]LEFMode{
	"direct":     LEFDirect,
	"decomposed": LEFDecomposed,
}

// ParseLEFMode returns the LEFMode for the given string, or an error if
// the string is not recognized.
func ParseLEFMode(s string) (LEFMode, error) {
	if m, ok := lefNames[s]; ok {
		return m, nil
	}
	return LEFDirect, fmt.Errorf("unknown LEF mode: %q", s)
}

// MarshalText implements encoding.TextMarshaler so LEFMode serializes
// as its lowercase name in JSON.
func (m LEFMode) MarshalText() ([]byte, error) {
	return []byte(m.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler for JSON decoding.
func (m *LEFMode) UnmarshalText(b []byte) error {
	mode, err := ParseLEFMode(string(b))
	if err != nil {
		return err
	}
	*m = mode
	return nil
}

// Scan implements database/sql.Scanner so LEFMode can be read from SQLite
// TEXT columns.
func (m *LEFMode) Scan(src any) error {
	switch v := src.(type) {
	case string:
		mode, err := ParseLEFMode(v)
		if err != nil {
			return err
		}
		*m = mode
		return nil
	case []byte:
		return m.Scan(string(v))
	case nil:
		*m = LEFDirect
		return nil
	default:
		return fmt.Errorf("cannot scan %T into LEFMode", src)
	}
}

// Value implements database/sql/driver.Valuer so LEFMode can be written to
// SQLite TEXT columns.
func (m LEFMode) Value() (driver.Value, error) {
	return m.String(), nil
}

// Estimate

// Estimate represents a three-point estimate used throughout FAIR analyses.
// Min and Max bound the range; ML is the most likely value.
// Rationale captures the analyst's reasoning for the chosen values.
// Dist selects which probability distribution to use in simulation;
// the zero value (DistDefault) lets the simulation engine decide.
type Estimate struct {
	Min       float64  `json:"min"`
	ML        float64  `json:"ml"`
	Max       float64  `json:"max"`
	Rationale string   `json:"rationale,omitempty"`
	Dist      DistType `json:"dist,omitempty"`
}

// Mid returns the midpoint of the estimate range: (Min + Max) / 2.
func (e Estimate) Mid() float64 { return (e.Min + e.Max) / 2 }

// IsZero reports whether all numeric fields are zero, meaning the
// estimate has not been set.
func (e Estimate) IsZero() bool { return e.Min == 0 && e.ML == 0 && e.Max == 0 }

// PERTMean returns the PERT-weighted mean: (Min + 4×ML + Max) / 6.
// This weights the most-likely value four times more than the extremes.
func (e Estimate) PERTMean() float64 { return (e.Min + 4*e.ML + e.Max) / 6 }

// Loss forms

// LossForm captures the six standard forms of loss defined by FAIR.
type LossForm struct {
	ProdL Estimate `json:"prodl"` // Productivity Loss
	RespC Estimate `json:"respc"` // Response Costs
	ReplC Estimate `json:"replc"` // Replacement Costs
	FinJu Estimate `json:"finju"` // Fines and Judgments
	RepuD Estimate `json:"repud"` // Reputation Damage
	CAdvL Estimate `json:"cadvl"` // Competitive Advantage Loss
}

// Estimates returns all six loss form estimates as a slice, in the
// canonical order: ProdL, RespC, ReplC, FinJu, RepuD, CAdvL.
func (lf LossForm) Estimates() []Estimate {
	return []Estimate{lf.ProdL, lf.RespC, lf.ReplC, lf.FinJu, lf.RepuD, lf.CAdvL}
}

// Loss event

// LossEvent is a single FAIR loss event scenario.
//
// Risk = LEF × LM, where:
//   - LEF (Loss Event Frequency) is either estimated directly or decomposed
//     into TEF × Susceptibility
//   - LM (Loss Magnitude) is the sum of primary and secondary loss forms
type LossEvent struct {
	LEFMode   LEFMode  `json:"lef_mode"`   // direct or decomposed
	DirectLEF Estimate `json:"direct_lef"` // used when LEFMode == LEFDirect
	TEF       Estimate `json:"tef"`        // Threat Event Frequency (LEFDecomposed)
	Susc      Estimate `json:"susc"`       // Susceptibility 0–1 (LEFDecomposed)
	PL        LossForm `json:"pl"`         // Primary Loss
	SL        LossForm `json:"sl"`         // Secondary Loss
}

// LEF returns a point estimate of Loss Event Frequency using the
// midpoint of the range. For display/sorting purposes only; the
// simulation engine uses full distributions, not this value.
func (le LossEvent) LEF() float64 {
	if le.LEFMode == LEFDirect {
		return le.DirectLEF.Mid()
	}
	return le.TEF.Mid() * le.Susc.Mid()
}
